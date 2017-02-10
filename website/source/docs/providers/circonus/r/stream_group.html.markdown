---
layout: "circonus"
page_title: "Circonus: circonus_stream_group"
sidebar_current: "docs-circonus-resource-circonus_stream_group"
description: |-
  Manages a Circonus Stream Group.
---

# circonus\_stream\_group

The ``circonus_stream_group`` resource creates and manages a
[Circonus Stream Group](https://login.circonus.com/user/docs/Data/View/MetricClusters).

~> **NOTE regarding `cirocnus_stream_group`:** The `circonus_stream_group`
resource is a renamed Circonus ["metric
cluster"](https://login.circonus.com/resources/api/calls/metric_cluster).  A
"stream group" is a collection of metric streams which is more accurate than a
"metric cluster" which indicates locality and clustering of data points within a
stream.

## Usage

```
resource "circonus_stream_group" "nomad-job-memory-rss" {
  name = "My Job's Resident Memory"
  description = <<-EOF
An aggregation of all resident memory metric streams for a Nomad job.
EOF

  group {
    query = "*`nomad-jobname`memory`rss"
    type = "average"
  }
  tags = ["source:nomad","resource:memory"]
}
```

## Argument Reference

* `description` - (Optional) A long-form description of the stream group.

* `name` - (Required) The name of the stream group.  This name must be unique
  across all stream groups in a given Circonus Account.

* `group` - (Required) One or more `group` attributes must be present.  Each
  `group` must contain both a `query` and a `type`.  See below for details on
  supported attributes.

* `tags` - (Optional) A list of tags attached to the stream group.

## Supported Stream Group `group` Attributes

* `query` - (Required) A stream group [query](https://login.circonus.com/resources/api/calls/metric_cluster).

* `type` - (Required) The query type to execute per stream group.  Valid query
  types are: `average`, `count`, `counter`, `counter2`, `counter2_stddev`,
  `counter_stddev`, `derive`, `derive2`, `derive2_stddev`, `derive_stddev`,
  `histogram`, `stddev`, `text`.

## Out parameters

* `id` - ID of the Stream Group

## Import Example

`circonus_stream_group` supports importing resources.  Supposing the following
Terraform:

```
provider "circonus" {
  alias = "b8fec159-f9e5-4fe6-ad2c-dc1ec6751586"
}

resource "circonus_stream_group" "mystreamgroup" {
  name = "Stream group for a particular metric in a job"

  group {
    query = "*`nomad-jobname`memory`rss"
    type = "average"
  }
}
```

It is possible to import a `circonus_stream_group` resource with the following
command:

```
$ terraform import circonus_stream_group.mystreamgroup ID
```

Where `ID` is the `_cid` or Circonus ID of the Stream Group
(e.g. `/metric_cluster/12345`) and `circonus_stream_group.mystreamgroup` is the
name of the resource whose state will be populated as a result of the command.
