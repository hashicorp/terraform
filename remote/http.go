package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HTTPRemoteClient implements the RemoteClient interface
// for an HTTP compatible server.
type HTTPRemoteClient struct {
	// url is the URL that we GET / POST / DELETE to
	url *url.URL
}

func NewHTTPRemoteClient(conf map[string]string) (*HTTPRemoteClient, error) {
	client := &HTTPRemoteClient{}
	if err := client.validateConfig(conf); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *HTTPRemoteClient) validateConfig(conf map[string]string) error {
	urlRaw, ok := conf["address"]
	if !ok || urlRaw == "" {
		return fmt.Errorf("missing 'address' configuration")
	}
	url, err := url.Parse(urlRaw)
	if err != nil {
		return fmt.Errorf("failed to parse url: %v", err)
	}
	if url.Scheme != "http" && url.Scheme != "https" {
		return fmt.Errorf("invalid url: %s", url)
	}
	c.url = url
	return nil
}

func (c *HTTPRemoteClient) GetState() (*RemoteStatePayload, error) {
	// Request the url
	resp, err := http.Get(c.url.String())
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
		return nil, ErrRequireAuth
	case http.StatusForbidden:
		return nil, ErrInvalidAuth
	case http.StatusInternalServerError:
		return nil, ErrRemoteInternal
	default:
		return nil, fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}

	// Read in the body
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %v", err)
	}

	// Create the payload
	payload := &RemoteStatePayload{
		State: buf.Bytes(),
	}

	// Check for the MD5
	if raw := resp.Header.Get("Content-MD5"); raw != "" {
		md5, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode Content-MD5 '%s': %v", raw, err)
		}
		payload.MD5 = md5

	} else {
		// Generate the MD5
		hash := md5.Sum(payload.State)
		payload.MD5 = hash[:md5.Size]
	}

	return payload, nil
}

func (c *HTTPRemoteClient) PutState(state []byte, force bool) error {
	// Copy the target URL
	base := new(url.URL)
	*base = *c.url

	// Generate the MD5
	hash := md5.Sum(state)
	b64 := base64.StdEncoding.EncodeToString(hash[:md5.Size])

	// Set the force query parameter if needed
	if force {
		values := base.Query()
		values.Set("force", "true")
		base.RawQuery = values.Encode()
	}

	// Make the HTTP client and request
	req, err := http.NewRequest("POST", base.String(), bytes.NewReader(state))
	if err != nil {
		return fmt.Errorf("Failed to make HTTP request: %v", err)
	}

	// Prepare the request
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-MD5", b64)
	req.ContentLength = int64(len(state))

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to upload state: %v", err)
	}
	defer resp.Body.Close()

	// Handle the error codes
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return ErrConflict
	case http.StatusPreconditionFailed:
		return ErrServerNewer
	case http.StatusUnauthorized:
		return ErrRequireAuth
	case http.StatusForbidden:
		return ErrInvalidAuth
	case http.StatusInternalServerError:
		return ErrRemoteInternal
	default:
		return fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}

func (c *HTTPRemoteClient) DeleteState() error {
	// Make the HTTP request
	req, err := http.NewRequest("DELETE", c.url.String(), nil)
	if err != nil {
		return fmt.Errorf("Failed to make HTTP request: %v", err)
	}

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to delete state: %v", err)
	}
	defer resp.Body.Close()

	// Handle the error codes
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return nil
	case http.StatusUnauthorized:
		return ErrRequireAuth
	case http.StatusForbidden:
		return ErrInvalidAuth
	case http.StatusInternalServerError:
		return ErrRemoteInternal
	default:
		return fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}
