package brigade

import (
	"context"
	"log"

	"github.com/lovethedrake/drakecore/config"
)

func (e *executor) runStage(
	ctx context.Context,
	project Project,
	event Event,
	pipelineName string,
	stageIndex int,
	jobs []config.Job,
	environment []string,
) error {
	log.Printf("executing pipeline \"%s\" stage %d", pipelineName, stageIndex)
	errCh := make(chan error)
	var runningJobs int
	for _, job := range jobs {
		log.Printf(
			"executing pipeline \"%s\" stage %d job \"%s\"",
			pipelineName,
			stageIndex,
			job.Name(),
		)
		runningJobs++
		go e.runJobPod(
			ctx,
			project,
			event,
			pipelineName,
			stageIndex,
			job,
			environment,
			errCh,
		)
	}
	// Wait for all the jobs to finish.
	errs := []error{}
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
		runningJobs--
		if runningJobs == 0 {
			break
		}
	}
	if len(errs) > 1 {
		return &multiError{errs: errs}
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return nil
}
