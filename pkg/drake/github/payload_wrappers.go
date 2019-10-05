package github

import (
	"github.com/google/go-github/github"
)

// checkSuiteEventPayloadWrapper is a thin wrapper around a
// github.CheckSuiteEvent. The brigade-github-app, when it receives a
// github.CheckSuiteEvent from GitHub wraps it in another struct that is
// augmented with additional information, including security context (i.e. an
// installation token). The brigade-github-app marshals that and uses it as the
// payload of the event it emits into Brigade. checkSuiteEventPayloadWrapper
// exists to facilitate unmarshaling of that payload.
type checkSuiteEventPayloadWrapper struct {
	Token           string                 `json:"token"`
	CheckSuiteEvent github.CheckSuiteEvent `json:"body"`
}
