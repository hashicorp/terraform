---
layout: "google"
page_title: "Google: google_compute_subnetwork"
sidebar_current: "docs-google-compute-subnetwork"
description: |-
  Manages a subnetwork within GCE.
---

# google\_compute\_subnetwork

Manages a subnetwork within GCE.

## Example Usage

```
resource "google_compute_subnetwork" "default-us-east1" {
	name = "default-us-east1"
	ip_cidr_range = "10.0.0.0/16"
	network = "${google_compute_network.default.self_link}"
	region = "us-east1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `network` - (Required) A link to the parent network of this subnetwork.
     The parent network must have been created in custom subnet mode.

* `ip_cidr_range` - (Required) The IP address range that machines in this
     network are assigned to, represented as a CIDR block.
     
* `region` - (Required) The region this subnetwork will be created in. 

* `description` - (Optional) Description of this subnetwork.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `ip_cidr_range` - The CIDR block of this network.
* `gateway_address` - The IP address of the gateway.
