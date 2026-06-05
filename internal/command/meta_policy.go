// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"log"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/apparentlymart/go-versions/versions/constraints"

	"github.com/hashicorp/terraform/internal/policy"
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
		if client := c.testingOverrides.PolicyClient; client != nil {
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

	if len(diags) > 0 {
		return nil, diags, closer
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
