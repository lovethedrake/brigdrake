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
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	sourceVolumeName        = "source"
	sharedStorageVolumeName = "shared-storage"
	dockerSocketVolumeName  = "docker-socket"
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
			Volumes:       []v1.Volume{},
		},
	}

	primaryContainer := job.PrimaryContainer()
	sidecarContainers := job.SidecarContainers()

	// All the volumes we might need
	var jobUsesSource bool
	var jobUsesSharedStorage bool
	var jobUsessDockerSocket bool
	if primaryContainer.SourceMountPath() != "" {
		jobUsesSource = true
	}
	if primaryContainer.SharedStorageMountPath() != "" {
		jobUsesSharedStorage = true
	}
	if primaryContainer.MountDockerSocket() {
		jobUsessDockerSocket = true
	}
	for _, sidecarContainer := range job.SidecarContainers() {
		if sidecarContainer.SourceMountPath() != "" {
			jobUsesSource = true
		}
		if sidecarContainer.SharedStorageMountPath() != "" {
			jobUsesSharedStorage = true
		}
		if sidecarContainer.MountDockerSocket() {
			jobUsessDockerSocket = true
		}
	}
	if jobUsesSource {
		pod.Spec.Volumes = append(
			pod.Spec.Volumes,
			v1.Volume{
				Name: sourceVolumeName,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		)
	}
	if jobUsesSharedStorage {
		pod.Spec.Volumes = append(
			pod.Spec.Volumes,
			v1.Volume{
				Name: sharedStorageVolumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: sharedStoragePVCName(event.WorkerID, pipelineName),
					},
				},
			},
		)
	}
	if jobUsessDockerSocket {
		pod.Spec.Volumes = append(
			pod.Spec.Volumes,
			v1.Volume{
				Name: dockerSocketVolumeName,
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/var/run/docker.sock",
					},
				},
			},
		)
	}

	// If needed, use an init container for fetching source
	if jobUsesSource {
		sourceCloneContainer, err := buildSourceCloneContainer(project, event)
		if err != nil {
			err = errors.Wrap(
				err,
				"error building container spec for source clone init container",
			)
			return nil, err
		}
		pod.Spec.InitContainers = []v1.Container{sourceCloneContainer}
	}

	pod.Spec.ImagePullSecrets =
		make([]v1.LocalObjectReference, len(project.Kubernetes.ImagePullSecrets))
	for i, imagePullSecret := range project.Kubernetes.ImagePullSecrets {
		pod.Spec.ImagePullSecrets[i] = v1.LocalObjectReference{
			Name: imagePullSecret,
		}
	}

	// First the primary container
	pod.Spec.Containers = make([]v1.Container, 1+len(sidecarContainers))
	jobPodPrimaryContainer, err := buildJobPodContainer(
		project,
		event,
		primaryContainer,
		job.SourceMountMode(),
	)
	if err != nil {
		err = errors.Wrapf(
			err,
			"error building container spec for primary container %q of job %q",
			primaryContainer.Name(),
			job.Name(),
		)
		return nil, err
	}
	pod.Spec.Containers[0] = jobPodPrimaryContainer

	// Then the sidecars...
	for i, sidecarContainer := range sidecarContainers {
		jobPodSidecarContainer, err := buildJobPodContainer(
			project,
			event,
			sidecarContainer,
			job.SourceMountMode(),
		)
		if err != nil {
			err = errors.Wrapf(
				err,
				"error building container spec for sidecar container %q of job %q",
				sidecarContainer.Name(),
				job.Name(),
			)
			return nil, err
		}
		// +1 because the 0 element has the primary container.
		pod.Spec.Containers[i+1] = jobPodSidecarContainer
	}
	return pod, nil
}

func buildJobPodContainer(
	project brigade.Project,
	event brigade.Event,
	container config.Container,
	sourceMountMode config.SourceMountMode,
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
	c := v1.Container{
		Name:            container.Name(),
		Image:           container.Image(),
		ImagePullPolicy: v1.PullAlways,
		Env:             []v1.EnvVar{},
		SecurityContext: &v1.SecurityContext{
			Privileged: &privileged,
		},
		VolumeMounts: []v1.VolumeMount{},
		Stdin:        container.TTY(),
		TTY:          container.TTY(),
	}
	cmd := container.Command()
	if len(cmd) > 0 {
		c.Command = cmd
	}
	args := container.Args()
	if len(args) > 0 {
		c.Args = args
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
				Name:      sourceVolumeName,
				MountPath: container.SourceMountPath(),
				ReadOnly:  sourceMountMode == config.SourceMountModeReadOnly,
			},
		)
	}
	if container.WorkingDirectory() != "" {
		c.WorkingDir = container.WorkingDirectory()
	}
	if container.SharedStorageMountPath() != "" {
		c.VolumeMounts = append(
			c.VolumeMounts,
			v1.VolumeMount{
				Name:      sharedStorageVolumeName,
				MountPath: container.SharedStorageMountPath(),
			},
		)
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
