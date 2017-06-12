---
layout: "google"
page_title: "Google: google_compute_subnetwork"
sidebar_current: "docs-google-compute-subnetwork"
description: |-
  Manages a subnetwork within GCE.
---

# google\_compute\_subnetwork

Manages a subnetwork within GCE. For more information see 
[the official documentation](https://cloud.google.com/compute/docs/vpc/#vpc_networks_and_subnets)
and 
[API](https://cloud.google.com/compute/docs/reference/latest/subnetworks).

## Example Usage

```hcl
resource "google_compute_subnetwork" "default-us-east1" {
  name          = "default-us-east1"
  ip_cidr_range = "10.0.0.0/16"
  network       = "${google_compute_network.default.self_link}"
  region        = "us-east1"
}
```

## Argument Reference

The following arguments are supported:

* `ip_cidr_range` - (Required) The IP address range that machines in this
    network are assigned to, represented as a CIDR block.

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `network` - (Required) The network name or resource link to the parent
    network of this subnetwork. The parent network must have been created
    in custom subnet mode.

- - -

* `description` - (Optional) Description of this subnetwork.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `region` - (Optional) The region this subnetwork will be created in. If
    unspecified, this defaults to the region configured in the provider.

* `private_ip_google_access` - (Optional) Whether the VMs in this subnet
    can access Google services without assigned external IP
    addresses.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `gateway_address` - The IP address of the gateway.

* `self_link` - The URI of the created resource.
