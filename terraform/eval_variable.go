package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/mitchellh/mapstructure"
)

// EvalTypeCheckVariable is an EvalNode which ensures that the variable
// values which are assigned as inputs to a module (including the root)
// match the types which are either declared for the variables explicitly
// or inferred from the default values.
//
// In order to achieve this three things are required:
//     - a map of the proposed variable values
//     - the configuration tree of the module in which the variable is
//       declared
//     - the path to the module (so we know which part of the tree to
//       compare the values against).
//
// Currently since the type system is simple, we currently do not make
// use of the values since it is only valid to pass string values. The
// structure is in place for extension of the type system, however.
type EvalTypeCheckVariable struct {
	Variables  map[string]string
	ModulePath []string
	ModuleTree *module.Tree
}

func (n *EvalTypeCheckVariable) Eval(ctx EvalContext) (interface{}, error) {
	currentTree := n.ModuleTree
	for _, pathComponent := range n.ModulePath[1:] {
		currentTree = currentTree.Children()[pathComponent]
	}
	targetConfig := currentTree.Config()

	prototypes := make(map[string]config.VariableType)
	for _, variable := range targetConfig.Variables {
		prototypes[variable.Name] = variable.Type()
	}

	for name, declaredType := range prototypes {
		// This is only necessary when we _actually_ check. It is left as a reminder
		// that at the current time we are dealing with a type system consisting only
		// of strings and maps - where the only valid inter-module variable type is
		// string.
		_, ok := n.Variables[name]
		if !ok {
			// This means the default value should be used as no overriding value
			// has been set. Therefore we should continue as no check is necessary.
			continue
		}

		switch declaredType {
		case config.VariableTypeString:
			// This will need actual verification once we aren't dealing with
			// a map[string]string but this is sufficient for now.
			continue
		default:
			// Only display a module if we are not in the root module
			modulePathDescription := fmt.Sprintf(" in module %s", strings.Join(n.ModulePath[1:], "."))
			if len(n.ModulePath) == 1 {
				modulePathDescription = ""
			}
			// This will need the actual type substituting when we have more than
			// just strings and maps.
			return nil, fmt.Errorf("variable %s%s should be type %s, got type string",
				name, modulePathDescription, declaredType.Printable())
		}
	}

	return nil, nil
}

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
