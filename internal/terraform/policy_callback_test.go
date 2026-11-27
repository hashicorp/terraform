// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestGetResourcesByIdentifierScope(t *testing.T) {
	graph := newPolicyWalkState(false)
	subject := mustResourceInstanceAddr("test_resource.subject")
	related := mustResourceInstanceAddr("test_resource.related")
	unrelated := mustResourceInstanceAddr("test_resource.unrelated")
	missing := mustResourceInstanceAddr("test_resource.missing")
	for _, addr := range []addrs.AbsResourceInstance{subject, related, unrelated} {
		graph.resourceMap[addr.String()] = &policyResource{
			Addr: addr, ProviderAddr: mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
			Value: policy.CtyToPolicyValue(cty.EmptyObjectVal), PriorValue: policy.CtyToPolicyValue(cty.EmptyObjectVal), Action: plans.NoOp,
		}
	}
	relations := []policy.Relation{{
		Relationship: "related",
		Resources: map[string][]policy.Relation{
			related.String(): {{Relationship: "nested", Resources: map[string][]policy.Relation{missing.String(): nil}}},
		},
	}}
	callback := (&PolicyCallbackManager{}).GetResourcesByIdentifierCallback(&MockEvalContext{PolicyStateValue: graph}, subject, relations)

	identifiers := []string{related.String(), subject.String(), related.String()}
	resources, err := callback(context.Background(), identifiers)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != len(identifiers) {
		t.Fatalf("got %d resources, want %d", len(resources), len(identifiers))
	}
	for i, resource := range resources {
		if resource.Identifier != identifiers[i] {
			t.Errorf("resource %d identifier = %q, want %q", i, resource.Identifier, identifiers[i])
		}
	}
	// Another evaluation of the same subject must use only its own Relations.
	subjectOnly := (&PolicyCallbackManager{}).GetResourcesByIdentifierCallback(&MockEvalContext{PolicyStateValue: graph}, subject, nil)
	if resources, err := subjectOnly(context.Background(), []string{related.String()}); err == nil || resources != nil {
		t.Errorf("callback without Relations allowed a related identifier: %v, %v", resources, err)
	}

	for name, identifier := range map[string]string{
		"unknown": "not-an-identifier", "unsent": unrelated.String(), "missing snapshot": missing.String(),
	} {
		t.Run(name, func(t *testing.T) {
			resources, err := callback(context.Background(), []string{subject.String(), identifier})
			if err == nil || resources != nil {
				t.Errorf("expected an error and no partial results, got %v, %v", resources, err)
			}
		})
	}
	if resources, err := callback(context.Background(), nil); err != nil || len(resources) != 0 {
		t.Errorf("empty batch: got %v, %v", resources, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := callback(cancelled, identifiers); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled callback: got %v, want context.Canceled", err)
	}
}

