---
layout: "heroku"
page_title: "Heroku: heroku_cert"
sidebar_current: "docs-heroku-resource-cert"
description: |-
  Provides a Heroku SSL certificate resource. It allows to set a given certificate for a Heroku app.
---

# heroku\_cert

Provides a Heroku SSL certificate resource. It allows to set a given certificate for a Heroku app.

## Example Usage

```
# Create a new heroku app
resource "heroku_app" "default" {
    name = "test-app"
}

# Add-on SSL to application
resource "heroku_addon" "ssl" {
    app = "${heroku_app.service.name}"
    plan = "ssl"
}

# Establish certificate for a given application
resource "heroku_cert" "ssl_certificate" {
    app = "${heroku_app.service.name}"
    certificate_chain = "${file("server.crt")}"
    private_key = "${file("server.key")}"
    depends_on = "heroku_addon.ssl"
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The Heroku app to add to.
* `certificate_chain` - (Required) The certificate chain to add
* `private_key` - (Required) The private key for a given certificate chain

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the add-on
* `cname` - The CNAME of ssl endpoint
* `name` - The name of the SSL certificate

