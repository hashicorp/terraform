package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
)

type ConcreteModuleNodeFunc func(n *nodeExpandModule) dag.Vertex

// nodeExpandModule represents a module call in the configuration that
// might expand into multiple module instances depending on how it is
// configured.
type nodeExpandModule struct {
	Addr       addrs.Module
	Config     *configs.Module
	ModuleCall *configs.ModuleCall
}

var (
	_ RemovableIfNotTargeted    = (*nodeExpandModule)(nil)
	_ GraphNodeEvalable         = (*nodeExpandModule)(nil)
	_ GraphNodeReferencer       = (*nodeExpandModule)(nil)
	_ GraphNodeReferenceOutside = (*nodeExpandModule)(nil)

	// modules both record their expansion, and require expansion from their
	// parent modules.
	_ requiresInstanceExpansion = (*nodeExpandModule)(nil)
	_ instanceExpander          = (*nodeExpandModule)(nil)
)

// requiresInstanceExpansion implementation
func (n *nodeExpandModule) requiresExpansion() {}

// instanceExander implementation
func (n *nodeExpandModule) expandsInstances() addrs.Module {
	return n.Addr
}

func (n *nodeExpandModule) Name() string {
	return n.Addr.String() + " (expand)"
}

// GraphNodeModulePath implementation
func (n *nodeExpandModule) ModulePath() addrs.Module {
	// This node represents the module call within a module,
	// so return the CallerAddr as the path as the module
	// call may expand into multiple child instances
	return n.Addr
}

// GraphNodeReferencer implementation
func (n *nodeExpandModule) References() []*addrs.Reference {
	var refs []*addrs.Reference

	if n.ModuleCall == nil {
		return nil
	}

	for _, traversal := range n.ModuleCall.DependsOn {
		ref, diags := addrs.ParseRef(traversal)
		if diags.HasErrors() {
			// We ignore this here, because this isn't a suitable place to return
			// errors. This situation should be caught and rejected during
			// validation.
			log.Printf("[ERROR] Can't parse %#v from depends_on as reference: %s", traversal, diags.Err())
			continue
		}

		refs = append(refs, ref)
	}

	// Expansion only uses the count and for_each expressions, so this
	// particular graph node only refers to those.
	// Individual variable values in the module call definition might also
	// refer to other objects, but that's handled by
	// NodeApplyableModuleVariable.
	//
	// Because our Path method returns the module instance that contains
	// our call, these references will be correctly interpreted as being
	// in the calling module's namespace, not the namespaces of any of the
	// child module instances we might expand to during our evaluation.

	if n.ModuleCall.Count != nil {
		countRefs, _ := lang.ReferencesInExpr(n.ModuleCall.Count)
		refs = append(refs, countRefs...)
	}
	if n.ModuleCall.ForEach != nil {
		forEachRefs, _ := lang.ReferencesInExpr(n.ModuleCall.ForEach)
		refs = append(refs, forEachRefs...)
	}
	return appendResourceDestroyReferences(refs)
}

// GraphNodeReferenceOutside
func (n *nodeExpandModule) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	return n.Addr, n.Addr.Parent()
}

// RemovableIfNotTargeted implementation
func (n *nodeExpandModule) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeEvalable
func (n *nodeExpandModule) EvalTree() EvalNode {
	return &evalPrepareModuleExpansion{
		Addr:       n.Addr,
		Config:     n.Config,
		ModuleCall: n.ModuleCall,
	}
}

// nodeCloseModule represents an expanded module during apply, and is visited
// after all other module instance nodes. This node will depend on all module
// instance resource and outputs, and anything depending on the module should
// wait on this node.
// Besides providing a root node for dependency ordering, nodeCloseModule also
// cleans up state after all the module nodes have been evaluated, removing
// empty resources and modules from the state.
type nodeCloseModule struct {
	Addr addrs.Module
}

var (
	_ GraphNodeReferenceable    = (*nodeCloseModule)(nil)
	_ GraphNodeReferenceOutside = (*nodeCloseModule)(nil)
)

func (n *nodeCloseModule) ModulePath() addrs.Module {
	return n.Addr
}

func (n *nodeCloseModule) ReferenceOutside() (selfPath, referencePath addrs.Module) {
	return n.Addr.Parent(), n.Addr
}

func (n *nodeCloseModule) ReferenceableAddrs() []addrs.Referenceable {
	_, call := n.Addr.Call()
	return []addrs.Referenceable{
		call,
	}
}

