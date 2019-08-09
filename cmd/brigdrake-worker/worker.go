package main

import (
	"log"
	"os"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/brigdrake/pkg/signals"
	"github.com/lovethedrake/brigdrake/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {

	log.Printf(
		"Starting brigdrake worker -- version %s -- commit %s",
		version.Version(),
		version.Commit(),
	)

	clientConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	workerConfig, err := brigade.GetWorkerConfigFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	project, err := brigade.GetProjectFromEnvironmentAndSecret(kubeClient)
	if err != nil {
		log.Fatal(err)
	}

	event, err := brigade.GetEventFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	buildExecutor, err := brigade.NewBuildExecutor(
		project,
		event,
		workerConfig,
		kubeClient,
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := signals.Context()
	if err = buildExecutor.ExecuteBuild(ctx); err != nil {
		log.Fatal(err)
	}

	select {
	case <-ctx.Done():
		os.Exit(1)
	default:
	}

}
