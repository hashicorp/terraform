---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_domain"
sidebar_current: "docs-do-resource-domain"
description: |-
  Provides a DigitalOcean domain resource.
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
* `ip_address` - (Required) The IP address of the domain. This IP
   is used to created an initial A record for the domain. It is required
   upstream by the DigitalOcean API.

## Attributes Reference

The following attributes are exported:

* `id` - The name of the domain

