// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/ctyutil"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestPolicyRelationCompleteness(t *testing.T) {
	tests := map[string]struct {
		deferredType string
		expansion    bool
		unknownType  string
		wantComplete bool
		wantNested   bool
	}{
		"complete":                    {wantComplete: true, wantNested: true},
		"deferred instance":           {deferredType: "test_access_block", wantNested: true},
		"deferred expansion":          {deferredType: "test_access_block", expansion: true, wantNested: true},
		"unknown expansion":           {unknownType: "test_access_block", wantNested: true},
		"nested deferred instance":    {deferredType: "test_rule", wantComplete: true},
		"unrelated unknown expansion": {unknownType: "test_other", wantComplete: true, wantNested: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := policy.NewTestMockClient(t)
			client.ConfigValue.Relationships = map[string][]policy.Relationship{
				"test_bucket": {{
					Name: "access_blocks", KeyType: "test_bucket", ValueType: "test_access_block",
					Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}}},
					Relationships: []policy.Relationship{{
						Name: "rules", KeyType: "test_access_block", ValueType: "test_rule",
					}},
				}},
			}
			graph := newPolicyWalkState(false)
			bucketAddr := mustResourceInstanceAddr("test_bucket.logs")
			blockAddr := mustResourceInstanceAddr("test_access_block.logs")
			for _, addr := range []addrs.AbsResourceInstance{bucketAddr, blockAddr} {
				graph.addResource(&policyResource{
					Addr:  addr,
					Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("logs")})),
				})
			}
			deferred := deferring.NewDeferred(true)
			if tc.deferredType != "" {
				addr := mustResourceInstanceAddr(tc.deferredType + ".deferred")
				change := &plans.ResourceInstanceChange{Addr: addr}
				if tc.expansion {
					deferred.ReportResourceExpansionDeferred(addr.ContainingResource().UnexpandedResource(), change)
				} else {
					deferred.ReportResourceInstanceDeferred(addr, providers.DeferredReasonProviderConfigUnknown, change)
				}
			}
			if tc.unknownType != "" {
				addr := mustResourceInstanceAddr(tc.unknownType + ".unknown")
				graph.markUnknownResourceType(addr.Resource.Resource.Type)
			}
			ctx := &MockEvalContext{
				PolicyStateValue:  graph,
				PolicyClientValue: client,
				ConfigValue:       &configs.Config{Module: &configs.Module{}},
				ChangesChanges:    plans.NewChanges().SyncWrapper(),
				StateState:        states.NewState().SyncWrapper(),
				DeferralsState:    deferred,
			}
			diags := (&nodePolicyEval{}).Execute(ctx, walkPlan)
			tfdiags.AssertNoErrors(t, diags)
			if len(graph.typeResourceMap["test_bucket"]) != 1 || graph.typeResourceMap["test_bucket"][0] != graph.resourceMap[bucketAddr.String()] {
				t.Fatal("resource index was not prepared")
			}
			relation := graph.followRelationships(graph.resourceMap[bucketAddr.String()], client.Config().Relationships["test_bucket"])[0]
			if relation.Complete != tc.wantComplete {
				t.Errorf("Complete = %t, want %t", relation.Complete, tc.wantComplete)
			}
			nested := relation.Resources[blockAddr.String()][0]
			if nested.Complete != tc.wantNested {
				t.Errorf("nested Complete = %t, want %t", nested.Complete, tc.wantNested)
			}
		})
	}
}

