package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	rm "github.com/brigadecore/brigade/sdk/v2/internal/restmachinery"
	"github.com/brigadecore/brigade/sdk/v2/meta"
	"github.com/brigadecore/brigade/sdk/v2/restmachinery"
)

// LogLevel represents the desired granularity of Worker log output.
type LogLevel string

const (
	// LogLevelDebug represents DEBUG level granularity in Worker log output.
	LogLevelDebug LogLevel = "DEBUG"
	// LogLevelInfo represents INFO level granularity in Worker log output.
	LogLevelInfo LogLevel = "INFO"
	// LogLevelWarn represents WARN level granularity in Worker log output.
	LogLevelWarn LogLevel = "WARN"
	// LogLevelError represents ERROR level granularity in Worker log output.
	LogLevelError LogLevel = "ERROR"
)

// WorkerPhase represents where a Worker is within its lifecycle.
type WorkerPhase string

const (
	// WorkerPhaseAborted represents the state wherein a worker was forcefully
	// stopped during execution.
	WorkerPhaseAborted WorkerPhase = "ABORTED"
	// WorkerPhaseCanceled represents the state wherein a pending worker was
	// canceled prior to execution.
	WorkerPhaseCanceled WorkerPhase = "CANCELED"
	// WorkerPhaseFailed represents the state wherein a worker has run to
	// completion but experienced errors.
	WorkerPhaseFailed WorkerPhase = "FAILED"
	// WorkerPhasePending represents the state wherein a worker is awaiting
	// execution.
	WorkerPhasePending WorkerPhase = "PENDING"
	// WorkerPhaseRunning represents the state wherein a worker is currently
	// being executed.
	WorkerPhaseRunning WorkerPhase = "RUNNING"
	// WorkerPhaseSchedulingFailed represents the state wherein a worker was not
	// scheduled due to some unexpected and unrecoverable error encountered by the
	// scheduler.
	WorkerPhaseSchedulingFailed WorkerPhase = "SCHEDULING_FAILED"
	// WorkerPhaseStarting represents the state wherein a Worker is starting on
	// the substrate but isn't running yet.
	WorkerPhaseStarting WorkerPhase = "STARTING"
	// WorkerPhaseSucceeded represents the state where a worker has run to
	// completion without error.
	WorkerPhaseSucceeded WorkerPhase = "SUCCEEDED"
	// WorkerPhaseTimedOut represents the state wherein a worker has has not
	// completed within a designated timeframe.
	WorkerPhaseTimedOut WorkerPhase = "TIMED_OUT"
	// WorkerPhaseUnknown represents the state wherein a worker's state is
	// unknown. Note that this is possible if and only if the underlying Worker
	// execution substrate (Kubernetes), for some unanticipated reason, does not
	// know the Worker's (Pod's) state.
	WorkerPhaseUnknown WorkerPhase = "UNKNOWN"
)

// WorkerPhasesAll returns a slice of WorkerPhases containing ALL possible
// phases. Note that instead of utilizing a package-level slice, this a function
// returns ad-hoc copies of the slice in order to preclude the possibility of
// this important collection being modified at runtime by a client.
func WorkerPhasesAll() []WorkerPhase {
	return []WorkerPhase{
		WorkerPhaseAborted,
		WorkerPhaseCanceled,
		WorkerPhaseFailed,
		WorkerPhasePending,
		WorkerPhaseRunning,
		WorkerPhaseSucceeded,
		WorkerPhaseTimedOut,
		WorkerPhaseUnknown,
	}
}

// WorkerPhasesTerminal returns a slice of WorkerPhases containing ALL phases
// that are considered terminal. Note that instead of utilizing a package-level
// slice, this a function returns ad-hoc copies of the slice in order to
// preclude the possibility of this important collection being modified at
// runtime by a client.
func WorkerPhasesTerminal() []WorkerPhase {
	return []WorkerPhase{
		WorkerPhaseAborted,
		WorkerPhaseCanceled,
		WorkerPhaseFailed,
		WorkerPhaseSucceeded,
		WorkerPhaseTimedOut,
	}
}

// WorkerPhasesNonTerminal returns a slice of WorkerPhases containing ALL phases
// that are considered non-terminal. Note that instead of utilizing a
// package-level slice, this a function returns ad-hoc copies of the slice in
// order to preclude the possibility of this important collection being modified
// at runtime by a client.
func WorkerPhasesNonTerminal() []WorkerPhase {
	return []WorkerPhase{
		WorkerPhasePending,
		WorkerPhaseRunning,
		WorkerPhaseUnknown,
	}
}

// IsTerminal returns a bool indicating whether the WorkerPhase is terminal.
func (w WorkerPhase) IsTerminal() bool {
	switch w {
	case WorkerPhaseAborted:
		fallthrough
	case WorkerPhaseCanceled:
		fallthrough
	case WorkerPhaseFailed:
		fallthrough
	case WorkerPhaseSchedulingFailed:
		fallthrough
	case WorkerPhaseSucceeded:
		fallthrough
	case WorkerPhaseTimedOut:
		return true
	}
	return false
}

