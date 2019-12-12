package applying

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

// providerConfigActions gathers together all of the actions for a specific
// provider configuration.
type providerConfigActions struct {
	Addr        addrs.AbsProviderConfig
	Instantiate *instantiateProviderAction
	Close       *closeProviderAction
}

// instantiateProviderAction is an action that creates an instance of a
// provider and configures it so that it's ready to accept further requests,
// before registering it in the shared actionData object for use by
// other downstream actions.
type instantiateProviderAction struct {
	Addr   addrs.AbsProviderConfig
	Config *configs.Provider // can be nil if no provider blocks are present
}

func (a *instantiateProviderAction) Name() string {
	return "Init " + a.Addr.String()
}

func (a *instantiateProviderAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	providerAddr := a.Addr.ProviderConfig.Type
	inst, err := data.StartProviderInstance(providerAddr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to start provider instance",
			Detail:   fmt.Sprintf("Error while launching provider instance for %s: %s.", a.Addr, err),
			Subject:  a.Config.DeclRange.Ptr(),
		})
		return diags
	}

	// If we encounter errors below that prevent running to completion then
	// we'll try to close the provider before we return.
	defer func() {
		if diags.HasErrors() {
			err := inst.Close()
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to terminate provider instance",
					Detail:   fmt.Sprintf("Error while shutting down provider instance for %s: %s.", a.Addr, err),
					Subject:  a.Config.DeclRange.Ptr(),
				})
			}
		}
	}()

	// TODO: Evaluate the config and configure this provider.
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Configuration of provider instances not yet implemented",
		"The prototype apply codepath does not yet support configuring provider instances.",
	))

	data.SetConfiguredProviderInstance(a.Addr, inst)

	return diags
}

// closeProviderAction is an action that terminates a provider instance
// previously created by instantiateProviderAction.
type closeProviderAction struct {
	Addr   addrs.AbsProviderConfig
	Config *configs.Provider // can be nil if no provider blocks are present
}

func (a *closeProviderAction) Name() string {
	return "Close " + a.Addr.String()
}

func (a *closeProviderAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	err := data.CloseProviderInstance(a.Addr)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to terminate provider instance",
			Detail:   fmt.Sprintf("Error while shutting down provider instance for %s: %s.", a.Addr, err),
			Subject:  a.Config.DeclRange.Ptr(),
		})
	}
	return diags
}
