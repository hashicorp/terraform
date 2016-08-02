package terraform

import (
	"fmt"
	"log"

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
	Name      string
	Sensitive bool
	Value     *config.RawConfig
}

// TODO: test
func (n *EvalWriteOutput) Eval(ctx EvalContext) (interface{}, error) {
	cfg, err := ctx.Interpolate(n.Value, nil)
	if err != nil {
		// Log error but continue anyway
		log.Printf("[WARN] Output interpolation %q failed: %s", n.Name, err)
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

	switch valueTyped := valueRaw.(type) {
	case string:
		mod.Outputs[n.Name] = &OutputState{
			Type:      "string",
			Sensitive: n.Sensitive,
			Value:     valueTyped,
		}
	case []interface{}:
		mod.Outputs[n.Name] = &OutputState{
			Type:      "list",
			Sensitive: n.Sensitive,
			Value:     valueTyped,
		}
	case map[string]interface{}:
		mod.Outputs[n.Name] = &OutputState{
			Type:      "map",
			Sensitive: n.Sensitive,
			Value:     valueTyped,
		}
	default:
		return nil, fmt.Errorf("output %s is not a valid type (%T)\n", n.Name, valueTyped)
	}

	return nil, nil
}
