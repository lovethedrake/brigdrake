package github

import (
	"encoding/json"
	"log"

	"github.com/google/go-github/github"
	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/pkg/errors"
)

type pullRequestEventSelector struct {
	TargetBranchSelector *refSelector `json:"targetBranches,omitempty"`
}

func (p *pullRequestEventSelector) matches(
	event brigade.Event,
) (bool, error) {
	if p.TargetBranchSelector == nil {
		log.Printf(
			"check suite request event does not match nil target branch selector",
		)
		return false, nil
	}
	pre := github.PullRequestEvent{}
	if err := json.Unmarshal(event.Payload, &pre); err != nil {
		return false, errors.Wrap(err, "error unmarshaling event payload")
	}
	branch := *pre.PullRequest.Base.Ref
	match, err := p.TargetBranchSelector.matches(branch)
	if err != nil {
		return false, errors.Wrap(
			err,
			"error matching branch %q to target branch selector",
		)
	}
	return match, nil
}
