package brig

import (
	"testing"

	"github.com/lovethedrake/canard/pkg/brigade"
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
			name:    "unsupported event type",
			trigger: &trigger{},
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
			name: "non-match",
			trigger: &trigger{
				EventTypes: []string{"foo"},
			},
			event: brigade.Event{
				Source: BrigadeCLIEventSource,
				Type:   "bar",
			},
			assertions: func(t *testing.T, matches bool, err error) {
				require.NoError(t, err)
				require.False(t, matches)
			},
		},
		{
			name: "match",
			trigger: &trigger{
				EventTypes: []string{"foo"},
			},
			event: brigade.Event{
				Source: BrigadeCLIEventSource,
				Type:   "foo",
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
