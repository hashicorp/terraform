package cloudapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"strings"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// Machine represent a provisioned virtual machines
type Machine struct {
	Id              string            // Unique identifier for the image
	Name            string            // Machine friendly name
	Type            string            // Machine type, one of 'smartmachine' or 'virtualmachine'
	State           string            // Current state of the machine
	Dataset         string            // The dataset URN the machine was provisioned with. For new images/datasets this value will be the dataset id, i.e, same value than the image attribute
	Memory          int               // The amount of memory the machine has (in Mb)
	Disk            int               // The amount of disk the machine has (in Gb)
	IPs             []string          // The IP addresses the machine has
	Metadata        map[string]string // Map of the machine metadata, e.g. authorized-keys
	Tags            map[string]string // Map of the machine tags
	Created         string            // When the machine was created
	Updated         string            // When the machine was updated
	Package         string            // The name of the package used to create the machine
	Image           string            // The image id the machine was provisioned with
	PrimaryIP       string            // The primary (public) IP address for the machine
	Networks        []string          // The network IDs for the machine
	FirewallEnabled bool              `json:"firewall_enabled"` // whether or not the firewall is enabled
}

// Equals compares two machines. Ignores state and timestamps.
func (m Machine) Equals(other Machine) bool {
	if m.Id == other.Id && m.Name == other.Name && m.Type == other.Type && m.Dataset == other.Dataset &&
		m.Memory == other.Memory && m.Disk == other.Disk && m.Package == other.Package && m.Image == other.Image &&
		m.compareIPs(other) && m.compareMetadata(other) {
		return true
	}
	return false
}

// Helper method to compare two machines IPs
func (m Machine) compareIPs(other Machine) bool {
	if len(m.IPs) != len(other.IPs) {
		return false
	}
	for i, v := range m.IPs {
		if v != other.IPs[i] {
			return false
		}
	}
	return true
}

// Helper method to compare two machines metadata
func (m Machine) compareMetadata(other Machine) bool {
	if len(m.Metadata) != len(other.Metadata) {
		return false
	}
	for k, v := range m.Metadata {
		if v != other.Metadata[k] {
			return false
		}
	}
	return true
}

// CreateMachineOpts represent the option that can be specified
// when creating a new machine.
type CreateMachineOpts struct {
	Name            string            `json:"name"`             // Machine friendly name, default is a randomly generated name
	Package         string            `json:"package"`          // Name of the package to use on provisioning
	Image           string            `json:"image"`            // The image UUID
	Networks        []string          `json:"networks"`         // Desired networks IDs
	Metadata        map[string]string `json:"-"`                // An arbitrary set of metadata key/value pairs can be set at provision time
	Tags            map[string]string `json:"-"`                // An arbitrary set of tags can be set at provision time
	FirewallEnabled bool              `json:"firewall_enabled"` // Completely enable or disable firewall for this machine (new in API version 7.0)
}

// AuditAction represents an action/event accomplished by a machine.
type AuditAction struct {
	Action     string                 // Action name
	Parameters map[string]interface{} // Original set of parameters sent when the action was requested
	Time       string                 // When the action finished
	Success    string                 // Either 'yes' or 'no', depending on the action successfulness
	Caller     Caller                 // Account requesting the action
}

// Caller represents an account requesting an action.
type Caller struct {
	Type  string // Authentication type for the action request. One of 'basic', 'operator', 'signature' or 'token'
	User  string // When the authentication type is 'basic', this member will be present and include user login
	IP    string // The IP addresses this from which the action was requested. Not present if type is 'operator'
	KeyId string // When authentication type is either 'signature' or 'token', SSH key identifier
}

// appendJSON marshals the given attribute value and appends it as an encoded value to the given json data.
// The newly encode (attr, value) is inserted just before the closing "}" in the json data.
func appendJSON(data []byte, attr string, value interface{}) ([]byte, error) {
	newData, err := json.Marshal(&value)
	if err != nil {
		return nil, err
	}
	strData := string(data)
	result := fmt.Sprintf(`%s, "%s":%s}`, strData[:len(strData)-1], attr, string(newData))
	return []byte(result), nil
}

type jsonOpts CreateMachineOpts

