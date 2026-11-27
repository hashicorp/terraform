// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/configs"
	"os"
	"path/filepath"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/ctyutil"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestRecordResourceReferencesComparisonPaths(t *testing.T) {
	settingsID := cty.GetAttrPath("settings").GetAttr("id")
	deep := cty.GetAttrPath("settings").GetAttr("a").GetAttr("b")
	dotted := cty.GetAttrPath("settings").IndexString("a.b")
	blockID := cty.GetAttrPath("rule").IndexInt(0).GetAttr("id")
	for name, tc := range map[string]struct {
		path        cty.Path
		wholeObject bool
	}{
		"nil selection":                 {},
		"exact leaf":                    {path: settingsID},
		"marked leaf":                   {path: cty.GetAttrPath("settings").Index(cty.StringVal("id").Mark("selection"))},
		"deep leaf":                     {path: deep},
		"literal dotted key":            {path: dotted},
		"static block":                  {path: blockID},
		"whole object is not projected": {path: settingsID, wholeObject: true},
	} {
		t.Run(name, func(t *testing.T) {
			assignment := `settings = { id = test_source.a.identity, sibling = test_source.a.identity, "a.b" = test_source.a.identity, a = { b = test_source.a.identity } }`
			if tc.wholeObject {
				assignment = "settings = test_source.a.settings"
			}
			config := testModuleInline(t, map[string]string{"main.tf": `
resource "test_source" "a" {}
resource "test_target" "subject" {
  ` + assignment + `
  rule { id = test_source.a.identity }
}`})
			schema := &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"settings": {Optional: true, Type: cty.Object(map[string]cty.Type{
						"id": cty.String, "sibling": cty.String, "a.b": cty.String,
						"a": cty.Object(map[string]cty.Type{"b": cty.String}),
					})},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"rule": {Nesting: configschema.NestingList, Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Optional: true}},
					}},
				},
			}
			addr := mustResourceInstanceAddr("test_target.subject")
			resource := config.Module.ResourceByAddr(addr.Resource.Resource)
			refs := newPolicyWalkState(true)
			ctx := &MockEvalContext{ConfigValue: config, PolicyStateValue: refs}
			var paths []cty.Path
			if tc.path != nil {
				paths = []cty.Path{ctyutil.NormalizePath(tc.path)}
			}
			recordPolicyOrigins(ctx, addr, resource, schema, paths)
			path := tc.path
			if path == nil {
				path = settingsID
			}
			origin, ok := refs.origin(globalref.ResourceAttr{Resource: addr, Attr: path})
			want := tc.path != nil && !tc.wholeObject
			if ok != want {
				t.Errorf("origin recorded = %t, want %t", ok, want)
			}
			if ok && origin.DebugString() != "test_source.a.identity" {
				t.Errorf("unexpected origin %s", origin.DebugString())
			}
			if name == "exact leaf" || name == "nil selection" {
				for _, unselected := range []cty.Path{cty.GetAttrPath("settings"), cty.GetAttrPath("settings").GetAttr("sibling"), deep, dotted, blockID} {
					if _, ok := refs.origin(globalref.ResourceAttr{Resource: addr, Attr: unselected}); ok {
						t.Errorf("unselected path %s has an origin", format.CtyPath(unselected))
					}
				}
			}
			if name == "exact leaf" {
				source := &policyResource{Addr: mustResourceInstanceAddr("test_source.a"), Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{
					"identity": cty.UnknownVal(cty.String).RefineNotNull(),
				}))}
				for _, value := range []cty.Value{
					cty.EmptyObjectVal,
					cty.ObjectVal(map[string]cty.Value{"settings": cty.ObjectVal(map[string]cty.Value{"id": cty.NullVal(cty.String)})}),
				} {
					target := &policyResource{Addr: addr, Value: policy.CtyToPolicyValue(value)}
					if got := ctx.PolicyState().checkEquality(source, target, policy.RelationshipKey{KeyPath: cty.GetAttrPath("identity"), ValuePath: settingsID}); !got.RawEquals(cty.False) {
						t.Errorf("origin rescued an absent or null runtime value: %#v", got)
					}
				}
			}
			if ctx.EvaluateExprCalled || ctx.EvaluateBlockCalled {
				t.Error("recording must not reevaluate configuration")
			}
		})
	}
}

