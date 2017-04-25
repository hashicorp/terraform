package atlas

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-rootcerts"
)

const (
	// atlasDefaultEndpoint is the default base URL for connecting to Atlas.
	atlasDefaultEndpoint = "https://atlas.hashicorp.com"

	// atlasEndpointEnvVar is the environment variable that overrrides the
	// default Atlas address.
	atlasEndpointEnvVar = "ATLAS_ADDRESS"

	// atlasCAFileEnvVar is the environment variable that causes the client to
	// load trusted certs from a file
	atlasCAFileEnvVar = "ATLAS_CAFILE"

	// atlasCAPathEnvVar is the environment variable that causes the client to
	// load trusted certs from a directory
	atlasCAPathEnvVar = "ATLAS_CAPATH"

	// atlasTLSNoVerifyEnvVar disables TLS verification, similar to curl -k
	// This defaults to false (verify) and will change to true (skip
	// verification) with any non-empty value
	atlasTLSNoVerifyEnvVar = "ATLAS_TLS_NOVERIFY"

	// atlasTokenHeader is the header key used for authenticating with Atlas
	atlasTokenHeader = "X-Atlas-Token"
)

var projectURL = "https://github.com/hashicorp/atlas-go"
var userAgent = fmt.Sprintf("AtlasGo/1.0 (+%s; %s)",
	projectURL, runtime.Version())

// ErrAuth is the error returned if a 401 is returned by an API request.
var ErrAuth = fmt.Errorf("authentication failed")

// ErrNotFound is the error returned if a 404 is returned by an API request.
var ErrNotFound = fmt.Errorf("resource not found")

// RailsError represents an error that was returned from the Rails server.
type RailsError struct {
	Errors []string `json:"errors"`
}

// Error collects all of the errors in the RailsError and returns a comma-
// separated list of the errors that were returned from the server.
func (re *RailsError) Error() string {
	return strings.Join(re.Errors, ", ")
}

// Client represents a single connection to a Atlas API endpoint.
type Client struct {
	// URL is the full endpoint address to the Atlas server including the
	// protocol, port, and path.
	URL *url.URL

	// Token is the Atlas authentication token
	Token string

	// HTTPClient is the underlying http client with which to make requests.
	HTTPClient *http.Client

	// DefaultHeaders is a set of headers that will be added to every request.
	// This minimally includes the atlas user-agent string.
	DefaultHeader http.Header
}

// DefaultClient returns a client that connects to the Atlas API.
func DefaultClient() *Client {
	atlasEndpoint := os.Getenv(atlasEndpointEnvVar)
	if atlasEndpoint == "" {
		atlasEndpoint = atlasDefaultEndpoint
	}

	client, err := NewClient(atlasEndpoint)
	if err != nil {
		panic(err)
	}

	return client
}

// NewClient creates a new Atlas Client from the given URL (as a string). If
// the URL cannot be parsed, an error is returned. The HTTPClient is set to
// an empty http.Client, but this can be changed programmatically by setting
// client.HTTPClient. The user can also programmatically set the URL as a
// *url.URL.
func NewClient(urlString string) (*Client, error) {
	if len(urlString) == 0 {
		return nil, fmt.Errorf("client: missing url")
	}

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	token := os.Getenv("ATLAS_TOKEN")
	if token != "" {
		log.Printf("[DEBUG] using ATLAS_TOKEN (%s)", maskString(token))
	}

	client := &Client{
		URL:           parsedURL,
		Token:         token,
		DefaultHeader: make(http.Header),
	}

	client.DefaultHeader.Set("User-Agent", userAgent)

	if err := client.init(); err != nil {
		return nil, err
	}

	return client, nil
}

// init() sets defaults on the client.
func (c *Client) init() error {
	c.HTTPClient = cleanhttp.DefaultClient()
	tlsConfig := &tls.Config{}
	if os.Getenv(atlasTLSNoVerifyEnvVar) != "" {
		tlsConfig.InsecureSkipVerify = true
	}
	err := rootcerts.ConfigureTLS(tlsConfig, &rootcerts.Config{
		CAFile: os.Getenv(atlasCAFileEnvVar),
		CAPath: os.Getenv(atlasCAPathEnvVar),
	})
	if err != nil {
		return err
	}
	t := cleanhttp.DefaultTransport()
	t.TLSClientConfig = tlsConfig
	c.HTTPClient.Transport = t
	return nil
}

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