// Worker represents a component that orchestrates handling of a single Event.
type Worker struct {
	// Spec is the technical blueprint for the Worker.
	Spec WorkerSpec `json:"spec"`
	// Status contains details of the Worker's current state.
	Status WorkerStatus `json:"status"`
	// Jobs contains details of all Jobs spawned by the Worker during handling of
	// the Event.
	Jobs []Job `json:"jobs,omitempty"`
}

// Job retrieves a Job by name. It returns a boolean indicating whether the
// returned Job is the one requested (true) or a zero value (false) because no
// Job with the specified name belongs to this Worker.
func (w *Worker) Job(jobName string) (Job, bool) {
	for _, j := range w.Jobs {
		if j.Name == jobName {
			return j, true
		}
	}
	return Job{}, false
}

// WorkerSpec is the technical blueprint for a Worker.
type WorkerSpec struct {
	// Container specifies the details of an OCI container that forms the
	// cornerstone of the Worker.
	Container *ContainerSpec `json:"container,omitempty"`
	// UseWorkspace indicates whether the Worker and/or any Jobs it may spawn
	// requires access to a shared workspace. When false, no such workspace is
	// provisioned prior to Worker creation.
	UseWorkspace bool `json:"useWorkspace"`
	// WorkspaceSize specifies the size of a volume that will be provisioned as
	// a shared workspace for the Worker and any Jobs it spawns.
	// The value can be expressed in bytes (as a plain integer) or as a
	// fixed-point integer using one of these suffixes: E, P, T, G, M, K.
	// Power-of-two equivalents may also be used: Ei, Pi, Ti, Gi, Mi, Ki.
	WorkspaceSize string `json:"workspaceSize,omitempty"`
	// Git contains git-specific Worker details.
	Git *GitConfig `json:"git,omitempty"`
	// Kubernetes contains Kubernetes-specific Worker details.
	Kubernetes *KubernetesConfig `json:"kubernetes,omitempty"`
	// JobPolicies specifies policies for any Jobs spawned by the Worker.
	JobPolicies *JobPolicies `json:"jobPolicies,omitempty"`
	// LogLevel specifies the desired granularity of Worker log output.
	LogLevel LogLevel `json:"logLevel,omitempty"`
	// ConfigFilesDirectory specifies a directory within the Worker's workspace
	// where any relevant configuration files (e.g. brigade.json, brigade.js,
	// etc.) can be located.
	ConfigFilesDirectory string `json:"configFilesDirectory,omitempty"`
	// DefaultConfigFiles is a map of configuration file names to configuration
	// file content. This is useful for Workers that do not integrate with any
	// source control system and would like to embed configuration (e.g.
	// brigade.json) or scripts (e.g. brigade.js) directly within the WorkerSpec.
	DefaultConfigFiles map[string]string `json:"defaultConfigFiles,omitempty"`
}

// GitConfig represents git-specific Worker details.
type GitConfig struct {
	// CloneURL specifies the location from where a source code repository may
	// be cloned.
	CloneURL string `json:"cloneURL,omitempty"`
	// Commit specifies a revision (by SHA) to be checked out. If non-empty, this
	// field takes precedence over any value in the Ref field.
	Commit string `json:"commit,omitempty"`
	// Ref is a symbolic reference to a revision to be checked out. If non-empty,
	// the value of the Commit field supercedes any value in this field. Example
	// uses of this field include referencing a branch (refs/heads/<branch name>)
	// or a tag (refs/tags/<tag name>). If left blank, the default value
	// refs/heads/master will be assumed at runtime.
	Ref string `json:"ref,omitempty"`
	// InitSubmodules indicates whether to clone the repository's submodules.
	InitSubmodules bool `json:"initSubmodules"`
}

