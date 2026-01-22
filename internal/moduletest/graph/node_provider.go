// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

var (
	_ GraphNodeExecutable = (*NodeProviderConfigure)(nil)
	_ GraphNodeReferencer = (*NodeProviderConfigure)(nil)
)

type NodeProviderConfigure struct {
	name, alias string

	Addr     addrs.RootProviderConfig
	File     *moduletest.File
	Config   *configs.Provider
	Provider providers.Interface
	Schema   providers.GetProviderSchemaResponse
}

func (n *NodeProviderConfigure) Name() string {
	if len(n.alias) > 0 {
		return fmt.Sprintf("provider.%s.%s", n.name, n.alias)
	}
	return fmt.Sprintf("provider.%s", n.name)
}

func (n *NodeProviderConfigure) Execute(ctx *EvalContext) {
	log.Printf("[TRACE]: NodeProviderConfigure: configuring provider %s for tests", n.Addr)
	if ctx.Cancelled() || ctx.Stopped() {
		return
	}

	// first, set the provider so everything else can use it
	ctx.SetProvider(n.Addr, n.Provider)

	spec := n.Schema.Provider.Body.DecoderSpec()

	var references []*addrs.Reference
	var referenceDiags tfdiags.Diagnostics
	for _, traversal := range hcldec.Variables(n.Config.Config, spec) {
		ref, moreDiags := addrs.ParseRefFromTestingScope(traversal)
		referenceDiags = referenceDiags.Append(moreDiags)
		if ref != nil {
			references = append(references, ref)
		}
	}
	n.File.AppendDiagnostics(referenceDiags)
	if referenceDiags.HasErrors() {
		ctx.SetProviderStatus(n.Addr, moduletest.Error)
		return
	}

	if !ctx.ReferencesCompleted(references) {
		ctx.SetProviderStatus(n.Addr, moduletest.Skip)
		return
	}

	hclContext, moreDiags := ctx.HclContext(references)
	n.File.AppendDiagnostics(moreDiags)
	if moreDiags.HasErrors() {
		ctx.SetProviderStatus(n.Addr, moduletest.Error)
		return
	}

	// This means we are using a mock provider, which may contain not-yet-evaluated
	// mock data, so we will evaluate the data here.
	if mock, ok := n.Provider.(*providers.Mock); ok {
		for _, res := range mock.Data.MockResources {
			values, exprHclDiags := res.RawExpr.Value(hclContext)
			moreDiags = moreDiags.Append(exprHclDiags)
			res.Defaults = values
		}
		for _, res := range mock.Data.MockDataSources {
			values, exprHclDiags := res.RawExpr.Value(hclContext)
			moreDiags = moreDiags.Append(exprHclDiags)
			res.Defaults = values
		}
	}

	body, decHclDiags := hcldec.Decode(n.Config.Config, spec, hclContext)
	moreDiags = moreDiags.Append(decHclDiags)
	if moreDiags.HasErrors() {
		n.File.AppendDiagnostics(moreDiags)
		ctx.SetProviderStatus(n.Addr, moduletest.Error)
		return
	}

	unmarkedBody, _ := body.UnmarkDeep()
	response := n.Provider.ConfigureProvider(providers.ConfigureProviderRequest{
		TerraformVersion: version.SemVer.String(),
		Config:           unmarkedBody,
		ClientCapabilities: providers.ClientCapabilities{
			DeferralAllowed:            ctx.deferralAllowed,
			WriteOnlyAttributesAllowed: true,
		},
	})

	n.File.AppendDiagnostics(response.Diagnostics)
	if response.Diagnostics.HasErrors() {
		ctx.SetProviderStatus(n.Addr, moduletest.Error)
		return
	}
}

func (n *NodeProviderConfigure) References() []*addrs.Reference {
	var refs []*addrs.Reference
	for _, variable := range hcldec.Variables(n.Config.Config, n.Schema.Provider.Body.DecoderSpec()) {
		ref, _ := addrs.ParseRefFromTestingScope(variable)
		if ref != nil {
			refs = append(refs, ref)
		}
	}
	return refs
}

var (
	_ GraphNodeExecutable = (*NodeProviderClose)(nil)
)

type NodeProviderClose struct {
	name, alias string

	Addr     addrs.RootProviderConfig
	File     *moduletest.File
	Config   *configs.Provider
	Provider providers.Interface
}

func (n *NodeProviderClose) Name() string {
	if len(n.alias) > 0 {
		return fmt.Sprintf("provider.%s.%s (close)", n.name, n.alias)
	}
	return fmt.Sprintf("provider.%s (close)", n.name)
}

func (n *NodeProviderClose) Execute(ctx *EvalContext) {
	log.Printf("[TRACE]: NodeProviderClose: closing provider %s for tests", n.Addr)

	// we don't check for cancelled or stopped here - we still want to kill
	// any running providers even if someone has stopped or cancelled the
	// process

	if err := n.Provider.Close(); err != nil {
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to close provider",
			Detail:   fmt.Sprintf("Failed to close provider: %s", err.Error()),
			Subject:  n.Config.DeclRange.Ptr(),
		})
		n.File.AppendDiagnostics(diags)
	}
}
