package github

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-github/github"
	"github.com/lovethedrake/drakecore/config"
	"github.com/stretchr/testify/require"
)

var (
	repoName          = "foobar"
	repoOwner         = "krancour"
	headSHA           = "1234567"
	csePayloadWrapper = checkSuiteEventPayloadWrapper{
		Token: "fakeToken",
		CheckSuiteEvent: github.CheckSuiteEvent{
			Repo: &github.Repository{
				Name: &repoName,
				Owner: &github.User{
					Login: &repoOwner,
				},
			},
			CheckSuite: &github.CheckSuite{
				HeadSHA: &headSHA,
			},
		},
	}
)

type fakeGithubClient struct {
	t                  *testing.T
	expectedConclusion string
}

func (f *fakeGithubClient) NewRequest(
	_ string,
	_ string,
	body interface{},
) (*http.Request, error) {
	require.IsType(f.t, github.CheckRun{}, body)
	run := body.(github.CheckRun)
	require.NotNil(f.t, run.Status)
	switch *run.Status {
	case "in_progress":
		require.Nil(f.t, run.Conclusion)
	default:
		require.NotNil(f.t, run.Conclusion)
		require.Equal(f.t, f.expectedConclusion, *run.Conclusion)
	}
	return &http.Request{
		Header: http.Header{},
	}, nil
}

func (f *fakeGithubClient) Do(
	context.Context,
	*http.Request,
	interface{},
) (*github.Response, error) {
	return nil, nil
}

type fakeJob struct {
	name string
}

func (f *fakeJob) Name() string {
	return f.name
}

func (f *fakeJob) Containers() []config.Container {
	return nil
}

func TestNewJobStatusNotifier(t *testing.T) {
	jsnIFace := newJobStatusNotifier(csePayloadWrapper)
	require.IsType(t, &jobStatusNotifier{}, jsnIFace)
	jsn := jsnIFace.(*jobStatusNotifier)
	require.Equal(
		t,
		fmt.Sprintf("repos/%s/%s/check-runs", repoOwner, repoName),
		jsn.checkRunsURL,
	)
	require.Equal(t, headSHA, jsn.commit)
	require.NotNil(t, jsn.githubClient)
}

func TestSendNotifications(t *testing.T) {
	jsnIFace := newJobStatusNotifier(csePayloadWrapper)
	require.IsType(t, &jobStatusNotifier{}, jsnIFace)
	jsn := jsnIFace.(*jobStatusNotifier)

	testCases := []struct {
		name               string
		notificationFn     func(job config.Job) error
		expectedConclusion string
	}{
		{
			name:               "in progress",
			notificationFn:     jsn.SendInProgressNotification,
			expectedConclusion: "in_progress",
		},
		{
			name:               "success",
			notificationFn:     jsn.SendSuccessNotification,
			expectedConclusion: "success",
		},
		{
			name:               "cancelled",
			notificationFn:     jsn.SendCancelledNotification,
			expectedConclusion: "cancelled",
		},
		{
			name:               "timed out",
			notificationFn:     jsn.SendTimedOutNotification,
			expectedConclusion: "timed_out",
		},
		{
			name:               "failure",
			notificationFn:     jsn.SendFailureNotification,
			expectedConclusion: "failure",
		},
	}

	job := &fakeJob{
		name: "fakeJobName",
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Swap real github client for a fake one
			jsn.githubClient = &fakeGithubClient{
				t:                  t,
				expectedConclusion: testCase.expectedConclusion,
			}
			err := testCase.notificationFn(job)
			require.NoError(t, err)
		})
	}
}