// KubernetesConfig represents Kubernetes-specific Worker or Job configuration.
type KubernetesConfig struct {
	// ImagePullSecrets enumerates any image pull secrets that Kubernetes may use
	// when pulling the OCI image on which a Worker's or Job's container is based.
	// This field only needs to be utilized in the case of private, custom Worker
	// or Job images. The image pull secrets in question must be created
	// out-of-band by a sufficiently authorized user of the Kubernetes cluster.
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

// JobPolicies represents policies for any Jobs spawned by a Worker.
type JobPolicies struct {
	// AllowPrivileged specifies whether the Worker is permitted to launch Jobs
	// that utilize privileged containers.
	AllowPrivileged bool `json:"allowPrivileged"`
	// AllowDockerSocketMount specifies whether the Worker is permitted to launch
	// Jobs that mount the underlying host's Docker socket into its own file
	// system.
	AllowDockerSocketMount bool `json:"allowDockerSocketMount"`
}

// WorkerStatus represents the status of a Worker.
type WorkerStatus struct {
	// Started indicates the time the Worker began execution. It will be nil for
	// a Worker that is not yet executing.
	Started *time.Time `json:"started,omitempty"`
	// Ended indicates the time the Worker concluded execution. It will be nil
	// for a Worker that is not done executing (or hasn't started).
	Ended *time.Time `json:"ended,omitempty"`
	// Phase indicates where the Worker is in its lifecycle.
	Phase WorkerPhase `json:"phase,omitempty"`
}

// MarshalJSON amends WorkerStatus instances with type metadata so that clients
// do not need to be concerned with the tedium of doing so.
func (w WorkerStatus) MarshalJSON() ([]byte, error) {
	type Alias WorkerStatus
	return json.Marshal(
		struct {
			meta.TypeMeta `json:",inline"`
			Alias         `json:",inline"`
		}{
			TypeMeta: meta.TypeMeta{
				APIVersion: meta.APIVersion,
				Kind:       "WorkerStatus",
			},
			Alias: (Alias)(w),
		},
	)
}

// WorkersClient is the specialized client for managing Event Workers with the
// Brigade API.
type WorkersClient interface {
	// Start starts the indicated Event's Worker on Brigade's workload execution
	// substrate.
	Start(ctx context.Context, eventID string) error
	// Get returns an Event's Worker's status.
	GetStatus(ctx context.Context, eventID string) (WorkerStatus, error)
	// WatchStatus returns a channel over which an Event's Worker's status is
	// streamed. The channel receives a new WorkerStatus every time there is any
	// change in that status.
	WatchStatus(
		ctx context.Context,
		eventID string,
	) (<-chan WorkerStatus, <-chan error, error)
	// UpdateStatus updates the status of an Event's Worker.
	UpdateStatus(
		ctx context.Context,
		eventID string,
		status WorkerStatus,
	) error
	Cleanup(ctx context.Context, eventID string) error

	Jobs() JobsClient
}

type workersClient struct {
	*rm.BaseClient
	jobsClient JobsClient
}

// NewWorkersClient returns a specialized client for managing Event Workers.
func NewWorkersClient(
	apiAddress string,
	apiToken string,
	opts *restmachinery.APIClientOptions,
) WorkersClient {
	return &workersClient{
		BaseClient: rm.NewBaseClient(apiAddress, apiToken, opts),
		jobsClient: NewJobsClient(apiAddress, apiToken, opts),
	}
}

func (w *workersClient) Start(ctx context.Context, eventID string) error {
	return w.ExecuteRequest(
		ctx,
		rm.OutboundRequest{
			Method:      http.MethodPut,
			Path:        fmt.Sprintf("v2/events/%s/worker/start", eventID),
			SuccessCode: http.StatusOK,
		},
	)
}

func (w *workersClient) GetStatus(
	ctx context.Context,
	eventID string,
) (WorkerStatus, error) {
	status := WorkerStatus{}
	return status, w.ExecuteRequest(
		ctx,
		rm.OutboundRequest{
			Method:      http.MethodGet,
			Path:        fmt.Sprintf("v2/events/%s/worker/status", eventID),
			SuccessCode: http.StatusOK,
			RespObj:     &status,
		},
	)
}

func (w *workersClient) WatchStatus(
	ctx context.Context,
	eventID string,
) (<-chan WorkerStatus, <-chan error, error) {
	resp, err := w.SubmitRequest(
		ctx,
		rm.OutboundRequest{
			Method: http.MethodGet,
			Path:   fmt.Sprintf("v2/events/%s/worker/status", eventID),
			QueryParams: map[string]string{
				"watch": "true",
			},
			SuccessCode: http.StatusOK,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	statusCh := make(chan WorkerStatus)
	errCh := make(chan error)

	go w.receiveStatusStream(ctx, resp.Body, statusCh, errCh)

	return statusCh, errCh, nil
}

func (w *workersClient) UpdateStatus(
	ctx context.Context,
	eventID string,
	status WorkerStatus,
) error {
	return w.ExecuteRequest(
		ctx,
		rm.OutboundRequest{
			Method:      http.MethodPut,
			Path:        fmt.Sprintf("v2/events/%s/worker/status", eventID),
			ReqBodyObj:  status,
			SuccessCode: http.StatusOK,
		},
	)
}

func (w *workersClient) Cleanup(ctx context.Context, eventID string) error {
	return w.ExecuteRequest(
		ctx,
		rm.OutboundRequest{
			Method:      http.MethodPut,
			Path:        fmt.Sprintf("v2/events/%s/worker/cleanup", eventID),
			SuccessCode: http.StatusOK,
		},
	)
}

func (w *workersClient) Jobs() JobsClient {
	return w.jobsClient
}

func (w *workersClient) receiveStatusStream(
	ctx context.Context,
	reader io.ReadCloser,
	statusCh chan<- WorkerStatus,
	errCh chan<- error,
) {
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	for {
		status := WorkerStatus{}
		if err := decoder.Decode(&status); err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
			return
		}
		select {
		case statusCh <- status:
		case <-ctx.Done():
			return
		}
	}
}
