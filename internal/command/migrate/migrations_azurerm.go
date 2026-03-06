// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"fmt"
	"regexp"
)

func azurermMigrations() []Migration {
	return []Migration{
		{
			Namespace:   "hashicorp",
			Provider:    "azurerm",
			Name:        "v3-to-v4",
			Description: "Migrates Azure provider configuration from v3 to v4, extracting inline sub-resources.",
			SubMigrations: []SubMigration{
				{
					Name:        "subnet-delegation",
					Description: "Extracts delegation block from azurerm_subnet into a separate azurerm_subnet_delegation resource.",
					Apply:       applySubnetDelegation,
				},
				{
					Name:        "network-security-rule",
					Description: "Extracts security_rule block from azurerm_network_security_group into a separate azurerm_network_security_rule resource.",
					Apply:       applyNetworkSecurityRule,
				},
			},
		},
	}
}

var subnetResourceRe = regexp.MustCompile(`resource\s+"azurerm_subnet"\s+"([^"]+)"`)
var nsgResourceRe = regexp.MustCompile(`resource\s+"azurerm_network_security_group"\s+"([^"]+)"`)

func applySubnetDelegation(_ string, src []byte) ([]byte, error) {
	s := string(src)

	m := subnetResourceRe.FindStringSubmatch(s)
	if m == nil {
		return src, nil
	}
	name := m[1]

	fullBlock, _, _, _, ok := extractNestedBlock(s, "delegation")
	if !ok {
		return src, nil
	}

	result := removeBlockFromSource(s, fullBlock)

	dedented := dedentBlock(fullBlock)
	reindented := indentBlock(dedented, "  ")

	newResource := fmt.Sprintf("\nresource \"azurerm_subnet_delegation\" %q {\n  subnet_id = azurerm_subnet.%s.id\n\n%s\n}\n", name, name, reindented)
	result = result + newResource

	return []byte(result), nil
}

func applyNetworkSecurityRule(_ string, src []byte) ([]byte, error) {
	s := string(src)

	m := nsgResourceRe.FindStringSubmatch(s)
	if m == nil {
		return src, nil
	}
	name := m[1]

	fullBlock, _, _, _, ok := extractNestedBlock(s, "security_rule")
	if !ok {
		return src, nil
	}

	result := removeBlockFromSource(s, fullBlock)

	dedented := dedentBlock(fullBlock)
	reindented := indentBlock(dedented, "  ")

	newResource := fmt.Sprintf("\nresource \"azurerm_network_security_rule\" %q {\n  network_security_group_name = azurerm_network_security_group.%s.name\n  resource_group_name         = azurerm_resource_group.%s.name\n\n%s\n}\n", name, name, name, reindented)
	result = result + newResource

	return []byte(result), nil
}
