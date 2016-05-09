---
layout: "librato"
page_title: "Librato: librato_space_chart"
sidebar_current: "docs-librato-resource-space-chart"
description: |-
  Provides a Librato Space Chart resource. This can be used to create and manage charts in Librato Spaces.
---

# librato\_space\_chart

Provides a Librato Space Chart resource. This can be used to
create and manage charts in Librato Spaces.

## Example Usage

```
# Create a new Librato space
resource "librato_space" "my_space" {
    name = "My New Space"
}

# Create a new chart
resource "librato_space_chart" "server_temperature" {
  name = "Server Temperature"
  space_id = "${librato_space.my_space.id}"

  stream {
    metric = "server_temp"
    source = "app1"
  }

  stream {
    metric = "environmental_temp"
    source = "*"
    group_function = "breakout"
    summary_function = "average"
  }

  stream {
    metric = "server_temp"
    source = "%"
    group_function = "average"
  }
}
```

## Argument Reference

The following arguments are supported:

* `space_id` - (Required) The ID of the space this chart should be in.
* `name` - (Required) The title of the chart when it is displayed.
* `type` - (Optional) Indicates the type of chart. Must be one of line or
  stacked (default to line).
* `min` - (Optional) The minimum display value of the chart's Y-axis.
* `max` - (Optional) The maximum display value of the chart's Y-axis.
* `label` - (Optional) The Y-axis label.
* `related_space` - (Optional) The ID of another space to which this chart is
  related.
* `stream` - (Optional) Nested block describing a metric to use for data in the
  chart. The structure of this block is described below.

The `stream` block supports:

* `metric` - (Required) The name of the metric. May not be specified if
  `composite` is specified.
* `source` - (Required) The name of a source, or `*` to include all sources.
  This field will also accept specific wildcard entries. For example
  us-west-\*-app will match us-west-21-app but not us-west-12-db. Use % to
  specify a dynamic source that will be provided after the instrument or
  dashboard has loaded, or in the URL. May not be specified if `composite` is
  specified.
* `group_function` - (Required) How to process the results when multiple sources
  will be returned. Value must be one of average, sum, breakout. If average or
  sum, a single line will be drawn representing the average or sum
  (respectively) of all sources. If the group_function is breakout, a separate
  line will be drawn for each source. If this property is not supplied, the
  behavior will default to average. May not be specified if `composite` is
  specified.
* `composite` - (Required) A composite metric query string to execute when this
  stream is displayed. May not be specified if `metric`, `source` or
  `group_function` is specified.
* `summary_function` - (Optional) When visualizing complex measurements or a
  rolled-up measurement, this allows you to choose which statistic to use.
  Defaults to "average". Valid options are: "max", "min", "average", "sum" or
  "count".
* `name` - (Optional) A display name to use for the stream when generating the
  tooltip.
* `color` - (Optional) Sets a color to use when rendering the stream. Must be a
  seven character string that represents the hex code of the color e.g.
  "#52D74C".
* `units_short` - (Optional) Unit value string to use as the tooltip label.
* `units_long` - (Optional) String value to set as they Y-axis label. All
  streams that share the same units_long value will be plotted on the same
  Y-axis.
* `min` - (Optional) Theoretical minimum Y-axis value.
* `max` - (Optional) Theoretical maximum Y-axis value.
* `transform_function` - (Optional) Linear formula to run on each measurement
  prior to visualizaton.
* `period` - (Optional) An integer value of seconds that defines the period this
  stream reports at. This aids in the display of the stream and allows the
  period to be used in stream display transforms.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the chart.
* `space_id` - The ID of the space this chart should be in.
* `title` - The title of the chart when it is displayed.
