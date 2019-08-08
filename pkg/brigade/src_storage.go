package brigade

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (b *buildExecutor) createSrcPVC(pipelineName string) error {
	storageQuantity, err := resource.ParseQuantity(
		b.project.Kubernetes.BuildStorageSize,
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error parsing storage quantity %s",
			b.project.Kubernetes.BuildStorageSize,
		)
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: srcPVCName(b.event.WorkerID, pipelineName),
			Labels: map[string]string{
				"heritage":  "brigade",
				"component": "buildStorage",
				"project":   b.project.ID,
				"worker":    strings.ToLower(b.event.WorkerID),
				"build":     b.event.BuildID,
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
	if b.project.Kubernetes.BuildStorageClass != "" {
		pvc.Spec.StorageClassName = &b.project.Kubernetes.BuildStorageClass
	} else if b.workerConfig.DefaultBuildStorageClass != "" {
		pvc.Spec.StorageClassName = &b.workerConfig.DefaultBuildStorageClass
	}
	_, err = b.kubeClient.CoreV1().PersistentVolumeClaims(
		b.project.Kubernetes.Namespace,
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

func (b *buildExecutor) destroySrcPVC(pipelineName string) error {
	if err := b.kubeClient.CoreV1().PersistentVolumeClaims(
		b.project.Kubernetes.Namespace,
	).Delete(
		srcPVCName(b.event.WorkerID, pipelineName),
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
