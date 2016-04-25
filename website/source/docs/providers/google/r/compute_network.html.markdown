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

```js
resource "google_compute_network" "default" {
  name       = "test"
  ipv4_range = "10.0.0.0/16"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

- - -

* `auto_create_subnetworks` - (Optional) If set to true, this network will be
    created in auto subnet mode, and Google will create a subnet for each region
    automatically. If set to false, and `ipv4_range` is not set, a custom
    subnetted network will be created that can support
    `google_compute_subnetwork` resources. This attribute may not be used if
    `ipv4_range` is specified.

* `description` - (Optional) A brief description of this resource.

* `ipv4_range` - (Optional) The IPv4 address range that machines in this network
    are assigned to, represented as a CIDR block. If not set, an auto or custom
    subnetted network will be created, depending on the value of  
    `auto_create_subnetworks` attribute. This attribute may not be used if
    `auto_create_subnets` is specified. This attribute is deprecated.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `gateway_ipv4` - The IPv4 address of the gateway.

* `self_link` - The URI of the created resource.
