package executor

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/brigdrake/pkg/drake"
	"github.com/lovethedrake/brigdrake/pkg/drake/brig"
	"github.com/lovethedrake/brigdrake/pkg/drake/github"
	"github.com/lovethedrake/drakecore/config"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

var triggerBuilderFns = map[string]func([]byte) (drake.Trigger, error){
	"github.com/lovethedrake/drakespec-github": github.NewTriggerFromJSON,
	"github.com/lovethedrake/drakespec-brig":   brig.NewTriggerFromJSON,
}

// ExecuteBuild can execute a Brigade build driven via Drakefile.yaml when
// supplied with a Brigade project, event, and worker configuration, as well
// as a Kubernetes client.
func ExecuteBuild(
	ctx context.Context,
	project brigade.Project,
	event brigade.Event,
	workerConfig brigade.WorkerConfig,
	kubeClient kubernetes.Interface,
) error {
	// nolint: lll
	possibleDrakefileLocations := []string{
		"/etc/brigade/script",                        // data mounted from event secret (e.g. brig run)
		"/vcs/Drakefile.yaml",                        // checked out in repo
		"/etc/brigade-project/defaultScript",         // data mounted from project.DefaultScript
		"/etc/brigade-default-script/Drakefile.yaml", // mounted configmap named in brigade.sh/project.DefaultScriptName
	}
	var drakefileLocation string
	for _, possibleDrakefileLocation := range possibleDrakefileLocations {
		fileInfo, err := os.Stat(possibleDrakefileLocation)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return errors.Wrapf(
				err,
				"error getting info for file %q",
				possibleDrakefileLocation,
			)
		}
		if fileInfo.Size() == 0 {
			continue
		}
		drakefileLocation = possibleDrakefileLocation
		break
	}
	if drakefileLocation == "" {
		return errors.New("could not locate Drakefile.yaml")
	}

	log.Printf("loading configuration from %q", drakefileLocation)
	cfg, err := config.NewConfigFromFile(drakefileLocation)
	if err != nil {
		return errors.Wrapf(err, "error reading %s", drakefileLocation)
	}

	// Find all pipelines that are eligible for execution.
	pipelinesToExecute := []config.Pipeline{}
	for _, pipeline := range cfg.AllPipelines() {
		log.Printf("evaluating triggers for pipeline %q", pipeline.Name())
		for i, pipelineTrigger := range pipeline.Triggers() {
			triggerBuilderFn, ok := triggerBuilderFns[pipelineTrigger.SpecURI()]
			if !ok {
				// Don't know what to do with this trigger...
				continue // Next trigger
			}
			trigger, err := triggerBuilderFn(pipelineTrigger.Config())
			if err != nil {
				return errors.Wrapf(
					err,
					"error parsing trigger %d (%q) configuration for pipeline %q",
					i,
					pipelineTrigger.SpecURI(),
					pipeline.Name(),
				)
			}
			meetsCriteria, err := trigger.Matches(event)
			if err != nil {
				return errors.Wrapf(
					err,
					"error evaluating execution criteria for trigger %d (%q) "+
						"configuration for pipeline %q",
					i,
					pipelineTrigger.SpecURI(),
					pipeline.Name(),
				)
			}
			if meetsCriteria {
				pipelinesToExecute = append(pipelinesToExecute, pipeline)
				break // Stop iterating over triggers; move on to the next pipeline
			}
		}
	}

	// Bail if we found no pipelines to execute
	if len(pipelinesToExecute) == 0 {
		return nil
	}

	// Create build secret
	if err := createBuildSecret(project, event, kubeClient); err != nil {
		return err
	}
	defer func() {
		if err := destroyBuildSecret(project, event, kubeClient); err != nil {
			log.Printf("error destroying build secret: %s", err)
		}
	}()

	// Execute all pipelines we have identified-- each in their own goroutine
	wg := &sync.WaitGroup{}
	errCh := make(chan error)
	for _, pipeline := range pipelinesToExecute {
		p := pipeline // Avoid closing over a variable we're using for iteration
		wg.Add(1)
		go executePipeline(
			ctx,
			project,
			event,
			workerConfig,
			p,
			kubeClient,
			wg,
			errCh,
		)
	}

	// Convert wg to a channel so we can use it in selects
	allExecutorsDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allExecutorsDone)
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
		case err := <-errCh:
			if err != nil {
				errs = append(errs, err)
			}
		case <-allExecutorsDone:
			break errLoop
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
