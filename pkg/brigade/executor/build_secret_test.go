package executor

import (
	"testing"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBuildBuildSecret(t *testing.T) {
	buildBuildSecret(
		brigade.Project{},
		brigade.Event{},
	)
	// TODO: What's worth asserting here?
}

func TestDestroyBuildSecret(t *testing.T) {
	const secretName = "foo"
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      secretName,
		},
	}
	kubeClient := fake.NewSimpleClientset(secret)
	err := destroyBuildSecret(
		brigade.Project{
			Kubernetes: brigade.KubernetesConfig{
				Namespace: testNamespace,
			},
		},
		brigade.Event{
			BuildID: "foo",
		},
		kubeClient,
	)
	require.NoError(t, err)
}
