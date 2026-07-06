// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (c *Meta) PolicyClient(ctx context.Context, policyPaths []string, ent *policy.Entitlement) (policy.Client, policy.Diagnostics, func()) {
	var client policy.Client
	closer := func() {
		if client != nil {
			client.Stop()
		}
	}
	if !c.AllowExperimentalFeatures {
		log.Printf("[DEBUG] Policies are not supported without experiments enabled, skipping policy client setup")
		return client, nil, closer
	}
	if len(policyPaths) == 0 {
		log.Printf("[DEBUG] No policy paths configured, skipping policy client setup")
		return client, nil, closer
	}

	// Use a pre-initialized client for tests if one is available
	if c.testingOverrides != nil {
		client = c.testingOverrides.PolicyClient
		if client != nil {
			return client, nil, closer
		}
	}

	var diags policy.Diagnostics
	client, diags = policy.NewPolicyClient(ctx, os.Getenv(policy.TerraformPolicyPluginEnvVar), policyPaths, ent)
	if diags.HasErrors() {
		return nil, diags, closer
	}

	log.Printf("[DEBUG] backend/operation/policy: Policy engine initialized")
	return client, diags, closer
}

var _ initwd.ModuleInstallHook = &policyModuleInstallHook{}

// backendPolicyEntitlement returns the entitlement from the backend if it
// implements policy.EntitlementProvider, or nil otherwise (e.g. a local
// backend).
func backendPolicyEntitlement(be backend.Backend) *policy.Entitlement {
	provider, ok := be.(policy.EntitlementProvider)
	if !ok {
		return nil
	}
	return provider.PolicyEntitlement()
}

// policyModuleInstallHook enables policy evaluation during module installation.
type policyModuleInstallHook struct {
	initwd.ModuleInstallHookImpl
	client     policy.Client
	rootModule *configs.Module
	view       views.Init
}

// ModuleSourceResolved implements [initwd.ModuleInstallHook] and is called after a module source is resolved, and enables policy evaluation for the module before
// it is installed.
func (h *policyModuleInstallHook) ModuleSourceResolved(ctx context.Context, req *configs.ModuleRequest, version string) tfdiags.Diagnostics {
	// If the client is nil, then policy evaluation is disabled, so we can skip.
	if h.client == nil {
		return nil
	}

	log.Println("[DEBUG] init: evaluating policy for module", req.Path.String(), version)
	moduleAddr := req.Path.String()
	result := h.client.EvaluateModule(ctx, policy.EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]{
		// Configuration attributes may not be available during init, so we will send an unknown
		// dynamic value to the policy client.
		Attrs:  policy.CtyToPolicyValue(cty.DynamicVal),
		Target: req.SourceAddr.String(),
		Meta: &proto.PolicyEvaluateModuleRequest_ModuleMetadata{
			Address: moduleAddr,
			Version: version,
		},
	})
	var moduleCall *configs.ModuleCall
	if req.Parent == nil || req.Parent.Module == nil {
		log.Printf("[DEBUG] backend/operation/policy: No parent config for module %q. Diagnostics may not contain enough source context", moduleAddr)
	} else {
		moduleCall = req.Parent.Module.ModuleCalls[req.Name]
	}

	var rng hcl.Range
	if moduleCall != nil {
		rng = moduleCall.DeclRange
		ptr := rng.Ptr()
		for idx, diag := range result.Diagnostics {
			result.Diagnostics[idx] = diag.WithLocalRange(ptr)
		}
		for idx := range result.Enforcements {
			result.Enforcements[idx].LocalRange = ptr
		}
	}
	h.view.PolicyResult(req.Path.String(), result, rng)

	// Return a generic error here that the init command returns to the CLI.
	// The detailed policy diagnostics are included in the policy results
	// and will be formatted in the CLI output. Init uses diagnostics as the
	// blocking signal because advisory policies may return deny without any
	// error diagnostics.
	if result.Diagnostics.HasErrors() {
		return tfdiags.Diagnostics{
			policy.NewErrorDiagnostic(
				"Policy evaluation failed",
				"Module download blocked due to policy violations. Please review other diagnostics for details.",
				policy.SetupErrorResult,
			),
		}
	}
	return nil
}

var _ providercache.InstallerHook = &providerPolicyHook{}

// providerPolicyHook enables policy evaluation during provider installation.
type providerPolicyHook struct {
	client     policy.Client
	rootModule *configs.Module
	view       views.Init
}

// ProviderVersionSelected satisfies the [providers.InstallerHook] interface.
// When a provider version is selected, this method performs policy evaluation for the provider,
// and aborts the installation if the policy evaluation fails.
func (p *providerPolicyHook) ProviderVersionSelected(ctx context.Context, provider addrs.Provider, version string) error {
	// If the client is nil, then policy evaluation is disabled, so we can skip.
	if p.client == nil {
		return nil
	}
	log.Println("[DEBUG] init: evaluating policy for provider", provider.String(), version)
	result := p.client.EvaluateProvider(ctx, policy.EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]{
		Target: provider.Type,

		// Configuration attributes may not be available during init, so we will send an unknown
		// dynamic value to the policy client.
		Attrs: policy.CtyToPolicyValue(cty.DynamicVal),
		Meta: &proto.PolicyEvaluateProviderRequest_ProviderMetadata{
			Name:      provider.Type,
			Namespace: provider.Namespace,
			Source:    provider.String(),
			Version:   version,
		},
	})
	// We use the root module as the module for provider configs since the version resolution
	// is ambiguous, and we do not know which module the provider config belongs to.
	addr := addrs.AbsProviderConfig{Provider: provider, Module: addrs.RootModule}
	providerConfig := p.rootModule.ProviderConfigs[provider.Type]

	var rng hcl.Range
	if providerConfig != nil {
		// Annotate the result diagnostics with the local range so that diagnostics can be rendered with both the
		// policy source and the object being enforced.
		rng = providerConfig.DeclRange
		result = result.WithLocalRange(rng.Ptr())
	}
	p.view.PolicyResult(addr.String(), result, rng)
	log.Println("[DEBUG] init: policy result for provider", provider.String(), version, "overall", result.Overall)
	// Init uses diagnostics as the blocking signal because advisory policies
	// may return deny without any error diagnostics.
	if result.Diagnostics.HasErrors() {
		return fmt.Errorf("Provider download blocked due to policy violations. Please review other diagnostics for details.")
	}

	return nil
}
