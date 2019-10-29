package executor

import (
	"strconv"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func buildSourceCloneContainer(
	project brigade.Project,
	event brigade.Event,
) (v1.Container, error) {
	const srcDir = "/src"
	container := v1.Container{
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
				Name:      sourceVolumeName,
				MountPath: srcDir,
			},
		},
		Resources: v1.ResourceRequirements{
			Limits:   v1.ResourceList{},
			Requests: v1.ResourceList{},
		},
	}
	if project.Repo.SSHKey != "" {
		container.Env = append(
			container.Env,
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
		container.Env = append(
			container.Env,
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
			return container, err
		}
		container.Resources.Limits["cpu"] = cpuQuantity
	}
	if project.Kubernetes.VCSSidecarResourcesLimitsMemory != "" {
		memoryQuantity, err := resource.ParseQuantity(
			project.Kubernetes.VCSSidecarResourcesLimitsMemory,
		)
		if err != nil {
			return container, err
		}
		container.Resources.Limits["memory"] = memoryQuantity
	}
	if project.Kubernetes.VCSSidecarResourcesRequestsCPU != "" {
		cpuQuantity, err := resource.ParseQuantity(
			project.Kubernetes.VCSSidecarResourcesRequestsCPU,
		)
		if err != nil {
			return container, err
		}
		container.Resources.Requests["cpu"] = cpuQuantity
	}
	if project.Kubernetes.VCSSidecarResourcesRequestsMemory != "" {
		memoryQuantity, err := resource.ParseQuantity(
			project.Kubernetes.VCSSidecarResourcesRequestsMemory,
		)
		if err != nil {
			return container, err
		}
		container.Resources.Requests["memory"] = memoryQuantity
	}
	return container, nil
}