func TestGetResourcesByIdentifierPlanAndApply(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
resource "test_resource" "a" {
  value = "new"
}
resource "test_resource" "b" {
  value = "b"
  parent = test_resource.a.value
}
`})
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {Attributes: map[string]*configschema.Attribute{
				"value":  {Type: cty.String, Required: true, Sensitive: true},
				"parent": {Type: cty.String, Optional: true},
			}},
		},
	})
	ctx := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
	}})
	state := states.NewState()
	state.SyncWrapper().SetResourceInstanceCurrent(mustResourceInstanceAddr("test_resource.a"), &states.ResourceInstanceObjectSrc{
		Status: states.ObjectReady, AttrsJSON: []byte(`{"value":"old","parent":null}`),
	}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))

	for _, phase := range []string{"plan", "apply"} {
		t.Run(phase, func(t *testing.T) {
			client := policy.NewTestMockClient(t)
			client.ConfigValue.Relationships = map[string][]policy.Relationship{"test_resource": {{
				Name: "family", KeyType: "test_resource", ValueType: "test_resource",
				Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("value"), ValuePath: cty.GetAttrPath("parent")}}},
			}}}
			calls := 0
			client.EvaluateFn = func(callbackCtx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
				calls++
				resources, err := req.Callbacks.GetResourcesByIdentifier(callbackCtx, []string{req.Identifier})
				if err != nil || len(resources) != 1 {
					t.Errorf("%s: fetch failed: %v", req.Identifier, err)
					return policy.EvaluationResponse{}
				}
				resource := resources[0]
				if resource.Type != req.Target {
					t.Errorf("%s: type = %q, want %q", req.Identifier, resource.Type, req.Target)
				}
				if diff := cmp.Diff(req.Meta, resource.Metadata, protocmp.Transform()); diff != "" {
					t.Errorf("%s: wrong fetched metadata (-want +got):\n%s", req.Identifier, diff)
				}
				for name, pair := range map[string]struct {
					got  *proto.ResourceAttributes
					want policy.PolicyValue
				}{"attrs": {resource.Attrs, req.Attrs}, "prior_attrs": {resource.PriorAttrs, req.PriorAttrs}} {
					value, err := msgpack.Unmarshal(pair.got.Raw, cty.DynamicPseudoType)
					if err != nil || !value.RawEquals(pair.want.Raw) {
						t.Errorf("%s: fetched %s differ from evaluation: %v", req.Identifier, name, err)
					}
					want, err := policy.ResourceAttributesToProto(pair.want)
					if err != nil {
						t.Errorf("%s: failed to serialize expected %s: %v", req.Identifier, name, err)
						continue
					}
					if diff := cmp.Diff(want.RedactedPaths, pair.got.RedactedPaths, protocmp.Transform()); diff != "" {
						t.Errorf("%s: wrong %s redaction (-want +got):\n%s", req.Identifier, name, diff)
					}
				}
				return policy.EvaluationResponse{}
			}
			planOpts := &PlanOpts{Mode: plans.NormalMode}
			if phase == "plan" {
				planOpts.PolicyClient = client
			}
			plan, diags := ctx.Plan(mod, state, planOpts)
			tfdiags.AssertNoErrors(t, diags)
			if phase == "apply" {
				_, diags = ctx.Apply(plan, mod, &ApplyOpts{PolicyClient: client})
				tfdiags.AssertNoErrors(t, diags)
			}
			if calls != 2 {
				t.Errorf("got %d evaluations, want 2", calls)
			}
		})
	}
}

func TestGetResourcesByIdentifierPartialApply(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
resource "test_resource" "a" {
  value = "new"
}
resource "test_resource" "b" {
  value = "b"
  parent = "new"
}
`})
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {Attributes: map[string]*configschema.Attribute{
				"value":  {Type: cty.String, Required: true},
				"parent": {Type: cty.String, Optional: true},
			}},
		},
	})
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		response := providers.ApplyResourceChangeResponse{NewState: req.PlannedState}
		if req.PlannedState.GetAttr("value").AsString() == "new" {
			response.Diagnostics = response.Diagnostics.Append(tfdiags.Sourceless(tfdiags.Error, "Apply failed", "Partial state returned."))
		}
		return response
	}
	ctx := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
	}})
	state := states.NewState()
	state.SyncWrapper().SetResourceInstanceCurrent(mustResourceInstanceAddr("test_resource.a"), &states.ResourceInstanceObjectSrc{
		Status: states.ObjectReady, AttrsJSON: []byte(`{"value":"old","parent":null}`),
	}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
	plan, diags := ctx.Plan(mod, state, &PlanOpts{Mode: plans.NormalMode})
	tfdiags.AssertNoErrors(t, diags)

	client := policy.NewTestMockClient(t)
	client.ConfigValue.Relationships = map[string][]policy.Relationship{"test_resource": {{
		Name: "family", KeyType: "test_resource", ValueType: "test_resource",
		Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("value"), ValuePath: cty.GetAttrPath("parent")}}},
	}}}
	fetched := false
	client.EvaluateFn = func(callbackCtx context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		if req.Identifier != "test_resource.b" {
			return policy.EvaluationResponse{}
		}
		resources, err := req.Callbacks.GetResourcesByIdentifier(callbackCtx, []string{"test_resource.a"})
		if err != nil || len(resources) != 1 {
			t.Errorf("failed to fetch partial state: %v", err)
			return policy.EvaluationResponse{}
		}
		fetched = true
		resource := resources[0]
		if resource.Metadata.Operation != proto.Operation_UPDATE {
			t.Errorf("partial state operation = %s, want UPDATE", resource.Metadata.Operation)
		}
		for name, pair := range map[string]struct {
			attrs *proto.ResourceAttributes
			want  string
		}{"attrs": {resource.Attrs, "new"}, "prior_attrs": {resource.PriorAttrs, "old"}} {
			value, err := msgpack.Unmarshal(pair.attrs.Raw, cty.DynamicPseudoType)
			if err != nil {
				t.Errorf("failed to decode %s: %v", name, err)
				continue
			}
			if got := value.GetAttr("value").AsString(); got != pair.want {
				t.Errorf("%s value = %q, want %q", name, got, pair.want)
			}
		}
		return policy.EvaluationResponse{}
	}
	_, diags = ctx.Apply(plan, mod, &ApplyOpts{PolicyClient: client})
	if !diags.HasErrors() {
		t.Fatal("expected the apply error")
	}
	if !fetched {
		t.Error("successful subject did not fetch the failed resource's partial state")
	}
}

