package mailgun

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/hashicorp/go-cleanhttp"
)

// Client provides a client to the Mailgun API
type Client struct {
	// Authentication key
	ApiKey string

	// URL to the API to use
	URL string

	// HttpClient is the client to use. Default will be
	// used if not provided.
	Http *http.Client
}

// MailgunError is the error format that they return
// to us if there is a problem
type MailgunError struct {
	Message string `json:"message"`
}

// NewClient returns a new mailgun client,
// requires an api key.
func NewClient(key string) (*Client, error) {
	// If it exists, grab teh token from the environment
	if key == "" {
		key = os.Getenv("MAILGUN_API_KEY")
	}

	client := Client{
		ApiKey: key,
		URL:    "https://api.mailgun.net/v2",
		Http:   cleanhttp.DefaultClient(),
	}

	return &client, nil
}

// Creates a new request with the params
func (c *Client) NewRequest(params map[string]string, method string, endpoint string) (*http.Request, error) {
	p := url.Values{}
	u, err := url.Parse(c.URL + endpoint)

	if err != nil {
		return nil, fmt.Errorf("Error parsing base URL: %s", err)
	}

	// Build up our request parameters
	for k, v := range params {
		p.Add(k, v)
	}

	// Add the params to our URL
	u.RawQuery = p.Encode()

	// Build the request
	req, err := http.NewRequest(method, u.String(), nil)

	if err != nil {
		return nil, fmt.Errorf("Error creating request: %s", err)
	}

	// Add the basic auth header
	req.SetBasicAuth("api", c.ApiKey)

	return req, nil
}

// parseErr is used to take an error json resp
// and return a single string for use in error messages
func parseErr(resp *http.Response) error {
	errBody := new(MailgunError)

	err := decodeBody(resp, &errBody)

	// if there was an error decoding the body, just return that
	if err != nil {
		return fmt.Errorf("Error parsing error body for non-200 request: %s", err)
	}

	return fmt.Errorf("API Error %s: %s", resp.Status, errBody.Message)
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
	case i == 400:
		return nil, parseErr(resp)
	case i == 401:
		return nil, parseErr(resp)
	case i == 402:
		return nil, parseErr(resp)
	case i == 422:
		return nil, parseErr(resp)
	default:
		return nil, fmt.Errorf("API Error: %s", resp.Status)
	}
}
