package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/hashicorp/terraform/terraform"
)

var (
	// ErrConflict is used to indicate the upload was rejected
	// due to a conflict on the state
	ErrConflict = fmt.Errorf("Conflicting state file")
)

// RemoteStatePayload is used to return the remote state
// along with associated meta data when we do a remote fetch.
type RemoteStatePayload struct {
	MD5   []byte
	State []byte
}

// remoteStateClient is used to interact with a remote state store
// using the API
type remoteStateClient struct {
	conf *terraform.RemoteState
}

// URL is used to return an appropriate URL to hit for the
// given server and remote name
func (r *remoteStateClient) URL() (*url.URL, error) {
	// Get the base URL configuration
	base, err := url.Parse(r.conf.Server)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse remote server '%s': %v", r.conf.Server, err)
	}

	// Compute the full path by just appending the name
	base.Path = path.Join(base.Path, r.conf.Name)

	// Add the request token if any
	if r.conf.AuthToken != "" {
		values := base.Query()
		values.Set("access_token", r.conf.AuthToken)
		base.RawQuery = values.Encode()
	}
	return base, nil
}

// GetState is used to read the remote state
func (r *remoteStateClient) GetState() (*RemoteStatePayload, error) {
	// Get the target URL
	base, err := r.URL()
	if err != nil {
		return nil, err
	}

	// Request the url
	resp, err := http.Get(base.String())
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
		return nil, fmt.Errorf("Remote server requires authentication")
	case http.StatusForbidden:
		return nil, fmt.Errorf("Invalid authentication")
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("Remote server reporting internal error")
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

	// Check if this is Consul
	if raw := resp.Header.Get("X-Consul-Index"); raw != "" {
		// Check if we used the ?raw query param, otherwise decode
		if _, ok := base.Query()["raw"]; !ok {
			type kv struct {
				Value []byte
			}
			var values []*kv
			if err := json.Unmarshal(buf.Bytes(), &values); err != nil {
				return nil, fmt.Errorf("Failed to decode Consul response: %v", err)
			}

			// Setup the reader to pull the value from Consul
			payload.State = values[0].Value
		}
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

// Put is used to update the remote state
func (r *remoteStateClient) PutState(state []byte, force bool) error {
	// Get the target URL
	base, err := r.URL()
	if err != nil {
		return err
	}

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
	client := http.Client{}
	req, err := http.NewRequest("PUT", base.String(), bytes.NewReader(state))
	if err != nil {
		return fmt.Errorf("Failed to make HTTP request: %v", err)
	}

	// Prepare the request
	req.Header.Set("Content-MD5", b64)
	req.ContentLength = int64(len(state))

	// Make the request
	resp, err := client.Do(req)
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
	case http.StatusUnauthorized:
		return fmt.Errorf("Remote server requires authentication")
	case http.StatusForbidden:
		return fmt.Errorf("Invalid authentication")
	case http.StatusInternalServerError:
		return fmt.Errorf("Remote server reporting internal error")
	default:
		return fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}
