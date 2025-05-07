// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestContext2Plan_resource_identity_refresh(t *testing.T) {
	for name, tc := range map[string]struct {
		StoredIdentitySchemaVersion         uint64
		StoredIdentityJSON                  []byte
		IdentitySchema                      providers.IdentitySchema
		IdentityData                        cty.Value
		ExpectedIdentity                    cty.Value
		ExpectedError                       error
		ExpectUpgradeResourceIdentityCalled bool
		UpgradeResourceIdentityResponse     providers.UpgradeResourceIdentityResponse
	}{
		"no previous identity": {
			IdentitySchema: providers.IdentitySchema{
				Version: 0,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			ExpectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
		},
		"identity version mismatch": {
			StoredIdentitySchemaVersion: 1,
			StoredIdentityJSON:          []byte(`{"id": "foo"}`),
			IdentitySchema: providers.IdentitySchema{
				Version: 0,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			ExpectedError: fmt.Errorf("Resource instance managed by newer provider version: The current state of aws_instance.web was created by a newer provider version than is currently selected. Upgrade the aws provider to work with this state."),
		},
		"identity type mismatch": {
			StoredIdentitySchemaVersion: 0,
			StoredIdentityJSON:          []byte(`{"arn": "foo"}`),
			IdentitySchema: providers.IdentitySchema{
				Version: 0,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			ExpectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			ExpectedError: fmt.Errorf("failed to decode identity: unsupported attribute \"arn\". This is most likely a bug in the Provider, providers must not change the identity schema without updating the identity schema version"),
		},
		"identity upgrade succeeds": {
			StoredIdentitySchemaVersion: 1,
			StoredIdentityJSON:          []byte(`{"arn": "foo"}`),
			IdentitySchema: providers.IdentitySchema{
				Version: 2,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			UpgradeResourceIdentityResponse: providers.UpgradeResourceIdentityResponse{
				UpgradedIdentity: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("foo"),
				}),
			},
			ExpectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			ExpectUpgradeResourceIdentityCalled: true,
		},
		"identity upgrade failed": {
			StoredIdentitySchemaVersion: 1,
			StoredIdentityJSON:          []byte(`{"id": "foo"}`),
			IdentitySchema: providers.IdentitySchema{
				Version: 2,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"arn": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"arn": cty.StringVal("arn:foo"),
			}),
			UpgradeResourceIdentityResponse: providers.UpgradeResourceIdentityResponse{
				UpgradedIdentity: cty.NilVal,
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.Sourceless(tfdiags.Error, "failed to upgrade resource identity", "provider was unable to do so"),
				},
			},
			ExpectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"arn": cty.StringVal("arn:foo"),
			}),
			ExpectUpgradeResourceIdentityCalled: true,
			ExpectedError:                       fmt.Errorf("failed to upgrade resource identity: provider was unable to do so"),
		},
		"identity sent to provider differs from returned one": {
			// We don't throw an error here, because there are resource types with mutable identities
			StoredIdentitySchemaVersion: 0,
			StoredIdentityJSON:          []byte(`{"id": "foo"}`),
			IdentitySchema: providers.IdentitySchema{
				Version: 0,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
			ExpectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
		},
		"identity with unknowns": {
			IdentitySchema: providers.IdentitySchema{
				Version: 0,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}),
			ExpectedError: fmt.Errorf("Provider produced invalid identity: Provider \"registry.terraform.io/hashicorp/aws\" returned an identity with unknown values for aws_instance.web. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
		},

		"identity with marks": {
			IdentitySchema: providers.IdentitySchema{
				Version: 0,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("marked value").Mark(marks.Sensitive),
			}),
			ExpectedError: fmt.Errorf("Provider produced invalid identity: Provider \"registry.terraform.io/hashicorp/aws\" returned an identity with marks for aws_instance.web. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := testProvider("aws")
			m := testModule(t, "refresh-basic")
			p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
				ResourceTypes: map[string]*configschema.Block{
					"aws_instance": {
						Attributes: map[string]*configschema.Attribute{
							"id": {
								Type:     cty.String,
								Computed: true,
							},
							"foo": {
								Type:     cty.String,
								Optional: true,
								Computed: true,
							},
						},
					},
				},
				IdentityTypes: map[string]*configschema.Object{
					"aws_instance": tc.IdentitySchema.Body,
				},
				IdentityTypeSchemaVersions: map[string]uint64{
					"aws_instance": uint64(tc.IdentitySchema.Version),
				},
			})

			state := states.NewState()
			root := state.EnsureModule(addrs.RootModuleInstance)

			root.SetResourceInstanceCurrent(
				mustResourceInstanceAddr("aws_instance.web").Resource,
				&states.ResourceInstanceObjectSrc{
					Status:                states.ObjectReady,
					AttrsJSON:             []byte(`{"id":"foo","foo":"bar"}`),
					IdentitySchemaVersion: tc.StoredIdentitySchemaVersion,
					IdentityJSON:          tc.StoredIdentityJSON,
				},
				mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			)

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
				},
			})

			schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"]

			p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
				return providers.ReadResourceResponse{
					NewState: req.PriorState,
					Identity: tc.IdentityData,
				}
			}
			p.UpgradeResourceIdentityResponse = &tc.UpgradeResourceIdentityResponse

			s, diags := ctx.Plan(m, state, &PlanOpts{Mode: plans.RefreshOnlyMode})

			// TODO: maybe move to comparing diagnostics instead
			if tc.ExpectedError != nil {
				if !diags.HasErrors() {
					t.Fatal("expected error, got none")
				}
				if diags.Err().Error() != tc.ExpectedError.Error() {
					t.Fatalf("unexpected error\nwant: %v\ngot:  %v", tc.ExpectedError, diags.Err())
				}

				return
			} else {
				if diags.HasErrors() {
					t.Fatal(diags.Err())
				}
			}

			if !p.ReadResourceCalled {
				t.Fatal("ReadResource should be called")
			}

			if tc.ExpectUpgradeResourceIdentityCalled && !p.UpgradeResourceIdentityCalled {
				t.Fatal("UpgradeResourceIdentity should be called")
			}

			mod := s.PriorState.RootModule()
			fromState, err := mod.Resources["aws_instance.web"].Instances[addrs.NoKey].Current.Decode(schema)
			if err != nil {
				t.Fatal(err)
			}

			if tc.ExpectedIdentity.Equals(fromState.Identity).False() {
				t.Fatalf("wrong identity\nwant: %s\ngot: %s", tc.ExpectedIdentity.GoString(), fromState.Identity.GoString())
			}
		})
	}
}

