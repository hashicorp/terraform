---
layout: "datadog"
page_title: "Datadog: datadog_timeboard"
sidebar_current: "docs-datadog-resource-timeboard"
description: |-
  Provides a Datadog timeboard resource. This can be used to create and manage timeboards.
---

# datadog\_timeboard

Provides a Datadog timeboard resource. This can be used to create and manage Datadog timeboards.

## Example Usage

```
# Create a new Datadog timeboard
resource "datadog_timeboard" "redis" {

  title = "Redis Timeboard (created via Terraform)"
  description = "created using the Datadog provider in Terraform"
  read_only = true

  graph {
    title = "Redis latency (ms)"
    viz = "timeseries"
    request {
      q = "avg:redis.info.latency_ms{$host}"
      type = "bars"
    }
  }
  
  graph {
    title = "Redis memory usage"
    viz = "timeseries"
    request {
      q = "avg:redis.mem.used{$host} - avg:redis.mem.lua{$host}, avg:redis.mem.lua{$host}"
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
    viz = "toplist"
    request {
      q = "top(avg:docker.cpu.system{*} by {container_name}, 10, 'mean', 'desc')"
    }
  }

  template_variable {
    name = "host"
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

#### Nested `graph` `request` blocks

Nested `graph` `request` blocks have the following structure:

* `q` - (Required) The query of the request. Pro tip: Use the JSON tab inside the Datadog UI to help build you query strings.
* `stacked` - (Optional) Boolean value to determin if this is this a stacked area graph. Default: false (line chart).
* `type` - (Optional) Choose how to draw the graph. For example: "lines", "bars" or "areas". Default: "lines".
* `style` - (Optional) Nested block to customize the graph style.

### Nested `style` block

The nested `style` blocks has the following structure (only `palette` is supported right now):

* `palette` - (Optional) Color of the line drawn. For example: "classic", "cool", "warm", "purple", "orange" or "gray". Default: "classic".

### Nested `template_variable` blocks

Nested `template_variable` blocks have the following structure:

* `name` - (Required) The variable name. Can be referenced as $name in `graph` `request` `q` query strings.
* `prefix` - (Optional) The tag group. Default: no tag group.
* `default` - (Required) The default tag. Default: "*" (match all).
