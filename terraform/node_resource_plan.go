package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandPlannableResource handles the first layer of resource
// expansion.  We need this extra layer so DynamicExpand is called twice for
// the resource, the first to expand the Resource for each module instance, and
// the second to expand each ResourceInstance for the expanded Resources.
type nodeExpandPlannableResource struct {
	*NodeAbstractResource

	// ForceCreateBeforeDestroy might be set via our GraphNodeDestroyerCBD
	// during graph construction, if dependencies require us to force this
	// on regardless of what the configuration says.
	ForceCreateBeforeDestroy *bool

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

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
	dependencies []addrs.ConfigResource
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
)

func (n *nodeExpandPlannableResource) Name() string {
	return n.NodeAbstractResource.Name() + " (expand)"
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

	// Add the current expanded resource to the graph
	for _, module := range moduleInstances {
		resAddr := n.Addr.Resource.Absolute(module)
		g.Add(&NodePlannableResource{
			NodeAbstractResource:     n.NodeAbstractResource,
			Addr:                     resAddr,
			ForceCreateBeforeDestroy: n.ForceCreateBeforeDestroy,
			dependencies:             n.dependencies,
			skipRefresh:              n.skipRefresh,
			skipPlanChanges:          n.skipPlanChanges,
			forceReplace:             n.forceReplace,
		})
	}

	// Lock the state while we inspect it
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	var orphans []*states.Resource
	for _, res := range state.Resources(n.Addr) {
		found := false
		for _, m := range moduleInstances {
			if m.Equal(res.Addr.Module) {
				found = true
				break
			}
		}
		// Address form state was not found in the current config
		if !found {
			orphans = append(orphans, res)
		}
	}

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

	return &g, nil
}

// NodePlannableResource represents a resource that is "plannable":
// it is ready to be planned in order to create a diff.
type NodePlannableResource struct {
	*NodeAbstractResource

	Addr addrs.AbsResource

	// ForceCreateBeforeDestroy might be set via our GraphNodeDestroyerCBD
	// during graph construction, if dependencies require us to force this
	// on regardless of what the configuration says.
	ForceCreateBeforeDestroy *bool

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool

	// forceReplace are resource instance addresses where the user wants to
	// force generating a replace action. This set isn't pre-filtered, so
	// it might contain addresses that have nothing to do with the resource
	// that this node represents, which the node itself must therefore ignore.
	forceReplace []addrs.AbsResourceInstance

	dependencies []addrs.ConfigResource
}

var (
	_ GraphNodeModuleInstance       = (*NodePlannableResource)(nil)
	_ GraphNodeDestroyerCBD         = (*NodePlannableResource)(nil)
	_ GraphNodeDynamicExpandable    = (*NodePlannableResource)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResource)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResource)(nil)
	_ GraphNodeConfigResource       = (*NodePlannableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResource)(nil)
)

func (n *NodePlannableResource) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

func (n *NodePlannableResource) Name() string {
	return n.Addr.String()
}

// GraphNodeExecutable
func (n *NodePlannableResource) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if n.Config == nil {
		// Nothing to do, then.
		log.Printf("[TRACE] NodeApplyableResource: no configuration present for %s", n.Name())
		return diags
	}

	// writeResourceState is responsible for informing the expander of what
	// repetition mode this resource has, which allows expander.ExpandResource
	// to work below.
	moreDiags := n.writeResourceState(ctx, n.Addr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// Before we expand our resource into potentially many resource instances,
	// we'll verify that any mention of this resource in n.forceReplace is
	// consistent with the repetition mode of the resource. In other words,
	// we're aiming to catch a situation where naming a particular resource
	// instance would require an instance key but the given address has none.
	expander := ctx.InstanceExpander()
	instanceAddrs := expander.ExpandResource(n.ResourceAddr().Absolute(ctx.Path()))

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
	// actions is in NodeAbstractResourceInstance.plan, because we must do so
	// on a per-instance basis rather than for the whole resource.

	return diags
}

// GraphNodeDestroyerCBD
func (n *NodePlannableResource) CreateBeforeDestroy() bool {
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
func (n *NodePlannableResource) ModifyCreateBeforeDestroy(v bool) error {
	n.ForceCreateBeforeDestroy = &v
	return nil
}

// GraphNodeDynamicExpandable
func (n *NodePlannableResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var diags tfdiags.Diagnostics

	// We need to potentially rename an instance address in the state
	// if we're transitioning whether "count" is set at all.
	fixResourceCountSetTransition(ctx, n.Addr.Config(), n.Config.Count != nil)

	// Our instance expander should already have been informed about the
	// expansion of this resource and of all of its containing modules, so
	// it can tell us which instance addresses we need to process.
	expander := ctx.InstanceExpander()
	instanceAddrs := expander.ExpandResource(n.ResourceAddr().Absolute(ctx.Path()))

	// Our graph transformers require access to the full state, so we'll
	// temporarily lock it while we work on this.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	// The concrete resource factory we'll use
	concreteResource := func(a *NodeAbstractResourceInstance) dag.Vertex {
		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.Schema = n.Schema
		a.ProvisionerSchemas = n.ProvisionerSchemas
		a.ProviderMetas = n.ProviderMetas
		a.dependsOn = n.dependsOn
		a.Dependencies = n.dependencies

		return &NodePlannableResourceInstance{
			NodeAbstractResourceInstance: a,

			// By the time we're walking, we've figured out whether we need
			// to force on CreateBeforeDestroy due to dependencies on other
			// nodes that have it.
			ForceCreateBeforeDestroy: n.CreateBeforeDestroy(),
			skipRefresh:              n.skipRefresh,
			skipPlanChanges:          n.skipPlanChanges,
			forceReplace:             n.forceReplace,
		}
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
			Addr:          n.Addr,
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
		Steps:    steps,
		Validate: true,
		Name:     "NodePlannableResource",
	}
	graph, diags := b.Build(ctx.Path())
	return graph, diags.ErrWithWarnings()
}
