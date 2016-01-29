package dnsimple

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
)

// Client provides a client to the DNSimple API
type Client struct {
	// Access Token
	Token string

	// User Email
	Email string

	// Domain Token
	DomainToken string

	// URL to the DO API to use
	URL string

	// HttpClient is the client to use. A client with
	// default values will be used if not provided.
	Http *http.Client
}

// DNSimpleError is the error format that they return
// to us if there is a problem
type DNSimpleError struct {
	Errors map[string][]string `json:"errors"`
}

func (d *DNSimpleError) Join() string {
	var errs []string

	for k, v := range d.Errors {
		errs = append(errs, fmt.Sprintf("%s errors: %s", k, strings.Join(v, ", ")))
	}

	return strings.Join(errs, ", ")
}

// NewClient returns a new dnsimple client,
// requires an authorization token. You can generate
// an OAuth token by visiting the Apps & API section
// of the DNSimple control panel for your account.
func NewClient(email string, token string) (*Client, error) {
	client := Client{
		Token: token,
		Email: email,
		URL:   "https://api.dnsimple.com/v1",
		Http:  cleanhttp.DefaultClient(),
	}
	return &client, nil
}

func NewClientWithDomainToken(domainToken string) (*Client, error) {
	client := Client{
		DomainToken: domainToken,
		URL:         "https://api.dnsimple.com/v1",
		Http:        cleanhttp.DefaultClient(),
	}
	return &client, nil
}

// Creates a new request with the params
func (c *Client) NewRequest(body map[string]interface{}, method string, endpoint string) (*http.Request, error) {
	u, err := url.Parse(c.URL + endpoint)

	if err != nil {
		return nil, fmt.Errorf("Error parsing base URL: %s", err)
	}

	rBody, err := encodeBody(body)
	if err != nil {
		return nil, fmt.Errorf("Error encoding request body: %s", err)
	}

	// Build the request
	req, err := http.NewRequest(method, u.String(), rBody)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %s", err)
	}

	// Add the authorization header
	if c.DomainToken != "" {
		req.Header.Add("X-DNSimple-Domain-Token", c.DomainToken)
	} else {
		req.Header.Add("X-DNSimple-Token", fmt.Sprintf("%s:%s", c.Email, c.Token))
	}
	req.Header.Add("Accept", "application/json")

	// If it's a not a get, add a content-type
	if method != "GET" {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil

}

// parseErr is used to take an error json resp
// and return a single string for use in error messages
func parseErr(resp *http.Response) error {
	dnsError := DNSimpleError{}

	err := decodeBody(resp, &dnsError)

	// if there was an error decoding the body, just return that
	if err != nil {
		return fmt.Errorf("Error parsing error body for non-200 request: %s", err)
	}

	return fmt.Errorf("API Error: %s", dnsError.Join())
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	if err = json.Unmarshal(body, &out); err != nil {
		return err
	}

	return nil
}

func encodeBody(obj interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

// checkResp wraps http.Client.Do() and verifies that the
// request was successful. A non-200 request returns an error
// formatted to included any validation problems or otherwise
func checkResp(resp *http.Response, err error) (*http.Response, error) {
	// If the err is already there, there was an error higher
	// up the chain, so just return that
	if err != nil {
		return resp, err
	}

	switch i := resp.StatusCode; {
	case i == 200:
		return resp, nil
	case i == 201:
		return resp, nil
	case i == 202:
		return resp, nil
	case i == 204:
		return resp, nil
	case i == 422:
		return nil, fmt.Errorf("API Error: %s", resp.Status)
	case i == 400:
		return nil, parseErr(resp)
	default:
		return nil, fmt.Errorf("API Error: %s", resp.Status)
	}
}