func TestRecordResourceReferencesSchemaSafety(t *testing.T) {
	for name, tc := range map[string]struct {
		attribute  *configschema.Attribute
		sourceType cty.Type
		expression string
		path       cty.Path
		sourcePath cty.Path
		want       bool
	}{
		"required string": {
			attribute: &configschema.Attribute{Type: cty.String, Required: true}, want: true,
		},
		"required number": {
			attribute: &configschema.Attribute{Type: cty.Number, Required: true}, sourceType: cty.Number, want: true,
		},
		"required bool": {
			attribute: &configschema.Attribute{Type: cty.Bool, Required: true}, sourceType: cty.Bool, want: true,
		},
		"direct string to number conversion": {
			attribute: &configschema.Attribute{Type: cty.Number, Required: true},
		},
		"direct number to string conversion": {
			attribute: &configschema.Attribute{Type: cty.String, Required: true}, sourceType: cty.Number,
		},
		"optional noncomputed string": {
			attribute: &configschema.Attribute{Type: cty.String, Optional: true}, want: true,
		},
		"optional computed can default": {
			attribute: &configschema.Attribute{Type: cty.String, Optional: true, Computed: true},
		},
		"write only is not persisted": {
			attribute: &configschema.Attribute{Type: cty.String, Required: true, WriteOnly: true},
		},
		"required object member": {
			attribute:  &configschema.Attribute{Type: cty.Object(map[string]cty.Type{"id": cty.String}), Required: true},
			expression: "{ id = test_source.a.id }", path: cty.GetAttrPath("value").GetAttr("id"), want: true,
		},
		"computed object ancestor": {
			attribute:  &configschema.Attribute{Type: cty.Object(map[string]cty.Type{"id": cty.String}), Optional: true, Computed: true},
			expression: "{ id = test_source.a.id }", path: cty.GetAttrPath("value").GetAttr("id"),
		},
		"required nested attribute with computed sibling": {
			attribute: &configschema.Attribute{Required: true, NestedType: &configschema.Object{
				Nesting: configschema.NestingSingle,
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Required: true}, "other": {Type: cty.String, Computed: true},
				},
			}},
			expression: "{ id = test_source.a.id }", path: cty.GetAttrPath("value").GetAttr("id"), want: true,
		},
		"computed nested attribute ancestor": {
			attribute: &configschema.Attribute{Optional: true, Computed: true, NestedType: &configschema.Object{
				Nesting:    configschema.NestingSingle,
				Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Required: true}},
			}},
			expression: "{ id = test_source.a.id }", path: cty.GetAttrPath("value").GetAttr("id"),
		},
		"write only nested attribute ancestor": {
			attribute: &configschema.Attribute{Required: true, WriteOnly: true, NestedType: &configschema.Object{
				Nesting:    configschema.NestingSingle,
				Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Required: true}},
			}},
			expression: "{ id = test_source.a.id }", path: cty.GetAttrPath("value").GetAttr("id"),
		},
		"computed nested leaf": {
			attribute: &configschema.Attribute{Required: true, NestedType: &configschema.Object{
				Nesting:    configschema.NestingSingle,
				Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Optional: true, Computed: true}},
			}},
			expression: "{ id = test_source.a.id }", path: cty.GetAttrPath("value").GetAttr("id"),
		},
		"static source list element": {
			attribute: &configschema.Attribute{Type: cty.String, Required: true}, sourceType: cty.List(cty.String),
			expression: "test_source.a.id[0]", sourcePath: cty.GetAttrPath("id").IndexInt(0), want: true,
		},
		"static source map element": {
			attribute: &configschema.Attribute{Type: cty.String, Required: true}, sourceType: cty.Map(cty.String),
			expression: `test_source.a.id["key"]`, sourcePath: cty.GetAttrPath("id").IndexString("key"), want: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.sourceType == cty.NilType {
				tc.sourceType = cty.String
			}
			if tc.expression == "" {
				tc.expression = "test_source.a.id"
			}
			if tc.path == nil {
				tc.path = cty.GetAttrPath("value")
			}
			if tc.sourcePath == nil {
				tc.sourcePath = cty.GetAttrPath("id")
			}
			config := testModuleInline(t, map[string]string{"main.tf": fmt.Sprintf(`
resource "test_source" "a" {}
resource "test_target" "b" { value = %s }
`, tc.expression)})
			graph := newPolicyWalkState(true)
			ctx := &MockEvalContext{
				PolicyStateValue: graph,
				ConfigValue:      config,
				ProviderSchemaSchema: providers.ProviderSchema{ResourceTypes: map[string]providers.Schema{
					"test_source": {Body: &configschema.Block{Attributes: map[string]*configschema.Attribute{
						"id": {Type: tc.sourceType, Computed: true},
					}}},
				}},
			}
			addr := mustResourceInstanceAddr("test_target.b")
			schema := &configschema.Block{Attributes: map[string]*configschema.Attribute{"value": tc.attribute}}
			recordPolicyOrigins(ctx, addr, config.Module.ResourceByAddr(addr.Resource.Resource), schema, []cty.Path{ctyutil.NormalizePath(tc.path)})
			origin, ok := graph.origin(globalref.ResourceAttr{Resource: addr, Attr: tc.path})
			if ok != tc.want {
				t.Fatalf("origin recorded = %t, want %t; origin = %s", ok, tc.want, origin.DebugString())
			}
			if ok && (!origin.Resource.Equal(mustResourceInstanceAddr("test_source.a")) || !origin.Attr.Equals(tc.sourcePath)) {
				t.Errorf("unexpected source %s, want test_source.a with path %#v", origin.DebugString(), tc.sourcePath)
			}
			if ctx.ProviderSchemaCalled && !ctx.ProviderSchemaAddr.Provider.Equals(addrs.NewDefaultProvider("test")) {
				t.Errorf("source provider = %s, want test", ctx.ProviderSchemaAddr)
			}
			if ctx.EvaluateExprCalled || ctx.EvaluateBlockCalled {
				t.Error("recording origins must not reevaluate configuration")
			}
		})
	}
}

