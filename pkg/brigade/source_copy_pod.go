package brigade

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	api "k8s.io/kubernetes/pkg/apis/core"
)

func (e *executor) runSourceClonePod(
	ctx context.Context,
	project Project,
	event Event,
	pipelineName string,
) error {
	const srcDir = "/src"
	jobName := fmt.Sprintf("%s-source-clone", pipelineName)
	podName := fmt.Sprintf("%s-%s", jobName, event.BuildID)
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"heritage":  "brigade",
				"component": "job",
				"jobname":   jobName,
				"project":   project.ID,
				"worker":    event.WorkerID,
				"build":     event.BuildID,
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:            "source-cloner",
					Image:           "brigadecore/git-sidecar:v1.1.0",
					ImagePullPolicy: v1.PullAlways,
					Env: []v1.EnvVar{
						{
							Name:  "CI",
							Value: "true",
						},
						{
							Name:  "BRIGADE_BUILD_ID",
							Value: event.BuildID,
						},
						{
							Name:  "BRIGADE_COMMIT_ID",
							Value: event.Revision.Commit,
						},
						{
							Name:  "BRIGADE_COMMIT_REF",
							Value: event.Revision.Ref,
						},
						{
							Name:  "BRIGADE_EVENT_PROVIDER",
							Value: event.Provider,
						},
						{
							Name:  "BRIGADE_EVENT_TYPE",
							Value: event.Type,
						},
						{
							Name:  "BRIGADE_PROJECT_ID",
							Value: project.ID,
						},
						{
							Name:  "BRIGADE_REMOTE_URL",
							Value: project.Repo.CloneURL,
						},
						{
							Name:  "BRIGADE_WORKSPACE",
							Value: srcDir,
						},
						{
							Name:  "BRIGADE_PROJECT_NAMESPACE",
							Value: project.Kubernetes.Namespace,
						},
						{
							Name:  "BRIGADE_SUBMODULES",
							Value: strconv.FormatBool(project.Repo.InitGitSubmodules),
						},
						// TODO: Not really sure where I can get this from
						// {
						// 	Name:  "BRIGADE_LOG_LEVEL",
						// 	Value: "info",
						// },
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      srcVolumeName,
							MountPath: srcDir,
						},
					},
					Resources: v1.ResourceRequirements{
						Limits:   v1.ResourceList{},
						Requests: v1.ResourceList{},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: srcVolumeName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: srcPVCName(event.WorkerID, pipelineName),
						},
					},
				},
			},
		},
	}
	if project.Repo.SSHKey != "" {
		pod.Spec.Containers[0].Env = append(
			pod.Spec.Containers[0].Env,
			v1.EnvVar{
				Name: "BRIGADE_REPO_KEY",
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: project.ID,
						},
						Key: "sshKey",
					},
				},
			},
		)
	}
	if project.Repo.Token != "" {
		pod.Spec.Containers[0].Env = append(
			pod.Spec.Containers[0].Env,
			v1.EnvVar{
				Name: "BRIGADE_REPO_AUTH_TOKEN",
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: project.ID,
						},
						Key: "github.token",
					},
				},
			},
		)
	}
	if project.Kubernetes.VCSSidecarResourcesLimitsCPU != "" {
		cpuQuantity, err := resource.ParseQuantity(
			project.Kubernetes.VCSSidecarResourcesLimitsCPU,
		)
		if err != nil {
			return err
		}
		pod.Spec.Containers[0].Resources.Limits["cpu"] = cpuQuantity
	}
	if project.Kubernetes.VCSSidecarResourcesLimitsMemory != "" {
		memoryQuantity, err := resource.ParseQuantity(
			project.Kubernetes.VCSSidecarResourcesLimitsMemory,
		)
		if err != nil {
			return err
		}
		pod.Spec.Containers[0].Resources.Limits["memory"] = memoryQuantity
	}
	if project.Kubernetes.VCSSidecarResourcesRequestsCPU != "" {
		cpuQuantity, err := resource.ParseQuantity(
			project.Kubernetes.VCSSidecarResourcesRequestsCPU,
		)
		if err != nil {
			return err
		}
		pod.Spec.Containers[0].Resources.Requests["cpu"] = cpuQuantity
	}
	if project.Kubernetes.VCSSidecarResourcesRequestsMemory != "" {
		memoryQuantity, err := resource.ParseQuantity(
			project.Kubernetes.VCSSidecarResourcesRequestsMemory,
		)
		if err != nil {
			return err
		}
		pod.Spec.Containers[0].Resources.Requests["memory"] = memoryQuantity
	}

	_, err := e.kubeClient.CoreV1().Pods(project.Kubernetes.Namespace).Create(pod)
	if err != nil {
		return errors.Wrapf(
			err,
			"error creating source clone pod for pipeline \"%s\"",
			pipelineName,
		)
	}

	podsWatcher, err :=
		e.kubeClient.CoreV1().Pods(project.Kubernetes.Namespace).Watch(
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
	// TODO: This probably should not be hard-coded
	timer := time.NewTimer(10 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case event := <-podsWatcher.ResultChan():
			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				return errors.New(
					"received unexpected object when watching source clone pod for " +
						"completion")
			}
			if pod.Status.Phase == v1.PodFailed {
				return errors.New("source clone pod failed")
			}
			if pod.Status.Phase == v1.PodSucceeded {
				return nil
			}
		case <-timer.C:
			return errors.New("timed out waiting for source clone pod to complete")
		case <-ctx.Done():
			return nil
		}
	}
}
