package azurerm

import "testing"

func TestResourceAzureRMNetworkSecurityRuleProtocol_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "tcp",
			ErrCount: 0,
		},
		{
			Value:    "TCP",
			ErrCount: 0,
		},
		{
			Value:    "*",
			ErrCount: 0,
		},
		{
			Value:    "Udp",
			ErrCount: 0,
		},
		{
			Value:    "Tcp",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateNetworkSecurityRuleProtocol(tc.Value, "azurerm_network_security_rule")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Network Security Rule protocol to trigger a validation error")
		}
	}
}

func TestResourceAzureRMNetworkSecurityRuleAccess_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Allow",
			ErrCount: 0,
		},
		{
			Value:    "Deny",
			ErrCount: 0,
		},
		{
			Value:    "ALLOW",
			ErrCount: 0,
		},
		{
			Value:    "deny",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateNetworkSecurityRuleAccess(tc.Value, "azurerm_network_security_rule")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Network Security Rule access to trigger a validation error")
		}
	}
}

func TestResourceAzureRMNetworkSecurityRuleDirection_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Inbound",
			ErrCount: 0,
		},
		{
			Value:    "Outbound",
			ErrCount: 0,
		},
		{
			Value:    "INBOUND",
			ErrCount: 0,
		},
		{
			Value:    "Inbound",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateNetworkSecurityRuleDirection(tc.Value, "azurerm_network_security_rule")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Network Security Rule direction to trigger a validation error")
		}
	}
}
