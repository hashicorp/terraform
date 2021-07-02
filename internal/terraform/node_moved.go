package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandMoved is the placeholder for a moved block that hasn't yet
// had its module path expanded to take into account modules with count
// or for_each.
//
// The graph builder must ensure that any nodeExpandMoved instance in
// the graph depends on any nodes that represent whatever the "from"
// address refers to, because the validation rules involve inspecting
// the plan for that object to make sure that it's either already gone
// or planned for removal.
//
// The graph builder should also make node representing our "to" object
// depend on this node, so that move-related errors will prevent the graph
// walk from reaching the downstream object and thus trying to plan
// something nonsensical.
type nodeExpandMoved struct {
	Module addrs.Module
	Config *configs.Moved

	// AllConfig is the config object representing the root
	// module in the module tree. We need this so that we
	// can find the objects that the moved block refers to,
	// if present.
	AllConfig *configs.Config

	// Targets is the full set of target addresses included in the
	// plan options, populated by the target graph transformer when
	// it calls our SetTargets method.
	Targets []addrs.Targetable
}

var (
	_ GraphNodeDynamicExpandable = (*nodeExpandMoved)(nil)
	_ graphNodeExpandsInstances  = (*nodeExpandMoved)(nil)
	_ dag.GraphNodeDotter        = (*nodeExpandMoved)(nil)
	_ GraphNodeTargetable        = (*nodeExpandMoved)(nil)
	_ graphNodeExemptFromTarget  = (*nodeExpandMoved)(nil)
)

func newNodeExpandMoved(modAddr addrs.Module, movedConfig *configs.Moved, fullConfig *configs.Config) dag.Vertex {
	return &nodeExpandMoved{
		Module:    modAddr,
		Config:    movedConfig,
		AllConfig: fullConfig,
	}
}

func (n *nodeExpandMoved) Name() string {
	var prefix string
	if !n.Module.IsRoot() {
		prefix = n.Module.String() + ": "
	}
	return fmt.Sprintf(
		"%smoved %s -> %s (expand)",
		prefix,
		n.Config.From.String(),
		n.Config.To.String(),
	)
}

func (n *nodeExpandMoved) ModulePath() addrs.Module {
	return n.Module
}

func (n *nodeExpandMoved) expandsInstances() {}

func (n *nodeExpandMoved) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var g Graph
	expander := ctx.InstanceExpander()
	for _, module := range expander.ExpandModule(n.Module) {
		g.Add(&nodeMovedValidate{
			Module:    module,
			Config:    n.Config,
			AllConfig: n.AllConfig,
			Targets:   n.Targets,
		})
	}
	return &g, nil
}

func (n *nodeExpandMoved) MovedFromAddr() addrs.ConfigMoveable {
	return n.Config.From.ConfigMoveable(n.Module)
}

func (n *nodeExpandMoved) MovedToAddr() addrs.ConfigMoveable {
	return n.Config.To.ConfigMoveable(n.Module)
}

// SetTargets is an implementation of GraphNodeTargetable to capture the
// target addresses included in the plan options.
//
// We don't do anything with the target addresses directly in
// nodeExpandMoved, but we do pass them on to nodeMovedValidate so that
// it can avoid trying to validate against working state for resources and
// modules that the plan didn't visit due to being excluded by these.
func (n *nodeExpandMoved) SetTargets(addrs []addrs.Targetable) {
	n.Targets = addrs
}

// NodeExemptFromTarget is an implementation of graphNodeExemptFromTarget which
// exempts moved-block nodes from being removed by the -target option.
//
// We do this because the nodeMovedValidate.Execute function handles targets
// itself, in a more subtle way than just removing the entire graph node.
func (n *nodeExpandMoved) NodeExemptFromTarget() bool {
	return true
}

