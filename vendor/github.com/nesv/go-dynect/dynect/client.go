package dynect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	DynAPIPrefix = "https://api.dynect.net/REST"
)

// A client for use with DynECT's REST API.
type Client struct {
	Token        string
	CustomerName string
	httpclient   *http.Client
	verbose      bool
}

// Creates a new Httpclient.
func NewClient(customerName string) *Client {
	return &Client{
		CustomerName: customerName,
		httpclient:   &http.Client{}}
}

// Enable, or disable verbose output from the client.
//
// This will enable (or disable) logging messages that explain what the client
// is about to do, like the endpoint it is about to make a request to. If the
// request fails with an unexpected HTTP response code, then the response body
// will be logged out, as well.
func (c *Client) Verbose(p bool) {
	c.verbose = p
}

// Establishes a new session with the DynECT API.
func (c *Client) Login(username, password string) error {
	var req = LoginBlock{
		Username:     username,
		Password:     password,
		CustomerName: c.CustomerName}

	var resp LoginResponse

	err := c.Do("POST", "Session", req, &resp)
	if err != nil {
		return err
	}

	c.Token = resp.Data.Token
	return nil
}

func (c *Client) LoggedIn() bool {
	return len(c.Token) > 0
}

func (c *Client) Logout() error {
	return c.Do("DELETE", "Session", nil, nil)
}

func (c *Client) Do(method, endpoint string, requestData, responseData interface{}) error {
	// Throw an error if the user tries to make a request if the client is
	// logged out/unauthenticated, but make an exemption for when the
	// caller is trying to log in.
	if !c.LoggedIn() && method != "POST" && endpoint != "Session" {
		return errors.New("Will not perform request; httpclient is closed")
	}

	var err error

	// Marshal the request data into a byte slice.
	if c.verbose {
		log.Println("Marshaling request data")
	}
	var js []byte
	if requestData != nil {
		js, err = json.Marshal(requestData)
	} else {
		js = []byte("")
	}
	if err != nil {
		return err
	}

	// Create a new http.Request object, and set the necessary headers to
	// authorize the request, and specify the content type.
	url := fmt.Sprintf("%s/%s", DynAPIPrefix, endpoint)
	var req *http.Request
	req, err = http.NewRequest(method, url, bytes.NewReader(js))
	if err != nil {
		return err
	}
	req.Header.Set("Auth-Token", c.Token)
	req.Header.Set("Content-Type", "application/json")

	if c.verbose {
		log.Printf("Making %s request to %q", method, url)
	}

	var resp *http.Response
	resp, err = c.httpclient.Do(req)
	if err != nil {
		if c.verbose {
			respBody, _ := ioutil.ReadAll(resp.Body)
			log.Printf("%s", string(respBody))
		}
		return err
	} else if resp.StatusCode != 200 {
		if c.verbose {
			// Print out the response body.
			respBody, _ := ioutil.ReadAll(resp.Body)
			log.Printf("%s", string(respBody))
		}
		return errors.New(fmt.Sprintf("Bad response, got %q", resp.Status))
	}

	// Unmarshal the response data into the provided struct.
	if c.verbose {
		log.Println("Reading in response data")
	}
	var respBody []byte
	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if c.verbose {
		log.Println("Unmarshaling response data")
	}
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		respBody, _ := ioutil.ReadAll(resp.Body)
		if resp.ContentLength == 0 || resp.ContentLength == -1 {
			log.Println("Zero-length content body")
		} else {
			log.Printf("%s", string(respBody))
		}
	}

	return err
}
