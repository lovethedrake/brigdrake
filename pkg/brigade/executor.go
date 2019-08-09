package brigade

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"

	"github.com/brigadecore/brigade-github-app/pkg/webhook"
	"github.com/lovethedrake/brigdrake/pkg/vcs"
	"github.com/lovethedrake/brigdrake/pkg/vcs/github"
	"github.com/lovethedrake/drakecore/config"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

var tagRefRegex = regexp.MustCompile("refs/tags/(.*)")

// BuildExecutor is the public interface for a component that can drive brigade
// builds off of a Drakefile.yaml file.
type BuildExecutor interface {
	ExecuteBuild(ctx context.Context) error
}

type buildExecutor struct {
	project           Project
	event             Event
	workerConfig      WorkerConfig
	kubeClient        kubernetes.Interface
	jobStatusNotifier vcs.JobStatusNotifier
}

// NewBuildExecutor returns a component that can drive brigade builds off of a
// Drakefile.yaml file.
func NewBuildExecutor(
	project Project,
	event Event,
	workerConfig WorkerConfig,
	kubeClient kubernetes.Interface,
) (BuildExecutor, error) {
	var jobStatusNotifier vcs.JobStatusNotifier
	// This switch is all about obtaining event-provider-specific implementation
	// of various interfaces that are used throughout the builder executor. While
	// we're at it, we'll return an error if the event provider is one we don't
	// accommodate.
	switch event.Provider {
	case "github":
		webhookPayload := &webhook.Payload{}
		if err := json.Unmarshal(event.Payload, webhookPayload); err != nil {
			return nil, err
		}
		var err error
		if webhookPayload.Type == "check_run" ||
			webhookPayload.Type == "check_suite" {
			jobStatusNotifier, err = github.NewJobStatusNotifier(webhookPayload)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, errors.Errorf(
			"cannot build executor for unrecognized event provider %s",
			event.Provider,
		)
	}
	return &buildExecutor{
		project:           project,
		event:             event,
		workerConfig:      workerConfig,
		kubeClient:        kubeClient,
		jobStatusNotifier: jobStatusNotifier,
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
	var runningPipelines int
	for _, pipeline := range pipelines {
		if meetsCriteria, err := pipeline.Matches(branch, tag); err != nil {
			return err
		} else if meetsCriteria {
			runningPipelines++
			go b.runPipeline(ctx, pipeline, environment, errCh)
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
