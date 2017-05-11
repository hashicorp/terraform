---
layout: "rabbitmq"
page_title: "RabbitMQ: rabbitmq_exchange"
sidebar_current: "docs-rabbitmq-resource-exchange"
description: |-
  Creates and manages an exchange on a RabbitMQ server.
---

# rabbitmq\_exchange

The ``rabbitmq_exchange`` resource creates and manages an exchange.

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
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the exchange.

* `vhost` - (Required) The vhost to create the resource in.

* `settings` - (Required) The settings of the exchange. The structure is
  described below.

The `settings` block supports:

* `type` - (Required) The type of exchange.

* `durable` - (Optional) Whether the exchange survives server restarts.
  Defaults to `false`.

* `auto_delete` - (Optional) Whether the exchange will self-delete when all
  queues have finished using it.

* `arguments` - (Optional) Additional key/value settings for the exchange.

## Attributes Reference

No further attributes are exported.

## Import

Exchanges can be imported using the `id` which is composed of  `name@vhost`.
E.g.

```
terraform import rabbitmq_exchange.test test@vhost
```
