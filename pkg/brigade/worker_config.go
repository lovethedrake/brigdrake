package brigade

import (
	"github.com/kelseyhightower/envconfig"
)

// WorkerConfig represents worker configuration specified by the Brigade
// controller when it launches the worker.
type WorkerConfig struct {
	DefaultBuildStorageClass string `envconfig:"BRIGADE_DEFAULT_BUILD_STORAGE_CLASS"` // nolint: lll
}

// NewWorkerConfigWithDefaults returns a WorkerConfig object with default values
// already applied. Callers are then free to set custom values for the remaining
// fields and/or override default values.
func NewWorkerConfigWithDefaults() WorkerConfig {
	return WorkerConfig{}
}

// GetWorkerConfigFromEnvironment returns a WorkerConfig object with values
// derived from environment variables.
func GetWorkerConfigFromEnvironment() (WorkerConfig, error) {
	wc := NewWorkerConfigWithDefaults()
	err := envconfig.Process("", &wc)
	return wc, err
}
