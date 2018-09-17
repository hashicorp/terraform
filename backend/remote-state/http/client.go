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

	multierror "github.com/hashicorp/go-multierror"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

// RemoteClient is used by "state/remote".State to read and write
// blobs representing state.
// Implements "state/remote".ClientLocker
type RemoteClient struct {
	client *http.Client

	address       string
	updateMethod  string
	lockAddress   string
	unlockMethod  string
	lockMethod    string
	unlockAddress string
	username      string
	password      string

	lockID       string
	jsonLockInfo []byte
}

// Get state file and return the payload.
func (c *RemoteClient) Get() (*remote.Payload, error) {
	// Convert address to type URL
	addressURL, _ := url.Parse(c.address)

	resp, err := c.httpRequest(http.MethodGet, addressURL, nil, "get state")
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
	default:
		return nil, fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}

	// Read in the body
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}
	// If there was no data, then return nil
	bufBytes := buf.Bytes()
	if len(bufBytes) == 0 {
		return nil, nil
	}

	// Create the payload
	payload := &remote.Payload{
		Data: bufBytes,
	}

	md5 := md5.Sum(bufBytes)

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
		payload.MD5 = md5[:]
	}

	return payload, nil
}

// Put state file
func (c *RemoteClient) Put(data []byte) error {
	// Copy the target URL
	addressURL, _ := url.Parse(c.address)

	base := *addressURL

	if c.lockID != "" {
		query := base.Query()
		query.Set("ID", c.lockID)
		base.RawQuery = query.Encode()
	}

	resp, err := c.httpRequest(c.updateMethod, &base, &data, "upload state")
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

// Delete state file
func (c *RemoteClient) Delete() error {
	// Make the request
	addressURL, _ := url.Parse(c.address)
	resp, err := c.httpRequest(http.MethodDelete, addressURL, nil, "delete state")
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

// Lock writes to a lock file, ensuring file creation. Returns the generation number, which must be passed to Unlock().
func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {

	lockURL, _ := url.Parse(c.lockAddress)
	c.lockID = ""

	jsonLockInfo := info.Marshal()

	resp, err := c.httpRequest(c.lockMethod, lockURL, &jsonLockInfo, "lock")
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		c.lockID = info.ID
		c.jsonLockInfo = jsonLockInfo
		return info.ID, nil
	case http.StatusConflict, http.StatusLocked:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("HTTP remote state already locked, failed to read body")
		}
		existing := state.LockInfo{}
		err = json.Unmarshal(body, &existing)
		if err != nil {
			return "", fmt.Errorf("HTTP remote state already locked, failed to unmarshal body")
		}
		return "", c.lockError(fmt.Errorf("HTTP remote state already locked: ID=%s", existing.ID))
	default:
		return "", fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}

// Unlock the state with id from lock.
func (c *RemoteClient) Unlock(id string) error {

	lockErr := &state.LockError{}

	lockInfo, err := c.lockInfo()
	if err != nil {
		lockErr.Err = fmt.Errorf("failed to retrieve lock info: %s", err)
		return lockErr
	}
	lockErr.Info = lockInfo

	if lockInfo.ID != id {
		lockErr.Err = fmt.Errorf("lock id %q does not match existing lock", id)
		return lockErr
	}

	unlockURL, _ := url.Parse(c.unlockAddress)
	resp, err := c.httpRequest(c.unlockMethod, unlockURL, &c.jsonLockInfo, "unlock")
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

// lockError appends the lockID and lockInfo in case of error
func (c *RemoteClient) lockError(err error) *state.LockError {
	lockErr := &state.LockError{
		Err: err,
	}

	info, infoErr := c.lockInfo()
	if infoErr != nil {
		lockErr.Err = multierror.Append(lockErr.Err, infoErr)
	} else {
		lockErr.Info = info
	}
	return lockErr
}

// lockInfo reads the lock file, parses its contents and returns the parsed
// LockInfo struct.
func (c *RemoteClient) lockInfo() (*state.LockInfo, error) {
	// Convert address to type URL
	lockURL, _ := url.Parse(c.lockAddress)

	resp, err := c.httpRequest(http.MethodGet, lockURL, nil, "get lock")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Read in the body
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, fmt.Errorf("Failed to read lock file: %s", err)
	}

	info := &state.LockInfo{}

	switch resp.StatusCode {
	case http.StatusOK:
		readBuf, err := ioutil.ReadAll(buf)
		if err != nil {
			return nil, fmt.Errorf("ioutil.ReadAll(%T) error: %v", buf, err)
		}
		if err := json.Unmarshal(readBuf, info); err != nil {
			return nil, err
		}
		return info, nil
	default:
		return nil, fmt.Errorf("Unexpected HTTP response code %d", resp.StatusCode)
	}
}

// httpRequest is used to make http call with the given method on the given URL and returning the response.
func (c *RemoteClient) httpRequest(method string, url *url.URL, data *[]byte, what string) (*http.Response, error) {
	// If we have data we need a reader
	var reader io.Reader
	if data != nil {
		reader = bytes.NewReader(*data)
	}
	// Create the request
	req, err := http.NewRequest(method, url.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to make %s HTTP request: %s", what, err)
	}
	// Setup basic auth
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
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
	return c.client.Do(req)
}
