---
layout: "circonus"
page_title: "Circonus: circonus_graph"
sidebar_current: "docs-circonus-resource-circonus_graph"
description: |-
  Manages a Circonus graph.
---

# circonus\_graph

The ``circonus_graph`` resource creates and manages a
[Circonus Graph](https://login.circonus.com/user/docs/Visualization/Graph/Create).

https://login.circonus.com/resources/api/calls/graph).

## Usage

```hcl
variable "myapp-tags" {
  type    = "list"
  default = [ "app:myapp", "owner:myteam" ]
}

resource "circonus_graph" "latency-graph" {
  name        = "Latency Graph"
  description = "A sample graph showing off two data points"
  notes       = "Misc notes about this graph"
  graph_style = "line"
  line_style  = "stepped"

  metric {
    check       = "${circonus_check.api_latency.checks[0]}"
    metric_name = "maximum"
    metric_type = "numeric"
    name        = "Maximum Latency"
    axis        = "left"
    color       = "#657aa6"
  }

  metric {
    check       = "${circonus_check.api_latency.checks[0]}"
    metric_name = "minimum"
    metric_type = "numeric"
    name        = "Minimum Latency"
    axis        = "right"
    color       = "#0000ff"
  }

  tags = [ "${var.myapp-tags}" ]
}
```

## Argument Reference

* `description` - (Optional) Description of what the graph is for.

* `graph_style` - (Optional) How the graph should be rendered.  Valid options
  are `area` or `line` (default).

* `left` - (Optional) A map of graph left axis options.  Valid values in `left`
  include: `logarithmic` can be set to `0` (default) or `1`; `min` is the `min`
  Y axis value on the left; and `max` is the Y axis max value on the left.

* `line_style` - (Optional) How the line should change between points.  Can be
  either `stepped` (default) or `interpolated`.

* `name` - (Required) The title of the graph.

* `notes` - (Optional) A place for storing notes about this graph.

* `right` - (Optional) A map of graph right axis options.  Valid values in
  `right` include: `logarithmic` can be set to `0` (default) or `1`; `min` is
  the `min` Y axis value on the right; and `max` is the Y axis max value on the
  right.

* `metric` - (Optional) A list of metric streams to graph.  See below for
  options.

* `metric_cluster` - (Optional) A metric cluster to graph.  See below for options.

* `tags` - (Optional) A list of tags assigned to this graph.

## `metric` Configuration

An individual metric stream is the underlying source of data points used for
visualization in a graph. Either a `caql` attribute is required or a `check` and
`metric` must be set. The `metric` attribute can have the following options
set.

* `active` - (Optional) A boolean if the metric stream is enabled or not.

* `alpha` - (Optional) A floating point number between 0 and 1.

* `axis` - (Optional) The axis that the metric stream will use.  Valid options
  are `left` (default) or `right`.

* `caql` - (Optional) A CAQL formula.  Conflicts with the `check` and `metric`
  attributes.

* `check` - (Optional) The check that this metric stream belongs to.

* `color` - (Optional) A hex-encoded color of the line / area on the graph.

* `formula` - (Optional) Formula that should be aplied to both the values in the
  graph and the legend.

* `legend_formula` - (Optional) Formula that should be applied to values in the
  legend.

* `function` - (Optional) What derivative value, if any, should be used.  Valid
  values are: `gauge` (default), `derive`, and `counter (_stddev)`

* `metric_type` - (Required) The type of the metric.  Valid values are:
  `numeric`, `text`, `histogram`, `composite`, or `caql`.

* `name` - (Optional) A name which will appear in the graph legend.

* `metric_name` - (Optional) The name of the metric stream within the check to
  graph.

* `stack` - (Optional) If this metric is to be stacked, which stack set does it
  belong to (starting at `0`).

## `metric_cluster` Configuration

A metric cluster selects multiple metric streams together dynamically using a
query language and returns the set of matching metric streams as a single result
set to the graph rendering engine.

* `active` - (Optional) A boolean if the metric cluster is enabled or not.

* `aggregate` - (Optional) The aggregate function to apply across this metric
  cluster to create a single value.  Valid values are: `none` (default), `min`,
  `max`, `sum`, `mean`, or `geometric_mean`.

* `axis` - (Optional) The axis that the metric cluster will use.  Valid options
  are `left` (default) or `right`.

* `color` - (Optional) A hex-encoded color of the line / area on the graph.
  This is a required attribute when `aggregate` is specified.

* `group` - (Optional) The `metric_cluster` that will provide datapoints for this
  graph.

* `name` - (Optional) A name which will appear in the graph legend for this
  metric cluster.

## Import Example

`circonus_graph` supports importing resources.  Supposing the following
Terraform (and that the referenced [`circonus_metric`](metric.html)
and [`circonus_check`](check.html) have already been imported):

```text
resource "circonus_graph" "icmp-graph" {
  name        = "Test graph"
  graph_style = "line"
  line_style  = "stepped"

  metric {
    check       = "${circonus_check.api_latency.checks[0]}"
    metric_name = "maximum"
    metric_type = "numeric"
    name        = "Maximum Latency"
    axis        = "left"
  }
}
```

It is possible to import a `circonus_graph` resource with the following command:

```
$ terraform import circonus_graph.icmp-graph ID
```

Where `ID` is the `_cid` or Circonus ID of the graph
(e.g. `/graph/bd72aabc-90b9-4039-cc30-c9ab838c18f5`) and
`circonus_graph.icmp-graph` is the name of the resource whose state will be
populated as a result of the command.
