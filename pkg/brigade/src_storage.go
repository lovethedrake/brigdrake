package brigade

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (e *executor) createSrcPVC(
	project Project,
	event Event,
	pipelineName string,
) error {
	storageQuantity, err := resource.ParseQuantity(
		project.Kubernetes.BuildStorageSize,
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error parsing storage quantity %s",
			project.Kubernetes.BuildStorageSize,
		)
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: srcPVCName(event.WorkerID, pipelineName),
			Labels: map[string]string{
				"heritage":  "brigade",
				"component": "buildStorage",
				"project":   project.ID,
				"worker":    strings.ToLower(event.WorkerID),
				"build":     event.BuildID,
				"pipeline":  pipelineName,
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"storage": storageQuantity,
				},
			},
		},
	}
	if project.Kubernetes.BuildStorageClass != "" {
		pvc.Spec.StorageClassName = &project.Kubernetes.BuildStorageClass
	} else if e.workerConfig.DefaultBuildStorageClass != "" {
		pvc.Spec.StorageClassName = &e.workerConfig.DefaultBuildStorageClass
	}
	_, err = e.kubeClient.CoreV1().PersistentVolumeClaims(
		project.Kubernetes.Namespace,
	).Create(pvc)
	if err != nil {
		return errors.Wrapf(
			err,
			"error creating source PVC for pipeline \"%s\"",
			pipelineName,
		)
	}
	return nil
}

func (e *executor) destroySrcPVC(
	project Project,
	event Event,
	pipelineName string,
) error {
	if err := e.kubeClient.CoreV1().PersistentVolumeClaims(
		project.Kubernetes.Namespace,
	).Delete(
		srcPVCName(event.WorkerID, pipelineName),
		&metav1.DeleteOptions{},
	); err != nil {
		return errors.Wrapf(
			err,
			"error deleting source PVC for pipeline \"%s\"",
			pipelineName,
		)
	}
	return nil
}

func srcPVCName(workerID, pipelineName string) string {
	workerIDLower := strings.ToLower(workerID)
	pipelineNameLower := strings.ToLower(pipelineName)
	return fmt.Sprintf("%s-%s", workerIDLower, pipelineNameLower)
}