// Request creates a new HTTP request using the given verb and sub path.
func (c *Client) Request(verb, spath string, ro *RequestOptions) (*http.Request, error) {
	log.Printf("[INFO] request: %s %s", verb, spath)

	// Ensure we have a RequestOptions struct (passing nil is an acceptable)
	if ro == nil {
		ro = new(RequestOptions)
	}

	// Create a new URL with the appended path
	u := *c.URL
	u.Path = path.Join(c.URL.Path, spath)

	// Add the token and other params
	if c.Token != "" {
		log.Printf("[DEBUG] request: appending token (%s)", maskString(c.Token))
		if ro.Headers == nil {
			ro.Headers = make(map[string]string)
		}

		ro.Headers[atlasTokenHeader] = c.Token
	}

	return c.rawRequest(verb, &u, ro)
}

func (c *Client) putFile(rawURL string, r io.Reader, size int64) error {
	log.Printf("[INFO] putting file: %s", rawURL)

	url, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	request, err := c.rawRequest("PUT", url, &RequestOptions{
		Body:       r,
		BodyLength: size,
	})
	if err != nil {
		return err
	}

	if _, err := checkResp(c.HTTPClient.Do(request)); err != nil {
		return err
	}

	return nil
}

// rawRequest accepts a verb, URL, and RequestOptions struct and returns the
// constructed http.Request and any errors that occurred
func (c *Client) rawRequest(verb string, u *url.URL, ro *RequestOptions) (*http.Request, error) {
	if verb == "" {
		return nil, fmt.Errorf("client: missing verb")
	}

	if u == nil {
		return nil, fmt.Errorf("client: missing URL.url")
	}

	if ro == nil {
		return nil, fmt.Errorf("client: missing RequestOptions")
	}

	// Add the token and other params
	var params = make(url.Values)
	for k, v := range ro.Params {
		params.Add(k, v)
	}
	u.RawQuery = params.Encode()

	// Create the request object
	request, err := http.NewRequest(verb, u.String(), ro.Body)
	if err != nil {
		return nil, err
	}

	// set our default headers first
	for k, v := range c.DefaultHeader {
		request.Header[k] = v
	}

	// Add any request headers (auth will be here if set)
	for k, v := range ro.Headers {
		request.Header.Add(k, v)
	}

	// Add content-length if we have it
	if ro.BodyLength > 0 {
		request.ContentLength = ro.BodyLength
	}

	log.Printf("[DEBUG] raw request: %#v", request)

	return request, nil
}

// checkResp wraps http.Client.Do() and verifies that the request was
// successful. A non-200 request returns an error formatted to included any
// validation problems or otherwise.
func checkResp(resp *http.Response, err error) (*http.Response, error) {
	// If the err is already there, there was an error higher up the chain, so
	// just return that
	if err != nil {
		return resp, err
	}

	log.Printf("[INFO] response: %d (%s)", resp.StatusCode, resp.Status)
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		log.Printf("[ERR] response: error copying response body")
	} else {
		log.Printf("[DEBUG] response: %s", buf.String())

		// We are going to reset the response body, so we need to close the old
		// one or else it will leak.
		resp.Body.Close()
		resp.Body = &bytesReadCloser{&buf}
	}

	switch resp.StatusCode {
	case 200:
		return resp, nil
	case 201:
		return resp, nil
	case 202:
		return resp, nil
	case 204:
		return resp, nil
	case 400:
		return nil, parseErr(resp)
	case 401:
		return nil, ErrAuth
	case 404:
		return nil, ErrNotFound
	case 422:
		return nil, parseErr(resp)
	default:
		return nil, fmt.Errorf("client: %s", resp.Status)
	}
}

// parseErr is used to take an error JSON response and return a single string
// for use in error messages.
func parseErr(r *http.Response) error {
	re := &RailsError{}

	if err := decodeJSON(r, &re); err != nil {
		return fmt.Errorf("error decoding JSON body: %s", err)
	}

	return re
}

// decodeJSON is used to JSON decode a body into an interface.
func decodeJSON(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// bytesReadCloser is a simple wrapper around a bytes buffer that implements
// Close as a noop.
type bytesReadCloser struct {
	*bytes.Buffer
}

func (nrc *bytesReadCloser) Close() error {
	// we don't actually have to do anything here, since the buffer is just some
	// data in memory  and the error is initialized to no-error
	return nil
}
