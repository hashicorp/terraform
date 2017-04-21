---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_eventhub_consumer_group"
sidebar_current: "docs-azurerm-resource-eventhub-consumer-group"
description: |-
  Creates a new Event Hub Consumer Group as a nested resource within an Event Hub.
---

# azurerm\_eventhub\_consumer\_group

Creates a new Event Hub Consumer Group as a nested resource within an Event Hub.

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

resource "azurerm_eventhub_consumer_group" "test" {
  name                = "acceptanceTestEventHubConsumerGroup"
  namespace_name      = "${azurerm_eventhub_namespace.test.name}"
  eventhub_name       = "${azurerm_eventhub.test.name}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  location            = "${azurerm_resource_group.test.location}"
  user_metadata       = "some-meta-data"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the EventHub Consumer Group resource. Changing this forces a new resource to be created.

* `namespace_name` - (Required) Specifies the name of the grandparent EventHub Namespace. Changing this forces a new resource to be created.

* `eventhub_name` - (Required) Specifies the name of the EventHub. Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which the EventHub Consumer Group's grandparent Namespace exists. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `user_metadata` - (Optional) Specifies the user metadata.

## Attributes Reference

The following attributes are exported:

* `id` - The EventHub Consumer Group ID.

## Import

EventHub Consumer Groups can be imported using the `resource id`, e.g.

```
terraform import azurerm_eventhub_consumer_group.consumerGroup1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.EventHub/namespaces/namespace1/eventhubs/eventhub1/consumergroups/consumerGroup1
```
