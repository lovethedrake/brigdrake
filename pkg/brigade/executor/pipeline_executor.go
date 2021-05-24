package executor

import (
	"context"
	"log"
	"sync"

	"github.com/lovethedrake/canard/pkg/brigade"
	"github.com/lovethedrake/go-drake/config"
)

func executePipeline(
	ctx context.Context,
	event brigade.Event,
	pipeline config.Pipeline,
	wg *sync.WaitGroup,
	errCh chan<- error,
) {
	defer wg.Done()
	log.Printf("executing pipeline %q", pipeline.Name())
	jobs := pipeline.Jobs()

	// Build a map of channels that lets the job scheduler subscribe to the
	// completion of each job's dependencies. (A given dependency is complete if
	// its channel is closed.)
	dependencyChs := map[string]chan struct{}{}
	for _, job := range jobs {
		dependencyChs[job.Job().Name()] = make(chan struct{})
	}

	// We'll cancel this context if a job fails and we don't want to start any
	// new ones that may be pending. This does NOT mean we cancel jobs that are
	// already in-progress.
	pendingJobsCtx, cancelPendingJobs := context.WithCancel(ctx)
	defer cancelPendingJobs()

	// Start a goroutine to manage each job. This doesn't automatically run
	// it; rather it waits for all the job's dependencies to be filled before
	// executing it.
	managersWg := &sync.WaitGroup{}
	localErrCh := make(chan error)
	for _, j := range jobs {
		job := j
		managersWg.Add(1)
		go func() {
			defer managersWg.Done()
			// Wait for the job's dependencies to complete
			for _, dependency := range job.Dependencies() {
				select {
				case <-dependencyChs[dependency.Job().Name()]:
					// Continue to wait for the next dependency
				case <-pendingJobsCtx.Done():
					// Pending jobs were canceled; abort
					localErrCh <- &pendingJobCanceledError{job: job.Job().Name()}
					return
				case <-ctx.Done():
					// Everything was canceled; abort
					localErrCh <- &pendingJobCanceledError{job: job.Job().Name()}
					return
				}
			}
			if err := runJob(
				ctx,
				event,
				pipeline.Name(),
				job.Job(),
			); err != nil {
				// This localErrCh write isn't in a select because we don't want it to
				// be interruptable since we never want to lose an error message. And we
				// know the goroutine that is collecting errors is also not
				// interruptable and won't stop listening until all the manager
				// goroutines return, so this is ok.
				localErrCh <- err
			} else {
				// Unblock anything that's waiting for this job to complete
				close(dependencyChs[job.Job().Name()])
			}
		}()
	}

	// Convert managersWg to a channel so we can use it in selects
	allManagersDone := make(chan struct{})
	go func() {
		managersWg.Wait()
		close(allManagersDone)
	}()

	// Collect errors from all the executors until they have all completed
	errs := []error{}
errLoop:
	for {
		// Note this select isn't interruptable by canceled contexts because we
		// never want to lose an error message. We know this will inevitably unblock
		// when all the executor goroutines conclude-- which they WILL since those
		// are interruptable.
		select {
		case err := <-localErrCh:
			if err != nil {
				errs = append(errs, err)
				// Once we've had any error, we know the pipeline is failed. We can
				// let jobs already in-progress continue executing, but we don't want
				// to start any new ones. We can signal that by closing this context.
				cancelPendingJobs()
			}
		case <-allManagersDone:
			break errLoop
		}
	}

	if len(errs) > 1 {
		errCh <- &multiError{errs: errs}
	} else if len(errs) == 1 {
		errCh <- errs[0]
	}
}
