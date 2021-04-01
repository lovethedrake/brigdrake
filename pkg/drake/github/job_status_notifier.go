package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v18/github"
	"github.com/lovethedrake/brigdrake/pkg/drake"
	"github.com/lovethedrake/drakecore/config"
	"github.com/pkg/errors"
)

// simpleGithubClient is an interface for a github client that contains
// a subset of the functions from github.Client that we actually use.
// This permits us to much more easily swap in a fake github client during
// testing.
type simpleGithubClient interface {
	NewRequest(string, string, interface{}) (*http.Request, error)
	Do(
		ctx context.Context,
		req *http.Request,
		v interface{},
	) (*github.Response, error)
}

// jobStatusNotifier is an implementation of the drake.JobStatusNotifier
// interface that can report Brigade / Drake job statuses to GitHub as check
// runs.
type jobStatusNotifier struct {
	checkRunsURL string
	commit       string
	githubClient simpleGithubClient
}

// newJobStatusNotifier returns an implementation of the drake.JobStatusNotifier
// interface that can report Brigade / Drake job statuses to GitHub as check
// runs.
func newJobStatusNotifier(
	appID int64,
	installationID int64,
	base64EncodedGithubKey string,
	repoOwner string,
	repoName string,
	commit string,
) (drake.JobStatusNotifier, error) {
	githubKey, err := base64.StdEncoding.DecodeString(base64EncodedGithubKey)
	if err != nil {
		return nil, errors.Wrap(err, "error base64 decoding github key")
	}
	githubClient, err := newClientFromKeyPEM(
		appID,
		installationID,
		[]byte(githubKey),
	)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"error creating github client for job status notifier",
		)
	}
	return &jobStatusNotifier{
		checkRunsURL: fmt.Sprintf("repos/%s/%s/check-runs", repoOwner, repoName),
		commit:       commit,
		githubClient: githubClient,
	}, nil
}

func (j *jobStatusNotifier) SendInProgressNotification(job config.Job) error {
	jobName := job.Name()
	status := "in_progress"
	blankSummary := ""
	return j.notifyGithub(
		github.CheckRun{
			Name:      &jobName,
			HeadSHA:   &j.commit,
			StartedAt: &github.Timestamp{Time: time.Now()},
			Output: &github.CheckRunOutput{
				Title:   &jobName,
				Summary: &blankSummary,
			},
			Status: &status,
		},
	)
}

func (j *jobStatusNotifier) SendSuccessNotification(job config.Job) error {
	return j.sendCompletedNotification(job, "success")
}

func (j *jobStatusNotifier) SendCancelledNotification(job config.Job) error {
	return j.sendCompletedNotification(job, "cancelled")
}

func (j *jobStatusNotifier) SendTimedOutNotification(job config.Job) error {
	return j.sendCompletedNotification(job, "timed_out")
}

func (j *jobStatusNotifier) SendFailureNotification(job config.Job) error {
	return j.sendCompletedNotification(job, "failure")
}

func (j *jobStatusNotifier) sendCompletedNotification(
	job config.Job,
	conclusion string,
) error {
	jobName := job.Name()
	status := "completed"
	blankSummary := ""
	return j.notifyGithub(
		github.CheckRun{
			Name:    &jobName,
			HeadSHA: &j.commit,
			Output: &github.CheckRunOutput{
				Title:   &jobName,
				Summary: &blankSummary,
			},
			Status:      &status,
			CompletedAt: &github.Timestamp{Time: time.Now()},
			Conclusion:  &conclusion,
		},
	)
}

func (j *jobStatusNotifier) notifyGithub(run github.CheckRun) error {
	req, err := j.githubClient.NewRequest("POST", j.checkRunsURL, run)
	if err != nil {
		return err
	}
	// Turn on beta feature.
	req.Header.Set("Accept", "application/vnd.github.antiope-preview+json")
	_, err = j.githubClient.Do(context.TODO(), req, bytes.NewBuffer(nil))
	return err
}
