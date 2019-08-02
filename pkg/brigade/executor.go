package brigade

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"

	"github.com/lovethedrake/drakecore/config"
	"k8s.io/client-go/kubernetes"
)

var tagRefRegex = regexp.MustCompile("refs/tags/(.*)")

// Executor is the public interface for the Brigade executor
type Executor interface {
	ExecuteBuild(
		ctx context.Context,
		project Project,
		event Event,
	) error
}

type executor struct {
	workerConfig WorkerConfig
	kubeClient   kubernetes.Interface
}

// NewExecutor returns an executor suitable for use with Brigade
func NewExecutor(
	workerConfig WorkerConfig,
	kubeClient kubernetes.Interface,
) Executor {
	return &executor{
		workerConfig: workerConfig,
		kubeClient:   kubeClient,
	}
}

// TODO: Implement this
func (e *executor) ExecuteBuild(
	ctx context.Context,
	project Project,
	event Event,
) error {
	var branch, tag string
	// We're only interested in events from the Github gateway.
	if event.Provider != "github" {
		log.Printf(
			"received event from non-github provider \"%s\"-- nothing to execute",
			event.Provider,
		)
		return nil
	}
	// There are really only two things we're interested in:
	//
	//   1. Check suite requested / re-requested. Github will send one of these
	//      anytime there is a push to a branch, in which case, the
	//      check_suite.head_branch field will indicate the branch that was pushed
	//      to, OR in the case of a PR, the Brigade Gtihub gateway will compell
	//      Github to forward a check suite request, in which case,
	//      check_suite.head_branch will be null (JS) or nil (Go). No branch name
	//      is a valid as SOME branch name for the purposes of determining whether
	//      some pipeline needs to be executed.
	//
	//   2. A push request whose ref field indicates it is a tag.
	//
	// Nothing else will trigger anything.
	switch event.Type {
	case "check_suite:requested", "check_suite:rerequested":
		cse := checkSuiteEvent{}
		if err := json.Unmarshal(event.Payload, &cse); err != nil {
			return err
		}
		if cse.Body.CheckSuite.HeadBranch != nil {
			branch = *cse.Body.CheckSuite.HeadBranch
		}
	case "push":
		pe := pushEvent{}
		if err := json.Unmarshal(event.Payload, &pe); err != nil {
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
			event.Type,
		)
		return nil
	}
	log.Printf("branch: \"%s\"; tag: \"%s\"", branch, tag)
	config, err := config.NewConfigFromFile("/vcs/Drakefile.yaml")
	if err != nil {
		return err
	}

	// Create build secret
	if err := e.createBuildSecret(project, event); err != nil {
		return err
	}
	defer func() {
		if err := e.destroyBuildSecret(project, event); err != nil {
			log.Println(err)
		}
	}()

	pipelines := config.AllPipelines()
	errCh := make(chan error)
	environment := []string{
		fmt.Sprintf("DRAKE_SHA1=%s", event.Revision.Commit),
		fmt.Sprintf("DRAKE_BRANCH=%s", branch),
		fmt.Sprintf("DRAKE_TAG=%s", tag),
	}
	var runningPipelines int
	for _, pipeline := range pipelines {
		if meetsCriteria, err := pipeline.Matches(branch, tag); err != nil {
			return err
		} else if meetsCriteria {
			runningPipelines++
			go e.runPipeline(ctx, project, event, pipeline, environment, errCh)
		}
	}
	if runningPipelines == 0 {
		return nil
	}
	// Wait for all the pipelines to finish.
	errs := []error{}
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
		runningPipelines--
		if runningPipelines == 0 {
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
