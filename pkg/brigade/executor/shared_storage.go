package executor

import (
	"fmt"
	"strings"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func createSharedStoragePVC(
	project brigade.Project,
	event brigade.Event,
	workerConfig brigade.WorkerConfig,
	pipelineName string,
	kubeClient kubernetes.Interface,
) error {
	pvc, err := buildSharedStoragePVC(project, event, workerConfig, pipelineName)
	if err != nil {
		return err
	}
	_, err = kubeClient.CoreV1().PersistentVolumeClaims(
		project.Kubernetes.Namespace,
	).Create(pvc)
	if err != nil {
		return errors.Wrapf(
			err,
			"error creating source PVC for pipeline %q",
			pipelineName,
		)
	}
	return nil
}

func buildSharedStoragePVC(
	project brigade.Project,
	event brigade.Event,
	workerConfig brigade.WorkerConfig,
	pipelineName string,
) (*v1.PersistentVolumeClaim, error) {
	storageQuantity, err := resource.ParseQuantity(
		project.Kubernetes.BuildStorageSize,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error parsing storage quantity %s",
			project.Kubernetes.BuildStorageSize,
		)
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: sharedStoragePVCName(event.WorkerID, pipelineName),
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
	} else if workerConfig.DefaultBuildStorageClass != "" {
		pvc.Spec.StorageClassName = &workerConfig.DefaultBuildStorageClass
	}
	return pvc, nil
}

func destroySharedStoragePVC(
	project brigade.Project,
	event brigade.Event,
	pipelineName string,
	kubeClient kubernetes.Interface,
) error {
	if err := kubeClient.CoreV1().PersistentVolumeClaims(
		project.Kubernetes.Namespace,
	).Delete(
		sharedStoragePVCName(event.WorkerID, pipelineName),
		&metav1.DeleteOptions{},
	); err != nil {
		return errors.Wrapf(
			err,
			"error deleting source PVC for pipeline %q",
			pipelineName,
		)
	}
	return nil
}

// sharedStoragePVCName permits all callers who need to reference the shared
// storage PVC by name to reliably use the correct name as long as they have the
// workerID and pipelineName.
func sharedStoragePVCName(workerID, pipelineName string) string {
	workerIDLower := strings.ToLower(workerID)
	pipelineNameLower := strings.ToLower(pipelineName)
	return fmt.Sprintf("%s-%s", workerIDLower, pipelineNameLower)
}
