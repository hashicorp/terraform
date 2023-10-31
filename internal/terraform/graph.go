// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/tfdiags"

	"github.com/hashicorp/terraform/internal/addrs"

	"github.com/hashicorp/terraform/internal/dag"
)

// Graph represents the graph that Terraform uses to represent resources
// and their dependencies.
type Graph struct {
	// Graph is the actual DAG. This is embedded so you can call the DAG
	// methods directly.
	dag.AcyclicGraph

	// Path is the path in the module tree that this Graph represents.
	Path addrs.ModuleInstance
}

func (g *Graph) DirectedGraph() dag.Grapher {
	return &g.AcyclicGraph
}

// Walk walks the graph with the given walker for callbacks. The graph
// will be walked with full parallelism, so the walker should expect
// to be called in concurrently.
func (g *Graph) Walk(walker GraphWalker) tfdiags.Diagnostics {
	return g.walk(walker)
}

func (g *Graph) walk(walker GraphWalker) tfdiags.Diagnostics {
	// The callbacks for enter/exiting a graph
	ctx := walker.EvalContext()

	// Walk the graph.
	walkFn := func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		// the walkFn is called asynchronously, and needs to be recovered
		// separately in the case of a panic.
		defer logging.PanicHandler()

		log.Printf("[TRACE] vertex %q: starting visit (%T)", dag.VertexName(v), v)

		defer func() {
			if diags.HasErrors() {
				for _, diag := range diags {
					if diag.Severity() == tfdiags.Error {
						desc := diag.Description()
						log.Printf("[ERROR] vertex %q error: %s", dag.VertexName(v), desc.Summary)
					}
				}
				log.Printf("[TRACE] vertex %q: visit complete, with errors", dag.VertexName(v))
			} else {
				log.Printf("[TRACE] vertex %q: visit complete", dag.VertexName(v))
			}
		}()

		// vertexCtx is the context that we use when evaluating. This
		// is normally the context of our graph but can be overridden
		// with a GraphNodeModuleInstance impl.
		vertexCtx := ctx
		if pn, ok := v.(GraphNodeModuleInstance); ok {
			vertexCtx = walker.EnterPath(pn.Path())
			defer walker.ExitPath(pn.Path())
		}

		if g.checkAndApplyOverrides(ctx.Overrides(), v) {
			// We can skip whole vertices if they are in a module that has been
			// overridden.
			return
		}

		// If the node is exec-able, then execute it.
		if ev, ok := v.(GraphNodeExecutable); ok {
			diags = diags.Append(walker.Execute(vertexCtx, ev))
			if diags.HasErrors() {
				return
			}
		}

		// If the node is dynamically expanded, then expand it
		if ev, ok := v.(GraphNodeDynamicExpandable); ok {
			log.Printf("[TRACE] vertex %q: expanding dynamic subgraph", dag.VertexName(v))

			g, moreDiags := ev.DynamicExpand(vertexCtx)
			diags = diags.Append(moreDiags)
			if diags.HasErrors() {
				log.Printf("[TRACE] vertex %q: failed expanding dynamic subgraph: %s", dag.VertexName(v), diags.Err())
				return
			}
			if g != nil {
				// The subgraph should always be valid, per our normal acyclic
				// graph validation rules.
				if err := g.Validate(); err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Graph node has invalid dynamic subgraph",
						fmt.Sprintf("The internal logic for %q generated an invalid dynamic subgraph: %s.\n\nThis is a bug in Terraform. Please report it!", dag.VertexName(v), err),
					))
					return
				}
				// If we passed validation then there is exactly one root node.
				// That root node should always be "rootNode", the singleton
				// root node value.
				if n, err := g.Root(); err != nil || n != dag.Vertex(rootNode) {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Graph node has invalid dynamic subgraph",
						fmt.Sprintf("The internal logic for %q generated an invalid dynamic subgraph: the root node is %T, which is not a suitable root node type.\n\nThis is a bug in Terraform. Please report it!", dag.VertexName(v), n),
					))
					return
				}

				// Walk the subgraph
				log.Printf("[TRACE] vertex %q: entering dynamic subgraph", dag.VertexName(v))
				subDiags := g.walk(walker)
				diags = diags.Append(subDiags)
				if subDiags.HasErrors() {
					var errs []string
					for _, d := range subDiags {
						errs = append(errs, d.Description().Summary)
					}
					log.Printf("[TRACE] vertex %q: dynamic subgraph encountered errors: %s", dag.VertexName(v), strings.Join(errs, ","))
					return
				}
				log.Printf("[TRACE] vertex %q: dynamic subgraph completed successfully", dag.VertexName(v))
			} else {
				log.Printf("[TRACE] vertex %q: produced no dynamic subgraph", dag.VertexName(v))
			}
		}
		return
	}

	return g.AcyclicGraph.Walk(walkFn)
}

// checkAndApplyOverrides checks if target has any data that needs to be overridden.
//
// If this function returns true, then the whole vertex should be skipped and
// not executed.
//
// The logic for a vertex is that if it is within an overridden module then we
// don't want to execute it. Instead, we want to just set the values on the
// output nodes for that module directly. So if a node is a
// GraphNodeModuleInstance we want to skip it if there is an entry in our
// overrides data structure that either matches the module for the vertex or
// is a parent of the module for the vertex.
//
// We also want to actually set the new values for any outputs, resources or
// data sources we encounter that should be overridden.
func (g *Graph) checkAndApplyOverrides(overrides *mocking.Overrides, target dag.Vertex) bool {
	if overrides.Empty() {
		return false
	}

	switch v := target.(type) {
	case GraphNodeOverridable:
		// For resource and data sources, we want to skip them completely if
		// they are within an overridden module.
		resourceInstance := v.ResourceInstanceAddr()
		if overrides.IsDirectOrNestedOverriddenModule(resourceInstance.Module) {
			return true
		}

		if override, ok := overrides.GetOverrideInclProviders(resourceInstance, v.ConfigProvider()); ok {
			v.SetOverride(override)
			return false
		}

		if override, ok := overrides.GetOverrideInclProviders(resourceInstance.ContainingResource(), v.ConfigProvider()); ok {
			v.SetOverride(override)
			return false
		}

	case *NodeApplyableOutput:
		// For outputs, we want to skip them completely if they are deeply
		// nested within an overridden module.
		module := v.Path()
		if overrides.IsNestedOverriddenModule(module) {
			// If the output is deeply nested under an overridden module we want
			// to skip
			return true
		}

		// Otherwise, if we are in a directly overridden module then we want to
		// apply the overridden output values.
		if override, ok := overrides.GetOverride(module); ok {
			key := v.Addr.OutputValue.Name
			if override.Values.Type().HasAttribute(key) {
				v.override = override.Values.GetAttr(key)
			} else {
				// If we don't have a value provided for an output, then we'll
				// just set it to be null.
				//
				// TODO(liamcervante): Can we generate a value here? Probably
				//   not as we don't know the type.
				v.override = cty.NullVal(cty.DynamicPseudoType)
			}
		}

	case GraphNodeModuleInstance:
		// Then this node is simply in a module. It might be that this entire
		// module has been overridden, in which case this node shouldn't
		// execute.
		//
		// We checked for resources and outputs earlier, so we know this isn't
		// anything special.
		module := v.Path()
		if overrides.IsDirectOrNestedOverriddenModule(module) {
			return true
		}
	}

	return false
}
