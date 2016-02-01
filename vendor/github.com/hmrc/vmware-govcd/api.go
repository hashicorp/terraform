/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

// Package govcloudair provides a simple binding for vCloud Air / vCloud Director REST APIs.
package govcd

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	types "github.com/hmrc/vmware-govcd/types/v56"
)

// Client provides a client to vCloud Air, values can be populated automatically using the Authenticate method.
type Client struct {
	APIVersion    string      // The API version required
	VCDToken      string      // Access Token (authorization header)
	VCDAuthHeader string      // Authorization header
	VCDVDCHREF    url.URL     // HREF of the backend VDC you're using
	Http          http.Client // HttpClient is the client to use. Default will be used if not provided.
}

// NewRequest creates a new HTTP request and applies necessary auth headers if
// set.
func (c *Client) NewRequest(params map[string]string, method string, u url.URL, body io.Reader) *http.Request {

	p := url.Values{}

	// Build up our request parameters
	for k, v := range params {
		p.Add(k, v)
	}

	// Add the params to our URL
	u.RawQuery = p.Encode()

	// Build the request, no point in checking for errors here as we're just
	// passing a string version of an url.URL struct and http.NewRequest returns
	// error only if can't process an url.ParseRequestURI().
	req, _ := http.NewRequest(method, u.String(), body)

	if c.VCDAuthHeader != "" && c.VCDToken != "" {
		// Add the authorization header
		req.Header.Add(c.VCDAuthHeader, c.VCDToken)
		// Add the Accept header for VCD
		req.Header.Add("Accept", "application/*+xml;version="+c.APIVersion)
	}

	return req

}

// parseErr takes an error XML resp and returns a single string for use in error messages.
func parseErr(resp *http.Response) error {

	errBody := new(types.Error)

	// if there was an error decoding the body, just return that
	if err := decodeBody(resp, errBody); err != nil {
		return fmt.Errorf("error parsing error body for non-200 request: %s", err)
	}

	return fmt.Errorf("API Error: %d: %s", errBody.MajorErrorCode, errBody.Message)
}

// decodeBody is used to XML decode a response body
func decodeBody(resp *http.Response, out interface{}) error {

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Unmarshal the XML.
	if err = xml.Unmarshal(body, &out); err != nil {
		return err
	}

	return nil
}

// checkResp wraps http.Client.Do() and verifies the request, if status code
// is 2XX it passes back the response, if it's a known invalid status code it
// parses the resultant XML error and returns a descriptive error, if the
// status code is not handled it returns a generic error with the status code.
func checkResp(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return resp, err
	}

	switch i := resp.StatusCode; {
	// Valid request, return the response.
	case i == 200 || i == 201 || i == 202 || i == 204:
		return resp, nil
	// Invalid request, parse the XML error returned and return it.
	case i == 400 || i == 401 || i == 403 || i == 404 || i == 405 || i == 406 || i == 409 || i == 415 || i == 500 || i == 503 || i == 504:
		return nil, parseErr(resp)
	// Unhandled response.
	default:
		return nil, fmt.Errorf("unhandled API response, please report this issue, status code: %s", resp.Status)
	}
}