// TestNodePolicyEval_DynamicExpand_FinishWiring verifies that DynamicExpand
// appends a nodePolicyEvalFinish node to the policy subgraph and wires it to
// depend on every nodeResourcePolicy and nodeQueryResourcePolicy node already
// present in the subgraph.
func TestNodePolicyEval_DynamicExpand_FinishWiring(t *testing.T) {
	rp := &nodeResourcePolicy{
		ResourceAddr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "foo",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
	}
	qrp := &nodeQueryResourcePolicy{
		ResourceAddr: addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "aws_instance",
			Name: "bar",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
	}

	ps := newPolicyWalkState(false)
	ps.addResourceNode(rp)
	ps.addQueryNode(qrp)

	ctx := &MockEvalContext{
		PolicyStateValue: ps,
		ChangesChanges:   plans.NewChanges().SyncWrapper(),
		StateState:       states.NewState().SyncWrapper(),
		StopCtxValue:     context.Background(),
	}

	n := &nodePolicyEval{}
	g, diags := n.DynamicExpand(ctx)
	if diags.HasErrors() {
		t.Fatalf("DynamicExpand returned errors: %s", diags.Err())
	}
	if g == nil {
		t.Fatal("DynamicExpand returned nil graph")
	}

	// nodePolicyEvalFinish must be present in the expanded graph.
	finishNodes := dag.SelectSeq[*nodePolicyEvalFinish](g.VerticesSeq()).Collect()
	if len(finishNodes) != 1 {
		t.Fatalf("expected 1 nodePolicyEvalFinish node, got %d", len(finishNodes))
	}

	// Both policy node types must be scheduled before the finish node.
	testGraphHappensBefore(t, g, rp.Name(), "(policy evaluation complete)")
	testGraphHappensBefore(t, g, qrp.Name(), "(policy evaluation complete)")

}

// TestNodePolicyEval_DynamicExpand_ConcurrentExecution verifies that when
// DynamicExpand returns a graph with multiple nodeQueryResourcePolicy nodes,
// those nodes have no dependency edges between them, allowing the graph walker
// to execute them concurrently.
func TestNodePolicyEval_DynamicExpand_ConcurrentExecution(t *testing.T) {
	const count = 10

	ps := newPolicyWalkState(false)

	// Add multiple nodeQueryResourcePolicy nodes to simulate a list block
	// returning multiple resources
	for i := 0; i < count; i++ {
		qrp := &nodeQueryResourcePolicy{
			ResourceAddr: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "discovered",
			}.Instance(addrs.IntKey(i)).Absolute(addrs.RootModuleInstance),
		}
		ps.addQueryNode(qrp)
	}

	ctx := &MockEvalContext{
		PolicyStateValue: ps,
		ChangesChanges:   plans.NewChanges().SyncWrapper(),
		StateState:       states.NewState().SyncWrapper(),
		StopCtxValue:     context.Background(),
	}

	n := &nodePolicyEval{}
	g, diags := n.DynamicExpand(ctx)
	if diags.HasErrors() {
		t.Fatalf("DynamicExpand returned errors: %s", diags.Err())
	}
	if g == nil {
		t.Fatal("DynamicExpand returned nil graph")
	}

	// Collect all nodeQueryResourcePolicy nodes in the expanded graph
	policyNodes := dag.SelectSeq[*nodeQueryResourcePolicy](g.VerticesSeq()).Collect()
	if len(policyNodes) != count {
		t.Fatalf("expected %d policy nodes in expanded graph, got %d", count, len(policyNodes))
	}

	// The critical requirement for concurrent execution: verify no edges exist
	// between any pair of nodeQueryResourcePolicy nodes. This allows the graph
	// walker to execute all policy nodes in parallel.
	for _, node := range policyNodes {
		// Check upstream edges (what this node depends on)
		for dep := range g.EdgesTo(node).All() {
			if otherPolicy, ok := dep.(*nodeQueryResourcePolicy); ok {
				t.Errorf("policy node %s has upstream dependency on another policy node %s; nodes must be independent for concurrent execution",
					node.Name(), otherPolicy.Name())
			}
		}

		// Check downstream edges (what depends on this node)
		for dep := range g.EdgesFrom(node).All() {
			if otherPolicy, ok := dep.(*nodeQueryResourcePolicy); ok {
				t.Errorf("policy node %s has downstream dependency on another policy node %s; nodes must be independent for concurrent execution",
					node.Name(), otherPolicy.Name())
			}
		}
	}

	// Verify finish node exists (wiring is tested in TestNodePolicyEval_DynamicExpand_FinishWiring)
	finishNodes := dag.SelectSeq[*nodePolicyEvalFinish](g.VerticesSeq()).Collect()
	if len(finishNodes) != 1 {
		t.Errorf("expected 1 finish node in expanded graph, got %d", len(finishNodes))
	}
}

