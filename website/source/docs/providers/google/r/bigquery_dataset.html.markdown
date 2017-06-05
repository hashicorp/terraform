---
layout: "google"
page_title: "Google: google_bigquery_dataset"
sidebar_current: "docs-google-bigquery-dataset"
description: |-
  Creates a dataset resource for Google BigQuery.
---

# google_bigquery_dataset

Creates a dataset resource for Google BigQuery. For more information see
[the official documentation](https://cloud.google.com/bigquery/docs/) and
[API](https://cloud.google.com/bigquery/docs/reference/rest/v2/datasets).


## Example Usage

```hcl
resource "google_bigquery_dataset" "default" {
  dataset_id                  = "test"
  friendly_name               = "test"
  description                 = "This is a test description"
  location                    = "EU"
  default_table_expiration_ms = 3600000

  labels {
    env = "default"
  }
}
```

## Argument Reference

The following arguments are supported:

* `dataset_id` - (Required) A unique ID for the resource.
    Changing this forces a new resource to be created.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `friendly_name` - (Optional) A descriptive name for the dataset.

* `description` - (Optional) A user-friendly description of the dataset.

* `location` - (Optional) The geographic location where the dataset should reside.

    Possible values include `EU` and `US`. The default value is `US`.

    Changing this forces a new resource to be created.

* `default_table_expiration_ms` - (Optional) The default lifetime of all
    tables in the dataset, in milliseconds. The minimum value is 3600000
    milliseconds (one hour).

    Once this property is set, all newly-created
    tables in the dataset will have an expirationTime property set to the
    creation time plus the value in this property, and changing the value
    will only affect new tables, not existing ones. When the
    expirationTime for a given table is reached, that table will be
    deleted automatically. If a table's expirationTime is modified or
    removed before the table expires, or if you provide an explicit
    expirationTime when creating a table, that value takes precedence
    over the default expiration time indicated by this property.

  * `labels` - (Optional) A mapping of labels to assign to the resource.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.

* `etag` - A hash of the resource.

* `creation_time` - The time when this dataset was created, in milliseconds since the epoch.

* `last_modified_time` -  The date when this dataset or any of its tables was last modified,
  in milliseconds since the epoch.
