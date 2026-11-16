// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"sort"
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
func relationshipsTestDefinition(name, keyType, keyPath, valueType, valuePath string) *policyproto.Relationship {
	return &policyproto.Relationship{
		ProviderSource: relationshipsTestProvider,
		Name:           name,
		KeyType:        keyType,
		ValueType:      valueType,
		Matches: []*policyproto.Relationship_Match{{Keys: []*policyproto.Relationship_Key{{
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

// relationshipsTestTree returns the tree of a subject type of the testing
// provider, with one node per definition.
func relationshipsTestTree(typ string, defs ...*policyproto.Relationship) *policyproto.RelationshipTree {
	return &policyproto.RelationshipTree{ProviderSource: relationshipsTestProvider, ResourceType: typ, Relationships: defs}
}

// newRelationshipsTestClient returns a client with relationship
// "block_bucket": a block (testing_deferred_resource) relates to the buckets
// (testing_resource) whose keyPath is its value. Policies of both types use
// it.
func newRelationshipsTestClient(t *testing.T, bucketPath string) *relationshipsTestClient {
	t.Helper()
	c := &relationshipsTestClient{
		MockClient:   policy.NewTestMockClient(t),
		getResources: make(map[string]int),
	}
	def := relationshipsTestDefinition("block_bucket", "testing_deferred_resource", "value", "testing_resource", bucketPath)
	c.RelationshipsResponse = policy.Relationships{
		Trees: []*policyproto.RelationshipTree{
			relationshipsTestTree("testing_deferred_resource", def),
			relationshipsTestTree("testing_resource", def),
		},
	}
	c.EvaluateFn = func(ctx context.Context, req policy.EvaluationRequest[*policyproto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		resources, _, err := req.Callbacks.GetResources(ctx, "testing_resource", cty.NullVal(cty.DynamicPseudoType))
		if err != nil {
			t.Errorf("getresources failed: %s", err)
		}
		c.mu.Lock()
		c.getResources[relationshipsTestSubject(req)] = len(resources)
		c.mu.Unlock()
		return policy.EvaluationResponse{
			Overall:      policy.AllowResult,
			Policies:     []*policy.Policy{mockPolicyObj(policy.AllowResult)},
			Enforcements: []policy.EnforcementResult{{Result: policy.AllowResult}},
		}
	}
	return c
}

// relationshipsTestSubject returns the key of an evaluated subject, "ID
// ADDRESS OPERATION": its id attribute, which tells the component instance,
// since addresses are relative to it.
func relationshipsTestSubject(req policy.EvaluationRequest[*policyproto.PolicyEvaluateResourceRequest_ResourceMetadata]) string {
	return req.Object.Attrs.GetAttr("id").AsString() + " " + req.Meta.Address + " " + req.Meta.Operation.String()
}

// relationshipsTestTrees returns the relations of every evaluated subject, by
// subject key: a line per relation, "RELATIONSHIP COMPLETENESS
// [DEFERRED...]", and per related resource, "  KEY", the key of the subject
// with its identifier. It fails the test if a subject is evaluated twice, if
// two subjects share an identifier, if the subjects of a component instance
// don't share a generation or two component instances do, or if a related
// resource isn't a subject.
func relationshipsTestTrees(t *testing.T, c *relationshipsTestClient) map[string][]string {
	t.Helper()
	keys := make(map[string]string)
	generations := make(map[uint64]string)
	for _, req := range c.EvaluateRequests {
		key := relationshipsTestSubject(req)
		id := req.Object.Identifier()
		if other, ok := keys[id]; ok {
			t.Fatalf("%s and %s have the same identifier", key, other)
		}
		keys[id] = key
		// The component instance's name ends the id attribute.
		component := key[strings.Index(key, "-")+1 : strings.Index(key, " ")]
		if other, ok := generations[req.Object.Generation]; ok && other != component {
			t.Errorf("component instances %s and %s have the same generation", component, other)
		}
		generations[req.Object.Generation] = component
	}
	if len(keys) != len(c.EvaluateRequests) {
		t.Fatal("a subject was evaluated twice")
	}

	ret := make(map[string][]string)
	for _, req := range c.EvaluateRequests {
		lines := []string{}
		for _, rel := range req.Relationships {
			line := []string{rel.Relationship, strings.TrimSuffix(rel.Completeness.String(), "_COMPLETENESS")}
			lines = append(lines, strings.Join(append(line, rel.Deferred...), " "))
			var members []string
			for id := range rel.Resources {
				key, ok := keys[id]
				if !ok {
					t.Fatalf("related resource %s isn't a subject", id)
				}
				members = append(members, "  "+key)
			}
			sort.Strings(members)
			lines = append(lines, members...)
		}
		ret[relationshipsTestSubject(req)] = lines
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

// Each component instance's walk relates its own instances, which have the
// same addresses and values as the other's, and has its own identifiers.
func TestPlan_PolicyRelationshipsPerComponent(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t, "value")

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-per-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := relationshipsTestPairs("CREATE")
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong relations (-want +got):\n%s", diff)
	}

	// getresources stays scoped to the component of the subject.
	wantGetResources := map[string]int{
		"block-a testing_deferred_resource.block CREATE": 1,
		"block-b testing_deferred_resource.block CREATE": 1,
		"bucket-a testing_resource.bucket CREATE":        1,
		"bucket-b testing_resource.bucket CREATE":        1,
	}
	if diff := cmp.Diff(wantGetResources, client.getResources); diff != "" {
		t.Errorf("wrong getresources results (-want +got):\n%s", diff)
	}
}

func TestApply_PolicyRelationshipsPerComponent(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "policy-relationships-per-component")

	plan := planForApplyTest(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            cfg,
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
	})

	client := newRelationshipsTestClient(t, "value")
	applyAndCollectPolicyResults(t, ctx, ApplyRequest{
		Config:            cfg,
		Plan:              plan,
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := relationshipsTestPairs("CREATE")
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong relations (-want +got):\n%s", diff)
	}
}

// relationshipsTestPairs returns the relations of the instances of
// policy-relationships-per-component: each block and bucket relates to the
// other of its component instance only.
func relationshipsTestPairs(op string) map[string][]string {
	ret := make(map[string][]string)
	for _, name := range []string{"a", "b"} {
		block := "block-" + name + " testing_deferred_resource.block " + op
		bucket := "bucket-" + name + " testing_resource.bucket " + op
		ret[block] = []string{"block_bucket COMPLETE", "  " + bucket}
		ret[bucket] = []string{"block_bucket COMPLETE", "  " + block}
	}
	return ret
}

// A deferred instance is a deferred address of its own component instance's
// relations only.
func TestPlan_PolicyRelationshipsDeferredInComponent(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t, "value")

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-per-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			{Name: "b_deferred"}: {Value: cty.True},
		},
	})

	want := relationshipsTestPairs("CREATE")
	delete(want, "block-b testing_deferred_resource.block CREATE")
	want["bucket-b testing_resource.bucket CREATE"] = []string{"block_bucket DEFERRED testing_deferred_resource.block"}
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong relations (-want +got):\n%s", diff)
	}
}

// The block and the bucket are in different components, so they aren't
// related, although the block's value is the bucket's id.
func TestPlan_PolicyRelationshipsNotAcrossComponents(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t, "id")

	planAndCollectPolicyResults(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-cross-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := map[string][]string{
		"block testing_deferred_resource.block CREATE": {"block_bucket COMPLETE"},
		"bucket testing_resource.bucket CREATE":        {"block_bucket COMPLETE"},
	}
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong relations (-want +got):\n%s", diff)
	}
}

