package brigade

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brigadecore/brigade-github-app/pkg/check"
	"github.com/brigadecore/brigade-github-app/pkg/webhook"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

func notifyCheckStart(payload *webhook.Payload, name, title string) error {
	token := payload.Token
	repo, commit, branch, err := repoCommitBranch(payload)
	if err != nil {
		return err
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return errors.New("Error: CheckSuite.Repository.FullName is required")
	}
	run := check.Run{
		Name:       name,
		HeadBranch: branch,
		HeadSHA:    commit,
		StartedAt:  time.Now().Format(check.RFC8601),
		Output: check.Output{
			Title: title,
		},
		Status: "in_progress",
	}
	return notifyGithub(run, parts[0], parts[1], token)
}

func notifyCheckCompleted(
	payload *webhook.Payload,
	name string,
	title string,
	conclusion string,
) error {
	token := payload.Token
	repo, commit, branch, err := repoCommitBranch(payload)
	if err != nil {
		return err
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return errors.New("Error: CheckSuite.Repository.FullName is required")
	}
	run := check.Run{
		Name:       name,
		HeadBranch: branch,
		HeadSHA:    commit,
		Output: check.Output{
			Title: title,
		},
		Status:      "completed",
		CompletedAt: time.Now().Format(check.RFC8601),
		Conclusion:  conclusion,
	}
	return notifyGithub(run, parts[0], parts[1], token)
}

func notifyGithub(run check.Run, owner, repo, token string) error {
	// Once we have the token, we can switch from the app token to the
	// installation token.
	ghc, err := webhook.InstallationTokenClient(token, "", "")
	if err != nil {
		return err
	}
	ct := &checkTool{
		client: ghc,
		owner:  owner,
		repo:   repo,
	}
	return ct.createRun(run)
}

type checkTool struct {
	client *github.Client
	owner  string
	repo   string
}

func (c *checkTool) createRun(cr check.Run) error {
	u := fmt.Sprintf("repos/%s/%s/check-runs", c.owner, c.repo)
	req, err := c.client.NewRequest("POST", u, cr)
	if err != nil {
		return err
	}
	// Turn on beta feature.
	req.Header.Set("Accept", "application/vnd.github.antiope-preview+json")
	ctx := context.Background()
	_, err = c.client.Do(ctx, req, bytes.NewBuffer(nil))
	return err
}

func repoCommitBranch(payload *webhook.Payload) (
	string,
	string,
	string,
	error,
) {
	var repo, commit, branch string
	// As ridiculous as this is, we have to remarshal the Body and unmarshal it
	// into the right object.
	tmp, err := json.Marshal(payload.Body)
	if err != nil {
		return repo, commit, branch, err
	}
	switch payload.Type {
	case "check_run":
		event := &github.CheckRunEvent{}
		if err = json.Unmarshal(tmp, event); err != nil {
			return repo, commit, branch, err
		}
		repo = event.Repo.GetFullName()
		commit = event.CheckRun.CheckSuite.GetHeadSHA()
		branch = event.CheckRun.CheckSuite.GetHeadBranch()
	case "check_suite":
		event := &github.CheckSuiteEvent{}
		if err = json.Unmarshal(tmp, event); err != nil {
			return repo, commit, branch, err
		}
		repo = event.Repo.GetFullName()
		commit = event.CheckSuite.GetHeadSHA()
		branch = event.CheckSuite.GetHeadBranch()
	default:
		return repo, commit, branch,
			fmt.Errorf("unknown payload type %s", payload.Type)
	}
	return repo, commit, branch, nil
}
