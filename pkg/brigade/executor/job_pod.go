package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/brigadecore/brigade/sdk/v2/core"
	"github.com/brigadecore/brigade/sdk/v2/restmachinery"
	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/brigdrake/pkg/brigade/drakespec"
	"github.com/lovethedrake/go-drake/config"
	"github.com/pkg/errors"
)

func runJob(
	ctx context.Context,
	event brigade.Event,
	pipelineName string,
	jobDef config.Job,
) error {
	job := drakespec.ToBrigadeJob(jobDef)

	// TODO(carolynvs): Make allow insecure connections configurable
	jobsClient := core.NewJobsClient(event.Worker.ApiAddress, event.Worker.ApiToken, &restmachinery.APIClientOptions{AllowInsecureConnections: true})

	err := jobsClient.Create(ctx, event.ID, job)
	if err != nil {
		return errors.Wrapf(err, "could not create job %s for pipeline %s on event %s", jobDef.Name(), pipelineName, event.ID)
	}

	jobStatus, jobErr, err := jobsClient.WatchStatus(ctx, event.ID, job.Name)
	if err != nil {
		return errors.Wrapf(err, "could not watch job %s for pipeline %s on event %s", job.Name, pipelineName, event.ID)
	}

	return waitForJobCompletion(ctx, job, jobStatus, jobErr)
}

func waitForJobCompletion(
	ctx context.Context,
	job core.Job,
	jobStatus <-chan core.JobStatus,
	jobErr <-chan error,
) error {
	timer := &time.Timer{}
	if job.Spec.TimeoutSeconds > 0 {
		timer = time.NewTimer(time.Duration(job.Spec.TimeoutSeconds) * time.Second)
		defer timer.Stop()
	}

	for {
		select {
		case status := <-jobStatus:
			if status.Phase.IsTerminal() {
				if status.Phase != core.JobPhaseSucceeded {
					return errors.Errorf("job %s did not succeed, ended in state %s", job.Name, status.Phase)
				}
				return nil
			}
			continue
		case err := <-jobErr:
			fmt.Println("Job Error:", err.Error())
			continue
		case <-timer.C:
			err := &timedOutError{job: job.Name}
			return err
		case <-ctx.Done():
			err := &inProgressJobAbortedError{job: job.Name}
			return err
		}
	}
}
