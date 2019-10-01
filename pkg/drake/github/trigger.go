package github

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/brigdrake/pkg/drake"
	"github.com/pkg/errors"
)

var tagRefRegex = regexp.MustCompile("refs/tags/(.*)")

type trigger struct {
	BranchSelector *refSelector `json:"branches,omitempty"`
	TagSelector    *refSelector `json:"tags,omitempty"`
}

type refSelector struct {
	WhitelistedRefs []string `json:"only,omitempty"`
	BlacklistedRefs []string `json:"ignore,omitempty"`
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
		return false, nil
	}
	var branch, tag string
	switch event.Type {
	case "check_suite:requested", "check_suite:rerequested":
		csePayloadWrapper := checkSuiteEventPayloadWrapper{}
		if err := json.Unmarshal(event.Payload, &csePayloadWrapper); err != nil {
			return false, errors.Wrap(err, "error unmarshaling event payload")
		}
		cse := csePayloadWrapper.CheckSuiteEvent
		if cse.CheckSuite.HeadBranch != nil {
			branch = *cse.CheckSuite.HeadBranch
		}
	case "push":
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
			log.Printf(
				"push event that isn't for a new tag does not match trigger %s",
				t.string(),
			)
			return false, nil
		}
		tag = refSubmatches[1]
	default:
		log.Printf(
			"unsupported event type %q does not match trigger %s",
			event.Type,
			t.string(),
		)
		return false, nil
	}

	matchMessages := map[bool]string{
		false: fmt.Sprintf(
			"branch %q, tag %q does not match trigger %s",
			branch,
			tag,
			t.string(),
		),
		true: fmt.Sprintf(
			"branch %q, tag %q matches trigger %s",
			branch,
			tag,
			t.string(),
		),
	}

	// If a tag is specified, we match purely on the basis of the tag. This means
	// "" is not a valid tag.
	if tag != "" {
		if t.TagSelector == nil {
			log.Println(matchMessages[false])
			return false, nil
		}
		matches, err := t.TagSelector.matches(tag)
		if err != nil {
			return false, err
		}
		log.Println(matchMessages[matches])
		return matches, nil
	}

	// Fall back to matching on the branch. "" is considered a valid branch.
	// It's basically a PR from a fork.
	if t.BranchSelector == nil {
		log.Println(matchMessages[false])
		return false, nil
	}
	matches, err := t.BranchSelector.matches(branch)
	if err != nil {
		return false, err
	}
	log.Println(matchMessages[matches])
	return matches, nil
}

func (t *trigger) JobStatusNotifier(
	event brigade.Event,
) (drake.JobStatusNotifier, error) {
	switch event.Type {
	case "check_suite:requested", "check_suite:rerequested":
		csePayloadWrapper := checkSuiteEventPayloadWrapper{}
		if err := json.Unmarshal(event.Payload, &csePayloadWrapper); err != nil {
			return nil, errors.Wrap(err, "error unmarshaling event payload")
		}
		return newJobStatusNotifier(csePayloadWrapper), nil
	}
	return nil, nil
}

func (r *refSelector) matches(ref string) (bool, error) {
	var matchesWhitelist bool
	if len(r.WhitelistedRefs) == 0 {
		matchesWhitelist = true
	} else {
		for _, whitelistedRef := range r.WhitelistedRefs {
			var err error
			matchesWhitelist, err = refMatch(ref, whitelistedRef)
			if err != nil {
				return false, err
			}
			if matchesWhitelist {
				break
			}
		}
	}
	var matchesBlacklist bool
	for _, blacklistedRef := range r.BlacklistedRefs {
		var err error
		matchesBlacklist, err = refMatch(ref, blacklistedRef)
		if err != nil {
			return false, err
		}
		if matchesBlacklist {
			break
		}
	}
	return matchesWhitelist && !matchesBlacklist, nil
}

func (t *trigger) string() string {
	jsonBytes, _ := json.Marshal(t)
	return string(jsonBytes)
}

func refMatch(ref, valueOrPattern string) (bool, error) {
	if strings.HasPrefix(valueOrPattern, "/") &&
		strings.HasSuffix(valueOrPattern, "/") {
		pattern := valueOrPattern[1 : len(valueOrPattern)-1]
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return false, errors.Wrapf(
				err,
				"error compiling regular expression %s",
				valueOrPattern,
			)
		}
		return regex.MatchString(ref), nil
	}
	return ref == valueOrPattern, nil
}
