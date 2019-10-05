package executor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pkg/errors"
)

func TestMultiError(t *testing.T) {
	err := &multiError{
		errs: []error{
			errors.New("foo"),
			errors.New("bar"),
		},
	}
	errStr := err.Error()
	require.Contains(t, errStr, "2 errors encountered:")
	require.Contains(t, errStr, "foo")
	require.Contains(t, errStr, "bar")
}

func TestTimedOutError(t *testing.T) {
	const jobName = "foo"
	err := timedOutError{
		job: "foo",
	}
	require.Contains(t, err.Error(), jobName)
}

func TestPendingJobCanceledError(t *testing.T) {
	const jobName = "foo"
	err := pendingJobCanceledError{
		job: "foo",
	}
	require.Contains(t, err.Error(), jobName)
}

func TestInProgressJobAbortedError(t *testing.T) {
	const jobName = "foo"
	err := inProgressJobAbortedError{
		job: "foo",
	}
	require.Contains(t, err.Error(), jobName)
}
