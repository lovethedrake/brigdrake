package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/brigadecore/brigade/sdk/v2/core"
	"github.com/stretchr/testify/require"
)

func TestWaitForJobPodCompletionWithPodCompleted(t *testing.T) {
	job := core.Job{
		Name: "foo",
	}
	testCases := []struct {
		name       string
		state      core.JobPhase
		assertions func(*testing.T, error)
	}{
		{
			name:  "job failure",
			state: core.JobPhaseFailed,
			assertions: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(
					t,
					fmt.Sprintf("job %s did not succeed, ended in state %s", job.Name, core.JobPhaseFailed),
					err.Error(),
				)
			},
		},
		{
			name:  "job success",
			state: core.JobPhaseSucceeded,
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			jobStatus := make(chan core.JobStatus)
			jobErr := make(chan error)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			errCh := make(chan error)
			go func() {
				errCh <- waitForJobCompletion(
					ctx,
					job,
					jobStatus,
					jobErr,
				)
			}()
			jobStatus <- core.JobStatus{
				Phase: testCase.state,
			}

			select {
			case err := <-errCh:
				testCase.assertions(t, err)
			case <-time.After(3 * time.Second):
				require.Fail(
					t,
					"timed out waiting for job status",
				)
			}
		})
	}
}

func TestWaitForJobPodCompletionWithPodThatTimesOut(t *testing.T) {
	job := core.Job{
		Name: "foo",
		Spec: core.JobSpec{TimeoutSeconds: 1},
	}

	jobStatus := make(chan core.JobStatus)
	jobErr := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error)
	go func() {
		errCh <- waitForJobCompletion(
			ctx,
			job,
			jobStatus,
			jobErr,
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
	job := core.Job{
		Name: "foo",
		Spec: core.JobSpec{TimeoutSeconds: 1},
	}

	jobStatus := make(chan core.JobStatus)
	jobErr := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error)
	go func() {
		errCh <- waitForJobCompletion(
			ctx,
			job,
			jobStatus,
			jobErr,
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
