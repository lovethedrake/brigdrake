package github

import (
	"testing"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/stretchr/testify/require"
)

func TestMatches(t *testing.T) {
	testCases := []struct {
		name       string
		trigger    *trigger
		event      brigade.Event
		assertions func(*testing.T, bool, error)
	}{
		{
			name:    "non-github event",
			trigger: &trigger{},
			event: brigade.Event{
				Provider: "bitbucket",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name:    "unsupported event type",
			trigger: &trigger{},
			event: brigade.Event{
				Provider: "github",
				Type:     "pull_request",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "check suite request event with unconfigured check suite " +
				"request event selector",
			trigger: &trigger{},
			event: brigade.Event{
				Provider: "github",
				Type:     "check_suite:requested",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "check suite request event with unconfigured branch selector",
			trigger: &trigger{
				CheckSuiteRequestEventSelector: &checkSuiteRequestEventSelector{},
			},
			event: brigade.Event{
				Provider: "github",
				Type:     "check_suite:requested",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "check suite request event that does not match trigger",
			trigger: &trigger{
				CheckSuiteRequestEventSelector: &checkSuiteRequestEventSelector{
					BranchSelector: &refSelector{
						WhitelistedRefs: []string{"master"},
					},
				},
			},
			event: brigade.Event{
				Provider: "github",
				Type:     "check_suite:requested",
				// Looks like a check suite request for a commit from a fork
				Payload: []byte(`{"body":{"action":"rerequested","check_suite":{"head_branch":null}}}`), // nolint: lll
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "check suite request event that matches trigger",
			trigger: &trigger{
				CheckSuiteRequestEventSelector: &checkSuiteRequestEventSelector{
					BranchSelector: &refSelector{
						WhitelistedRefs: []string{"master"},
					},
				},
			},
			event: brigade.Event{
				Provider: "github",
				Type:     "check_suite:requested",
				// Looks like a check suite request for a commit on master
				Payload: []byte(`{"body":{"action":"rerequested","check_suite":{"head_branch":"master"}}}`), // nolint: lll
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.True(t, matches)
			},
		},
		{
			name: "push event with unconfigured tag push event selector",
			trigger: &trigger{
				TagPushEventSelector: &tagPushEventSelector{},
			},
			event: brigade.Event{
				Provider: "github",
				Type:     "push",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event that isn't for a new tag",
			trigger: &trigger{
				TagPushEventSelector: &tagPushEventSelector{
					TagSelector: &refSelector{
						WhitelistedRefs: []string{"v0.0.1"},
					},
				},
			},
			event: brigade.Event{
				Provider: "github",
				Type:     "push",
				// Looks like a push request that isn't for a new tag
				Payload: []byte(`{"ref":"refs/head/master"}`),
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event that does not match trigger",
			trigger: &trigger{
				TagPushEventSelector: &tagPushEventSelector{
					TagSelector: &refSelector{
						WhitelistedRefs: []string{"v0.0.1"},
					},
				},
			},
			event: brigade.Event{
				Provider: "github",
				Type:     "push",
				Payload:  []byte(`{"ref":"refs/tags/foobar"}`),
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event that matches trigger",
			trigger: &trigger{
				TagPushEventSelector: &tagPushEventSelector{
					TagSelector: &refSelector{
						WhitelistedRefs: []string{"v0.0.1"},
					},
				},
			},
			event: brigade.Event{
				Provider: "github",
				Type:     "push",
				Payload:  []byte(`{"ref":"refs/tags/v0.0.1"}`),
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.True(t, matches)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			matches, err := testCase.trigger.Matches(testCase.event)
			testCase.assertions(t, matches, err)
		})
	}
}
