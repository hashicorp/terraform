---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_firewall"
sidebar_current: "docs-do-resource-firewall"
description: |-
  Provides a DigitalOcean Cloud Firewall resource. This can be used to create, modify, and delete Firewalls.
---

# digitalocean\_firewall

Provides a DigitalOcean Cloud Firewall resource. This can be used to create,
modify, and delete Firewalls.

## Example Usage

```hcl
resource "digitalocean_droplet" "web" {
  name      = "web-1"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
}

resource "digitalocean_firewall" "web" {
  name = "only-22-80-and-443"

  droplet_ids = ["${digitalocean_droplet.web.id}"]

  inbound_rules = [
    {
      protocol           = "tcp"
      port_range         = "22"
      source_addresses   = ["192.168.1.0/24", "2002:1:2::/48"]
    },
    {
      protocol           = "tcp"
      port_range         = "80"
      source_addresses   = ["0.0.0.0/0", "::/0"]
    },
    {
      protocol           = "tcp"
      port_range         = "443"
      source_addresses   = ["0.0.0.0/0", "::/0"]
    },
  ]

  outbound_rules = [
    {
      protocol                = "udp"
      port_range              = "53"
      destination_addresses   = ["0.0.0.0/0", "::/0"]
    },
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The Firewall name
* `droplet_ids` (Optional) - The list of the IDs of the Droplets assigned
  to the Firewall.
* `tags` (Optional) - The names of the Tags assigned to the Firewall.
* `inbound_rules` - (Optional) The inbound access rule block for the Firewall.
  The `inbound_rules` block is documented below.
* `outbound_rules` - (Optional) The outbound access rule block for the Firewall.
  The `outbound_rules` block is documented below.

`inbound_rule` supports the following:

* `protocol` - (Optional) The type of traffic to be allowed.
  This may be one of "tcp", "udp", or "icmp".
* `port_range` - (Optional) The ports on which traffic will be allowed
  specified as a string containing a single port, a range (e.g. "8000-9000"),
  or "all" to open all ports for a protocol.
* `source_addresses` - (Optional) An array of strings containing the IPv4
  addresses, IPv6 addresses, IPv4 CIDRs, and/or IPv6 CIDRs from which the
  inbound traffic will be accepted.
* `source_droplet_ids` - (Optional) An array containing the IDs of
  the Droplets from which the inbound traffic will be accepted.
* `source_tags` - (Optional) An array containing the names of Tags
  corresponding to groups of Droplets from which the inbound traffic
  will be accepted.
* `source_load_balancer_uids` - (Optional) An array containing the IDs
  of the Load Balancers from which the inbound traffic will be accepted.

`outbound_rule` supports the following:

* `protocol` - (Optional) The type of traffic to be allowed.
  This may be one of "tcp", "udp", or "icmp".
* `port_range` - (Optional) The ports on which traffic will be allowed
  specified as a string containing a single port, a range (e.g. "8000-9000"),
  or "all" to open all ports for a protocol.
* `destination_addresses` - (Optional) An array of strings containing the IPv4
  addresses, IPv6 addresses, IPv4 CIDRs, and/or IPv6 CIDRs to which the
  outbound traffic will be allowed.
* `destination_droplet_ids` - (Optional) An array containing the IDs of
  the Droplets to which the outbound traffic will be allowed.
* `destination_tags` - (Optional) An array containing the names of Tags
  corresponding to groups of Droplets to which the outbound traffic will
  be allowed.
  traffic.
* `destination_load_balancer_uids` - (Optional) An array containing the IDs
  of the Load Balancers to which the outbound traffic will be allowed.


## Attributes Reference

The following attributes are exported:

* `id` - A unique ID that can be used to identify and reference a Firewall.
* `status` - A status string indicating the current state of the Firewall.
  This can be "waiting", "succeeded", or "failed".
* `created_at` - A time value given in ISO8601 combined date and time format
  that represents when the Firewall was created.
* `pending_changes` - An list of object containing the fields, "droplet_id",
  "removing", and "status".  It is provided to detail exactly which Droplets
  are having their security policies updated.  When empty, all changes
  have been successfully applied.
* `name` - The name of the Firewall.
* `droplet_ids` - The list of the IDs of the Droplets assigned to
  the Firewall.
* `tags` - The names of the Tags assigned to the Firewall.
* `inbound_rules` - The inbound access rule block for the Firewall.
* `outbound_rules` - The outbound access rule block for the Firewall.

## Import

Firewalls can be imported using the firewall `id`, e.g.

```
terraform import digitalocean_firewall.myfirewall b8ecd2ab-2267-4a5e-8692-cbf1d32583e3
```
