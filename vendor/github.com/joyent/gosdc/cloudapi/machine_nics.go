package cloudapi

import (
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// NICState represents the state of a NIC
type NICState string

var (
	NICStateProvisioning NICState = "provisioning"
	NICStateRunning      NICState = "running"
	NICStateStopped      NICState = "stopped"
)

// NIC represents a NIC on a machine
type NIC struct {
	IP      string   `json:"ip"`      // NIC's IPv4 Address
	MAC     string   `json:"mac"`     // NIC's MAC address
	Primary bool     `json:"primary"` // Whether this is the machine's primary NIC
	Netmask string   `json:"netmask"` // IPv4 netmask
	Gateway string   `json:"gateway"` // IPv4 gateway
	State   NICState `json:"state"`   // Describes the state of the NIC (e.g. provisioning, running, or stopped)
	Network string   `json:"network"` // Network ID this NIC is attached to
}

type addNICOptions struct {
	Network string `json:"network"` // UUID of network this NIC should attach to
}

// ListNICs lists all the NICs on a machine belonging to a given account
// See API docs: https://apidocs.joyent.com/cloudapi/#ListNics
func (c *Client) ListNICs(machineID string) ([]NIC, error) {
	var resp []NIC
	req := request{
		method: client.GET,
		url:    makeURL(apiMachines, machineID, apiNICs),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to list NICs")
	}
	return resp, nil
}

// GetNIC gets a specific NIC on a machine belonging to a given account
// See API docs: https://apidocs.joyent.com/cloudapi/#GetNic
func (c *Client) GetNIC(machineID, MAC string) (*NIC, error) {
	resp := new(NIC)
	req := request{
		method: client.GET,
		url:    makeURL(apiMachines, machineID, apiNICs, MAC),
		resp:   resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get NIC with MAC: %s", MAC)
	}
	return resp, nil
}

// AddNIC creates a new NIC on a machine belonging to a given account.
// *WARNING*: this causes the machine to reboot while adding the NIC.
// See API docs: https://apidocs.joyent.com/cloudapi/#AddNic
func (c *Client) AddNIC(machineID, networkID string) (*NIC, error) {
	resp := new(NIC)
	req := request{
		method:         client.POST,
		url:            makeURL(apiMachines, machineID, apiNICs),
		reqValue:       addNICOptions{networkID},
		resp:           resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to add NIC to machine %s on network: %s", machineID, networkID)
	}
	return resp, nil
}

// RemoveNIC removes a NIC on a machine belonging to a given account.
// *WARNING*: this causes the machine to reboot while removing the NIC.
// See API docs: https://apidocs.joyent.com/cloudapi/#RemoveNic
func (c *Client) RemoveNIC(machineID, MAC string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiMachines, machineID, apiNICs, MAC),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to remove NIC: %s", MAC)
	}
	return nil
}
