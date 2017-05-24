---
layout: "rabbitmq"
page_title: "RabbitMQ: rabbitmq_permissions"
sidebar_current: "docs-rabbitmq-resource-permissions"
description: |-
  Creates and manages a user's permissions on a RabbitMQ server.
---

# rabbitmq\_permissions

The ``rabbitmq_permissions`` resource creates and manages a user's set of
permissions.

## Example Usage

```hcl
resource "rabbitmq_vhost" "test" {
  name = "test"
}

resource "rabbitmq_user" "test" {
  name     = "mctest"
  password = "foobar"
  tags     = ["administrator"]
}

resource "rabbitmq_permissions" "test" {
  user  = "${rabbitmq_user.test.name}"
  vhost = "${rabbitmq_vhost.test.name}"

  permissions {
    configure = ".*"
    write     = ".*"
    read      = ".*"
  }
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) The user to apply the permissions to.

* `vhost` - (Required) The vhost to create the resource in.

* `permissions` - (Required) The settings of the permissions. The structure is
  described below.

The `permissions` block supports:

* `configure` - (Required) The "configure" ACL.
* `write` - (Required) The "write" ACL.
* `read` - (Required) The "read" ACL.

## Attributes Reference

No further attributes are exported.

## Import

Permissions can be imported using the `id` which is composed of  `user@vhost`.
E.g.

```
terraform import rabbitmq_permissions.test user@vhost
```
