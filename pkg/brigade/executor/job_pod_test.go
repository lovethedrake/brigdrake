package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/drakecore/config"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWaitForJobPodCompletionWithPodCompleted(t *testing.T) {
	const jobName = "foo"
	const podName = "bar"

	testCases := []struct {
		name           string
		containerState v1.ContainerState
		assertions     func(*testing.T, error)
	}{
		{
			name: "pod failure",
			containerState: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{
					Reason: "Failed",
				},
			},
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					fmt.Sprintf("pod %q failed", podName),
					err.Error(),
				)
			},
		},
		{
			name: "pod success",
			containerState: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{
					Reason: "Completed",
				},
			},
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
				errCh <- waitForJobPodCompletion(
					ctx,
					testNamespace,
					jobName,
					podName,
					time.Minute,
					kubeClient,
				)
			}()
			// This isn't ideal, but we need to wait a moment to make sure the pod
			// watcher in the above goroutine is up and running before we proceed with
			// trying to modify the status of the pod it's watching.
			<-time.After(2 * time.Second)
			pod.Status.ContainerStatuses = []v1.ContainerStatus{
				{
					Name:  pod.Spec.Containers[0].Name,
					State: testCase.containerState,
				},
			}
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

func TestWaitForJobPodCompletionWithPodThatTimesOut(t *testing.T) {
	const jobName = "foo"
	const podName = "bar"
	pod := newRunningTestPod(podName)
	kubeClient := fake.NewSimpleClientset(pod)
	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		errCh <- waitForJobPodCompletion(
			ctx,
			testNamespace,
			jobName,
			podName,
			time.Second, // A short timeout on the watch
			kubeClient,
		)
	}()
	select {
	case err := <-errCh:
		require.Error(t, err)
		require.IsType(t, &timedOutError{}, err)
	case <-time.After(5 * time.Second):
		require.Fail(t, "test timed out waiting for the watcher to time out")
	}
}

func TestWaitForJobPodCompletionWithContextCanceled(t *testing.T) {
	const jobName = "foo"
	const podName = "bar"
	pod := newRunningTestPod(podName)
	kubeClient := fake.NewSimpleClientset(pod)
	errCh := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		errCh <- waitForJobPodCompletion(
			ctx,
			testNamespace,
			jobName,
			podName,
			time.Minute,
			kubeClient,
		)
	}()
	cancel()
	select {
	case err := <-errCh:
		require.Error(t, err)
		require.IsType(t, &inProgressJobAbortedError{}, err)
	case <-time.After(5 * time.Second):
		require.Fail(
			t,
			"timed out waiting for the watcher to exit due to canceled context",
		)
	}
}

func TestBuildJobPod(t *testing.T) {
	testCases := []struct {
		name       string
		project    brigade.Project
		assertions func(*testing.T, *v1.Pod, error)
	}{
		{
			name: "base case",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace: testNamespace,
				},
			},
			assertions: func(t *testing.T, _ *v1.Pod, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "with image pull secrets",
			project: brigade.Project{
				Kubernetes: brigade.KubernetesConfig{
					Namespace:        testNamespace,
					ImagePullSecrets: []string{"foo", "bar"},
				},
			},
			assertions: func(t *testing.T, pod *v1.Pod, err error) {
				require.NoError(t, err)
				require.Len(t, pod.Spec.ImagePullSecrets, 2)
			},
		},
	}

	event := brigade.Event{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pod, err := buildJobPod(
				testCase.project,
				event,
				"foo",
				&fakeJob{
					name: "bar",
					containers: []config.Container{
						&fakeContainer{
							name: "bat",
						},
						&fakeContainer{
							name: "baz",
						},
					},
				},
			)
			testCase.assertions(t, pod, err)
		})
	}
}

func TestBuildJobPodContainer(t *testing.T) {
	testCases := []struct {
		name         string
		project      brigade.Project
		containerCfg config.Container
		assertions   func(*testing.T, v1.Container, error)
	}{
		{
			name:    "base case",
			project: brigade.Project{},
			containerCfg: &fakeContainer{
				name: "foo",
			},
			assertions: func(_ *testing.T, _ v1.Container, _ error) {},
		},
		{
			name: "privileged container requested but not allowed by project",
			project: brigade.Project{
				AllowPrivilegedJobs: false,
			},
			containerCfg: &fakeContainer{
				name:       "foo",
				privileged: true,
			},
			assertions: func(t *testing.T, _ v1.Container, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "requested to be privileged")
			},
		},
		{
			name: "mounting docker socket requested but not allowed by project",
			project: brigade.Project{
				AllowHostMounts: false,
			},
			containerCfg: &fakeContainer{
				name:              "foo",
				mountDockerSocket: true,
			},
			assertions: func(t *testing.T, _ v1.Container, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "requested to mount the docker socket")
			},
		},
		{
			name: "with project secrets",
			project: brigade.Project{
				Secrets: map[string]string{
					"foo": "bar",
					"bat": "baz",
				},
			},
			containerCfg: &fakeContainer{
				name: "foo",
			},
			assertions: func(t *testing.T, container v1.Container, err error) {
				require.NoError(t, err)
				require.Len(t, container.Env, 2)
			},
		},
		{
			name:    "with container env vars",
			project: brigade.Project{},
			containerCfg: &fakeContainer{
				name: "foo",
				environment: []string{
					"foo=bar",
					"bat", // This is valid
				},
			},
			assertions: func(t *testing.T, container v1.Container, err error) {
				require.NoError(t, err)
				require.Len(t, container.Env, 2)
			},
		},
		{
			name:    "with source mount path",
			project: brigade.Project{},
			containerCfg: &fakeContainer{
				name:            "foo",
				sourceMountPath: "/app",
			},
			assertions: func(t *testing.T, container v1.Container, err error) {
				require.NoError(t, err)
				require.Len(t, container.VolumeMounts, 1)
			},
		},
		{
			name: "with docker socket mounted",
			project: brigade.Project{
				AllowHostMounts: true,
			},
			containerCfg: &fakeContainer{
				name:              "foo",
				mountDockerSocket: true,
			},
			assertions: func(t *testing.T, container v1.Container, err error) {
				require.NoError(t, err)
				require.Len(t, container.VolumeMounts, 1)
			},
		},
	}

	event := brigade.Event{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container, err := buildJobPodContainer(
				testCase.project,
				event,
				testCase.containerCfg,
			)
			testCase.assertions(t, container, err)
		})
	}
}

type fakeJob struct {
	name       string
	containers []config.Container
}

func (f *fakeJob) Name() string {
	return f.name
}

func (f *fakeJob) Containers() []config.Container {
	return f.containers
}

type fakeContainer struct {
	name              string
	image             string
	environment       []string
	workingDirectory  string
	command           string
	tty               bool
	privileged        bool
	mountDockerSocket bool
	sourceMountPath   string
}

func (f *fakeContainer) Name() string {
	return f.name
}

func (f *fakeContainer) Image() string {
	return f.image
}

func (f *fakeContainer) Environment() []string {
	return f.environment
}

func (f *fakeContainer) WorkingDirectory() string {
	return f.workingDirectory
}

func (f *fakeContainer) Command() string {
	return f.command
}

func (f *fakeContainer) TTY() bool {
	return f.tty
}

func (f *fakeContainer) Privileged() bool {
	return f.privileged
}

func (f *fakeContainer) MountDockerSocket() bool {
	return f.mountDockerSocket
}

func (f *fakeContainer) SourceMountPath() string {
	return f.sourceMountPath
}