package brigade

import (
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (b *buildExecutor) createBuildSecret() error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.ToLower(b.event.BuildID),
			Labels: map[string]string{
				"heritage":  "brigade",
				"component": "buildSecret",
				"project":   b.project.ID,
				"worker":    strings.ToLower(b.event.WorkerID),
				"build":     strings.ToLower(b.event.BuildID),
			},
		},
		StringData: b.project.Secrets,
	}
	if _, err := b.kubeClient.CoreV1().Secrets(
		b.project.Kubernetes.Namespace,
	).Create(secret); err != nil {
		return errors.Wrapf(
			err,
			"error creating secret for build \"%s\"",
			b.event.BuildID,
		)
	}
	return nil
}

func (b *buildExecutor) destroyBuildSecret() error {
	if err := b.kubeClient.CoreV1().Secrets(
		b.project.Kubernetes.Namespace,
	).Delete(
		strings.ToLower(b.event.BuildID),
		&metav1.DeleteOptions{},
	); err != nil {
		return errors.Wrapf(
			err,
			"error deleting build secret for build \"%s\"",
			b.event.BuildID,
		)
	}
	return nil
}
