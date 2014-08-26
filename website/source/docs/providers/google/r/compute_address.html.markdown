---
layout: "google"
page_title: "Google: google_compute_address"
sidebar_current: "docs-google-resource-address"
---

# google\_compute\_address

Creates a static IP address resource for Google Compute Engine.

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

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `address` - The IP address that was allocated.
