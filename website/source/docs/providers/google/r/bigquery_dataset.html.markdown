---
layout: "google"
page_title: "Google: google_bigquery_dataset"
sidebar_current: "docs-google-bigquery-dataset"
description: |-
  Manages a bigquery dataset
---

# google\_bigquery\_dataset

Manages a bigquery dataset

## Example Usage

```
resource "google_bigquery_dataset" "default" {
	name = "test"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
