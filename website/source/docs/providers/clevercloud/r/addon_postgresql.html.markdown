---
layout: "clevercloud"
page_title: "Clever Cloud: clevercloud_application_postgresql"
sidebar_current: "docs-clevercloud-resource-application-postgresql"
description: |-
  Allows Terraform to manage a PostgreSQL addon on Clever Cloud.
---

# clevercloud\_application\_postgresql

Allows Terraform to manage a PostgreSQL application on Clever Cloud.

## Example Usage

```
resource "clevercloud_addon_postgresql" "mydb" {
    name = "Hello World Database"
    plan = "dev"
    region = "eu"
}
```

## Argument Reference

The following arguments are supported:

* `Name` - (Required, string) The name of the addon.

* `Plan` - (Required, string) The plan of the addon.

* `Region` - (Optional, string, default: `eu` - Europe) The region of the addon.

## Attributes Reference

The following attributes are exported:

* `id` - The addon ID.

* `real_id` - The addon real ID.

* `price` - The addon monthly cost.

* `environment` - Exports credentials of the addon/database.
