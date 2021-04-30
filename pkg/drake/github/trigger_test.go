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
				Source: "bitbucket",
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
				Source: "github",
				Type:   "pull_request",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "pull request event with unconfigured pull request event " +
				"selector",
			trigger: &trigger{},
			event: brigade.Event{
				Source: "github",
				Type:   "check_suite:requested",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "pull request event with unconfigured target branch selector",
			trigger: &trigger{
				PullRequestEventSelector: &pullRequestEventSelector{},
			},
			event: brigade.Event{
				Source: "github",
				Type:   "pull_request:opened",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "pull request event that does not match trigger",
			trigger: &trigger{
				PullRequestEventSelector: &pullRequestEventSelector{
					TargetBranchSelector: &refSelector{
						WhitelistedRefs: []string{"master"},
					},
				},
			},
			event: brigade.Event{
				Source:  "github",
				Type:    "pull_request:opened",
				Payload: `{"action":"opened","pull_request":{"base":{"ref":"foo"}}}`, // nolint: lll
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "pull request event that matches trigger",
			trigger: &trigger{
				PullRequestEventSelector: &pullRequestEventSelector{
					TargetBranchSelector: &refSelector{
						WhitelistedRefs: []string{"master"},
					},
				},
			},
			event: brigade.Event{
				Source:  "github",
				Type:    "pull_request:opened",
				Payload: `{"action":"opened","pull_request":{"base":{"ref":"master"}}}`, // nolint: lll
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.True(t, matches)
			},
		},
		{
			name:    "push event for branch with no push event selector",
			trigger: &trigger{},
			event: brigade.Event{
				Source:  "github",
				Type:    "push",
				Payload: `{"ref":"refs/heads/master"}`,
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event for branch with push event selector that has no " +
				"branch selector",
			trigger: &trigger{
				PushEventSelector: &pushEventSelector{},
			},
			event: brigade.Event{
				Source:  "github",
				Type:    "push",
				Payload: `{"ref":"refs/heads/master"}`,
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event for branch that does not match push event selector's " +
				"branch selector",
			trigger: &trigger{
				PushEventSelector: &pushEventSelector{
					BranchSelector: &refSelector{
						WhitelistedRefs: []string{"master"},
					},
				},
			},
			event: brigade.Event{
				Source:  "github",
				Type:    "push",
				Payload: `{"ref":"refs/heads/foo"}`,
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event for branch that that matches push event selector's " +
				"branch selector",
			trigger: &trigger{
				PushEventSelector: &pushEventSelector{
					BranchSelector: &refSelector{
						WhitelistedRefs: []string{"master"},
					},
				},
			},
			event: brigade.Event{
				Source:  "github",
				Type:    "push",
				Payload: `{"ref":"refs/heads/master"}`,
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.True(t, matches)
			},
		},
		{
			name:    "push event for tag with no push event selector",
			trigger: &trigger{},
			event: brigade.Event{
				Source:  "github",
				Type:    "push",
				Payload: `{"ref":"refs/tags/foo"}`,
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event for tag with push event selector that has no tag " +
				"selector",
			trigger: &trigger{
				PushEventSelector: &pushEventSelector{},
			},
			event: brigade.Event{
				Source:  "github",
				Type:    "push",
				Payload: `{"ref":"refs/tags/foo"}`,
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event for tag that does not match push event selector's " +
				"tag selector",
			trigger: &trigger{
				PushEventSelector: &pushEventSelector{
					TagSelector: &refSelector{
						WhitelistedRefs: []string{"foo"},
					},
				},
			},
			event: brigade.Event{
				Source:  "github",
				Type:    "push",
				Payload: `{"ref":"refs/tags/bar"}`,
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "push event for tag that matches push event selector's tag " +
				"selector",
			trigger: &trigger{
				PushEventSelector: &pushEventSelector{
					TagSelector: &refSelector{
						WhitelistedRefs: []string{"foo"},
					},
				},
			},
			event: brigade.Event{
				Source:  "github",
				Type:    "push",
				Payload: `{"ref":"refs/tags/foo"}`,
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
