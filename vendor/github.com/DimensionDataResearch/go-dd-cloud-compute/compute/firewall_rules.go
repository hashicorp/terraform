package compute

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// FirewallRule represents a firewall rule.
type FirewallRule struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Action          string            `json:"action"`
	IPVersion       string            `json:"ipVersion"`
	Protocol        string            `json:"protocol"`
	Source          FirewallRuleScope `json:"source"`
	Destination     FirewallRuleScope `json:"destination"`
	Enabled         bool              `json:"enabled"`
	State           string            `json:"state"`
	NetworkDomainID string            `json:"networkDomainId"`
	DataCenterID    string            `json:"datacenterId"`
	RuleType        string            `json:"ruleType"`
}

// GetID returns the firewall rule's Id.
func (rule *FirewallRule) GetID() string {
	return rule.ID
}

// GetName returns the firewall rule's name.
func (rule *FirewallRule) GetName() string {
	return rule.Name
}

// GetState returns the firewall rule's current state.
func (rule *FirewallRule) GetState() string {
	return rule.State
}

// IsDeleted determines whether the firewall rule has been deleted (is nil).
func (rule *FirewallRule) IsDeleted() bool {
	return rule == nil
}

var _ Resource = &FirewallRule{}

// FirewallRuleScope represents a scope (IP and / or port) for firewall configuration (source or destination).
type FirewallRuleScope struct {
	IPAddress   *FirewallRuleIPAddress `json:"ip,omitempty"`
	AddressList *EntitySummary         `json:"ipAddressList,omitempty"`
	Port        *FirewallRulePort      `json:"port,omitempty"`
}

// FirewallRuleIPAddress represents represents an IP address for firewall configuration.
type FirewallRuleIPAddress struct {
	Address    string `json:"address"`
	PrefixSize *int   `json:"PrefixSize,omitempty"`
}

// FirewallRulePort represents a firewall port configuration.
type FirewallRulePort struct {
	Begin int  `json:"begin"`
	End   *int `json:"end"`
}

// FirewallRules represents a page of FirewallRule results.
type FirewallRules struct {
	Rules []FirewallRule `json:"firewallRule"`

	PagedResult
}

const firewallMatchAny = "ANY"

// FirewallRuleConfiguration represents the configuration for a new firewall rule.
type FirewallRuleConfiguration struct {
	Name            string                `json:"name"`
	Action          string                `json:"action"`
	Enabled         bool                  `json:"enabled"`
	Placement       FirewallRulePlacement `json:"placement"`
	IPVersion       string                `json:"ipVersion"`
	Protocol        string                `json:"protocol"`
	Source          FirewallRuleScope     `json:"source"`
	Destination     FirewallRuleScope     `json:"destination"`
	NetworkDomainID string                `json:"networkDomainId"`
}

// PlaceFirst modifies the configuration so that the firewall rule will be placed in the first available position.
func (configuration *FirewallRuleConfiguration) PlaceFirst() {
	configuration.Placement = FirewallRulePlacement{
		Position: "FIRST",
	}
}

// PlaceBefore modifies the configuration so that the firewall rule will be placed before the specified rule.
func (configuration *FirewallRuleConfiguration) PlaceBefore(beforeRuleName string) {
	configuration.Placement = FirewallRulePlacement{
		Position:           "BEFORE",
		RelativeToRuleName: &beforeRuleName,
	}
}

// PlaceAfter modifies the configuration so that the firewall rule will be placed after the specified rule.
func (configuration *FirewallRuleConfiguration) PlaceAfter(afterRuleName string) {
	configuration.Placement = FirewallRulePlacement{
		Position:           "AFTER",
		RelativeToRuleName: &afterRuleName,
	}
}

// MatchAnySource modifies the configuration so that the firewall rule will match any combination of source IP address and port.
func (configuration *FirewallRuleConfiguration) MatchAnySource() {
	configuration.MatchAnySourceAddress(nil)
}

// MatchAnySourceAddress modifies the configuration so that the firewall rule will match any source IP address (and, optionally,port).
func (configuration *FirewallRuleConfiguration) MatchAnySourceAddress(port *int) {
	var sourcePort *FirewallRulePort
	if port != nil {
		sourcePort = &FirewallRulePort{
			Begin: *port,
		}
	}

	configuration.Source = FirewallRuleScope{
		IPAddress: &FirewallRuleIPAddress{
			Address: firewallMatchAny,
		},
		Port: sourcePort,
	}
}

