---
layout: "heroku"
page_title: "Heroku: heroku_domain"
sidebar_current: "docs-heroku-resource-domain"
description: |-
  Provides a Heroku App resource. This can be used to create and manage applications on Heroku.
---

# heroku\_domain

Provides a Heroku App resource. This can be used to
create and manage applications on Heroku.

## Example Usage

```hcl
# Create a new Heroku app
resource "heroku_app" "default" {
  name = "test-app"
}

# Associate a custom domain
resource "heroku_domain" "default" {
  app      = "${heroku_app.default.name}"
  hostname = "terraform.example.com"
}
```

## Argument Reference

The following arguments are supported:

* `hostname` - (Required) The hostname to serve requests from.
* `app` - (Required) The Heroku app to link to.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the of the domain record.
* `hostname` - The hostname traffic will be served as.
* `cname` - The CNAME traffic should route to.

