---
layout: "mysql"
page_title: "MySQL: mysql_user"
sidebar_current: "docs-mysql-resource-user"
description: |-
  Creates and manages a user on a MySQL server.
---

# mysql\_user

The ``mysql_user`` resource creates and manages a user on a MySQL
server.

~> **Note:** All arguments including username and password will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html).

## Example Usage

```hcl
resource "mysql_user" "jdoe" {
  user     = "jdoe"
  host     = "example.com"
  password = "password"
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) The name of the user.

* `host` - (Optional) The source host of the user. Defaults to "localhost".

* `password` - (Optional) The password of the user. The value of this
  argument is plain-text so make sure to secure where this is defined.

## Attributes Reference

No further attributes are exported.
