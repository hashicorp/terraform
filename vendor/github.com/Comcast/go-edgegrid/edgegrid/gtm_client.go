package edgegrid

import "net/http"

// GTMClient is an Akamai GTM API client.
// https://developer.akamai.com/api/luna/config-gtm/overview.html
type GTMClient struct {
	Credentials *AuthCredentials
	HTTPClient  *http.Client
}

// GetCredentials takes a GTMClient and returns its Credentials.
func (c GTMClient) GetCredentials() *AuthCredentials {
	return c.Credentials
}

// GetHTTPClient takes a GTMClient and returns its HTTPClient.
func (c GTMClient) GetHTTPClient() *http.Client {
	return c.HTTPClient
}