func (n *nodeExpandMoved) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	fromAddr := n.MovedFromAddr()
	toAddr := n.MovedToAddr()
	label := fmt.Sprintf("%s\n↓\n%s", fromAddr, toAddr)
	return &dag.DotNode{
		Name: dag.VertexName(n),
		Attrs: map[string]string{
			"shape": "cds",
			"label": label,
		},
	}
}

// nodeMovedValidate is a node type which deals with the validation
// rules for a particular "moved" block in isolation.
//
// Although this node is called "validate", it can actually achieve
// full validation only during the plan walk, because it needs to
// consider the final expanded set of instance keys for any module
// or resource that is mentioned in the move statement.
//
// It can't consider the validation rules which relate to interactions
// _between_ moved blocks. This must be handled elsewhere.
type nodeMovedValidate struct {
	Module addrs.ModuleInstance
	Config *configs.Moved

	// AllConfig is the config object representing the root
	// module in the module tree. We need this so that we
	// can find the objects that the moved block refers to,
	// if present.
	AllConfig *configs.Config

	// Targets is the full set of target addresses included in the
	// plan options. If this has non-zero length then we must skip
	// any of our validation rules which refer to the working state
	// of resources or modules _not_ included in this set, because
	// otherwise we'd refer to the prior state rather than the
	// current configuration effect.
	Targets []addrs.Targetable
}

var (
	_ GraphNodeExecutable     = (*nodeMovedValidate)(nil)
	_ GraphNodeModuleInstance = (*nodeMovedValidate)(nil)
)

func (n *nodeMovedValidate) Name() string {
	var prefix string
	if !n.Module.IsRoot() {
		prefix = n.Module.String() + ": "
	}
	return fmt.Sprintf(
		"%smoved %s -> %s",
		prefix,
		n.Config.From.String(),
		n.Config.To.String(),
	)
}

func (n *nodeMovedValidate) Path() addrs.ModuleInstance {
	return n.Module
}

