package compute

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// NATRule represents a Network Address Translation (NAT) rule.
// NAT rules are used to forward IPv4 traffic from a public IP address to a server's private IP address.
type NATRule struct {
	ID                string `json:"id"`
	NetworkDomainID   string `json:"networkDomainId"`
	InternalIPAddress string `json:"internalIp"`
	ExternalIPAddress string `json:"externalIp"`
	CreateTime        string `json:"createTime"`
	State             string `json:"state"`
	DataCenterID      string `json:"datacenterId"`
}

// NATRules represents a page of NATRule results.
type NATRules struct {
	Rules []NATRule `json:"natRule"`

	PagedResult
}

// Request body for adding a NAT rule.
type createNATRule struct {
	NetworkDomainID   string  `json:"networkDomainId"`
	InternalIPAddress string  `json:"internalIp"`
	ExternalIPAddress *string `json:"externalIp,omitempty"`
}

// Request body for deleting a NAT rule.
type deleteNATRule struct {
	RuleID string `json:"id"`
}

// GetNATRule retrieves the NAT rule with the specified Id.
// Returns nil if no NAT rule is found with the specified Id.
func (client *Client) GetNATRule(id string) (rule *NATRule, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return nil, err
	}

	requestURI := fmt.Sprintf("%s/network/natRule/%s", organizationID, id)
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

		return nil, apiResponse.ToError("Request to retrieve NAT rule failed with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	rule = &NATRule{}
	err = json.Unmarshal(responseBody, rule)
	if err != nil {
		return nil, err
	}

	return rule, nil
}

// ListNATRules retrieves all NAT rules defined for the specified network domain.
func (client *Client) ListNATRules(networkDomainID string) (rules *NATRules, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return nil, err
	}

	requestURI := fmt.Sprintf("%s/network/natRule?networkDomainId=%s", organizationID, networkDomainID)
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

		return nil, apiResponse.ToError("Request to list NAT rules for network domain '%s' failed with status code %d (%s): %s", networkDomainID, statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	rules = &NATRules{}
	err = json.Unmarshal(responseBody, rules)

	return rules, err
}

// AddNATRule creates a new NAT rule to forward traffic from the specified external IPv4 address to the specified internal IPv4 address.
// If externalIPAddress is not specified, an unallocated IPv4 address will be used (if available).
//
// This operation is synchronous.
func (client *Client) AddNATRule(networkDomainID string, internalIPAddress string, externalIPAddress *string) (natRuleID string, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return "", err
	}

	requestURI := fmt.Sprintf("%s/network/createNatRule", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &createNATRule{
		NetworkDomainID:   networkDomainID,
		InternalIPAddress: internalIPAddress,
		ExternalIPAddress: externalIPAddress,
	})
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return "", err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return "", err
	}

	if apiResponse.ResponseCode != ResponseCodeOK {
		return "", apiResponse.ToError("Request to create NAT rule in network domain '%s' failed with unexpected status code %d (%s): %s", networkDomainID, statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	// Expected: "info" { "name": "natRuleId", "value": "the-Id-of-the-new-NAT-rule" }
	if len(apiResponse.FieldMessages) != 1 || apiResponse.FieldMessages[0].FieldName != "natRuleId" {
		return "", apiResponse.ToError("Received an unexpected response (missing 'natRuleId') with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return apiResponse.FieldMessages[0].Message, nil
}

// DeleteNATRule deletes the specified NAT rule.
// This operation is synchronous.
func (client *Client) DeleteNATRule(id string) error {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return err
	}

	requestURI := fmt.Sprintf("%s/network/deleteNatRule", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost,
		&deleteNATRule{id},
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
		return apiResponse.ToError("Request to delete NAT rule '%s' failed with unexpected status code %d (%s): %s", id, statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return nil
}
