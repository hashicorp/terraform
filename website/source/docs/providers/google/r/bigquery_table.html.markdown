---
layout: "google"
page_title: "Google: google_bigquery_table"
sidebar_current: "docs-google-bigquery-table"
description: |-
  Creates a table resource in a dataset for Google BigQuery.
---

# google_bigquery_table

Creates a table resource in a dataset for Google BigQuery. For more information see
[the official documentation](https://cloud.google.com/bigquery/docs/) and
[API](https://cloud.google.com/bigquery/docs/reference/rest/v2/tables).


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

resource "google_bigquery_table" "default" {
  dataset_id = "${google_bigquery_dataset.default.id}"
  table_id   = "test"

  time_partitioning {
    type = "DAY"
  }

  labels {
    env = "default"
  }

  schema = "${file("schema.json")}"
}
```

## Argument Reference

The following arguments are supported:

* `dataset_id` - (Required) The dataset ID to create the table in.
    Changing this forces a new resource to be created.

* `table_id` - (Required) A unique ID for the resource.
    Changing this forces a new resource to be created.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `description` - (Optional) The field description.

* `expiration_time` - (Optional) The time when this table expires, in
    milliseconds since the epoch. If not present, the table will persist
    indefinitely. Expired tables will be deleted and their storage
    reclaimed.

* `friendly_name` - (Optional) A descriptive name for the table.

* `labels` - (Optional) A mapping of labels to assign to the resource.

* `schema` - (Optional) A JSON schema for the table.

* `time_partitioning` - (Optional) If specified, configures time-based
    partitioning for this table. Structure is documented below.

The `time_partitioning` block supports:

* `expiration_ms` -  (Optional) Number of milliseconds for which to keep the
    storage for a partition.

* `type` - (Required) The only type supported is DAY, which will generate
    one partition per day based on data loading time.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `creation_time` - The time when this table was created, in milliseconds since the epoch.

* `etag` - A hash of the resource.

* `last_modified_time` - The time when this table was last modified, in milliseconds since the epoch.

* `location` - The geographic location where the table resides. This value is inherited from the dataset.

* `num_bytes` - The size of this table in bytes, excluding any data in the streaming buffer.

* `num_long_term_bytes` - The number of bytes in the table that are considered "long-term storage".

* `num_rows` - The number of rows of data in this table, excluding any data in the streaming buffer.

* `self_link` - The URI of the created resource.

* `type` - Describes the table type.

## Import

Tables can be imported using ID of the table (`projectID`:`datasetID`.`tableID`), e.g.

```
$ terraform import bigquery_table.default testproject:testdataset.testtable
```
