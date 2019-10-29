package executor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testNamespace = "test"

func newRunningTestPod(name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test-container",
				},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}
}

func TestBuildSourceCloneContainer(t *testing.T) {
	testCases := []struct {
		name       string
		project    brigade.Project
		assertions func(*testing.T, brigade.Project, v1.Container, error)
	}{
		{
			name: "base case",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace: testNamespace,
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				_ v1.Container,
				err error,
			) {
				require.NoError(t, err)
			},
		},
		{
			name: "with project repo ssh key specified",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace: testNamespace,
				},
				Repo: brigade.Repository{
					SSHKey: "foo",
				},
			},
			assertions: func(
				t *testing.T,
				project brigade.Project,
				container v1.Container,
				err error,
			) {
				require.NoError(t, err)
				require.Contains(
					t,
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
			},
		},
		{
			name: "with project repo auth token specified",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace: testNamespace,
				},
				Repo: brigade.Repository{
					Token: "foo",
				},
			},
			assertions: func(
				t *testing.T,
				project brigade.Project,
				container v1.Container,
				err error,
			) {
				require.NoError(t, err)
				require.Contains(
					t,
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
			},
		},
		{
			name: "with invalid vcs sidecar cpu limit",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:                    testNamespace,
					VCSSidecarResourcesLimitsCPU: "You can't parse this... do do do do",
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				_ v1.Container,
				err error,
			) {
				require.Error(t, err)
			},
		},
		{
			name: "with valid vcs sidecar cpu limit",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:                    testNamespace,
					VCSSidecarResourcesLimitsCPU: "100m",
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				container v1.Container,
				err error,
			) {
				require.NoError(t, err)
				_, ok := container.Resources.Limits["cpu"]
				require.True(t, ok)
			},
		},
		{
			name: "with invalid vcs sidecar memory limit",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:                       testNamespace,
					VCSSidecarResourcesLimitsMemory: "You can't parse this... do do do do", // nolint: lll
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				_ v1.Container,
				err error,
			) {
				require.Error(t, err)
			},
		},
		{
			name: "with valid vcs sidecar memory limit",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:                       testNamespace,
					VCSSidecarResourcesLimitsMemory: "256M",
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				container v1.Container,
				err error,
			) {
				require.NoError(t, err)
				_, ok := container.Resources.Limits["memory"]
				require.True(t, ok)
			},
		},
		{
			name: "with invalid vcs sidecar cpu request",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:                      testNamespace,
					VCSSidecarResourcesRequestsCPU: "You can't parse this... do do do do",
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				_ v1.Container,
				err error,
			) {
				require.Error(t, err)
			},
		},
		{
			name: "with valid vcs sidecar cpu request",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:                      testNamespace,
					VCSSidecarResourcesRequestsCPU: "100m",
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				container v1.Container,
				err error,
			) {
				require.NoError(t, err)
				_, ok := container.Resources.Requests["cpu"]
				require.True(t, ok)
			},
		},
		{
			name: "with invalid vcs sidecar memory request",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:                         testNamespace,
					VCSSidecarResourcesRequestsMemory: "You can't parse this... do do do do", // nolint: lll
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				_ v1.Container,
				err error,
			) {
				require.Error(t, err)
			},
		},
		{
			name: "with valid vcs sidecar memory request",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:                         testNamespace,
					VCSSidecarResourcesRequestsMemory: "256M",
				},
			},
			assertions: func(
				t *testing.T,
				_ brigade.Project,
				container v1.Container,
				err error,
			) {
				require.NoError(t, err)
				_, ok := container.Resources.Requests["memory"]
				require.True(t, ok)
			},
		},
	}

	event := brigade.Event{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container, err := buildSourceCloneContainer(testCase.project, event)
			testCase.assertions(t, testCase.project, container, err)
		})
	}
}
