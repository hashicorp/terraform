// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	policyproto "github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const relationshipsTestProvider = "registry.terraform.io/hashicorp/testing"

// relationshipsTestClient is a policy client with relationships. Its
// policies call getresources for testing_resource.
type relationshipsTestClient struct {
	*policy.MockClient

	mu sync.Mutex
	// getResources counts the testing_resource objects getresources returned
	// to an evaluation, by its target type.
	getResources map[string]int
}

func relationshipsTestPath(name string) *policyproto.AttributePath {
	return &policyproto.AttributePath{Steps: []*policyproto.AttributePath_Step{{Selector: &policyproto.AttributePath_Step_AttributeName{AttributeName: name}}}}
}

// relationshipsTestDefinition returns a relationship definition of the
// testing provider whose only key reads keyPath of keyType and valuePath of
// valueType.
func relationshipsTestDefinition(name, keyType, keyPath, valueType, valuePath string) *policyproto.RelationshipDefinition {
	return &policyproto.RelationshipDefinition{
		ProviderSource: relationshipsTestProvider,
		Name:           name,
		KeyResource:    keyType,
		ValueResource:  valueType,
		Match: []*policyproto.MatchAlternative{{Keys: []*policyproto.MatchKey{{
			KeyPath:   relationshipsTestPath(keyPath),
			ValuePath: relationshipsTestPath(valuePath),
		}}}},
		DeclRange: &policyproto.Range{
			Filename: "relationships.tfpolicy.hcl",
			Start:    &policyproto.Position{Line: 1, Column: 1},
			End:      &policyproto.Position{Line: 1, Column: 10, Byte: 9},
		},
	}
}

func relationshipsTestUsage(typ string, relationships ...string) *policyproto.RelationshipUsage {
	u := &policyproto.RelationshipUsage{ProviderSource: relationshipsTestProvider, Type: typ}
	for _, r := range relationships {
		u.Hops = append(u.Hops, &policyproto.RelationshipHop{Relationship: r})
	}
	return u
}

// newRelationshipsTestClient returns a client with relationship
// "block_bucket": a block (testing_deferred_resource) relates to the buckets
// (testing_resource) whose id is its value. Policies of both types use it.
func newRelationshipsTestClient(t *testing.T) *relationshipsTestClient {
	t.Helper()
	c := &relationshipsTestClient{
		MockClient:   policy.NewTestMockClient(t),
		getResources: make(map[string]int),
	}
	c.RelationshipsResponse = policy.Relationships{
		Definitions: []*policyproto.RelationshipDefinition{
			relationshipsTestDefinition("block_bucket", "testing_deferred_resource", "value", "testing_resource", "id"),
		},
		Usages: []*policyproto.RelationshipUsage{
			relationshipsTestUsage("testing_deferred_resource", "block_bucket"),
			relationshipsTestUsage("testing_resource", "block_bucket"),
		},
	}
	c.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*policyproto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		resources, _, err := req.Callbacks.GetResources(ctx, "testing_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("getresources failed: %s", err)
		}
		c.mu.Lock()
		c.getResources[req.Target] = len(resources)
		c.mu.Unlock()
		return policy.EvaluationResponse{
			Overall:      policy.AllowResult,
			Policies:     []*policy.Policy{mockPolicyObj(policy.AllowResult)},
			Enforcements: []policy.EnforcementResult{{Result: policy.AllowResult}},
		}
	}
	return c
}

// relationshipsTestTrees returns the related trees of every evaluated
// subject, by "ADDRESS OPERATION": a line per hop, "RELATIONSHIP
// COMPLETENESS [DEFERRED...]", and per member, "  ADDRESS OPERATION".
func relationshipsTestTrees(t *testing.T, c *relationshipsTestClient) map[string][]string {
	t.Helper()
	ret := make(map[string][]string)
	for _, req := range c.EvaluateRequests {
		key := req.Meta.Address + " " + req.Meta.Operation.String()
		if _, ok := ret[key]; ok {
			t.Fatalf("subject %s evaluated twice", key)
		}
		lines := []string{}
		for _, hop := range req.Related {
			line := []string{hop.Relationship, strings.TrimSuffix(hop.Completeness.String(), "_COMPLETENESS")}
			lines = append(lines, strings.Join(append(line, hop.Deferred...), " "))
			for _, m := range hop.Members {
				lines = append(lines, "  "+m.Address+" "+m.Operation.String())
			}
		}
		ret[key] = lines
	}
	return ret
}

