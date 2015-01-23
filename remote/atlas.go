package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

const (
	// defaultAtlasServer is used when no address is given
	defaultAtlasServer = "https://atlas.hashicorp.com/"
)

// AtlasRemoteClient implements the RemoteClient interface
// for an Atlas compatible server.
type AtlasRemoteClient struct {
	server      string
	serverURL   *url.URL
	user        string
	name        string
	accessToken string
}

func NewAtlasRemoteClient(conf map[string]string) (*AtlasRemoteClient, error) {
	client := &AtlasRemoteClient{}
	if err := client.validateConfig(conf); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *AtlasRemoteClient) validateConfig(conf map[string]string) error {
	server, ok := conf["address"]
	if !ok || server == "" {
		server = defaultAtlasServer
	}
	url, err := url.Parse(server)
	if err != nil {
		return err
	}
	c.server = server
	c.serverURL = url

	token, ok := conf["access_token"]
	if token == "" {
		token = os.Getenv("ATLAS_TOKEN")
		ok = true
	}
	if !ok || token == "" {
		return fmt.Errorf(
			"missing 'access_token' configuration or ATLAS_TOKEN environmental variable")
	}
	c.accessToken = token

	name, ok := conf["name"]
	if !ok || name == "" {
		return fmt.Errorf("missing 'name' configuration")
	}

	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return fmt.Errorf("malformed name '%s'", name)
	}
	c.user = parts[0]
	c.name = parts[1]
	return nil
}

func (c *AtlasRemoteClient) GetState() (*RemoteStatePayload, error) {
	// Make the HTTP request
	req, err := http.NewRequest("GET", c.url().String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to make HTTP request: %v", err)
	}

	// Request the url
	resp, err := http.DefaultClient.Do(req)
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

func (c *AtlasRemoteClient) PutState(state []byte, force bool) error {
	// Get the target URL
	base := c.url()

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
	req, err := http.NewRequest("PUT", base.String(), bytes.NewReader(state))
	if err != nil {
		return fmt.Errorf("Failed to make HTTP request: %v", err)
	}

	// Prepare the request
	req.Header.Set("Content-MD5", b64)
	req.Header.Set("Content-Type", "application/json")
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

func (c *AtlasRemoteClient) DeleteState() error {
	// Make the HTTP request
	req, err := http.NewRequest("DELETE", c.url().String(), nil)
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
	return nil
}

func (c *AtlasRemoteClient) url() *url.URL {
	return &url.URL{
		Scheme:   c.serverURL.Scheme,
		Host:     c.serverURL.Host,
		Path:     path.Join("api/v1/terraform/state", c.user, c.name),
		RawQuery: fmt.Sprintf("access_token=%s", c.accessToken),
	}
}
