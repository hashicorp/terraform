---
layout: "rabbitmq"
page_title: "RabbitMQ: rabbitmq_user"
sidebar_current: "docs-rabbitmq-resource-user"
description: |-
  Creates and manages a user on a RabbitMQ server.
---

# rabbitmq\_user

The ``rabbitmq_user`` resource creates and manages a user.

## Example Usage

```
resource "rabbitmq_user" "test" {
    name = "mctest"
    password = "foobar"
    tags = ["administrator", "management"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the user.

* `password` - (Required) The password of the user. The value of this argument
  is plain-text so make sure to secure where this is defined.

* `tags` - (Required) Which permission model to apply to the user. Valid
  options are: management, policymaker, monitoring, and administrator.

## Attributes Reference

No further attributes are exported.

## Import

Users can be imported using the `name`, e.g.

```
terraform import rabbitmq_user.test mctest
```