// These reproductions use scalar string relationship keys. The implicit
// transformations happen between those keys, not in the relationship itself.
func TestNodePolicyEval_OriginBoundaries(t *testing.T) {
	variableConfig := `
resource "test_source" "a" {}
module "child" {
  source = "./child"
  id     = test_source.a.id
}
resource "test_target" "b" { id = module.child.id }
`
	objectConfig := `
resource "test_source" "a" {}
module "child" {
  source   = "./child"
  settings = test_source.a
}
resource "test_target" "b" { id = module.child.id }
`
	for name, tc := range map[string]struct {
		main, child     string
		source, target  cty.Value
		providerDefault bool
	}{
		"variable default": {main: variableConfig, child: `
variable "id" {
  type     = string
  nullable = false
  default  = "fallback"
}
output "id" { value = var.id }
`, source: cty.NullVal(cty.String), target: cty.StringVal("fallback")},
		"optional attribute default": {main: objectConfig, child: `
variable "settings" { type = object({ id = optional(string, "fallback") }) }
output "id" { value = var.settings.id }
`, source: cty.NullVal(cty.String), target: cty.StringVal("fallback")},
		"object conversion": {main: objectConfig, child: `
variable "settings" { type = object({ id = number }) }
output "id" { value = var.settings.id }
`, source: cty.StringVal("01"), target: cty.StringVal("1")},
		"resource schema conversion": {main: `
resource "test_source" "a" {}
resource "test_number" "middle" { id = test_source.a.id }
resource "test_target" "b" { id = test_number.middle.id }
`, source: cty.StringVal("01"), target: cty.StringVal("1")},
		"provider default": {main: `
resource "test_source" "a" {}
resource "test_target" "b" { id = test_source.a.id }
`, source: cty.NullVal(cty.String), target: cty.StringVal("fallback"), providerDefault: true},
		"variable conversion": {main: variableConfig, child: `
variable "id" { type = number }
output "id" { value = var.id }
`, source: cty.StringVal("01"), target: cty.StringVal("1")},
	} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"main.tf": tc.main}
			if tc.child != "" {
				files["child/main.tf"] = tc.child
			}
			testPolicyOriginBoundary(t, files, tc.source, tc.target, tc.providerDefault)
		})
	}
}

