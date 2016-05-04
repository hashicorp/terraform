package terraform

import (
	"fmt"
)

// EvalImportState is an EvalNode implementation that performs an
// ImportState operation on a provider. This will return the imported
// states but won't modify any actual state.
type EvalImportState struct {
	Provider *ResourceProvider
	Info     *InstanceInfo
	Id       string
	Output   *[]*InstanceState
}

// TODO: test
func (n *EvalImportState) Eval(ctx EvalContext) (interface{}, error) {
	provider := *n.Provider

	{
		// Call pre-import hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreImportState(n.Info, n.Id)
		})
		if err != nil {
			return nil, err
		}
	}

	// Import!
	state, err := provider.ImportState(n.Info, n.Id)
	if err != nil {
		return nil, fmt.Errorf(
			"import %s (id: %s): %s", n.Info.HumanId(), n.Id, err)
	}

	if n.Output != nil {
		*n.Output = state
	}

	{
		// Call post-import hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostImportState(n.Info, state)
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
