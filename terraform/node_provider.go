package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
)

// NodeApplyableProvider represents a provider during an apply.
type NodeApplyableProvider struct {
	*NodeAbstractProvider
}

var (
	_ GraphNodeExecutable = (*NodeApplyableProvider)(nil)
)

// GraphNodeExecutable
func (n *NodeApplyableProvider) Execute(ctx EvalContext, op walkOperation) error {
	_, err := ctx.InitProvider(n.Addr)
	if err != nil {
		return err
	}
	provider, _, err := GetProvider(ctx, n.Addr)
	if err != nil {
		return err
	}

	switch op {
	case walkValidate:
		return n.ValidateProvider(ctx, provider)
	case walkRefresh, walkPlan, walkApply, walkDestroy:
		return n.ConfigureProvider(ctx, provider, false)
	case walkImport:
		return n.ConfigureProvider(ctx, provider, true)
	}
	return nil
}

func (n *NodeApplyableProvider) ValidateProvider(ctx EvalContext, provider providers.Interface) error {
	var diags tfdiags.Diagnostics

	configBody := buildProviderConfig(ctx, n.Addr, n.ProviderConfig())

	resp := provider.GetSchema()
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	configSchema := resp.Provider.Block
	if configSchema == nil {
		// Should never happen in real code, but often comes up in tests where
		// mock schemas are being used that tend to be incomplete.
		log.Printf("[WARN] ValidateProvider: no config schema is available for %s, so using empty schema", n.Addr)
		configSchema = &configschema.Block{}
	}

	configVal, configBody, evalDiags := ctx.EvaluateBlock(configBody, configSchema, nil, EvalDataForNoInstanceKey)
	diags = diags.Append(evalDiags)
	if evalDiags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	req := providers.PrepareProviderConfigRequest{
		Config: configVal,
	}

	validateResp := provider.PrepareProviderConfig(req)
	diags = diags.Append(validateResp.Diagnostics)

	return diags.ErrWithWarnings()
}

// ConfigureProvider configures a provider that is already initialized and retrieved.
// If verifyConfigIsKnown is true, ConfigureProvider will return an error if the
// provider configVal is not wholly known and is meant only for use during import.
func (n *NodeApplyableProvider) ConfigureProvider(ctx EvalContext, provider providers.Interface, verifyConfigIsKnown bool) error {
	var diags tfdiags.Diagnostics
	config := n.ProviderConfig()

	configBody := buildProviderConfig(ctx, n.Addr, config)

	resp := provider.GetSchema()
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	configSchema := resp.Provider.Block
	configVal, configBody, evalDiags := ctx.EvaluateBlock(configBody, configSchema, nil, EvalDataForNoInstanceKey)
	diags = diags.Append(evalDiags)
	if evalDiags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	if verifyConfigIsKnown && !configVal.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider configuration",
			Detail:   fmt.Sprintf("The configuration for %s depends on values that cannot be determined until apply.", n.Addr),
			Subject:  &config.DeclRange,
		})
		return diags.ErrWithWarnings()
	}

	configDiags := ctx.ConfigureProvider(n.Addr, configVal)
	configDiags = configDiags.InConfigBody(configBody)

	return configDiags.ErrWithWarnings()
}
