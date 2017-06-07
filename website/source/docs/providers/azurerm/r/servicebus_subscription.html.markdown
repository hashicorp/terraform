---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_servicebus_subscription"
sidebar_current: "docs-azurerm-resource-servicebus-subscription"
description: |-
  Create a ServiceBus Subscription.
---

# azurerm\_servicebus\_subscription

Create a ServiceBus Subscription.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "resourceGroup1"
  location = "West US"
}

resource "azurerm_servicebus_namespace" "test" {
  name                = "acceptanceTestServiceBusNamespace"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku                 = "standard"

  tags {
    environment = "Production"
  }
}

resource "azurerm_servicebus_topic" "test" {
  name                = "testTopic"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  namespace_name      = "${azurerm_servicebus_namespace.test.name}"

  enable_partitioning = true
}

resource "azurerm_servicebus_subscription" "test" {
  name                = "testSubscription"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  namespace_name      = "${azurerm_servicebus_namespace.test.name}"
  topic_name          = "${azurerm_servicebus_topic.test.name}"
  max_delivery_count  = 1
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the ServiceBus Subscription resource.
    Changing this forces a new resource to be created.

* `namespace_name` - (Required) The name of the ServiceBus Namespace to create
    this Subscription in. Changing this forces a new resource to be created.

* `topic_name` - (Required) The name of the ServiceBus Topic to create
    this Subscription in. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists.
    Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the namespace. Changing this forces a new resource to be created.

* `max_delivery_count` - (Required) The maximum number of deliveries.

* `auto_delete_on_idle` - (Optional) The idle interval after which the
    Subscription is automatically deleted, minimum of 5 minutes. Provided in the
    [TimeSpan](#timespan-format) format.

* `default_message_ttl` - (Optional) The TTL of messages sent to this Subscription
    if no TTL value is set on the message itself. Provided in the [TimeSpan](#timespan-format)
    format.

* `lock_duration` - (Optional) The lock duration for the subscription, maximum
    supported value is 5 minutes. Defaults to 1 minute.

* `dead_lettering_on_filter_evaluation_exceptions` - (Optional) Boolean flag which
    controls whether the Subscription has dead letter support on Filter evaluation
    exceptions. Defaults to false.

* `dead_lettering_on_message_expiration` - (Optional) Boolean flag which controls
    whether the Subscription has dead letter support when a message expires. Defaults
    to false.

* `enable_batched_operations` - (Optional) Boolean flag which controls whether the
    Subscription supports batched operations. Defaults to false.

* `requires_session` - (Optional) Boolean flag which controls whether this Subscription
    supports the concept of a session. Defaults to false. Changing this forces a
    new resource to be created.

### TimeSpan Format

Some arguments for this resource are required in the TimeSpan format which is
used to represent a length of time. The supported format is documented [here](https://msdn.microsoft.com/en-us/library/se73z7b9(v=vs.110).aspx#Anchor_2)

## Attributes Reference

The following attributes are exported:

* `id` - The ServiceBus Subscription ID.

## Import

Service Bus Subscriptions can be imported using the `resource id`, e.g.

```
terraform import azurerm_servicebus_subscription.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/microsoft.servicebus/namespaces/sbns1/topics/sntopic1/subscriptions/sbsub1
```
