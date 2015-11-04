---
layout: "google"
page_title: "Google: google_bigquery_table"
sidebar_current: "docs-google-bigquery-table"
description: |-
  Manages a bigquery table
---

# google\_bigquery\_table

Manages a bigquery table

## Example Usage

```
resource "google_bigquery_table" "default" {
	name = "test"
	datasetId = "dataset_test"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `datasetId` - (Required) The name of a dataset that this table will
    be created in.  Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `datasetId` - The name of the containing dataset.
