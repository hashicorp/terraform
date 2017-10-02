package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// NodeApplyableModuleVariable represents a module variable input during
// the apply step.
type NodeApplyableModuleVariable struct {
	PathValue []string
	Config    *config.Variable  // Config is the var in the config
	Value     *config.RawConfig // Value is the value that is set

	Module *module.Tree // Antiquated, want to remove
}

func (n *NodeApplyableModuleVariable) Name() string {
	result := fmt.Sprintf("var.%s", n.Config.Name)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeApplyableModuleVariable) Path() []string {
	// We execute in the parent scope (above our own module) so that
	// we can access the proper interpolations.
	if len(n.PathValue) > 2 {
		return n.PathValue[:len(n.PathValue)-1]
	}

	return rootModulePath
}

// RemovableIfNotTargeted
func (n *NodeApplyableModuleVariable) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeReferenceGlobal
func (n *NodeApplyableModuleVariable) ReferenceGlobal() bool {
	// We have to create fully qualified references because we cross
	// boundaries here: our ReferenceableName is in one path and our
	// References are from another path.
	return true
}

// GraphNodeReferenceable
func (n *NodeApplyableModuleVariable) ReferenceableName() []string {
	return []string{n.Name()}
}

// GraphNodeReferencer
func (n *NodeApplyableModuleVariable) References() []string {
	// If we have no value set, we depend on nothing
	if n.Value == nil {
		return nil
	}

	// Can't depend on anything if we're in the root
	if len(n.PathValue) < 2 {
		return nil
	}

	// Otherwise, we depend on anything that is in our value, but
	// specifically in the namespace of the parent path.
	// Create the prefix based on the path
	var prefix string
	if p := n.Path(); len(p) > 0 {
		prefix = modulePrefixStr(p)
	}

	result := ReferencesFromConfig(n.Value)
	return modulePrefixList(result, prefix)
}

// GraphNodeEvalable
func (n *NodeApplyableModuleVariable) EvalTree() EvalNode {
	// If we have no value, do nothing
	if n.Value == nil {
		return &EvalNoop{}
	}

	// Otherwise, interpolate the value of this variable and set it
	// within the variables mapping.
	var config *ResourceConfig
	variables := make(map[string]interface{})

	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalOpFilter{
				Ops: []walkOperation{walkInput},
				Node: &EvalInterpolate{
					Config:        n.Value,
					Output:        &config,
					ContinueOnErr: true,
				},
			},
			&EvalOpFilter{
				Ops: []walkOperation{walkRefresh, walkPlan, walkApply,
					walkDestroy, walkValidate},
				Node: &EvalInterpolate{
					Config: n.Value,
					Output: &config,
				},
			},

			&EvalVariableBlock{
				Config:         &config,
				VariableValues: variables,
			},

			&EvalCoerceMapVariable{
				Variables:  variables,
				ModulePath: n.PathValue,
				ModuleTree: n.Module,
			},

			&EvalTypeCheckVariable{
				Variables:  variables,
				ModulePath: n.PathValue,
				ModuleTree: n.Module,
			},

			&EvalSetVariables{
				Module:    &n.PathValue[len(n.PathValue)-1],
				Variables: variables,
			},
		},
	}
}
