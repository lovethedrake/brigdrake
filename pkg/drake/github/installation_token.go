package github

import (
	"context"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// getInstallationToken returns an installation token for the given appID, and
// installationID. It uses the provided ASCII-armored x509 certificate key to
// sign a JSON web token that is then exchanged for the installation token.
func getInstallationToken(
	appID int64,
	installationID int64,
	keyPEM []byte,
) (string, error) {
	// Construct a JSON web token to use as the bearer token to create a new
	// client that we can use to, in turn, create the installation token.
	jsonWebToken, err := getSignedJSONWebToken(appID, keyPEM)
	if err != nil {
		return "", err
	}
	githubClient := newClientFromBearerToken(jsonWebToken)
	installationToken, _, err := githubClient.Apps.CreateInstallationToken(
		context.TODO(),
		installationID,
	)
	if err != nil {
		return "", err
	}
	return installationToken.GetToken(), nil
}

// getSignedJSONWebToken constructs, signs, and returns a JSON web token.
func getSignedJSONWebToken(appID int64, keyPEM []byte) (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyPEM)
	if err != nil {
		return "", err
	}
	now := time.Now()
	return jwt.NewWithClaims(
		jwt.SigningMethodRS256,
		jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(5 * time.Minute).Unix(),
			Issuer:    strconv.FormatInt(appID, 10),
		},
	).SignedString(key)
}
