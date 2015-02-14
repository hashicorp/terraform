package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// EvalSetVariables is an EvalNode implementation that sets the variables
// explicitly for interpolation later.
type EvalSetVariables struct {
	Variables map[string]string
}

// TODO: test
func (n *EvalSetVariables) Eval(ctx EvalContext) (interface{}, error) {
	ctx.SetVariables(n.Variables)
	return nil, nil
}

// EvalVariableBlock is an EvalNode implementation that evaluates the
// given configuration, and uses the final values as a way to set the
// mapping.
type EvalVariableBlock struct {
	Config    **ResourceConfig
	Variables map[string]string
}

// TODO: test
func (n *EvalVariableBlock) Eval(ctx EvalContext) (interface{}, error) {
	// Clear out the existing mapping
	for k, _ := range n.Variables {
		delete(n.Variables, k)
	}

	// Get our configuration
	rc := *n.Config
	for k, v := range rc.Config {
		n.Variables[k] = v.(string)
	}
	for k, _ := range rc.Raw {
		if _, ok := n.Variables[k]; !ok {
			n.Variables[k] = config.UnknownVariableValue
		}
	}

	return nil, nil
}
