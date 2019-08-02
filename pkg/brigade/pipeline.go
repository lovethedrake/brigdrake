package brigade

import (
	"context"
	"log"

	"github.com/lovethedrake/drakecore/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func (e *executor) runPipeline(
	ctx context.Context,
	project Project,
	event Event,
	pipeline config.Pipeline,
	environment []string,
	errCh chan<- error,
) {
	log.Printf("executing pipeline \"%s\"", pipeline.Name())
	log.Printf("creating shared storage for pipeline \"%s\"", pipeline.Name())
	var err error
	if err = e.createSrcPVC(project, event, pipeline.Name()); err != nil {
		errCh <- err
		return
	}
	log.Printf("created shared storage for pipeline \"%s\"", pipeline.Name())
	defer func() {
		select {
		// If context was canceled, we have a bunch of pods to get rid of that
		// we'd like to keep otherwise.
		case <-ctx.Done():
			labelSelector := labels.NewSelector()
			workerRequirement, rerr := labels.NewRequirement(
				"worker",
				selection.Equals,
				[]string{event.WorkerID},
			)
			if rerr != nil {
				log.Printf(
					"error deleting pods for pipeline \"%s\": %s",
					pipeline.Name(),
					rerr,
				)
			} else {
				labelSelector = labelSelector.Add(*workerRequirement)
				log.Printf("deleting pods \"%s\"", labelSelector.String())
				if derr := e.kubeClient.CoreV1().Pods(
					project.Kubernetes.Namespace,
				).DeleteCollection(
					&metav1.DeleteOptions{},
					metav1.ListOptions{
						LabelSelector: labelSelector.String(),
					},
				); derr != nil {
					log.Printf(
						"error deleting pods for pipeline \"%s\": %s",
						pipeline.Name(),
						derr,
					)
				}
			}
		default:
		}
		log.Printf("destroying shared storage for pipeline \"%s\"", pipeline.Name())
		if derr := e.destroySrcPVC(project, event, pipeline.Name()); derr != nil {
			log.Printf(
				"error destroying shared storage for pipeline \"%s\": %s",
				pipeline.Name(),
				derr,
			)
		} else {
			log.Printf(
				"destroyed shared storage for pipeline \"%s\"",
				pipeline.Name(),
			)
		}
		errCh <- err
	}()
	log.Printf(
		"cloning source to shared storage for pipeline \"%s\"",
		pipeline.Name(),
	)
	if err =
		e.runSourceClonePod(ctx, project, event, pipeline.Name()); err != nil {
		return
	}
	log.Printf(
		"cloned source to shared storage for pipeline \"%s\"",
		pipeline.Name(),
	)
	select {
	case <-ctx.Done():
		return
	default:
	}
	for stageIndex, stageJobs := range pipeline.Jobs() {
		if err = e.runStage(
			ctx,
			project,
			event,
			pipeline.Name(),
			stageIndex,
			stageJobs,
			environment,
		); err != nil {
			return
		}
	}
}
