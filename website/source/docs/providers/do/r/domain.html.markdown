---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_domain"
sidebar_current: "docs-do-resource-domain"
---

# digitalocean\_domain

Provides a DigitalOcean domain resource.

## Example Usage

```
# Create a new domain record
resource "digitalocean_domain" "default" {
    name = "www.example.com"
    ip_address = "${digitalocean_droplet.foo.ipv4_address}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the domain
* `ip_address` - (Required) The IP address of the domain

## Attributes Reference

The following attributes are exported:

* `id` - The name of the domain

