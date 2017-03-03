// Package password implements the OAuth2.0 "password credentials" token flow.
// See https://tools.ietf.org/html/rfc6749#section-4.3
package password

import (
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// Config describes a Resource Owner Password Credentials OAuth2 flow, with the
// client application information, resource owner credentials and the server's
// endpoint URLs.
type Config struct {
	// ClientID is the application's ID.
	ClientID string

	// ClientSecret is the application's secret.
	ClientSecret string

	// Resource owner username
	Username string

	// Resource owner password
	Password string

	// Endpoint contains the resource server's token endpoint
	// URLs. These are constants specific to each server and are
	// often available via site-specific packages, such as
	// google.Endpoint or github.Endpoint.
	Endpoint oauth2.Endpoint

	// Scope specifies optional requested permissions.
	Scopes []string
}

// Client returns an HTTP client using the provided token.
// The token will auto-refresh as necessary. The underlying
// HTTP transport will be obtained using the provided context.
// The returned client and its Transport should not be modified.
func (c *Config) Client(ctx context.Context) *http.Client {
	return oauth2.NewClient(ctx, c.TokenSource(ctx))
}

// TokenSource returns a TokenSource that returns t until t expires,
// automatically refreshing it as necessary using the provided context and the
// client ID and client secret.
//
// Most users will use Config.Client instead.
func (c *Config) TokenSource(ctx context.Context) oauth2.TokenSource {
	source := &tokenSource{
		ctx:  ctx,
		conf: c,
	}
	return oauth2.ReuseTokenSource(nil, source)
}

type tokenSource struct {
	ctx  context.Context
	conf *Config
}

// Token refreshes the token by using a new password credentials request.
// tokens received this way do not include a refresh token
func (c *tokenSource) Token() (*oauth2.Token, error) {
	config := oauth2.Config{
		ClientID:     c.conf.ClientID,
		ClientSecret: c.conf.ClientSecret,
		Endpoint:     c.conf.Endpoint,
		Scopes:       c.conf.Scopes,
	}
	return config.PasswordCredentialsToken(c.ctx, c.conf.Username, c.conf.Password)
}
