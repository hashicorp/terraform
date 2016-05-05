package cloudapi

import (
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// Key represent a public key
type Key struct {
	Name        string // Name for the key
	Fingerprint string // Key Fingerprint
	Key         string // OpenSSH formatted public key
}

/*func (k Key) Equals(other Key) bool {
	if k.Name == other.Name && k.Fingerprint == other.Fingerprint && k.Key == other.Key {
		return true
	}
	return false
}*/

// CreateKeyOpts represent the option that can be specified
// when creating a new key.
type CreateKeyOpts struct {
	Name string `json:"name"` // Name for the key, optional
	Key  string `json:"key"`  // OpenSSH formatted public key
}

// ListKeys returns a list of public keys registered with a specific account.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListKeys
func (c *Client) ListKeys() ([]Key, error) {
	var resp []Key
	req := request{
		method: client.GET,
		url:    apiKeys,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of keys")
	}
	return resp, nil
}

// GetKey returns the key identified by keyName.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetKey
func (c *Client) GetKey(keyName string) (*Key, error) {
	var resp Key
	req := request{
		method: client.GET,
		url:    makeURL(apiKeys, keyName),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get key with name: %s", keyName)
	}
	return &resp, nil
}

// CreateKey creates a new key with the specified options.
// See API docs: http://apidocs.joyent.com/cloudapi/#CreateKey
func (c *Client) CreateKey(opts CreateKeyOpts) (*Key, error) {
	var resp Key
	req := request{
		method:         client.POST,
		url:            apiKeys,
		reqValue:       opts,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to create key with name: %s", opts.Name)
	}
	return &resp, nil
}

// DeleteKey deletes the key identified by keyName.
// See API docs: http://apidocs.joyent.com/cloudapi/#DeleteKey
func (c *Client) DeleteKey(keyName string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiKeys, keyName),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete key with name: %s", keyName)
	}
	return nil
}
