// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandPlannableResource represents an addrs.ConfigResource and implements
// DynamicExpand to a subgraph containing all of the addrs.AbsResourceInstance
// resulting from both the containing module and resource-specific expansion.
type nodeExpandPlannableResource struct {
	*NodeAbstractResource

	// ForceCreateBeforeDestroy might be set via our GraphNodeDestroyerCBD
	// during graph construction, if dependencies require us to force this
	// on regardless of what the configuration says.
	ForceCreateBeforeDestroy *bool

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	preDestroyRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool

	// forceReplace are resource instance addresses where the user wants to
	// force generating a replace action. This set isn't pre-filtered, so
	// it might contain addresses that have nothing to do with the resource
	// that this node represents, which the node itself must therefore ignore.
	forceReplace []addrs.AbsResourceInstance

	// We attach dependencies to the Resource during refresh, since the
	// instances are instantiated during DynamicExpand.
	// FIXME: These would be better off converted to a generic Set data
	// structure in the future, as we need to compare for equality and take the
	// union of multiple groups of dependencies.
	dependencies []addrs.ConfigResource

	// legacyImportMode is set if the graph is being constructed following an
	// invocation of the legacy "terraform import" CLI command.
	legacyImportMode bool
}

var (
	_ GraphNodeDestroyerCBD         = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeDynamicExpandable    = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeReferenceable        = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeReferencer           = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeConfigResource       = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeAttachDependencies   = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeTargetable           = (*nodeExpandPlannableResource)(nil)
	_ graphNodeExpandsInstances     = (*nodeExpandPlannableResource)(nil)
)

func (n *nodeExpandPlannableResource) Name() string {
	return n.NodeAbstractResource.Name() + " (expand)"
}

func (n *nodeExpandPlannableResource) expandsInstances() {
}

// GraphNodeAttachDependencies
func (n *nodeExpandPlannableResource) AttachDependencies(deps []addrs.ConfigResource) {
	n.dependencies = deps
}

// GraphNodeDestroyerCBD
func (n *nodeExpandPlannableResource) CreateBeforeDestroy() bool {
	if n.ForceCreateBeforeDestroy != nil {
		return *n.ForceCreateBeforeDestroy
	}

	// If we have no config, we just assume no
	if n.Config == nil || n.Config.Managed == nil {
		return false
	}

	return n.Config.Managed.CreateBeforeDestroy
}

// GraphNodeDestroyerCBD
func (n *nodeExpandPlannableResource) ModifyCreateBeforeDestroy(v bool) error {
	n.ForceCreateBeforeDestroy = &v
	return nil
}

func (n *nodeExpandPlannableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph

	expander := ctx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module)

	// Lock the state while we inspect it
	state := ctx.State().Lock()

	var orphans []*states.Resource
	for _, res := range state.Resources(n.Addr) {
		found := false
		for _, m := range moduleInstances {
			if m.Equal(res.Addr.Module) {
				found = true
				break
			}
		}
		// The module instance of the resource in the state doesn't exist
		// in the current config, so this whole resource is orphaned.
		if !found {
			orphans = append(orphans, res)
		}
	}

	// We'll no longer use the state directly here, and the other functions
	// we'll call below may use it so we'll release the lock.
	state = nil
	ctx.State().Unlock()

	// The concrete resource factory we'll use for orphans
	concreteResourceOrphan := func(a *NodeAbstractResourceInstance) *NodePlannableResourceInstanceOrphan {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.Schema = n.Schema
		a.ProvisionerSchemas = n.ProvisionerSchemas
		a.ProviderMetas = n.ProviderMetas
		a.Dependencies = n.dependencies

		return &NodePlannableResourceInstanceOrphan{
			NodeAbstractResourceInstance: a,
			skipRefresh:                  n.skipRefresh,
			skipPlanChanges:              n.skipPlanChanges,
		}
	}

	for _, res := range orphans {
		for key := range res.Instances {
			addr := res.Addr.Instance(key)
			abs := NewNodeAbstractResourceInstance(addr)
			abs.AttachResourceState(res)
			n := concreteResourceOrphan(abs)
			g.Add(n)
		}
	}

	// The above dealt with the expansion of the containing module, so now
	// we need to deal with the expansion of the resource itself across all
	// instances of the module.
	//
	// We'll gather up all of the leaf instances we learn about along the way
	// so that we can inform the checks subsystem of which instances it should
	// be expecting check results for, below.
	var diags tfdiags.Diagnostics
	instAddrs := addrs.MakeSet[addrs.Checkable]()
	for _, module := range moduleInstances {
		resAddr := n.Addr.Resource.Absolute(module)
		err := n.expandResourceInstances(ctx, resAddr, &g, instAddrs)
		diags = diags.Append(err)
	}
	if diags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	// If this is a resource that participates in custom condition checks
	// (i.e. it has preconditions or postconditions) then the check state
	// wants to know the addresses of the checkable objects so that it can
	// treat them as unknown status if we encounter an error before actually
	// visiting the checks.
	if checkState := ctx.Checks(); checkState.ConfigHasChecks(n.NodeAbstractResource.Addr) {
		checkState.ReportCheckableObjects(n.NodeAbstractResource.Addr, instAddrs)
	}

	addRootNodeToGraph(&g)

	return &g, diags.ErrWithWarnings()
}

