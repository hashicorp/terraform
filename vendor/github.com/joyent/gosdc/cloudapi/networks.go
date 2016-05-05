package cloudapi

import (
	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// Network represents a network available to a given account
type Network struct {
	Id          string // Unique identifier for the network
	Name        string // Network name
	Public      bool   // Whether this a public or private (rfc1918) network
	Description string // Optional description for this network, when name is not enough
}

// ListNetworks lists all the networks which can be used by the given account.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListNetworks
func (c *Client) ListNetworks() ([]Network, error) {
	var resp []Network
	req := request{
		method: client.GET,
		url:    apiNetworks,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of networks")
	}
	return resp, nil
}

// GetNetwork retrieves an individual network record.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetNetwork
func (c *Client) GetNetwork(networkID string) (*Network, error) {
	var resp Network
	req := request{
		method: client.GET,
		url:    makeURL(apiNetworks, networkID),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get network with id %s", networkID)
	}
	return &resp, nil
}
