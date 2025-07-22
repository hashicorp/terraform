// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Apply_identity(t *testing.T) {
	for name, tc := range map[string]struct {
		mode            plans.Mode
		prevRunState    *states.State
		requiresReplace []cty.Path
		plannedIdentity cty.Value
		appliedIdentity cty.Value

		expectedIdentity  cty.Value
		expectDiagnostics tfdiags.Diagnostics
	}{
		"create": {
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			expectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
		},
		"create - invalid applied identity schema": {
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			appliedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.BoolVal(false),
			}),
			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Error, "Provider produced an identity that doesn't match the schema", "Provider \"registry.terraform.io/hashicorp/test\" returned an identity for test_resource.test that doesn't match the identity schema: .id: string required, but received bool. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
			},
		},

		"update": {
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "test",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"foo"}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			}),
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
			expectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
		},

		"delete": {
			mode: plans.DestroyMode,
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "test",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:                states.ObjectReady,
						AttrsJSON:             []byte(`{"id":"bar"}`),
						IdentitySchemaVersion: 0,
						IdentityJSON:          []byte(`{"id":"bar"}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			}),
			plannedIdentity: cty.NilVal,
			expectedIdentity: cty.NullVal(cty.Object(map[string]cty.Type{
				"id": cty.String,
			})),
		},
		"replace": {
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "test",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:                states.ObjectReady,
						AttrsJSON:             []byte(`{"id":"foo"}`),
						IdentitySchemaVersion: 0,
						IdentityJSON:          []byte(`{"id":"foo"}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			}),
			requiresReplace: []cty.Path{cty.GetAttrPath("id")},
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
			expectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, map[string]string{
				"main.tf": `
                resource "test_resource" "test" {
                  id = "bar"
                }
                `,
			})
			p := testProvider("test")
			p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
				ResourceTypes: map[string]*configschema.Block{
					"test_resource": {
						Attributes: map[string]*configschema.Attribute{
							"id": {
								Type:     cty.String,
								Optional: true,
							},
						},
					},
				},
				IdentityTypes: map[string]*configschema.Object{
					"test_resource": &configschema.Object{
						Attributes: map[string]*configschema.Attribute{
							"id": {
								Type:     cty.String,
								Required: true,
							},
						},
						Nesting: configschema.NestingSingle,
					},
				},
				IdentityTypeSchemaVersions: map[string]uint64{
					"test_resource": 0,
				},
			})

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
			})

			p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				return providers.PlanResourceChangeResponse{
					PlannedState:    req.ProposedNewState,
					PlannedIdentity: tc.plannedIdentity,
					RequiresReplace: tc.requiresReplace,
				}
			}

			if !tc.appliedIdentity.IsNull() {
				p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
					resp := providers.ApplyResourceChangeResponse{}
					resp.NewState = req.PlannedState
					resp.NewIdentity = tc.appliedIdentity
					return resp
				}
			}

			plan, diags := ctx.Plan(m, tc.prevRunState, &PlanOpts{Mode: tc.mode})
			tfdiags.AssertNoDiagnostics(t, diags)

			state, diags := ctx.Apply(plan, m, nil)
			if tc.expectDiagnostics.HasErrors() {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectDiagnostics)
				return
			}
			tfdiags.AssertNoDiagnostics(t, diags)

			if !tc.expectedIdentity.IsNull() {
				schema := p.GetProviderSchemaResponse.ResourceTypes["test_resource"]

				resourceInstanceStateSrc := state.Modules[""].Resources["test_resource.test"].Instance(addrs.NoKey).Current

				resourceInstanceState, err := resourceInstanceStateSrc.Decode(schema)
				if err != nil {
					t.Fatalf("failed to decode resource instance state: %s", err)
				}

				if !resourceInstanceState.Identity.RawEquals(tc.expectedIdentity) {
					t.Fatalf("unexpected identity: \n expected: %s\n got: %s", tc.expectedIdentity.GoString(), resourceInstanceState.Identity.GoString())
				}
			}
		})
	}
}
