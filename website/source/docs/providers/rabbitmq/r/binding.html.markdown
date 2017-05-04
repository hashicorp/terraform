---
layout: "rabbitmq"
page_title: "RabbitMQ: rabbitmq_binding"
sidebar_current: "docs-rabbitmq-resource-binding"
description: |-
  Creates and manages a binding on a RabbitMQ server.
---

# rabbitmq\_binding

The ``rabbitmq_binding`` resource creates and manages a binding relationship
between a queue an exchange.

## Example Usage

```hcl
resource "rabbitmq_vhost" "test" {
  name = "test"
}

resource "rabbitmq_permissions" "guest" {
  user  = "guest"
  vhost = "${rabbitmq_vhost.test.name}"

  permissions {
    configure = ".*"
    write     = ".*"
    read      = ".*"
  }
}

resource "rabbitmq_exchange" "test" {
  name  = "test"
  vhost = "${rabbitmq_permissions.guest.vhost}"

  settings {
    type        = "fanout"
    durable     = false
    auto_delete = true
  }
}

resource "rabbitmq_queue" "test" {
  name  = "test"
  vhost = "${rabbitmq_permissions.guest.vhost}"

  settings {
    durable     = true
    auto_delete = false
  }
}

resource "rabbitmq_binding" "test" {
  source           = "${rabbitmq_exchange.test.name}"
  vhost            = "${rabbitmq_vhost.test.name}"
  destination      = "${rabbitmq_queue.test.name}"
  destination_type = "queue"
  routing_key      = "#"
  properties_key   = "%23"
}
```

## Argument Reference

The following arguments are supported:

* `source` - (Required) The source exchange.

* `vhost` - (Required) The vhost to create the resource in.

* `destination` - (Required) The destination queue or exchange.

* `destination_type` - (Required) The type of destination (queue or exchange).

* `properties_key` - (Required) A unique key to refer to the binding.

* `routing_key` - (Optional) A routing key for the binding.

* `arguments` - (Optional) Additional key/value arguments for the binding.

## Attributes Reference

No further attributes are exported.

## Import

Bindings can be imported using the `id` which is composed of
  `vhost/source/destination/destination_type/properties_key`. E.g.

```
$ terraform import rabbitmq_binding.test test/test/test/queue/%23
```
