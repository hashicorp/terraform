---
layout: "datadog"
page_title: "Datadog: datadog_timeboard"
sidebar_current: "docs-datadog-resource-timeboard"
description: |-
  Provides a Datadog timeboard resource. This can be used to create and manage timeboards.
---

# datadog_timeboard

Provides a Datadog timeboard resource. This can be used to create and manage Datadog timeboards.

## Example Usage

```hcl
# Create a new Datadog timeboard
resource "datadog_timeboard" "redis" {
  title       = "Redis Timeboard (created via Terraform)"
  description = "created using the Datadog provider in Terraform"
  read_only   = true

  graph {
    title = "Redis latency (ms)"
    viz   = "timeseries"

    request {
      q    = "avg:redis.info.latency_ms{$host}"
      type = "bars"
    }
  }

  graph {
    title = "Redis memory usage"
    viz   = "timeseries"

    request {
      q       = "avg:redis.mem.used{$host} - avg:redis.mem.lua{$host}, avg:redis.mem.lua{$host}"
      stacked = true
    }

    request {
      q = "avg:redis.mem.rss{$host}"

      style {
        palette = "warm"
      }
    }
  }

  graph {
    title = "Top System CPU by Docker container"
    viz   = "toplist"

    request {
      q = "top(avg:docker.cpu.system{*} by {container_name}, 10, 'mean', 'desc')"
    }
  }

  template_variable {
    name   = "host"
    prefix = "host"
  }
}
```

## Argument Reference

The following arguments are supported:

* `title` - (Required) The name of the dashboard.
* `description` - (Required) A description of the dashboard's content.
* `read_only` - (Optional) The read-only status of the timeboard. Default is false.
* `graph` - (Required) Nested block describing a graph definition. The structure of this block is described below. Multiple graph blocks are allowed within a datadog_timeboard resource.
* `template_variable` - (Optional) Nested block describing a template variable. The structure of this block is described below. Multiple template_variable blocks are allowed within a datadog_timeboard resource.

### Nested `graph` blocks

Nested `graph` blocks have the following structure:

* `title` - (Required) The name of the graph.
* `viz` - (Required) The type of visualization to use for the graph. Valid choices are "change", "distribution", "heatmap", "hostmap", "query_value", timeseries", and "toplist".
* `request` - Nested block describing a graph definition request (a metric query to plot on the graph). The structure of this block is described below. Multiple request blocks are allowed within a graph block.
* `events` - (Optional) A list of event filter strings. Note that, while supported by the Datadog API, the Datadog UI does not (currently) support multiple event filters very well, so use at your own risk.
* `autoscale` - (Optional) Boolean that determines whether to autoscale graphs.
* `precision` - (Optional) Number of digits displayed, use `*` for full precision.
* `custom_unit` - (Optional) Display a custom unit on the graph (such as 'hertz')
* `text_align` - (Optional) How to align text in the graph, can be one of 'left', 'center', or 'right'.
* `style` - (Optional) Nested block describing hostmaps. The structure of this block is described below.
* `group` - (Optional) List of groups for hostmaps (shown as 'group by' in the UI).
* `include_no_metric_hosts` - (Optional) If set to true, will display hosts on hostmap that have no reported metrics.
* `include_ungrouped_hosts` - (Optional) If set to true, will display hosts without groups on hostmaps.
* `scope` - (Optional) List of scopes for hostmaps (shown as 'filter by' in the UI).
* `yaxis` - (Optional) Nested block describing modifications to the yaxis rendering. The structure of this block is described below.
* `marker` - (Optional) Nested block describing lines / ranges added to graph for formatting. The structure of this block is described below. Multiple marker blocks are allowed within a graph block.

#### Nested `graph` `marker` blocks

Nested `graph` `marker` blocks have the following structure:

* `type` - (Required) How the marker lines will look. Possible values are {"error", "warning", "info", "ok"} {"dashed", "solid", "bold"}. Example: "error dashed".
* `value` - (Required) Mathematical expression describing the marker. Examples: "y > 1", "-5 < y < 0", "y = 19".
* `label` - (Optional) A label for the line or range.

{error, warning, info, ok} {dashed, solid, bold}

#### Nested `graph` `yaxis` block
* `min` - (Optional) Minimum bound for the graph's yaxis, a string.
* `max` - (Optional) Maximum bound for the graph's yaxis, a string.
* `scale` - (Optional) How to scale the yaxis. Possible values are: "linear", "log", "sqrt", "pow##" (eg. pow2, pow0.5, 2 is used if only "pow" was provided). Default: "linear".

#### Nested `graph` `request` blocks

Nested `graph` `request` blocks have the following structure:

* `q` - (Required) The query of the request. Pro tip: Use the JSON tab inside the Datadog UI to help build you query strings.
* `aggregator` - (Optional) The aggregation method used when the number of data points outnumbers the max that can be shown.
* `stacked` - (Optional) Boolean value to determine if this is this a stacked area graph. Default: false (line chart).
* `type` - (Optional) Choose how to draw the graph. For example: "line", "bar" or "area". Default: "line".
* `style` - (Optional) Nested block to customize the graph style.

### Nested `graph` `style` block
The nested `style` block is used specifically for styling `hostmap` graphs, and has the following structure:

* `palette` - (Optional) Spectrum of colors to use when styling a hostmap. For example: "green_to_orange", "yellow_to_green", "YlOrRd", or "hostmap_blues". Default: "green_to_orange".
* `palette_flip` - (Optional) Flip how the hostmap is rendered. For example, with the default palette, low values are represented as green, with high values as orange. If palette_flip is "true", then low values will be orange, and high values will be green.

### Nested `graph` `request` `style` block

The nested `style` blocks has the following structure:

* `palette` - (Optional) Color of the line drawn. For example: "classic", "cool", "warm", "purple", "orange" or "gray". Default: "classic".
* `width` - (Optional) Line width. Possible values: "thin", "normal", "thick". Default: "normal".
* `type` - (Optional) Type of line drawn. Possible values: "dashed", "solid", "dotted". Default: "solid".

### Nested `template_variable` blocks

Nested `template_variable` blocks have the following structure:

* `name` - (Required) The variable name. Can be referenced as $name in `graph` `request` `q` query strings.
* `prefix` - (Optional) The tag group. Default: no tag group.
* `default` - (Required) The default tag. Default: "*" (match all).
