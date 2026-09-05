// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

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

	t.Run("an apply validation error blocks provider operations", func(t *testing.T) {
		plan, planDiags := newContext().Plan(m, states.NewState(), &PlanOpts{Mode: plans.NormalMode})
		tfdiags.AssertNoDiagnostics(t, planDiags)
		provider.ApplyResourceChangeCalled = false

		policyClient := policy.NewTestMockClient(t)
		calls := 0
		policyClient.ValidateProviderSchemasFn = func(context.Context, policy.ValidateProviderSchemasRequest) policy.ValidateProviderSchemasResponse {
			calls++
			return policy.ValidateProviderSchemasResponse{Diagnostics: policy.Diagnostics{
				policy.NewErrorDiagnostic("Invalid policy", "references an attribute the provider does not have", policy.SetupErrorResult),
			}}
		}
		_, diags := newContext().Apply(plan, m, &ApplyOpts{PolicyClient: policyClient})
		if !diags.HasErrors() {
			t.Fatal("expected the schema-validation error to block apply")
		}
		if calls != 1 {
			t.Fatalf("ValidateProviderSchemas called %d times during apply, want once", calls)
		}
		if provider.ApplyResourceChangeCalled {
			t.Error("provider apply must not run after an early schema-validation failure")
		}
		if policyClient.EvaluateCalled || policyClient.EvaluateProviderCalled || policyClient.EvaluateModuleCalled {
			t.Error("policy evaluation must not run after an early schema-validation failure")
		}
	})
}

func TestValidateProviderSchemasRequest(t *testing.T) {
	testAddr := addrs.MustParseProviderSourceString("example/test")
	otherAddr := addrs.MustParseProviderSourceString("example/other")
	schemas := &Schemas{Providers: map[addrs.Provider]providers.ProviderSchema{
		testAddr: {
			Provider: providers.Schema{},
			ResourceTypes: map[string]providers.Schema{
				"test_empty": {},
			},
			DataSources: map[string]providers.Schema{
				"test_data": {},
			},
		},
		otherAddr: {Provider: providers.Schema{Body: &configschema.Block{}}},
	}}
	client := policy.NewTestMockClient(t)

	diags := validateProviderSchemas(t.Context(), client, schemas)
	tfdiags.AssertNoDiagnostics(t, diags)
	got := client.ValidateProviderSchemasRequest.ProviderSchemas
	if len(got) != 2 {
		t.Fatalf("got %d provider schemas, want 2", len(got))
	}
	if types := []string{got[0].Type, got[1].Type}; !slices.Equal(types, []string{"other", "test"}) {
		t.Fatalf("provider schemas are not ordered deterministically: %v", types)
	}
	if !got[1].Config.Equals(cty.EmptyObject) || !got[1].Resources["test_empty"].Equals(cty.EmptyObject) || !got[1].DataSources["test_data"].Equals(cty.EmptyObject) {
		t.Fatalf("nil schema bodies were not preserved as empty objects: %#v", got[1])
	}
}

func TestValidateProviderSchemasTypeCollision(t *testing.T) {
	first := addrs.MustParseProviderSourceString("example/test")
	second := addrs.MustParseProviderSourceString("other/test")
	client := policy.NewTestMockClient(t)
	schemas := &Schemas{Providers: map[addrs.Provider]providers.ProviderSchema{
		first:  {Provider: providers.Schema{Body: &configschema.Block{}}},
		second: {Provider: providers.Schema{Body: &configschema.Block{}}},
	}}

	diags := validateProviderSchemas(t.Context(), client, schemas)
	if !diags.HasErrors() {
		t.Fatal("expected colliding provider types to fail before the RPC")
	}
	if client.ValidateProviderSchemasCalled {
		t.Fatal("ambiguous provider schemas must not be sent to the policy plugin")
	}
	detail := diags.Err().Error()
	if !strings.Contains(detail, first.ForDisplay()) || !strings.Contains(detail, second.ForDisplay()) {
		t.Fatalf("collision diagnostic does not identify both providers: %s", detail)
	}
}

func TestContext2Plan_PolicySchemaValidationCancellation(t *testing.T) {
	m := testModuleInline(t, map[string]string{"main.tf": `resource "test_resource" "test" {}`})
	providerAddr := addrs.NewDefaultProvider("test")
	provider := testProvider("test")
	provider.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		Provider:      &configschema.Block{},
		ResourceTypes: map[string]*configschema.Block{"test_resource": {}},
	})
	c := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
		providerAddr: testProviderFuncFixed(provider),
	}})
	policyClient := policy.NewTestMockClient(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	policyClient.ValidateProviderSchemasFn = func(ctx context.Context, req policy.ValidateProviderSchemasRequest) policy.ValidateProviderSchemasResponse {
		close(entered)
		select {
		case <-ctx.Done():
		case <-release:
		}
		return policy.ValidateProviderSchemasResponse{}
	}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		c.Plan(m, states.NewState(), &PlanOpts{Mode: plans.NormalMode, PolicyClient: policyClient})
	}()

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("schema validation RPC was not entered")
	}
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		c.Stop()
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		close(release)
		<-stopped
		t.Fatal("schema validation did not receive run context cancellation")
	}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("plan did not finish after cancellation")
	}
}
