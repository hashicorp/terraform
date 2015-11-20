---
layout: "google"
page_title: "Google: google_compute_network"
sidebar_current: "docs-google-compute-network"
description: |-
  Manages a network within GCE.
---

# google\_compute\_network

Manages a network within GCE.

## Example Usage

```
resource "google_compute_network" "default" {
	name = "test"
	ipv4_range = "10.0.0.0/16"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `ipv4_range` - (Required) The IPv4 address range that machines in this
     network are assigned to, represented as a CIDR block.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `ipv4_range` - The CIDR block of this network.
* `gateway_ipv4` - The IPv4 address of the gateway.
