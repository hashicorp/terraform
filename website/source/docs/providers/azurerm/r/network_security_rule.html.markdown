---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_network_security_rule"
sidebar_current: "docs-azurerm-resource-network-security-rule"
description: |-
  Create a Network Security Rule.
---

# azurerm\_network\_security\_rule

Create a Network Security Rule.

## Example Usage

```
resource "azurerm_resource_group" "test" {
    name = "acceptanceTestResourceGroup1"
    location = "West US"
}

resource "azurerm_network_security_group" "test" {
    name = "acceptanceTestSecurityGroup1"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
}

resource "azurerm_network_security_rule" "test" {
	name = "test123"
    priority = 100
    direction = "Outbound"
    access = "Allow"
    protocol = "Tcp"
    source_port_range = "*"
    destination_port_range = "*"
    source_address_prefix = "*"
    destination_address_prefix = "*"
    resource_group_name = "${azurerm_resource_group.test.name}"
    network_security_group_name = "${azurerm_network_security_group.test.name}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the security rule.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the Network Security Rule.
 
* `network_security_group_name` - (Required) The name of the Network Security Group that we want to attach the rule to.

* `description` - (Optional) A description for this rule. Restricted to 140 characters.

* `protocol` - (Required) Network protocol this rule applies to. Can be Tcp, Udp or * to match both.

* `source_port_range` - (Required) Source Port or Range. Integer or range between 0 and 65535 or * to match any.

* `destination_port_range` - (Required) Destination Port or Range. Integer or range between 0 and 65535 or * to match any.

* `source_address_prefix` - (Required) CIDR or source IP range or * to match any IP. Tags such as ‘VirtualNetwork’, ‘AzureLoadBalancer’ and ‘Internet’ can also be used.

* `destination_address_prefix` - (Required) CIDR or destination IP range or * to match any IP. Tags such as ‘VirtualNetwork’, ‘AzureLoadBalancer’ and ‘Internet’ can also be used.

* `access` - (Required) Specifies whether network traffic is allowed or denied. Possible values are “Allow” and “Deny”.

* `priority` - (Required) Specifies the priority of the rule. The value can be between 100 and 4096. The priority number must be unique for each rule in the collection. The lower the priority number, the higher the priority of the rule.

* `direction` - (Required) The direction specifies if rule will be evaluated on incoming or outgoing traffic. Possible values are “Inbound” and “Outbound”.

## Attributes Reference

The following attributes are exported:

* `id` - The Network Security Rule ID.