// expandResourceInstances calculates the dynamic expansion for the resource
// itself in the context of a particular module instance.
//
// It has several side-effects:
//   - Adds a node to Graph g for each leaf resource instance it discovers, whether present or orphaned.
//   - Registers the expansion of the resource in the "expander" object embedded inside EvalContext ctx.
//   - Adds each present (non-orphaned) resource instance address to instAddrs (guaranteed to always be addrs.AbsResourceInstance, despite being declared as addrs.Checkable).
//
// After calling this for each of the module instances the resource appears
// within, the caller must register the final superset instAddrs with the
// checks subsystem so that it knows the fully expanded set of checkable
// object instances for this resource instance.
func (n *nodeExpandPlannableResource) expandResourceInstances(globalCtx EvalContext, resAddr addrs.AbsResource, g *Graph, instAddrs addrs.Set[addrs.Checkable]) error {
	var diags tfdiags.Diagnostics

	// The rest of our work here needs to know which module instance it's
	// working in, so that it can evaluate expressions in the appropriate scope.
	moduleCtx := globalCtx.WithPath(resAddr.Module)

	// writeResourceState is responsible for informing the expander of what
	// repetition mode this resource has, which allows expander.ExpandResource
	// to work below.
	moreDiags := n.writeResourceState(moduleCtx, resAddr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	// Before we expand our resource into potentially many resource instances,
	// we'll verify that any mention of this resource in n.forceReplace is
	// consistent with the repetition mode of the resource. In other words,
	// we're aiming to catch a situation where naming a particular resource
	// instance would require an instance key but the given address has none.
	expander := moduleCtx.InstanceExpander()
	instanceAddrs := expander.ExpandResource(resAddr)

	// If there's a number of instances other than 1 then we definitely need
	// an index.
	mustHaveIndex := len(instanceAddrs) != 1
	// If there's only one instance then we might still need an index, if the
	// instance address has one.
	if len(instanceAddrs) == 1 && instanceAddrs[0].Resource.Key != addrs.NoKey {
		mustHaveIndex = true
	}
	if mustHaveIndex {
		for _, candidateAddr := range n.forceReplace {
			if candidateAddr.Resource.Key == addrs.NoKey {
				if n.Addr.Resource.Equal(candidateAddr.Resource.Resource) {
					switch {
					case len(instanceAddrs) == 0:
						// In this case there _are_ no instances to replace, so
						// there isn't any alternative address for us to suggest.
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Warning,
							"Incompletely-matched force-replace resource instance",
							fmt.Sprintf(
								"Your force-replace request for %s doesn't match any resource instances because this resource doesn't have any instances.",
								candidateAddr,
							),
						))
					case len(instanceAddrs) == 1:
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Warning,
							"Incompletely-matched force-replace resource instance",
							fmt.Sprintf(
								"Your force-replace request for %s doesn't match any resource instances because it lacks an instance key.\n\nTo force replacement of the single declared instance, use the following option instead:\n  -replace=%q",
								candidateAddr, instanceAddrs[0],
							),
						))
					default:
						var possibleValidOptions strings.Builder
						for _, addr := range instanceAddrs {
							fmt.Fprintf(&possibleValidOptions, "\n  -replace=%q", addr)
						}

						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Warning,
							"Incompletely-matched force-replace resource instance",
							fmt.Sprintf(
								"Your force-replace request for %s doesn't match any resource instances because it lacks an instance key.\n\nTo force replacement of particular instances, use one or more of the following options instead:%s",
								candidateAddr, possibleValidOptions.String(),
							),
						))
					}
				}
			}
		}
	}
	// NOTE: The actual interpretation of n.forceReplace to produce replace
	// actions is in the per-instance function we're about to call, because
	// we need to evaluate it on a per-instance basis.

	for _, addr := range instanceAddrs {
		// If this resource is participating in the "checks" mechanism then our
		// caller will need to know all of our expanded instance addresses as
		// checkable object instances.
		// (NOTE: instAddrs probably already has other instance addresses in it
		// from earlier calls to this function with different resource addresses,
		// because its purpose is to aggregate them all together into a single set.)
		instAddrs.Add(addr)
	}

	// Our graph builder mechanism expects to always be constructing new
	// graphs rather than adding to existing ones, so we'll first
	// construct a subgraph just for this individual modules's instances and
	// then we'll steal all of its nodes and edges to incorporate into our
	// main graph which contains all of the resource instances together.
	instG, err := n.resourceInstanceSubgraph(moduleCtx, resAddr, instanceAddrs)
	if err != nil {
		diags = diags.Append(err)
		return diags.ErrWithWarnings()
	}
	g.Subsume(&instG.AcyclicGraph.Graph)

	return diags.ErrWithWarnings()
}

