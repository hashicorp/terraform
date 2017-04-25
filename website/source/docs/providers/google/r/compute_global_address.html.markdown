---
layout: "google"
page_title: "Google: google_compute_global_address"
sidebar_current: "docs-google-compute-global-address"
description: |-
  Creates a static global IP address resource for a Google Compute Engine project.
---

# google\_compute\_global\_address

Creates a static IP address resource global to a Google Compute Engine project. For more information see
[the official documentation](https://cloud.google.com/compute/docs/instances-and-network) and
[API](https://cloud.google.com/compute/docs/reference/latest/globalAddresses).


## Example Usage

```hcl
resource "google_compute_global_address" "default" {
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

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `address` - The assigned address.

* `self_link` - The URI of the created resource.
