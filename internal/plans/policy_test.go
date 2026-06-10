// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/policy"
)

func TestPolicyResults(t *testing.T) {
	resourceAddr, _ := addrs.ParseAbsResourceInstanceStr("test_instance.example")
	providerAddr, _ := addrs.ParseAbsProviderConfigStr("provider[\"registry.terraform.io/hashicorp/test\"]")
	moduleAddr := addrs.Module{"child"}
	resourceConfig := &configs.Resource{DeclRange: hcl.Range{Filename: "resource.tf", Start: hcl.Pos{Line: 10, Column: 1}, End: hcl.Pos{Line: 10, Column: 20}}}
	providerConfig := &configs.Provider{DeclRange: hcl.Range{Filename: "provider.tf", Start: hcl.Pos{Line: 20, Column: 1}, End: hcl.Pos{Line: 20, Column: 20}}}
	moduleConfig := &configs.ModuleCall{DeclRange: hcl.Range{Filename: "module.tf", Start: hcl.Pos{Line: 30, Column: 1}, End: hcl.Pos{Line: 30, Column: 20}}}

	t.Run("empty result", func(t *testing.T) {
		pr := NewPolicyResults()
		// this is an empty result because it contains no diagnostics or enforcements
		allow := policy.EvaluationResponse{Overall: policy.AllowResult}

		pr.AddResource(resourceAddr, allow, resourceConfig)
		pr.AddProvider(providerAddr, allow, providerConfig)
		pr.AddModule(moduleAddr, allow, moduleConfig)

		// Empty results should be skipped, so the length should still be 0
		if got := pr.Len(); got != 0 {
			t.Fatalf("unexpected number of stored results: got %d, want 0", got)
		}
	})

	t.Run("Add non-empty result", func(t *testing.T) {
		pr := NewPolicyResults()
		resourceResult := policy.EvaluationResponse{Overall: policy.DenyResult}
		providerResult := policy.EvaluationResponse{Overall: policy.PolicyErrorResult}
		moduleResult := policy.EvaluationResponse{Overall: policy.DenyResult}

		pr.AddResource(resourceAddr, resourceResult, resourceConfig)
		pr.AddProvider(providerAddr, providerResult, providerConfig)
		pr.AddModule(moduleAddr, moduleResult, moduleConfig)

		if got := pr.Len(); got != 3 {
			t.Fatalf("unexpected number of stored results: got %d, want 3", got)
		}

		got := map[string]PolicyEvaluation{}
		for addr, result := range pr.Iter() {
			got[addr] = result
		}

		want := map[string]PolicyEvaluation{
			resourceAddr.String(): {
				EvaluationResponse: resourceResult,
				ConfigDeclRange:    resourceConfig.DeclRange,
			},
			providerAddr.String(): {
				EvaluationResponse: providerResult,
				ConfigDeclRange:    providerConfig.DeclRange,
			},
			moduleAddr.String(): {
				EvaluationResponse: moduleResult,
				ConfigDeclRange:    moduleConfig.DeclRange,
			},
		}

		if len(got) != len(want) {
			t.Fatalf("unexpected number of iterated results: got %d, want %d", len(got), len(want))
		}

		if diff := cmp.Diff(got, want); diff != "" {
			t.Fatalf("unexpected results: %s", diff)
		}
	})
}
