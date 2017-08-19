package remote

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform/state"
)

func httpFactory(conf map[string]string) (Client, error) {
	address, ok := conf["address"]
	if !ok {
		return nil, fmt.Errorf("missing 'address' configuration")
	}

	storeURL, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address URL: %s", err)
	}
	if storeURL.Scheme != "http" && storeURL.Scheme != "https" {
		return nil, fmt.Errorf("address must be HTTP or HTTPS")
	}
	storeMethod, ok := conf["store_method"]
	if !ok {
		storeMethod = "POST"
	}

	var lockURL *url.URL
	lockAddress, ok := conf["lock_address"]
	if ok {
		lockURL, err := url.Parse(lockAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse lockAddress URL: %s", err)
		}
		if lockURL.Scheme != "http" && lockURL.Scheme != "https" {
			return nil, fmt.Errorf("lockAddress must be HTTP or HTTPS")
		}
	} else {
		lockURL = nil
	}
	lockMethod, ok := conf["lock_method"]
	if !ok {
		lockMethod = "LOCK"
	}

	var unlockURL *url.URL
	unlockAddress, ok := conf["unlock_address"]
	if ok {
		unlockURL, err := url.Parse(unlockAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unlockAddress URL: %s", err)
		}
		if unlockURL.Scheme != "http" && unlockURL.Scheme != "https" {
			return nil, fmt.Errorf("unlockAddress must be HTTP or HTTPS")
		}
	} else {
		unlockURL = nil
	}
	unlockMethod, ok := conf["unlock_method"]
	if !ok {
		unlockMethod = "UNLOCK"
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

	ret := &HTTPClient{
		URL:         storeURL,
		StoreMethod: storeMethod,

		LockURL:      lockURL,
		LockMethod:   lockMethod,
		UnlockURL:    unlockURL,
		UnlockMethod: unlockMethod,

		Client:   client,
		Username: conf["username"],
		Password: conf["password"],
	}

	return ret, nil
}

// HTTPClient is a remote client that stores data in Consul or HTTP REST.
type HTTPClient struct {
	// Store & Retrieve
	URL         *url.URL
	StoreMethod string

	// Locking
	LockURL      *url.URL
	LockMethod   string
	UnlockURL    *url.URL
	UnlockMethod string

	// HTTP
	Client   *http.Client
	Username string
	Password string

	lockID       string
	jsonLockInfo []byte
}

func (c *HTTPClient) httpRequest(method string, url *url.URL, data *[]byte, what string) (*http.Response, error) {
	// If we have data we need a reader
	var reader io.Reader = nil
	if data != nil {
		reader = bytes.NewReader(*data)
	}

	// Create the request
	req, err := http.NewRequest(method, url.String(), reader)
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

func (c *HTTPClient) Lock(info *state.LockInfo) (string, error) {
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
	case http.StatusConflict:
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

func (c *HTTPClient) Unlock(id string) error {
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

func (c *HTTPClient) Get() (*Payload, error) {
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
	if c.StoreMethod != "" {
		method = c.StoreMethod
	}
	resp, err := c.httpRequest(method, &base, &data, "upload state")
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

func (c *HTTPClient) Delete() error {
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
