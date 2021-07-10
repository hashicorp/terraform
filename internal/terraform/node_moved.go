package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeModuleMovesExpand is a graph node representing all of the "moved" blocks
// declared in a particular module.
//
// This node type expands to a dynamic subgraph containing one node for each
// moved block in each instance of the module.
//
// Nodes of this type should be included only in a plan graph. The move rules
// interact with other actions taken during planning, so cannot be handled
// in any other walk.
type nodeMovedExpand struct {
	Module      addrs.Module
	MoveConfigs []*configs.Moved

	// AllConfig is the config object containing the root module, which we
	// need for some of our pre-validation rules.
	AllConfig *configs.Config

	Targets []addrs.Targetable
}

var (
	_ graphNodeMovedModule       = (*nodeMovedExpand)(nil)
	_ GraphNodeDynamicExpandable = (*nodeMovedExpand)(nil)
)

func (n *nodeMovedExpand) FromConfigAddrs() []addrs.ConfigMoveable {
	if len(n.MoveConfigs) == 0 {
		// Weird, cause we shouldn've have created the node at all in that
		// case, but... fine!
		return nil
	}

	ret := make([]addrs.ConfigMoveable, len(n.MoveConfigs))
	for i, mc := range n.MoveConfigs {
		ret[i] = mc.From.ConfigMoveable(n.Module)
	}
	return ret
}

