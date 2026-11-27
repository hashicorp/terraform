// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestKeyEqual(t *testing.T) {
	unknown := cty.UnknownVal(cty.String)
	nonNull := unknown.RefineNotNull()
	undecided := cty.UnknownVal(cty.Bool)
	aAttr := globalref.ResourceAttr{Resource: mustResourceInstanceAddr("test_bucket.source"), Attr: cty.GetAttrPath("id")}
	bAttr := globalref.ResourceAttr{Resource: mustResourceInstanceAddr("test_access_block.subject"), Attr: cty.GetAttrPath("bucket")}
	key := policy.RelationshipKey{KeyPath: aAttr.Attr, ValuePath: bAttr.Attr}

	for name, tc := range map[string]struct {
		a, b   cty.Value
		shared bool
		want   cty.Value
	}{
		"equal known values":                   {a: cty.StringVal("logs"), b: cty.StringVal("logs"), want: cty.True},
		"known inequality overrides origins":   {a: cty.StringVal("logs"), b: cty.StringVal("other"), shared: true, want: cty.False},
		"refined inequality overrides origins": {a: unknown.Refine().StringPrefix("arn:").NewValue(), b: cty.StringVal("logs"), shared: true, want: cty.False},
		"missing path":                         {a: nonNull, b: cty.NilVal, shared: true, want: cty.False},
		"two nulls":                            {a: cty.NullVal(cty.String), b: cty.NullVal(cty.String), shared: true, want: cty.False},
		"null and unknown":                     {a: cty.NullVal(cty.String), b: unknown, shared: true, want: cty.False},
		"shared nullable unknowns":             {a: unknown, b: unknown, shared: true, want: undecided},
		"nonnull source":                       {a: nonNull, b: unknown, shared: true, want: cty.True},
		"nonnull destination":                  {a: unknown, b: nonNull, shared: true, want: cty.True},
		"marks alone are not origins":          {a: nonNull.Mark("shared"), b: nonNull.Mark("shared"), want: undecided},
		"known source and unknown copy":        {a: cty.StringVal("logs"), b: unknown, shared: true, want: cty.True},
		"marked known values":                  {a: cty.StringVal("logs").Mark(marks.Sensitive), b: cty.StringVal("logs").Mark(marks.Sensitive), want: cty.True},
		"marked symbolic proof":                {a: nonNull.Mark(marks.Sensitive), b: unknown.Mark(marks.Sensitive), shared: true, want: cty.True},
	} {
		t.Run(name, func(t *testing.T) {
			graph := newPolicyWalkState(true)
			if tc.shared {
				graph.recordOrigin(bAttr, aAttr)
			}
			a := &policyResource{Addr: aAttr.Resource, Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"id": tc.a}))}
			b := &policyResource{Addr: bAttr.Resource, Value: policy.CtyToPolicyValue(cty.EmptyObjectVal)}
			if tc.b != cty.NilVal {
				b.Value = policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"bucket": tc.b}))
			}
			beforeA, beforeB := a.Value.Raw, b.Value.Raw
			if got := graph.checkEquality(a, b, key); !got.RawEquals(tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
			if name == "marked symbolic proof" && (!a.Value.Raw.RawEquals(beforeA) || !b.Value.Raw.RawEquals(beforeB)) {
				t.Error("comparison changed the marked resource snapshots")
			}
		})
	}

	// An unknown descendant can be selected through a marked ancestor, but
	// symbolic proof is specific to the exact path and requires a graph.
	t.Run("descendant identity", func(t *testing.T) {
		settingsType := cty.Object(map[string]cty.Type{"id": cty.String, "other": cty.String})
		a := &policyResource{Addr: aAttr.Resource, Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{
			"settings": cty.UnknownVal(settingsType).Mark(marks.Sensitive),
		}))}
		b := &policyResource{Addr: bAttr.Resource, Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"bucket": nonNull}))}
		idPath := cty.GetAttrPath("settings").GetAttr("id")
		graph := newPolicyWalkState(true)
		graph.recordOrigin(bAttr, globalref.ResourceAttr{Resource: a.Addr, Attr: idPath})
		for name, tc := range map[string]struct {
			path  cty.Path
			graph *policyWalkState
			want  cty.Value
		}{
			"exact":     {path: idPath, graph: graph, want: cty.True},
			"sibling":   {path: cty.GetAttrPath("settings").GetAttr("other"), graph: graph, want: undecided},
			"nil graph": {path: idPath, want: undecided},
		} {
			t.Run(name, func(t *testing.T) {
				got := tc.graph.checkEquality(a, b, policy.RelationshipKey{KeyPath: tc.path, ValuePath: bAttr.Attr})
				if !got.RawEquals(tc.want) {
					t.Errorf("got %#v, want %#v", got, tc.want)
				}
			})
		}
	})
}

