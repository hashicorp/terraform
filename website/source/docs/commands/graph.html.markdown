---
layout: "docs"
page_title: "Command: graph"
sidebar_current: "docs-commands-graph"
description: |-
  The `terraform graph` command is used to generate a visual representation of either a configuration or execution plan. The output is in the DOT format, which can be used by GraphViz to generate charts.
---

# Command: graph

The `terraform graph` command is used to generate a visual
representation of either a configuration or execution plan.
The output is in the DOT format, which can be used by
[GraphViz](http://www.graphviz.org) to generate charts.


## Usage

Usage: `terraform graph [options] [DIR]`

Outputs the visual dependency graph of Terraform resources according to
configuration files in DIR (or the current directory if omitted).

The graph is outputted in DOT format. The typical program that can
read this format is GraphViz, but many web services are also available
to read this format.

Options:

* `-draw-cycles`    - Highlight any cycles in the graph with colored edges.
                      This helps when diagnosing cycle errors.

* `-module-depth=n` - The maximum depth to expand modules. By default this is
                      zero, which will not expand modules at all.

* `-verbose`        - Generate a verbose, "worst-case" graph, with all nodes
                      for potential operations in place.

## Generating Images

The output of `terraform graph` is in the DOT format, which can
easily be converted to an image by making use of `dot` provided
by GraphViz:

```
$ terraform graph | dot -Tpng > graph.png
```

Here is an example graph output:
![Graph Example](graph-example.png)

