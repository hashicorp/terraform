package spotinst

import (
	"io"
	"net/http"
	"net/url"
)

// request is used to help build up a request
type request struct {
	config *clientConfig
	method string
	url    *url.URL
	params url.Values
	body   io.Reader
	header http.Header
	obj    interface{}
}

// toHTTP converts the request to an HTTP request
func (r *request) toHTTP() (*http.Request, error) {
	// Encode the query parameters
	r.url.RawQuery = r.params.Encode()

	// Check if we should encode the body
	if r.body == nil && r.obj != nil {
		if b, err := encodeBody(r.obj); err != nil {
			return nil, err
		} else {
			r.body = b
		}
	}

	// Create the HTTP request
	req, err := http.NewRequest(r.method, r.url.RequestURI(), r.body)
	if err != nil {
		return nil, err
	}

	req.URL.Host = r.url.Host
	req.URL.Scheme = r.url.Scheme
	req.Host = r.url.Host
	req.Header = r.header

	req.Header.Set("Content-Type", r.config.contentType)
	req.Header.Add("Accept", r.config.contentType)
	req.Header.Add("User-Agent", r.config.userAgent)

	return req, nil
}
