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
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TODO: Add tests for deposed resource instances
func TestContext2Plan_resource_identity_adds_missing(t *testing.T) {
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
				Attributes: configschema.IdentityAttributes{
					"id": {
						Type:              cty.String,
						RequiredForImport: true,
					},
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
				Attributes: configschema.IdentityAttributes{
					"id": {
						Type:              cty.String,
						RequiredForImport: true,
					},
				},
			},
			IdentityData: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("foo"),
			}),
			ExpectedError: fmt.Errorf("identity schema version mismatch: got 1, want 0"),
		},
		"identity type mismatch": {
			StoredIdentitySchemaVersion: 0,
			StoredIdentityJSON:          []byte(`{"arn": "foo"}`),
			IdentitySchema: providers.IdentitySchema{
				Version: 0,
				Attributes: configschema.IdentityAttributes{
					"id": {
						Type:              cty.String,
						RequiredForImport: true,
					},
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
				Attributes: configschema.IdentityAttributes{
					"id": {
						Type:              cty.String,
						RequiredForImport: true,
					},
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
				Attributes: configschema.IdentityAttributes{
					"arn": {
						Type:              cty.String,
						RequiredForImport: true,
					},
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
				Attributes: configschema.IdentityAttributes{
					"id": {
						Type:              cty.String,
						RequiredForImport: true,
					},
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
					IdentitySchemaJSON:    tc.StoredIdentityJSON,
				},
				mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			)

			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("aws"): testProviderFuncFixed(p),
				},
			})

			schema := p.GetProviderSchemaResponse.ResourceTypes["aws_instance"].Block
			ty := schema.ImpliedType()
			readState, err := hcl2shim.HCL2ValueFromFlatmap(map[string]string{"id": "foo", "foo": "baz"}, ty)
			if err != nil {
				t.Fatal(err)
			}

			p.GetResourceIdentitySchemasResponse = &providers.GetResourceIdentitySchemasResponse{
				IdentityTypes: map[string]providers.IdentitySchema{
					"aws_instance": tc.IdentitySchema,
				},
			}
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
			fromState, err := mod.Resources["aws_instance.web"].Instances[addrs.NoKey].Current.DecodeWithIdentity(ty, tc.IdentitySchema.Attributes.ImpliedType(), uint64(tc.IdentitySchema.Version))
			if err != nil {
				t.Fatal(err)
			}

			newState, err := schema.CoerceValue(fromState.Value)
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
