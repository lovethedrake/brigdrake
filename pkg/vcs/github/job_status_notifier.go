package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brigadecore/brigade-github-app/pkg/check"
	"github.com/brigadecore/brigade-github-app/pkg/webhook"
	"github.com/google/go-github/github"
	"github.com/lovethedrake/brigdrake/pkg/vcs"
	"github.com/lovethedrake/drakecore/config"
)

type jobStatusNotifier struct {
	owner        string
	repo         string
	commit       string
	branch       string
	githubClient *github.Client
}

// NewJobStatusNotifier returns an implementation of the vcs.JobStatusNotifier
// interface that is capable of sending job status notifications to GitHub.
func NewJobStatusNotifier(
	payload *webhook.Payload,
) (vcs.JobStatusNotifier, error) {
	jsn := &jobStatusNotifier{}
	var err error
	if jsn.githubClient, err =
		webhook.InstallationTokenClient(payload.Token, "", ""); err != nil {
		return nil, err
	}
	jsn.owner, jsn.repo, jsn.commit, jsn.branch, err =
		ownerRepoCommitBranch(payload)
	return jsn, err
}

func (j *jobStatusNotifier) SendInProgressNotification(job config.Job) error {
	run := check.Run{
		Name:       job.Name(),
		HeadBranch: j.branch,
		HeadSHA:    j.commit,
		StartedAt:  time.Now().Format(check.RFC8601),
		Output: check.Output{
			Title: job.Name(),
		},
		Status: "in_progress",
	}
	return j.notifyGithub(run)
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
	run := check.Run{
		Name:       job.Name(),
		HeadBranch: j.branch,
		HeadSHA:    j.commit,
		Output: check.Output{
			Title: job.Name(),
		},
		Status:      "completed",
		CompletedAt: time.Now().Format(check.RFC8601),
		Conclusion:  conclusion,
	}
	return j.notifyGithub(run)
}

func (j *jobStatusNotifier) notifyGithub(run check.Run) error {
	u := fmt.Sprintf("repos/%s/%s/check-runs", j.owner, j.repo)
	req, err := j.githubClient.NewRequest("POST", u, run)
	if err != nil {
		return err
	}
	// Turn on beta feature.
	req.Header.Set("Accept", "application/vnd.github.antiope-preview+json")
	ctx := context.Background()
	_, err = j.githubClient.Do(ctx, req, bytes.NewBuffer(nil))
	return err
}

func ownerRepoCommitBranch(payload *webhook.Payload) (
	string,
	string,
	string,
	string,
	error,
) {
	var owner, repo, commit, branch string
	// As ridiculous as this is, we have to remarshal the Body and unmarshal it
	// into the right object.
	tmp, err := json.Marshal(payload.Body)
	if err != nil {
		return owner, repo, commit, branch, err
	}
	switch payload.Type {
	case "check_run":
		event := &github.CheckRunEvent{}
		if err = json.Unmarshal(tmp, event); err != nil {
			return owner, repo, commit, branch, err
		}
		owner = *event.Repo.Owner.Login
		repo = event.Repo.GetName()
		commit = event.CheckRun.CheckSuite.GetHeadSHA()
		branch = event.CheckRun.CheckSuite.GetHeadBranch()
	case "check_suite":
		event := &github.CheckSuiteEvent{}
		if err = json.Unmarshal(tmp, event); err != nil {
			return "", repo, commit, branch, err
		}
		owner = *event.Repo.Owner.Login
		repo = event.Repo.GetName()
		commit = event.CheckSuite.GetHeadSHA()
		branch = event.CheckSuite.GetHeadBranch()
	default:
		return owner, repo, commit, branch,
			fmt.Errorf("unknown payload type %s", payload.Type)
	}
	return owner, repo, commit, branch, nil
}
