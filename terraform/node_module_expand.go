package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/states"
)

// nodeExpandModule represents a module call in the configuration that
// might expand into multiple module instances depending on how it is
// configured.
type nodeExpandModule struct {
	CallerAddr addrs.ModuleInstance
	Addr       addrs.Module
	Call       addrs.ModuleCall
	Config     *configs.Module
	ModuleCall *configs.ModuleCall
}

var (
	_ GraphNodeSubPath       = (*nodeExpandModule)(nil)
	_ RemovableIfNotTargeted = (*nodeExpandModule)(nil)
	_ GraphNodeEvalable      = (*nodeExpandModule)(nil)
	_ GraphNodeReferencer    = (*nodeExpandModule)(nil)
)

func (n *nodeExpandModule) Name() string {
	return n.CallerAddr.Child(n.Call.Name, addrs.NoKey).String()
}

// GraphNodeSubPath implementation
func (n *nodeExpandModule) Path() addrs.ModuleInstance {
	// This node represents the module call within a module,
	// so return the CallerAddr as the path as the module
	// call may expand into multiple child instances
	return n.CallerAddr
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
		refs, _ = lang.ReferencesInExpr(n.ModuleCall.Count)
	}
	if n.ModuleCall.ForEach != nil {
		refs, _ = lang.ReferencesInExpr(n.ModuleCall.ForEach)
	}
	return appendResourceDestroyReferences(refs)
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
		CallerAddr: n.CallerAddr,
		Call:       n.Call,
		Config:     n.Config,
		ModuleCall: n.ModuleCall,
	}
}

// evalPrepareModuleExpansion is an EvalNode implementation
// that sets the count or for_each on the instance expander
type evalPrepareModuleExpansion struct {
	CallerAddr addrs.ModuleInstance
	Call       addrs.ModuleCall
	Config     *configs.Module
	ModuleCall *configs.ModuleCall
}

func (n *evalPrepareModuleExpansion) Eval(ctx EvalContext) (interface{}, error) {
	eachMode := states.NoEach
	expander := ctx.InstanceExpander()

	if n.ModuleCall == nil {
		// FIXME: should we have gotten here with no module call?
		log.Printf("[TRACE] evalPrepareModuleExpansion: %s is a singleton", n.CallerAddr.Child(n.Call.Name, addrs.NoKey))
		expander.SetModuleSingle(n.CallerAddr, n.Call)
		return nil, nil
	}

	count, countDiags := evaluateResourceCountExpression(n.ModuleCall.Count, ctx)
	if countDiags.HasErrors() {
		return nil, countDiags.Err()
	}

	if count >= 0 { // -1 signals "count not set"
		eachMode = states.EachList
	}

	forEach, forEachDiags := evaluateResourceForEachExpression(n.ModuleCall.ForEach, ctx)
	if forEachDiags.HasErrors() {
		return nil, forEachDiags.Err()
	}

	if forEach != nil {
		eachMode = states.EachMap
	}

	switch eachMode {
	case states.EachList:
		expander.SetModuleCount(ctx.Path(), n.Call, count)
	case states.EachMap:
		expander.SetModuleForEach(ctx.Path(), n.Call, forEach)
	default:
		expander.SetModuleSingle(n.CallerAddr, n.Call)
	}

	return nil, nil
}
