package edgegrid

import "net/http"

// PAPIClient is an Akamai PAPI API client.
// https://developer.akamai.com/api/luna/papi/overview.html
type PAPIClient struct {
	Credentials *AuthCredentials
	HTTPClient  *http.Client
}

// GetCredentials takes a PAPIClient and returns its credentials.
func (c PAPIClient) GetCredentials() *AuthCredentials {
	return c.Credentials
}

// GetHTTPClient takes a PAPIClient and returns its HTTPClient.
func (c PAPIClient) GetHTTPClient() *http.Client {
	return c.HTTPClient
}
