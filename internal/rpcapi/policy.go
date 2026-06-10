// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"fmt"
	"log"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/version"
)

// This is shares a lot of the same logic that will eventually be merged at
// - internal/command/meta_policy.go, from: https://github.com/hashicorp/terraform/pull/38518
//
// TODO: Potentially refactor/share that logic?
func initializePolicyClient(ctx context.Context, policyPluginPath string, policyPaths []string) (policy.Client, error) {
	log.Printf("[DEBUG] rpcapi: policy client setup with paths: %v", policyPaths)
	client, err := policy.Connect(ctx, policyPluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to policy engine: %w", err)
	}

	req := policy.SetupRequest{SourceLocations: policyPaths}
	if srv, ok := client.(policy.CallbackService); ok {
		callbackServer, cbDiags := srv.RegisterCallbackService(ctx)
		if cbDiags != nil {
			client.Stop()
			return nil, cbDiags.AsTerraformDiags().Err()
		}
		req.CallbackService = callbackServer.ID
	}

	resp := client.Setup(ctx, req)
	diags := resp.Diagnostics.AsTerraformDiags()
	if diags.HasErrors() {
		client.Stop()
		return nil, diags.Err()
	}

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
		return nil, diags.Err()
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
		return nil, diags.Err()
	}

	log.Printf("[DEBUG] rpcapi: Policy engine initialized")
	return client, nil
}
