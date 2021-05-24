// +build mage

// This is a magefile, and is a "makefile for go".
// See https://magefile.org/
package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/carolynvs/magex/shx"
	"github.com/lovethedrake/brigdrake/pkg/drake/brig"

	// Import shared mage targets for drake
	// mage:import
	"github.com/lovethedrake/go-drake/mage"
)

// Any commands executed by "must" (as opposed to shx.RunV for example), will stop
// the build immediately when the command fails.
var must = shx.CommandBuilder{StopOnError: true}

// Compile the drake CLI with Docker
func Build() {
	pwd, _ := os.Getwd()
	must.RunV("docker", "run", "--rm",
		"-v", pwd+":/go/src/github.com/lovethedrake/brigdrake",
		"-w", "/go/src/github.com/lovethedrake/brigdrake",
		"-v", pwd+"/bin:/shared/bin/drake",
		"brigadecore/go-tools:v0.1.0", "scripts/build-worker-binary.sh", runtime.GOOS, runtime.GOARCH)
}

func BuildImage() {
	must.RunV("./scripts/build-worker-dood.sh")
}

// Run go tests
func Test() {
	coverageFile := filepath.Join(mage.GetOutputDir(), "coverage.txt")
	must.RunV("go", "test", "-timeout=30s", "-race", "-coverprofile="+coverageFile, "-covermode=atomic", "./cmd/...", "./pkg/...")
}

// Build locally and run the hello-world example
func Example() {
	must.RunV("drake", "run", "build-worker-dood")
	img := "carolynvs/brigdrake-worker:unstable"
	must.RunV("docker", "tag", "brigdrake-worker:unstable", img)
	must.RunV("docker", "push", img)
	output, _ := must.OutputV("brig", "event", "create", "-p", "hello-world", "-s", brig.BrigadeCLIEventSource)
	r := regexp.MustCompile(`Created event "(.*)"`)
	matches := r.FindStringSubmatch(output)
	eventId := matches[1]
	time.Sleep(5 * time.Second)
	must.RunV("brig", "event", "logs", "-f", "-i", eventId)
}

// Check go code for lint errors
func Lint() {
	must.RunV("golangci-lint", "run")
}