func TestSameTypeRelationshipMatching(t *testing.T) {
	known := cty.StringVal
	unknown := cty.UnknownVal(cty.String)
	idKey := policy.RelationshipKey{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("parent_id")}
	zoneKey := policy.RelationshipKey{KeyPath: cty.GetAttrPath("zone"), ValuePath: cty.GetAttrPath("parent_zone")}
	value := func(id, parent, zone, parentZone cty.Value) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{"id": id, "parent_id": parent, "zone": zone, "parent_zone": parentZone})
	}
	for name, tc := range map[string]struct {
		a, b     cty.Value
		matches  []policy.Match
		symbolic bool
		want     cty.Value
	}{
		"forward":           {a: value(known("a"), known("root"), known("east"), known("north")), b: value(known("b"), known("a"), known("west"), known("east")), want: cty.True},
		"reverse":           {a: value(known("b"), known("a"), known("west"), known("east")), b: value(known("a"), known("root"), known("east"), known("north")), want: cty.True},
		"neither":           {a: value(known("a"), known("root"), known("east"), known("north")), b: value(known("b"), known("root"), known("west"), known("north")), want: cty.False},
		"unknown":           {a: value(unknown, known("root"), known("east"), known("north")), b: value(known("b"), known("a"), known("west"), known("east")), want: cty.UnknownVal(cty.Bool)},
		"true over unknown": {a: value(known("a"), unknown, known("east"), known("west")), b: value(known("b"), known("a"), known("west"), known("east")), want: cty.True},
		// The id key matches forward and zone matches in reverse; neither
		// orientation satisfies the whole AND match.
		"mixed orientation AND": {a: value(known("a"), known("root"), known("east"), known("west")), b: value(known("b"), known("a"), known("west"), known("north")), want: cty.False},
		"OR alternative":        {a: value(known("a"), known("root"), known("east"), known("west")), b: value(known("b"), known("a"), known("west"), known("north")), matches: []policy.Match{{idKey, zoneKey}, {zoneKey}}, want: cty.True},
		"symbolic reverse":      {a: value(unknown.RefineNotNull(), unknown, known("west"), known("east")), b: value(unknown.RefineNotNull(), unknown, known("east"), known("west")), symbolic: true, want: cty.True},
	} {
		t.Run(name, func(t *testing.T) {
			a := &policyResource{Addr: mustResourceInstanceAddr("test_node.a"), Value: policy.CtyToPolicyValue(tc.a)}
			b := &policyResource{Addr: mustResourceInstanceAddr("test_node.b"), Value: policy.CtyToPolicyValue(tc.b)}
			matches := tc.matches
			if matches == nil {
				matches = []policy.Match{{idKey, zoneKey}}
			}
			rel := policy.Relationship{Name: "family", KeyType: "test_node", ValueType: "test_node", Matches: matches}
			graph := newPolicyWalkState(true)
			if tc.symbolic {
				graph.recordOrigin(globalref.ResourceAttr{Resource: a.Addr, Attr: idKey.ValuePath}, globalref.ResourceAttr{Resource: b.Addr, Attr: idKey.KeyPath})
				if got := newPolicyWalkState(false).resourcesRelated(a, b, rel); got.IsKnown() {
					t.Errorf("without origins: got %#v, want unknown", got)
				}
			}
			if got := graph.resourcesRelated(a, b, rel); got.IsKnown() != tc.want.IsKnown() || (got.IsKnown() && !got.RawEquals(tc.want)) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestSameTypeRelationshipTree(t *testing.T) {
	rel := policy.Relationship{Name: "family", KeyType: "test_node", ValueType: "test_node",
		Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("parent_id")}}},
	}
	for name, id := range map[string]cty.Value{"known self": cty.StringVal("self"), "nullable self": cty.UnknownVal(cty.String)} {
		t.Run(name, func(t *testing.T) {
			resource := &policyResource{Addr: mustResourceInstanceAddr("test_node.self"), Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"id": id, "parent_id": id}))}
			refs := newPolicyWalkState(true)
			refs.recordOrigin(globalref.ResourceAttr{Resource: resource.Addr, Attr: cty.GetAttrPath("parent_id")}, globalref.ResourceAttr{Resource: resource.Addr, Attr: cty.GetAttrPath("id")})
			graph := refs
			graph.addResource(resource)
			relations := graph.followRelationships(resource, []policy.Relationship{rel})
			if len(relations) != 1 || !relations[0].Complete || len(relations[0].Resources) != 0 {
				t.Fatalf("self must neither match nor make the Relation incomplete: %#v", relations)
			}
		})
	}

	a := &policyResource{Addr: mustResourceInstanceAddr("test_node.a"), Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("a"), "parent_id": cty.StringVal("b")}))}
	b := &policyResource{Addr: mustResourceInstanceAddr("test_node.b"), Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("b"), "parent_id": cty.StringVal("a")}))}
	graph := newPolicyWalkState(false)
	graph.addResource(a)
	graph.addResource(b)
	nested := rel
	nested.Relationships = []policy.Relationship{rel}
	relation := graph.followRelationships(a, []policy.Relationship{nested})[0]
	if !relation.Complete || len(relation.Resources) != 1 {
		t.Fatalf("expected one related resource and a complete Relation: %#v", relation)
	}
	child := relation.Resources[b.Addr.String()][0]
	if !child.Complete || len(child.Resources) != 1 {
		t.Fatalf("expected one nested resource and a complete Relation: %#v", child)
	}
	if leaf, ok := child.Resources[a.Addr.String()]; !ok || len(leaf) != 0 {
		t.Error("expected subject to reappear only at the leaf")
	}
	// Results belong to this evaluation, not the shared index.
	delete(relation.Resources, b.Addr.String())
	if again := graph.followRelationships(a, []policy.Relationship{nested}); len(again[0].Resources) != 1 {
		t.Error("changing one result affected another evaluation")
	}
}

