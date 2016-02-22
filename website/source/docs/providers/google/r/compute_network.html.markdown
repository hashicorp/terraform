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

* `ipv4_range` - (Optional) The IPv4 address range that machines in this
     network are assigned to, represented as a CIDR block. If not
     set, an auto or custom subnetted network will be created, depending
     on the value of `auto_create_subnetworks` attribute. This attribute
     may not be used if `auto_create_subnets` is specified.
     
* `auto_create_subnetworks` - (Optional) If set to true, this network 
     will be created in auto subnet mode, and Google will create a
     subnet for each region automatically.
     If set to false, and `ipv4_range` is not set, a custom subnetted
     network will be created that can support `google_compute_subnetwork`
     resources. This attribute may not be used if `ipv4_range` is specified.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `ipv4_range` - The CIDR block of this network.
* `gateway_ipv4` - The IPv4 address of the gateway.
