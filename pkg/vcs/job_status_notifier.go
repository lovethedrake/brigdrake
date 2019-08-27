package vcs

import (
	"github.com/lovethedrake/drakecore/config"
)

// JobStatusNotifier is an interface to be implemented by components that can
// report job status back to the VCS that triggered the build.
type JobStatusNotifier interface {
	SendInProgressNotification(config.Job) error
	SendSuccessNotification(config.Job) error
	SendCancelledNotification(config.Job) error
	SendTimedOutNotification(config.Job) error
	SendFailureNotification(config.Job) error
}