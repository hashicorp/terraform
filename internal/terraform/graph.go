// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/internal/logging"
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
	ctx := walker.EvalContext()
	return g.walk(ctx, walker, true)
}

// getTargetable extracts the targetable address from a node. The order
// of the checks is important, as the GraphNodeResourceInstance takes precedence
// over the GraphNodeConfigResource.
func getTargetable(node dag.Vertex) addrs.Targetable {
	switch n := node.(type) {
	case GraphNodeResourceInstance:
		return n.ResourceInstanceAddr()
	case GraphNodeConfigResource:
		return n.ResourceAddr()
	default:
		return nil
	}
}

// setContains checks if a given node or any of its ancestors are present
// in the set. It first checks if the node itself is excluded,
// and if not, it recursively checks all ancestor nodes.
func (g *Graph) setContains(node dag.Vertex, targets addrs.Set[addrs.Targetable]) bool {
	targetable := getTargetable(node)
	if targetable == nil {
		return false
	}

	contains := func(t addrs.Targetable) bool {
		for _, target := range targets {
			if target.TargetContains(t) {
				return true
			}
		}
		return false
	}

	if contains(targetable) {
		return true
	}

	for _, dep := range g.Ancestors(node) {
		if targetable := getTargetable(dep); targetable != nil && contains(targetable) {
			return true
		}
	}
	return false
}

// applyTargeting processes the targeting rules for the graph, handling both inclusion
// and exclusion logic. It returns the set of directly targeted nodes when targeting is enabled.
//
// When targeting is enabled, only nodes that are explicitly targeted or that are ancestors
// of targeted nodes will be included in the traversal.
//
// When exclusion is applied, any node that is explicitly excluded or has an excluded
// ancestor will be excluded from the traversal.
func (g *Graph) applyTargeting(ctx EvalContext, walker GraphWalker, targeted bool) (directlyTargetedNodes dag.Set) {
	filter := ctx.Filter()

	// Exclude any node that is either directly excluded or has an excluded ancestor
	if excludeAddrs := walker.ExcludedAddrs(); excludeAddrs.Size() > 0 {
		for _, node := range g.Vertices() {
			// Skip nodes that are already marked as excluded
			if filter.Matches(node, dag.ExplicitlyExcluded) {
				continue
			}

			// Check if this node should be excluded based on itself or its ancestors
			if g.setContains(node, excludeAddrs) {
				filter.Exclude(node)
			}
		}
	}

	// No graph nodes directly targeted. Includes all nodes that are not explicitly excluded.
	if !targeted {
		for _, node := range g.Vertices() {
			if !filter.Matches(node, dag.ExplicitlyExcluded) {
				filter.Include(node)
			}
		}
		return nil
	}

	// Get and sort target addresses for deterministic behavior
	less := func(i, j addrs.Targetable) bool {
		return i.String() < j.String()
	}
	targets := walker.TargetAddrs().Sorted(less)

	// If we have targeting enabled but no specific targets,
	// include everything not excluded (same as !targeted case)
	if len(targets) == 0 {
		for _, node := range g.Vertices() {
			if !filter.Matches(node, dag.ExplicitlyExcluded) {
				filter.Include(node)
			}
		}
		return nil
	}

	// Process targeted nodes
	var allTargetedNodes dag.Set
	directlyTargetedNodes, allTargetedNodes = selectTargetedNodes(g, targets)

	// Include all nodes that are either directly targeted or ancestors of targeted nodes
	for _, node := range allTargetedNodes {
		filter.Include(node)
	}

	// Exclude everything else
	for _, node := range g.Vertices() {
		if !filter.Matches(node, dag.Allowed) {
			filter.Exclude(node)
		}
	}

	return directlyTargetedNodes
}

