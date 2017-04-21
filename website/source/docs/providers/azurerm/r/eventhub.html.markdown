---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_eventhub"
sidebar_current: "docs-azurerm-resource-eventhub"
description: |-
  Creates a new Event Hub as a nested resource within an Event Hub Namespace.
---

# azurerm\_eventhub

Creates a new Event Hub as a nested resource within a Event Hub Namespace.

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
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the EventHub Namespace resource. Changing this forces a new resource to be created.

* `namespace_name` - (Required) Specifies the name of the EventHub Namespace. Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which the EventHub's parent Namespace exists. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `partition_count` - (Required) Specifies the current number of shards on the Event Hub.

* `message_retention` - (Required) Specifies the number of days to retain the events for this Event Hub. Needs to be between 1 and 7 days; or 1 day when using a Basic SKU for the parent EventHub Namespace.

## Attributes Reference

The following attributes are exported:

* `id` - The EventHub ID.

* `partition_ids` - The identifiers for partitions created for Event Hubs.

## Import

EventHubs can be imported using the `resource id`, e.g.

```
terraform import azurerm_eventhub.eventhub1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.EventHub/namespaces/namespace1/eventhubs/eventhub1
```
