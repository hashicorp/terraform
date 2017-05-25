---
layout: "google"
page_title: "Google: google_compute_router"
sidebar_current: "docs-google-compute-router"
description: |-
  Manages a Cloud Router resource.
---

# google\_compute\_router

Manages a Cloud Router resource. For more info, read the
[documentation](https://cloud.google.com/compute/docs/cloudrouter).

## Example Usage

```hcl
resource "google_compute_network" "foobar" {
  name = "network-1"
}

resource "google_compute_subnetwork" "foobar" {
  name          = "subnet-1"
  network       = "${google_compute_network.foobar.self_link}"
  ip_cidr_range = "10.0.0.0/16"
  region        = "us-central1"
}

resource "google_compute_address" "foobar" {
  name   = "vpn-gateway-1-address"
  region = "${google_compute_subnetwork.foobar.region}"
}

resource "google_compute_vpn_gateway" "foobar" {
  name    = "vpn-gateway-1"
  network = "${google_compute_network.foobar.self_link}"
  region  = "${google_compute_subnetwork.foobar.region}"
}

resource "google_compute_forwarding_rule" "foobar_esp" {
  name        = "vpn-gw-1-esp"
  region      = "${google_compute_vpn_gateway.foobar.region}"
  ip_protocol = "ESP"
  ip_address  = "${google_compute_address.foobar.address}"
  target      = "${google_compute_vpn_gateway.foobar.self_link}"
}

resource "google_compute_forwarding_rule" "foobar_udp500" {
  name        = "vpn-gw-1-udp-500"
  region      = "${google_compute_forwarding_rule.foobar_esp.region}"
  ip_protocol = "UDP"
  port_range  = "500-500"
  ip_address  = "${google_compute_address.foobar.address}"
  target      = "${google_compute_vpn_gateway.foobar.self_link}"
}

resource "google_compute_forwarding_rule" "foobar_udp4500" {
  name        = "vpn-gw-1-udp-4500"
  region      = "${google_compute_forwarding_rule.foobar_udp500.region}"
  ip_protocol = "UDP"
  port_range  = "4500-4500"
  ip_address  = "${google_compute_address.foobar.address}"
  target      = "${google_compute_vpn_gateway.foobar.self_link}"
}

resource "google_compute_router" "foobar" {
  name    = "router-1"
  region  = "${google_compute_forwarding_rule.foobar_udp500.region}"
  network = "${google_compute_network.foobar.self_link}"

  bgp {
    asn = 64512
  }
}

resource "google_compute_vpn_tunnel" "foobar" {
  name               = "vpn-tunnel-1"
  region             = "${google_compute_forwarding_rule.foobar_udp4500.region}"
  target_vpn_gateway = "${google_compute_vpn_gateway.foobar.self_link}"
  shared_secret      = "unguessable"
  peer_ip            = "8.8.8.8"
  router             = "${google_compute_router.foobar.name}"
}

resource "google_compute_router_interface" "foobar" {
  name       = "interface-1"
  router     = "${google_compute_router.foobar.name}"
  region     = "${google_compute_router.foobar.region}"
  ip_range   = "169.254.1.1/30"
  vpn_tunnel = "${google_compute_vpn_tunnel.foobar.name}"
}

resource "google_compute_router_peer" "foobar" {
  name                      = "peer-1"
  router                    = "${google_compute_router.foobar.name}"
  region                    = "${google_compute_router.foobar.region}"
  peer_ip_address           = "169.254.1.2"
  peer_asn                  = 65513
  advertised_route_priority = 100
  interface                 = "${google_compute_router_interface.foobar.name}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the router, required by GCE. Changing
    this forces a new router to be created.

* `network` - (Required) The name or resource link to the network this Cloud Router
    will use to learn and announce routes. Changing this forces a new router to be created.

* `bgp` - (Required) BGP information specific to this router.
    Changing this forces a new router to be created.
    Structure is documented below.

- - -

* `description` - (Optional) A description of the resource.
    Changing this forces a new router to be created.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.
    Changing this forces a new router to be created.

* `region` - (Optional) The region this router should sit in. If not specified,
    the project region will be used. Changing this forces a new router to be
    created.

- - -

The `bgp` block supports:

* `asn` - (Required) Local BGP Autonomous System Number (ASN). Must be an
  RFC6996 private ASN.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.

## Import

Routers can be imported using the `region` and `name`, e.g.

```
$ terraform import google_compute_router.router-1 us-central1/router-1
```

