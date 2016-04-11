---
layout: "google"
page_title: "Google: google_compute_address"
sidebar_current: "docs-google-compute-address"
description: |-
  Creates a static IP address resource for Google Compute Engine.
---

# google\_compute\_address

Creates a static IP address resource for Google Compute Engine. For more information see
[the official documentation](https://cloud.google.com/compute/docs/instances-and-network) and
[API](https://cloud.google.com/compute/docs/reference/latest/addresses).


## Example Usage

```js
resource "google_compute_address" "default" {
  name = "test-address"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

- - -

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `region` - (Optional) The Region in which the created address should reside.
    If it is not provided, the provider region is used.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.
