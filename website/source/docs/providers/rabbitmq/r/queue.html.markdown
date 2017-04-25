---
layout: "rabbitmq"
page_title: "RabbitMQ: rabbitmq_queue"
sidebar_current: "docs-rabbitmq-resource-queue"
description: |-
  Creates and manages a queue on a RabbitMQ server.
---

# rabbitmq\_queue

The ``rabbitmq_queue`` resource creates and manages a queue.

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

resource "rabbitmq_queue" "test" {
  name  = "test"
  vhost = "${rabbitmq_permissions.guest.vhost}"

  settings {
    durable     = false
    auto_delete = true
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the queue.

* `vhost` - (Required) The vhost to create the resource in.

* `settings` - (Required) The settings of the queue. The structure is
  described below.

The `settings` block supports:

* `durable` - (Optional) Whether the queue survives server restarts.
  Defaults to `false`.

* `auto_delete` - (Optional) Whether the queue will self-delete when all
  consumers have unsubscribed.

* `arguments` - (Optional) Additional key/value settings for the queue.

## Attributes Reference

No further attributes are exported.

## Import

Queues can be imported using the `id` which is composed of `name@vhost`. E.g.

```
terraform import rabbitmq_queue.test name@vhost
```
