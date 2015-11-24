---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_floating_ip"
sidebar_current: "docs-do-resource-floating-ip"
description: |-
  Provides a DigitalOcean Floating IP resource.
---

# digitalocean\_floating_ip

Provides a DigitalOcean Floating IP for the specified region. If the droplet_id 
is given, the Floating IP will be assigned to that droplet.

## Example Usage

```
# Create a new Floating IP
resource "digitalocean_floating_ip" "foobar" {
    region = "${digitalocean_droplet.foo.region}"
    droplet_id = "${digitalocean_droplet.foo.id}"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region to create the Floating IP in
* `droplet_id` - (Optional) The ID of the droplet to assign the Floating IP to

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the Floating IP. This is the actual IPv4 address.
* `region` - The region of the Floating IP
* `droplet_id` - The ID of the droplet to which the Floating IP is assigned