func TestResourceOriginsRefreshOnly(t *testing.T) {
	config := testModuleInline(t, map[string]string{"main.tf": `
resource "test_source" "a" {}
resource "test_target" "b" { id = test_source.a.id }
`})
	p := new(testing_provider.MockProvider)
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_source": {Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Computed: true}}},
			"test_target": {Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Required: true}}},
		},
	})
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		t.Error("refresh-only must not plan configuration changes")
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}
	ctx := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
	}})
	state := states.BuildState(func(s *states.SyncState) {
		for addr, attrs := range map[string]string{"test_source.a": `{"id":"01"}`, "test_target.b": `{"id":"1"}`} {
			s.SetResourceInstanceCurrent(mustResourceInstanceAddr(addr), &states.ResourceInstanceObjectSrc{
				Status: states.ObjectReady, AttrsJSON: []byte(attrs),
			}, addrs.AbsProviderConfig{Provider: addrs.NewDefaultProvider("test")})
		}
	})
	client := policy.NewTestMockClient(t)
	rel := policy.Relationship{
		Name: "same_id", KeyType: "test_source", ValueType: "test_target",
		Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}}},
	}
	client.ConfigValue.Relationships = map[string][]policy.Relationship{"test_source": {rel}}
	defer ctx.acquireRun("plan")()
	graph, op, diags := ctx.planGraph(config, state, &PlanOpts{Mode: plans.RefreshOnlyMode, PolicyClient: client})
	tfdiags.AssertNoErrors(t, diags)
	walker, diags := ctx.walk(graph, op, &graphWalkOpts{
		Config: config, InputState: state, Changes: plans.NewChanges(), PolicyClient: client,
	})
	tfdiags.AssertNoErrors(t, diags)
	if walker.PolicyState.origins == nil {
		t.Fatal("relationships must enable the reference graph")
	}
	targetAddr := mustResourceInstanceAddr("test_target.b")
	if origin, ok := walker.PolicyState.origin(globalref.ResourceAttr{Resource: targetAddr, Attr: cty.GetAttrPath("id")}); ok {
		t.Errorf("refresh-only recorded a copy without planning configuration: %s", origin.DebugString())
	}
	for vertex := range walker.PolicyState.graph.VerticesSeq() {
		if node, ok := vertex.(*nodeResourcePolicy); ok && node.ResourceAddr.Equal(targetAddr) {
			if !node.After.GetAttr("id").RawEquals(cty.StringVal("1")) {
				t.Errorf("refresh-only did not retain the target's prior representation: %#v", node.After)
			}
			return
		}
	}
	t.Fatal("refresh-only did not schedule target policy evaluation")
}

func TestResourceOriginsAfterPlanFinalization(t *testing.T) {
	config := testModuleInline(t, map[string]string{"main.tf": `
resource "test_source" "a" {}
resource "test_target" "b" { id = test_source.a.id }
`})
	schema := &configschema.Block{Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Required: true}}}
	before := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("prior")})
	after := cty.ObjectVal(map[string]cty.Value{"id": cty.UnknownVal(cty.String).RefineNotNull()})
	for name, tc := range map[string]struct {
		action    plans.Action
		prior     cty.Value
		importing bool
		failWrite bool
		want      bool
	}{
		"fresh create":                           {action: plans.Create, prior: cty.NullVal(before.Type()), want: true},
		"replacement":                            {action: plans.DeleteThenCreate, prior: before},
		"replacement create with nonnull before": {action: plans.Create, prior: before},
		"imported prior":                         {action: plans.NoOp, prior: before, importing: true},
		"failed create finalization":             {action: plans.Create, prior: cty.NullVal(before.Type()), failWrite: true},
	} {
		t.Run(name, func(t *testing.T) {
			addr := mustResourceInstanceAddr("test_target.b")
			providerAddr := addrs.AbsProviderConfig{Provider: addrs.NewDefaultProvider("test")}
			graph := newPolicyWalkState(true)
			client := policy.NewTestMockClient(t)
			client.ConfigValue.Relationships = map[string][]policy.Relationship{"test_source": {{
				Name: "targets", KeyType: "test_source", ValueType: "test_target",
				Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}}},
			}}}
			ctx := &MockEvalContext{
				PolicyStateValue:  graph,
				PolicyClientValue: client,
				ConfigValue:       config,
				ProviderProvider:  new(testing_provider.MockProvider),
				ProviderSchemaSchema: providers.ProviderSchema{ResourceTypes: map[string]providers.Schema{
					"test_source": {Body: schema}, "test_target": {Body: schema},
				}},
				StateState:     states.NewState().SyncWrapper(),
				ChangesChanges: plans.NewChanges().SyncWrapper(),
				ChecksState:    checks.NewState(config),
				DeferralsState: deferring.NewDeferred(false),
			}
			instancePlanState := &states.ResourceInstanceObject{Value: after, Status: states.ObjectPlanned}
			if tc.failWrite {
				// Fail state serialization without disabling source-schema lookup:
				// recording before this failure would incorrectly leave an edge.
				instancePlanState.Value = cty.StringVal("invalid resource object")
			}
			n := &NodePlannableResourceInstance{
				NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
					Addr: addr,
					NodeAbstractResource: NodeAbstractResource{
						Config: config.Module.ResourceByAddr(addr.Resource.Resource), Schema: &providers.Schema{Body: schema},
						ResolvedProvider: providerAddr,
					},
				},
				importTarget: importTarget{target: cty.StringVal("prior")},
			}
			change := &plans.ResourceInstanceChange{
				Addr: addr, PrevRunAddr: addr, ProviderAddr: providerAddr,
				Change: plans.Change{Action: tc.action, Before: tc.prior, After: after},
			}
			diags := n.finalizePlan(ctx, nil, nil, tc.importing, change,
				&states.ResourceInstanceObject{Value: tc.prior, Status: states.ObjectReady},
				instancePlanState, instances.RepetitionData{})
			if diags.HasErrors() != tc.failWrite {
				t.Fatalf("finalization diagnostics = %s", diags.Err())
			}
			_, ok := graph.origin(globalref.ResourceAttr{Resource: addr, Attr: cty.GetAttrPath("id")})
			if ok != tc.want {
				t.Errorf("origin recorded = %t, want %t", ok, tc.want)
			}
		})
	}
}

