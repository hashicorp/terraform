// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (n *nodeExpandPlannableResource) dynamicExpandPartial(ctx EvalContext, knownModules []addrs.ModuleInstance, partialModules addrs.Set[addrs.PartialExpandedModule], imports addrs.Map[addrs.AbsResourceInstance, string]) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	var diags tfdiags.Diagnostics

	knownResources := addrs.MakeSet[addrs.AbsResourceInstance]()
	partialResources := addrs.MakeSet[addrs.PartialExpandedResource]()
	maybeOrphanResources := addrs.MakeSet[addrs.AbsResourceInstance]()

	for _, moduleAddr := range knownModules {
		resourceAddr := n.Addr.Resource.Absolute(moduleAddr)
		resources, partials, maybeOrphans, moreDiags := n.expandKnownModule(ctx, resourceAddr, imports, &g)
		diags = diags.Append(moreDiags)

		// Track all the resources we know about.
		knownResources = knownResources.Union(resources)
		partialResources = partialResources.Union(partials)
		maybeOrphanResources = maybeOrphanResources.Union(maybeOrphans)
	}

	// TODO: What about targeting and force replacement for these resources?
	//   For now, it actually kind of works out because we don't want to make
	//   any changes for these and that's what happens. Later, when we start
	//   tracking deferrals in the plan, we'll need to make sure that the
	//   targeting is applied properly.

	for _, moduleAddr := range partialModules {
		resourceAddr := moduleAddr.Resource(n.Addr.Resource)
		partialResources.Add(resourceAddr)

		// And add a node to the graph for this resource.
		g.Add(&nodePlannablePartialExpandedResource{
			addr:             resourceAddr,
			config:           n.Config,
			resolvedProvider: n.ResolvedProvider,
			skipPlanChanges:  n.skipPlanChanges,
		})
	}

	func() {

		ss := ctx.PrevRunState()
		state := ss.Lock()
		defer ss.Unlock()

	Resources:
		for _, res := range state.Resources(n.Addr) {

			for _, knownModule := range knownModules {
				if knownModule.Equal(res.Addr.Module) {
					// Then we handled this resource as part of the known
					// modules processing.
					continue Resources
				}
			}

			for _, partialResource := range partialResources {
				if partialResource.MatchesResource(res.Addr) {

					for key := range res.Instances {
						// Then each of the instances is a "maybe orphan"
						// instance, and we need to add a node for that.
						maybeOrphanResources.Add(res.Addr.Instance(key))
						g.Add(n.concreteResource(addrs.MakeMap[addrs.AbsResourceInstance, string](), true)(NewNodeAbstractResourceInstance(res.Addr.Instance(key))))

					}

					// Move onto the next resource.
					continue Resources
				}
			}

			// Otherwise, everything in here is just a simple orphaned instance.

			for key := range res.Instances {
				inst := res.Addr.Instance(key)
				abs := NewNodeAbstractResourceInstance(inst)
				abs.AttachResourceState(res)
				g.Add(n.concreteResourceOrphan(abs))
			}

		}

	}()

	// We need to ensure that all of the expanded import targets are actually
	// present in the configuration, because we can't import something that
	// doesn't exist.
ImportValidation:
	for _, addr := range imports.Keys() {
		if knownResources.Has(addr) {
			// Simple case, this is known to be in the configuration so we
			// skip it.
			continue
		}

		for _, partialAddr := range partialResources {
			if partialAddr.MatchesInstance(addr) {
				// This is a partial-expanded address, so we can't yet know
				// whether it's in the configuration or not, and so we'll
				// defer dealing with it to a future round.
				continue ImportValidation
			}
		}

		if maybeOrphanResources.Has(addr) {
			// This is in the previous state but we can't yet know whether
			// it's still desired, so we'll defer dealing with it to a future
			// round.
			continue
		}

		// If we get here then the import target is not in the configuration
		// at all, and so we'll report an error.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Configuration for import target does not exist",
			fmt.Sprintf("The configuration for the given import %s does not exist. All target instances must have an associated configuration to be imported.", addr),
		))
	}

	// If this is a resource that participates in custom condition checks
	// (i.e. it has preconditions or postconditions) then the check state
	// wants to know the addresses of the checkable objects so that it can
	// treat them as unknown status if we encounter an error before actually
	// visiting the checks.
	if checkState := ctx.Checks(); checkState.ConfigHasChecks(n.NodeAbstractResource.Addr) {
		checkables := addrs.MakeSet[addrs.Checkable]()
		for _, addr := range knownResources {
			checkables.Add(addr)
		}
		for _, addr := range maybeOrphanResources {
			checkables.Add(addr)
		}

		checkState.ReportCheckableObjects(n.NodeAbstractResource.Addr, checkables)
	}

	addRootNodeToGraph(&g)
	return &g, diags
}

