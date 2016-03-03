package dnsmadeeasy

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SandboxURL is the URL of the DNS Made Easy Sandbox
const SandboxURL = "http://api.sandbox.dnsmadeeasy.com/V2.0"

// Client provides a client to the dnsmadeeasy API
type Client struct {
	// API Key
	AKey string

	// Secret Key
	SKey string

	// URL to the API to use
	URL string

	// HttpClient is the client to use. Default will be
	// used if not provided.
	HTTP *http.Client
}

// Body is the body of a request
type Body map[string]interface{}

// Error is the error format that they return
// to us if there is a problem
type Error struct {
	Errors []string `json:"error"`
}

// Join joins all the errors together, separated by spaces.
func (d *Error) Join() string {
	return strings.Join(d.Errors, " ")
}

// NewClient returns a new dnsmadeeasy client. It requires an API key and
// secret key. You can generate them by visiting the Config, Account
// Information section of the dnsmadeeasy control panel for your account.
func NewClient(akey string, skey string) (*Client, error) {
	client := Client{
		AKey: akey,
		SKey: skey,
		URL:  "https://api.dnsmadeeasy.com/V2.0",
		HTTP: http.DefaultClient,
	}
	return &client, nil
}

// NewRequest creates a new request with the params
func (c *Client) NewRequest(method, path string, body *bytes.Buffer,
	requestDate string) (*http.Request, error) {

	url, err := url.Parse(c.URL + path)
	if err != nil {
		return nil, fmt.Errorf("Error parsing base URL: %s", err)
	}

	// Build the request
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %s", err)
	}

	// Calculate the hexadecimal HMAC SHA1 of requestDate using sKey
	key := []byte(c.SKey)
	h := hmac.New(sha1.New, key)
	if len(requestDate) == 0 {
		requestDate = time.Now().UTC().Format(http.TimeFormat)
	}
	h.Write([]byte(requestDate))
	hmacString := hex.EncodeToString(h.Sum(nil))

	// Add the authorization header
	req.Header.Add("X-Dnsme-Apikey", c.AKey)
	req.Header.Add("X-Dnsme-Requestdate", requestDate)
	req.Header.Add("X-Dnsme-Hmac", hmacString)
	req.Header.Add("Accept", "application/json")

	// If it's a not a get, add a content-type
	if method != "GET" {
		req.Header.Add("Content-Type", "application/json")
	}
	return req, nil
}

// parseError is used to take an error json resp
// and return a single string for use in error messages
func parseError(resp *http.Response) error {
	dnsError := Error{}
	err := decodeBody(resp, &dnsError)

	// if there was an error decoding the body, just return that
	if err != nil {
		return fmt.Errorf("Error parsing error body for non-200 request: %s", err)
	}
	return fmt.Errorf("API Error (%d): %s", resp.StatusCode, dnsError.Join())
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

	if resp.StatusCode/100 == 2 {
		return resp, nil
	} else if resp.StatusCode == 404 {
		return nil, fmt.Errorf("Not found")
	}
	return nil, fmt.Errorf("API Error: %s", resp.Status)
}
