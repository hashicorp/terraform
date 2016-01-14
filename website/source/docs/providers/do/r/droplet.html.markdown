---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_droplet"
sidebar_current: "docs-do-resource-droplet"
description: |-
  Provides a DigitalOcean droplet resource. This can be used to create, modify, and delete droplets. Droplets also support provisioning.
---

# digitalocean\_droplet

Provides a DigitalOcean droplet resource. This can be used to create,
modify, and delete droplets. Droplets also support
[provisioning](/docs/provisioners/index.html).

## Example Usage

```
# Create a new Web droplet in the nyc2 region
resource "digitalocean_droplet" "web" {
    image = "ubuntu-14-04-x64"
    name = "web-1"
    region = "nyc2"
    size = "512mb"
}
```

## Argument Reference

The following arguments are supported:

* `image` - (Required) The droplet image ID or slug.
* `name` - (Required) The droplet name
* `region` - (Required) The region to start in
* `size` - (Required) The instance size to start
* `backups` - (Optional) Boolean controlling if backups are made. Defaults to
   false.
* `ipv6` - (Optional) Boolean controlling if IPv6 is enabled. Defaults to false.
* `private_networking` - (Optional) Boolean controlling if private networks are
   enabled. Defaults to false.
* `ssh_keys` - (Optional) A list of SSH IDs or fingerprints to enable in
   the format `[12345, 123456]`. To retrieve this info, use a tool such
   as `curl` with the [DigitalOcean API](https://developers.digitalocean.com/#keys),
   to retrieve them.
* `user_data` (Optional) - A string of the desired User Data for the Droplet.
   User Data is currently only available in regions with metadata
   listed in their features.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the droplet
* `name`- The name of the droplet
* `region` - The region of the droplet
* `image` - The image of the droplet
* `ipv6` - Is IPv6 enabled
* `ipv6_address` - The IPv6 address
* `ipv6_address_private` - The private networking IPv6 address
* `ipv4_address` - The IPv4 address
* `ipv4_address_private` - The private networking IPv4 address
* `locked` - Is the Droplet locked
* `private_networking` - Is private networking enabled
* `size` - The instance size
* `status` - The status of the droplet

