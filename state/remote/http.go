package remote

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

func httpFactory(conf map[string]string) (Client, error) {
	address, ok := conf["address"]
	if !ok {
		return nil, fmt.Errorf("missing 'address' configuration")
	}

	url, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTTP URL: %s", err)
	}
	if url.Scheme != "http" && url.Scheme != "https" {
		return nil, fmt.Errorf("address must be HTTP or HTTPS")
	}

	client := &http.Client{}
	if skipRaw, ok := conf["skip_cert_verification"]; ok {
		skip, err := strconv.ParseBool(skipRaw)
		if err != nil {
			return nil, fmt.Errorf("skip_cert_verification must be boolean")
		}
		if skip {
			// Replace the client with one that ignores TLS verification
			client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}
	}

	return &HTTPClient{
		URL:    url,
		Client: client,
	}, nil
}

// HTTPClient is a remote client that stores data in Consul or HTTP REST.
type HTTPClient struct {
	URL    *url.URL
	Client *http.Client
}

func (c *HTTPClient) Get() (*Payload, error) {
	resp, err := c.Client.Get(c.URL.String())
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
	payload := &Payload{
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

func (c *HTTPClient) Put(data []byte) error {
	// Copy the target URL
	base := *c.URL

	// Generate the MD5
	hash := md5.Sum(data)
	b64 := base64.StdEncoding.EncodeToString(hash[:])

	/*
		// Set the force query parameter if needed
		if force {
			values := base.Query()
			values.Set("force", "true")
			base.RawQuery = values.Encode()
		}
	*/

	req, err := http.NewRequest("POST", base.String(), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("Failed to make HTTP request: %s", err)
	}

	// Prepare the request
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-MD5", b64)
	req.ContentLength = int64(len(data))

	// Make the request
	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to upload state: %v", err)
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

func (c *HTTPClient) Delete() error {
	req, err := http.NewRequest("DELETE", c.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("Failed to make HTTP request: %s", err)
	}

	// Make the request
	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to delete state: %s", err)
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
