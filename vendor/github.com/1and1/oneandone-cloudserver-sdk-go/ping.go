package oneandone

import "net/http"

// GET /ping
// Returns "PONG" if API is running
func (api *API) Ping() ([]string, error) {
	url := createUrl(api, pingPathSegment)
	result := []string{}
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GET /ping_auth
// Returns "PONG" if the API is running and the authentication token is valid
func (api *API) PingAuth() ([]string, error) {
	url := createUrl(api, pingAuthPathSegment)
	result := []string{}
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return result, nil
}
