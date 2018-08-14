package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalImportState is an EvalNode implementation that performs an
// ImportState operation on a provider. This will return the imported
// states but won't modify any actual state.
type EvalImportState struct {
	Addr     addrs.ResourceInstance
	Provider *ResourceProvider
	Id       string
	Output   *[]*states.ImportedObject
}

// TODO: test
func (n *EvalImportState) Eval(ctx EvalContext) (interface{}, error) {
	return nil, fmt.Errorf("EvalImportState not yet updated for new state/provider types")
	/*
		absAddr := n.Addr.Absolute(ctx.Path())
		provider := *n.Provider

		{
			// Call pre-import hook
			err := ctx.Hook(func(h Hook) (HookAction, error) {
				return h.PreImportState(absAddr, n.Id)
			})
			if err != nil {
				return nil, err
			}
		}

		// Import!
		state, err := provider.ImportState(n.Info, n.Id)
		if err != nil {
			return nil, fmt.Errorf("import %s (id: %s): %s", absAddr.String(), n.Id, err)
		}

		for _, s := range state {
			if s == nil {
				log.Printf("[TRACE] EvalImportState: import %s %q produced a nil state", absAddr.String(), n.Id)
				continue
			}
			log.Printf("[TRACE] EvalImportState: import %s %q produced state for %s with id %q", absAddr.String(), n.Id, s.Ephemeral.Type, s.ID)
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
	*/
}

// EvalImportStateVerify verifies the state after ImportState and
// after the refresh to make sure it is non-nil and valid.
type EvalImportStateVerify struct {
	Addr  addrs.ResourceInstance
	State **states.ResourceInstanceObject
}

// TODO: test
func (n *EvalImportStateVerify) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics

	state := *n.State
	if state.Value.IsNull() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot import non-existent remote object",
			fmt.Sprintf(
				"While attempting to import an existing object to %s, the provider detected that no object exists with the given id. Only pre-existing objects can be imported; check that the id is correct and that it is associated with the provider's configured region or endpoint, or use \"terraform apply\" to create a new remote object for this resource.",
				n.Addr.String(),
			),
		))
	}

	return nil, diags.ErrWithWarnings()
}
