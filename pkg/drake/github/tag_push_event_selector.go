package github

import (
	"encoding/json"
	"log"
	"regexp"

	"github.com/google/go-github/github"
	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/pkg/errors"
)

var tagRefRegex = regexp.MustCompile("refs/tags/(.+)")

type tagPushEventSelector struct {
	TagSelector *refSelector `json:"tags,omitempty"`
}

func (t *tagPushEventSelector) matches(event brigade.Event) (bool, error) {
	if t.TagSelector == nil {
		log.Printf("push event does not match nil tag selector")
		return false, nil
	}
	pe := github.PushEvent{}
	if err := json.Unmarshal(event.Payload, &pe); err != nil {
		return false, errors.Wrap(err, "error unmarshaling event payload")
	}
	var ref string
	if pe.Ref != nil {
		ref = *pe.Ref
	}
	refSubmatches := tagRefRegex.FindStringSubmatch(ref)
	if len(refSubmatches) != 2 {
		log.Println(
			"push event that isn't for a new tag does not match tag push event " +
				"selector",
		)
		return false, nil
	}
	match, err := t.TagSelector.matches(refSubmatches[1])
	if err != nil {
		return false, errors.Wrap(err, "error matching tag %q to tag selector")
	}
	return match, nil
}
