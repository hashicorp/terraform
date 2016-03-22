package cloudapi

import (
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// FirewallRule represent a firewall rule that can be specifed for a machine.
type FirewallRule struct {
	Id      string // Unique identifier for the rule
	Enabled bool   // Whether the rule is enabled or not
	Rule    string // Firewall rule in the form 'FROM <target a> TO <target b> <action> <protocol> <port>'
}

// CreateFwRuleOpts represent the option that can be specified
// when creating a new firewall rule.
type CreateFwRuleOpts struct {
	Enabled bool   `json:"enabled"` // Whether to enable the rule or not
	Rule    string `json:"rule"`    // Firewall rule in the form 'FROM <target a> TO <target b> <action> <protocol> <port>'
}

// ListFirewallRules lists all the firewall rules on record for a specified account.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListFirewallRules
func (c *Client) ListFirewallRules() ([]FirewallRule, error) {
	var resp []FirewallRule
	req := request{
		method: client.GET,
		url:    apiFirewallRules,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of firewall rules")
	}
	return resp, nil
}

// GetFirewallRule returns the specified firewall rule.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetFirewallRule
func (c *Client) GetFirewallRule(fwRuleID string) (*FirewallRule, error) {
	var resp FirewallRule
	req := request{
		method: client.GET,
		url:    makeURL(apiFirewallRules, fwRuleID),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get firewall rule with id %s", fwRuleID)
	}
	return &resp, nil
}

// CreateFirewallRule creates the firewall rule with the specified options.
// See API docs: http://apidocs.joyent.com/cloudapi/#CreateFirewallRule
func (c *Client) CreateFirewallRule(opts CreateFwRuleOpts) (*FirewallRule, error) {
	var resp FirewallRule
	req := request{
		method:         client.POST,
		url:            apiFirewallRules,
		reqValue:       opts,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to create firewall rule: %s", opts.Rule)
	}
	return &resp, nil
}

// UpdateFirewallRule updates the specified firewall rule.
// See API docs: http://apidocs.joyent.com/cloudapi/#UpdateFirewallRule
func (c *Client) UpdateFirewallRule(fwRuleID string, opts CreateFwRuleOpts) (*FirewallRule, error) {
	var resp FirewallRule
	req := request{
		method:   client.POST,
		url:      makeURL(apiFirewallRules, fwRuleID),
		reqValue: opts,
		resp:     &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to update firewall rule with id %s to %s", fwRuleID, opts.Rule)
	}
	return &resp, nil
}

// EnableFirewallRule enables the given firewall rule record if it is disabled.
// See API docs: http://apidocs.joyent.com/cloudapi/#EnableFirewallRule
func (c *Client) EnableFirewallRule(fwRuleID string) (*FirewallRule, error) {
	var resp FirewallRule
	req := request{
		method: client.POST,
		url:    makeURL(apiFirewallRules, fwRuleID, apiFirewallRulesEnable),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to enable firewall rule with id %s", fwRuleID)
	}
	return &resp, nil
}

// DisableFirewallRule disables the given firewall rule record if it is enabled.
// See API docs: http://apidocs.joyent.com/cloudapi/#DisableFirewallRule
func (c *Client) DisableFirewallRule(fwRuleID string) (*FirewallRule, error) {
	var resp FirewallRule
	req := request{
		method: client.POST,
		url:    makeURL(apiFirewallRules, fwRuleID, apiFirewallRulesDisable),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to disable firewall rule with id %s", fwRuleID)
	}
	return &resp, nil
}

// DeleteFirewallRule removes the given firewall rule record from all the required account machines.
// See API docs: http://apidocs.joyent.com/cloudapi/#DeleteFirewallRule
func (c *Client) DeleteFirewallRule(fwRuleID string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiFirewallRules, fwRuleID),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete firewall rule with id %s", fwRuleID)
	}
	return nil
}

// ListFirewallRuleMachines return the list of machines affected by the given firewall rule.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListFirewallRuleMachines
func (c *Client) ListFirewallRuleMachines(fwRuleID string) ([]Machine, error) {
	var resp []Machine
	req := request{
		method: client.GET,
		url:    makeURL(apiFirewallRules, fwRuleID, apiMachines),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of machines affected by firewall rule wit id %s", fwRuleID)
	}
	return resp, nil
}
