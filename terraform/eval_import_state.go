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
	Output   *[]*InstanceState
}

// TODO: test
func (n *EvalImportState) Eval(ctx EvalContext) (interface{}, error) {
	provider := *n.Provider

	// Refresh!
	state, err := provider.ImportState(n.Info)
	if err != nil {
		return nil, fmt.Errorf(
			"import %s (id: %s): %s", n.Info.Type, n.Info.Id, err)
	}

	if n.Output != nil {
		*n.Output = state
	}

	return nil, nil
}
