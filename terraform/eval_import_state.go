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

// EvalImportStateVerify verifies the state after ImportState and
// after the refresh to make sure it is non-nil and valid.
type EvalImportStateVerify struct {
	Info  *InstanceInfo
	Id    string
	State **InstanceState
}

// TODO: test
func (n *EvalImportStateVerify) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State
	if state.Empty() {
		return nil, fmt.Errorf(
			"import %s (id: %s): Terraform detected a resource with this ID doesn't\n"+
				"exist. Please verify the ID is correct. You cannot import non-existent\n"+
				"resources using Terraform import.",
			n.Info.HumanId(),
			n.Id)
	}

	return nil, nil
}