func TestRecordResourceReferences(t *testing.T) {
	t.Run("configuration", func(t *testing.T) {
		sourceSchema := &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"id":       {Type: cty.String, Optional: true},
				"settings": {Type: cty.Object(map[string]cty.Type{"id": cty.String}), Optional: true},
			},
		}
		subjectSchema := &configschema.Block{Attributes: map[string]*configschema.Attribute{
			"bucket": {Type: cty.String, Required: true},
		}}
		for name, tc := range map[string]struct {
			expression       string
			middle           string
			middlePath       cty.Path
			lifecycle        string
			want             string
			wantMiddle       string
			wantSharedSource bool
		}{
			"direct": {expression: "test_bucket.source.id", want: "test_bucket.source.id", wantSharedSource: true},
			"local":  {expression: "local.source_id", want: "test_bucket.source.id", wantSharedSource: true},
			"resource chain": {
				expression: "test_bucket.middle.id", middle: "id = test_bucket.source.id",
				want: "test_bucket.middle.id", wantMiddle: "test_bucket.source.id",
			},
			"ignored terminal": {
				expression: "test_bucket.middle.id", middle: "id = test_bucket.source.id\nlifecycle { ignore_changes = [id] }",
				want: "test_bucket.middle.id",
			},
			"ignored destination": {
				expression: "test_bucket.source.id", lifecycle: "lifecycle { ignore_changes = [bucket] }",
			},
			"ignore all destination": {
				expression: "test_bucket.source.id", lifecycle: "lifecycle { ignore_changes = all }",
			},
			"data source terminal":  {expression: "data.test_bucket.lookup.id", want: "data.test_bucket.lookup.id"},
			"function":              {expression: "upper(test_bucket.source.id)"},
			"module output barrier": {expression: "module.child.id"},
			"nested object in chain": {
				expression: "test_bucket.middle.settings.id", middle: "settings = { id = test_bucket.source.id }", middlePath: cty.GetAttrPath("settings").GetAttr("id"),
				want: "test_bucket.middle.settings.id", wantMiddle: "test_bucket.source.id",
			},
			"missing configuration": {expression: "test_bucket.absent.id"},
		} {
			t.Run(name, func(t *testing.T) {
				config := testModuleInline(t, map[string]string{
					"main.tf": fmt.Sprintf(`
resource "test_bucket" "source" {}
resource "test_bucket" "middle" {
  %s
}
data "test_bucket" "lookup" {
  id = test_bucket.source.id
}
locals { source_id = test_bucket.source.id }
module "child" { source = "./child" }
resource "test_access_block" "subject" {
  bucket = %s
  %s
}
`, tc.middle, tc.expression, tc.lifecycle),
					"child/main.tf": `resource "test_bucket" "source" {}
output "id" { value = test_bucket.source.id }`,
				})
				graph := newPolicyWalkState(true)
				addr := mustResourceInstanceAddr("test_access_block.subject")
				ctx := &MockEvalContext{
					PolicyStateValue: graph,
					ConfigValue:      config,
					ProviderSchemaSchema: providers.ProviderSchema{
						ResourceTypes: map[string]providers.Schema{"test_bucket": {Body: sourceSchema}},
						DataSources:   map[string]providers.Schema{"test_bucket": {Body: sourceSchema}},
					},
				}
				recordPolicyOrigins(ctx, addr, config.Module.ResourceByAddr(addr.Resource.Resource), subjectSchema, []cty.Path{ctyutil.NormalizePath(cty.GetAttrPath("bucket"))})
				middleAddr := mustResourceInstanceAddr("test_bucket.middle")
				middlePath := tc.middlePath
				if middlePath == nil {
					middlePath = cty.GetAttrPath("id")
				}
				recordPolicyOrigins(ctx, middleAddr, config.Module.ResourceByAddr(middleAddr.Resource.Resource), sourceSchema, []cty.Path{ctyutil.NormalizePath(middlePath)})
				subject := globalref.ResourceAttr{Resource: addr, Attr: cty.GetAttrPath("bucket")}
				origin, ok := graph.origin(subject)
				if tc.want == "" {
					if ok {
						t.Fatalf("unexpected origin %s", origin.DebugString())
					}
				} else {
					if !ok {
						t.Fatal("expected a direct origin")
					}
					if got := origin.DebugString(); got != tc.want {
						t.Errorf("origin = %s, want %s", got, tc.want)
					}
					if !graph.shareOrigin(subject, origin) {
						t.Errorf("subject does not share the recorded origin %s", tc.want)
					}
					// Check the source's edge separately; it must not be followed
					// when looking up the subject's origin.
					middleOrigin, ok := graph.origin(origin)
					if tc.wantMiddle == "" {
						if ok {
							t.Errorf("unexpected source origin %s", middleOrigin.DebugString())
						}
					} else if !ok {
						t.Error("expected the source to have its own direct origin")
					} else if got := middleOrigin.DebugString(); got != tc.wantMiddle {
						t.Errorf("source origin = %s, want %s", got, tc.wantMiddle)
					}
				}
				source := globalref.ResourceAttr{Resource: mustResourceInstanceAddr("test_bucket.source"), Attr: cty.GetAttrPath("id")}
				if got := graph.shareOrigin(subject, source); got != tc.wantSharedSource {
					t.Errorf("shared source origin = %t, want %t", got, tc.wantSharedSource)
				}
				if ctx.EvaluateExprCalled || ctx.EvaluateBlockCalled {
					t.Error("recording origins must not reevaluate configuration")
				}
			})
		}
	})

	t.Run("static and dynamic blocks", func(t *testing.T) {
		leaf := configschema.Block{Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Required: true}}}
		schema := &configschema.Block{BlockTypes: map[string]*configschema.NestedBlock{
			"metadata": {Nesting: configschema.NestingSingle, Block: leaf},
			"rule":     {Nesting: configschema.NestingList, Block: leaf},
		}}
		for name, dynamic := range map[string]string{
			"static": "",
			"dynamic suppresses only its own block type": `dynamic "rule" {
  for_each = [1]
  content { id = test_source.a.id }
}`,
		} {
			t.Run(name, func(t *testing.T) {
				config := testModuleInline(t, map[string]string{"main.tf": fmt.Sprintf(`
resource "test_source" "a" {}
resource "test_target" "b" {
  metadata { id = test_source.a.id }
  rule { id = test_source.a.id }
  rule { id = test_source.a.other }
  %s
}
`, dynamic)})
				graph := newPolicyWalkState(true)
				ctx := &MockEvalContext{
					ConfigValue: config, PolicyStateValue: graph,
					ProviderSchemaSchema: providers.ProviderSchema{ResourceTypes: map[string]providers.Schema{
						"test_source": {Body: &configschema.Block{Attributes: map[string]*configschema.Attribute{
							"id":    {Type: cty.String, Computed: true},
							"other": {Type: cty.String, Computed: true},
						}}},
					}},
				}
				addr := mustResourceInstanceAddr("test_target.b")
				paths := []cty.Path{
					cty.GetAttrPath("metadata").GetAttr("id"),
					cty.GetAttrPath("rule").IndexInt(0).GetAttr("id"),
					cty.GetAttrPath("rule").IndexInt(1).GetAttr("id"),
				}
				var comparisonPaths []cty.Path
				for _, path := range paths {
					comparisonPaths = append(comparisonPaths, ctyutil.NormalizePath(path))
				}
				recordPolicyOrigins(ctx, addr, config.Module.ResourceByAddr(addr.Resource.Resource), schema, comparisonPaths)
				for i, path := range paths {
					origin, ok := graph.origin(globalref.ResourceAttr{Resource: addr, Attr: path})
					want := i == 0 || dynamic == ""
					if ok != want {
						t.Errorf("path %#v: origin recorded = %t, want %t", path, ok, want)
					}
					wantSource := "test_source.a.id"
					if i == 2 {
						wantSource = "test_source.a.other"
					}
					if ok && origin.DebugString() != wantSource {
						t.Errorf("origin = %s, want %s", origin.DebugString(), wantSource)
					}
				}
				if ctx.EvaluateExprCalled || ctx.EvaluateBlockCalled {
					t.Error("recording origins must not reevaluate configuration")
				}
			})
		}
	})

	t.Run("map paths", func(t *testing.T) {
		for _, ignored := range []bool{false, true} {
			t.Run(map[bool]string{false: "configured", true: "ignored"}[ignored], func(t *testing.T) {
				schema := &configschema.Block{Attributes: map[string]*configschema.Attribute{"settings": {Type: cty.Map(cty.String), Required: true}}}
				lifecycle := ""
				if ignored {
					lifecycle = "lifecycle { ignore_changes = [settings.id] }"
				}
				config := testModuleInline(t, map[string]string{"main.tf": fmt.Sprintf(`
resource "test_bucket" "source" {}
resource "test_access_block" "subject" {
  settings = { id = test_bucket.source.id }
  %s
}
`, lifecycle)})
				addr := mustResourceInstanceAddr("test_access_block.subject")
				graph := newPolicyWalkState(true)
				ctx := &MockEvalContext{
					PolicyStateValue: graph,
					ConfigValue:      config,
					ProviderSchemaSchema: providers.ProviderSchema{ResourceTypes: map[string]providers.Schema{
						"test_bucket": {Body: &configschema.Block{Attributes: map[string]*configschema.Attribute{
							"id": {Type: cty.String, Computed: true},
						}}},
					}},
				}
				paths := []cty.Path{cty.GetAttrPath("settings").GetAttr("id"), cty.GetAttrPath("settings").IndexString("id")}
				comparisonPaths := []cty.Path{ctyutil.NormalizePath(paths[0])}
				recordPolicyOrigins(ctx, addr, config.Module.ResourceByAddr(addr.Resource.Resource), schema, comparisonPaths)
				// Dot and bracket access identify the same compared member.
				source := globalref.ResourceAttr{Resource: mustResourceInstanceAddr("test_bucket.source"), Attr: cty.GetAttrPath("id")}
				for _, path := range paths {
					attr := globalref.ResourceAttr{Resource: addr, Attr: path}
					if got := graph.shareOrigin(attr, source); got == ignored {
						t.Error("map-member origin did not respect lifecycle configuration")
					}
				}
			})
		}
	})

	t.Run("constructors", func(t *testing.T) {
		for name, tc := range map[string]struct {
			filename, body string
			want           bool
		}{
			"native":                            {"subject.tf", `resource "test_target" "subject" { settings = { id = test_source.a.id } }`, true},
			"parenthesized remains unsupported": {"subject.tf", `resource "test_target" "subject" { settings = ({ id = test_source.a.id }) }`, false},
			"JSON literal is not a reference":   {"subject.tf.json", `{"resource":{"test_target":{"subject":{"settings":{"id":"test_source.a.id"}}}}}`, false},
			"last literal wins":                 {"subject.tf", `resource "test_target" "subject" { settings = { id = test_source.a.id, id = "literal" } }`, false},
			"last traversal wins":               {"subject.tf", `resource "test_target" "subject" { settings = { id = "literal", id = test_source.a.id } }`, true},
			"unknown key after":                 {"subject.tf", `resource "test_target" "subject" { settings = { id = test_source.a.id, (var.key) = "literal" } }`, false},
		} {
			t.Run(name, func(t *testing.T) {
				config := testModuleInline(t, map[string]string{
					"main.tf": `resource "test_source" "a" {}
variable "key" { type = string }`,
					tc.filename: tc.body,
				})
				graph := newPolicyWalkState(true)
				ctx := &MockEvalContext{ConfigValue: config, PolicyStateValue: graph}
				addr := mustResourceInstanceAddr("test_target.subject")
				schema := &configschema.Block{Attributes: map[string]*configschema.Attribute{
					"settings": {Type: cty.Map(cty.String), Required: true},
				}}
				recordPolicyOrigins(ctx, addr, config.Module.ResourceByAddr(addr.Resource.Resource), schema, []cty.Path{ctyutil.NormalizePath(cty.GetAttrPath("settings").GetAttr("id"))})
				origin, ok := graph.origin(globalref.ResourceAttr{Resource: addr, Attr: cty.GetAttrPath("settings").GetAttr("id")})
				if ok != tc.want {
					t.Fatalf("origin recorded = %t (%s), want %t", ok, origin.DebugString(), tc.want)
				}
				if ok && origin.DebugString() != "test_source.a.id" {
					t.Errorf("origin = %s, want test_source.a.id", origin.DebugString())
				}
				if ctx.EvaluateExprCalled || ctx.EvaluateBlockCalled {
					t.Error("origin inspection must not reevaluate configuration")
				}
			})
		}
	})
}

