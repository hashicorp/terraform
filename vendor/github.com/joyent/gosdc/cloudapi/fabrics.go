package cloudapi

import (
	"net/http"
	"strconv"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

type FabricVLAN struct {
	Id          int16  `json:"vlan_id"`               // Number between 0-4095 indicating VLAN Id
	Name        string `json:"name"`                  // Unique name to identify VLAN
	Description string `json:"description,omitempty"` // Optional description of the VLAN
}

type FabricNetwork struct {
	Id               string            `json:"id"`                  // Unique identifier for network
	Name             string            `json:"name"`                // Network name
	Public           bool              `json:"public"`              // Whether or not this is an RFC1918 network
	Fabric           bool              `json:"fabric"`              // Whether this network is on a fabric
	Description      string            `json:"description"`         // Optional description of network
	Subnet           string            `json:"subnet"`              // CIDR formatted string describing network
	ProvisionStartIp string            `json:"provision_start_ip"`  // First IP on the network that can be assigned
	ProvisionEndIp   string            `json:"provision_end_ip"`    // Last assignable IP on the network
	Gateway          string            `json:"gateway"`             // Optional Gateway IP
	Resolvers        []string          `json:"resolvers,omitempty"` // Array of IP addresses for resolvers
	Routes           map[string]string `json:"routes,omitempty"`    // Map of CIDR block to Gateway IP Address
	InternetNAT      bool              `json:"internet_nat"`        // If a NAT zone is provisioned at Gateway IP Address
	VLANId           int16             `json:"vlan_id"`             // VLAN network is on
}

type CreateFabricNetworkOpts struct {
	Name             string            `json:"name"`                  // Network name
	Description      string            `json:"description,omitempty"` // Optional description of network
	Subnet           string            `json:"subnet"`                // CIDR formatted string describing network
	ProvisionStartIp string            `json:"provision_start_ip"`    // First IP on the network that can be assigned
	ProvisionEndIp   string            `json:"provision_end_ip"`      // Last assignable IP on the network
	Gateway          string            `json:"gateway,omitempty"`     // Optional Gateway IP
	Resolvers        []string          `json:"resolvers,omitempty"`   // Array of IP addresses for resolvers
	Routes           map[string]string `json:"routes,omitempty"`      // Map of CIDR block to Gateway IP Address
	InternetNAT      bool              `json:"internet_nat"`          // If a NAT zone is provisioned at Gateway IP Address
}

// ListFabricVLANs lists VLANs
// See API docs: https://apidocs.joyent.com/cloudapi/#ListFabricVLANs
func (c *Client) ListFabricVLANs() ([]FabricVLAN, error) {
	var resp []FabricVLAN
	req := request{
		method: client.GET,
		url:    apiFabricVLANs,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of fabric VLANs")
	}
	return resp, nil
}

// GetFabricLAN retrieves a single VLAN by ID
// See API docs: https://apidocs.joyent.com/cloudapi/#GetFabricVLAN
func (c *Client) GetFabricVLAN(vlanID int16) (*FabricVLAN, error) {
	var resp FabricVLAN
	req := request{
		method: client.GET,
		url:    makeURL(apiFabricVLANs, strconv.Itoa(int(vlanID))),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get fabric VLAN with id %d", vlanID)
	}
	return &resp, nil
}

// CreateFabricVLAN creates a new VLAN with the specified options
// See API docs: https://apidocs.joyent.com/cloudapi/#CreateFabricVLAN
func (c *Client) CreateFabricVLAN(vlan FabricVLAN) (*FabricVLAN, error) {
	var resp FabricVLAN
	req := request{
		method:         client.POST,
		url:            apiFabricVLANs,
		reqValue:       vlan,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to create fabric VLAN: %d - %s", vlan.Id, vlan.Name)
	}
	return &resp, nil
}

// UpdateFabricVLAN updates a given VLAN with new fields
// See API docs: https://apidocs.joyent.com/cloudapi/#UpdateFabricVLAN
func (c *Client) UpdateFabricVLAN(vlan FabricVLAN) (*FabricVLAN, error) {
	var resp FabricVLAN
	req := request{
		method:         client.PUT,
		url:            makeURL(apiFabricVLANs, strconv.Itoa(int(vlan.Id))),
		reqValue:       vlan,
		resp:           &resp,
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to update fabric VLAN with id %d to %s - %s", vlan.Id, vlan.Name, vlan.Description)
	}
	return &resp, nil
}

// DeleteFabricVLAN delets a given VLAN as specified by ID
// See API docs: https://apidocs.joyent.com/cloudapi/#DeleteFabricVLAN
func (c *Client) DeleteFabricVLAN(vlanID int16) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiFabricVLANs, strconv.Itoa(int(vlanID))),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete fabric VLAN with id %d", vlanID)
	}
	return nil
}

// ListFabricNetworks lists the networks inside the given VLAN
// See API docs: https://apidocs.joyent.com/cloudapi/#ListFabricNetworks
func (c *Client) ListFabricNetworks(vlanID int16) ([]FabricNetwork, error) {
	var resp []FabricNetwork
	req := request{
		method: client.GET,
		url:    makeURL(apiFabricVLANs, strconv.Itoa(int(vlanID)), apiFabricNetworks),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of networks on fabric %d", vlanID)
	}
	return resp, nil
}

// GetFabricNetwork gets a single network by VLAN and Network IDs
// See API docs: https://apidocs.joyent.com/cloudapi/#GetFabricNetwork
func (c *Client) GetFabricNetwork(vlanID int16, networkID string) (*FabricNetwork, error) {
	var resp FabricNetwork
	req := request{
		method: client.GET,
		url:    makeURL(apiFabricVLANs, strconv.Itoa(int(vlanID)), apiFabricNetworks, networkID),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get fabric network %s on vlan %d", networkID, vlanID)
	}
	return &resp, nil
}

// CreateFabricNetwork creates a new fabric network
// See API docs: https://apidocs.joyent.com/cloudapi/#CreateFabricNetwork
func (c *Client) CreateFabricNetwork(vlanID int16, opts CreateFabricNetworkOpts) (*FabricNetwork, error) {
	var resp FabricNetwork
	req := request{
		method:         client.POST,
		url:            makeURL(apiFabricVLANs, strconv.Itoa(int(vlanID)), apiFabricNetworks),
		reqValue:       opts,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to create fabric network %s on vlan %d", opts.Name, vlanID)
	}
	return &resp, nil
}

// DeleteFabricNetwork deletes an existing fabric network
// See API docs: https://apidocs.joyent.com/cloudapi/#DeleteFabricNetwork
func (c *Client) DeleteFabricNetwork(vlanID int16, networkID string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiFabricVLANs, strconv.Itoa(int(vlanID)), apiFabricNetworks, networkID),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete fabric network %s on vlan %d", networkID, vlanID)
	}
	return nil
}
