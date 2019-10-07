package github

import (
	"encoding/json"
	"log"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/brigdrake/pkg/drake"
	"github.com/pkg/errors"
)

// nolint: lll
type trigger struct {
	CheckSuiteRequestEventSelector *checkSuiteRequestEventSelector `json:"checkSuiteRequest,omitempty"`
	TagPushEventSelector           *tagPushEventSelector           `json:"tagPush,omitempty"`
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
	case "check_suite:requested", "check_suite:rerequested":
		if t.CheckSuiteRequestEventSelector == nil {
			log.Println(
				"check suite request event does not match trigger with unconfigured " +
					"check suite request event selector",
			)
			return false, nil
		}
		matches, err := t.CheckSuiteRequestEventSelector.matches(event)
		if err != nil {
			return false, errors.Wrap(
				err,
				"error matching check suite request event to check suite request "+
					"event selector",
			)
		}
		if matches {
			log.Println("check suite request event matches trigger")
		} else {
			log.Println("check suite request event does not match trigger")
		}
		return matches, nil
	case "push":
		if t.TagPushEventSelector == nil {
			log.Println(
				"push event does not match trigger with unconfigured tag push event " +
					"selector",
			)
			return false, nil
		}
		matches, err := t.TagPushEventSelector.matches(event)
		if err != nil {
			return false, errors.Wrap(
				err,
				"error matching push event to tag push event selector ",
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