// This test validates if a resource identity that is deposed and will be destroyed
// can be refreshed with an identity during the plan.
func TestContext2Plan_resource_identity_refresh_destroy_deposed(t *testing.T) {
	p := testProvider("aws")
	m := testModule(t, "refresh-basic")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&providerSchema{
		ResourceTypes: map[string]*configschema.Block{
			"aws_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
					"foo": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
				},
			},
		},
		IdentityTypes: map[string]*configschema.Object{
			"aws_instance": {
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
			"aws_instance": 0,
		},
	})

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)

	deposedKey := states.DeposedKey("00000001")
	root.SetResourceInstanceDeposed(
		mustResourceInstanceAddr("aws_instance.web").Resource,
		deposedKey,
		&states.ResourceInstanceObjectSrc{ // no identity recorded
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo","foo":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
		},
	})

	schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"]
	ty := schema.Body.ImpliedType()
	readState, err := hcl2shim.HCL2ValueFromFlatmap(map[string]string{"id": "foo", "foo": "baz"}, ty)
	if err != nil {
		t.Fatal(err)
	}

	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: readState,
		Identity: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("foo"),
		}),
	}

	s, diags := ctx.Plan(m, state, &PlanOpts{Mode: plans.RefreshOnlyMode})

	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}

	mod := s.PriorState.RootModule()
	fromState, err := mod.Resources["aws_instance.web"].Instances[addrs.NoKey].Deposed[deposedKey].Decode(schema)
	if err != nil {
		t.Fatal(err)
	}

	newState, err := schema.Body.CoerceValue(fromState.Value)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(readState, newState, valueComparer) {
		t.Fatal(cmp.Diff(readState, newState, valueComparer, equateEmpty))
	}
	expectedIdentity := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("foo"),
	})
	if expectedIdentity.Equals(fromState.Identity).False() {
		t.Fatalf("wrong identity\nwant: %s\ngot: %s", expectedIdentity.GoString(), fromState.Identity.GoString())
	}

}

