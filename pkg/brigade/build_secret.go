package brigade

import (
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (e *executor) createBuildSecret(project Project, event Event) error {
	secret := &v1.Secret{
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
	if _, err := e.kubeClient.CoreV1().Secrets(
		project.Kubernetes.Namespace,
	).Create(secret); err != nil {
		return errors.Wrapf(
			err,
			"error creating secret for build \"%s\"",
			event.BuildID,
		)
	}
	return nil
}

func (e *executor) destroyBuildSecret(project Project, event Event) error {
	if err := e.kubeClient.CoreV1().Secrets(
		project.Kubernetes.Namespace,
	).Delete(
		strings.ToLower(event.BuildID),
		&metav1.DeleteOptions{},
	); err != nil {
		return errors.Wrapf(
			err,
			"error deleting build secret for build \"%s\"",
			event.BuildID,
		)
	}
	return nil
}
