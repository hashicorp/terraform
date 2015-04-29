package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// EvalDeleteOutput is an EvalNode implementation that deletes an output
// from the state.
type EvalDeleteOutput struct {
	Name string
}

// TODO: test
func (n *EvalDeleteOutput) Eval(ctx EvalContext) (interface{}, error) {
	state, lock := ctx.State()
	if state == nil {
		return nil, nil
	}

	// Get a write lock so we can access this instance
	lock.Lock()
	defer lock.Unlock()

	// Look for the module state. If we don't have one, create it.
	mod := state.ModuleByPath(ctx.Path())
	if mod == nil {
		return nil, nil
	}

	delete(mod.Outputs, n.Name)

	return nil, nil
}

// EvalWriteOutput is an EvalNode implementation that writes the output
// for the given name to the current state.
type EvalWriteOutput struct {
	Name  string
	Value *config.RawConfig
}

// TODO: test
func (n *EvalWriteOutput) Eval(ctx EvalContext) (interface{}, error) {
	cfg, err := ctx.Interpolate(n.Value, nil)
	if err != nil {
		// Ignore it
	}

	state, lock := ctx.State()
	if state == nil {
		return nil, fmt.Errorf("cannot write state to nil state")
	}

	// Get a write lock so we can access this instance
	lock.Lock()
	defer lock.Unlock()

	// Look for the module state. If we don't have one, create it.
	mod := state.ModuleByPath(ctx.Path())
	if mod == nil {
		mod = state.AddModule(ctx.Path())
	}

	// Get the value from the config
	var valueRaw interface{} = config.UnknownVariableValue
	if cfg != nil {
		var ok bool
		valueRaw, ok = cfg.Get("value")
		if !ok {
			valueRaw = ""
		}
		if cfg.IsComputed("value") {
			valueRaw = config.UnknownVariableValue
		}
	}

	// If it is a list of values, get the first one
	if list, ok := valueRaw.([]interface{}); ok {
		valueRaw = list[0]
	}
	if _, ok := valueRaw.(string); !ok {
		return nil, fmt.Errorf("output %s is not a string", n.Name)
	}

	// Write the output
	mod.Outputs[n.Name] = valueRaw.(string)

	return nil, nil
}
