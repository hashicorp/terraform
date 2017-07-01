package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// EvalLocal is an EvalNode implementation that evaluates the
// expression for a local value and writes it into a transient part of
// the state.
type EvalLocal struct {
	Name  string
	Value *config.RawConfig
}

func (n *EvalLocal) Eval(ctx EvalContext) (interface{}, error) {
	cfg, err := ctx.Interpolate(n.Value, nil)
	if err != nil {
		return nil, fmt.Errorf("local.%s: %s", n.Name, err)
	}

	state, lock := ctx.State()
	if state == nil {
		return nil, fmt.Errorf("cannot write local value to nil state")
	}

	// Get a write lock so we can access the state
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

	if mod.Locals == nil {
		// initialize
		mod.Locals = map[string]interface{}{}
	}
	mod.Locals[n.Name] = valueRaw

	return nil, nil
}
