// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

var _ GraphTransformer = (*checkStartTransformer)(nil)

// checkStartTransformer checks if the configuration has any data blocks nested
// within check blocks, and if it does then it introduces a nodeCheckStart
// vertex that ensures all resources have been applied before it starts loading
// the nested data sources.
type checkStartTransformer struct {
	// Config for the entire module.
	Config *configs.Config

	// Operation is the current operation this node will be part of.
	Operation walkOperation
}

func (s *checkStartTransformer) Transform(graph *Graph) error {
	if s.Operation != walkApply && s.Operation != walkPlan {
		// We only actually execute the checks during plan apply operations
		// so if we are doing something else we can just skip this and
		// leave the graph alone.
		return nil
	}

	var resources []dag.Vertex
	var nested []dag.Vertex

	// We're going to step through all the vertices and pull out the relevant
	// resources and data sources.
	for _, vertex := range graph.Vertices() {
		if node, isResource := vertex.(GraphNodeCreator); isResource {
			addr := node.CreateAddr()

			if addr.Resource.Resource.Mode == addrs.ManagedResourceMode {
				// This is a resource, so we want to make sure it executes
				// before any nested data sources.

				// We can reduce the number of additional edges we write into
				// the graph by only including "leaf" resources, that is
				// resources that aren't referenced by other resources. If a
				// resource is referenced by another resource then we know that
				// it will execute before that resource so we only need to worry
				// about the referencing resource.

				leafResource := true
				for _, other := range graph.UpEdges(vertex) {
					if otherResource, isResource := other.(GraphNodeCreator); isResource {
						otherAddr := otherResource.CreateAddr()
						if otherAddr.Resource.Resource.Mode == addrs.ManagedResourceMode {
							// Then this resource is being referenced so skip
							// it.
							leafResource = false
							break
						}
					}
				}

				if leafResource {
					resources = append(resources, vertex)
				}

				// We've handled the resource so move to the next vertex.
				continue
			}

			// Now, we know we are processing a data block.

			config := s.Config
			if !addr.Module.IsRoot() {
				config = s.Config.Descendent(addr.Module.Module())
			}
			if config == nil {
				// might have been deleted, so it won't be subject to any checks
				// anyway.
				continue
			}

			resource := config.Module.ResourceByAddr(addr.Resource.Resource)
			if resource == nil {
				// might have been deleted, so it won't be subject to any checks
				// anyway.
				continue
			}

			if _, ok := resource.Container.(*configs.Check); ok {
				// Then this is a data source within a check block, so let's
				// make a note of it.
				nested = append(nested, vertex)
			}

			// Otherwise, it's just a normal data source. From a check block we
			// don't really care when Terraform is loading non-nested data
			// sources so we'll just forget about it and move on.
		}
	}

	if len(nested) > 0 {

		// We don't need to do any of this if we don't have any nested data
		// sources, so we check that first.
		//
		// Otherwise we introduce a vertex that can act as a pauser between
		// our nested data sources and leaf resources.

		check := &nodeCheckStart{}
		graph.Add(check)

		// Finally, connect everything up so it all executes in order.

		for _, vertex := range nested {
			graph.Connect(dag.BasicEdge(vertex, check))
		}

		for _, vertex := range resources {
			graph.Connect(dag.BasicEdge(check, vertex))
		}
	}

	return nil
}
