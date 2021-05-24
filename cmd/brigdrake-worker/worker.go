package main

import (
	"log"

	"github.com/lovethedrake/canard/pkg/brigade"
	"github.com/lovethedrake/canard/pkg/brigade/executor"
	"github.com/lovethedrake/canard/pkg/signals"
	"github.com/lovethedrake/canard/pkg/version"
	"github.com/lovethedrake/go-drake/config"
)

func main() {

	log.Printf(
		"Starting Canard worker -- version %s -- commit %s -- supports "+
			"DrakeSpec %s",
		version.Version(),
		version.Commit(),
		config.SupportedSpecVersions,
	)

	event, err := brigade.LoadEvent()
	if err != nil {
		log.Fatal(err)
	}

	ctx := signals.Context()
	if err = executor.ExecuteBuild(ctx, event); err != nil {
		log.Fatal(err)
	}
}