func TestNodePolicyEval_DirectOriginsPlanAndApply(t *testing.T) {
	config := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_source" "root" { scope = "root" }
locals { root = test_source.root }
resource "test_copy" "direct" { id = local.root.id }
resource "test_copy" "peer" { id = test_source.root.id }
resource "test_copy" "chain" { id = test_copy.direct.id }
module "child" {
  source = "./child"
  id     = test_source.root.id
}
resource "test_copy" "output" { id = module.child.id }
`,
		"child/main.tf": `
variable "id" { type = string }
resource "test_source" "child" { scope = "child" }
locals { id = test_source.child.id }
resource "test_copy" "direct" { id = local.id }
resource "test_copy" "input" { id = var.id }
output "id" { value = test_source.child.id }
`,
	})
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_source": {Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Computed: true}, "scope": {Type: cty.String, Required: true},
			}},
			"test_copy": {Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Required: true},
			}},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		value := req.ProposedNewState
		if req.TypeName == "test_source" {
			value = cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String).RefineNotNull(), "scope": value.GetAttr("scope"),
			})
		}
		return providers.PlanResourceChangeResponse{PlannedState: value}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		value := req.PlannedState
		if req.TypeName == "test_source" {
			value = cty.ObjectVal(map[string]cty.Value{
				"id": value.GetAttr("scope"), "scope": value.GetAttr("scope"),
			})
		}
		return providers.ApplyResourceChangeResponse{NewState: value}
	}
	ctx := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
	}})
	client := policy.NewTestMockClient(t)
	match := policy.Match{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}}
	client.ConfigValue.Relationships = map[string][]policy.Relationship{
		"test_source": {{Name: "copies", KeyType: "test_source", ValueType: "test_copy", Matches: []policy.Match{match}}},
		"test_copy":   {{Name: "peers", KeyType: "test_copy", ValueType: "test_copy", Matches: []policy.Match{match}}},
	}
	requests := make(map[string]policy.EvaluationRequest[*proto.ResourceMetadata])
	client.EvaluateFn = func(_ context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		if _, exists := requests[req.Identifier]; exists {
			t.Errorf("duplicate resource evaluation for %s", req.Identifier)
		}
		requests[req.Identifier] = req
		return policy.EvaluationResponse{}
	}
	assertRelations := func(t *testing.T, want map[string][]string, known bool) {
		t.Helper()
		if len(requests) != 8 {
			t.Errorf("got %d resource evaluations, want 8", len(requests))
		}
		for identifier, related := range want {
			req, ok := requests[identifier]
			if !ok {
				t.Errorf("missing resource evaluation for %s", identifier)
				continue
			}
			id := req.Attrs.Raw.GetAttr("id")
			if id.IsKnown() != known || !id.Range().DefinitelyNotNull() {
				t.Errorf("%s: expected non-null id with known=%t, got %#v", identifier, known, id)
			}
			if req.Meta.Address != identifier || len(req.Relations) != 1 {
				t.Errorf("%s: incorrect metadata or Relations: %#v", identifier, req)
				continue
			}
			relation := req.Relations[0]
			if relation.Complete != known || len(relation.Resources) != len(related) {
				t.Errorf("%s: got %#v, want resources %v and complete=%t", identifier, relation, related, known)
			}
			for _, other := range related {
				if _, ok := relation.Resources[other]; !ok {
					t.Errorf("%s: missing related resource %s", identifier, other)
				}
			}
		}
	}

	// Retain the actual walker to inspect origins, then apply the changes from
	// that same plan walk. This fixture has no root variables or prior objects.
	plan, refs := func() (*plans.Plan, *policyWalkState) {
		defer ctx.acquireRun("plan")()
		inputState := states.NewState()
		tfdiags.AssertNoErrors(t, ctx.checkPolicyRelationships(client, config, inputState, nil))
		graph, op, diags := ctx.planGraph(config, inputState, &PlanOpts{Mode: plans.NormalMode, PolicyClient: client})
		tfdiags.AssertNoErrors(t, diags)
		changes := plans.NewChanges()
		timestamp := time.Now().UTC()
		walker, diags := ctx.walk(graph, op, &graphWalkOpts{
			Config: config, InputState: inputState, Changes: changes, PolicyClient: client, PlanTimeTimestamp: timestamp,
		})
		tfdiags.AssertNoErrors(t, diags)
		tfdiags.AssertNoErrors(t, walker.NonFatalDiagnostics)
		prevRunState := walker.PrevRunState.Close()
		walker.RefreshState.RemovePlannedResourceInstanceObjects()
		priorState := walker.RefreshState.Close()
		schemas, diags := ctx.Schemas(config, prevRunState)
		tfdiags.AssertNoErrors(t, diags)
		changesSrc, err := changes.Encode(schemas)
		if err != nil {
			t.Fatal(err)
		}
		return &plans.Plan{
			UIMode: plans.NormalMode, Changes: changesSrc, PrevRunState: prevRunState, PriorState: priorState,
			Checks: states.NewCheckResults(walker.Checks), Timestamp: timestamp, Complete: true, Applyable: !changesSrc.Empty(),
		}, walker.PolicyState
	}()
	for addr, want := range map[string]string{
		"test_source.root":               "",
		"module.child.test_source.child": "",
		"test_copy.direct":               "test_source.root.id",
		"test_copy.peer":                 "test_source.root.id",
		"test_copy.chain":                "test_copy.direct.id",
		"module.child.test_copy.direct":  "module.child.test_source.child.id",
		"module.child.test_copy.input":   "",
		"test_copy.output":               "",
	} {
		origin, ok := refs.origin(globalref.ResourceAttr{Resource: mustResourceInstanceAddr(addr), Attr: cty.GetAttrPath("id")})
		if want == "" {
			if ok {
				t.Errorf("%s: unexpected origin %s", addr, origin.DebugString())
			}
		} else if !ok || origin.DebugString() != want {
			t.Errorf("%s: origin = %s (recorded=%t), want %s", addr, origin.DebugString(), ok, want)
		}
	}
	t.Run("unknown planned values", func(t *testing.T) {
		assertRelations(t, map[string][]string{
			// A plain local alias preserves a direct source in either module.
			// Resource chains and module inputs/outputs supply no extra proof.
			"test_source.root":               {"test_copy.direct", "test_copy.peer"},
			"module.child.test_source.child": {"module.child.test_copy.direct"},
			// Direct and peer share a third resource's attribute. Chain can
			// match its immediate source, but not peer or the original source.
			"test_copy.direct":              {"test_copy.peer", "test_copy.chain"},
			"test_copy.peer":                {"test_copy.direct"},
			"test_copy.chain":               {"test_copy.direct"},
			"test_copy.output":              nil,
			"module.child.test_copy.direct": nil,
			"module.child.test_copy.input":  nil,
		}, false)
	})

	requests = make(map[string]policy.EvaluationRequest[*proto.ResourceMetadata])
	_, diags := ctx.Apply(plan, config, &ApplyOpts{PolicyClient: client})
	tfdiags.AssertNoErrors(t, diags)
	t.Run("known applied values", func(t *testing.T) {
		// Representative subjects cover known equality across resource chains
		// and module inputs/outputs; symbolic origin restrictions no longer apply.
		assertRelations(t, map[string][]string{
			"test_source.root":               {"test_copy.direct", "test_copy.peer", "test_copy.chain", "module.child.test_copy.input"},
			"module.child.test_source.child": {"module.child.test_copy.direct", "test_copy.output"},
			"test_copy.direct":               {"test_copy.peer", "test_copy.chain", "module.child.test_copy.input"},
			"test_copy.output":               {"module.child.test_copy.direct"},
		}, true)
	})
}

func TestNodePolicyEval_OriginDirectCopyNullability(t *testing.T) {
	unknown := cty.UnknownVal(cty.String)
	nonnull := unknown.RefineNotNull()
	for name, tc := range map[string]struct {
		source, target cty.Value
		writeOnly      bool
		wantRelated    bool
		wantComplete   bool
	}{
		"both nullable":                    {source: unknown, target: unknown},
		"one hop conversion with nonnulls": {source: nonnull, target: cty.UnknownVal(cty.Number).RefineNotNull()},
		"write only source is known null":  {source: cty.NullVal(cty.String), target: nonnull, writeOnly: true, wantComplete: true},
	} {
		t.Run(name, func(t *testing.T) {
			config := testModuleInline(t, map[string]string{"main.tf": `
resource "test_source" "a" {}
resource "test_target" "b" { id = test_source.a.id }
`})
			client := policy.NewTestMockClient(t)
			client.ConfigValue.Relationships = map[string][]policy.Relationship{"test_source": {{
				Name: "same_id", KeyType: "test_source", ValueType: "test_target",
				Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}}},
			}}}
			refs := newPolicyWalkState(true)
			graph := refs
			sourceAddr := mustResourceInstanceAddr("test_source.a")
			targetAddr := mustResourceInstanceAddr("test_target.b")
			for _, resource := range []struct {
				addr  addrs.AbsResourceInstance
				value cty.Value
			}{{sourceAddr, tc.source}, {targetAddr, tc.target}} {
				graph.addResource(&policyResource{
					Addr:  resource.addr,
					Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"id": resource.value})),
				})
			}
			ctx := &MockEvalContext{
				PolicyStateValue: graph, PolicyClientValue: client, ConfigValue: config,
				ProviderSchemaSchema: providers.ProviderSchema{ResourceTypes: map[string]providers.Schema{
					"test_source": {Body: &configschema.Block{Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Computed: !tc.writeOnly, Optional: tc.writeOnly, WriteOnly: tc.writeOnly},
					}}},
				}},
				ChangesChanges: plans.NewChanges().SyncWrapper(), StateState: states.NewState().SyncWrapper(),
				DeferralsState: deferring.NewDeferred(false),
			}
			recordPolicyOrigins(ctx, targetAddr, config.Module.ResourceByAddr(targetAddr.Resource.Resource),
				&configschema.Block{Attributes: map[string]*configschema.Attribute{"id": {Type: tc.target.Type(), Required: true}}}, []cty.Path{ctyutil.NormalizePath(cty.GetAttrPath("id"))})
			origin, ok := refs.origin(globalref.ResourceAttr{Resource: targetAddr, Attr: cty.GetAttrPath("id")})
			if tc.writeOnly || !tc.source.Type().Equals(tc.target.Type()) {
				if ok {
					t.Error("write-only sources and conversions must not establish an exact copy")
				}
			} else if !ok || !origin.Resource.Equal(sourceAddr) || !origin.Attr.Equals(cty.GetAttrPath("id")) {
				t.Fatal("direct edge must be retained regardless of the endpoints' nullable values")
			}
			tfdiags.AssertNoErrors(t, (&nodePolicyEval{}).Execute(ctx, walkPlan))
			relation := graph.followRelationships(graph.resourceMap[sourceAddr.String()], client.Config().Relationships["test_source"])[0]
			_, related := relation.Resources[targetAddr.String()]
			if related != tc.wantRelated || relation.Complete != tc.wantComplete {
				t.Errorf("related=%t complete=%t, want related=%t complete=%t", related, relation.Complete, tc.wantRelated, tc.wantComplete)
			}
		})
	}
}

func TestNodePolicyEval_OriginExistingRequiredAttribute(t *testing.T) {
	for name, tc := range map[string]struct {
		forceReplace bool
		update       bool
		wantAction   plans.Action
	}{
		"retained prior":         {wantAction: plans.NoOp},
		"apply replan retention": {update: true, wantAction: plans.Update},
		"replacement":            {forceReplace: true, wantAction: plans.DeleteThenCreate},
	} {
		t.Run(name, func(t *testing.T) {
			config := testModuleInline(t, map[string]string{"main.tf": `
resource "test_source" "a" {}
resource "test_target" "b" {
  id       = test_source.a.id
  revision = "new"
}
`})
			sourceResult, priorID := cty.StringVal("01"), cty.StringVal("1")
			priorJSON := []byte(`{"id":"1","revision":"new"}`)
			if tc.update {
				sourceResult = cty.StringVal(`{"a":1}`)
				priorID = cty.StringVal(`{ "a": 1 }`)
				priorJSON = []byte(`{"id":"{ \"a\": 1 }","revision":"old"}`)
			}
			p := new(testing_provider.MockProvider)
			p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
				ResourceTypes: map[string]*configschema.Block{
					"test_source": {Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Computed: true}}},
					"test_target": {Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Required: true}, "revision": {Type: cty.String, Required: true},
					}},
				},
			})
			p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				if req.TypeName == "test_source" {
					return providers.PlanResourceChangeResponse{PlannedState: cty.ObjectVal(map[string]cty.Value{
						"id": cty.UnknownVal(cty.String).RefineNotNull(),
					})}
				}
				id := req.ProposedNewState.GetAttr("id")
				if !req.PriorState.IsNull() && (!tc.update || req.Config.GetAttr("id").IsKnown()) {
					// Even Required can retain a semantically equivalent prior.
					// The update case waits until apply-time replanning to do so.
					id = req.PriorState.GetAttr("id")
				}
				return providers.PlanResourceChangeResponse{PlannedState: cty.ObjectVal(map[string]cty.Value{
					"id": id, "revision": req.ProposedNewState.GetAttr("revision"),
				})}
			}
			p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
				value := req.PlannedState
				if req.TypeName == "test_source" {
					value = cty.ObjectVal(map[string]cty.Value{"id": sourceResult})
				}
				return providers.ApplyResourceChangeResponse{NewState: value}
			}
			ctx := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
			}})
			targetAddr := mustResourceInstanceAddr("test_target.b")
			state := states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(targetAddr, &states.ResourceInstanceObjectSrc{
					Status: states.ObjectReady, AttrsJSON: priorJSON,
				}, addrs.AbsProviderConfig{Provider: addrs.NewDefaultProvider("test")})
			})
			client := policy.NewTestMockClient(t)
			client.ConfigValue.Relationships = map[string][]policy.Relationship{"test_source": {{
				Name: "same_id", KeyType: "test_source", ValueType: "test_target",
				Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}}},
			}}}
			var relation policy.Relation
			var sourceID, targetID cty.Value
			var targetOperation proto.Operation
			client.EvaluateFn = func(_ context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
				// Replacement also evaluates the delete, which has no After attrs.
				if req.Attrs.Raw == cty.NilVal || req.Attrs.Raw.IsNull() {
					return policy.EvaluationResponse{}
				}
				if req.Identifier == "test_source.a" {
					sourceID = req.Attrs.Raw.GetAttr("id")
					if len(req.Relations) != 1 {
						t.Errorf("expected one Relation, got %d", len(req.Relations))
					} else {
						relation = req.Relations[0]
					}
				} else if req.Identifier == "test_target.b" {
					targetID = req.Attrs.Raw.GetAttr("id")
					targetOperation = req.Meta.Operation
				}
				return policy.EvaluationResponse{}
			}
			opts := &PlanOpts{Mode: plans.NormalMode, PolicyClient: client}
			if tc.forceReplace {
				opts.ForceReplace = []addrs.AbsResourceInstance{targetAddr}
			}
			plan, diags := ctx.Plan(config, state, opts)
			tfdiags.AssertNoErrors(t, diags)
			if change := plan.Changes.ResourceInstance(targetAddr); change == nil || change.Action != tc.wantAction {
				t.Fatalf("target change = %#v, want %s", change, tc.wantAction)
			}
			if sourceID == cty.NilVal || sourceID.IsKnown() || !sourceID.Range().DefinitelyNotNull() || targetID == cty.NilVal {
				t.Fatalf("unexpected plan keys: source=%#v target=%#v", sourceID, targetID)
			}
			if tc.wantAction != plans.NoOp {
				if targetID.IsKnown() {
					t.Fatalf("changing target must initially have an unknown id: %#v", targetID)
				}
			} else if !targetID.RawEquals(cty.StringVal("1")) {
				t.Fatalf("target did not retain its prior representation: %#v", targetID)
			}
			if _, related := relation.Resources[targetAddr.String()]; related || relation.Complete {
				t.Error("an existing Required attribute is not a copy proof: its prior representation can be retained")
			}
			if tc.wantAction != plans.NoOp && tc.wantAction != plans.Update {
				return
			}
			_, diags = ctx.Apply(plan, config, &ApplyOpts{PolicyClient: client})
			tfdiags.AssertNoErrors(t, diags)
			if !sourceID.RawEquals(sourceResult) || !targetID.RawEquals(priorID) {
				t.Fatalf("unexpected applied keys: source=%#v target=%#v", sourceID, targetID)
			}
			if tc.update && targetOperation != proto.Operation_UPDATE {
				t.Fatalf("revision change must keep UPDATE during apply, got %s", targetOperation)
			}
			if _, related := relation.Resources[targetAddr.String()]; related || !relation.Complete {
				t.Error("known distinct string representations must be unrelated and complete after apply")
			}
		})
	}
}

// testPolicyOriginBoundary runs a fresh plan and apply, exposing both sets of
// values and Relations for debugging. Transformations must not establish
// relationships before their resulting keys are known.
func testPolicyOriginBoundary(t *testing.T, files map[string]string, sourceID, targetID cty.Value, providerDefault bool) {
	t.Helper()
	config := testModuleInline(t, files)
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_source": {Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Computed: true},
			}},
			"test_number": {Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.Number, Required: true},
			}},
			"test_target": {Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Required: !providerDefault, Optional: providerDefault, Computed: providerDefault},
			}},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		value := req.ProposedNewState
		if req.TypeName == "test_source" {
			value = cty.ObjectVal(map[string]cty.Value{"id": cty.UnknownVal(cty.String)})
		}
		if req.TypeName == "test_target" {
			id := value.GetAttr("id")
			if !id.IsKnown() {
				// Isolate each transformation from the matcher's nullability
				// guard: every fixture eventually supplies a non-null target.
				id = id.RefineNotNull()
			}
			if providerDefault && req.Config.GetAttr("id").IsNull() {
				// Default only a known null, never an unknown configured value.
				id = cty.StringVal("fallback")
			}
			value = cty.ObjectVal(map[string]cty.Value{"id": id})
		}
		return providers.PlanResourceChangeResponse{PlannedState: value}
	}
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		value := req.PlannedState
		if req.TypeName == "test_source" {
			value = cty.ObjectVal(map[string]cty.Value{"id": sourceID})
		}
		return providers.ApplyResourceChangeResponse{NewState: value}
	}
	ctx := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
	}})
	client := policy.NewTestMockClient(t)
	client.ConfigValue.Relationships = map[string][]policy.Relationship{
		"test_source": {
			{
				Name: "same_id", KeyType: "test_source", ValueType: "test_target",
				Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}}},
			},
		},
	}
	var planned, applied struct {
		sourceID, targetID cty.Value
		relation           policy.Relation
	}
	current := &planned
	client.EvaluateFn = func(_ context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		switch req.Identifier {
		case "test_source.a":
			current.sourceID = req.Attrs.Raw.GetAttr("id")
			if len(req.Relations) != 1 {
				t.Errorf("expected one Relation, got %d", len(req.Relations))
				return policy.EvaluationResponse{}
			}
			current.relation = req.Relations[0]
		case "test_target.b":
			current.targetID = req.Attrs.Raw.GetAttr("id")
		}
		return policy.EvaluationResponse{}
	}

	plan, diags := ctx.Plan(config, states.NewState(), &PlanOpts{Mode: plans.NormalMode, PolicyClient: client})
	tfdiags.AssertNoErrors(t, diags)
	_, planRelated := planned.relation.Resources["test_target.b"]
	t.Logf("plan: source=%#v target=%#v related=%t complete=%t", planned.sourceID, planned.targetID, planRelated, planned.relation.Complete)
	if planned.sourceID == cty.NilVal || planned.targetID == cty.NilVal {
		t.Fatal("missing plan evaluation")
	}
	if planned.sourceID.IsKnown() || planned.targetID.IsKnown() {
		t.Fatal("both relationship keys must be unknown during planning for this reproduction")
	}
	if !planned.targetID.Range().DefinitelyNotNull() {
		t.Fatal("transformations must be rejected independently of the non-null guard")
	}

	current = &applied
	_, diags = ctx.Apply(plan, config, &ApplyOpts{PolicyClient: client})
	tfdiags.AssertNoErrors(t, diags)
	_, applyRelated := applied.relation.Resources["test_target.b"]
	t.Logf("apply: source=%#v target=%#v related=%t complete=%t", applied.sourceID, applied.targetID, applyRelated, applied.relation.Complete)
	if !applied.sourceID.RawEquals(sourceID) || !applied.targetID.RawEquals(targetID) {
		t.Errorf("wrong applied values: want source=%#v target=%#v", sourceID, targetID)
	}
	if applyRelated || !applied.relation.Complete {
		t.Error("known unequal/null keys must give an unrelated, complete result after apply")
	}
	if planRelated || planned.relation.Complete {
		t.Error("plan must leave this relationship undecided: the plain-reference chain does not preserve equality")
	}
}
