package terraform

import (
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/config"
	"github.com/mitchellh/mapstructure"
)

// EvalSetVariables is an EvalNode implementation that sets the variables
// explicitly for interpolation later.
type EvalSetVariables struct {
	Module    *string
	Variables map[string]string
}

// TODO: test
func (n *EvalSetVariables) Eval(ctx EvalContext) (interface{}, error) {
	ctx.SetVariables(*n.Module, n.Variables)
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
		var vStr string
		if err := mapstructure.WeakDecode(v, &vStr); err != nil {
			return nil, errwrap.Wrapf(fmt.Sprintf(
				"%s: error reading value: {{err}}", k), err)
		}

		n.Variables[k] = vStr
	}
	for k, _ := range rc.Raw {
		if _, ok := n.Variables[k]; !ok {
			n.Variables[k] = config.UnknownVariableValue
		}
	}

	return nil, nil
}
