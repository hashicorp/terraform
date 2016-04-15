package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeConfigVariable represents a Variable in the config.
type GraphNodeConfigVariable struct {
	Variable *config.Variable

	// Value, if non-nil, will be used to set the value of the variable
	// during evaluation. If this is nil, evaluation will do nothing.
	//
	// Module is the name of the module to set the variables on.
	Module string
	Value  *config.RawConfig

	ModuleTree *module.Tree
	ModulePath []string
}

func (n *GraphNodeConfigVariable) Name() string {
	return fmt.Sprintf("var.%s", n.Variable.Name)
}

func (n *GraphNodeConfigVariable) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeVariable
}

func (n *GraphNodeConfigVariable) DependableName() []string {
	return []string{n.Name()}
}

func (n *GraphNodeConfigVariable) DependentOn() []string {
	// If we don't have any value set, we don't depend on anything
	if n.Value == nil {
		return nil
	}

	// Get what we depend on based on our value
	vars := n.Value.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

func (n *GraphNodeConfigVariable) VariableName() string {
	return n.Variable.Name
}

// GraphNodeDestroyEdgeInclude impl.
func (n *GraphNodeConfigVariable) DestroyEdgeInclude(v dag.Vertex) bool {
	// Only include this variable in a destroy edge if the source vertex
	// "v" has a count dependency on this variable.
	cv, ok := v.(GraphNodeCountDependent)
	if !ok {
		return false
	}

	for _, d := range cv.CountDependentOn() {
		for _, d2 := range n.DependableName() {
			if d == d2 {
				return true
			}
		}
	}

	return false
}

// GraphNodeNoopPrunable
func (n *GraphNodeConfigVariable) Noop(opts *NoopOpts) bool {
	// If we have no diff, always keep this in the graph. We have to do
	// this primarily for validation: we want to validate that variable
	// interpolations are valid even if there are no resources that
	// depend on them.
	if opts.Diff == nil || opts.Diff.Empty() {
		return false
	}

	for _, v := range opts.Graph.UpEdges(opts.Vertex).List() {
		// This is terrible, but I can't think of a better way to do this.
		if dag.VertexName(v) == rootNodeName {
			continue
		}

		return false
	}

	return true
}

// GraphNodeProxy impl.
func (n *GraphNodeConfigVariable) Proxy() bool {
	return true
}

// GraphNodeEvalable impl.
func (n *GraphNodeConfigVariable) EvalTree() EvalNode {
	// If we have no value, do nothing
	if n.Value == nil {
		return &EvalNoop{}
	}

	// Otherwise, interpolate the value of this variable and set it
	// within the variables mapping.
	var config *ResourceConfig
	variables := make(map[string]string)
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{
				Config: n.Value,
				Output: &config,
			},

			&EvalVariableBlock{
				Config:    &config,
				Variables: variables,
			},

			&EvalTypeCheckVariable{
				Variables:  variables,
				ModulePath: n.ModulePath,
				ModuleTree: n.ModuleTree,
			},

			&EvalSetVariables{
				Module:    &n.Module,
				Variables: variables,
			},
		},
	}
}

// GraphNodeFlattenable impl.
func (n *GraphNodeConfigVariable) Flatten(p []string) (dag.Vertex, error) {
	return &GraphNodeConfigVariableFlat{
		GraphNodeConfigVariable: n,
		PathValue:               p,
	}, nil
}

type GraphNodeConfigVariableFlat struct {
	*GraphNodeConfigVariable

	PathValue []string
}

func (n *GraphNodeConfigVariableFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.GraphNodeConfigVariable.Name())
}

func (n *GraphNodeConfigVariableFlat) DependableName() []string {
	return []string{n.Name()}
}

func (n *GraphNodeConfigVariableFlat) DependentOn() []string {
	// We only wrap the dependencies and such if we have a path that is
	// longer than 2 elements (root, child, more). This is because when
	// flattened, variables can point outside the graph.
	prefix := ""
	if len(n.PathValue) > 2 {
		prefix = modulePrefixStr(n.PathValue[:len(n.PathValue)-1])
	}

	return modulePrefixList(
		n.GraphNodeConfigVariable.DependentOn(),
		prefix)
}

func (n *GraphNodeConfigVariableFlat) Path() []string {
	if len(n.PathValue) > 2 {
		return n.PathValue[:len(n.PathValue)-1]
	}

	return nil
}
