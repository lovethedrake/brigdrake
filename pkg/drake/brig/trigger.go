package brig

import (
	"encoding/json"
	"log"

	"github.com/lovethedrake/canard/pkg/brigade"
	"github.com/lovethedrake/canard/pkg/drake"
)

const BrigadeCLIEventSource = "brigade.sh/cli"

type trigger struct {
	EventTypes []string `json:"eventTypes"`
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
	if event.Source != BrigadeCLIEventSource {
		log.Printf(
			"event from source %q does not match brig trigger",
			event.Source,
		)
		return false, nil
	}

	for _, eventType := range t.EventTypes {
		if event.Type == eventType {
			log.Printf("%q event matches trigger", event.Type)
			return true, nil
		}
	}

	log.Printf("%q event does not match trigger", event.Type)
	return false, nil
}
