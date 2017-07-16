// Package backblaze B2 API for Golang
package backblaze // import "gopkg.in/kothar/go-backblaze.v0"

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/pquerna/ffjson/ffjson"
)

const (
	b2Host = "https://api.backblazeb2.com"
	v1     = "/b2api/v1/"
)

// Credentials are the identification required by the Backblaze B2 API
//
// The account ID is a 12-digit hex number that you can get from
// your account page on backblaze.com.
//
// The application key is a 40-digit hex number that you can get from
// your account page on backblaze.com.
type Credentials struct {
	AccountID      string
	ApplicationKey string
}

// B2 implements a B2 API client. Do not modify state concurrently.
type B2 struct {
	Credentials

	// If true, don't retry requests if authorization has expired
	NoRetry bool

	// If true, display debugging information about API calls
	Debug bool

	// State
	mutex      sync.Mutex
	host       string
	auth       *authorizationState
	httpClient http.Client
}

// The current auth state of the client. Can be individually invalidated by
// requests which fail, tringgering a reauth the next time its validity is
// checked
type authorizationState struct {
	sync.Mutex
	*authorizeAccountResponse

	valid bool
}

func (a *authorizationState) isValid() bool {
	if a == nil {
		return false
	}

	a.Lock()
	defer a.Unlock()

	return a.valid
}

// Marks the authorization as invalid. This will result in a new Authorization
// on the next API call.
func (a *authorizationState) invalidate() {
	if a == nil {
		return
	}

	a.Lock()
	defer a.Unlock()

	a.valid = false
	a.authorizeAccountResponse = nil
}

// NewB2 creates a new Client for accessing the B2 API.
// The AuthorizeAccount method will be called immediately.
func NewB2(creds Credentials) (*B2, error) {
	c := &B2{
		Credentials: creds,
	}

	// Authorize account
	if err := c.AuthorizeAccount(); err != nil {
		return nil, err
	}

	return c, nil
}

// AuthorizeAccount is used to log in to the B2 API.
func (c *B2) AuthorizeAccount() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.internalAuthorizeAccount()
}

// The body of AuthorizeAccount without a mutex
func (c *B2) internalAuthorizeAccount() error {
	if c.host == "" {
		c.host = b2Host
	}

	req, err := http.NewRequest("GET", c.host+v1+"b2_authorize_account", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.AccountID, c.ApplicationKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	authResponse := &authorizeAccountResponse{}
	if err = c.parseResponse(resp, authResponse, nil); err != nil {
		return err
	}

	// Store token
	c.auth = &authorizationState{
		authorizeAccountResponse: authResponse,
		valid: true,
	}

	return nil
}

// DownloadURL returns the URL prefix needed to construct download links.
// Bucket.FileURL will costruct a full URL for given file names.
func (c *B2) DownloadURL() (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.auth.isValid() {
		if err := c.internalAuthorizeAccount(); err != nil {
			return "", err
		}
	}
	return c.auth.DownloadURL, nil
}

// Create an authorized request using the client's credentials
func (c *B2) authRequest(method, apiPath string, body io.Reader) (*http.Request, *authorizationState, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.auth.isValid() {
		if c.Debug {
			log.Println("No valid authorization token, re-authorizing client")
		}
		if err := c.internalAuthorizeAccount(); err != nil {
			return nil, nil, err
		}
	}

	path := c.auth.APIEndpoint + v1 + apiPath

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("Authorization", c.auth.AuthorizationToken)

	if c.Debug {
		log.Printf("authRequest: %s %s\n", method, req.URL)
	}

	return req, c.auth, nil
}

// Dispatch an authorized API GET request
func (c *B2) authGet(apiPath string) (*http.Response, *authorizationState, error) {
	req, auth, err := c.authRequest("GET", apiPath, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.httpClient.Do(req)
	return resp, auth, err
}

// Dispatch an authorized POST request
func (c *B2) authPost(apiPath string, body io.Reader) (*http.Response, *authorizationState, error) {
	req, auth, err := c.authRequest("POST", apiPath, body)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.httpClient.Do(req)
	return resp, auth, err
}

// Looks for an error message in the response body and parses it into a
// B2Error object
func (c *B2) parseError(body []byte) error {
	b2err := &B2Error{}
	if ffjson.Unmarshal(body, b2err) != nil {
		return nil
	}
	return b2err
}

// Attempts to parse a response body into the provided result struct
func (c *B2) parseResponse(resp *http.Response, result interface{}, auth *authorizationState) error {
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if c.Debug {
		log.Printf("Response: %s", body)
	}

	// Check response code
	switch resp.StatusCode {
	case 200: // Response is OK
	case 401:
		auth.invalidate()
		if err := c.parseError(body); err != nil {
			return err
		}
		return &B2Error{
			Code:    "UNAUTHORIZED",
			Message: "The account ID is wrong, the account does not have B2 enabled, or the application key is not valid",
			Status:  resp.StatusCode,
		}
	default:
		if err := c.parseError(body); err != nil {
			return err
		}
		return &B2Error{
			Code:    "UNKNOWN",
			Message: "Unrecognised status code",
			Status:  resp.StatusCode,
		}
	}

	return ffjson.Unmarshal(body, result)
}

// Perform a B2 API request with the provided request and response objects
func (c *B2) apiRequest(apiPath string, request interface{}, response interface{}) error {
	body, err := ffjson.Marshal(request)
	if err != nil {
		return err
	}
	defer ffjson.Pool(body)

	if c.Debug {
		log.Println("----")
		log.Printf("apiRequest: %s %s", apiPath, body)
	}

	err = c.tryAPIRequest(apiPath, body, response)

	// Retry after non-fatal errors
	if b2err, ok := err.(*B2Error); ok {
		if !b2err.IsFatal() && !c.NoRetry {
			if c.Debug {
				log.Printf("Retrying request %q due to error: %v", apiPath, err)
			}

			return c.tryAPIRequest(apiPath, body, response)
		}
	}
	return err
}

func (c *B2) tryAPIRequest(apiPath string, body []byte, response interface{}) error {
	resp, auth, err := c.authPost(apiPath, bytes.NewReader(body))
	if err != nil {
		if c.Debug {
			log.Println("B2.post returned an error: ", err)
		}
		return err
	}

	return c.parseResponse(resp, response, auth)
}
