---
layout: "heroku"
page_title: "Heroku: heroku_addon"
sidebar_current: "docs-heroku-resource-addon"
description: |-
  Provides a Heroku Add-On resource. These can be attach services to a Heroku app.
---

# heroku\_addon

Provides a Heroku Add-On resource. These can be attach
services to a Heroku app.

## Example Usage

```hcl
# Create a new Heroku app
resource "heroku_app" "default" {
  name = "test-app"
}

# Create a database, and configure the app to use it
resource "heroku_addon" "database" {
  app  = "${heroku_app.default.name}"
  plan = "heroku-postgresql:hobby-basic"
}

# Add a web-hook addon for the app
resource "heroku_addon" "webhook" {
  app  = "${heroku_app.default.name}"
  plan = "deployhooks:http"

  config {
    url = "http://google.com"
  }
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The Heroku app to add to.
* `plan` - (Required) The addon to add.
* `config` - (Optional) Optional plan configuration.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the add-on
* `name` - The add-on name
* `plan` - The plan name
* `provider_id` - The ID of the plan provider


## Import

Addons can be imported using the Addon `id`, e.g.

```
$ terraform import heroku_addon.foobar 12345678
```
