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

// RemoteStatePayload is used to return the remote state
// along with associated meta data when we do a remote fetch.
type RemoteStatePayload struct {
	MD5   []byte
	State []byte
}

// GetState is used to read the remote state
func GetState(conf *terraform.RemoteState) (*RemoteStatePayload, error) {
	// Get the base URL configuration
	base, err := url.Parse(conf.Server)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse remote server '%s': %v", conf.Server, err)
	}

	// Compute the full path by just appending the name
	base.Path = path.Join(base.Path, conf.Name)

	// Add the request token if any
	if conf.AuthToken != "" {
		values := base.Query()
		values.Set("access_token", conf.AuthToken)
		base.RawQuery = values.Encode()
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
