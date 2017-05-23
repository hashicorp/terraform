package fastly

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RequestOptions is the list of options to pass to the request.
type RequestOptions struct {
	// Params is a map of key-value pairs that will be added to the Request.
	Params map[string]string

	// Headers is a map of key-value pairs that will be added to the Request.
	Headers map[string]string

	// Body is an io.Reader object that will be streamed or uploaded with the
	// Request. BodyLength is the final size of the Body.
	Body       io.Reader
	BodyLength int64
}

// RawRequest accepts a verb, URL, and RequestOptions struct and returns the
// constructed http.Request and any errors that occurred
func (c *Client) RawRequest(verb, p string, ro *RequestOptions) (*http.Request, error) {
	// Ensure we have request options.
	if ro == nil {
		ro = new(RequestOptions)
	}

	// Append the path to the URL.
	u := *c.url
	u.Path = strings.TrimRight(c.url.Path, "/") + "/" + strings.TrimLeft(p, "/")

	// Add the token and other params.
	var params = make(url.Values)
	for k, v := range ro.Params {
		params.Add(k, v)
	}
	u.RawQuery = params.Encode()

	// Create the request object.
	request, err := http.NewRequest(verb, u.String(), ro.Body)
	if err != nil {
		return nil, err
	}

	// Set the API key.
	if len(c.apiKey) > 0 {
		request.Header.Set(APIKeyHeader, c.apiKey)
	}

	// Set the User-Agent.
	request.Header.Set("User-Agent", UserAgent)

	// Add any custom headers.
	for k, v := range ro.Headers {
		request.Header.Add(k, v)
	}

	// Add Content-Length if we have it.
	if ro.BodyLength > 0 {
		request.ContentLength = ro.BodyLength
	}

	return request, nil
}
