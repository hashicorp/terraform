// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"log"
	"maps"
	"slices"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

func (c *Meta) PolicyClient(ctx context.Context, policyPaths []string) (policy.Client, policy.Diagnostics) {
	var client policy.Client
	// Policies are currently only supported in alpha versions.
	// TODO: Uncomment in the public release
	// if !c.AllowExperimentalFeatures {
	// 	log.Printf("[DEBUG] Policies are not supported, skipping policy client setup")
	// 	return client, nil
	// }
	if len(policyPaths) == 0 {
		log.Printf("[DEBUG] No policy paths configured, skipping policy client setup")
		return client, nil
	}

	// Use a pre-initialized client for tests if one is available
	if c.testingOverrides != nil {
		if client := c.testingOverrides.PolicyClient; client != nil {
			return client, nil
		}
	}

	var diags policy.Diagnostics
	client, err := policy.Connect(ctx)
	if client == nil {
		diags = append(diags, policy.NewErrorDiagnostic(
			"Failed to connect to policy engine",
			fmt.Sprintf("Failed to connect to policy engine: %s.", err),
			policy.SetupErrorResult,
		))
		return nil, diags
	}

	var callbackServiceID uint32

	// initialize the callback service if the client supports it
	if srv, ok := client.(policy.CallbackService); ok {
		callbackServer, cbDiags := srv.RegisterCallbackService(ctx)
		if cbDiags != nil {
			return nil, cbDiags
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

	if len(diags) > 0 {
		client.Stop()
		return nil, diags
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
		client.Stop()
		return nil, diags
	}

	log.Printf("[INFO] backend/operation/policy: Policy engine initialized")
	return client, diags
}

// policyModuleInstallHook implements initwd.ModuleInstallHook and
// enables policy evaluation during module installation.
type policyModuleInstallHook struct {
	initwd.ModuleInstallHooksImpl
	client        policy.Client
	rootModule    *configs.Module
	policyResults *plans.PolicyResults
}

func (h *policyModuleInstallHook) EvaluatePolicy(ctx context.Context, req *configs.ModuleRequest, source, version string) tfdiags.Diagnostics {
	moduleAddr := req.Path.String()
	moduleCall := h.rootModule.ModuleCalls[req.Name]
	result := h.client.EvaluateModule(ctx, policy.EvaluationRequest[*proto.ModuleMetadata]{
		Attrs:  cty.NilVal,
		Target: source,
		Meta: &proto.ModuleMetadata{
			Address: moduleAddr,
			Source:  source,
			Version: version,
		},
	})

	if moduleCall != nil && moduleCall.Config != nil {
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
	if len(result.Diagnostics) > 0 && result.Diagnostics.AsTerraformDiags().HasErrors() {
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

type providerInstallerHook struct {
	Reqs          *configs.ModuleRequirements
	Client        policy.Client
	moduleMap     map[addrs.Provider]string
	policyResults *plans.PolicyResults
	config        *configs.Config
}

func (p *providerInstallerHook) moduleSources() map[addrs.Provider]string {
	if p.moduleMap != nil {
		return p.moduleMap
	}
	// We iterate through the module requirements to build a map of providers
	// to their module source addresses. The first module requirement we encounter
	// for each provider will be recorded as the provider's module.
	// This matches how Terraform adds providers to the graph.
	p.moduleMap = map[addrs.Provider]string{}
	moduleReqs := []*configs.ModuleRequirements{p.Reqs}
	for len(moduleReqs) != 0 {
		moduleReq := moduleReqs[0]
		for reqProvider := range moduleReq.Requirements {
			if _, ok := p.moduleMap[reqProvider]; ok {
				// if we already have a module for this provider, skip
				continue
			}

			// The source is nil in the root module, so we use the root module address.
			if moduleReq.SourceAddr == nil {
				p.moduleMap[reqProvider] = addrs.RootModule.String()
			} else {
				p.moduleMap[reqProvider] = moduleReq.SourceAddr.String()
			}
		}

		newReqs := slices.Collect(maps.Values(moduleReq.Children))
		moduleReqs = append(moduleReqs[1:], newReqs...)
	}
	return p.moduleMap
}

func (p *providerInstallerHook) EvaluatePolicy(ctx context.Context, provider addrs.Provider, version string) policy.EvaluationResponse {
	// If the client is nil, then policy evaluation is disabled, so we can skip.
	if p.Client == nil {
		return policy.EvaluationResponse{}
	}
	moduleSources := p.moduleSources()
	log.Println("[DEBUG] init: evaluating policy for provider", provider.String(), version)
	result := p.Client.EvaluateProvider(ctx, policy.EvaluationRequest[*proto.ProviderMetadata]{
		Target: provider.Type,

		// Configuration attributes may not be available during init, so we will not
		// send any attributes to the policy client.
		Attrs: cty.NilVal,
		Meta: &proto.ProviderMetadata{
			Name:       provider.Type,
			Namespace:  provider.Namespace,
			Type:       provider.Type,
			Source:     provider.String(),
			ModulePath: moduleSources[provider],
			Version:    version,
		},
	})
	// We use the root module as the module for provider configs since the version resolution
	// is ambiguous, and we do not know which module the provider config belongs to.
	addr := addrs.AbsProviderConfig{Provider: provider, Module: addrs.RootModule}
	providerConfig := p.config.Module.ProviderConfigs[provider.Type]

	p.policyResults.AddProvider(addr, result, providerConfig)

	return result
}
