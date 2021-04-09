package brigade

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/brigadecore/brigade/sdk/v2/core"
	"github.com/pkg/errors"
)

// Event represents a Brigade event.
type Event struct {
	// ID is the unique identifier for the event.
	ID string

	// Project that registered the handler being called for the event.
	Project Project

	// Source is the unique identifier of the gateway which created the event.
	Source string

	// Type of event. Values and meanings are source-specific.
	Type string

	// ShortTitle for the event, suitable for display in space-limited UI such
	// as lists.
	ShortTitle string

	// LongTitle for the event, containing additional details.
	LongTitle string

	// Payload is the content of the event. This is source- and type-specific.
	Payload string

	// Worker assigned to handle the event.
	Worker Worker
}

type Project struct {
	// ID is the unique identifier of the project.
	ID string

	// Secrets is a map of secret key/value pairs defined in the project.
	Secrets map[string]string
}

type Worker struct {
	// ApiAddress is the endpoint of the Brigade API server.
	//nolint
	ApiAddress string

	// ApiToken which can be used to authenticate to the API server.
	// The token is specific to the current event and allows you to create
	// jobs for that event. It has no other permissions.
	//nolint
	ApiToken string

	// ConfigFilesDirectory where the worker stores configuration files,
	// including event handler code files such as brigade.js and brigade.json.
	ConfigFilesDirectory string

	// DefaultConfigFiles to use for any configuration files that are not
	// present.
	DefaultConfigFiles map[string]string

	// LogLevel is the desired granularity of worker logs. Worker logs are
	// distinct from job logs - the containers in a job will emit logs
	// according to their own configuration.
	LogLevel string

	// Git contains git-specific Worker configuration.
	Git core.GitConfig
}

// Revision represents VCS-related details.
type Revision struct {
	// Commit is the VCS commit ID (e.g. the Git commit)
	Commit string `envconfig:"BRIGADE_COMMIT_ID"`
	// Ref is the VCS full reference, defaults to `refs/heads/master`
	Ref string `envconfig:"BRIGADE_COMMIT_REF"`
}

// LoadEvent returns an Event object with values derived from
// /var/event/event.json
func LoadEvent() (Event, error) {
	eventPath := "/var/event/event.json"
	contents, err := ioutil.ReadFile(eventPath)
	if err != nil {
		return Event{}, fmt.Errorf("error reading %s", eventPath)
	}

	evt := Event{}
	err = json.Unmarshal(contents, &evt)
	return evt, errors.Wrap(err, "error loading event json")
}
