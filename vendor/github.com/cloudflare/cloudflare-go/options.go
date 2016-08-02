package cloudflare

import "net/http"

// Option is a functional option for configuring the API client.
type Option func(*API) error

// HTTPClient accepts a custom *http.Client for making API calls.
func HTTPClient(client *http.Client) Option {
	return func(api *API) error {
		api.httpClient = client
		return nil
	}
}

// Headers allows you to set custom HTTP headers when making API calls (e.g. for
// satisfying HTTP proxies, or for debugging).
func Headers(headers http.Header) Option {
	return func(api *API) error {
		api.headers = headers
		return nil
	}
}

// parseOptions parses the supplied options functions and returns a configured
// *API instance.
func (api *API) parseOptions(opts ...Option) error {
	// Range over each options function and apply it to our API type to
	// configure it. Options functions are applied in order, with any
	// conflicting options overriding earlier calls.
	for _, option := range opts {
		err := option(api)
		if err != nil {
			return err
		}
	}

	return nil
}
