package executor

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreatePVC(t *testing.T) {
	project := brigade.Project{
		Kubernetes: brigade.KubernetesConfig{
			Namespace:        "test",
			BuildStorageSize: "1G",
		},
	}
	event := brigade.Event{}
	workerConfig := brigade.WorkerConfig{}
	kubeClient := fake.NewSimpleClientset()
	err := createSrcPVC(project, event, workerConfig, "foo", kubeClient)
	require.NoError(t, err)
}

func TestBuildSrcPVC(t *testing.T) {
	testCases := []struct {
		name         string
		project      brigade.Project
		workerConfig brigade.WorkerConfig
		assertions   func(*testing.T, *v1.PersistentVolumeClaim, error)
	}{
		{
			name: "invalid src storage size",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:        testNamespace,
					BuildStorageSize: "You can't parse this... do do do do",
				},
			},
			workerConfig: brigade.WorkerConfig{},
			assertions: func(t *testing.T, _ *v1.PersistentVolumeClaim, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "no build storage class specified",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:        testNamespace,
					BuildStorageSize: "1G",
				},
			},
			workerConfig: brigade.WorkerConfig{},
			assertions: func(t *testing.T, pvc *v1.PersistentVolumeClaim, err error) {
				require.NoError(t, err)
				require.Nil(t, pvc.Spec.StorageClassName)
			},
		},
		{
			name: "build storage class specified at project level",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:         testNamespace,
					BuildStorageClass: "nfs",
					BuildStorageSize:  "1G",
				},
			},
			workerConfig: brigade.WorkerConfig{},
			assertions: func(t *testing.T, pvc *v1.PersistentVolumeClaim, err error) {
				require.NoError(t, err)
				require.NotNil(t, pvc.Spec.StorageClassName)
				require.Equal(t, "nfs", *pvc.Spec.StorageClassName)
			},
		},
		{
			name: "build storage class specified at brigade level",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:        testNamespace,
					BuildStorageSize: "1G",
				},
			},
			workerConfig: brigade.WorkerConfig{
				DefaultBuildStorageClass: "nfs",
			},
			assertions: func(t *testing.T, pvc *v1.PersistentVolumeClaim, err error) {
				require.NoError(t, err)
				require.NotNil(t, pvc.Spec.StorageClassName)
				require.Equal(t, "nfs", *pvc.Spec.StorageClassName)
			},
		},
	}

	event := brigade.Event{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pvc, err := buildSrcPVC(
				testCase.project,
				event,
				testCase.workerConfig,
				"foo",
			)
			testCase.assertions(t, pvc, err)
		})
	}
}

func TestDestroySrcPVC(t *testing.T) {
	const pipelineName = "foo"
	project := brigade.Project{
		Kubernetes: brigade.KubernetesConfig{
			Namespace:        testNamespace,
			BuildStorageSize: "1G",
		},
	}
	event := brigade.Event{
		WorkerID: "12356",
	}
	kubeClient := fake.NewSimpleClientset(
		&v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      srcPVCName(event.WorkerID, pipelineName),
			},
		},
	)
	err := destroySrcPVC(project, event, pipelineName, kubeClient)
	require.NoError(t, err)
}

func TestSrcPVCName(t *testing.T) {
	require.Equal(t, "foo-bar", srcPVCName("FOO", "BAR"))
}
