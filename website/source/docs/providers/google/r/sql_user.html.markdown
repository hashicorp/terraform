---
layout: "google"
page_title: "Google: google_sql_user"
sidebar_current: "docs-google-sql-user"
description: |-
  Creates a new SQL user in Google Cloud SQL.
---

# google\_sql\_user

Creates a new Google SQL User on a Google SQL User Instance. For more information, see the [official documentation](https://cloud.google.com/sql/), or the [JSON API](https://cloud.google.com/sql/docs/admin-api/v1beta4/users).

~> **Note:** All arguments including the username and password will be stored in the raw state as plain-text.
[Read more about sensitive data in state](/docs/state/sensitive-data.html). Passwords will not be retrieved when running
"terraform import".

## Example Usage

Example creating a SQL User.

```hcl
resource "google_sql_database_instance" "master" {
  name = "master-instance"

  settings {
    tier = "D0"
  }
}

resource "google_sql_user" "users" {
  name     = "me"
  instance = "${google_sql_database_instance.master.name}"
  host     = "me.com"
  password = "changeme"
}
```

## Argument Reference

The following arguments are supported:

* `host` - (Required) The host the user can connect from. Can be an IP address.
    Changing this forces a new resource to be created.

* `instance` - (Required) The name of the Cloud SQL instance. Changing this
    forces a new resource to be created.

* `name` - (Required) The name of the user. Changing this forces a new resource
    to be created.

* `password` - (Required) The users password. Can be updated.

- - -

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

Only the arguments listed above are exposed as attributes.

## Import Format

Importing an SQL user is formatted as:

```bash
terraform import google_sql_user.$RESOURCENAME $INSTANCENAME/$SQLUSERNAME
```

For example, the sample at the top of this page could be imported with:

```bash
terraform import google_sql_user.users master-instance/me
```