// MatchSourceAddressAndPort modifies the configuration so that the firewall rule will match a specific source IP address (and, optionally, port).
func (configuration *FirewallRuleConfiguration) MatchSourceAddressAndPort(address string, port *int) {
	sourceScope := &FirewallRuleScope{
		IPAddress: &FirewallRuleIPAddress{
			Address: strings.ToUpper(address),
		},
	}
	if port != nil {
		sourceScope.Port = &FirewallRulePort{
			Begin: *port,
		}
	}
	configuration.Source = *sourceScope
}

// MatchSourceNetworkAndPort modifies the configuration so that the firewall rule will match any source IP address on the specified network (and, optionally, port).
func (configuration *FirewallRuleConfiguration) MatchSourceNetworkAndPort(baseAddress string, prefixSize int, port *int) {
	sourceScope := &FirewallRuleScope{
		IPAddress: &FirewallRuleIPAddress{
			Address:    baseAddress,
			PrefixSize: &prefixSize,
		},
	}
	if port != nil {
		sourceScope.Port = &FirewallRulePort{
			Begin: *port,
		}
	}
	configuration.Source = *sourceScope
}

// MatchDestinationAddressAndPort modifies the configuration so that the firewall rule will match a specific destination IP address (and, optionally, port).
func (configuration *FirewallRuleConfiguration) MatchDestinationAddressAndPort(address string, port *int) {
	destinationScope := &FirewallRuleScope{
		IPAddress: &FirewallRuleIPAddress{
			Address: strings.ToUpper(address),
		},
	}
	if port != nil {
		destinationScope.Port = &FirewallRulePort{
			Begin: *port,
		}
	}
	configuration.Destination = *destinationScope
}

// MatchDestinationNetworkAndPort modifies the configuration so that the firewall rule will match any destination IP address on the specified network (and, optionally, port).
func (configuration *FirewallRuleConfiguration) MatchDestinationNetworkAndPort(baseAddress string, prefixSize int, port *int) {
	destinationScope := &FirewallRuleScope{
		IPAddress: &FirewallRuleIPAddress{
			Address:    baseAddress,
			PrefixSize: &prefixSize,
		},
	}
	if port != nil {
		destinationScope.Port = &FirewallRulePort{
			Begin: *port,
		}
	}
	configuration.Destination = *destinationScope
}

// MatchSourceAddressListAndPort modifies the configuration so that the firewall rule will match a specific source IP address list (and, optionally, port).
func (configuration *FirewallRuleConfiguration) MatchSourceAddressListAndPort(addressListID string, port *int) {
	sourceScope := &FirewallRuleScope{
		AddressList: &EntitySummary{
			ID: addressListID,
		},
	}
	if port != nil {
		sourceScope.Port = &FirewallRulePort{
			Begin: *port,
		}
	}
	configuration.Source = *sourceScope
}

// MatchAnyDestination modifies the configuration so that the firewall rule will match any combination of destination IP address and port.
func (configuration *FirewallRuleConfiguration) MatchAnyDestination() {
	configuration.MatchAnyDestinationAddress(nil)
}

// MatchAnyDestinationAddress modifies the configuration so that the firewall rule will match any destination IP address (and, optionally, port).
func (configuration *FirewallRuleConfiguration) MatchAnyDestinationAddress(port *int) {
	var destinationPort *FirewallRulePort
	if port != nil {
		destinationPort = &FirewallRulePort{
			Begin: *port,
		}
	}

	configuration.Destination = FirewallRuleScope{
		IPAddress: &FirewallRuleIPAddress{
			Address: firewallMatchAny,
		},
		Port: destinationPort,
	}
}

// MatchDestinationAddressListAndPort modifies the configuration so that the firewall rule will match a specific destination IP address list (and, optionally, port).
func (configuration *FirewallRuleConfiguration) MatchDestinationAddressListAndPort(addressListID string, port *int) {
	destinationScope := &FirewallRuleScope{
		AddressList: &EntitySummary{
			ID: addressListID,
		},
	}
	if port != nil {
		destinationScope.Port = &FirewallRulePort{
			Begin: *port,
		}
	}
	configuration.Destination = *destinationScope
}

