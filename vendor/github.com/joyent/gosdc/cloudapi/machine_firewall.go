package cloudapi

import (
	"fmt"
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// ListMachineFirewallRules lists all the firewall rules for the specified machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListMachineFirewallRules
func (c *Client) ListMachineFirewallRules(machineID string) ([]FirewallRule, error) {
	var resp []FirewallRule
	req := request{
		method: client.GET,
		url:    makeURL(apiMachines, machineID, apiFirewallRules),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of firewall rules for machine with id %s", machineID)
	}
	return resp, nil
}

// EnableFirewallMachine enables the firewall for the specified machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#EnableMachineFirewall
func (c *Client) EnableFirewallMachine(machineID string) error {
	req := request{
		method:         client.POST,
		url:            fmt.Sprintf("%s/%s?action=%s", apiMachines, machineID, actionEnableFw),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to enable firewall on machine with id: %s", machineID)
	}
	return nil
}

// DisableFirewallMachine disables the firewall for the specified machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#DisableMachineFirewall
func (c *Client) DisableFirewallMachine(machineID string) error {
	req := request{
		method:         client.POST,
		url:            fmt.Sprintf("%s/%s?action=%s", apiMachines, machineID, actionDisableFw),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to disable firewall on machine with id: %s", machineID)
	}
	return nil
}