func relationshipsTestLocks() depsfile.Locks {
	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.1.0"),
		providerreqs.MustParseVersionConstraints("0.1.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)
	return *lock
}

func relationshipsTestProviders(t *testing.T) map[addrs.Provider]providers.Factory {
	return map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
			return stacks_testing_provider.NewProvider(t), nil
		},
	}
}

// The block and the bucket are in different components; the deployment
// relates them, with component-qualified addresses.
func TestPlan_PolicyRelationshipsAcrossComponents(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-cross-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := map[string][]string{
		"component.block.testing_deferred_resource.block CREATE": {
			"block_bucket PLAN",
			"  component.bucket.testing_resource.bucket CREATE",
		},
		"component.bucket.testing_resource.bucket CREATE": {
			"block_bucket PLAN",
			"  component.block.testing_deferred_resource.block CREATE",
		},
	}
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong related (-want +got):\n%s", diff)
	}

	// getresources stays scoped to the component of the subject.
	wantGetResources := map[string]int{
		"testing_resource":          1,
		"testing_deferred_resource": 0,
	}
	if diff := cmp.Diff(wantGetResources, client.getResources); diff != "" {
		t.Errorf("wrong getresources results (-want +got):\n%s", diff)
	}
}

func TestApply_PolicyRelationshipsAcrossComponents(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "policy-relationships-cross-component")

	plan := planForApplyTest(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            cfg,
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
	})

	client := newRelationshipsTestClient(t)
	applyAndCollectPolicyResults(t, ctx, ApplyRequest{
		Config:            cfg,
		Plan:              plan,
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := map[string][]string{
		"component.block.testing_deferred_resource.block CREATE": {
			"block_bucket COMPLETE",
			"  component.bucket.testing_resource.bucket CREATE",
		},
		"component.bucket.testing_resource.bucket CREATE": {
			"block_bucket COMPLETE",
			"  component.block.testing_deferred_resource.block CREATE",
		},
	}
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong related (-want +got):\n%s", diff)
	}
}

// A deferred instance in one component is a deferred address of the hops
// that could include it in another.
func TestPlan_PolicyRelationshipsDeferredComponent(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-cross-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			{Name: "block_deferred"}: {Value: cty.True},
		},
	})

	want := map[string][]string{
		"component.bucket.testing_resource.bucket CREATE": {
			"block_bucket DEFERRED component.block.testing_deferred_resource.block",
		},
	}
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong related (-want +got):\n%s", diff)
	}
}

// The bucket already exists and doesn't change, so the apply skips its
// component. Its state is still a member, NO_OP, of the block's hop, like an
// unchanged instance of a Terraform CLI apply.
func TestApply_PolicyRelationshipsSkippedComponent(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "policy-relationships-cross-component")

	store := stacks_testing_provider.NewResourceStoreBuilder().
		AddResource("bucket", cty.ObjectVal(map[string]cty.Value{
			"id":    cty.StringVal("bucket"),
			"value": cty.StringVal("bucket"),
		})).
		Build()
	providerFactories := map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
			return stacks_testing_provider.NewProviderWithData(t, store), nil
		},
	}
	prevState := stackstate.NewStateBuilder().
		AddComponentInstance(stackstate.NewComponentInstanceBuilder(mustAbsComponentInstance("component.bucket"))).
		AddResourceInstance(stackstate.NewResourceInstanceBuilder().
			SetAddr(mustAbsResourceInstanceObject("component.bucket.testing_resource.bucket")).
			SetProviderAddr(mustDefaultRootProvider("testing")).
			SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
				Status: states.ObjectReady,
				AttrsJSON: mustMarshalJSONAttrs(map[string]any{
					"id":    "bucket",
					"value": "bucket",
				}),
			})).
		Build()

	plan := planForApplyTest(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            cfg,
		PrevState:         prevState,
		ProviderFactories: providerFactories,
		DependencyLocks:   relationshipsTestLocks(),
	})
	if c := plan.GetComponent(mustAbsComponentInstance("component.bucket")); c == nil || c.PlanApplyable {
		t.Fatalf("want the bucket's component not applyable, got %#v", c)
	}

	client := newRelationshipsTestClient(t)
	applyAndCollectPolicyResults(t, ctx, ApplyRequest{
		Config:            cfg,
		Plan:              plan,
		ProviderFactories: providerFactories,
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := map[string][]string{
		"component.block.testing_deferred_resource.block CREATE": {
			"block_bucket COMPLETE",
			"  component.bucket.testing_resource.bucket NO_OP",
		},
		"component.bucket.testing_resource.bucket NO_OP": {
			"block_bucket COMPLETE",
			"  component.block.testing_deferred_resource.block CREATE",
		},
	}
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong related (-want +got):\n%s", diff)
	}
}

