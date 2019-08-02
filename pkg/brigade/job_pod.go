package brigade

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/brigadecore/brigade-github-app/pkg/webhook"
	"github.com/lovethedrake/drakecore/config"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	api "k8s.io/kubernetes/pkg/apis/core"
)

const (
	srcVolumeName          = "src"
	dockerSocketVolumeName = "docker-socket"
)

func (e *executor) runJobPod(
	ctx context.Context,
	project Project,
	event Event,
	pipelineName string,
	stage int,
	job config.Job,
	environment []string,
	errCh chan<- error,
) {
	var err error

	webhookPayload := &webhook.Payload{}
	if err = json.Unmarshal(event.Payload, webhookPayload); err != nil {
		errCh <- err
		return
	}

	if webhookPayload.Type == "check_run" ||
		webhookPayload.Type == "check_suite" {
		if err = notifyCheckStart(
			webhookPayload,
			job.Name(),
			job.Name(),
		); err != nil {
			errCh <- err
			return
		}
	}

	jobName := fmt.Sprintf("%s-stage%d-%s", pipelineName, stage, job.Name())
	podName := fmt.Sprintf("%s-%s", jobName, event.BuildID)

	defer func() {
		// Ensure notification, if applicable
		if webhookPayload.Type == "check_run" ||
			webhookPayload.Type == "check_suite" {
			conclusion := "failure"
			select {
			case <-ctx.Done():
				conclusion = "cancelled"
			default:
				if err == nil {
					conclusion = "success"
				} else if _, ok := err.(*timedOutError); ok {
					conclusion = "timed_out"
				}
			}
			if nerr := notifyCheckCompleted(
				webhookPayload,
				job.Name(),
				job.Name(),
				conclusion,
			); nerr != nil {
				log.Printf("error sending notification to github: %s", nerr)
			}
		}
		// Return the error, even if it's nil
		errCh <- err
	}()

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
				"thedrake.io/stage":    strconv.Itoa(stage),
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

	var mainContainerName string
	containers := job.Containers()
	pod.Spec.Containers = make([]v1.Container, len(containers))
	for i, container := range containers {
		var jobPodContainer v1.Container
		jobPodContainer, err = getJobPodContainer(
			project,
			event,
			container,
			environment,
		)
		if err != nil {
			err = errors.Wrapf(
				err,
				"error building container spec for container \"%s\" of job \"%s\"",
				container.Name(),
				job.Name(),
			)
			return
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
		mainContainerName = container.Name()
		pod.Spec.Containers[0] = jobPodContainer
	}

	if _, err = e.kubeClient.CoreV1().Pods(
		project.Kubernetes.Namespace,
	).Create(pod); err != nil {
		err = errors.Wrapf(err, "error creating pod \"%s\"", podName)
		return
	}

	var podsWatcher watch.Interface
	podsWatcher, err =
		e.kubeClient.CoreV1().Pods(project.Kubernetes.Namespace).Watch(
			metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(
					api.ObjectNameField,
					podName,
				).String(),
			},
		)
	if err != nil {
		return
	}

	// Timeout
	// TODO: This probably should not be hard-coded
	timer := time.NewTimer(10 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case event := <-podsWatcher.ResultChan():
			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				err = errors.Errorf(
					"received unexpected object when watching pod \"%s\" for completion",
					podName,
				)
				return
			}
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Name == mainContainerName {
					if containerStatus.State.Terminated != nil {
						if containerStatus.State.Terminated.Reason == "Completed" {
							return
						}
						err = errors.Errorf("pod \"%s\" failed", podName)
						return
					}
					break
				}
			}
		case <-timer.C:
			err = &timedOutError{podName: podName}
			return
		case <-ctx.Done():
			return
		}
	}
}

func getJobPodContainer(
	project Project,
	event Event,
	container config.Container,
	environment []string,
) (v1.Container, error) {
	privileged := container.Privileged()
	if privileged && !project.AllowPrivilegedJobs {
		return v1.Container{}, errors.Errorf(
			"container \"%s\" requested to be privileged, but privileged jobs are "+
				"not permitted by this project",
			container.Name(),
		)
	}
	if container.MountDockerSocket() && !project.AllowHostMounts {
		return v1.Container{}, errors.Errorf(
			"container \"%s\" requested to mount the docker socket, but host "+
				"mounts are not permitted by this project",
			container.Name(),
		)
	}
	command, err := shellwords.Parse(container.Command())
	if err != nil {
		return v1.Container{}, err
	}
	env := make([]string, len(environment))
	copy(env, environment)
	env = append(env, container.Environment()...)
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
	for _, kv := range env {
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
