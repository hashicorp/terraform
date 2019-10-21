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

The -type flag can be used to control the type of graph shown. Terraform
creates different graphs for different operations. See the options below
for the list of types supported. The default type is "plan" if a
configuration is given, and "apply" if a plan file is passed as an
argument.

Options:

* `-draw-cycles`    - Highlight any cycles in the graph with colored edges.
                      This helps when diagnosing cycle errors.

* `-module-depth=n` - Specifies the depth of modules to show in the output.
                      By default this is `-1`, which will expand all.

* `-type=plan`      - Type of graph to output. Can be: `plan`, `plan-destroy`, `apply`,
                      `validate`, `input`, `refresh`.

## Generating Images

The output of `terraform graph` is in the DOT format, which can
easily be converted to an image by making use of `dot` provided
by GraphViz:

```shell
$ terraform graph | dot -Tsvg > graph.svg
```

Here is an example graph output:
![Graph Example](docs/graph-example.png)
