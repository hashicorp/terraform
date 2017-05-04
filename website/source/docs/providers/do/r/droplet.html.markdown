---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_droplet"
sidebar_current: "docs-do-resource-droplet"
description: |-
  Provides a DigitalOcean Droplet resource. This can be used to create, modify, and delete Droplets. Droplets also support provisioning.
---

# digitalocean\_droplet

Provides a DigitalOcean Droplet resource. This can be used to create,
modify, and delete Droplets. Droplets also support
[provisioning](/docs/provisioners/index.html).

## Example Usage

```hcl
# Create a new Web Droplet in the nyc2 region
resource "digitalocean_droplet" "web" {
  image  = "ubuntu-14-04-x64"
  name   = "web-1"
  region = "nyc2"
  size   = "512mb"
}
```

## Argument Reference

The following arguments are supported:

* `image` - (Required) The Droplet image ID or slug.
* `name` - (Required) The Droplet name
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
* `resize_disk` - (Optional) Boolean controlling whether to increase the disk
   size when resizing a Droplet. It defaults to `true`. When set to `false`,
   only the Droplet's RAM and CPU will be resized. **Increasing a Droplet's disk
   size is a permanent change**. Increasing only RAM and CPU is reversible.
* `tags` - (Optional) A list of the tags to label this droplet. A tag resource
   must exist before it can be associated with a droplet.
* `user_data` (Optional) - A string of the desired User Data for the Droplet.
* `volume_ids` (Optional) - A list of the IDs of each [block storage volume](/docs/providers/do/r/volume.html) to be attached to the Droplet.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Droplet
* `name`- The name of the Droplet
* `region` - The region of the Droplet
* `image` - The image of the Droplet
* `ipv6` - Is IPv6 enabled
* `ipv6_address` - The IPv6 address
* `ipv6_address_private` - The private networking IPv6 address
* `ipv4_address` - The IPv4 address
* `ipv4_address_private` - The private networking IPv4 address
* `locked` - Is the Droplet locked
* `private_networking` - Is private networking enabled
* `price_hourly` - Droplet hourly price
* `price_monthly` - Droplet monthly price
* `size` - The instance size
* `disk` - The size of the instance's disk in GB
* `vcpus` - The number of the instance's virtual CPUs
* `status` - The status of the droplet
* `tags` - The tags associated with the droplet
* `volume_ids` - A list of the attached block storage volumes

## Import

Droplets can be imported using the droplet `id`, e.g.

```
terraform import digitalocean_droplet.mydroplet 100823
```
