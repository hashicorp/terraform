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

```hcl
resource "google_compute_network" "foobar" {
  name       = "test"
  ipv4_range = "10.0.0.0/16"
}

resource "google_compute_route" "foobar" {
  name        = "test"
  dest_range  = "15.0.0.0/24"
  network     = "${google_compute_network.foobar.name}"
  next_hop_ip = "10.0.1.5"
  priority    = 100
}
```

## Argument Reference

The following arguments are supported:

* `dest_range` - (Required) The destination IPv4 address range that this
    route applies to.

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `network` - (Required) The name or self_link of the network to attach this route to.

* `priority` - (Required) The priority of this route, used to break ties.

- - -

* `next_hop_gateway` - (Optional) The URL of the internet gateway to route
    to if this route is matched. The alias "default-internet-gateway" can also
    be used.

* `next_hop_instance` - (Optional) The name of the VM instance to route to
    if this route is matched.

* `next_hop_instance_zone` - (Required when `next_hop_instance` is specified)
    The zone of the instance specified in `next_hop_instance`.

* `next_hop_ip` - (Optional) The IP address of the next hop if this route
    is matched.

* `next_hop_vpn_tunnel` - (Optional) The name of the VPN to route to if this
    route is matched.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `tags` - (Optional) The tags that this route applies to.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `next_hop_network` - The name of the next hop network, if available.

* `self_link` - The URI of the created resource.
