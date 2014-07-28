---
layout: "docs"
page_title: "Resource Graph"
sidebar_current: "docs-internals-graph"
---

# Resource Graph

Terraform builds a
[dependency graph](http://en.wikipedia.org/wiki/Dependency_graph)
from the Terraform configurations, and walks this graph to
generate plans, refresh state, and more. This page documents
the details of what are contained in this graph, what types
of nodes there are, and how the edges of the graph are determined.

<div class="alert alert-block alert-warning">
<strong>Advanced Topic!</strong> This page covers technical details
of Terraform. You don't need to understand these details to
effectively use Terraform. The details are documented here for
those who wish to learn about them without having to go
spelunking through the source code.
</div>

## Graph Nodes

There are only a handful of node types that can exist within the
graph. We'll cover these first before explaining how they're
determined and built:

  * **Resource Node** - Represents a single resource. If you have
    the `count` metaparameter set, then there will be one resource
    node for each count. The configuration, diff, state, etc. of
    the resource under change is attached to this node.

  * **Provider Configuration Node** - Represents the time to fully
    configure a provider. This is when the provider configuration
    block is given to a provider, such as AWS security credentials.

  * **Resource Meta-Node** - Represents a group of resources, but
    does not represent any action on its own. This is done for
    convenience on dependencies and making a prettier graph. This
    node is only present for resources that have a `count`
    parameter greater than 1.

When visualizing a configuration with `terraform graph`, you can
see all of these nodes present.

## Building the Graph

Building the graph is done in a series of sequential steps:

  1. Resources nodes are added based on the configuration. If a
     diff (plan) or state is present, that meta-data is attached
     to each resource node.

  1. Resources are mapped to provisioners if they have any
     defined. This must be done after all resource nodes are
     created so resources with the same provisioner type can
     share the provisioner implementation.

  1. Explicit dependencies from the `depends_on` meta-parameter
     are used to create edges between resources.

  1. If a state is present, any "orphan" resources are added to
     the graph. Orphan resources are any resources that are no
     longer present in the configuration but are present in the
     state file. Orphans never have any configuration associated
     with them, since the state file does not store configuration.

  1. Resources are mapped to providers. Provider configuration
     nodes are created for these providers, and edges are created
     such that the resources depend on their respective provider
     being configured.

  1. Interpolations are parsed in resource and provider configurations
     to determine dependencies. References to resource attributes
     are turned into dependencies from the resource with the interpolation
     to the resource being referenced.

  1. Create a root node. The root node points to all resources and
     is created so there is a single root to the dependency graph. When
     traversing the graph, the root node is ignored.

  1. If a diff is present, traverse all resource nodes and find resources
     that are being destroyed. These resource nodes are split into two:
     one node that destroys the resource and another that creates
     the resource (if it is being recreated). The reason the nodes must
     be split is because the destroy order is often different from the
     create order, and so they can't be represented by a single graph
     node.

  1. Validate the graph has no cycles and has a single root.

## Walking the Graph

To walk the graph, a standard depth-first traversal is done. Graph
walking is done with as much parallelism as possible: a node is walked
as soon as all of its dependencies are walked.
