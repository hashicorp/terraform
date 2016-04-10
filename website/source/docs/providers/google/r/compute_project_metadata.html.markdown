---
layout: "google"
page_title: "Google: google_compute_project_metadata"
sidebar_current: "docs-google-compute-project-metadata"
description: |-
  Manages common instance metadata
---

# google\_compute\_project\_metadata

Manages metadata common to all instances for a project in GCE.

## Example Usage

```js
resource "google_compute_project_metadata" "default" {
  metadata {
    foo  = "bar"
    fizz = "buzz"
    13   = "42"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) A series of key value pairs. Changing this resource
    updates the GCE state.

- - -

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

Only the arguments listed above are exposed as attributes.
