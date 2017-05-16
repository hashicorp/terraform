---
layout: "circonus"
page_title: "Circonus: circonus_metric_cluster"
sidebar_current: "docs-circonus-resource-circonus_metric_cluster"
description: |-
  Manages a Circonus Metric Cluster.
---

# circonus\_metric\_cluster

The ``circonus_metric_cluster`` resource creates and manages a
[Circonus Metric Cluster](https://login.circonus.com/user/docs/Data/View/MetricClusters).

## Usage

```hcl
resource "circonus_metric_cluster" "nomad-job-memory-rss" {
  name = "My Job's Resident Memory"
  description = <<-EOF
An aggregation of all resident memory metric streams across allocations in a Nomad job.
EOF

  query {
    definition = "*`nomad-jobname`memory`rss"
    type       = "average"
  }
  tags = ["source:nomad","resource:memory"]
}
```

## Argument Reference

* `description` - (Optional) A long-form description of the metric cluster.

* `name` - (Required) The name of the metric cluster.  This name must be unique
  across all metric clusters in a given Circonus Account.

* `query` - (Required) One or more `query` attributes must be present.  Each
  `query` must contain both a `definition` and a `type`.  See below for details
  on supported attributes.

* `tags` - (Optional) A list of tags attached to the metric cluster.

## Supported Metric Cluster `query` Attributes

* `definition` - (Required) The definition of a metric cluster [query](https://login.circonus.com/resources/api/calls/metric_cluster).

* `type` - (Required) The query type to execute per metric cluster.  Valid query
  types are: `average`, `count`, `counter`, `counter2`, `counter2_stddev`,
  `counter_stddev`, `derive`, `derive2`, `derive2_stddev`, `derive_stddev`,
  `histogram`, `stddev`, `text`.

## Out parameters

* `id` - ID of the Metric Cluster.

## Import Example

`circonus_metric_cluster` supports importing resources.  Supposing the following
Terraform:

```hcl
provider "circonus" {
  alias = "b8fec159-f9e5-4fe6-ad2c-dc1ec6751586"
}

resource "circonus_metric_cluster" "mymetriccluster" {
  name = "Metric Cluster for a particular metric in a job"

  query {
    definition = "*`nomad-jobname`memory`rss"
    type       = "average"
  }
}
```

It is possible to import a `circonus_metric_cluster` resource with the following
command:

```
$ terraform import circonus_metric_cluster.mymetriccluster ID
```

Where `ID` is the `_cid` or Circonus ID of the Metric Cluster
(e.g. `/metric_cluster/12345`) and `circonus_metric_cluster.mymetriccluster` is the
name of the resource whose state will be populated as a result of the command.
