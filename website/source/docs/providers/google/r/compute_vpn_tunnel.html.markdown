---
layout: "google"
page_title: "Google: google_compute_vpn_tunnel"
sidebar_current: "docs-google-compute-vpn-tunnel"
description: |-
  Manages a VPN Tunnel to the GCE network
---

# google\_compute\_vpn\_tunnel

Manages a VPN Tunnel to the GCE network. For more info, read the 
[documentation](https://cloud.google.com/compute/docs/vpn).

## Example Usage

```
resource "google_compute_network" "network1" {
    name = "network1"
    ipv4_range = "10.120.0.0/16"
}

resource "google_compute_vpn_gateway" "target_gateway" {
    name = "vpn1"
    network = "${google_compute_network.network1.self_link}"
    region = "${var.region}"
}

resource "google_compute_address" "vpn_static_ip" {
    name = "vpn-static-ip"
    region = "${var.region}"
}

resource "google_compute_forwarding_rule" "fr_esp" {
    name = "fr-esp"
    region = "${var.region}"
    ip_protocol = "ESP"
    ip_address = "${google_compute_address.vpn_static_ip.address}"
    target = "${google_compute_vpn_gateway.target_gateway.self_link}"
}

resource "google_compute_forwarding_rule" "fr_udp500" {
    name = "fr-udp500"
    region = "${var.region}"
    ip_protocol = "UDP"
    port_range = "500"
    ip_address = "${google_compute_address.vpn_static_ip.address}"
    target = "${google_compute_vpn_gateway.target_gateway.self_link}"
}

resource "google_compute_forwarding_rule" "fr_udp4500" {
    name = "fr-udp4500"
    region = "${var.region}"
    ip_protocol = "UDP"
    port_range = "4500"
    ip_address = "${google_compute_address.vpn_static_ip.address}"
    target = "${google_compute_vpn_gateway.target_gateway.self_link}"
}

resource "google_compute_vpn_tunnel" "tunnel1" {
    name = "tunnel1"
    region = "${var.region}"
    peer_ip = "15.0.0.120"
    shared_secret = "a secret message"
    target_vpn_gateway = "${google_compute_vpn_gateway.target_gateway.self_link}"
    depends_on = ["google_compute_forwarding_rule.fr_esp",
        "google_compute_forwarding_rule.fr_udp500",
        "google_compute_forwarding_rule.fr_udp4500"]
}

resource "google_compute_route" "route1" {
    name = "route1"
    network = "${google_compute_network.network1.name}"
    next_hop_vpn_tunnel = "${google_compute_vpn_tunnel.tunnel1.self_link}"
    dest_range = "15.0.0.0/24"
    priority = 1000
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `description` - (Optional) A description of the resource.
    Changing this forces a new resource to be created.

* `peer_ip` - (Required) The VPN gateway sitting outside of GCE.
    Changing this forces a new resource to be created.

* `region` - (Optional) The region this tunnel should sit in. If not specified,
    the project region will be used. Changing this forces a new resource to be 
    created.

* `shared_secret` - (Required) A passphrase shared between the two VPN gateways.
    Changing this forces a new resource to be created.

* `target_vpn_gateway` - (Required) A link to the VPN gateway sitting inside GCE.
    Changing this forces a new resource to be created.

* `ike_version` - (Optional) Either version 1 or 2. Default is 2.
    Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `self_link` - A GCE server assigned link to this resource.

* `detailed_status` - Information about the status of the VPN tunnel. 