func TestGetResourcesByIdentifierSnapshot(t *testing.T) {
	graph := newPolicyWalkState(false)
	subject := mustResourceInstanceAddr("test_resource.subject")
	identifier := "opaque:walk:component:related"
	original := cty.ObjectVal(map[string]cty.Value{"value": cty.StringVal("original").Mark(marks.Sensitive)})
	prior := cty.ObjectVal(map[string]cty.Value{"value": cty.StringVal("prior")}).Mark(marks.Sensitive)
	prepared := policy.CtyToPolicyValue(original)
	preparedPrior := policy.CtyToPolicyValue(prior)
	if prepared.Raw.ContainsMarked() || preparedPrior.Raw.ContainsMarked() {
		t.Fatal("prepared inventory values must be deeply unmarked")
	}
	resource := &policyResource{
		Addr:         mustResourceInstanceAddr("test_resource.related"),
		ProviderAddr: mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		Value:        prepared,
		PriorValue:   preparedPrior,
		Action:       plans.NoOp,
	}
	graph.resourceMap[identifier] = resource
	callback := (&PolicyCallbackManager{}).GetResourcesByIdentifierCallback(&MockEvalContext{PolicyStateValue: graph}, subject, []policy.Relation{{
		Relationship: "related", Resources: map[string][]policy.Relation{identifier: nil},
	}})

	// Bind records, not a live inventory lookup. cty values themselves are immutable.
	resource.Value = policy.CtyToPolicyValue(cty.EmptyObjectVal)
	resource.Action = plans.Update
	delete(graph.resourceMap, identifier)
	for range 2 {
		responses, err := callback(context.Background(), []string{identifier, identifier})
		if err != nil {
			t.Fatal(err)
		}
		for _, response := range responses {
			if response.Identifier != identifier || response.Metadata.Operation != proto.Operation_NO_OP {
				t.Fatalf("callback did not retain its bound snapshot: %v", response)
			}
			for _, pair := range []struct {
				got  *proto.ResourceAttributes
				want policy.PolicyValue
			}{{response.Attrs, prepared}, {response.PriorAttrs, preparedPrior}} {
				value, err := msgpack.Unmarshal(pair.got.Raw, cty.DynamicPseudoType)
				if err != nil || !value.RawEquals(pair.want.Raw) {
					t.Fatalf("got %#v (%v), want the original value", value, err)
				}
				want, err := policy.ResourceAttributesToProto(pair.want)
				if err != nil {
					t.Fatal(err)
				}
				if len(pair.got.RedactedPaths) != 1 {
					t.Fatal("callback lost the sensitive path")
				}
				if diff := cmp.Diff(want, pair.got, protocmp.Transform()); diff != "" {
					t.Fatalf("callback changed prepared attributes (-want +got):\n%s", diff)
				}
			}
		}
		responses[0].Attrs.RedactedPaths[0].Steps[0].Selector = &proto.AttributePath_Step_AttributeName{AttributeName: "changed"}
		if responses[1].Attrs.RedactedPaths[0].Steps[0].GetAttributeName() != "value" || prepared.RedactedPaths[0][0].(cty.GetAttrStep).Name != "value" {
			t.Fatal("callback redaction paths share mutable storage")
		}
		responses[0].Metadata.Address = "changed"
		firstByte := responses[1].Attrs.Raw[0]
		responses[0].Attrs.Raw[0] ^= 0xff
		if responses[1].Metadata.Address != identifier || responses[1].Attrs.Raw[0] != firstByte {
			t.Fatal("callback responses share mutable protobuf data")
		}
	}

	for _, tc := range []struct{ identifier, want string }{
		{subject.String(), "is unavailable"},
		{"not-emitted", "was not sent"},
	} {
		if _, err := callback(context.Background(), []string{tc.identifier}); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want %q", tc.identifier, err, tc.want)
		}
	}
}
