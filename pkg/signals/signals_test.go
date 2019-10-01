package signals

import (
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	ctx := Context()
	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	require.NoError(t, err)
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		require.Fail(t, "SIGINT did not cancel context as expected")
	}
}
