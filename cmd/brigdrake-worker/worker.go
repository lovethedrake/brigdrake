package main

import (
	"log"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/lovethedrake/brigdrake/pkg/brigade/executor"
	"github.com/lovethedrake/brigdrake/pkg/signals"
	"github.com/lovethedrake/brigdrake/pkg/version"
	"github.com/lovethedrake/drakecore/config"
)

func main() {

	log.Printf(
		"Starting BrigDrake worker -- version %s -- commit %s -- supports "+
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
