// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/terraform/internal/policy"
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
	client, diags = policy.NewPolicyClient(ctx, os.Getenv(policy.TerraformPolicyPluginEnvVar), policyPaths)
	if diags.HasErrors() {
		return nil, diags, closer
	}

	log.Printf("[DEBUG] backend/operation/policy: Policy engine initialized")
	return client, diags, closer
}