func (n *nodeMovedValidate) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	if op != walkPlan {
		// We use the planned new state for the "from" object as a
		// proxy to understand the effect of its configuration, if any,
		// and so the logic below can't produce a correct result
		// for any other walk.
		return nil
	}
	// Note: this logic assumes that the nodeMovedValidate for a particular
	// moved block will execute only after we've created the plan for the
	// "from" endpoint. We have no way to check or enforce that here, so
	// it's up to the graph builder to create the necessary dependency
	// edges with our associated nodeExpandMoved so that'll be true.
	// Otherwise, the following will produce an incorrect result.

	var diags tfdiags.Diagnostics
	fromAddrRange := n.Config.From.SourceRange.ToHCL()
	toAddrRange := n.Config.To.SourceRange.ToHCL()

	const incompatibleEndpoints = "Incompatible move endpoints"
	from, to := addrs.UnifyMoveEndpoints(n.Module, n.Config.From, n.Config.To)
	if from == nil || to == nil {
		// This means that the two addresses are not of compatible kinds.
		// We'll somewhat-arbitrarily blame the "to" address for being
		// wrong here.
		configFrom := n.Config.From.ConfigMoveable(n.Module.Module())
		switch configFrom.(type) {
		case addrs.Module:
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  incompatibleEndpoints,
				Detail: fmt.Sprintf(
					"Moving from %s to %s is impossible: when the source object is a module, the destination object must also be a module.",
					n.Config.From.String(), n.Config.To.String(),
				),
				Subject: toAddrRange.Ptr(),
			})
		case addrs.ConfigResource:
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  incompatibleEndpoints,
				Detail: fmt.Sprintf(
					"Moving from %s to %s is impossible: when the source object is a resource, the destination object must also be a resource.",
					n.Config.From.String(), n.Config.To.String(),
				),
				Subject: toAddrRange.Ptr(),
			})
		default:
			// Weird to get here, because the above types should be exhaustive
			// for addrs.ConfigMoveable, but we'll return a generic error to
			// be robust.
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  incompatibleEndpoints,
				Detail: fmt.Sprintf(
					"Moving from %s to %s is impossible: the source and destination objects must be of the same kind.",
					n.Config.From.String(), n.Config.To.String(),
				),
				Subject: toAddrRange.Ptr(),
			})
		}
	}

	// From here on we're assuming that UnifyMoveEndpoints honors its contract
	// to always give both "from" and "to" the same dynamic address type.

	if from.String() == to.String() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid declaration of moved object",
			Detail: fmt.Sprintf(
				"Cannot move from %s to itself: the source and destination addresses must be different.",
				from.String(),
			),
			Subject: toAddrRange.Ptr(),
		})
	}

	var fromMode, toMode addrs.ResourceMode
	switch from := from.(type) {
	case addrs.AbsResource:
		to := to.(addrs.AbsResource)
		fromMode = from.Resource.Mode
		toMode = to.Resource.Mode
	case addrs.AbsResourceInstance:
		to := to.(addrs.AbsResourceInstance)
		fromMode = from.Resource.Resource.Mode
		toMode = to.Resource.Resource.Mode
	default:
		// We only care about resource references for this particular
		// rule. For others we'll leave the modes set to zero value
		// and thus equal to one another.
	}
	if fromMode != toMode {
		switch {
		case fromMode == addrs.ManagedResourceMode && toMode == addrs.DataResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  incompatibleEndpoints,
				Detail: fmt.Sprintf(
					"Moving from %s to %s is impossible: can't move from managed resource to data resource.",
					n.Config.From.String(), n.Config.To.String(),
				),
				Subject: toAddrRange.Ptr(),
			})
		case fromMode == addrs.DataResourceMode && toMode == addrs.ManagedResourceMode:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  incompatibleEndpoints,
				Detail: fmt.Sprintf(
					"Moving from %s to %s is impossible: can't move from data resource to managed resource.",
					n.Config.From.String(), n.Config.To.String(),
				),
				Subject: toAddrRange.Ptr(),
			})
		default:
			// Fallback error in case we add new resource modes in future and
			// neglect to update this. However, this is a poor error message
			// so better to avoid this if possible.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  incompatibleEndpoints,
				Detail: fmt.Sprintf(
					"Moving from %s to %s is impossible: source and destination objects must have the same resource mode.",
					n.Config.From.String(), n.Config.To.String(),
				),
				Subject: toAddrRange.Ptr(),
			})
		}
	}

	// A "moved" is invalid if its from address refers to an object that's
	// still in the configuration. Since earlier planning steps already did
	// the work of analyzing the configuration and summarizing it in the
	// planned new state, we can refer to what was planned rather than
	// referring directly to the configuration for situations where we need
	// to check individual module/resource instances, and thus need to
	// refer to the result of processing count and for_each.
	//
	// (Note: our validation rules are defined in terms of configuration only,
	// so it may seem contradictory to be referring to state here. The crucial
	// detail is that we're referring to the _working state_, which captures
	// how we're responding to what's in the configuration, and not to the
	// prior state which captures the situation before we take any actions
	// at all.)
	//
	// The exact details of this depend on what kind of object we're checking,
	// but in any case we want to make sure the state doesn't change out from
	// under us while we're working.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()
	config := n.AllConfig
	const moveSourceExists = "Move source still exists"
	const cannotValidateMove = "Cannot validate moved block"
	switch from := from.(type) {
	case addrs.AbsResource:
		// We need to refer to the configuration directly for this case,
		// because the resource state alone can't distinguish between a
		// resource block with count = 0 vs. a resource block not existing
		// at all; the move is only valid if the block was removed entirely.
		if rc := config.ResourceInModuleInstance(from); rc != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  moveSourceExists,
				Detail: fmt.Sprintf(
					"Can't declare move from %s while it's still declared in configuration. Move or rename the block at %s so it will represent %s instead.",
					from.String(), rc.DeclRange.String(), to.String(),
				),
				Subject: fromAddrRange.Ptr(),
			})
		}
	case addrs.AbsResourceInstance:
		// When moving from a resource instance it's okay for the resource
		// block to still exist as long as its count or for_each arguments
		// no longer declare a matching key. To avoid re-evaluating count
		// and for_each here, we'll just refer to the working state.
		if !n.resourceInstanceIncludedInTargets(from) {
			// It's not safe for us to refer to the working state if the
			// instance we're checking was excluded by targeting.
			// TODO: This can hopefully become an error once we've actually
			// implemented the step of acting on the moves in the state, which
			// should allow us to determine whether this particular moved block
			// was acted upon in this particular plan. In that case, the
			// error would be something like "you must include X in your
			// targets because it's subject to a moved block".
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  cannotValidateMove,
				Detail: fmt.Sprintf(
					"Your -target options have excluded %s, so Terraform cannot use its result to validate this moved block.\n\nThe result of this plan may be incorrect if this move was handled as part of the current plan.",
					from.String(),
				),
				Subject: fromAddrRange.Ptr(),
			})
			break
		}
		if rs := state.ResourceInstance(from); rs != nil && rs.Current != nil {
			// We'll tailor our error message depending on whether the "from"
			// resource looks like it was declared by count, for_each, or
			// neither.
			switch key := from.Resource.Key.(type) {
			case nil: // single-instance resource
				to := to.(addrs.AbsResourceInstance)
				if from.Module.Equal(to.Module) && from.Resource.Resource == to.Resource.Resource {
					// If the "to" address is for the same resource as "from"
					// then that suggests that the user intent was to switch to
					// using count or for_each, but they haven't done it yet.
					argName := "repetition" // unhelpful generic name hopefully overridden below
					switch to.Resource.Key.(type) {
					case addrs.IntKey:
						argName = `"count"`
					case addrs.StringKey:
						argName = `"for_each"`
					}
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  moveSourceExists,
						Detail: fmt.Sprintf(
							"Can't declare move from %s while it's still declared in configuration. Did you intend to add a %s argument to this resource?",
							from.String(), argName,
						),
						Subject: fromAddrRange.Ptr(),
					})
				} else {
					// Otherwise, a less specific error.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  moveSourceExists,
						Detail: fmt.Sprintf(
							"Can't declare move from %s while it's still declared in configuration. Either remove the resource block or add a \"count\" or \"for_each\" argument to declare multiple instances.",
							from.String(),
						),
						Subject: fromAddrRange.Ptr(),
					})
				}
			case addrs.IntKey: // from a "count" resource
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  moveSourceExists,
					Detail: fmt.Sprintf(
						"Can't declare move from %s while it's still declared in configuration. Either remove the resource block or change the \"count\" argument to be less than %d.",
						from.String(), int(key)+1,
					),
					Subject: fromAddrRange.Ptr(),
				})
			case addrs.StringKey: // from a "for_each" resource
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  moveSourceExists,
					Detail: fmt.Sprintf(
						"Can't declare move from %s while it's still declared in configuration. Either remove the resource block or change the \"for_each\" argument to no longer declare the key %q.",
						from.String(), key,
					),
					Subject: fromAddrRange.Ptr(),
				})
			default: // perhaps a new key type was added later and we neglected to update this?
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  moveSourceExists,
					Detail: fmt.Sprintf(
						"Can't declare move from %s while it's still declared in configuration. Either remove the resource block or change its repetition argument to no longer declare this instance.",
						from.String(),
					),
					Subject: fromAddrRange.Ptr(),
				})
			}
		}
	case addrs.AbsModuleCall:
		// We need to refer to the configuration directly for this case,
		// because our working state doesn't reliably distinguish between
		// a module with no resources and no module at all.
		if callerCfg := config.DescendentForInstance(from.Module); callerCfg != nil {
			if callCfg := callerCfg.Module.ModuleCalls[from.Call.Name]; callCfg != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  moveSourceExists,
					Detail: fmt.Sprintf(
						"Can't declare move from %s while it's still declared in configuration. Move or rename the block at %s so it will represent %s instead.",
						from.String(), callCfg.DeclRange.String(), to.String(),
					),
					Subject: fromAddrRange.Ptr(),
				})
			}
		}
	case addrs.ModuleInstance:
		// This one is a tricky case because the state doesn't accurately
		// represent all of the module instances in the configuration: it'll
		// skip any that don't have any current resource instances inside
		// them. That means we won't necessarily reject a move from an instance
		// of a hypothetical module that doesn't have anything declared inside
		// of it, but we'll accept that as an okay edge case because moving
		// a totally-empty module doesn't have any significant effect anyway.
		if !n.moduleInstanceIncludedInTargets(from) {
			// It's not safe for us to refer to the working state if the
			// instance we're checking was excluded by targeting.
			// TODO: This can hopefully become an error once we've actually
			// implemented the step of acting on the moves in the state, which
			// should allow us to determine whether this particular moved block
			// was acted upon in this particular plan. In that case, the
			// error would be something like "you must include X in your
			// targets because it's subject to a moved block".
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  cannotValidateMove,
				Detail: fmt.Sprintf(
					"Your -target options have excluded %s, so Terraform cannot use its result to validate this moved block.\n\nThe result of this plan may be incorrect if this move was handled as part of the current plan.",
					from.String(),
				),
				Subject: fromAddrRange.Ptr(),
			})
			break
		}
		if ms := state.Module(from); ms != nil {
			// We'll tailor our error message depending on whether the "from"
			// module looks like it was declared by count, for_each, or
			// neither.
			switch key := from[len(from)-1].InstanceKey.(type) {
			case nil: // single-instance module
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  moveSourceExists,
					Detail: fmt.Sprintf(
						"Can't declare move from %s while it's still declared in configuration. Either remove the module block or add a \"count\" or \"for_each\" argument to declare multiple instances.",
						from.String(),
					),
					Subject: fromAddrRange.Ptr(),
				})
			case addrs.IntKey: // from a "count" module
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  moveSourceExists,
					Detail: fmt.Sprintf(
						"Can't declare move from %s while it's still declared in configuration. Either remove the module block or change the \"count\" argument to no longer declare index %d.",
						from.String(), key,
					),
					Subject: fromAddrRange.Ptr(),
				})
			case addrs.StringKey: // from a "for_each" module
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  moveSourceExists,
					Detail: fmt.Sprintf(
						"Can't declare move from %s while it's still declared in configuration. Either remove the module block or change the \"for_each\" argument to no longer declare the key %q.",
						from.String(), key,
					),
					Subject: fromAddrRange.Ptr(),
				})
			default: // perhaps a new key type was added later and we neglected to update this?
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  moveSourceExists,
					Detail: fmt.Sprintf(
						"Can't declare move from %s while it's still declared in configuration. Either remove the module block or change its repetition argument to no longer declare this instance.",
						from.String(),
					),
					Subject: fromAddrRange.Ptr(),
				})
			}
		}
	}

	return diags
}

func (n *nodeMovedValidate) resourceInstanceIncludedInTargets(addr addrs.AbsResourceInstance) bool {
	if len(n.Targets) == 0 {
		// If we have no targets then everything's selected
		return true
	}

	for _, targetAddr := range n.Targets {
		if targetAddr.TargetContains(addr) {
			return true
		}
	}
	return false
}

func (n *nodeMovedValidate) moduleInstanceIncludedInTargets(addr addrs.ModuleInstance) bool {
	if len(n.Targets) == 0 {
		// If we have no targets then everything's selected
		return true
	}

	for _, targetAddr := range n.Targets {
		if targetAddr.TargetContains(addr) {
			return true
		}
	}
	return false
}