func (n *nodeMovedExpand) DynamicExpand(ctx EvalContext) (*Graph, error) {
	var diags tfdiags.Diagnostics

	// This DynamicExpand is expanding two levels at once: per moved
	// configuration and per instance of the module they are all declared in.
	// This allows us to detect cyclic move chains. We only need to check
	// cycles for configurations in one module at a time because

	var g Graph

	// Some temporary data structures to support our analysis and validation.
	var nodes []*nodeMovedInstance
	nodesFromResource := map[string][]*nodeMovedInstance{}
	nodesFromResourceInstance := map[string][]*nodeMovedInstance{}
	nodesFromModuleCall := map[string][]*nodeMovedInstance{}
	nodesFromModuleInstance := map[string][]*nodeMovedInstance{}
	nodesToResource := map[string][]*nodeMovedInstance{}
	nodesToResourceInstance := map[string][]*nodeMovedInstance{}
	nodesToModuleCall := map[string][]*nodeMovedInstance{}
	nodesToModuleInstance := map[string][]*nodeMovedInstance{}

	expander := ctx.InstanceExpander()
MovedConfigs:
	for _, mc := range n.MoveConfigs {
		for _, moduleAddr := range expander.ExpandModule(n.Module) {
			from, to := addrs.UnifyMoveEndpoints(moduleAddr, mc.From, mc.To)
			if from == nil || to == nil {
				// Indicates that the two endpoints are not of compatible kinds.
				//
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Incompatible move addresses",
					Detail: fmt.Sprintf(
						"Cannot move from %s to %s: source and destination must either both be resources or both be modules.",
						mc.From.String(), mc.To.String(),
					),
					// We'll somewhat-arbitrarily blame the "to" address as being
					// the one that's wrong.
					Subject: mc.To.SourceRange.ToHCL().Ptr(),
				})

				// Unification failure never depends on the given module address,
				// so we'll stop processing all of the other instances of this
				// particular block to avoid emitting the same message multiple
				// times.
				continue MovedConfigs
			}

			n := &nodeMovedInstance{
				From: from,
				To:   to,

				DeclRange: tfdiags.SourceRangeFromHCL(mc.DeclRange),
				FromRange: mc.From.SourceRange,
				ToRange:   mc.To.SourceRange,

				AllConfig: n.AllConfig,
			}

			nodes = append(nodes, n)
			fromString := n.From.String()
			switch addr := n.From.(type) {
			case addrs.AbsResource:
				nodesFromResource[fromString] = append(nodesFromResource[fromString], n)
			case addrs.AbsResourceInstance:
				nodesFromResourceInstance[fromString] = append(nodesFromResourceInstance[fromString], n)
			case addrs.AbsModuleCall:
				nodesFromModuleCall[fromString] = append(nodesFromModuleCall[fromString], n)
			case addrs.ModuleInstance:
				nodesFromModuleInstance[fromString] = append(nodesFromModuleInstance[fromString], n)
			default:
				panic(fmt.Sprintf("unsupported addrs.AbsMovable %T", addr))
			}
			switch addr := n.To.(type) {
			case addrs.AbsResource:
				nodesToResource[fromString] = append(nodesToResource[fromString], n)
			case addrs.AbsResourceInstance:
				nodesToResourceInstance[fromString] = append(nodesToResourceInstance[fromString], n)
			case addrs.AbsModuleCall:
				nodesToModuleCall[fromString] = append(nodesToModuleCall[fromString], n)
			case addrs.ModuleInstance:
				nodesToModuleInstance[fromString] = append(nodesToModuleInstance[fromString], n)
			default:
				panic(fmt.Sprintf("unsupported addrs.AbsMovable %T", addr))
			}

			g.Add(n)
		}
	}

	// If the "from" address of one node matches or is contained within the
	// "to" address of another then that represents a move chain and possibly
	// also a nested move, and in both cases we need to ensure that the
	// "to" gets processed first so there's something for the "from" to
	// access.
	for _, successor := range nodes {
		// We must work our way "outwards" from the given address through
		// all of the possible containers it could belong to, checking
		// each one in turn.
		for nextAddr := successor.From; nextAddr != nil; {
			switch thisAddr := nextAddr.(type) {
			case addrs.AbsResourceInstance:
				if predecessor := nodesToResourceInstance[thisAddr.String()]; predecessor != nil {
					g.Connect(dag.BasicEdge(successor, predecessor))
				}
				nextAddr = thisAddr.ContainingResource()
			case addrs.AbsResource:
				if predecessor := nodesToResource[thisAddr.String()]; predecessor != nil {
					g.Connect(dag.BasicEdge(successor, predecessor))
				}
				nextAddr = thisAddr.Module
			case addrs.ModuleInstance:
				if predecessor := nodesToModuleInstance[thisAddr.String()]; predecessor != nil {
					g.Connect(dag.BasicEdge(successor, predecessor))
				}
				callerAddr, callAddr := thisAddr.Call()
				nextAddr = addrs.AbsModuleCall{
					Module: callerAddr,
					Call:   callAddr,
				}
			case addrs.AbsModuleCall:
				if predecessor := nodesToModuleCall[thisAddr.String()]; predecessor != nil {
					g.Connect(dag.BasicEdge(successor, predecessor))
				}
				if len(thisAddr.Module) > 0 {
					// Note: this will return to the addrs.ModuleInstance case
					// on the next iteration, flipping back and forth between
					// these two until we've reached a call in the root module.
					nextAddr = thisAddr.Module
				} else {
					// thisAddr is a call in the root module, so we're done
					nextAddr = nil
				}
			default:
				panic(fmt.Sprintf("unsupported addrs.AbsMovable %T", nextAddr))
			}
		}
	}

	// We need to run the targets transformer to exclude any specific instances
	// that aren't covered by the specified targets, if any, because the
	// top-level graph was only accurate enough to deal with whole
	// modules calls or resources.
	tt := &TargetsTransformer{
		Targets: n.Targets,
	}
	err := tt.Transform(&g)
	diags = diags.Append(err)

	return &g, diags.Err()
}

type nodeMovedInstance struct {
	From, To addrs.AbsMoveable

	DeclRange, FromRange, ToRange tfdiags.SourceRange

	AllConfig *configs.Config
}

var (
	_ graphNodeMovedSingle = (*nodeMovedInstance)(nil)
)

// graphNodeMovedModule is an interface implemented by nodes that represent
// all "moved" block defined inside a particular module, prior to expansion.
type graphNodeMovedModule interface {
	FromConfigAddrs() []addrs.ConfigMoveable
}

// graphNodeMovedSingle is an interface implemented by nodes that represent
// individual "moved" blocks after they are fully expanded.
type graphNodeMovedSingle interface {
	FromConfigAddr() addrs.AbsMoveable
}
