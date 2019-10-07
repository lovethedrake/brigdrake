package github

import (
	"encoding/json"
	"log"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/pkg/errors"
)

type checkSuiteRequestEventSelector struct {
	BranchSelector *refSelector `json:"branches,omitempty"`
}

func (c *checkSuiteRequestEventSelector) matches(
	event brigade.Event,
) (bool, error) {
	if c.BranchSelector == nil {
		log.Printf("check suite request event does not match nil branch selector")
		return false, nil
	}
	var branch string
	csePayloadWrapper := checkSuiteEventPayloadWrapper{}
	if err := json.Unmarshal(event.Payload, &csePayloadWrapper); err != nil {
		return false, errors.Wrap(err, "error unmarshaling event payload")
	}
	cse := csePayloadWrapper.CheckSuiteEvent
	if cse.CheckSuite.HeadBranch != nil {
		branch = *cse.CheckSuite.HeadBranch
	}
	match, err := c.BranchSelector.matches(branch)
	if err != nil {
		return false, errors.Wrap(
			err,
			"error matching branch %q to branch selector",
		)
	}
	return match, nil
}
