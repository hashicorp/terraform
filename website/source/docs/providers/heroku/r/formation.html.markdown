---
layout: "heroku"
page_title: "Heroku: heroku_formation"
sidebar_current: "docs-heroku-resource-formation"
description: |-
  Provides a Heroku Formation resource. This can be used to scale processes or change dyno sizes on Heroku.
---

# heroku\_formation

Provides a Heroku Formation resource. This can be used to scale processes or change dyno sizes on Heroku.

## Example Usage

```hcl
resource "heroku_formation" "default" {
  app = "test-app"
  formation = "01234567-89ab-cdef-0123-456789abcdef"
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The app identifier (name or id) that you want to scale or change the size of.
* `formation` - (Required) The formation identifier (name or id).

## Attributes Reference

The following attributes are exported:

* `app_id` - The unique identifier for the app that you want to scale or increase the size of.
* `app_name` - The name of the app. Pattern: ^[a-z][a-z0-9-]{2,29}$
* `command` -	The command to use to launch this process.
* `created_at` - The datetime	when process type was created.
* `id` - The unique identifier of this process type.
* `quantity` - The number of processes to maintain.
* `size` - The dyno size (default: “standard-1X”).
* `type` - The type of process to maintain. Pattern: ^[-\w]{1,128}$
* `updated_at` - When dyno type was updated.
