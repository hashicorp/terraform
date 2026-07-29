// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	providertest "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
)

func TestPlanPolicyValidationUsesGlobalProviderSchemas(t *testing.T) {
	first := addrs.MustParseProviderSourceString("example/first")
	second := addrs.MustParseProviderSourceString("example/second")
	cfg := testStackConfigEmpty(t)
	cfg.ProviderRefTypes = map[addrs.Provider]cty.Type{
		first:  cty.DynamicPseudoType,
		second: cty.DynamicPseudoType,
	}

	client := policy.NewTestMockClient(t)
	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode: plans.NormalMode,
		ProviderFactories: ProviderFactories{
			first:  policyValidationProviderFactory("first_resource"),
			second: policyValidationProviderFactory("second_resource"),
		},
		PolicyClient: client,
	})

	outp, tester := testPlanOutput(t)
	main.PlanAll(context.Background(), outp)
	if diags := tester.Diags(); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %s", diags.ErrWithWarnings())
	}

	got := client.ValidatePoliciesRequest.ProviderSchemas
	if len(got) != 2 {
		t.Fatalf("ValidatePolicies received %d provider schemas, want 2", len(got))
	}
	if got[0].Source != first.String() || got[1].Source != second.String() {
		t.Fatalf("ValidatePolicies received the wrong provider schemas: %#v", got)
	}
	if _, ok := got[0].Resources["first_resource"]; !ok {
		t.Fatalf("first provider schema is missing its resource type: %#v", got[0])
	}
	if _, ok := got[1].Resources["second_resource"]; !ok {
		t.Fatalf("second provider schema is missing its resource type: %#v", got[1])
	}
}

func TestPlanPolicyValidationDetectsGlobalProviderTypeCollision(t *testing.T) {
	first := addrs.MustParseProviderSourceString("example/test")
	second := addrs.MustParseProviderSourceString("other/test")
	cfg := testStackConfigEmpty(t)
	cfg.ProviderRefTypes = map[addrs.Provider]cty.Type{
		first:  cty.DynamicPseudoType,
		second: cty.DynamicPseudoType,
	}

	client := policy.NewTestMockClient(t)
	main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
		PlanningMode: plans.NormalMode,
		ProviderFactories: ProviderFactories{
			first:  policyValidationProviderFactory("example_resource"),
			second: policyValidationProviderFactory("other_resource"),
		},
		PolicyClient: client,
	})

	outp, tester := testPlanOutput(t)
	main.PlanAll(context.Background(), outp)
	diags := tester.Diags()
	if !diags.HasErrors() {
		t.Fatal("expected the global provider type collision to fail validation")
	}
	if client.ValidatePoliciesCalled {
		t.Fatal("ambiguous provider schemas must not be sent to the policy plugin")
	}
	detail := diags.Err().Error()
	if !strings.Contains(detail, first.ForDisplay()) || !strings.Contains(detail, second.ForDisplay()) {
		t.Fatalf("collision diagnostic does not identify both providers: %s", detail)
	}
}

func policyValidationProviderFactory(resourceType string) providers.Factory {
	return providers.FactoryFixed(&providertest.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Body: &configschema.Block{}},
			ResourceTypes: map[string]providers.Schema{
				resourceType: {Body: &configschema.Block{}},
			},
		},
	})
}