// FirewallRulePlacement describes the placement for a firewall rule.
type FirewallRulePlacement struct {
	Position           string  `json:"position"`
	RelativeToRuleName *string `json:"relativeToRule,omitempty"`
}

type editFirewallRule struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type deleteFirewallRule struct {
	ID string `json:"id"`
}

// GetFirewallRule retrieves the Firewall rule with the specified Id.
// Returns nil if no Firewall rule is found with the specified Id.
func (client *Client) GetFirewallRule(id string) (rule *FirewallRule, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return nil, err
	}

	requestURI := fmt.Sprintf("%s/network/firewallRule/%s", organizationID, id)
	request, err := client.newRequestV22(requestURI, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		var apiResponse *APIResponseV2

		apiResponse, err = readAPIResponseAsJSON(responseBody, statusCode)
		if err != nil {
			return nil, err
		}

		if apiResponse.ResponseCode == ResponseCodeResourceNotFound {
			return nil, nil // Not an error, but was not found.
		}

		return nil, apiResponse.ToError("Request to retrieve firewall rule failed with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	rule = &FirewallRule{}
	err = json.Unmarshal(responseBody, rule)
	if err != nil {
		return nil, err
	}

	return rule, nil
}

// ListFirewallRules lists all firewall rules that apply to the specified network domain.
func (client *Client) ListFirewallRules(networkDomainID string) (rules *FirewallRules, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return nil, err
	}

	requestURI := fmt.Sprintf("%s/network/firewallRule?networkDomainId=%s", organizationID, networkDomainID)
	request, err := client.newRequestV22(requestURI, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		var apiResponse *APIResponseV2

		apiResponse, err = readAPIResponseAsJSON(responseBody, statusCode)
		if err != nil {
			return nil, err
		}

		return nil, apiResponse.ToError("Request to list firewall rules for network domain '%s' failed with status code %d (%s): %s", networkDomainID, statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	rules = &FirewallRules{}
	err = json.Unmarshal(responseBody, rules)

	return rules, err
}

// CreateFirewallRule creates a new firewall rule.
func (client *Client) CreateFirewallRule(configuration FirewallRuleConfiguration) (firewallRuleID string, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return "", err
	}

	requestURI := fmt.Sprintf("%s/network/createFirewallRule", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &configuration)
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return "", err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return "", err
	}

	if apiResponse.ResponseCode != ResponseCodeOK {
		return "", apiResponse.ToError("Request to create firewall rule in network domain '%s' failed with unexpected status code %d (%s): %s", configuration.NetworkDomainID, statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	// Expected: "info" { "name": "firewallRuleId", "value": "the-Id-of-the-new-firewall-rule" }
	if len(apiResponse.FieldMessages) != 1 || apiResponse.FieldMessages[0].FieldName != "firewallRuleId" {
		return "", apiResponse.ToError("Received an unexpected response (missing 'firewallRuleId') with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return apiResponse.FieldMessages[0].Message, nil
}

// EditFirewallRule updates the configuration for a firewall rule (enable / disable).
// This operation is synchronous.
func (client *Client) EditFirewallRule(id string, enabled bool) error {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return err
	}

	requestURI := fmt.Sprintf("%s/network/editFirewallRule", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &editFirewallRule{
		ID:      id,
		Enabled: enabled,
	})
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return err
	}

	if apiResponse.ResponseCode != ResponseCodeOK {
		return apiResponse.ToError("Request to edit firewall rule failed with unexpected status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return nil
}

// DeleteFirewallRule deletes the specified FirewallRule rule.
func (client *Client) DeleteFirewallRule(id string) error {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return err
	}

	requestURI := fmt.Sprintf("%s/network/deleteFirewallRule", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost,
		&deleteFirewallRule{id},
	)
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return err
	}

	if apiResponse.ResponseCode != ResponseCodeOK {
		return apiResponse.ToError("Request to delete firewall rule '%s' failed with unexpected status code %d (%s): %s", id, statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return nil
}
