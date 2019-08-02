package brigade

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/kelseyhightower/envconfig"
	uuid "github.com/satori/go.uuid"
)

// Event represents a Brigade event.
type Event struct {
	// BuildID is the UUID for the build. This is unique.
	BuildID string `envconfig:"BRIGADE_BUILD_ID"`
	// WorkerID is the ID of the worker.
	WorkerID string `envconfig:"BRIGADE_BUILD_NAME"`
	// Type is the event type, such as `push`, `pull_request`
	Type string `envconfig:"BRIGADE_EVENT_TYPE"`
	// Provider is the name of the event provider, such as `github` or `dockerhub`
	Provider string `envconfig:"BRIGADE_EVENT_PROVIDER"`
	// Revision is VCS information
	Revision Revision
	// Payload is the payload from the original event trigger.
	Payload []byte
}

// Revision represents VCS-related details.
type Revision struct {
	// Commit is the VCS commit ID (e.g. the Git commit)
	Commit string `envconfig:"BRIGADE_COMMIT_ID"`
	// Ref is the VCS full reference, defaults to `refs/heads/master`
	Ref string `envconfig:"BRIGADE_COMMIT_REF"`
}

// NewEventWithDefaults returns an Event object with default values already
// applied. Callers are then free to set custom values for the remaining fields
// and/or override default values.
func NewEventWithDefaults() Event {
	defaultUUID := uuid.NewV4().String()
	return Event{
		BuildID:  defaultUUID,
		WorkerID: fmt.Sprintf("unknown-%s", defaultUUID),
		Type:     "ping",
		Provider: "unknown",
		Revision: Revision{},
	}
}

// GetEventFromEnvironment returns an Event object with values derived from
// environment variables.
func GetEventFromEnvironment() (Event, error) {
	e := NewEventWithDefaults()
	err := envconfig.Process("", &e)
	if err != nil {
		return e, err
	}
	e.Payload, err = ioutil.ReadFile("/etc/brigade/payload")
	if err != nil {
		log.Println("no payload loaded")
	}
	return e, nil
}
