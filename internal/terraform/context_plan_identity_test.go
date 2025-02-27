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
			ExpectedError: fmt.Errorf("failed to decode identity schema: unsupported attribute \"arn\". This is most likely a bug in the Provider, providers must not change the identity schema without updating the identity schema version"),
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
				"id": cty.StringVal("foo"),
			}),
			ExpectedError: fmt.Errorf("Provider produced different identity: Provider \"registry.terraform.io/hashicorp/aws\" planned an different identity for aws_instance.web during refresh. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
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
			ExpectedError: fmt.Errorf("Provider produced invalid identity: Provider \"registry.terraform.io/hashicorp/aws\" planned an identity with unknown values for aws_instance.web during refresh. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
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
			ExpectedError: fmt.Errorf("Provider produced invalid identity: Provider \"registry.terraform.io/hashicorp/aws\" planned an identity with marks for aws_instance.web during refresh. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := testProvider("aws")
			m := testModule(t, "refresh-basic")

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
			ty := schema.Body.ImpliedType()
			readState, err := hcl2shim.HCL2ValueFromFlatmap(map[string]string{"id": "foo", "foo": "baz"}, ty)
			if err != nil {
				t.Fatal(err)
			}

			p.GetResourceIdentitySchemasResponse = &providers.GetResourceIdentitySchemasResponse{
				IdentityTypes: map[string]providers.IdentitySchema{
					"aws_instance": tc.IdentitySchema,
				},
			}
			schema.Identity = p.GetResourceIdentitySchemasResponse.IdentityTypes["aws_instance"].Body
			p.ReadResourceResponse = &providers.ReadResourceResponse{
				NewState: readState,
				Identity: tc.IdentityData,
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

			if !p.GetResourceIdentitySchemasCalled {
				t.Fatal("GetResourceIdentitySchemas should be called")
			}

			if tc.ExpectUpgradeResourceIdentityCalled && !p.UpgradeResourceIdentityCalled {
				t.Fatal("UpgradeResourceIdentity should be called")
			}

			mod := s.PriorState.RootModule()
			fromState, err := mod.Resources["aws_instance.web"].Instances[addrs.NoKey].Current.Decode(schema)
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

	p.GetResourceIdentitySchemasResponse = &providers.GetResourceIdentitySchemasResponse{
		IdentityTypes: map[string]providers.IdentitySchema{
			"aws_instance": {
				Version: 0,
				Body: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		},
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

	if !p.GetResourceIdentitySchemasCalled {
		t.Fatal("GetResourceIdentitySchemas should be called")
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

func TestContext2Plan_resource_identity_DEBUG(t *testing.T) {
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
			ExpectedError: fmt.Errorf("Provider produced invalid identity: Provider \"registry.terraform.io/hashicorp/aws\" planned an identity with unknown values for aws_instance.web during refresh. \n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker."),
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := testProvider("aws")
			m := testModule(t, "refresh-basic")

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
			ty := schema.Body.ImpliedType()
			readState, err := hcl2shim.HCL2ValueFromFlatmap(map[string]string{"id": "foo", "foo": "baz"}, ty)
			if err != nil {
				t.Fatal(err)
			}

			p.GetResourceIdentitySchemasResponse = &providers.GetResourceIdentitySchemasResponse{
				IdentityTypes: map[string]providers.IdentitySchema{
					"aws_instance": tc.IdentitySchema,
				},
			}
			schema.Identity = p.GetResourceIdentitySchemasResponse.IdentityTypes["aws_instance"].Body
			p.ReadResourceResponse = &providers.ReadResourceResponse{
				NewState: readState,
				Identity: tc.IdentityData,
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

			if !p.GetResourceIdentitySchemasCalled {
				t.Fatal("GetResourceIdentitySchemas should be called")
			}

			if tc.ExpectUpgradeResourceIdentityCalled && !p.UpgradeResourceIdentityCalled {
				t.Fatal("UpgradeResourceIdentity should be called")
			}

			mod := s.PriorState.RootModule()
			fromState, err := mod.Resources["aws_instance.web"].Instances[addrs.NoKey].Current.Decode(schema)
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

			if tc.ExpectedIdentity.Equals(fromState.Identity).False() {
				t.Fatalf("wrong identity\nwant: %s\ngot: %s", tc.ExpectedIdentity.GoString(), fromState.Identity.GoString())
			}
		})
	}
}
