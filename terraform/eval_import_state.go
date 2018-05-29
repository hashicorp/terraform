package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
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

	for _, s := range state {
		if s == nil {
			log.Printf("[TRACE] EvalImportState: import %s %q produced a nil state", n.Info.HumanId(), n.Id)
			continue
		}
		log.Printf("[TRACE] EvalImportState: import %s %q produced state for %s with id %q", n.Info.HumanId(), n.Id, s.Ephemeral.Type, s.ID)
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
	Addr  addrs.ResourceInstance
	Id    string
	State **InstanceState
}

// TODO: test
func (n *EvalImportStateVerify) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics

	state := *n.State
	if state.Empty() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot import non-existent remote object",
			fmt.Sprintf(
				"While attempting to import an existing object to %s, the provider detected that no object exists with the id %q. Only pre-existing objects can be imported; check that the id is correct and that it is associated with the provider's configured region or endpoint, or use \"terraform apply\" to create a new remote object for this resource.",
				n.Addr.String(), n.Id,
			),
		))
	}

	return nil, diags.ErrWithWarnings()
}
