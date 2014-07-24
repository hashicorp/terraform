---
layout: "digitalocean"
page_title: "Provider: DigitalOcean"
sidebar_current: "docs-do-index"
---

# DigitalOcean Provider

The DigitalOcean (DO) provider is used to interact with the
resources supported by DigitalOcean. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the DigitalOcean Provider
provider "digitalocean" {
    token = "${var.do_token}"
}

# Create a web server
resource "digitalocean_droplet" "web" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `token` - (Required) This is the DO API token.