func (n *nodeCloseModule) Name() string {
	if len(n.Addr) == 0 {
		return "root"
	}
	return n.Addr.String() + " (close)"
}

// RemovableIfNotTargeted implementation
func (n *nodeCloseModule) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

func (n *nodeCloseModule) EvalTree() EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalOpFilter{
				Ops: []walkOperation{walkApply, walkDestroy},
				Node: &evalCloseModule{
					Addr: n.Addr,
				},
			},
		},
	}
}

type evalCloseModule struct {
	Addr addrs.Module
}

func (n *evalCloseModule) Eval(ctx EvalContext) (interface{}, error) {
	// We need the full, locked state, because SyncState does not provide a way to
	// transact over multiple module instances at the moment.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	for modKey, mod := range state.Modules {
		if !n.Addr.Equal(mod.Addr.Module()) {
			continue
		}

		// clean out any empty resources
		for resKey, res := range mod.Resources {
			if len(res.Instances) == 0 {
				delete(mod.Resources, resKey)
			}
		}

		// empty child modules are always removed
		if len(mod.Resources) == 0 && !mod.Addr.IsRoot() {
			delete(state.Modules, modKey)
		}
	}
	return nil, nil
}

// evalPrepareModuleExpansion is an EvalNode implementation
// that sets the count or for_each on the instance expander
type evalPrepareModuleExpansion struct {
	Addr       addrs.Module
	Config     *configs.Module
	ModuleCall *configs.ModuleCall
}

func (n *evalPrepareModuleExpansion) Eval(ctx EvalContext) (interface{}, error) {
	expander := ctx.InstanceExpander()
	_, call := n.Addr.Call()

	// nodeExpandModule itself does not have visibility into how its ancestors
	// were expanded, so we use the expander here to provide all possible paths
	// to our module, and register module instances with each of them.
	for _, module := range expander.ExpandModule(n.Addr.Parent()) {
		ctx = ctx.WithPath(module)

		switch {
		case n.ModuleCall.Count != nil:
			count, diags := evaluateCountExpression(n.ModuleCall.Count, ctx)
			if diags.HasErrors() {
				return nil, diags.Err()
			}
			expander.SetModuleCount(module, call, count)

		case n.ModuleCall.ForEach != nil:
			forEach, diags := evaluateForEachExpression(n.ModuleCall.ForEach, ctx)
			if diags.HasErrors() {
				return nil, diags.Err()
			}
			expander.SetModuleForEach(module, call, forEach)

		default:
			expander.SetModuleSingle(module, call)
		}
	}

	return nil, nil
}

// nodeValidateModule wraps a nodeExpand module for validation, ensuring that
// no expansion is attempted during evaluation, when count and for_each
// expressions may not be known.
type nodeValidateModule struct {
	nodeExpandModule
}

// GraphNodeEvalable
func (n *nodeValidateModule) EvalTree() EvalNode {
	return &evalValidateModule{
		Addr:       n.Addr,
		Config:     n.Config,
		ModuleCall: n.ModuleCall,
	}
}

type evalValidateModule struct {
	Addr       addrs.Module
	Config     *configs.Module
	ModuleCall *configs.ModuleCall
}

func (n *evalValidateModule) Eval(ctx EvalContext) (interface{}, error) {
	_, call := n.Addr.Call()
	expander := ctx.InstanceExpander()

	// Modules all evaluate to single instances during validation, only to
	// create a proper context within which to evaluate. All parent modules
	// will be a single instance, but still get our address in the expected
	// manner anyway to ensure they've been registered correctly.
	for _, module := range expander.ExpandModule(n.Addr.Parent()) {
		ctx = ctx.WithPath(module)

		// Validate our for_each and count expressions at a basic level
		// We skip validation on known, because there will be unknown values before
		// a full expansion, presuming these errors will be caught in later steps
		switch {
		case n.ModuleCall.Count != nil:
			_, diags := evaluateCountExpressionValue(n.ModuleCall.Count, ctx)
			if diags.HasErrors() {
				return nil, diags.Err()
			}

		case n.ModuleCall.ForEach != nil:
			_, diags := evaluateForEachExpressionValue(n.ModuleCall.ForEach, ctx)
			if diags.HasErrors() {
				return nil, diags.Err()
			}
		}

		// now set our own mode to single
		expander.SetModuleSingle(module, call)
	}
	return nil, nil
}
