---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_servicebus_topic"
sidebar_current: "docs-azurerm-resource-servicebus-topic"
description: |-
  Create a ServiceBus Topic.
---

# azurerm\_servicebus\_topic

Create a ServiceBus Topic.

**Note** Topics can only be created in Namespaces with an SKU or `standard` or
higher.

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
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the ServiceBus Topic resource. Changing this forces a
    new resource to be created.

* `namespace_name` - (Required) The name of the ServiceBus Namespace to create
    this topic in. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists.
    Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the namespace. Changing this forces a new resource to be created.

* `auto_delete_on_idle` - (Optional) The idle interval after which the
    Topic is automatically deleted, minimum of 5 minutes. Provided in the [TimeSpan](#timespan-format)
    format.

* `default_message_ttl` - (Optional) The TTL of messages sent to this topic if no
    TTL value is set on the message itself. Provided in the [TimeSpan](#timespan-format)
    format.

* `duplicate_detection_history_time_window` - (Optional) The duration during which
    duplicates can be detected. Provided in the [TimeSpan](#timespan-format) format. Defaults to 10 minutes (`00:10:00`)

* `enable_batched_operations` - (Optional) Boolean flag which controls if server-side
    batched operations are enabled. Defaults to false.

* `enable_express` - (Optional) Boolean flag which controls whether Express Entities
    are enabled. An express topic holds a message in memory temporarily before writing
    it to persistent storage. Defaults to false.

* `enable_filtering_messages_before_publishing` - (Optional) Boolean flag which
    controls whether messages should be filtered before publishing. Defaults to
    false.

* `enable_partitioning` - (Optional) Boolean flag which controls whether to enable
    the topic to be partitioned across multiple message brokers. Defaults to false.
    Changing this forces a new resource to be created.

* `max_size_in_megabytes` - (Optional) Integer value which controls the size of
    memory allocated for the topic. For supported values see the "Queue/topic size"
    section of [this document](https://docs.microsoft.com/en-us/azure/service-bus-messaging/service-bus-quotas).

* `requires_duplicate_detection` - (Optional) Boolean flag which controls whether
    the Topic requires duplicate detection. Defaults to false. Changing this forces
    a new resource to be created.

* `support_ordering` - (Optional) Boolean flag which controls whether the Topic
    supports ordering. Defaults to false.

### TimeSpan Format

Some arguments for this resource are required in the TimeSpan format which is
used to represent a lengh of time. The supported format is documented [here](https://msdn.microsoft.com/en-us/library/se73z7b9(v=vs.110).aspx#Anchor_2)

## Attributes Reference

The following attributes are exported:

* `id` - The ServiceBus Topic ID.

## Import

Service Bus Topics can be imported using the `resource id`, e.g.

```
terraform import azurerm_servicebus_topic.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/microsoft.servicebus/namespaces/sbns1/topics/sntopic1
```
