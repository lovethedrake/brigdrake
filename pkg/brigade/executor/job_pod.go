package executor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/brigdrake/pkg/drake"
	"github.com/lovethedrake/drakecore/config"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	srcVolumeName          = "src"
	dockerSocketVolumeName = "docker-socket"
)

func runJobPod(
	ctx context.Context,
	project brigade.Project,
	event brigade.Event,
	pipelineName string,
	job config.Job,
	jobStatusNotifier drake.JobStatusNotifier,
	kubeClient kubernetes.Interface,
) error {
	var err error
	if jobStatusNotifier != nil {
		if err = jobStatusNotifier.SendInProgressNotification(job); err != nil {
			return err
		}
		defer func() {
			jsnFn := jobStatusNotifier.SendFailureNotification
			select {
			case <-ctx.Done():
				jsnFn = jobStatusNotifier.SendCancelledNotification
			default:
				if err == nil {
					jsnFn = jobStatusNotifier.SendSuccessNotification
				} else if _, ok := err.(*timedOutError); ok {
					jsnFn = jobStatusNotifier.SendTimedOutNotification
				}
			}
			if err = jsnFn(job); err != nil {
				log.Printf("error sending job status notification: %s", err)
			}
		}()
	}

	// TODO: Let's not define the values for these in two places.
	jobName := fmt.Sprintf("%s-%s", pipelineName, job.Name())
	podName := fmt.Sprintf("%s-%s", jobName, event.BuildID)

	var pod *v1.Pod
	pod, err = buildJobPod(project, event, pipelineName, job)

	if _, err = kubeClient.CoreV1().Pods(
		project.Kubernetes.Namespace,
	).Create(pod); err != nil {
		err = errors.Wrapf(err, "error creating pod %q", podName)
		return err
	}

	return waitForJobPodCompletion(
		ctx,
		project.Kubernetes.Namespace,
		jobName,
		podName,
		10*time.Minute, // TODO: This probably shouldn't be hardcoded
		kubeClient,
	)
}

func waitForJobPodCompletion(
	ctx context.Context,
	namespace string,
	jobName string,
	podName string,
	timeout time.Duration,
	kubeClient kubernetes.Interface,
) error {
	podsWatcher, err := kubeClient.CoreV1().Pods(namespace).Watch(
		metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				api.ObjectNameField,
				podName,
			).String(),
		},
	)
	if err != nil {
		return err
	}

	// Timeout
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case event := <-podsWatcher.ResultChan():
			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				err = errors.Errorf(
					"received unexpected object when watching pod %q for completion",
					podName,
				)
				return err
			}
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Name == pod.Spec.Containers[0].Name {
					if containerStatus.State.Terminated != nil {
						if containerStatus.State.Terminated.Reason == "Completed" {
							return nil
						}
						err = errors.Errorf("pod %q failed", podName)
						return err
					}
					break
				}
			}
		case <-timer.C:
			err = &timedOutError{job: jobName}
			return err
		case <-ctx.Done():
			err = &inProgressJobAbortedError{job: jobName}
			return err
		}
	}
}

func buildJobPod(
	project brigade.Project,
	event brigade.Event,
	pipelineName string,
	job config.Job,
) (*v1.Pod, error) {
	jobName := fmt.Sprintf("%s-%s", pipelineName, job.Name())
	podName := fmt.Sprintf("%s-%s", jobName, event.BuildID)
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"heritage":             "brigade",
				"component":            "job",
				"jobname":              jobName,
				"project":              project.ID,
				"worker":               event.WorkerID,
				"build":                event.BuildID,
				"thedrake.io/pipeline": pipelineName,
				"thedrake.io/job":      job.Name(),
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: srcVolumeName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: srcPVCName(event.WorkerID, pipelineName),
						},
					},
				},
				{
					Name: dockerSocketVolumeName,
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/var/run/docker.sock",
						},
					},
				},
			},
		},
	}
	pod.Spec.ImagePullSecrets =
		make([]v1.LocalObjectReference, len(project.Kubernetes.ImagePullSecrets))
	for i, imagePullSecret := range project.Kubernetes.ImagePullSecrets {
		pod.Spec.ImagePullSecrets[i] = v1.LocalObjectReference{
			Name: imagePullSecret,
		}
	}
	containers := job.Containers()
	pod.Spec.Containers = make([]v1.Container, len(containers))
	for i, container := range containers {
		jobPodContainer, err := buildJobPodContainer(
			project,
			event,
			container,
		)
		if err != nil {
			err = errors.Wrapf(
				err,
				"error building container spec for container %q of job %q",
				container.Name(),
				job.Name(),
			)
			return nil, err
		}
		// We'll treat all but the last container as sidecars. i.e. The last
		// container in the job should be container 0 in the pod spec.
		if i < len(containers)-1 {
			// +1 because we want to leave room in the first (0th) position for the
			// primary container.
			pod.Spec.Containers[i+1] = jobPodContainer
			continue
		}
		// This is the primary container. Make it the first (0th) in the pod spec.
		pod.Spec.Containers[0] = jobPodContainer
	}
	return pod, nil
}

func buildJobPodContainer(
	project brigade.Project,
	event brigade.Event,
	container config.Container,
) (v1.Container, error) {
	privileged := container.Privileged()
	if privileged && !project.AllowPrivilegedJobs {
		return v1.Container{}, errors.Errorf(
			"container %q requested to be privileged, but privileged jobs are "+
				"not permitted by this project",
			container.Name(),
		)
	}
	if container.MountDockerSocket() && !project.AllowHostMounts {
		return v1.Container{}, errors.Errorf(
			"container %q requested to mount the docker socket, but host "+
				"mounts are not permitted by this project",
			container.Name(),
		)
	}
	command, err := shellwords.Parse(container.Command())
	if err != nil {
		return v1.Container{}, err
	}
	c := v1.Container{
		Name:            container.Name(),
		Image:           container.Image(),
		ImagePullPolicy: v1.PullAlways,
		Command:         command,
		Env:             []v1.EnvVar{},
		SecurityContext: &v1.SecurityContext{
			Privileged: &privileged,
		},
		VolumeMounts: []v1.VolumeMount{},
		Stdin:        container.TTY(),
		TTY:          container.TTY(),
	}
	for k := range project.Secrets {
		c.Env = append(
			c.Env,
			v1.EnvVar{
				Name: k,
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: strings.ToLower(event.BuildID),
						},
						Key: k,
					},
				},
			},
		)
	}
	for _, kv := range container.Environment() {
		kvTokens := strings.SplitN(kv, "=", 2)
		if len(kvTokens) == 2 {
			c.Env = append(
				c.Env,
				v1.EnvVar{
					Name:  kvTokens[0],
					Value: kvTokens[1],
				},
			)
			continue
		}
		if len(kvTokens) == 1 {
			c.Env = append(
				c.Env,
				v1.EnvVar{
					Name: kvTokens[0],
				},
			)
		}
	}
	if container.SourceMountPath() != "" {
		c.VolumeMounts = append(
			c.VolumeMounts,
			v1.VolumeMount{
				Name:      srcVolumeName,
				MountPath: container.SourceMountPath(),
			},
		)
	}
	if container.WorkingDirectory() != "" {
		c.WorkingDir = container.WorkingDirectory()
	}
	if container.MountDockerSocket() {
		c.VolumeMounts = append(
			c.VolumeMounts,
			v1.VolumeMount{
				Name:      dockerSocketVolumeName,
				MountPath: "/var/run/docker.sock",
			},
		)
	}
	return c, nil
}