func TestRelateResources(t *testing.T) {
	m := testModuleInline(t, map[string]string{"main.tf": `
resource "test_bucket" "logs" { name = "logs" }
resource "test_bucket" "other" { name = "other" }
resource "test_access_block" "logs" { bucket = test_bucket.logs.id }
resource "test_access_block" "literal" { bucket = "logs" }
resource "test_access_block" "by_name" {
  bucket      = null
  bucket_name = "other"
}
resource "test_rule" "logs" { bucket = test_access_block.logs.bucket }
`})
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_bucket": {Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Computed: true}, "name": {Type: cty.String, Optional: true},
			}},
			"test_access_block": {Attributes: map[string]*configschema.Attribute{
				"bucket": {Type: cty.String, Required: true}, "bucket_name": {Type: cty.String, Optional: true},
			}},
			"test_rule": {Attributes: map[string]*configschema.Attribute{"bucket": {Type: cty.String, Required: true}}},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		value := req.ProposedNewState
		if req.TypeName == "test_bucket" {
			value = cty.ObjectVal(map[string]cty.Value{"id": cty.UnknownVal(cty.String).RefineNotNull(), "name": value.GetAttr("name")})
		}
		return providers.PlanResourceChangeResponse{PlannedState: value}
	}
	ctx := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
	}})
	// Access blocks relate by id OR name, and nest rules matched by bucket.
	accessBlocks := policy.Relationship{Name: "access_blocks", KeyType: "test_bucket", ValueType: "test_access_block",
		Matches: []policy.Match{
			{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("bucket")}},
			{{KeyPath: cty.GetAttrPath("name"), ValuePath: cty.GetAttrPath("bucket_name")}},
		},
	}
	rules := policy.Relationship{Name: "rules", KeyType: "test_access_block", ValueType: "test_rule",
		Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("bucket"), ValuePath: cty.GetAttrPath("bucket")}}},
	}
	withRules := accessBlocks
	withRules.Relationships = []policy.Relationship{rules}
	client := policy.NewTestMockClient(t)
	client.ConfigValue.Relationships = map[string][]policy.Relationship{"test_bucket": {withRules}, "test_access_block": {accessBlocks}}
	requests := make(map[string][]policy.Relation)
	client.EvaluateFn = func(_ context.Context, req policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		requests[req.Identifier] = req.Relations
		return policy.EvaluationResponse{}
	}
	defer ctx.acquireRun("plan")()
	graph, walkOp, diags := ctx.planGraph(m, states.NewState(), &PlanOpts{Mode: plans.NormalMode, PolicyClient: client})
	tfdiags.AssertNoErrors(t, diags)
	_, diags = ctx.walk(graph, walkOp, &graphWalkOpts{
		Config: m, InputState: states.NewState(), Changes: plans.NewChanges(), PolicyClient: client,
	})
	tfdiags.AssertNoErrors(t, diags)

	want := map[string][]policy.Relation{
		// The direct copy matches symbolically; the literal candidate's id
		// comparison is unknown, so this Relation is incomplete.
		"test_bucket.logs": {{Relationship: "access_blocks", Resources: map[string][]policy.Relation{
			"test_access_block.logs": {{Relationship: "rules", Resources: map[string][]policy.Relation{"test_rule.logs": nil}, Complete: true}},
		}}},
		// The name alternative matches despite the unknown id alternative.
		"test_bucket.other": {{Relationship: "access_blocks", Resources: map[string][]policy.Relation{
			"test_access_block.by_name": {{Relationship: "rules", Resources: map[string][]policy.Relation{}, Complete: true}},
		}}},
		"test_access_block.logs":    {{Relationship: "access_blocks", Resources: map[string][]policy.Relation{"test_bucket.logs": nil}}},
		"test_access_block.literal": {{Relationship: "access_blocks", Resources: map[string][]policy.Relation{}}},
		"test_access_block.by_name": {{Relationship: "access_blocks", Resources: map[string][]policy.Relation{"test_bucket.other": nil}, Complete: true}},
		"test_rule.logs":            nil,
	}
	if diff := cmp.Diff(want, requests); diff != "" {
		t.Errorf("wrong pushed relations (-want +got):\n%s", diff)
	}
}
