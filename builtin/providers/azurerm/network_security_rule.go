package azurerm

import (
	"fmt"
	"strings"
)

func validateNetworkSecurityRuleProtocol(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	protocols := map[string]bool{
		"tcp": true,
		"udp": true,
		"*":   true,
	}

	if !protocols[value] {
		errors = append(errors, fmt.Errorf("Network Security Rule Protocol can only be Tcp, Udp or *"))
	}
	return
}

func validateNetworkSecurityRuleAccess(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	accessTypes := map[string]bool{
		"allow": true,
		"deny":  true,
	}

	if !accessTypes[value] {
		errors = append(errors, fmt.Errorf("Network Security Rule Access can only be Allow or Deny"))
	}
	return
}

func validateNetworkSecurityRuleDirection(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	directions := map[string]bool{
		"inbound":  true,
		"outbound": true,
	}

	if !directions[value] {
		errors = append(errors, fmt.Errorf("Network Security Rule Directions can only be Inbound or Outbound"))
	}
	return
}
