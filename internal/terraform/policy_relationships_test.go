// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestValidatePolicyRelationships(t *testing.T) {
	provider := addrs.NewDefaultProvider("test")
	schemas := &Schemas{Providers: map[addrs.Provider]providers.ProviderSchema{
		provider: {ResourceTypes: map[string]providers.Schema{
			"test_bucket": {Body: &configschema.Block{Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Computed: true},
			}}},
			"test_access_block": {Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{"bucket": {Type: cty.String, Optional: true}},
				BlockTypes: map[string]*configschema.NestedBlock{
					"metadata": {Nesting: configschema.NestingSingle, Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{"id": {Type: cty.String, Optional: true}},
					}},
				},
			}},
		}},
	}}
	locks := map[addrs.Provider]*depsfile.ProviderLock{
		provider: depsfile.NewProviderLock(provider, providerreqs.MustParseVersion("1.2.3"), nil, nil),
	}
	relationship := func(keyPath, valuePath cty.Path) policy.Relationship {
		return policy.Relationship{Name: "access_blocks", KeyType: "test_bucket", ValueType: "test_access_block",
			Matches: []policy.Match{{{KeyPath: keyPath, ValuePath: valuePath}}},
		}
	}
	for name, tc := range map[string]struct {
		relationships map[string][]policy.Relationship
		want          []string
	}{
		"valid": {relationships: map[string][]policy.Relationship{
			"test_bucket": {relationship(cty.GetAttrPath("id"), cty.GetAttrPath("bucket"))},
		}},
		"valid nested": {relationships: map[string][]policy.Relationship{
			"test_bucket": {relationship(cty.GetAttrPath("id"), cty.GetAttrPath("metadata").GetAttr("id"))},
		}},
		"invalid value path": {
			relationships: map[string][]policy.Relationship{"test_bucket": {relationship(cty.GetAttrPath("id"), cty.GetAttrPath("bukcet"))}},
			want:          []string{`The resource type test_access_block has no attribute "bukcet" in provider registry.terraform.io/hashicorp/test 1.2.3.`},
		},
		"invalid nested key path reported once": {
			relationships: map[string][]policy.Relationship{
				"test_bucket":       {{Name: "outer", KeyType: "test_bucket", ValueType: "test_access_block", Relationships: []policy.Relationship{relationship(cty.GetAttrPath("arn"), cty.GetAttrPath("bucket"))}}},
				"test_access_block": {relationship(cty.GetAttrPath("arn"), cty.GetAttrPath("bucket"))},
			},
			want: []string{`The resource type test_bucket has no attribute "arn" in provider registry.terraform.io/hashicorp/test 1.2.3.`},
		},
		"unknown type is not checked": {relationships: map[string][]policy.Relationship{
			"aws_lb": {{Name: "listeners", KeyType: "aws_lb", ValueType: "aws_lb_listener", Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("arn"), ValuePath: cty.GetAttrPath("load_balancer_arn")}}}}},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			diags := validatePolicyRelationships(&policy.Config{Relationships: tc.relationships}, schemas, locks)
			if len(diags) != len(tc.want) {
				t.Fatalf("wrong number of diagnostics: got %d, want %d\n%s", len(diags), len(tc.want), diags.AsTerraformDiags().Err())
			}
			for i, diag := range diags.AsTerraformDiags() {
				if got := diag.Description().Summary; got != "Invalid policy relationship" {
					t.Errorf("wrong summary: %s", got)
				}
				if got := diag.Description().Detail; got != tc.want[i] {
					t.Errorf("wrong detail\ngot:  %s\nwant: %s", got, tc.want[i])
				}
			}
		})
	}

	object := cty.Object(map[string]cty.Type{"id": cty.String})
	block := func(nesting configschema.NestingMode) *configschema.NestedBlock {
		return &configschema.NestedBlock{Nesting: nesting, Block: configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Optional: true}, "other": {Type: cty.DynamicPseudoType, Optional: true},
			},
		}}
	}
	// Endpoint wiring belongs here; exhaustive structural path permutations
	// belong to configschema. Each row explicitly selects one endpoint.
	for name, tc := range map[string]struct {
		side        string
		typ         cty.Type
		block       *configschema.NestedBlock
		path        cty.Path
		wantType    string
		wantMissing bool
	}{
		"string key":                           {side: "key", typ: cty.String},
		"number value":                         {side: "value", typ: cty.Number},
		"bool key":                             {side: "key", typ: cty.Bool},
		"list key":                             {side: "key", typ: cty.List(cty.String), wantType: "list of string"},
		"object value":                         {side: "value", typ: object, wantType: "object"},
		"dynamic key":                          {side: "key", typ: cty.DynamicPseudoType, wantType: "dynamic"},
		"static block leaf":                    {side: "value", block: block(configschema.NestingSingle), path: cty.GetAttrPath("value").GetAttr("id")},
		"list block leaf with dynamic sibling": {side: "key", block: block(configschema.NestingList), path: cty.GetAttrPath("value").IndexInt(0).GetAttr("id")},
		"map primitive value":                  {side: "value", typ: cty.Map(cty.String), path: cty.GetAttrPath("value").IndexString("Environment")},
		"nested composite key":                 {side: "key", typ: cty.List(object), path: cty.GetAttrPath("value").IndexInt(0), wantType: "object"},
		"missing object member":                {side: "value", typ: object, path: cty.GetAttrPath("value").GetAttr("missing"), wantMissing: true},
		"set element is not addressable":       {side: "key", typ: cty.Set(cty.String), path: cty.GetAttrPath("value").IndexString("id"), wantMissing: true},
	} {
		t.Run(name, func(t *testing.T) {
			bodies := map[string]*configschema.Block{}
			for _, side := range []string{"key", "value"} {
				bodies[side] = &configschema.Block{Attributes: map[string]*configschema.Attribute{"value": {Type: cty.String, Optional: true}}}
			}
			if tc.block != nil {
				bodies[tc.side] = &configschema.Block{BlockTypes: map[string]*configschema.NestedBlock{"value": tc.block}}
			} else {
				bodies[tc.side].Attributes["value"].Type = tc.typ
			}
			schemas := &Schemas{Providers: map[addrs.Provider]providers.ProviderSchema{
				provider: {ResourceTypes: map[string]providers.Schema{"test_key": {Body: bodies["key"]}, "test_value": {Body: bodies["value"]}}},
			}}
			paths := map[string]cty.Path{"key": cty.GetAttrPath("value"), "value": cty.GetAttrPath("value")}
			if tc.path != nil {
				paths[tc.side] = tc.path
			}
			diags := validatePolicyRelationships(&policy.Config{Relationships: map[string][]policy.Relationship{
				"test_key": {{Name: "values", KeyType: "test_key", ValueType: "test_value",
					Matches: []policy.Match{{{KeyPath: paths["key"], ValuePath: paths["value"]}}},
				}},
			}}, schemas, nil).AsTerraformDiags()
			if tc.wantType == "" && !tc.wantMissing {
				tfdiags.AssertNoDiagnostics(t, diags)
				return
			}
			if len(diags) != 1 {
				t.Fatalf("expected one diagnostic, got %d: %s", len(diags), diags.Err())
			}
			desc := diags[0].Description()
			want := "must have a primitive type (string, number, or bool), but has type " + tc.wantType + "."
			if tc.wantMissing {
				want = "has no attribute"
			}
			if desc.Summary != "Invalid policy relationship" || !strings.Contains(desc.Detail, want) || !strings.Contains(desc.Detail, "test_"+tc.side) {
				t.Errorf("unexpected diagnostic: %#v", desc)
			}
		})
	}

	for _, side := range []string{"key", "value"} {
		t.Run("repeated name/"+side, func(t *testing.T) {
			body := &configschema.Block{Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Optional: true}, "settings": {Type: object, Optional: true},
			}}
			schemas := &Schemas{Providers: map[addrs.Provider]providers.ProviderSchema{
				provider: {ResourceTypes: map[string]providers.Schema{"test_key": {Body: body}, "test_value": {Body: body}}},
			}}
			valid := policy.Relationship{Name: "reused", KeyType: "test_key", ValueType: "test_value",
				Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}}},
			}
			invalid := valid
			pair := policy.RelationshipKey{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}
			relationships := map[string][]policy.Relationship{"test_key": {valid}}
			if side == "key" {
				pair.KeyPath = cty.GetAttrPath("settings")
				invalid.Matches = []policy.Match{{pair}}
				valid.Relationships = []policy.Relationship{invalid}
				relationships["test_key"] = []policy.Relationship{valid}
			} else {
				pair.ValuePath = cty.GetAttrPath("settings")
				invalid.Matches = []policy.Match{{pair}}
				relationships["test_value"] = []policy.Relationship{invalid}
			}
			if diags := validatePolicyRelationships(&policy.Config{Relationships: relationships}, schemas, nil); len(diags) != 1 {
				t.Fatalf("expected the unvalidated composite endpoint to be rejected, got %d diagnostics", len(diags))
			}
		})
	}
}

