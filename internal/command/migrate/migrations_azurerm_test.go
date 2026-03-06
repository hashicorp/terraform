// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAzureSubnetDelegation(t *testing.T) {
	sub := findSubMigration(t, azurermMigrations(), "hashicorp/azurerm/v3-to-v4", "subnet-delegation")

	tests := map[string]struct {
		input    string
		expected string
	}{
		"extracts delegation into separate resource": {
			input: `resource "azurerm_subnet" "example" {
  name                 = "example-subnet"
  resource_group_name  = azurerm_resource_group.example.name
  virtual_network_name = azurerm_virtual_network.example.name

  delegation {
    name = "delegation"
    service_delegation {
      name = "Microsoft.ContainerInstance/containerGroups"
    }
  }
}
`,
			expected: `resource "azurerm_subnet" "example" {
  name                 = "example-subnet"
  resource_group_name  = azurerm_resource_group.example.name
  virtual_network_name = azurerm_virtual_network.example.name
}

resource "azurerm_subnet_delegation" "example" {
  subnet_id = azurerm_subnet.example.id

  delegation {
    name = "delegation"
    service_delegation {
      name = "Microsoft.ContainerInstance/containerGroups"
    }
  }
}
`,
		},
		"no match leaves input unchanged": {
			input: `resource "azurerm_subnet" "example" {
  name = "example-subnet"
}
`,
			expected: `resource "azurerm_subnet" "example" {
  name = "example-subnet"
}
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := sub.Apply("main.tf", []byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAzureNetworkSecurityRule(t *testing.T) {
	sub := findSubMigration(t, azurermMigrations(), "hashicorp/azurerm/v3-to-v4", "network-security-rule")

	tests := map[string]struct {
		input    string
		expected string
	}{
		"extracts security_rule into separate resource": {
			input: `resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  location            = "eastus"
  resource_group_name = azurerm_resource_group.example.name

  security_rule {
    name                       = "allow-ssh"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    destination_port_range     = "22"
  }
}
`,
			expected: `resource "azurerm_network_security_group" "example" {
  name                = "example-nsg"
  location            = "eastus"
  resource_group_name = azurerm_resource_group.example.name
}

resource "azurerm_network_security_rule" "example" {
  network_security_group_name = azurerm_network_security_group.example.name
  resource_group_name         = azurerm_resource_group.example.name

  security_rule {
    name                       = "allow-ssh"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    destination_port_range     = "22"
  }
}
`,
		},
		"no match leaves input unchanged": {
			input: `resource "azurerm_network_security_group" "example" {
  name = "example-nsg"
}
`,
			expected: `resource "azurerm_network_security_group" "example" {
  name = "example-nsg"
}
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := sub.Apply("main.tf", []byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}