func (n *nodeExpandPlannableResource) expandKnownModule(globalCtx EvalContext, resAddr addrs.AbsResource, imports addrs.Map[addrs.AbsResourceInstance, string], g *Graph) (addrs.Set[addrs.AbsResourceInstance], addrs.Set[addrs.PartialExpandedResource], addrs.Set[addrs.AbsResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	moduleCtx := evalContextForModuleInstance(globalCtx, resAddr.Module)

	moreDiags := n.writeResourceState(moduleCtx, resAddr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, nil, nil, diags
	}

	expander := moduleCtx.InstanceExpander()
	_, knownInstKeys, haveUnknownKeys := expander.ResourceInstanceKeys(resAddr)

	knownResources := addrs.MakeSet[addrs.AbsResourceInstance]()
	partialResources := addrs.MakeSet[addrs.PartialExpandedResource]()

	for _, key := range knownInstKeys {
		knownResources.Add(resAddr.Instance(key))
	}
	if haveUnknownKeys {
		partialResources.Add(resAddr.Module.UnexpandedResource(resAddr.Resource))
	}

	mustHaveIndex := len(knownInstKeys) != 1 || haveUnknownKeys
	if len(knownInstKeys) == 1 && knownInstKeys[0] != addrs.NoKey {
		mustHaveIndex = true
	}
	if mustHaveIndex {
		var instanceAddrs []addrs.AbsResourceInstance
		for _, key := range knownInstKeys {
			instanceAddrs = append(instanceAddrs, resAddr.Instance(key))
		}
		diags = diags.Append(n.validForceReplaceTargets(instanceAddrs))
	}

	instGraph, maybeOrphanResources, instDiags := n.knownModuleSubgraph(moduleCtx, resAddr, knownInstKeys, haveUnknownKeys, imports)
	diags = diags.Append(instDiags)
	if instDiags.HasErrors() {
		return nil, nil, nil, diags
	}
	g.Subsume(&instGraph.AcyclicGraph.Graph)
	return knownResources, partialResources, maybeOrphanResources, diags
}

func (n *nodeExpandPlannableResource) knownModuleSubgraph(ctx EvalContext, addr addrs.AbsResource, knownInstKeys []addrs.InstanceKey, haveUnknownKeys bool, imports addrs.Map[addrs.AbsResourceInstance, string]) (*Graph, addrs.Set[addrs.AbsResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if n.Config == nil && n.generateConfigPath != "" && imports.Len() == 0 {
		// We're generating configuration, but there's nothing to import, which
		// means the import block must have expanded to zero instances.
		// the instance expander will always return a single instance because
		// we have assumed there will eventually be a configuration for this
		// resource, so return here before we add that to the graph.
		return &Graph{}, nil, diags
	}

	// Our graph transformers require access to the full state, so we'll
	// temporarily lock it while we work on this.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	maybeOrphans := addrs.MakeSet[addrs.AbsResourceInstance]()

	steps := []GraphTransformer{

		DynamicTransformer(func(graph *Graph) error {
			// We'll add a node for all the known instance keys.
			for _, key := range knownInstKeys {
				graph.Add(n.concreteResource(imports, n.skipPlanChanges)(NewNodeAbstractResourceInstance(addr.Instance(key))))
			}
			return nil
		}),

		DynamicTransformer(func(graph *Graph) error {
			// We'll add a node if there are unknown instance keys.
			if haveUnknownKeys {
				graph.Add(&nodePlannablePartialExpandedResource{
					addr:             addr.Module.UnexpandedResource(addr.Resource),
					config:           n.Config,
					resolvedProvider: n.ResolvedProvider,
					skipPlanChanges:  n.skipPlanChanges,
				})
			}
			return nil
		}),

		DynamicTransformer(func(graph *Graph) error {
			// We'll add nodes for any orphaned resources.
			rs := state.Resource(addr)
		Instances:
			for key, inst := range rs.Instances {
				if inst.Current == nil {
					continue
				}

				for _, knownKey := range knownInstKeys {
					if knownKey == key {
						// Then we have a known instance, so we can skip this
						// one - it's definitely not an orphan.
						continue Instances
					}
				}

				if haveUnknownKeys {
					// Then this is a "maybe orphan" instance. It isn't mapped
					// to a known instance but we have unknown keys so we don't
					// know for sure that it's been deleted.
					maybeOrphans.Add(addr.Instance(key))
					graph.Add(n.concreteResource(addrs.MakeMap[addrs.AbsResourceInstance, string](), true)(NewNodeAbstractResourceInstance(addr.Instance(key))))
					continue
				}

				// If none of the above, then this is definitely an orphan.
				graph.Add(n.concreteResourceOrphan(NewNodeAbstractResourceInstance(addr.Instance(key))))
			}

			return nil
		}),

		// Attach the state
		&AttachStateTransformer{State: state},

		// Targeting
		&TargetsTransformer{Targets: n.Targets},

		// Connect references so ordering is correct
		&ReferenceTransformer{},

		// Make sure there is a single root
		&RootTransformer{},
	}

	b := &BasicGraphBuilder{
		Steps: steps,
		Name:  "nodeExpandPlannableResource",
	}
	graph, graphDiags := b.Build(addr.Module)
	diags = diags.Append(graphDiags)
	return graph, maybeOrphans, diags
}

// transformDynamic is a helper struct that wraps a single function, allowing
// us to transform a graph dynamically.
type transformDynamic struct {
	Transformer func(*Graph) error
}

// DynamicTransformer returns a GraphTransformer that will apply the given
// function to the graph during the dynamic expansion phase.
func DynamicTransformer(f func(*Graph) error) GraphTransformer {
	return &transformDynamic{Transformer: f}
}

// implements GraphTransformer
func (t *transformDynamic) Transform(g *Graph) error {
	return t.Transformer(g)
}
