---
layout: "google"
page_title: "Google: google_compute_router_peer"
sidebar_current: "docs-google-compute-router-peer"
description: |-
  Manages a Cloud Router BGP peer.
---

# google\_compute\_router

Manages a Cloud Router BGP peer. For more info, read the
[documentation](https://cloud.google.com/compute/docs/cloudrouter).

## Example Usage

```hcl
resource "google_compute_router_peer" "foobar" {
  name                      = "peer-1"
  router                    = "router-1"
  region                    = "us-central1"
  peer_ip_address           = "169.254.1.2"
  peer_asn                  = 65513
  advertised_route_priority = 100
  interface                 = "interface-1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for BGP peer, required by GCE. Changing
    this forces a new peer to be created.

* `router` - (Required) The name of the router in which this BGP peer will be configured.
    Changing this forces a new peer to be created.

* `interface` - (Required) The name of the interface the BGP peer is associated with.
    Changing this forces a new peer to be created.

* `peer_ip_address` - (Required) IP address of the BGP interface outside Google Cloud.
    Changing this forces a new peer to be created.

* `peer_asn` - (Required) Peer BGP Autonomous System Number (ASN).
    Changing this forces a new peer to be created.

- - -

* `advertised_route_priority` - (Optional) The priority of routes advertised to this BGP peer.
    Changing this forces a new peer to be created.

* `project` - (Optional) The project in which this peer's router belongs. If it
    is not provided, the provider project is used. Changing this forces a new peer to be created.

* `region` - (Optional) The region this peer's router sits in. If not specified,
    the project region will be used. Changing this forces a new peer to be
    created.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `ip_address` - IP address of the interface inside Google Cloud Platform.

## Import

Router BGP peers can be imported using the `region`, `router` and `name`, e.g.

```
$ terraform import google_compute_router_peer.peer-1 us-central1/router-1/peer-1
```
