package rata

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RequestGenerator creates http.Request objects with the correct path and method
// pre-filled for the given route object.  You can also set the the host and,
// optionally, any headers you would like included with every request.
type RequestGenerator struct {
	Header http.Header
	host   string
	routes Routes
}

// NewRequestGenerator creates a RequestGenerator for a given host and route set.
// Host is of the form "http://example.com".
func NewRequestGenerator(host string, routes Routes) *RequestGenerator {
	return &RequestGenerator{
		Header: make(http.Header),
		host:   host,
		routes: routes,
	}
}

// CreateRequest creates a new http Request for the matching handler. If the
// request cannot be created, either because the handler does not exist or because
// the given params do not match the params the route requires, then CreateRequest
// returns an error.
func (r *RequestGenerator) CreateRequest(
	name string,
	params Params,
	body io.Reader,
) (*http.Request, error) {
	route, ok := r.routes.FindRouteByName(name)
	if !ok {
		return &http.Request{}, fmt.Errorf("No route exists with the name %s", name)
	}
	path, err := route.CreatePath(params)
	if err != nil {
		return &http.Request{}, err
	}

	url := r.host + "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequest(route.Method, url, body)
	if err != nil {
		return &http.Request{}, err
	}

	for key, values := range r.Header {
		req.Header[key] = make([]string, len(values))
		copy(req.Header[key], values)
	}

	return req, nil
}
