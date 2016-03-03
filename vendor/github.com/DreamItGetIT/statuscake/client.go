package statuscake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const apiBaseURL = "https://www.statuscake.com/API"

type responseBody struct {
	io.Reader
}

func (r *responseBody) Close() error {
	return nil
}

// Auth wraps the authorisation headers required for each request
type Auth struct {
	Username string
	Apikey   string
}

func (a *Auth) validate() error {
	e := make(ValidationError)

	if a.Username == "" {
		e["Username"] = "is required"
	}

	if a.Apikey == "" {
		e["Apikey"] = "is required"
	}

	if len(e) > 0 {
		return e
	}

	return nil
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type apiClient interface {
	get(string, url.Values) (*http.Response, error)
	delete(string, url.Values) (*http.Response, error)
	put(string, url.Values) (*http.Response, error)
}

// Client is the http client that wraps the remote API.
type Client struct {
	c           httpClient
	username    string
	apiKey      string
	testsClient Tests
}

// New returns a new Client
func New(auth Auth) (*Client, error) {
	if err := auth.validate(); err != nil {
		return nil, err
	}

	return &Client{
		c:        &http.Client{},
		username: auth.Username,
		apiKey:   auth.Apikey,
	}, nil
}

func (c *Client) newRequest(method string, path string, v url.Values, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s", apiBaseURL, path)
	if v != nil {
		url = fmt.Sprintf("%s?%s", url, v.Encode())
	}

	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	r.Header.Set("Username", c.username)
	r.Header.Set("API", c.apiKey)

	return r, nil
}

func (c *Client) doRequest(r *http.Request) (*http.Response, error) {
	resp, err := c.c.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &httpError{
			status:     resp.Status,
			statusCode: resp.StatusCode,
		}
	}

	var aer autheticationErrorResponse

	// We read and save the response body so that if we don't have error messages
	// we can set it again for future usage
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &aer)
	if err == nil && aer.ErrNo == 0 && aer.Error != "" {
		return nil, &AuthenticationError{
			errNo:   aer.ErrNo,
			message: aer.Error,
		}
	}

	resp.Body = &responseBody{
		Reader: bytes.NewReader(b),
	}

	return resp, nil
}

func (c *Client) get(path string, v url.Values) (*http.Response, error) {
	r, err := c.newRequest("GET", path, v, nil)
	if err != nil {
		return nil, err
	}

	return c.doRequest(r)
}

func (c *Client) put(path string, v url.Values) (*http.Response, error) {
	r, err := c.newRequest("PUT", path, nil, strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		return nil, err
	}

	return c.doRequest(r)
}

func (c *Client) delete(path string, v url.Values) (*http.Response, error) {
	r, err := c.newRequest("DELETE", path, v, nil)
	if err != nil {
		return nil, err
	}

	return c.doRequest(r)
}

// Tests returns a client that implements the `Tests` API.
func (c *Client) Tests() Tests {
	if c.testsClient == nil {
		c.testsClient = newTests(c)
	}

	return c.testsClient
}
