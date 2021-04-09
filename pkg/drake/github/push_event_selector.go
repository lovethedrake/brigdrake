package github

import (
	"encoding/json"
	"log"
	"regexp"

	"github.com/google/go-github/v33/github"
	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/pkg/errors"
)

var (
	branchRefRegex = regexp.MustCompile("refs/heads/(.+)")
	tagRefRegex    = regexp.MustCompile("refs/tags/(.+)")
)

type pushEventSelector struct {
	BranchSelector *refSelector `json:"branches,omitempty"`
	TagSelector    *refSelector `json:"tags,omitempty"`
}

func (p *pushEventSelector) matches(event brigade.Event) (bool, error) {
	pe := github.PushEvent{}
	if err := json.Unmarshal([]byte(event.Payload), &pe); err != nil {
		return false, errors.Wrap(err, "error unmarshaling event payload")
	}
	var fullRef string
	if pe.Ref != nil {
		fullRef = *pe.Ref
	}
	var refSelector *refSelector
	var ref string
	if refSubmatches :=
		branchRefRegex.FindStringSubmatch(fullRef); len(refSubmatches) == 2 {
		refSelector = p.BranchSelector
		ref = refSubmatches[1]
	}
	if refSelector == nil {
		if refSubmatches :=
			tagRefRegex.FindStringSubmatch(fullRef); len(refSubmatches) == 2 {
			refSelector = p.TagSelector
			ref = refSubmatches[1]
		}
	}
	if refSelector == nil {
		log.Printf("no applicable selector found for ref %q", fullRef)
		return false, nil
	}
	match, err := refSelector.matches(ref)
	if err != nil {
		return false, errors.Wrapf(
			err,
			"error matching ref %q to selector",
			fullRef,
		)
	}
	return match, nil
}
