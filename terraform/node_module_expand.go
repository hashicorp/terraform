package terraform

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

// nodeExpandModule represents a module call in the configuration that
// might expand into multiple module instances depending on how it is
// configured.
type nodeExpandModule struct {
	CallerAddr addrs.ModuleInstance
	Call       addrs.ModuleCall
	Config     *configs.Module
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
	// Notice that the node represents the module call and so we report
	// the parent module as the path. The module call we're representing
	// might expand into multiple child module instances during our work here.
	return n.CallerAddr
}

// GraphNodeReferencer implementation
func (n *nodeExpandModule) References() []*addrs.Reference {
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
	var ret []*addrs.Reference
	// TODO: Once count and for_each are actually supported, analyze their
	// expressions for references here.
	/*
		if n.Config.Count != nil {
			ret = append(ret, n.Config.Count.References()...)
		}
		if n.Config.ForEach != nil {
			ret = append(ret, n.Config.ForEach.References()...)
		}
	*/
	return ret
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
	}
}

type evalPrepareModuleExpansion struct {
	CallerAddr addrs.ModuleInstance
	Call       addrs.ModuleCall
	Config     *configs.Module
}

func (n *evalPrepareModuleExpansion) Eval(ctx EvalContext) (interface{}, error) {
	// Modules don't support any of the repetition arguments yet, so their
	// expansion type is always "single". We just record this here to make
	// the expander data structure consistent for now.
	// FIXME: Once the rest of Terraform Core is ready to support expanding
	// modules, evaluate the "count" and "for_each" arguments here in a
	// similar way as in EvalWriteResourceState.
	log.Printf("[TRACE] evalPrepareModuleExpansion: %s is a singleton", n.CallerAddr.Child(n.Call.Name, addrs.NoKey))
	ctx.InstanceExpander().SetModuleSingle(n.CallerAddr, n.Call)
	return nil, nil
}
