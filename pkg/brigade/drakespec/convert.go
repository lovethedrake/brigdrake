package drakespec

import (
	"github.com/brigadecore/brigade/sdk/v2/core"
	"github.com/lovethedrake/drakecore/config"
)

func ToBrigadeJob(jobDef config.Job) core.Job {
	pc := jobDef.PrimaryContainer()
	brigJob := core.Job{
		Name: jobDef.Name(),
		Spec: core.JobSpec{

			PrimaryContainer:  ToBrigadeContainer(pc),
			SidecarContainers: make(map[string]core.JobContainerSpec, len(jobDef.SidecarContainers())),
			TimeoutSeconds:    jobDef.TimeoutSeconds(),
		},
	}

	for _, sc := range jobDef.SidecarContainers() {
		brigJob.Spec.SidecarContainers[sc.Name()] = ToBrigadeContainer(sc)
	}

	if jobDef.CPUArch() != "" || jobDef.OSFamily() != "" {
		brigJob.Spec.Host = &core.JobHost{
			OS: string(jobDef.OSFamily()),
		}

		if jobDef.CPUArch() != "" {
			brigJob.Spec.Host.NodeSelector = map[string]string{
				"arch": string(jobDef.CPUArch()),
			}
		}
	}

	return brigJob
}

func ToBrigadeContainer(containterDef config.Container) core.JobContainerSpec {
	return core.JobContainerSpec{
		ContainerSpec: core.ContainerSpec{
			Image:           containterDef.Image(),
			ImagePullPolicy: core.ImagePullPolicy(containterDef.ImagePullPolicy()),
			Command:         containterDef.Command(),
			Arguments:       containterDef.Args(),
			Environment:     containterDef.Environment(),
		},
		WorkingDirectory:    containterDef.WorkingDirectory(),
		WorkspaceMountPath:  containterDef.SharedStorageMountPath(),
		SourceMountPath:     containterDef.SourceMountPath(),
		Privileged:          containterDef.Privileged(),
		UseHostDockerSocket: containterDef.MountDockerSocket(),
	}
}
