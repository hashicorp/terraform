// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The plan flow sends the run's provider schemas to the policy plugin before the
// walk, and a schema-validation error blocks the run before any policy is
// evaluated against a real resource.
func TestContext2Plan_PolicyValidateProviderSchemas(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source  = "hashicorp/test"
						version = "1.0.0"
					}
				}
			}

			resource "test_resource" "test" {
				value = "x"
			}
		`,
	})

	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"region": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id":    {Type: cty.String, Computed: true},
					"value": {Type: cty.String, Optional: true},
				},
			},
		},
	})

	newContext := func() *Context {
		return testContext2(t, &ContextOpts{
			Providers: map[addrs.Provider]providers.Factory{
				providerAddr: testProviderFuncFixed(provider),
			},
		})
	}

	t.Run("schemas are sent and a clean run proceeds", func(t *testing.T) {
		policyClient := policy.NewTestMockClient(t)
		_, diags := newContext().Plan(m, states.NewState(), &PlanOpts{
			Mode:         plans.NormalMode,
			PolicyClient: policyClient,
		})
		tfdiags.AssertNoDiagnostics(t, diags)

		if !policyClient.ValidateProviderSchemasCalled {
			t.Fatal("expected ValidateProviderSchemas to be called before the walk")
		}
		got := policyClient.ValidateProviderSchemasRequest.ProviderSchemas
		if len(got) != 1 || got[0].Type != "test" {
			t.Fatalf("expected the test provider schema, got %+v", got)
		}
		if _, ok := got[0].Resources["test_resource"]; !ok {
			t.Errorf("expected test_resource in the sent schema, got %v", got[0].Resources)
		}
	})

	t.Run("a validation error blocks the run early", func(t *testing.T) {
		policyClient := policy.NewTestMockClient(t)
		policyClient.ValidateProviderSchemasFn = func(context.Context, policy.ValidateProviderSchemasRequest) policy.ValidateProviderSchemasResponse {
			return policy.ValidateProviderSchemasResponse{Diagnostics: policy.Diagnostics{
				policy.NewErrorDiagnostic("Invalid policy", "references an attribute the provider does not have", policy.SetupErrorResult),
			}}
		}
		_, diags := newContext().Plan(m, states.NewState(), &PlanOpts{
			Mode:         plans.NormalMode,
			PolicyClient: policyClient,
		})
		if !diags.HasErrors() {
			t.Fatal("expected the schema-validation error to block the plan")
		}
		if policyClient.EvaluateCalled {
			t.Error("resource policy evaluation must not run after an early schema-validation failure")
		}
	})
}
