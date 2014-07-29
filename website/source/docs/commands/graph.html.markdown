---
layout: "docs"
page_title: "Command: graph"
sidebar_current: "docs-commands-graph"
---

# Command: graph

The `terraform graph` command is used to generate a visual
representation of either a configuration or execution plan.
The output is in the DOT format, which can be used by
[GraphViz](http://www.graphviz.org) to generate charts.


## Usage

Usage: `terraform output [options] [input]`

By default, `output` scans the current directory for the configuration
and generates the output for that configuration. However, a path to
another configuration or an execution plan can be provided. Execution plans
provide more details on creation, deletion or changes.

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
![Graph Example](/images/graph-example.png)