// A component input that refers directly to another component's output
// carries the origins of that output, through other components too. The
// buckets' ids are unknown at plan, so only origins relate.
func TestPlan_PolicyRelationshipsOriginsAcrossComponents(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)
	client.RelationshipsResponse.Usages = client.RelationshipsResponse.Usages[:1]

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-origins"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := map[string][]string{
		"component.direct.testing_deferred_resource.block CREATE": {
			"block_bucket PLAN",
			"  component.bucket.testing_resource.bad CREATE",
		},
		"component.computed.testing_deferred_resource.block CREATE": {
			"block_bucket PLAN",
		},
		"component.chained.testing_deferred_resource.block CREATE": {
			"block_bucket PLAN",
			"  component.bucket.testing_resource.good CREATE",
		},
		`component.keyed["a"].testing_deferred_resource.block CREATE`: {
			"block_bucket PLAN",
			`  component.keyed_bucket["a"].testing_resource.good CREATE`,
		},
	}
	got := relationshipsTestTrees(t, client)
	for key := range got {
		if !strings.Contains(key, "testing_deferred_resource") {
			delete(got, key)
		}
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong related (-want +got):\n%s", diff)
	}
}

// Two instances of different components relate if they use the same stack
// provider configuration instance, or ones with equal configurations.
func TestPlan_PolicyRelationshipsProviderInstances(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)
	client.RelationshipsResponse = policy.Relationships{
		Definitions: []*policyproto.RelationshipDefinition{
			relationshipsTestDefinition("same_bucket", "testing_resource", "id", "testing_resource", "id"),
		},
		Usages: []*policyproto.RelationshipUsage{relationshipsTestUsage("testing_resource", "same_bucket")},
	}

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-providers"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := map[string][]string{
		"component.a1.testing_resource.bucket CREATE": {
			"same_bucket PLAN",
			"  component.a2.testing_resource.bucket CREATE",
			"  component.same_as_a.testing_resource.bucket CREATE",
		},
		"component.a2.testing_resource.bucket CREATE": {
			"same_bucket PLAN",
			"  component.a1.testing_resource.bucket CREATE",
			"  component.same_as_a.testing_resource.bucket CREATE",
		},
		"component.same_as_a.testing_resource.bucket CREATE": {
			"same_bucket PLAN",
			"  component.a1.testing_resource.bucket CREATE",
			"  component.a2.testing_resource.bucket CREATE",
		},
		"component.keyed.testing_resource.bucket CREATE": {
			"same_bucket PLAN",
		},
	}
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong related (-want +got):\n%s", diff)
	}
}

// A relationship definition that doesn't fit the provider's schema fails the
// plan before any component is planned.
func TestPlan_PolicyRelationshipsInvalidDefinition(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t)
	client.RelationshipsResponse.Definitions = []*policyproto.RelationshipDefinition{
		relationshipsTestDefinition("block_bucket", "testing_deferred_resource", "nope", "testing_resource", "id"),
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	go Plan(ctx, &PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-cross-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	}, &PlanResponse{PlannedChanges: changesCh, Diagnostics: diagsCh})
	changes, diags := collectPlanOutput(changesCh, diagsCh)

	if len(diags) != 1 {
		t.Fatalf("want one diagnostic, got %s", diags.ErrWithWarnings())
	}
	desc := diags[0].Description()
	wantDetail := "`nope` of testing_deferred_resource doesn't exist in the schema of " + relationshipsTestProvider + " v0.1.0."
	if diags[0].Severity() != tfdiags.Error || desc.Summary != `Invalid relationship "block_bucket"` || desc.Detail != wantDetail {
		t.Errorf("wrong diagnostic: %s: %s", desc.Summary, desc.Detail)
	}
	for _, change := range changes {
		if _, ok := change.(*stackplan.PlannedChangeResourceInstancePlanned); ok {
			t.Errorf("want no resource instance planned, got %#v", change)
		}
	}
	if len(client.EvaluateRequests) > 0 {
		t.Error("want no policy evaluated")
	}
}
