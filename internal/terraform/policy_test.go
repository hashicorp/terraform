// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

// policyResourceValues returns the values of getresources matches without
// their marks.
func policyResourceValues(resources []callback.Resource) []cty.Value {
	ret := make([]cty.Value, len(resources))
	for i, r := range resources {
		ret[i], _ = r.Value.UnmarkDeep()
	}
	return ret
}

func TestGetResourcesForPolicyCallback(t *testing.T) {
	providerSchema := *getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"name": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"child": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"value": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
		},
	})

	config := testModuleInline(t, map[string]string{
		"main.tf": `
resource "test_resource" "a" {
  name = "alpha"

  child {
    value = "one"
  }
}

resource "test_resource" "b" {
  name = "beta"

  child {
    value = "two"
  }
}
`,
	})

	state := states.NewState().SyncWrapper()
	state.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_resource.a"),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"a","name":"alpha","child":{"value":"one"}}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	state.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_resource.b"),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"b","name":"beta","child":{"value":"two"}}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	state.Close()
	before := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("a"), "name": cty.StringVal("old"), "child": cty.NullVal(cty.Object(map[string]cty.Type{"value": cty.String}))})

	ctx := &MockEvalContext{
		StateState:     state,
		ChangesChanges: plans.NewChanges().SyncWrapper(),
		DeferralsState: deferring.NewDeferred(false),
		PolicyGraphValue: &policySubgraph{generation: 7, run: &policyRunOpts{
			Apply: true,
			AppliedChanges: []*plans.ResourceInstanceChange{{
				Addr:         mustResourceInstanceAddr("test_resource.a"),
				ProviderAddr: mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
				Change:       plans.Change{Action: plans.Update, Before: before},
			}},
		}},
	}

	getResources := getResourcesForPolicyCallback(ctx, walkApply, nil, providerSchema, config)

	tests := []struct {
		name        string
		filter      cty.Value
		wantNames   []string
		wantUnknown bool
	}{
		{
			name:        "null filter returns all matching resources",
			filter:      cty.NullVal(cty.DynamicPseudoType),
			wantNames:   []string{"alpha", "beta"},
			wantUnknown: false,
		},
		{
			name: "scalar attribute filter matches a subset",
			filter: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("alpha"),
			}),
			wantNames:   []string{"alpha"},
			wantUnknown: false,
		},
		{
			name: "nested block filter filter matches a subset",
			filter: cty.ObjectVal(map[string]cty.Value{
				"child": cty.ObjectVal(map[string]cty.Value{
					"value": cty.StringVal("one"),
				}),
			}),
			wantNames:   []string{"alpha"},
			wantUnknown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got        []callback.Resource
				gotUnknown bool
				gotErr     error
			)

			got, gotUnknown, gotErr = getResources(t.Context(), "test_resource", tt.filter)
			if gotErr != nil {
				t.Fatalf("unexpected error: %v", gotErr)
			}
			if gotUnknown != tt.wantUnknown {
				t.Fatalf("wrong unknown result\ngot:  %t\nwant: %t", gotUnknown, tt.wantUnknown)
			}

			gotNames := make([]string, 0, len(got))
			for _, resource := range got {
				name := resource.Value.GetAttr("name").AsString()
				gotNames = append(gotNames, name)
				// Each match is the object of the candidates table, so the
				// policy client can send it by identifier.
				obj := resource.Object
				if obj == nil {
					t.Fatalf("no object for %s", name)
				}
				wantOp, wantPrior := proto.Operation_NO_OP, cty.NilVal
				if name == "alpha" {
					wantOp, wantPrior = proto.Operation_UPDATE, before
				}
				if obj.Generation != 7 || obj.Address != "test_resource."+name[:1] || obj.Type != "test_resource" || obj.Metadata.Operation != wantOp || obj.Metadata.Address != obj.Address || obj.Schema != providerSchema.ResourceTypes["test_resource"].Body {
					t.Errorf("wrong object of %s: %#v", name, obj)
				}
				if !obj.Attrs.RawEquals(resource.Value) || (obj.PriorAttrs == cty.NilVal) != (wantPrior == cty.NilVal) || (wantPrior != cty.NilVal && !obj.PriorAttrs.RawEquals(wantPrior)) {
					t.Errorf("wrong values of %s: %#v, %#v", name, obj.Attrs, obj.PriorAttrs)
				}
			}
			sort.Strings(gotNames)

			wantNames := append([]string{}, tt.wantNames...)
			sort.Strings(wantNames)
			if diff := cmp.Diff(wantNames, gotNames); diff != "" {
				t.Fatalf("wrong matched resources (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPolicyOperationForAction(t *testing.T) {
	tests := []struct {
		name      string
		action    plans.Action
		want      proto.Operation
		wantValid bool
	}{
		{name: "create", action: plans.Create, want: proto.Operation_CREATE, wantValid: true},
		{name: "update", action: plans.Update, want: proto.Operation_UPDATE, wantValid: true},
		{name: "delete-then-create", action: plans.DeleteThenCreate, wantValid: false},
		{name: "create-then-delete", action: plans.CreateThenDelete, wantValid: false},
		{name: "delete", action: plans.Delete, want: proto.Operation_DELETE, wantValid: true},
		{name: "no-op", action: plans.NoOp, want: proto.Operation_NO_OP, wantValid: true},
		{name: "read", action: plans.Read, wantValid: false},
		{name: "forget", action: plans.Forget, wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := policyOperationForAction(tt.action)
			if ok != tt.wantValid {
				t.Fatalf("wrong validity for %s\ngot:  %t\nwant: %t", tt.action, ok, tt.wantValid)
			}
			if !tt.wantValid {
				return
			}
			if got != tt.want {
				t.Fatalf("wrong operation for %s\ngot:  %s\nwant: %s", tt.action, got, tt.want)
			}
		})
	}
}
