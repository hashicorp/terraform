---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_floating_ip"
sidebar_current: "docs-do-resource-floating-ip"
description: |-
  Provides a DigitalOcean Floating IP resource.
---

# digitalocean\_floating_ip

Provides a DigitalOcean Floating IP to represent a publicly-accessible static IP addresses that can be mapped to one of your Droplets.

## Example Usage

```hcl
resource "digitalocean_droplet" "foobar" {
  name               = "baz"
  size               = "1gb"
  image              = "centos-5-8-x32"
  region             = "sgp1"
  ipv6               = true
  private_networking = true
}

resource "digitalocean_floating_ip" "foobar" {
  droplet_id = "${digitalocean_droplet.foobar.id}"
  region     = "${digitalocean_droplet.foobar.region}"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region that the Floating IP is reserved to.
* `droplet_id` - (Optional) The ID of Droplet that the Floating IP will be assigned to.

~> **NOTE:** A Floating IP can be assigned to a region OR a droplet_id. If both region AND droplet_id are specified, then the Floating IP will be assigned to the droplet and use that region

## Attributes Reference

The following attributes are exported:

* `ip_address` - The IP Address of the resource

## Import

Floating IPs can be imported using the `ip`, e.g.

```
terraform import digitalocean_floating_ip.myip 192.168.0.1
```