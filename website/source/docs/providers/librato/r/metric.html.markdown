---
layout: "librato"
page_title: "Librato: librato_metric"
sidebar_current: "docs-librato-resource-metric"
description: |-
  Provides a Librato Metric resource. This can be used to create and manage metrics on Librato.
---

# librato\_metric

Provides a Librato Metric resource. This can be used to create and manage metrics on Librato.

## Example Usage

```hcl
# Create a new Librato metric
resource "librato_metric" "mymetric" {
    name = "MyMetric"
    type = "counter"
    description = "A Test Metric"
    attributes {
      display_stacked = true
    }
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The type of metric to create (gauge, counter, or composite).
* `name` - (Required) The unique identifier of the metric.
* `display_name` - The name which will be used for the metric when viewing the Metrics website.
* `description` - Text that can be used to explain precisely what the metric is measuring.
* `period` - Number of seconds that is the standard reporting period of the metric.
* `attributes` - The attributes hash configures specific components of a metric’s visualization.
* `composite` - The definition of the composite metric.

## Attributes Reference

The following attributes are exported:

* `name` - The identifier for the metric.
* `display_name` - The name which will be used for the metric when viewing the Metrics website.
* `type` - The type of metric to create (gauge, counter, or composite).
* `description` - Text that describes precisely what the metric is measuring.
* `period` - Number of seconds that is the standard reporting period of the metric. Setting the period enables Metrics to detect abnormal interruptions in reporting and aids in analytics. For gauge metrics that have service-side aggregation enabled, this option will define the period that aggregation occurs on.
* `source_lag` -
* `composite` - The composite definition. Only used when type is composite.

Attributes (`attributes`) support the following:

* `color` - Sets a default color to prefer when visually rendering the metric. Must be a seven character string that represents the hex code of the color e.g. #52D74C.
* `display_max` - If a metric has a known theoretical maximum value, set display_max so that visualizations can provide perspective of the current values relative to the maximum value.
* `display_min` - If a metric has a known theoretical minimum value, set display_min so that visualizations can provide perspective of the current values relative to the minimum value.
* `display_units_long` - A string that identifies the unit of measurement e.g. Microseconds. Typically the long form of display_units_short and used in visualizations e.g. the Y-axis label on a graph.
* `display_units_short` -	A terse (usually abbreviated) string that identifies the unit of measurement e.g. uS (Microseconds). Typically the short form of display_units_long and used in visualizations e.g. the tooltip for a point on a graph.
* `display_stacked` -	A boolean value indicating whether or not multiple metric streams should be aggregated in a visualization (e.g. stacked graphs). By default counters have display_stacked enabled while gauges have it disabled.
* `summarize_function` -	Determines how to calculate values when rolling up from raw values to higher resolution intervals. Must be one of: ‘average’, 'sum’, 'count’, 'min’, 'max’. If summarize_function is not set the behavior defaults to average.

If the values of the measurements to be rolled up are: 2, 10, 5:

* average: 5.67
* sum: 17
* count: 3
* min: 2
* max: 10

* `aggregate`	- Enable service-side aggregation for this metric. When enabled, measurements sent using the same tag set will be aggregated into single measurements on an interval defined by the period of the metric. If there is no period defined for the metric then all measurements will be aggregated on a 60-second interval.

This option takes a value of true or false. If this option is not set for a metric it will default to false.