// The bucket already exists and doesn't change, so the apply skips its
// component. Like an unchanged instance of a Terraform CLI apply, it is
// still evaluated, NO_OP, with its component's relations: none.
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

	client := newRelationshipsTestClient(t, "id")
	applyAndCollectPolicyResults(t, ctx, ApplyRequest{
		Config:            cfg,
		Plan:              plan,
		ProviderFactories: providerFactories,
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	want := map[string][]string{
		"block testing_deferred_resource.block CREATE": {"block_bucket COMPLETE"},
		"bucket testing_resource.bucket NO_OP":         {"block_bucket COMPLETE"},
	}
	if diff := cmp.Diff(want, relationshipsTestTrees(t, client)); diff != "" {
		t.Errorf("wrong relations (-want +got):\n%s", diff)
	}
}

// A relationship definition that doesn't fit the provider's schema fails the
// deployment before any component instance plans or applies.
func TestPlan_PolicyRelationshipsInvalidDefinition(t *testing.T) {
	ctx := context.Background()
	client := newRelationshipsTestClient(t, "nope")

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	go Plan(ctx, &PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            loadMainBundleConfigForTest(t, "policy-relationships-per-component"),
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	}, &PlanResponse{PlannedChanges: changesCh, Diagnostics: diagsCh})
	changes, diags := collectPlanOutput(changesCh, diagsCh)

	relationshipsTestInvalid(t, diags)
	for _, change := range changes {
		if _, ok := change.(*stackplan.PlannedChangeResourceInstancePlanned); ok {
			t.Errorf("want no resource instance planned, got %#v", change)
		}
	}
	if len(client.EvaluateRequests) > 0 {
		t.Error("want no policy evaluated")
	}
}

func TestApply_PolicyRelationshipsInvalidDefinition(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "policy-relationships-per-component")

	plan := planForApplyTest(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            cfg,
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
	})

	client := newRelationshipsTestClient(t, "nope")
	changesCh := make(chan stackstate.AppliedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	go Apply(ctx, &ApplyRequest{
		Config:            cfg,
		Plan:              plan,
		ProviderFactories: relationshipsTestProviders(t),
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	}, &ApplyResponse{AppliedChanges: changesCh, Diagnostics: diagsCh})
	changes, diags := collectApplyOutput(changesCh, diagsCh)

	relationshipsTestInvalid(t, diags)
	for _, change := range changes {
		if ri, ok := change.(*stackstate.AppliedChangeResourceInstanceObject); ok && ri.NewStateSrc != nil {
			t.Errorf("want no resource instance applied, got %s", ri.ResourceInstanceObjectAddr)
		}
	}
	if len(client.EvaluateRequests) > 0 {
		t.Error("want no policy evaluated")
	}
}

// relationshipsTestInvalid fails the test unless diags is the one error of
// the invalid definition of block_bucket.
func relationshipsTestInvalid(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	var errs tfdiags.Diagnostics
	for _, diag := range diags {
		if diag.Severity() == tfdiags.Error {
			errs = errs.Append(diag)
		}
	}
	if len(errs) != 1 {
		t.Fatalf("want one error, got %s", diags.ErrWithWarnings())
	}
	desc := errs[0].Description()
	wantDetail := "`nope` of testing_resource doesn't exist in the schema of " + relationshipsTestProvider + " v0.1.0."
	if desc.Summary != `Invalid relationship "block_bucket"` || desc.Detail != wantDetail {
		t.Errorf("wrong diagnostic: %s: %s", desc.Summary, desc.Detail)
	}
}
