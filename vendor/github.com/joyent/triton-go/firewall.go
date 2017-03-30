package triton

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/errwrap"
)

type FirewallClient struct {
	*Client
}

// Firewall returns a client used for accessing functions pertaining to
// firewall functionality in the Triton API.
func (c *Client) Firewall() *FirewallClient {
	return &FirewallClient{c}
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	// ID is a unique identifier for this rule
	ID string `json:"id"`

	// Enabled indicates if the rule is enabled
	Enabled bool `json:"enabled"`

	// Rule is the firewall rule text
	Rule string `json:"rule"`

	// Global indicates if the rule is global. Optional.
	Global bool `json:"global"`

	// Description is a human-readable description for the rule. Optional
	Description string `json:"description"`
}

type ListFirewallRulesInput struct{}

func (client *FirewallClient) ListFirewallRules(*ListFirewallRulesInput) ([]*FirewallRule, error) {
	respReader, err := client.executeRequest(http.MethodGet, "/my/fwrules", nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListFirewallRules request: {{err}}", err)
	}

	var result []*FirewallRule
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListFirewallRules response: {{err}}", err)
	}

	return result, nil
}

type GetFirewallRuleInput struct {
	ID string
}

func (client *FirewallClient) GetFirewallRule(input *GetFirewallRuleInput) (*FirewallRule, error) {
	path := fmt.Sprintf("/%s/fwrules/%s", client.accountName, input.ID)
	respReader, err := client.executeRequest(http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetFirewallRule request: {{err}}", err)
	}

	var result *FirewallRule
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding GetFirewallRule response: {{err}}", err)
	}

	return result, nil
}

type CreateFirewallRuleInput struct {
	Enabled     bool   `json:"enabled"`
	Rule        string `json:"rule"`
	Description string `json:"description"`
}

func (client *FirewallClient) CreateFirewallRule(input *CreateFirewallRuleInput) (*FirewallRule, error) {
	respReader, err := client.executeRequest(http.MethodPost, fmt.Sprintf("/%s/fwrules", client.accountName), input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing CreateFirewallRule request: {{err}}", err)
	}

	var result *FirewallRule
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding CreateFirewallRule response: {{err}}", err)
	}

	return result, nil
}

type UpdateFirewallRuleInput struct {
	ID          string `json:"-"`
	Enabled     bool   `json:"enabled"`
	Rule        string `json:"rule"`
	Description string `json:"description"`
}

func (client *FirewallClient) UpdateFirewallRule(input *UpdateFirewallRuleInput) (*FirewallRule, error) {
	respReader, err := client.executeRequest(http.MethodPost, fmt.Sprintf("/%s/fwrules/%s", client.accountName, input.ID), input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing UpdateFirewallRule request: {{err}}", err)
	}

	var result *FirewallRule
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding UpdateFirewallRule response: {{err}}", err)
	}

	return result, nil
}

type EnableFirewallRuleInput struct {
	ID string `json:"-"`
}

func (client *FirewallClient) EnableFirewallRule(input *EnableFirewallRuleInput) (*FirewallRule, error) {
	respReader, err := client.executeRequest(http.MethodPost, fmt.Sprintf("/%s/fwrules/%s/enable", client.accountName, input.ID), input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing EnableFirewallRule request: {{err}}", err)
	}

	var result *FirewallRule
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding EnableFirewallRule response: {{err}}", err)
	}

	return result, nil
}

type DisableFirewallRuleInput struct {
	ID string `json:"-"`
}

func (client *FirewallClient) DisableFirewallRule(input *DisableFirewallRuleInput) (*FirewallRule, error) {
	respReader, err := client.executeRequest(http.MethodPost, fmt.Sprintf("/%s/fwrules/%s/disable", client.accountName, input.ID), input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing DisableFirewallRule request: {{err}}", err)
	}

	var result *FirewallRule
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding DisableFirewallRule response: {{err}}", err)
	}

	return result, nil
}

type DeleteFirewallRuleInput struct {
	ID string
}

func (client *FirewallClient) DeleteFirewallRule(input *DeleteFirewallRuleInput) error {
	path := fmt.Sprintf("/%s/fwrules/%s", client.accountName, input.ID)
	respReader, err := client.executeRequest(http.MethodDelete, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DeleteFirewallRule request: {{err}}", err)
	}

	return nil
}

type ListMachineFirewallRulesInput struct {
	MachineID string
}

func (client *FirewallClient) ListMachineFirewallRules(input *ListMachineFirewallRulesInput) ([]*FirewallRule, error) {
	respReader, err := client.executeRequest(http.MethodGet, fmt.Sprintf("/my/machines/%s/firewallrules", input.MachineID), nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListMachineFirewallRules request: {{err}}", err)
	}

	var result []*FirewallRule
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListFirewallRules response: {{err}}", err)
	}

	return result, nil
}
