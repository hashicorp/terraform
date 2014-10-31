---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_record"
sidebar_current: "docs-do-resource-record"
description: |-
  Provides a DigitalOcean domain resource.
---

# digitalocean\_record

Provides a DigitalOcean domain resource.

## Example Usage

```
# Create a new domain record
resource "digitalocean_domain" "default" {
    name = "www.example.com"
    ip_address = "${digitalocean_droplet.foo.ipv4_address}"
}

# Add a record to the domain
resource "digitalocean_record" "foobar" {
    domain = "${digitalocean_domain.default.name}"
    type = "A"
    name = "foobar"
    value = "192.168.0.11"
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The type of record
* `domain` - (Required) The domain to add the record to
* `value` - (Optional) The value of the record
* `name` - (Optional) The name of the record
* `weight` - (Optional) The weight of the record
* `port` - (Optional) The port of the record
* `priority` - (Optional) The priority of the record

## Attributes Reference

The following attributes are exported:

* `id` - The record ID

