package executor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "test"

func TestWaitForSourceClonePodCompletionWithPodCompleted(t *testing.T) {
	const podName = "foo"

	testCases := []struct {
		name       string
		phase      v1.PodPhase
		assertions func(*testing.T, error)
	}{
		{
			name:  "pod failure",
			phase: v1.PodFailed,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "source clone pod failed")
			},
		},
		{
			name:  "pod success",
			phase: v1.PodSucceeded,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pod := newRunningTestPod(podName)
			kubeClient := fake.NewSimpleClientset(pod)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			errCh := make(chan error)
			go func() {
				errCh <- waitForSourceClonePodCompletion(
					ctx,
					testNamespace,
					podName,
					time.Minute,
					kubeClient,
				)
			}()
			// This isn't ideal, but we need to wait a moment to make sure the pod
			// watcher in the above goroutine is up and running before we proceed with
			// trying to modify the status of the pod it's watching.
			<-time.After(2 * time.Second)
			pod.Status.Phase = testCase.phase
			_, err := kubeClient.CoreV1().Pods(testNamespace).Update(pod)
			require.NoError(t, err)
			select {
			case err := <-errCh:
				testCase.assertions(t, err)
			case <-time.After(3 * time.Second):
				require.Fail(
					t,
					"timed out waiting for pod completion to be acknowledged",
				)
			}
		})
	}
}

func TestWaitForSourceClonePodCompletionWithPodThatTimesOut(
	t *testing.T,
) {
	const podName = "foo"
	pod := newRunningTestPod(podName)
	kubeClient := fake.NewSimpleClientset(pod)
	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		errCh <- waitForSourceClonePodCompletion(
			ctx,
			testNamespace,
			podName,
			time.Second, // A short timeout on the watch
			kubeClient,
		)
	}()
	select {
	case err := <-errCh:
		require.Error(t, err)
		require.Equal(
			t,
			"timed out waiting for source clone pod to complete",
			err.Error(),
		)
	case <-time.After(5 * time.Second):
		require.Fail(t, "test timed out waiting for the watcher to time out")
	}
}

func TestWaitForSourceClonePodCompletionWithContextCanceled(
	t *testing.T,
) {
	const podName = "foo"
	pod := newRunningTestPod(podName)
	kubeClient := fake.NewSimpleClientset(pod)
	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		errCh <- waitForSourceClonePodCompletion(
			ctx,
			testNamespace,
			podName,
			time.Minute,
			kubeClient,
		)
	}()
	cancel()
	select {
	case err := <-errCh:
		require.Error(t, err)
		require.Equal(t, ctx.Err(), err)
	case <-time.After(5 * time.Second):
		require.Fail(
			t,
			"timed out waiting for the watcher to exit due to canceled context",
		)
	}
}

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

func TestBuildSourceClonePod(t *testing.T) {
	testCases := []struct {
		name       string
		project    brigade.Project
		assertions func(*testing.T, brigade.Project, *v1.Pod, error)
	}{
		{
			name: "base case",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace: testNamespace,
				},
			},
			assertions: func(t *testing.T, _ brigade.Project, _ *v1.Pod, err error) {
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
				pod *v1.Pod,
				err error,
			) {
				require.NoError(t, err)
				require.Contains(
					t,
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
				pod *v1.Pod,
				err error,
			) {
				require.NoError(t, err)
				require.Contains(
					t,
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
				pod *v1.Pod,
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
				pod *v1.Pod,
				err error,
			) {
				require.NoError(t, err)
				_, ok := pod.Spec.Containers[0].Resources.Limits["cpu"]
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
				pod *v1.Pod,
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
				pod *v1.Pod,
				err error,
			) {
				require.NoError(t, err)
				_, ok := pod.Spec.Containers[0].Resources.Limits["memory"]
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
				pod *v1.Pod,
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
				pod *v1.Pod,
				err error,
			) {
				require.NoError(t, err)
				_, ok := pod.Spec.Containers[0].Resources.Requests["cpu"]
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
				pod *v1.Pod,
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
				pod *v1.Pod,
				err error,
			) {
				require.NoError(t, err)
				_, ok := pod.Spec.Containers[0].Resources.Requests["memory"]
				require.True(t, ok)
			},
		},
	}

	event := brigade.Event{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pod, err := buildSourceClonePod(
				testCase.project,
				event,
				"foo",
			)
			testCase.assertions(t, testCase.project, pod, err)
		})
	}
}
