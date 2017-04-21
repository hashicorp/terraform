---
layout: "rabbitmq"
page_title: "RabbitMQ: rabbitmq_policy"
sidebar_current: "docs-rabbitmq-resource-policy"
description: |-
  Creates and manages a policy on a RabbitMQ server.
---

# rabbitmq\_policy

The ``rabbitmq_policy`` resource creates and manages policies for exchanges
and queues.

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

resource "rabbitmq_policy" "test" {
  name  = "test"
  vhost = "${rabbitmq_permissions.guest.vhost}"

  policy {
    pattern  = ".*"
    priority = 0
    apply_to = "all"

    definition {
      ha-mode = "all"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the policy.

* `vhost` - (Required) The vhost to create the resource in.

* `policy` - (Required) The settings of the policy. The structure is
  described below.

The `policy` block supports:

* `pattern` - (Required) A pattern to match an exchange or queue name.
* `priority` - (Required) The policy with the greater priority is applied first.
* `apply_to` - (Required) Can either be "exchange", "queues", or "all".
* `definition` - (Required) Key/value pairs of the policy definition. See the
  RabbitMQ documentation for definition references and examples.

## Attributes Reference

No further attributes are exported.

## Import

Policies can be imported using the `id` which is composed of `name@vhost`.
E.g.

```
terraform import rabbitmq_policy.test name@vhost
```
