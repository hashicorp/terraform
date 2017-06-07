---
layout: "google"
page_title: "Google: google_compute_subnetwork"
sidebar_current: "docs-google-datasource-compute-subnetwork"
description: |-
  Get a subnetwork within GCE.
---

# google\_compute\_subnetwork

Get a subnetwork within GCE from its name and region.

## Example Usage

```tf
data "google_compute_subnetwork" "my-subnetwork" {
  name   = "default-us-east1"
  region = "us-east1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - The name of the subnetwork.

- - -

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `region` - (Optional) The region this subnetwork has been created in. If
    unspecified, this defaults to the region configured in the provider.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `network` - The network name or resource link to the parent
    network of this subnetwork.

* `description` - Description of this subnetwork.

* `ip_cidr_range` - The IP address range that machines in this
    network are assigned to, represented as a CIDR block.

* `gateway_address` - The IP address of the gateway.

* `private_ip_google_access` - Whether the VMs in this subnet
    can access Google services without assigned external IP
    addresses.

* `self_link` - The URI of the created resource.
