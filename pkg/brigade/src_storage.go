package brigade

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *pipelineExecutor) createSrcPVC(pipelineName string) error {
	storageQuantity, err := resource.ParseQuantity(
		p.project.Kubernetes.BuildStorageSize,
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error parsing storage quantity %s",
			p.project.Kubernetes.BuildStorageSize,
		)
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: srcPVCName(p.event.WorkerID, pipelineName),
			Labels: map[string]string{
				"heritage":  "brigade",
				"component": "buildStorage",
				"project":   p.project.ID,
				"worker":    strings.ToLower(p.event.WorkerID),
				"build":     p.event.BuildID,
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
	if p.project.Kubernetes.BuildStorageClass != "" {
		pvc.Spec.StorageClassName = &p.project.Kubernetes.BuildStorageClass
	} else if p.workerConfig.DefaultBuildStorageClass != "" {
		pvc.Spec.StorageClassName = &p.workerConfig.DefaultBuildStorageClass
	}
	_, err = p.kubeClient.CoreV1().PersistentVolumeClaims(
		p.project.Kubernetes.Namespace,
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

func (p *pipelineExecutor) destroySrcPVC(pipelineName string) error {
	if err := p.kubeClient.CoreV1().PersistentVolumeClaims(
		p.project.Kubernetes.Namespace,
	).Delete(
		srcPVCName(p.event.WorkerID, pipelineName),
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
