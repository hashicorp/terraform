package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalImportState is an EvalNode implementation that performs an
// ImportState operation on a provider. This will return the imported
// states but won't modify any actual state.
type EvalImportState struct {
	Addr     addrs.ResourceInstance
	Provider *providers.Interface
	ID       string
	Output   *[]providers.ImportedResource
}

// TODO: test
func (n *EvalImportState) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())
	provider := *n.Provider
	var diags tfdiags.Diagnostics

	{
		// Call pre-import hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreImportState(absAddr, n.ID)
		})
		if err != nil {
			return nil, err
		}
	}

	resp := provider.ImportResourceState(providers.ImportResourceStateRequest{
		TypeName: n.Addr.Resource.Type,
		ID:       n.ID,
	})
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	imported := resp.ImportedResources

	for _, obj := range imported {
		log.Printf("[TRACE] EvalImportState: import %s %q produced instance object of type %s", absAddr.String(), n.ID, obj.TypeName)
	}

	if n.Output != nil {
		*n.Output = imported
	}

	{
		// Call post-import hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostImportState(absAddr, imported)
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
