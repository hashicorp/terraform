---
layout: "google"
page_title: "Google: google_compute_network"
sidebar_current: "docs-google-datasource-compute-network"
description: |-
  Get a network within GCE.
---

# google\_compute\_network

Get a network within GCE from its name.

## Example Usage

```tf
data "google_compute_network" "my-network" {
  name = "default-us-east1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the network.
    

- - -

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `network` - The network name or resource link to the parent
    network of this network. 

* `description` - Description of this network.

* `gateway_ipv4` - The IP address of the gateway.

* `subnetworks_self_links` - the list of subnetworks which belong to the network

* `self_link` - The URI of the resource.