func (n *nodeExpandPlannableResource) resourceInstanceSubgraph(ctx EvalContext, addr addrs.AbsResource, instanceAddrs []addrs.AbsResourceInstance) (*Graph, error) {
	var diags tfdiags.Diagnostics

	// Our graph transformers require access to the full state, so we'll
	// temporarily lock it while we work on this.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResourceInstance) dag.Vertex {
		var m *NodePlannableResourceInstance

		// If we're in legacy import mode (the import CLI command), we only need
		// to return the import node, not a plannable resource node.
		if n.legacyImportMode {
			for _, importTarget := range n.importTargets {
				if importTarget.Addr.Equal(a.Addr) {

					// The import ID was supplied as a string on the command
					// line and made into a synthetic HCL expression.
					importId, diags := evaluateImportIdExpression(importTarget.ID, ctx)
					if diags.HasErrors() {
						// This should be impossible, because the import command
						// arg parsing builds the synth expression from a
						// non-null string.
						panic(fmt.Sprintf("Invalid import id: %s. This is a bug in Terraform; please report it!", diags.Err()))
					}

					return &graphNodeImportState{
						Addr:             importTarget.Addr,
						ID:               importId,
						ResolvedProvider: n.ResolvedProvider,
					}
				}
			}
		}

		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.Schema = n.Schema
		a.ProvisionerSchemas = n.ProvisionerSchemas
		a.ProviderMetas = n.ProviderMetas
		a.dependsOn = n.dependsOn
		a.Dependencies = n.dependencies
		a.preDestroyRefresh = n.preDestroyRefresh
		a.generateConfigPath = n.generateConfigPath

		m = &NodePlannableResourceInstance{
			NodeAbstractResourceInstance: a,

			// By the time we're walking, we've figured out whether we need
			// to force on CreateBeforeDestroy due to dependencies on other
			// nodes that have it.
			ForceCreateBeforeDestroy: n.CreateBeforeDestroy(),
			skipRefresh:              n.skipRefresh,
			skipPlanChanges:          n.skipPlanChanges,
			forceReplace:             n.forceReplace,
		}

		for _, importTarget := range n.importTargets {
			if importTarget.Addr.Equal(a.Addr) {
				// If we get here, we're definitely not in legacy import mode,
				// so go ahead and plan the resource changes including import.
				m.importTarget = ImportTarget{
					ID:     importTarget.ID,
					Addr:   importTarget.Addr,
					Config: importTarget.Config,
				}
			}
		}

		return m
	}

	// The concrete resource factory we'll use for orphans
	concreteResourceOrphan := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.Schema = n.Schema
		a.ProvisionerSchemas = n.ProvisionerSchemas
		a.ProviderMetas = n.ProviderMetas

		return &NodePlannableResourceInstanceOrphan{
			NodeAbstractResourceInstance: a,
			skipRefresh:                  n.skipRefresh,
			skipPlanChanges:              n.skipPlanChanges,
		}
	}

	// Start creating the steps
	steps := []GraphTransformer{
		// Expand the count or for_each (if present)
		&ResourceCountTransformer{
			Concrete:      concreteResource,
			Schema:        n.Schema,
			Addr:          n.ResourceAddr(),
			InstanceAddrs: instanceAddrs,
		},

		// Add the count/for_each orphans
		&OrphanResourceInstanceCountTransformer{
			Concrete:      concreteResourceOrphan,
			Addr:          addr,
			InstanceAddrs: instanceAddrs,
			State:         state,
		},

		// Attach the state
		&AttachStateTransformer{State: state},

		// Targeting
		&TargetsTransformer{Targets: n.Targets},

		// Connect references so ordering is correct
		&ReferenceTransformer{},

		// Make sure there is a single root
		&RootTransformer{},
	}

	// Build the graph
	b := &BasicGraphBuilder{
		Steps: steps,
		Name:  "nodeExpandPlannableResource",
	}
	graph, diags := b.Build(addr.Module)
	return graph, diags.ErrWithWarnings()
}
