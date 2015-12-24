---
layout: "google"
page_title: "Google: google_compute_address"
sidebar_current: "docs-google-compute-address"
description: |-
  Creates a static IP address resource for Google Compute Engine.
---

# google\_compute\_address

Creates a static IP address resource for Google Compute Engine.  For more information see
[the official documentation](https://cloud.google.com/compute/docs/instances-and-network) and
[API](https://cloud.google.com/compute/docs/reference/latest/addresses).


## Example Usage

```
resource "google_compute_address" "default" {
	name = "test-address"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.
* `region` - (Optional) The Region in which the created address should reside. 
    If it is not provided, the provider region is used. 

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `address` - The IP address that was allocated.
* `self_link` - The URI of the created resource.
* `region` - The Region in which the created address does reside.