func (g *Graph) walk(ctx EvalContext, walker GraphWalker, targeted bool) tfdiags.Diagnostics {
	directTargets := g.applyTargeting(ctx, walker, targeted)

	// The callbacks for enter/exiting a graph
	// Walk the graph.
	walkFn := func(v dag.Vertex) (diags tfdiags.Diagnostics) {
		// the walkFn is called asynchronously, and needs to be recovered
		// separately in the case of a panic.
		defer logging.PanicHandler()

		log.Printf("[TRACE] vertex %q: starting visit (%T)", dag.VertexName(v), v)

		defer func() {
			if r := recover(); r != nil {
				// If the walkFn panics, we get confusing logs about how the
				// visit was complete. To stop this, we'll catch the panic log
				// that the vertex panicked without finishing and re-panic.
				log.Printf("[ERROR] vertex %q panicked", dag.VertexName(v))
				panic(r) // re-panic
			}

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

		haveOverrides := !ctx.Overrides().Empty()

		// If the graph node is overridable, we'll check our overrides to see
		// if we need to apply any overrides to the node.
		if overridable, ok := v.(GraphNodeOverridable); ok && haveOverrides {
			// It'd be nice if we could just pass the overrides directly into
			// the nodes, but the way the AbstractNodeResource is created is
			// complicated and it's not easy to make sure that every
			// implementation sets the overrides correctly. Instead, we just
			// do it from this single location to keep things simple.
			//
			// See the output node for an example of providing the overrides
			// directly to the node.
			if override, ok := ctx.Overrides().GetResourceOverride(overridable.ResourceInstanceAddr(), overridable.ConfigProvider()); ok {
				overridable.SetOverride(override)
			}
		}

		if provider, ok := v.(GraphNodeProvider); ok && haveOverrides {
			// If we find a legacy provider within an overridden module, we
			// can't evaluate the config so we have to skip it. We do this here
			// for the similar reasons as the resource overrides above, and to
			// keep all the override logic together.
			addr := provider.ProviderAddr()
			// UnkeyedInstanceShim is used by legacy provider configs within a
			// module to return an instance of that module, since they can never
			// exist within an expanded instance.
			if ctx.Overrides().IsOverridden(addr.Module.UnkeyedInstanceShim()) {
				log.Printf("[DEBUG] skipping provider %s found within overridden module", addr)
				return
			}
		}

		// vertexCtx is the context that we use when evaluating. This
		// is normally the global context but can be overridden
		// with either a GraphNodeModuleInstance, GraphNodePartialExpandedModule,
		// or graphNodeEvalContextScope implementation. (These interfaces are
		// all intentionally mutually-exclusive by having the same method
		// name but different signatures, since a node can only belong to
		// one context at a time.)
		vertexCtx := ctx
		if pn, ok := v.(graphNodeEvalContextScope); ok {
			scope := pn.Path()
			log.Printf("[TRACE] vertex %q: belongs to %s", dag.VertexName(v), scope)
			vertexCtx = walker.enterScope(scope)
			defer walker.exitScope(scope)
		} else if pn, ok := v.(GraphNodeModuleInstance); ok {
			moduleAddr := pn.Path() // An addrs.ModuleInstance
			log.Printf("[TRACE] vertex %q: belongs to %s", dag.VertexName(v), moduleAddr)
			scope := evalContextModuleInstance{
				Addr: moduleAddr,
			}
			vertexCtx = walker.enterScope(scope)
			defer walker.exitScope(scope)
		} else if pn, ok := v.(GraphNodePartialExpandedModule); ok {
			moduleAddr := pn.Path() // An addrs.PartialExpandedModule
			log.Printf("[TRACE] vertex %q: belongs to all of %s", dag.VertexName(v), moduleAddr)
			scope := evalContextPartialExpandedModule{
				Addr: moduleAddr,
			}
			vertexCtx = walker.enterScope(scope)
			defer walker.exitScope(scope)
		} else {
			log.Printf("[TRACE] vertex %q: does not belong to any module instance", dag.VertexName(v))
		}

		// When working with embedded objects (e.g NodeAbstractResourceInstance in NodePlannableResourceInstance),
		// the filter may contain the outer type (NodePlannableResourceInstance) but the current method might
		// be called on the inner embedded type (NodeAbstractResourceInstance). In this scenario,
		// filter.Allowed(v) would fail because the filter doesn't recognize the inner type directly.
		//
		// Therefore, we need to explicitly check if the node can be excluded, and if it's not allowed
		// by the filter, mark it as excluded
		filter := ctx.Filter()
		if !filter.Allowed(v) {
			if ev, ok := v.(GraphNodeExcludeable); ok {
				ev.SetExcluded(true)
			}
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
				// If the dynamic node is excluded, we should exclude all of the
				// nodes in its subgraph.
				if filter.Matches(v, dag.ExplicitlyExcluded) {
					for _, node := range g.Vertices() {
						filter.Exclude(node)
					}
				}

				// If the dynamic node was directly targeted with a target value,
				// but the target value is a more specific target inside
				// the dynamic node, we want to filter that specific target.
				// For example, when the target is "resource.foo[0]", but the
				// dynamic node represents the config resource "resource.foo".
				targeted := directTargets.Include(v)

				subDiags := g.walk(ctx, walker, targeted)
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

// ResourceGraph derives a graph containing addresses of only the nodes in the
// receiver which implement [GraphNodeConfigResource], describing the
// relationships between all of their [addrs.ConfigResource] addresses.
//
// Nodes that do not have resource addresses are discarded but the
// result preserves correct dependency relationships for the nodes that are
// left, still taking into account any indirect dependencies through nodes
// that were discarded.
func (g *Graph) ResourceGraph() addrs.DirectedGraph[addrs.ConfigResource] {
	// For now we're doing this in a kinda-janky way, by first constructing
	// a reduced graph containing only GraphNodeConfigResource implementations
	// and then using that temporary graph to construct the final graph to
	// return.

	log.Printf("[TRACE] ResourceGraph: copying source graph\n")
	tmpG := Graph{}
	tmpG.Subsume(&g.Graph)
	log.Printf("[TRACE] ResourceGraph: reducing graph\n")
	tmpG.reducePreservingRelationships(func(n dag.Vertex) bool {
		_, ret := n.(GraphNodeConfigResource)
		return ret
	})
	log.Printf("[TRACE] ResourceGraph: TransitiveReduction\n")

	// The resulting graph could have many more edges now, but alternate paths
	// are not a problem for the deferral system, so we may choose not to run
	// this as it may be very time consuming. The reducePreservingRelationships
	// method also doesn't add many (if any) redundant new edges to most graphs.
	tmpG.TransitiveReduction()

	log.Printf("[TRACE] ResourceGraph: creating address graph\n")
	ret := addrs.NewDirectedGraph[addrs.ConfigResource]()
	for _, n := range tmpG.Vertices() {
		sourceR := n.(GraphNodeConfigResource)
		sourceAddr := sourceR.ResourceAddr()
		ret.Add(sourceAddr)
		for _, dn := range tmpG.DownEdges(n) {
			targetR := dn.(GraphNodeConfigResource)

			ret.AddDependency(sourceAddr, targetR.ResourceAddr())
		}
	}
	log.Printf("[TRACE] ResourceGraph: completed with %d nodes\n", len(ret.AllNodes()))
	return ret
}

// reducePreservingRelationships modifies the receiver in-place so that it only
// contains the nodes for which keepNode returns true, but also adds new
// edges to preserve the dependency relationships for all of the nodes
// that still remain.
func (g *Graph) reducePreservingRelationships(keepNode func(dag.Vertex) bool) {
	for _, n := range g.Vertices() {
		if keepNode(n) {
			continue
		}

		// If we're not going to keep this node then we need to connect
		// all of its dependents to all of its dependencies so that the
		// ordering is still preserved for those nodes that remain.
		// However, this will often generate more edges than are strictly
		// required and so it could be productive to run a transitive
		// reduction afterwards.
		dependents := g.UpEdges(n)
		dependencies := g.DownEdges(n)
		for dependent := range dependents {
			for dependency := range dependencies {
				edge := dag.BasicEdge(dependent, dependency)
				g.Connect(edge)
			}
		}
		g.Remove(n)
	}
}
