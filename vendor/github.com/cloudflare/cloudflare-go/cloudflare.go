// Package cloudflare implements the Cloudflare v4 API.
package cloudflare

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

const apiURL = "https://api.cloudflare.com/client/v4"

// API holds the configuration for the current API client. A client should not
// be modified concurrently.
type API struct {
	APIKey     string
	APIEmail   string
	BaseURL    string
	headers    http.Header
	httpClient *http.Client
}

// New creates a new Cloudflare v4 API client.
func New(key, email string, opts ...Option) (*API, error) {
	if key == "" || email == "" {
		return nil, errors.New(errEmptyCredentials)
	}

	api := &API{
		APIKey:   key,
		APIEmail: email,
		BaseURL:  apiURL,
		headers:  make(http.Header),
	}

	err := api.parseOptions(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "options parsing failed")
	}

	// Fall back to http.DefaultClient if the package user does not provide
	// their own.
	if api.httpClient == nil {
		api.httpClient = http.DefaultClient
	}

	return api, nil
}

// ZoneIDByName retrieves a zone's ID from the name.
func (api *API) ZoneIDByName(zoneName string) (string, error) {
	res, err := api.ListZones(zoneName)
	if err != nil {
		return "", errors.Wrap(err, "ListZones command failed")
	}
	for _, zone := range res {
		if zone.Name == zoneName {
			return zone.ID, nil
		}
	}
	return "", errors.New("Zone could not be found")
}

// makeRequest makes a HTTP request and returns the body as a byte slice,
// closing it before returnng. params will be serialized to JSON.
func (api *API) makeRequest(method, uri string, params interface{}) ([]byte, error) {
	// Replace nil with a JSON object if needed
	var reqBody io.Reader
	if params != nil {
		json, err := json.Marshal(params)
		if err != nil {
			return nil, errors.Wrap(err, "error marshalling params to JSON")
		}
		reqBody = bytes.NewReader(json)
	} else {
		reqBody = nil
	}

	resp, err := api.request(method, uri, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read response body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusUnauthorized:
		return nil, errors.Errorf("HTTP status %d: invalid credentials", resp.StatusCode)
	case http.StatusForbidden:
		return nil, errors.Errorf("HTTP status %d: insufficient permissions", resp.StatusCode)
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout,
		522, 523, 524:
		return nil, errors.Errorf("HTTP status %d: service failure", resp.StatusCode)
	default:
		var s string
		if body != nil {
			s = string(body)
		}
		return nil, errors.Errorf("HTTP status %d: content %q", resp.StatusCode, s)
	}

	return body, nil
}

// request makes a HTTP request to the given API endpoint, returning the raw
// *http.Response, or an error if one occurred. The caller is responsible for
// closing the response body.
func (api *API) request(method, uri string, reqBody io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, api.BaseURL+uri, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request creation failed")
	}

	// Apply any user-defined headers first.
	req.Header = cloneHeader(api.headers)
	req.Header.Set("X-Auth-Key", api.APIKey)
	req.Header.Set("X-Auth-Email", api.APIEmail)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request failed")
	}

	return resp, nil
}

// cloneHeader returns a shallow copy of the header.
// copied from https://godoc.org/github.com/golang/gddo/httputil/header#Copy
func cloneHeader(header http.Header) http.Header {
	h := make(http.Header)
	for k, vs := range header {
		h[k] = vs
	}
	return h
}

// ResponseInfo contains a code and message returned by the API as errors or
// informational messages inside the response.
type ResponseInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Response is a template.  There will also be a result struct.  There will be a
// unique response type for each response, which will include this type.
type Response struct {
	Success  bool           `json:"success"`
	Errors   []ResponseInfo `json:"errors"`
	Messages []ResponseInfo `json:"messages"`
}

// ResultInfo contains metadata about the Response.
type ResultInfo struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Count   int `json:"count"`
	Total   int `json:"total_count"`
}