// MarshalJSON turns the given CreateMachineOpts into JSON
func (opts CreateMachineOpts) MarshalJSON() ([]byte, error) {
	jo := jsonOpts(opts)
	data, err := json.Marshal(&jo)
	if err != nil {
		return nil, err
	}
	for k, v := range opts.Tags {
		if !strings.HasPrefix(k, "tag.") {
			k = "tag." + k
		}
		data, err = appendJSON(data, k, v)
		if err != nil {
			return nil, err
		}
	}
	for k, v := range opts.Metadata {
		if !strings.HasPrefix(k, "metadata.") {
			k = "metadata." + k
		}
		data, err = appendJSON(data, k, v)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// ListMachines lists all machines on record for an account.
// You can paginate this API by passing in offset, and limit
// See API docs: http://apidocs.joyent.com/cloudapi/#ListMachines
func (c *Client) ListMachines(filter *Filter) ([]Machine, error) {
	var resp []Machine
	req := request{
		method: client.GET,
		url:    apiMachines,
		filter: filter,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of machines")
	}
	return resp, nil
}

// CountMachines returns the number of machines on record for an account.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListMachines
func (c *Client) CountMachines() (int, error) {
	var resp int
	req := request{
		method: client.HEAD,
		url:    apiMachines,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return -1, errors.Newf(err, "failed to get count of machines")
	}
	return resp, nil
}

// GetMachine returns the machine specified by machineId.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetMachine
func (c *Client) GetMachine(machineID string) (*Machine, error) {
	var resp Machine
	req := request{
		method: client.GET,
		url:    makeURL(apiMachines, machineID),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get machine with id: %s", machineID)
	}
	return &resp, nil
}

// CreateMachine creates a new machine with the options specified.
// See API docs: http://apidocs.joyent.com/cloudapi/#CreateMachine
func (c *Client) CreateMachine(opts CreateMachineOpts) (*Machine, error) {
	var resp Machine
	req := request{
		method:         client.POST,
		url:            apiMachines,
		reqValue:       opts,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to create machine with name: %s", opts.Name)
	}
	return &resp, nil
}

// StopMachine stops a running machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#StopMachine
func (c *Client) StopMachine(machineID string) error {
	req := request{
		method:         client.POST,
		url:            fmt.Sprintf("%s/%s?action=%s", apiMachines, machineID, actionStop),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to stop machine with id: %s", machineID)
	}
	return nil
}

// StartMachine starts a stopped machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#StartMachine
func (c *Client) StartMachine(machineID string) error {
	req := request{
		method:         client.POST,
		url:            fmt.Sprintf("%s/%s?action=%s", apiMachines, machineID, actionStart),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to start machine with id: %s", machineID)
	}
	return nil
}

// RebootMachine reboots (stop followed by a start) a machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#RebootMachine
func (c *Client) RebootMachine(machineID string) error {
	req := request{
		method:         client.POST,
		url:            fmt.Sprintf("%s/%s?action=%s", apiMachines, machineID, actionReboot),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to reboot machine with id: %s", machineID)
	}
	return nil
}

// ResizeMachine allows you to resize a SmartMachine. Virtual machines can also
// be resized, but only resizing virtual machines to a higher capacity package
// is supported.
// See API docs: http://apidocs.joyent.com/cloudapi/#ResizeMachine
func (c *Client) ResizeMachine(machineID, packageName string) error {
	req := request{
		method:         client.POST,
		url:            fmt.Sprintf("%s/%s?action=%s&package=%s", apiMachines, machineID, actionResize, packageName),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to resize machine with id: %s", machineID)
	}
	return nil
}

// RenameMachine renames an existing machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#RenameMachine
func (c *Client) RenameMachine(machineID, machineName string) error {
	req := request{
		method:         client.POST,
		url:            fmt.Sprintf("%s/%s?action=%s&name=%s", apiMachines, machineID, actionRename, machineName),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to rename machine with id: %s", machineID)
	}
	return nil
}

// DeleteMachine allows you to completely destroy a machine. Machine must be in the 'stopped' state.
// See API docs: http://apidocs.joyent.com/cloudapi/#DeleteMachine
func (c *Client) DeleteMachine(machineID string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiMachines, machineID),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete machine with id %s", machineID)
	}
	return nil
}

// MachineAudit provides a list of machine's accomplished actions, (sorted from
// latest to older one).
// See API docs: http://apidocs.joyent.com/cloudapi/#MachineAudit
func (c *Client) MachineAudit(machineID string) ([]AuditAction, error) {
	var resp []AuditAction
	req := request{
		method: client.GET,
		url:    makeURL(apiMachines, machineID, apiAudit),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get actions for machine with id %s", machineID)
	}
	return resp, nil
}
