package executor

import (
	"strings"

	"github.com/lovethedrake/brigdrake/pkg/brigade"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func createBuildSecret(
	project brigade.Project,
	event brigade.Event,
	kubeClient kubernetes.Interface,
) error {
	secret := buildBuildSecret(project, event)
	if _, err := kubeClient.CoreV1().Secrets(
		project.Kubernetes.Namespace,
	).Create(secret); err != nil {
		return errors.Wrapf(
			err,
			"error creating secret for build %q",
			event.BuildID,
		)
	}
	return nil
}

func buildBuildSecret(project brigade.Project, event brigade.Event) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.ToLower(event.BuildID),
			Labels: map[string]string{
				"heritage":  "brigade",
				"component": "buildSecret",
				"project":   project.ID,
				"worker":    strings.ToLower(event.WorkerID),
				"build":     strings.ToLower(event.BuildID),
			},
		},
		StringData: project.Secrets,
	}
}

func destroyBuildSecret(
	project brigade.Project,
	event brigade.Event,
	kubeClient kubernetes.Interface,
) error {
	if err := kubeClient.CoreV1().Secrets(
		project.Kubernetes.Namespace,
	).Delete(
		strings.ToLower(event.BuildID),
		&metav1.DeleteOptions{},
	); err != nil {
		return errors.Wrapf(
			err,
			"error deleting build secret for build %q",
			event.BuildID,
		)
	}
	return nil
}
