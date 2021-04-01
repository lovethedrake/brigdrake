package github

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/google/go-github/v18/github"
	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/brigdrake/pkg/drake"
	"github.com/pkg/errors"
)

// nolint: lll
type trigger struct {
	PullRequestEventSelector *pullRequestEventSelector `json:"pullRequest,omitempty"`
	PushEventSelector        *pushEventSelector        `json:"push,omitempty"`
}

// NewTriggerFromJSON takes a slice of bytes containing JSON as an argument and
// returns a Trigger that implements the
// github.com/lovethedrake/drakespec-github spec.
func NewTriggerFromJSON(jsonBytes []byte) (drake.Trigger, error) {
	t := &trigger{}
	err := json.Unmarshal(jsonBytes, t)
	return t, err
}

func (t *trigger) Matches(event brigade.Event) (bool, error) {
	if event.Provider != "github" {
		log.Printf(
			"event from provider %q does not match github trigger",
			event.Provider,
		)
		return false, nil
	}

	switch event.Type {
	case "pull_request:opened",
		"pull_request:synchronize",
		"pull_request:reopened":
		if t.PullRequestEventSelector == nil {
			log.Println(
				"pull request event does not match trigger with unconfigured pull " +
					"request event selector",
			)
			return false, nil
		}
		matches, err := t.PullRequestEventSelector.matches(event)
		if err != nil {
			return false, errors.Wrap(
				err,
				"error matching pull request event to pull request event selector",
			)
		}
		if matches {
			log.Println("pull request event matches trigger")
		} else {
			log.Println("pull request event does not match trigger")
		}
		return matches, nil
	case "push":
		if t.PushEventSelector == nil {
			log.Println(
				"push event does not match trigger with unconfigured push event " +
					"selector",
			)
			return false, nil
		}
		matches, err := t.PushEventSelector.matches(event)
		if err != nil {
			return false, errors.Wrap(
				err,
				"error matching push event to push event selector ",
			)
		}
		if matches {
			log.Println("push event matches trigger")
		} else {
			log.Println("push event does not match trigger")
		}
		return matches, nil
	default:
		log.Printf(
			"unsupported event type %q does not match github trigger",
			event.Type,
		)
		return false, nil
	}
}

func (t *trigger) JobStatusNotifier(
	project brigade.Project, event brigade.Event,
) (drake.JobStatusNotifier, error) {
	appIDStr, ok := project.Secrets["BRIGDRAKE_GITHUB_APP_ID"]
	if !ok {
		return nil, nil
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, nil
	}
	githubKey, ok := project.Secrets["BRIGDRAKE_GITHUB_KEY"]
	if !ok {
		return nil, nil
	}
	switch event.Type {
	case "pull_request:opened",
		"pull_request:synchronize",
		"pull_request:reopened":
		pre := github.PullRequestEvent{}
		if err := json.Unmarshal(event.Payload, &pre); err != nil {
			return nil, errors.Wrap(err, "error unmarshaling event payload")
		}
		return newJobStatusNotifier(
			appID,
			*pre.Installation.ID,
			githubKey,
			*pre.PullRequest.Base.Repo.Owner.Login,
			*pre.PullRequest.Base.Repo.Name,
			*pre.PullRequest.Head.SHA,
		)
	case "push":
		pe := github.PushEvent{}
		if err := json.Unmarshal(event.Payload, &pe); err != nil {
			return nil, errors.Wrap(err, "error unmarshaling event payload")
		}
		// We don't want a notifier if this is for a tag push
		if tagRefRegex.MatchString(*pe.Ref) {
			return nil, nil
		}
		return newJobStatusNotifier(
			appID,
			*pe.Installation.ID,
			githubKey,
			*pe.Repo.Owner.Login,
			*pe.Repo.Name,
			*pe.HeadCommit.ID,
		)
	}
	return nil, nil
}
