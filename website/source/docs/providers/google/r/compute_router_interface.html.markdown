---
layout: "google"
page_title: "Google: google_compute_router_interface"
sidebar_current: "docs-google-compute-router-interface"
description: |-
  Manages a Cloud Router interface.
---

# google\_compute\_router_interface

Manages a Cloud Router interface. For more info, read the
[documentation](https://cloud.google.com/compute/docs/cloudrouter).

## Example Usage

```hcl
resource "google_compute_router_interface" "foobar" {
  name       = "interface-1"
  router     = "router-1"
  region     = "us-central1"
  ip_range   = "169.254.1.1/30"
  vpn_tunnel = "tunnel-1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the interface, required by GCE. Changing
    this forces a new interface to be created.

* `router` - (Required) The name of the router this interface will be attached to.
    Changing this forces a new interface to be created.

* `vpn_tunnel` - (Required) The name or resource link to the VPN tunnel this
    interface will be linked to. Changing this forces a new interface to be created.

- - -

* `ip_range` - (Optional) IP address and range of the interface. The IP range must be 
    in the RFC3927 link-local IP space. Changing this forces a new interface to be created.

* `project` - (Optional) The project in which this interface's router belongs. If it
    is not provided, the provider project is used. Changing this forces a new interface to be created.

* `region` - (Optional) The region this interface's router sits in. If not specified,
    the project region will be used. Changing this forces a new interface to be
    created.

## Attributes Reference

Only the arguments listed above are exposed as attributes.

## Import

Router interfaces can be imported using the `region`, `router` and `name`, e.g.

```
$ terraform import google_compute_router_interface.interface-1 us-central1/router-1/interface-1
```

