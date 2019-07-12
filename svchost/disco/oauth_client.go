package disco

import (
	"net/url"

	"golang.org/x/oauth2"
)

// OAuthClient represents an OAuth client configuration, which is used for
// unusual services that require an entire OAuth client configuration as part
// of their service discovery, rather than just a URL.
type OAuthClient struct {
	// ID is the identifier for the client, to be used as "client_id" in
	// OAuth requests.
	ID string

	// Authorization URL is the URL of the authorization endpoint that must
	// be used for this OAuth client, as defined in the OAuth2 specifications.
	AuthorizationURL *url.URL

	// Token URL is the URL of the token endpoint that must be used for this
	// OAuth client, as defined in the OAuth2 specifications.
	TokenURL *url.URL

	// MinPort and MaxPort define a range of TCP ports on localhost that this
	// client is able to use as redirect_uri in an authorization request.
	// Terraform will select a port from this range for the temporary HTTP
	// server it creates to receive the authorization response, giving
	// a URL like http://localhost:NNN/ where NNN is the selected port number.
	//
	// Terraform will reject any port numbers in this range less than 1024,
	// to respect the common convention (enforced on some operating systems)
	// that lower port numbers are reserved for "privileged" services.
	MinPort, MaxPort uint16
}

// Endpoint returns an oauth2.Endpoint value ready to be used with the oauth2
// library, representing the URLs from the receiver.
func (c *OAuthClient) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  c.AuthorizationURL.String(),
		TokenURL: c.TokenURL.String(),

		// We don't actually auth because we're not a server-based OAuth client,
		// so this instead just means that we include client_id as an argument
		// in our requests.
		AuthStyle: oauth2.AuthStyleInParams,
	}
}
