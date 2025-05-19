// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeApplyableProvider represents a provider during an apply.
type NodeApplyableProvider struct {
	*NodeAbstractProvider
}

var (
	_ GraphNodeExecutable = (*NodeApplyableProvider)(nil)
)

// GraphNodeExecutable
func (n *NodeApplyableProvider) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	_, err := ctx.InitProvider(n.Addr, n.Config)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}
	provider, _, err := getProvider(ctx, n.Addr)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	switch op {
	case walkValidate:
		log.Printf("[TRACE] NodeApplyableProvider: validating configuration for %s", n.Addr)
		return diags.Append(n.ValidateProvider(ctx, provider))
	case walkPlan, walkPlanDestroy, walkApply, walkDestroy:
		log.Printf("[TRACE] NodeApplyableProvider: configuring %s", n.Addr)
		return diags.Append(n.ConfigureProvider(ctx, provider, false))
	case walkImport:
		log.Printf("[TRACE] NodeApplyableProvider: configuring %s (requiring that configuration is wholly known)", n.Addr)
		return diags.Append(n.ConfigureProvider(ctx, provider, true))
	}
	return diags
}

func (n *NodeApplyableProvider) ValidateProvider(ctx EvalContext, provider providers.Interface) (diags tfdiags.Diagnostics) {

	configBody := buildProviderConfig(ctx, n.Addr, n.ProviderConfig())

	// if a provider config is empty (only an alias), return early and don't continue
	// validation. validate doesn't need to fully configure the provider itself, so
	// skipping a provider with an implied configuration won't prevent other validation from completing.
	_, noConfigDiags := configBody.Content(&hcl.BodySchema{})
	if !noConfigDiags.HasErrors() {
		return nil
	}

	schemaResp := provider.GetProviderSchema()
	diags = diags.Append(schemaResp.Diagnostics.InConfigBody(configBody, n.Addr.String()))
	if diags.HasErrors() {
		return diags
	}

	configSchema := schemaResp.Provider.Body
	if configSchema == nil {
		// Should never happen in real code, but often comes up in tests where
		// mock schemas are being used that tend to be incomplete.
		log.Printf("[WARN] ValidateProvider: no config schema is available for %s, so using empty schema", n.Addr)
		configSchema = &configschema.Block{}
	}

	configVal, _, evalDiags := ctx.EvaluateBlock(configBody, configSchema, nil, EvalDataForNoInstanceKey)
	if evalDiags.HasErrors() {
		return diags.Append(evalDiags)
	}
	diags = diags.Append(evalDiags)

	// If our config value contains any marked values, ensure those are
	// stripped out before sending this to the provider
	unmarkedConfigVal, _ := configVal.UnmarkDeep()

	req := providers.ValidateProviderConfigRequest{
		Config: unmarkedConfigVal,
	}

	validateResp := provider.ValidateProviderConfig(req)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(configBody, n.Addr.String()))

	return diags
}

// ConfigureProvider configures a provider that is already initialized and retrieved.
// If verifyConfigIsKnown is true, ConfigureProvider will return an error if the
// provider configVal is not wholly known and is meant only for use during import.
func (n *NodeApplyableProvider) ConfigureProvider(ctx EvalContext, provider providers.Interface, verifyConfigIsKnown bool) (diags tfdiags.Diagnostics) {
	config := n.ProviderConfig()

	configBody := buildProviderConfig(ctx, n.Addr, config)

	resp := provider.GetProviderSchema()
	diags = diags.Append(resp.Diagnostics.InConfigBody(configBody, n.Addr.String()))
	if diags.HasErrors() {
		return diags
	}

	configSchema := resp.Provider.Body
	configVal, configBody, evalDiags := ctx.EvaluateBlock(configBody, configSchema, nil, EvalDataForNoInstanceKey)
	diags = diags.Append(evalDiags)
	if evalDiags.HasErrors() {
		if config == nil {
			// The error messages from the above evaluation will be confusing
			// if there isn't an explicit "provider" block in the configuration.
			// Add some detail to the error message in this case.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid provider configuration",
				fmt.Sprintf(providerConfigErr, n.Addr.Provider),
			))
		}
		return diags
	}

	if verifyConfigIsKnown && !configVal.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration",
			Detail:   fmt.Sprintf("The configuration for %s depends on values that cannot be determined until apply.", n.Addr),
			Subject:  &config.DeclRange,
		})
		return diags
	}

	// If our config value contains any marked values, ensure those are
	// stripped out before sending this to the provider
	unmarkedConfigVal, _ := configVal.UnmarkDeep()

	// Allow the provider to validate and insert any defaults into the full
	// configuration.
	req := providers.ValidateProviderConfigRequest{
		Config: unmarkedConfigVal,
	}

	// ValidateProviderConfig is only used for validation. We are intentionally
	// ignoring the PreparedConfig field to maintain existing behavior.
	validateResp := provider.ValidateProviderConfig(req)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(configBody, n.Addr.String()))
	if diags.HasErrors() && config == nil {
		// If there isn't an explicit "provider" block in the configuration,
		// this error message won't be very clear. Add some detail to the error
		// message in this case.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid provider configuration",
			fmt.Sprintf(providerConfigErr, n.Addr.Provider),
		))
	}

	if diags.HasErrors() {
		return diags
	}

	// If the provider returns something different, log a warning to help
	// indicate to provider developers that the value is not used.
	preparedCfg := validateResp.PreparedConfig
	if preparedCfg != cty.NilVal && !preparedCfg.IsNull() && !preparedCfg.RawEquals(unmarkedConfigVal) {
		log.Printf("[WARN] ValidateProviderConfig from %q changed the config value, but that value is unused", n.Addr)
	}

	configDiags := ctx.ConfigureProvider(n.Addr, unmarkedConfigVal)
	diags = diags.Append(configDiags.InConfigBody(configBody, n.Addr.String()))
	if diags.HasErrors() && config == nil {
		// If there isn't an explicit "provider" block in the configuration,
		// this error message won't be very clear. Add some detail to the error
		// message in this case.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid provider configuration",
			fmt.Sprintf(providerConfigErr, n.Addr.Provider),
		))
	}
	return diags
}

// nodeExternalProvider is used instead of [NodeApplyableProvider] when an
// already-configured provider instance has been provided by an external caller,
// and therefore we don't need to do anything to get the provider ready to
// use.
type nodeExternalProvider struct {
	*NodeAbstractProvider
}

var (
	_ GraphNodeExecutable = (*nodeExternalProvider)(nil)
)

// Execute implements GraphNodeExecutable.
func (n *nodeExternalProvider) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	log.Printf("[TRACE] nodeExternalProvider: using externally-configured instance for %s", n.Addr)
	var diags tfdiags.Diagnostics

	// Due to how the EvalContext provider cache works, we need to just poke
	// this method with our provider address so that a subsequent call
	// to ctx.Provider will return it successfully.
	// In this case the "config" argument is always ignored, so we leave it
	// set to nil to represent that.
	_, err := ctx.InitProvider(n.Addr, nil)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to initialize externally-configured provider",
			fmt.Sprintf("Despite it having been pre-initialized by an external caller, %s somehow failed to initialize. This is a bug in Terraform.", n.Addr),
		))
	}

	return diags
}

const providerConfigErr = `Provider %q requires explicit configuration. Add a provider block to the root module and configure the provider's required arguments as described in the provider documentation.
`
