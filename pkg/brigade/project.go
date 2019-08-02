package brigade

import (
	"encoding/json"
	"strings"

	"github.com/kelseyhightower/envconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type project struct {
	// ID is the project ID. This is used to load the Project object from
	// configuration.
	ID string `envconfig:"BRIGADE_PROJECT_ID" required:"true"`
	// Namespace is the Kubernetes namespace in which new jobs should be created.
	// The Brigade worker must have write access to this namespace.
	Namespace string `envconfig:"BRIGADE_PROJECT_NAMESPACE" required:"true"`
	// ServiceAccount is the service account to use.
	ServiceAccount string `envconfig:"BRIGADE_SERVICE_ACCOUNT" default:"brigade-worker"` // nolint: lll
}

// Project represents Brigade project configuration.
type Project struct {
	ID                  string
	Name                string
	Repo                Repository
	Kubernetes          KubernetesConfig
	Secrets             map[string]string
	AllowPrivilegedJobs bool
	AllowHostMounts     bool
}

// Repository represents VCS-related projects configuration.
type Repository struct {
	Name              string
	CloneURL          string
	SSHKey            string
	Token             string
	InitGitSubmodules bool
}

// KubernetesConfig represents Kubernetes-related project configuration.
type KubernetesConfig struct {
	Namespace                         string
	VCSSidecar                        string
	VCSSidecarResourcesLimitsCPU      string
	VCSSidecarResourcesLimitsMemory   string
	VCSSidecarResourcesRequestsCPU    string
	VCSSidecarResourcesRequestsMemory string
	BuildStorageSize                  string
	BuildStorageClass                 string
	ServiceAccount                    string
	ImagePullSecrets                  []string
}

// GetProjectFromEnvironmentAndSecret returns a Project object with values
// derived from environment variables and a project-specific Kubernetes secret.
func GetProjectFromEnvironmentAndSecret(
	kubeClient kubernetes.Interface,
) (Project, error) {
	internalP := project{}
	err := envconfig.Process("", &internalP)
	if err != nil {
		return Project{}, err
	}
	projectSecret, err := kubeClient.CoreV1().Secrets(internalP.Namespace).Get(
		internalP.ID,
		metav1.GetOptions{},
	)
	if err != nil {
		return Project{}, err
	}
	// nolint: lll
	p := Project{
		ID:   projectSecret.GetName(),
		Name: string(projectSecret.Data["repository"]),
		Kubernetes: KubernetesConfig{
			Namespace:                         projectSecret.GetNamespace(),
			BuildStorageSize:                  string(projectSecret.Data["buildStorageSize"]),
			ServiceAccount:                    internalP.ServiceAccount,
			VCSSidecar:                        string(projectSecret.Data["vcsSidecar"]),
			VCSSidecarResourcesLimitsCPU:      string(projectSecret.Data["vcsSidecarResources.limits.cpu"]),
			VCSSidecarResourcesLimitsMemory:   string(projectSecret.Data["vcsSidecarResources.limits.memory"]),
			VCSSidecarResourcesRequestsCPU:    string(projectSecret.Data["vcsSidecarResources.requests.cpu"]),
			VCSSidecarResourcesRequestsMemory: string(projectSecret.Data["vcsSidecarResources.requests.memory"]),
			BuildStorageClass:                 string(projectSecret.Data["kubernetes.buildStorageClass"]),
			ImagePullSecrets:                  strings.Split(string(projectSecret.Data["imagePullSecrets"]), ","),
		},
		Repo: Repository{
			Name:              projectSecret.GetAnnotations()["projectName"],
			CloneURL:          string(projectSecret.Data["cloneURL"]),
			InitGitSubmodules: string(projectSecret.Data["initGitSubmodules"]) == "true",
			SSHKey:            string(projectSecret.Data["sshKey"]),
			Token:             string(projectSecret.Data["github.token"]),
		},
		Secrets:             map[string]string{},
		AllowPrivilegedJobs: string(projectSecret.Data["allowPrivilegedJobs"]) == "true",
		AllowHostMounts:     string(projectSecret.Data["allowHostMounts"]) == "true",
	}
	if p.Kubernetes.BuildStorageSize == "" {
		p.Kubernetes.BuildStorageSize = "50Mi"
	}
	secretsBytes, ok := projectSecret.Data["secrets"]
	if ok {
		if ierr := json.Unmarshal(secretsBytes, &p.Secrets); ierr != nil {
			return p, ierr
		}
	}
	return p, nil
}
