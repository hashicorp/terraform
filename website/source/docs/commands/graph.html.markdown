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

Usage: `terraform graph [options] PATH`

Outputs the visual graph of Terraform resources. If the path given is
the path to a configuration, the dependency graph of the resources are
shown. If the path is a plan file, then the dependency graph of the
plan itself is shown.

Options:

* `-module-depth=n` - The maximum depth to expand modules. By default this is
                      zero, which will not expand modules at all.

## Generating Images

The output of `terraform graph` is in the DOT format, which can
easily be converted to an image by making use of `dot` provided
by GraphViz:

```
$ terraform graph | dot -Tpng > graph.png
```

Alternatively, the web-based [GraphViz Workspace](http://graphviz-dev.appspot.com)
can be used to quickly render DOT file inputs as well.

Here is an example graph output:
![Graph Example](graph-example.png)

