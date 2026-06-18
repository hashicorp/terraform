// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"log"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

func (c *Meta) PolicyClient(ctx context.Context, policyPaths []string) (policy.Client, policy.Diagnostics, func()) {
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
	client, err := policy.Connect(ctx)
	if err != nil {
		diags = append(diags, policy.NewErrorDiagnostic(
			"Failed to connect to policy engine",
			fmt.Sprintf("Failed to connect to policy engine: %s.", err),
			policy.SetupErrorResult,
		))
		return nil, diags, closer
	}

	var callbackServiceID uint32

	// initialize the callback service if the client supports it
	if srv, ok := client.(policy.CallbackService); ok {
		callbackServer, cbDiags := srv.RegisterCallbackService(ctx)
		if cbDiags != nil {
			return nil, cbDiags, closer
		}
		callbackServiceID = callbackServer.ID
	}

	resp := client.Setup(ctx, policy.SetupRequest{
		SourceLocations: policyPaths,
		CallbackService: callbackServiceID,
	})
	diags = append(diags, resp.Diagnostics...)

	var requiredVersions constraints.IntersectionSpec
	for _, config := range resp.ServerConfigurations() {
		version, err := constraints.ParseRubyStyleMulti(config.RequiredVersion)
		if err != nil {
			diags = append(diags, policy.NewErrorDiagnostic(
				"Failed to validate required Terraform version",
				fmt.Sprintf("The policy file %s had a Terraform version constraint that could not be parsed: %s.", config.File, err),
				policy.SetupErrorResult,
			))
			continue
		}

		requiredVersions = append(requiredVersions, version...)
	}

	if diags.HasErrors() {
		return client, diags, closer
	}

	terraformVersion, err := versions.ParseVersion(version.Version)
	if err != nil {
		client.Stop()
		// This is crazy, it means the internal version number is invalid.
		panic(err)
	}

	constraint := versions.MeetingConstraints(requiredVersions)
	if !constraint.Has(terraformVersion) {
		diags = append(diags, policy.NewErrorDiagnostic(
			"Invalid Terraform version for policies",
			fmt.Sprintf("The current version of Terraform is %s, and it is not compatible with the versions of Terraform required by the selected policies.", version.String()),
			policy.SetupErrorResult,
		))
		return nil, diags, closer
	}

	log.Printf("[DEBUG] backend/operation/policy: Policy engine initialized")
	return client, diags, closer
}

var _ initwd.ModuleInstallHook = &policyModuleInstallHook{}

// policyModuleInstallHook enables policy evaluation during module installation.
type policyModuleInstallHook struct {
	initwd.ModuleInstallHookImpl
	client        policy.Client
	rootModule    *configs.Module
	policyResults *plans.PolicyResults
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
		Attrs:  policy.PolicyValue{Raw: cty.DynamicVal},
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

	if moduleCall != nil {
		ptr := moduleCall.DeclRange.Ptr()
		for idx, diag := range result.Diagnostics {
			result.Diagnostics[idx] = diag.WithLocalRange(ptr)
		}
		for idx := range result.Enforcements {
			result.Enforcements[idx].LocalRange = ptr
		}
	}
	h.policyResults.AddModule(req.Path, result, moduleCall)

	// return a generic error here that the init command returns to the CLI.
	// The detailed policy diagnostics are included in the policy results
	// and will be formatted in the CLI output.
	allowed := result.Overall == policy.AllowResult || result.Overall == policy.UnknownResult
	if !allowed || result.Diagnostics.HasErrors() {
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
	client        policy.Client
	policyResults *plans.PolicyResults
	rootModule    *configs.Module
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
		Attrs: policy.PolicyValue{Raw: cty.DynamicVal},
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

	p.policyResults.AddProvider(addr, result, providerConfig)
	log.Println("[DEBUG] init: policy result for provider", provider.String(), version, "overall", result.Overall)
	allowed := result.Overall == policy.AllowResult || result.Overall == policy.UnknownResult
	if !allowed || result.Diagnostics.HasErrors() {
		return fmt.Errorf("Provider download blocked due to policy violations. Please review other diagnostics for details.")
	}

	return nil
}
