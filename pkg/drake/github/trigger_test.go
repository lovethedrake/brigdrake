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
		assertions func(*testing.T, *trigger)
	}{
		{
			name: "a trigger that tests PRs",
			trigger: &trigger{
				BranchSelector: &refSelector{
					BlacklistedRefs: []string{"master"},
				},
			},
			assertions: func(t *testing.T, trigger *trigger) {
				// Looks like a PR
				matches, err := trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "check_suite:rerequested",
						Payload:  []byte(`{"body":{"action":"rerequested","check_suite":{"head_branch":null}}}`), // nolint: lll
					},
				)
				require.NoError(t, err)
				require.True(t, matches)
				// Looks like a merge to master
				matches, err = trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "check_suite:requested",
						Payload:  []byte(`{"body":{"action":"requested","check_suite":{"head_branch":"master"}}}`), // nolint: lll
					},
				)
				require.NoError(t, err)
				require.False(t, matches)
				// Looks like a release
				matches, err = trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "push",
						Payload:  []byte(`{"ref":"refs/tags/v0.0.1"}`),
					},
				)
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "a pipeline that tests the master branch",
			trigger: &trigger{
				BranchSelector: &refSelector{
					WhitelistedRefs: []string{"master"},
				},
			},
			assertions: func(t *testing.T, trigger *trigger) {
				// Looks like a PR
				matches, err := trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "check_suite:rerequested",
						Payload:  []byte(`{"body":{"action":"rerequested","check_suite":{"head_branch":null}}}`), // nolint: lll
					},
				)
				require.NoError(t, err)
				require.False(t, matches)
				// Looks like a merge to master
				matches, err = trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "check_suite:requested",
						Payload:  []byte(`{"body":{"action":"requested","check_suite":{"head_branch":"master"}}}`), // nolint: lll
					},
				)
				require.NoError(t, err)
				require.True(t, matches)
				// Looks like a release
				matches, err = trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "push",
						Payload:  []byte(`{"ref":"refs/tags/v0.0.1"}`),
					},
				)
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "a pipeline for executing a release",
			trigger: &trigger{
				TagSelector: &refSelector{
					WhitelistedRefs: []string{`/v[0-9]+(\.[0-9]+)*(\-.+)?/`},
				},
			},
			assertions: func(t *testing.T, trigger *trigger) {
				// Looks like a PR
				matches, err := trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "check_suite:rerequested",
						Payload:  []byte(`{"body":{"action":"rerequested","check_suite":{"head_branch":null}}}`), // nolint: lll
					},
				)
				require.NoError(t, err)
				require.False(t, matches)
				// Looks like a merge to master
				matches, err = trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "check_suite:requested",
						Payload:  []byte(`{"body":{"action":"requested","check_suite":{"head_branch":"master"}}}`), // nolint: lll
					},
				)
				require.NoError(t, err)
				require.False(t, matches)
				// Looks like a release
				matches, err = trigger.Matches(
					brigade.Event{
						Provider: "github",
						Type:     "push",
						Payload:  []byte(`{"ref":"refs/tags/v0.0.1"}`),
					},
				)
				require.NoError(t, err)
				require.True(t, matches)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, testCase.trigger)
		})
	}
}