func TestContext2Plan_resource_identity_plan(t *testing.T) {
	for name, tc := range map[string]struct {
		mode                  plans.Mode
		prevRunState          *states.State
		requiresReplace       []cty.Path
		identitySchemaVersion int64

		readResourceIdentity cty.Value
		upgradedIdentity     cty.Value

		plannedIdentity       cty.Value
		expectedIdentity      cty.Value
		expectedPriorIdentity cty.Value

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
		"create - invalid planned identity schema": {
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.BoolVal(false),
			}),
			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Error, "Provider produced an identity that doesn't match the schema", "Provider \"registry.terraform.io/hashicorp/test\" returned an identity for test_resource.test that doesn't match the identity schema: .id: string required, but received bool. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
			},
		},
		"create - null planned identity schema": {
			// We allow null values in identities
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.NullVal(cty.String),
			}),
			expectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.NullVal(cty.String),
			}),
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

		"update - changing identity": {
			// We don't throw an error here, because there are resource types with mutable identities
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
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),

			expectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
		},

		"update - updating identity schema version": {
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
						IdentityJSON:          []byte(`{"id":"foo"}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			}),
			upgradedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
			identitySchemaVersion: 1,
			expectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
		},

		"update - downgrading identity schema version": {
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
						IdentitySchemaVersion: 2,
						IdentityJSON:          []byte(`{"id":"foo"}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			}),
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"arn": cty.StringVal("arn:foo"),
			}),
			identitySchemaVersion: 1,
			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Error, "Resource instance managed by newer provider version", "The current state of test_resource.test was created by a newer provider version than is currently selected. Upgrade the test provider to work with this state."),
			},
		},

		"read and update": {
			prevRunState: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_resource",
						Name: "test",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"id":"bar"}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				)
			}),
			readResourceIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
			plannedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),

			expectedPriorIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
			expectedIdentity: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
			}),
		},
		"create with unknown identity": {
			plannedIdentity: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"id": cty.String,
			})),
			expectedIdentity: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"id": cty.String,
			})),
		},
		"update with unknown identity": {
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
			plannedIdentity: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"id": cty.String,
			})),

			expectDiagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Error, "Provider produced invalid identity", "Provider \"registry.terraform.io/hashicorp/test\" returned an identity with unknown values for test_resource.test. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
			},
		},
		"replace with unknown identity": {
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
			plannedIdentity: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"id": cty.String,
			})),

			expectedIdentity: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"id": cty.String,
			})),
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := testModuleInline(t, map[string]string{
				"main.tf": `
       resource "test_resource" "test" {
         id = "newValue"
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
					"test_resource": {
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
					"test_resource": uint64(tc.identitySchemaVersion),
				},
			})

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
				},
			})
			p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
				identity := req.CurrentIdentity
				if !tc.readResourceIdentity.IsNull() {
					identity = tc.readResourceIdentity
				}

				return providers.ReadResourceResponse{
					NewState: req.PriorState,
					Identity: identity,
				}
			}
			var plannedPriorIdentity cty.Value
			p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				plannedPriorIdentity = req.PriorIdentity
				return providers.PlanResourceChangeResponse{
					PlannedState:    req.ProposedNewState,
					PlannedIdentity: tc.plannedIdentity,
					RequiresReplace: tc.requiresReplace,
				}
			}

			p.UpgradeResourceIdentityFn = func(req providers.UpgradeResourceIdentityRequest) providers.UpgradeResourceIdentityResponse {

				return providers.UpgradeResourceIdentityResponse{
					UpgradedIdentity: tc.upgradedIdentity,
				}
			}

			plan, diags := ctx.Plan(m, tc.prevRunState, &PlanOpts{Mode: plans.NormalMode})

			if tc.expectDiagnostics != nil {
				tfdiags.AssertDiagnosticsMatch(t, diags, tc.expectDiagnostics)
			} else {
				tfdiags.AssertNoDiagnostics(t, diags)

				if !tc.expectedPriorIdentity.IsNull() {
					if !p.PlanResourceChangeCalled {
						t.Fatal("PlanResourceChangeFn was not called")
					}

					if !plannedPriorIdentity.RawEquals(tc.expectedPriorIdentity) {
						t.Fatalf("wrong prior identity\nwant: %s\ngot: %s", tc.expectedPriorIdentity.GoString(), plannedPriorIdentity.GoString())
					}
				}

				schema := p.GetProviderSchemaResponse.ResourceTypes["test_resource"]

				change, err := plan.Changes.Resources[0].Decode(schema)

				if err != nil {
					t.Fatal(err)
				}

				if !tc.expectedIdentity.RawEquals(change.AfterIdentity) {
					t.Fatalf("wrong identity\nwant: %s\ngot: %s", tc.expectedIdentity.GoString(), change.AfterIdentity.GoString())
				}
			}
		})
	}
}
