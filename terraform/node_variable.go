package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// NodeApplyableVariable represents a variable during the apply step.
type NodeApplyableVariable struct {
	PathValue []string
	Config    *config.Variable  // Config is the var in the config
	Value     *config.RawConfig // Value is the value that is set

	Module *module.Tree // Antiquated, want to remove
}

func (n *NodeApplyableVariable) Name() string {
	result := fmt.Sprintf("var.%s", n.Config.Name)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeApplyableVariable) Path() []string {
	// We execute in the parent scope (above our own module) so that
	// we can access the proper interpolations.
	if len(n.PathValue) > 2 {
		return n.PathValue[:len(n.PathValue)-1]
	}

	return nil
}

// GraphNodeReferenceGlobal
func (n *NodeApplyableVariable) ReferenceGlobal() bool {
	// We have to create fully qualified references because we cross
	// boundaries here: our ReferenceableName is in one path and our
	// References are from another path.
	return true
}

// GraphNodeReferenceable
func (n *NodeApplyableVariable) ReferenceableName() []string {
	return []string{n.Name()}
}

// GraphNodeEvalable
func (n *NodeApplyableVariable) EvalTree() EvalNode {
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
			&EvalInterpolate{
				Config: n.Value,
				Output: &config,
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