func testResourceAttr(t *testing.T, src string) globalref.ResourceAttr {
	t.Helper()
	traversal, diags := hclsyntax.ParseTraversalAbs([]byte(src), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	ref, refDiags := addrs.ParseRef(traversal)
	if refDiags.HasErrors() {
		t.Fatal(refDiags.Err())
	}
	attr, ok := (globalref.Reference{ContainerAddr: addrs.RootModuleInstance, LocalRef: ref}).ResourceAttr()
	if !ok {
		t.Fatalf("not a resource attribute: %s", src)
	}
	return attr
}

func TestResourceOrigin(t *testing.T) {
	for name, tc := range map[string]struct {
		edges        [][2]string
		lookup, want string
	}{
		"absent": {lookup: "test_bucket.a.bucket"},
		"immediate, not transitive": {
			edges:  [][2]string{{"test_bucket.a.bucket", "test_bucket.b.id"}, {"test_bucket.b.id", "test_bucket.c.id"}},
			lookup: "test_bucket.a.bucket", want: "test_bucket.b.id",
		},
		"self edge omitted": {
			edges: [][2]string{{"test_bucket.a.settings.id", `test_bucket.a.settings["id"]`}}, lookup: "test_bucket.a.settings.id",
		},
		"dot and bracket lookup": {
			edges: [][2]string{{`test_bucket.a.settings["id"]`, "test_bucket.b.id"}}, lookup: "test_bucket.a.settings.id", want: "test_bucket.b.id",
		},
		"literal dotted key": {
			edges: [][2]string{{`test_bucket.a.settings["a.b"]`, "test_bucket.b.id"}}, lookup: "test_bucket.a.settings.a.b",
		},
		"instance identity": {
			edges: [][2]string{{"test_bucket.a[0].bucket", "test_bucket.b.id"}}, lookup: "test_bucket.a[1].bucket",
		},
		"data source is terminal": {
			edges:  [][2]string{{"test_bucket.a.bucket", "data.test_bucket.b.id"}, {"data.test_bucket.b.id", "test_bucket.c.id"}},
			lookup: "test_bucket.a.bucket", want: "data.test_bucket.b.id",
		},
		"data source has no outgoing edge": {
			edges: [][2]string{{"data.test_bucket.b.id", "test_bucket.c.id"}}, lookup: "data.test_bucket.b.id",
		},
	} {
		t.Run(name, func(t *testing.T) {
			graph := newPolicyWalkState(true)
			for _, edge := range tc.edges {
				graph.recordOrigin(testResourceAttr(t, edge[0]), testResourceAttr(t, edge[1]))
			}
			got, ok := graph.origin(testResourceAttr(t, tc.lookup))
			if tc.want == "" {
				if ok {
					t.Fatalf("unexpected origin %s", got.DebugString())
				}
				return
			}
			want := testResourceAttr(t, tc.want)
			if !ok || !got.Resource.Equal(want.Resource) || !got.Attr.Equals(want.Attr) {
				t.Fatalf("origin = %s (%t), want %s", got.DebugString(), ok, tc.want)
			}
		})
	}
	t.Run("module boundaries", func(t *testing.T) {
		root := addrs.RootModuleInstance
		child := root.Child("child", addrs.IntKey(0))
		for _, target := range []addrs.ModuleInstance{child, root, root.Child("child", addrs.IntKey(1))} {
			graph := newPolicyWalkState(true)
			from, to := testResourceAttr(t, "test_bucket.a.bucket"), testResourceAttr(t, "test_bucket.b.id")
			from.Resource.Module, to.Resource.Module = child, target
			graph.recordOrigin(from, to)
			got, ok := graph.origin(from)
			if ok != child.Equal(target) || (ok && !got.Resource.Equal(to.Resource)) {
				t.Fatalf("origin crossed module boundary: %s (%t)", got.DebugString(), ok)
			}
		}
	})
}

func TestCompareOrigins(t *testing.T) {
	source, middle := "test_bucket.source.id", "test_bucket.middle.id"
	subject, other := "test_access_block.subject.bucket", "test_access_block.other.bucket"
	dot, bracket := "test_bucket.source.settings.id", `test_bucket.source.settings["id"]`
	for name, tc := range map[string]struct {
		a, b           string
		edges          [][2]string
		nilGraph, want bool
	}{
		"self":                                   {a: source, b: source, want: true},
		"nil disables self proof":                {a: source, b: source, nilGraph: true},
		"direct source":                          {a: subject, b: source, edges: [][2]string{{subject, source}}, want: true},
		"shared direct source":                   {a: subject, b: other, edges: [][2]string{{subject, source}, {other, source}}, want: true},
		"source has its own origin":              {a: subject, b: middle, edges: [][2]string{{subject, middle}, {middle, source}}, want: true},
		"chain is not followed":                  {a: subject, b: source, edges: [][2]string{{subject, middle}, {middle, source}}},
		"indirect shared source is not followed": {a: subject, b: other, edges: [][2]string{{subject, middle}, {middle, source}, {other, source}}},
		"shared immediate origin in cycle":       {a: subject, b: source, edges: [][2]string{{subject, middle}, {middle, source}, {source, middle}}, want: true},
		"disjoint cycle":                         {a: subject, b: other, edges: [][2]string{{subject, middle}, {middle, subject}}},
		"dot and bracket":                        {a: dot, b: bracket, want: true},
		"canonical upstream source":              {a: subject, b: other, edges: [][2]string{{subject, dot}, {other, bracket}}, want: true},
		"dotted key is not a nested path":        {a: `test_bucket.source["settings.id"]`, b: dot},
	} {
		t.Run(name, func(t *testing.T) {
			var graph *policyWalkState
			if !tc.nilGraph {
				graph = newPolicyWalkState(true)
				for _, edge := range tc.edges {
					graph.recordOrigin(testResourceAttr(t, edge[0]), testResourceAttr(t, edge[1]))
				}
			}
			a, b := testResourceAttr(t, tc.a), testResourceAttr(t, tc.b)
			if graph.shareOrigin(a, b) != tc.want || graph.shareOrigin(b, a) != tc.want {
				t.Fatalf("CompareOrigins(%s, %s), want %t in both directions", tc.a, tc.b, tc.want)
			}
		})
	}
}

func TestResolvePolicyResourceAttr(t *testing.T) {
	module := &configs.Module{Locals: make(map[string]*configs.Local)}
	for name, src := range map[string]string{
		"source": "test_bucket.source", "bucket": "local.source", "settings": "local.bucket.settings",
		"input": "var.bucket_id", "output": "module.child[0].bucket_id",
		"function": "lower(test_bucket.source.id)", "literal": `"id"`,
		"object": "{ id = test_bucket.source.id }", "index": "test_bucket.source.settings[var.key]",
		"self": "local.self.id", "cycle_a": "local.cycle_b.id", "cycle_b": "local.cycle_a.id",
	} {
		expr, diags := hclsyntax.ParseExpression([]byte(src), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}
		module.Locals[name] = &configs.Local{Name: name, Expr: expr}
	}
	parseTraversal := func(src string) hcl.Traversal {
		t.Helper()
		traversal, diags := hclsyntax.ParseTraversalAbs([]byte(src), "test.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}
		return traversal
	}
	for name, tc := range map[string]struct{ ref, want string }{
		"direct":           {"test_bucket.source.id", "test_bucket.source.id"},
		"data source":      {"data.test_bucket.source.id", "data.test_bucket.source.id"},
		"local chain":      {"local.bucket.id", "test_bucket.source.id"},
		"remaining suffix": {`local.settings["id"]`, `test_bucket.source.settings["id"]`},
		"whole resource":   {ref: "test_bucket.source"}, "whole alias": {ref: "local.bucket"},
		"variable": {ref: "var.bucket_id"}, "variable alias": {ref: "local.input"},
		"module output": {ref: "module.child[0].bucket_id"}, "output alias": {ref: "local.output"},
		"function": {ref: "local.function"}, "literal": {ref: "local.literal"},
		"constructor": {ref: "local.object.id"}, "computed index": {ref: "local.index"},
		"self cycle with growing suffix": {ref: "local.self"}, "mutual cycle with growing suffix": {ref: "local.cycle_a"},
	} {
		t.Run(name, func(t *testing.T) {
			traversal := parseTraversal(tc.ref)
			before := append(hcl.Traversal(nil), traversal...)
			got, ok := resolvePolicyResourceAttr(module, addrs.RootModuleInstance, traversal)
			if tc.want == "" {
				if ok {
					t.Fatalf("unexpected resource attribute %s", got.DebugString())
				}
			} else {
				want := testResourceAttr(t, tc.want)
				if !ok || !got.Resource.Equal(want.Resource) || !got.Attr.Equals(want.Attr) {
					t.Errorf("got %s (%t), want %s", got.DebugString(), ok, tc.want)
				}
			}
			for i, step := range traversal {
				if step != before[i] {
					t.Fatal("resolver mutated the input traversal")
				}
			}
		})
	}
	t.Run("preserves module instance keys", func(t *testing.T) {
		root := addrs.RootModuleInstance
		for _, addr := range []addrs.ModuleInstance{root.Child("child", addrs.IntKey(0)), root.Child("child", addrs.IntKey(1)), root.Child("child", addrs.StringKey("key"))} {
			got, ok := resolvePolicyResourceAttr(module, addr, parseTraversal(`local.settings["id"]`))
			want := testResourceAttr(t, `test_bucket.source.settings["id"]`)
			want.Resource.Module = addr
			if !ok || !got.Resource.Equal(want.Resource) || !got.Attr.Equals(want.Attr) {
				t.Fatalf("got %s (%t), want %s", got.DebugString(), ok, want.DebugString())
			}
		}
	})
	if _, ok := resolvePolicyResourceAttr(nil, addrs.RootModuleInstance, parseTraversal("local.bucket.id")); ok {
		t.Fatal("a missing module must not resolve a local")
	}
}

func TestResolvePolicyResourceAttrEffectiveOverride(t *testing.T) {
	dir := t.TempDir()
	for name, src := range map[string]string{
		"main.tf":     `locals { source = test_bucket.original }`,
		"override.tf": `locals { source = test_bucket.replacement }`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
	}
	module, diags := configs.NewParser(nil).LoadConfigDir(dir)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	traversal, diags := hclsyntax.ParseTraversalAbs([]byte("local.source.id"), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	got, ok := resolvePolicyResourceAttr(module, addrs.RootModuleInstance, traversal)
	if !ok || got.DebugString() != "test_bucket.replacement.id" {
		t.Fatalf("got %s (%t), want the merged override expression", got.DebugString(), ok)
	}
}