func TestContext2Plan_PolicyRelationships(t *testing.T) {
	mainConfig := `
terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      version = "1.0.0"
    }
  }
}
resource "test_instance" "test" { value = "foo" }
`
	relationship := func(valuePath cty.Path) policy.Relationship {
		return policy.Relationship{Name: "instances", KeyType: "test_instance", ValueType: "test_resource",
			Matches: []policy.Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: valuePath}}},
		}
	}
	for name, tc := range map[string]struct {
		path         cty.Path
		unused       bool
		wantEvaluate bool
		wantDiags    int
		wantDetail   string
	}{
		"valid usage":             {path: cty.GetAttrPath("value"), wantEvaluate: true},
		"invalid usage":           {path: cty.GetAttrPath("valeu"), wantDiags: 1},
		"valid unused definition": {path: cty.GetAttrPath("value"), unused: true, wantEvaluate: true},
		"invalid unused definition without usage": {path: cty.GetAttrPath("valeu"), unused: true, wantDiags: 1},
		"composite fails before evaluation": {path: cty.GetAttrPath("nesting_single"), wantDiags: 1,
			wantDetail: "must have a primitive type (string, number, or bool), but has type object."},
	} {
		t.Run(name, func(t *testing.T) {
			mod := testModuleInline(t, map[string]string{"main.tf": mainConfig, "main.tfpolicy.hcl": samplePolicyConfig})
			client := policy.NewTestMockClient(t)
			if tc.unused {
				client.ConfigValue = &policy.Config{Definitions: map[string]policy.Relationship{"unused": relationship(tc.path)}}
			} else {
				client.ConfigValue = &policy.Config{Relationships: map[string][]policy.Relationship{"test_instance": {relationship(tc.path)}}}
			}
			ctx := testContext2(t, &ContextOpts{Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): testProviderFuncFixed(testProvider("test")),
			}})
			_, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{Mode: plans.NormalMode, PolicyClient: client})
			if client.EvaluateCalled != tc.wantEvaluate {
				t.Errorf("wrong EvaluateCalled: got %t, want %t", client.EvaluateCalled, tc.wantEvaluate)
			}
			tfdiags.AssertDiagnosticCount(t, diags, tc.wantDiags)
			for _, diag := range diags {
				if diag.Description().Summary != "Invalid policy relationship" || (tc.wantDetail != "" && !strings.Contains(diag.Description().Detail, tc.wantDetail)) {
					t.Errorf("wrong diagnostic: %#v", diag.Description())
				}
			}
		})
	}
}
