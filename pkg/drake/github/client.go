package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v18/github"
	"golang.org/x/oauth2"
)

// newClientFromBearerToken returns a new github.Client for the given bearer
// token.
func newClientFromBearerToken(token string) *github.Client {
	return github.NewClient(
		oauth2.NewClient(
			context.TODO(),
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token,
				},
			),
		),
	)
}

// newClientFromKeyPEM returns a new github.Client for the given appID and
// installationID. It uses the provided ASCII-armored x509 certificate key to
// sign a JSON web token that is then exchanged for an installation token that
// will ultimately be used by the returned client.
func newClientFromKeyPEM(
	appID int64,
	installationID int64,
	keyPEM []byte,
) (*github.Client, error) {
	installationToken, err := getInstallationToken(
		appID,
		installationID,
		keyPEM,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to negotiate an installation token: %s", err)
	}
	return github.NewClient(
		oauth2.NewClient(
			context.TODO(),
			oauth2.StaticTokenSource(
				&oauth2.Token{
					TokenType:   "token", // This type indicates an installation token
					AccessToken: installationToken,
				},
			),
		),
	), nil
}
