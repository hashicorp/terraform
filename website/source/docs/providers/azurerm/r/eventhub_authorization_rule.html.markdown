---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_eventhub_authorization_rule"
sidebar_current: "docs-azurerm-resource-eventhub-authorization-rule"
description: |-
  Creates a new Event Hub Authorization Rule within an Event Hub.
---

# azurerm\_eventhub\_authorization\_rule

Creates a new Event Hub Authorization Rule within an Event Hub.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "resourceGroup1"
  location = "West US"
}

resource "azurerm_eventhub_namespace" "test" {
  name                = "acceptanceTestEventHubNamespace"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku                 = "Basic"
  capacity            = 2

  tags {
    environment = "Production"
  }
}

resource "azurerm_eventhub" "test" {
  name                = "acceptanceTestEventHub"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  partition_count     = 2
  message_retention   = 2
}

resource "azurerm_eventhub_authorization_rule" "test" {
  name                = "navi"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  eventhub_name       = "${azurerm_eventhub.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  listen              = true
  send                = false
  manage              = false
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the EventHub Authorization Rule resource. Changing this forces a new resource to be created.

* `namespace_name` - (Required) Specifies the name of the grandparent EventHub Namespace. Changing this forces a new resource to be created.

* `eventhub_name` - (Required) Specifies the name of the EventHub. Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which the EventHub Namespace exists. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

~> **NOTE** At least one of the 3 permissions below needs to be set.

* `listen` - (Optional) Does this Authorization Rule have permissions to Listen to the Event Hub? Defaults to `false`.

* `send` - (Optional) Does this Authorization Rule have permissions to Send to the Event Hub? Defaults to `false`.

* `manage` - (Optional) Does this Authorization Rule have permissions to Manage to the Event Hub? When this property is `true` - both `listen` and `send` must be too. Defaults to `false`.

## Attributes Reference

The following attributes are exported:

* `id` - The EventHub ID.

* `primary_key` - The Primary Key for the Event Hub Authorization Rule.

* `primary_connection_string` - The Primary Connection String for the Event Hub Authorization Rule.

* `secondary_key` - The Secondary Key for the Event Hub Authorization Rule.

* `secondary_connection_string` - The Secondary Connection String for the Event Hub Authorization Rule.

## Import

EventHubs can be imported using the `resource id`, e.g.

```
terraform import azurerm_eventhub_authorization_rule.rule1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.EventHub/namespaces/namespace1/eventhubs/eventhub1/authorizationRules/rule1
```
