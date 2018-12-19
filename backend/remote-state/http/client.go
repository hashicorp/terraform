package http

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

// httpClient is a remote client that stores data in Consul or HTTP REST.
type httpClient struct {
	// Update & Retrieve
	URL          *url.URL
	UpdateMethod string

	// Locking
	LockURL      *url.URL
	LockMethod   string
	UnlockURL    *url.URL
	UnlockMethod string

	// HTTP
	Client   *retryablehttp.Client
	Username string
	Password string

	lockID       string
	jsonLockInfo []byte
}

func (c *httpClient) httpRequest(method string, url *url.URL, data *[]byte, what string) (*http.Response, error) {
	// If we have data we need a reader
	var reader io.Reader = nil
	if data != nil {
		reader = bytes.NewReader(*data)
	}

	// Create the request
	req, err := retryablehttp.NewRequest(method, url.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to make %s HTTP request: %s", what, err)
	}
	// Setup basic auth
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	// Work with data/body
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(*data))

		// Generate the MD5
		hash := md5.Sum(*data)
		b64 := base64.StdEncoding.EncodeToString(hash[:])
		req.Header.Set("Content-MD5", b64)
	}

	// Make the request
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to %s: %v", what, err)
	}

	return resp, nil
}

func (c *httpClient) Lock(info *state.LockInfo) (string, error) {
	if c.LockURL == nil {
		return "", nil
	}
	c.lockID = ""

	jsonLockInfo := info.Marshal()
	resp, err := c.httpRequest(c.LockMethod, c.LockURL, &jsonLockInfo, "lock")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		c.lockID = info.ID
		c.jsonLockInfo = jsonLockInfo
		return info.ID, nil
	case http.StatusUnauthorized:
		return "", fmt.Errorf("HTTP remote state endpoint requires auth")
	case http.StatusForbidden:
		return "", fmt.Errorf("HTTP remote state endpoint invalid auth")
	case http.StatusConflict, http.StatusLocked:
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("HTTP remote state already locked, failed to read body")
		}
		existing := state.LockInfo{}
		err = json.Unmarshal(body, &existing)
		if err != nil {
			return "", fmt.Errorf("HTTP remote state already locked, failed to unmarshal body")
		}
		return "", fmt.Errorf("HTTP remote state already locked: ID=%s", existing.ID)
	default:
		return "", fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}

func (c *httpClient) Unlock(id string) error {
	if c.UnlockURL == nil {
		return nil
	}

	resp, err := c.httpRequest(c.UnlockMethod, c.UnlockURL, &c.jsonLockInfo, "unlock")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}

func (c *httpClient) Get() (*remote.Payload, error) {
	resp, err := c.httpRequest("GET", c.URL, nil, "get state")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle the common status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Handled after
	case http.StatusNoContent:
		return nil, nil
	case http.StatusNotFound:
		return nil, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("HTTP remote state endpoint requires auth")
	case http.StatusForbidden:
		return nil, fmt.Errorf("HTTP remote state endpoint invalid auth")
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("HTTP remote state internal server error")
	default:
		return nil, fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}

	// Read in the body
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}

	// Create the payload
	payload := &remote.Payload{
		Data: buf.Bytes(),
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	// Check for the MD5
	if raw := resp.Header.Get("Content-MD5"); raw != "" {
		md5, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf(
				"Failed to decode Content-MD5 '%s': %s", raw, err)
		}

		payload.MD5 = md5
	} else {
		// Generate the MD5
		hash := md5.Sum(payload.Data)
		payload.MD5 = hash[:]
	}

	return payload, nil
}

func (c *httpClient) Put(data []byte) error {
	// Copy the target URL
	base := *c.URL

	if c.lockID != "" {
		query := base.Query()
		query.Set("ID", c.lockID)
		base.RawQuery = query.Encode()
	}

	/*
		// Set the force query parameter if needed
		if force {
			values := base.Query()
			values.Set("force", "true")
			base.RawQuery = values.Encode()
		}
	*/

	var method string = "POST"
	if c.UpdateMethod != "" {
		method = c.UpdateMethod
	}
	resp, err := c.httpRequest(method, &base, &data, "upload state")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle the error codes
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
}

func (c *httpClient) Delete() error {
	// Make the request
	resp, err := c.httpRequest("DELETE", c.URL, nil, "delete state")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle the error codes
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
}
