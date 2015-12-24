---
layout: "google"
page_title: "Google: google_compute_route"
sidebar_current: "docs-google-compute-route"
description: |-
  Manages a network route within GCE.
---

# google\_compute\_route

Manages a network route within GCE.

## Example Usage

```
resource "google_compute_network" "foobar" {
	name = "test"
	ipv4_range = "10.0.0.0/16"
}

resource "google_compute_route" "foobar" {
	name = "test"
	dest_range = "15.0.0.0/24"
	network = "${google_compute_network.foobar.name}"
	next_hop_ip = "10.0.1.5"
	priority = 100
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `dest_range` - (Required) The destination IPv4 address range that this
     route applies to.

* `network` - (Required) The name of the network to attach this route to.

* `next_hop_ip` - (Optional) The IP address of the next hop if this route
    is matched.

* `next_hop_instance` - (Optional) The name of the VM instance to route to
    if this route is matched.

* `next_hop_instance_zone` - (Optional) The zone of the instance specified
    in `next_hop_instance`.

* `next_hop_gateway` - (Optional) The name of the internet gateway to route
    to if this route is matched.

* `next_hop_network` - (Optional) The name of the network to route to if this
    route is matched.

* `next_hop_vpn_gateway` - (Optional) The name of the VPN to route to if this
    route is matched.
    
* `priority` - (Required) The priority of this route, used to break ties.

* `tags` - (Optional) The tags that this route applies to.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `dest_range` - The destination CIDR block of this route.
* `network` - The name of the network of this route.
* `next_hop_ip` - The IP address of the next hop, if available.
* `next_hop_instance` - The name of the instance of the next hop, if available.
* `next_hop_instance_zone` - The zone of the next hop instance, if available.
* `next_hop_gateway` - The name of the next hop gateway, if available.
* `next_hop_network` - The name of the next hop network, if available.
* `priority` - The priority of this route.
* `tags` - The tags this route applies to.
