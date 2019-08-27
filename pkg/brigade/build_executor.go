package brigade

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/lovethedrake/drakecore/config"
	"k8s.io/client-go/kubernetes"
)

var tagRefRegex = regexp.MustCompile("refs/tags/(.*)")

// BuildExecutor is the public interface for a component that can drive brigade
// builds off of a Drakefile.yaml file.
type BuildExecutor interface {
	ExecuteBuild(ctx context.Context) error
}

type buildExecutor struct {
	project          Project
	event            Event
	kubeClient       kubernetes.Interface
	pipelineExecutor *pipelineExecutor
}

// NewBuildExecutor returns a component that can drive brigade builds off of a
// Drakefile.yaml file.
func NewBuildExecutor(
	project Project,
	event Event,
	workerConfig WorkerConfig,
	kubeClient kubernetes.Interface,
) (BuildExecutor, error) {
	pipelineExecutor, err := newPipelineExecutor(
		project,
		event,
		workerConfig,
		kubeClient,
	)
	if err != nil {
		return nil, err
	}
	return &buildExecutor{
		project:          project,
		event:            event,
		kubeClient:       kubeClient,
		pipelineExecutor: pipelineExecutor,
	}, nil
}

func (b *buildExecutor) ExecuteBuild(ctx context.Context) error {
	var branch, tag string
	// TODO: This logic looks very github-specific and should be moved off into
	// some implementation of some new interface in the vcs package.
	//
	// There are really only two things we're interested in:
	//
	//   1. Check suite requested / re-requested. GitHub will send one of these
	//      anytime there is a push to a branch, in which case, the
	//      check_suite.head_branch field will indicate the branch that was pushed
	//      to, OR in the case of a PR, the Brigade GitHub gateway will compell
	//      GitHub to forward a check suite request, in which case,
	//      check_suite.head_branch will be null (JS) or nil (Go). No branch name
	//      is as valid as SOME branch name for the purposes of determining
	//      whether some pipeline needs to be executed.
	//
	//   2. A push request whose ref field indicates it is a tag.
	//
	// Nothing else will trigger anything.
	switch b.event.Type {
	case "check_suite:requested", "check_suite:rerequested":
		cse := checkSuiteEvent{}
		if err := json.Unmarshal(b.event.Payload, &cse); err != nil {
			return err
		}
		if cse.Body.CheckSuite.HeadBranch != nil {
			branch = *cse.Body.CheckSuite.HeadBranch
		}
	case "push":
		pe := pushEvent{}
		if err := json.Unmarshal(b.event.Payload, &pe); err != nil {
			return err
		}
		refSubmatches := tagRefRegex.FindStringSubmatch(pe.Ref)
		if len(refSubmatches) != 2 {
			log.Println(
				"received push event that wasn't for a new tag-- nothing to execute",
			)
			return nil
		}
		tag = refSubmatches[1]
	default:
		log.Printf(
			"received event type \"%s\"-- nothing to execute",
			b.event.Type,
		)
		return nil
	}
	log.Printf("branch: \"%s\"; tag: \"%s\"", branch, tag)
	config, err := config.NewConfigFromFile("/vcs/Drakefile.yaml")
	if err != nil {
		return err
	}

	// Create build secret
	if err := b.createBuildSecret(); err != nil {
		return err
	}
	defer func() {
		if err := b.destroyBuildSecret(); err != nil {
			log.Println(err)
		}
	}()

	pipelines := config.AllPipelines()
	errCh := make(chan error)
	environment := []string{
		fmt.Sprintf("DRAKE_SHA1=%s", b.event.Revision.Commit),
		fmt.Sprintf("DRAKE_BRANCH=%s", branch),
		fmt.Sprintf("DRAKE_TAG=%s", tag),
	}

	// Find all pipelines that are eligible for execution and start each in its
	// own goroutine.
	wg := &sync.WaitGroup{}
	for _, pipeline := range pipelines {
		if meetsCriteria, err := pipeline.Matches(branch, tag); err != nil {
			return err
		} else if meetsCriteria {
			wg.Add(1)
			go b.pipelineExecutor.executePipeline(
				ctx,
				pipeline,
				environment,
				wg,
				errCh,
			)
		}
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
