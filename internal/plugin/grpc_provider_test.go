// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plugin/convert"
	mockproto "github.com/hashicorp/terraform/internal/plugin/mock_proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/tfdiags"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
)

var _ providers.Interface = (*GRPCProvider)(nil)

func mockProviderClient(t *testing.T) *mockproto.MockProviderClient {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	// we always need a GetSchema method
	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(providerProtoSchema(), nil)

	// GetResourceIdentitySchemas is called as part of GetSchema
	client.EXPECT().GetResourceIdentitySchemas(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(providerResourceIdentitySchemas(), nil)

	return client
}

func checkDiags(t *testing.T, d tfdiags.Diagnostics) {
	t.Helper()
	if d.HasErrors() {
		t.Fatal(d.Err())
	}
}

// checkDiagsHasError ensures error diagnostics are present or fails the test.
func checkDiagsHasError(t *testing.T, d tfdiags.Diagnostics) {
	t.Helper()

	if !d.HasErrors() {
		t.Fatal("expected error diagnostics")
	}
}

func providerProtoSchema() *proto.GetProviderSchema_Response {
	return &proto.GetProviderSchema_Response{
		Provider: &proto.Schema{
			Block: &proto.Schema_Block{
				Attributes: []*proto.Schema_Attribute{
					{
						Name:     "attr",
						Type:     []byte(`"string"`),
						Required: true,
					},
				},
			},
		},
		ResourceSchemas: map[string]*proto.Schema{
			"resource": &proto.Schema{
				Version: 1,
				Block: &proto.Schema_Block{
					Attributes: []*proto.Schema_Attribute{
						{
							Name:     "attr",
							Type:     []byte(`"string"`),
							Required: true,
						},
					},
				},
			},
			"list": {
				Version: 1,
				Block: &proto.Schema_Block{
					Attributes: []*proto.Schema_Attribute{
						{
							Name:     "resource_attr",
							Type:     []byte(`"string"`),
							Required: true,
						},
					},
				},
			},
		},
		DataSourceSchemas: map[string]*proto.Schema{
			"data": &proto.Schema{
				Version: 1,
				Block: &proto.Schema_Block{
					Attributes: []*proto.Schema_Attribute{
						{
							Name:     "attr",
							Type:     []byte(`"string"`),
							Required: true,
						},
					},
				},
			},
		},
		EphemeralResourceSchemas: map[string]*proto.Schema{
			"ephemeral": &proto.Schema{
				Block: &proto.Schema_Block{
					Attributes: []*proto.Schema_Attribute{
						{
							Name:     "attr",
							Type:     []byte(`"string"`),
							Computed: true,
						},
					},
				},
			},
		},
		ListResourceSchemas: map[string]*proto.Schema{
			"list": &proto.Schema{
				Version: 1,
				Block: &proto.Schema_Block{
					Attributes: []*proto.Schema_Attribute{
						{
							Name:     "filter_attr",
							Type:     []byte(`"string"`),
							Required: true,
						},
					},
					BlockTypes: []*proto.Schema_NestedBlock{
						{
							TypeName: "nested_filter",
							Nesting:  proto.Schema_NestedBlock_SINGLE,
							Block: &proto.Schema_Block{
								Attributes: []*proto.Schema_Attribute{
									{
										Name:     "nested_attr",
										Type:     []byte(`"string"`),
										Required: false,
									},
								},
							},
							MinItems: 1,
							MaxItems: 1,
						},
					},
				},
			},
		},
		ActionSchemas: map[string]*proto.ActionSchema{
			"action": {
				Schema: &proto.Schema{
					Block: &proto.Schema_Block{
						Version: 1,
						Attributes: []*proto.Schema_Attribute{
							{
								Name: "attr",
								Type: []byte(`"string"`),
							},
						},
					},
				},
			},
		},
		ServerCapabilities: &proto.ServerCapabilities{
			GetProviderSchemaOptional: true,
		},
	}
}

func providerResourceIdentitySchemas() *proto.GetResourceIdentitySchemas_Response {
	return &proto.GetResourceIdentitySchemas_Response{
		IdentitySchemas: map[string]*proto.ResourceIdentitySchema{
			"resource": {
				Version: 1,
				IdentityAttributes: []*proto.ResourceIdentitySchema_IdentityAttribute{
					{
						Name:              "id_attr",
						Type:              []byte(`"string"`),
						RequiredForImport: true,
					},
				},
			},
			"list": {
				Version: 1,
				IdentityAttributes: []*proto.ResourceIdentitySchema_IdentityAttribute{
					{
						Name:              "id_attr",
						Type:              []byte(`"string"`),
						RequiredForImport: true,
					},
				},
			},
		},
	}
}

func TestGRPCProvider_GetSchema(t *testing.T) {
	p := &GRPCProvider{
		client: mockProviderClient(t),
	}

	resp := p.GetProviderSchema()
	checkDiags(t, resp.Diagnostics)
}

// ensure that the global schema cache is used when the provider supports
// GetProviderSchemaOptional
func TestGRPCProvider_GetSchema_globalCache(t *testing.T) {
	p := &GRPCProvider{
		Addr:   addrs.ImpliedProviderForUnqualifiedType("test"),
		client: mockProviderClient(t),
	}

	// first call primes the cache
	resp := p.GetProviderSchema()

	// create a new provider instance which does not expect a GetProviderSchemaCall
	p = &GRPCProvider{
		Addr:   addrs.ImpliedProviderForUnqualifiedType("test"),
		client: mockproto.NewMockProviderClient(gomock.NewController(t)),
	}

	resp = p.GetProviderSchema()
	checkDiags(t, resp.Diagnostics)
}

// Ensure that gRPC errors are returned early.
// Reference: https://github.com/hashicorp/terraform/issues/31047
func TestGRPCProvider_GetSchema_GRPCError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.GetProviderSchema_Response{}, fmt.Errorf("test error"))

	p := &GRPCProvider{
		client: client,
	}

	resp := p.GetProviderSchema()

	checkDiagsHasError(t, resp.Diagnostics)
}

// Ensure that provider error diagnostics are returned early.
// Reference: https://github.com/hashicorp/terraform/issues/31047
func TestGRPCProvider_GetSchema_ResponseErrorDiagnostic(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.GetProviderSchema_Response{
		Diagnostics: []*proto.Diagnostic{
			{
				Severity: proto.Diagnostic_ERROR,
				Summary:  "error summary",
				Detail:   "error detail",
			},
		},
		// Trigger potential panics
		Provider: &proto.Schema{},
	}, nil)

	p := &GRPCProvider{
		client: client,
	}

	resp := p.GetProviderSchema()

	checkDiagsHasError(t, resp.Diagnostics)
}

func TestGRPCProvider_GetSchema_IdentityError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(providerProtoSchema(), nil)

	client.EXPECT().GetResourceIdentitySchemas(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.GetResourceIdentitySchemas_Response{}, fmt.Errorf("test error"))

	p := &GRPCProvider{
		client: client,
	}

	resp := p.GetProviderSchema()

	checkDiagsHasError(t, resp.Diagnostics)
}

func TestGRPCProvider_GetSchema_IdentityUnimplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(providerProtoSchema(), nil)

	client.EXPECT().GetResourceIdentitySchemas(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.GetResourceIdentitySchemas_Response{}, status.Error(codes.Unimplemented, "test error"))

	p := &GRPCProvider{
		client: client,
	}

	resp := p.GetProviderSchema()

	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_GetSchema_IdentityErrorDiagnostic(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(providerProtoSchema(), nil)

	client.EXPECT().GetResourceIdentitySchemas(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.GetResourceIdentitySchemas_Response{
		Diagnostics: []*proto.Diagnostic{
			{
				Severity: proto.Diagnostic_ERROR,
				Summary:  "error summary",
				Detail:   "error detail",
			},
		},
		IdentitySchemas: map[string]*proto.ResourceIdentitySchema{},
	}, nil)

	p := &GRPCProvider{
		client: client,
	}

	resp := p.GetProviderSchema()

	checkDiagsHasError(t, resp.Diagnostics)
}

func TestGRPCProvider_GetResourceIdentitySchemas(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	client.EXPECT().GetResourceIdentitySchemas(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(providerResourceIdentitySchemas(), nil)

	p := &GRPCProvider{
		client: client,
	}

	resp := p.GetResourceIdentitySchemas()

	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_GetResourceIdentitySchemas_Unimplemented(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	client.EXPECT().GetResourceIdentitySchemas(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.GetResourceIdentitySchemas_Response{}, status.Error(codes.Unimplemented, "test error"))

	p := &GRPCProvider{
		client: client,
	}

	resp := p.GetResourceIdentitySchemas()

	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_PrepareProviderConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().PrepareProviderConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.PrepareProviderConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"attr": "value"})
	resp := p.ValidateProviderConfig(providers.ValidateProviderConfigRequest{Config: cfg})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_ValidateResourceConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ValidateResourceTypeConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateResourceTypeConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"attr": "value"})
	resp := p.ValidateResourceConfig(providers.ValidateResourceConfigRequest{
		TypeName: "resource",
		Config:   cfg,
	})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_ValidateDataSourceConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ValidateDataSourceConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateDataSourceConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"attr": "value"})
	resp := p.ValidateDataResourceConfig(providers.ValidateDataResourceConfigRequest{
		TypeName: "data",
		Config:   cfg,
	})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_ValidateListResourceConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ValidateListResourceConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateListResourceConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"config": map[string]interface{}{"filter_attr": "value", "nested_filter": map[string]interface{}{"nested_attr": "value"}}})
	resp := p.ValidateListResourceConfig(providers.ValidateListResourceConfigRequest{
		TypeName: "list",
		Config:   cfg,
	})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_ValidateListResourceConfig_OptionalCfg(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)
	sch := providerProtoSchema()

	// mock the schema in a way that makes the config attributes optional
	listSchema := sch.ListResourceSchemas["list"].Block
	// filter_attr is optional
	listSchema.Attributes[0].Optional = true
	listSchema.Attributes[0].Required = false

	// nested_filter is optional
	listSchema.BlockTypes[0].MinItems = 0
	listSchema.BlockTypes[0].MaxItems = 0

	sch.ListResourceSchemas["list"].Block = listSchema
	// we always need a GetSchema method
	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(sch, nil)

	// GetResourceIdentitySchemas is called as part of GetSchema
	client.EXPECT().GetResourceIdentitySchemas(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(providerResourceIdentitySchemas(), nil)

	p := &GRPCProvider{
		client: client,
	}
	client.EXPECT().ValidateListResourceConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateListResourceConfig_Response{}, nil)

	converted := convert.ProtoToListSchema(sch.ListResourceSchemas["list"])
	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]any{})
	coercedCfg, err := converted.Body.CoerceValue(cfg)
	if err != nil {
		t.Fatalf("failed to coerce config: %v", err)
	}
	resp := p.ValidateListResourceConfig(providers.ValidateListResourceConfigRequest{
		TypeName: "list",
		Config:   coercedCfg,
	})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_UpgradeResourceState(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().UpgradeResourceState(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.UpgradeResourceState_Response{
		UpgradedState: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
	}, nil)

	resp := p.UpgradeResourceState(providers.UpgradeResourceStateRequest{
		TypeName:     "resource",
		Version:      0,
		RawStateJSON: []byte(`{"old_attr":"bar"}`),
	})
	checkDiags(t, resp.Diagnostics)

	expected := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expected, resp.UpgradedState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, resp.UpgradedState, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_UpgradeResourceStateJSON(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().UpgradeResourceState(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.UpgradeResourceState_Response{
		UpgradedState: &proto.DynamicValue{
			Json: []byte(`{"attr":"bar"}`),
		},
	}, nil)

	resp := p.UpgradeResourceState(providers.UpgradeResourceStateRequest{
		TypeName:     "resource",
		Version:      0,
		RawStateJSON: []byte(`{"old_attr":"bar"}`),
	})
	checkDiags(t, resp.Diagnostics)

	expected := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expected, resp.UpgradedState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, resp.UpgradedState, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_UpgradeResourceIdentity(t *testing.T) {
	testCases := []struct {
		desc          string
		response      *proto.UpgradeResourceIdentity_Response
		expectError   bool
		expectedValue cty.Value
	}{
		{
			"successful upgrade",
			&proto.UpgradeResourceIdentity_Response{
				UpgradedIdentity: &proto.ResourceIdentityData{
					IdentityData: &proto.DynamicValue{
						Json: []byte(`{"id_attr":"bar"}`),
					},
				},
			},
			false,
			cty.ObjectVal(map[string]cty.Value{"id_attr": cty.StringVal("bar")}),
		},
		{
			"response with error diagnostic",
			&proto.UpgradeResourceIdentity_Response{
				Diagnostics: []*proto.Diagnostic{
					{
						Severity: proto.Diagnostic_ERROR,
						Summary:  "test error",
						Detail:   "test error detail",
					},
				},
			},
			true,
			cty.NilVal,
		},
		{
			"schema mismatch",
			&proto.UpgradeResourceIdentity_Response{
				UpgradedIdentity: &proto.ResourceIdentityData{
					IdentityData: &proto.DynamicValue{
						Json: []byte(`{"attr_new":"bar"}`),
					},
				},
			},
			true,
			cty.NilVal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			client := mockProviderClient(t)
			p := &GRPCProvider{
				client: client,
			}

			client.EXPECT().UpgradeResourceIdentity(
				gomock.Any(),
				gomock.Any(),
			).Return(tc.response, nil)

			resp := p.UpgradeResourceIdentity(providers.UpgradeResourceIdentityRequest{
				TypeName:        "resource",
				Version:         0,
				RawIdentityJSON: []byte(`{"old_attr":"bar"}`),
			})

			if tc.expectError {
				checkDiagsHasError(t, resp.Diagnostics)
			} else {
				checkDiags(t, resp.Diagnostics)

				if !cmp.Equal(tc.expectedValue, resp.UpgradedIdentity, typeComparer, valueComparer, equateEmpty) {
					t.Fatal(cmp.Diff(tc.expectedValue, resp.UpgradedIdentity, typeComparer, valueComparer, equateEmpty))
				}
			}
		})
	}
}

func TestGRPCProvider_Configure(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().Configure(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.Configure_Response{}, nil)

	resp := p.ConfigureProvider(providers.ConfigureProviderRequest{
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().Stop(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.Stop_Response{}, nil)

	err := p.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGRPCProvider_ReadResource(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ReadResource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ReadResource_Response{
		NewState: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
	}, nil)

	resp := p.ReadResource(providers.ReadResourceRequest{
		TypeName: "resource",
		PriorState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expected := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expected, resp.NewState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, resp.NewState, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_ReadResource_deferred(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ReadResource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ReadResource_Response{
		NewState: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
		Deferred: &proto.Deferred{
			Reason: proto.Deferred_ABSENT_PREREQ,
		},
	}, nil)

	resp := p.ReadResource(providers.ReadResourceRequest{
		TypeName: "resource",
		PriorState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expectedDeferred := &providers.Deferred{
		Reason: providers.DeferredReasonAbsentPrereq,
	}
	if !cmp.Equal(expectedDeferred, resp.Deferred, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedDeferred, resp.Deferred, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_ReadResourceJSON(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ReadResource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ReadResource_Response{
		NewState: &proto.DynamicValue{
			Json: []byte(`{"attr":"bar"}`),
		},
	}, nil)

	resp := p.ReadResource(providers.ReadResourceRequest{
		TypeName: "resource",
		PriorState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expected := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expected, resp.NewState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, resp.NewState, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_ReadEmptyJSON(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ReadResource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ReadResource_Response{
		NewState: &proto.DynamicValue{
			Json: []byte(``),
		},
	}, nil)

	obj := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("foo"),
	})
	resp := p.ReadResource(providers.ReadResourceRequest{
		TypeName:   "resource",
		PriorState: obj,
	})

	checkDiags(t, resp.Diagnostics)

	expected := cty.NullVal(obj.Type())

	if !cmp.Equal(expected, resp.NewState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, resp.NewState, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_PlanResourceChange(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	expectedPrivate := []byte(`{"meta": "data"}`)

	client.EXPECT().PlanResourceChange(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.PlanResourceChange_Response{
		PlannedState: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
		RequiresReplace: []*proto.AttributePath{
			{
				Steps: []*proto.AttributePath_Step{
					{
						Selector: &proto.AttributePath_Step_AttributeName{
							AttributeName: "attr",
						},
					},
				},
			},
		},
		PlannedPrivate: expectedPrivate,
	}, nil)

	resp := p.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName: "resource",
		PriorState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
		ProposedNewState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expectedState := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expectedState, resp.PlannedState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedState, resp.PlannedState, typeComparer, valueComparer, equateEmpty))
	}

	expectedReplace := `[]cty.Path{cty.Path{cty.GetAttrStep{Name:"attr"}}}`
	replace := fmt.Sprintf("%#v", resp.RequiresReplace)
	if expectedReplace != replace {
		t.Fatalf("expected %q, got %q", expectedReplace, replace)
	}

	if !bytes.Equal(expectedPrivate, resp.PlannedPrivate) {
		t.Fatalf("expected %q, got %q", expectedPrivate, resp.PlannedPrivate)
	}
}

func TestGRPCProvider_PlanResourceChangeJSON(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	expectedPrivate := []byte(`{"meta": "data"}`)

	client.EXPECT().PlanResourceChange(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.PlanResourceChange_Response{
		PlannedState: &proto.DynamicValue{
			Json: []byte(`{"attr":"bar"}`),
		},
		RequiresReplace: []*proto.AttributePath{
			{
				Steps: []*proto.AttributePath_Step{
					{
						Selector: &proto.AttributePath_Step_AttributeName{
							AttributeName: "attr",
						},
					},
				},
			},
		},
		PlannedPrivate: expectedPrivate,
	}, nil)

	resp := p.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName: "resource",
		PriorState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
		ProposedNewState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expectedState := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expectedState, resp.PlannedState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedState, resp.PlannedState, typeComparer, valueComparer, equateEmpty))
	}

	expectedReplace := `[]cty.Path{cty.Path{cty.GetAttrStep{Name:"attr"}}}`
	replace := fmt.Sprintf("%#v", resp.RequiresReplace)
	if expectedReplace != replace {
		t.Fatalf("expected %q, got %q", expectedReplace, replace)
	}

	if !bytes.Equal(expectedPrivate, resp.PlannedPrivate) {
		t.Fatalf("expected %q, got %q", expectedPrivate, resp.PlannedPrivate)
	}
}

func TestGRPCProvider_ApplyResourceChange(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	expectedPrivate := []byte(`{"meta": "data"}`)

	client.EXPECT().ApplyResourceChange(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ApplyResourceChange_Response{
		NewState: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
		Private: expectedPrivate,
	}, nil)

	resp := p.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName: "resource",
		PriorState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		PlannedPrivate: expectedPrivate,
	})

	checkDiags(t, resp.Diagnostics)

	expectedState := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expectedState, resp.NewState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedState, resp.NewState, typeComparer, valueComparer, equateEmpty))
	}

	if !bytes.Equal(expectedPrivate, resp.Private) {
		t.Fatalf("expected %q, got %q", expectedPrivate, resp.Private)
	}
}
func TestGRPCProvider_ApplyResourceChangeJSON(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	expectedPrivate := []byte(`{"meta": "data"}`)

	client.EXPECT().ApplyResourceChange(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ApplyResourceChange_Response{
		NewState: &proto.DynamicValue{
			Json: []byte(`{"attr":"bar"}`),
		},
		Private: expectedPrivate,
	}, nil)

	resp := p.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName: "resource",
		PriorState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
		PlannedState: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		PlannedPrivate: expectedPrivate,
	})

	checkDiags(t, resp.Diagnostics)

	expectedState := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expectedState, resp.NewState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedState, resp.NewState, typeComparer, valueComparer, equateEmpty))
	}

	if !bytes.Equal(expectedPrivate, resp.Private) {
		t.Fatalf("expected %q, got %q", expectedPrivate, resp.Private)
	}
}

func TestGRPCProvider_ImportResourceState(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	expectedPrivate := []byte(`{"meta": "data"}`)

	client.EXPECT().ImportResourceState(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ImportResourceState_Response{
		ImportedResources: []*proto.ImportResourceState_ImportedResource{
			{
				TypeName: "resource",
				State: &proto.DynamicValue{
					Msgpack: []byte("\x81\xa4attr\xa3bar"),
				},
				Private: expectedPrivate,
			},
		},
	}, nil)

	resp := p.ImportResourceState(providers.ImportResourceStateRequest{
		TypeName: "resource",
		ID:       "foo",
	})

	checkDiags(t, resp.Diagnostics)

	expectedResource := providers.ImportedResource{
		TypeName: "resource",
		State: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		Private: expectedPrivate,
	}

	imported := resp.ImportedResources[0]
	if !cmp.Equal(expectedResource, imported, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedResource, imported, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_ImportResourceStateJSON(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	expectedPrivate := []byte(`{"meta": "data"}`)

	client.EXPECT().ImportResourceState(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ImportResourceState_Response{
		ImportedResources: []*proto.ImportResourceState_ImportedResource{
			{
				TypeName: "resource",
				State: &proto.DynamicValue{
					Json: []byte(`{"attr":"bar"}`),
				},
				Private: expectedPrivate,
			},
		},
	}, nil)

	resp := p.ImportResourceState(providers.ImportResourceStateRequest{
		TypeName: "resource",
		ID:       "foo",
	})

	checkDiags(t, resp.Diagnostics)

	expectedResource := providers.ImportedResource{
		TypeName: "resource",
		State: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		Private: expectedPrivate,
	}

	imported := resp.ImportedResources[0]
	if !cmp.Equal(expectedResource, imported, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedResource, imported, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_ImportResourceState_Identity(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ImportResourceState(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ImportResourceState_Response{
		ImportedResources: []*proto.ImportResourceState_ImportedResource{
			{
				TypeName: "resource",
				State: &proto.DynamicValue{
					Msgpack: []byte("\x81\xa4attr\xa3bar"),
				},
				Identity: &proto.ResourceIdentityData{
					IdentityData: &proto.DynamicValue{
						Msgpack: []byte("\x81\xa7id_attr\xa3foo"),
					},
				},
			},
		},
	}, nil)

	resp := p.ImportResourceState(providers.ImportResourceStateRequest{
		TypeName: "resource",
		Identity: cty.ObjectVal(map[string]cty.Value{
			"id_attr": cty.StringVal("foo"),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expectedResource := providers.ImportedResource{
		TypeName: "resource",
		State: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("bar"),
		}),
		Identity: cty.ObjectVal(map[string]cty.Value{
			"id_attr": cty.StringVal("foo"),
		}),
	}

	imported := resp.ImportedResources[0]
	if !cmp.Equal(expectedResource, imported, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedResource, imported, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_MoveResourceState(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	expectedTargetPrivate := []byte(`{"target": "private"}`)
	expectedTargetState := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	client.EXPECT().MoveResourceState(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.MoveResourceState_Response{
		TargetState: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
		TargetPrivate: expectedTargetPrivate,
	}, nil)

	resp := p.MoveResourceState(providers.MoveResourceStateRequest{
		SourcePrivate:   []byte(`{"source": "private"}`),
		SourceStateJSON: []byte(`{"source_attr":"bar"}`),
		TargetTypeName:  "resource",
	})

	checkDiags(t, resp.Diagnostics)

	if !cmp.Equal(expectedTargetPrivate, resp.TargetPrivate, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedTargetPrivate, resp.TargetPrivate, typeComparer, valueComparer, equateEmpty))
	}

	if !cmp.Equal(expectedTargetState, resp.TargetState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedTargetState, resp.TargetState, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_MoveResourceStateJSON(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	expectedTargetPrivate := []byte(`{"target": "private"}`)
	expectedTargetState := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	client.EXPECT().MoveResourceState(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.MoveResourceState_Response{
		TargetState: &proto.DynamicValue{
			Json: []byte(`{"attr":"bar"}`),
		},
		TargetPrivate: expectedTargetPrivate,
	}, nil)

	resp := p.MoveResourceState(providers.MoveResourceStateRequest{
		SourcePrivate:   []byte(`{"source": "private"}`),
		SourceStateJSON: []byte(`{"source_attr":"bar"}`),
		TargetTypeName:  "resource",
	})

	checkDiags(t, resp.Diagnostics)

	if !cmp.Equal(expectedTargetPrivate, resp.TargetPrivate, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedTargetPrivate, resp.TargetPrivate, typeComparer, valueComparer, equateEmpty))
	}

	if !cmp.Equal(expectedTargetState, resp.TargetState, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expectedTargetState, resp.TargetState, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_ReadDataSource(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ReadDataSource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ReadDataSource_Response{
		State: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
	}, nil)

	resp := p.ReadDataSource(providers.ReadDataSourceRequest{
		TypeName: "data",
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expected := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expected, resp.State, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, resp.State, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_ReadDataSourceJSON(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ReadDataSource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ReadDataSource_Response{
		State: &proto.DynamicValue{
			Json: []byte(`{"attr":"bar"}`),
		},
	}, nil)

	resp := p.ReadDataSource(providers.ReadDataSourceRequest{
		TypeName: "data",
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expected := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expected, resp.State, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, resp.State, typeComparer, valueComparer, equateEmpty))
	}
}

func TestGRPCProvider_openEphemeralResource(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().OpenEphemeralResource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.OpenEphemeralResource_Response{
		Result: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
		RenewAt: timestamppb.New(time.Now().Add(time.Second)),
		Private: []byte("private data"),
	}, nil)

	resp := p.OpenEphemeralResource(providers.OpenEphemeralResourceRequest{
		TypeName: "ephemeral",
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.NullVal(cty.String),
		}),
	})

	checkDiags(t, resp.Diagnostics)

	expected := cty.ObjectVal(map[string]cty.Value{
		"attr": cty.StringVal("bar"),
	})

	if !cmp.Equal(expected, resp.Result, typeComparer, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, resp.Result, typeComparer, valueComparer, equateEmpty))
	}

	if !resp.RenewAt.After(time.Now()) {
		t.Fatal("invalid RenewAt:", resp.RenewAt)
	}

	if !bytes.Equal(resp.Private, []byte("private data")) {
		t.Fatalf("invalid private data: %q", resp.Private)
	}
}

func TestGRPCProvider_renewEphemeralResource(t *testing.T) {
	client := mockproto.NewMockProviderClient(gomock.NewController(t))
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().RenewEphemeralResource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.RenewEphemeralResource_Response{
		RenewAt: timestamppb.New(time.Now().Add(time.Second)),
		Private: []byte("private data"),
	}, nil)

	resp := p.RenewEphemeralResource(providers.RenewEphemeralResourceRequest{
		TypeName: "ephemeral",
		Private:  []byte("private data"),
	})

	checkDiags(t, resp.Diagnostics)

	if !resp.RenewAt.After(time.Now()) {
		t.Fatal("invalid RenewAt:", resp.RenewAt)
	}

	if !bytes.Equal(resp.Private, []byte("private data")) {
		t.Fatalf("invalid private data: %q", resp.Private)
	}
}

func TestGRPCProvider_closeEphemeralResource(t *testing.T) {
	client := mockproto.NewMockProviderClient(gomock.NewController(t))
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().CloseEphemeralResource(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.CloseEphemeralResource_Response{}, nil)

	resp := p.CloseEphemeralResource(providers.CloseEphemeralResourceRequest{
		TypeName: "ephemeral",
		Private:  []byte("private data"),
	})

	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_GetSchema_ListResourceTypes(t *testing.T) {
	p := &GRPCProvider{
		client: mockProviderClient(t),
		ctx:    context.Background(),
	}

	resp := p.GetProviderSchema()
	listResourceSchema := resp.ListResourceTypes
	expected := map[string]providers.Schema{
		"list": {
			Version: 1,
			Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"data": {
						Type:     cty.DynamicPseudoType,
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"config": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"filter_attr": {
									Type:     cty.String,
									Required: true,
								},
							},
							BlockTypes: map[string]*configschema.NestedBlock{
								"nested_filter": {
									Block: configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"nested_attr": {
												Type:     cty.String,
												Required: false,
											},
										},
									},
									Nesting:  configschema.NestingSingle,
									MinItems: 1,
									MaxItems: 1,
								},
							},
						},
						Nesting:  configschema.NestingSingle,
						MinItems: 1,
						MaxItems: 1,
					},
				},
			},
		},
	}
	checkDiags(t, resp.Diagnostics)

	actualBody := convert.ConfigSchemaToProto(listResourceSchema["list"].Body).String()
	expectedBody := convert.ConfigSchemaToProto(expected["list"].Body).String()
	if actualBody != expectedBody {
		t.Fatalf("expected %v, got %v", expectedBody, actualBody)
	}
}

func TestGRPCProvider_Encode(t *testing.T) {
	// TODO: This is the only test in this package that imports plans. If that
	// ever leads to a circular import, we should consider moving this test to
	// a different package or refactoring the test to not use plans.
	p := &GRPCProvider{
		client: mockProviderClient(t),
		ctx:    context.Background(),
		Addr:   addrs.ImpliedProviderForUnqualifiedType("testencode"),
	}
	resp := p.GetProviderSchema()

	src := plans.NewChanges()
	src.SyncWrapper().AppendResourceInstanceChange(&plans.ResourceInstanceChange{
		Addr: addrs.AbsResourceInstance{
			Module: addrs.RootModuleInstance,
			Resource: addrs.ResourceInstance{
				Resource: addrs.Resource{
					Mode: addrs.ListResourceMode,
					Type: "list",
					Name: "test",
				},
				Key: addrs.NoKey,
			},
		},
		ProviderAddr: addrs.AbsProviderConfig{
			Provider: p.Addr,
		},
		Change: plans.Change{
			Before: cty.NullVal(cty.Object(map[string]cty.Type{
				"config": cty.Object(map[string]cty.Type{
					"filter_attr": cty.String,
					"nested_filter": cty.Object(map[string]cty.Type{
						"nested_attr": cty.String,
					}),
				}),
				"data": cty.List(cty.Object(map[string]cty.Type{
					"state": cty.Object(map[string]cty.Type{
						"resource_attr": cty.String,
					}),
					"identity": cty.Object(map[string]cty.Type{
						"id_attr": cty.String,
					}),
				})),
			})),
			After: cty.ObjectVal(map[string]cty.Value{
				"config": cty.ObjectVal(map[string]cty.Value{
					"filter_attr": cty.StringVal("value"),
					"nested_filter": cty.ObjectVal(map[string]cty.Value{
						"nested_attr": cty.StringVal("value"),
					}),
				}),
				"data": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"state": cty.ObjectVal(map[string]cty.Value{
							"resource_attr": cty.StringVal("value"),
						}),
						"identity": cty.ObjectVal(map[string]cty.Value{
							"id_attr": cty.StringVal("value"),
						}),
					}),
				}),
			}),
		},
	})
	_, err := src.Encode(&schemarepo.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			p.Addr: {
				ResourceTypes:     resp.ResourceTypes,
				ListResourceTypes: resp.ListResourceTypes,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error encoding changes: %s", err)
	}
}

func TestGRPCProvider_planAction_valid(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().PlanAction(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.PlanAction_Response{}, nil)

	resp := p.PlanAction(providers.PlanActionRequest{
		ActionType: "action",
		ProposedActionData: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})

	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_planAction_valid_but_fails(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().PlanAction(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.PlanAction_Response{
		Diagnostics: []*proto.Diagnostic{
			{
				Severity: proto.Diagnostic_ERROR,
				Summary:  "Boom",
				Detail:   "Explosion",
			},
		},
	}, nil)

	resp := p.PlanAction(providers.PlanActionRequest{
		ActionType: "action",
		ProposedActionData: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})

	checkDiagsHasError(t, resp.Diagnostics)
}

func TestGRPCProvider_planAction_invalid_config(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	resp := p.PlanAction(providers.PlanActionRequest{
		ActionType: "action",
		ProposedActionData: cty.ObjectVal(map[string]cty.Value{
			"not_the_right_attr": cty.StringVal("foo"),
		}),
	})

	checkDiagsHasError(t, resp.Diagnostics)
}

// Mock implementation of the ListResource stream client
type mockListResourceStreamClient struct {
	events  []*proto.ListResource_Event
	current int
	proto.Provider_ListResourceClient
}

func (m *mockListResourceStreamClient) Recv() (*proto.ListResource_Event, error) {
	if m.current >= len(m.events) {
		return nil, io.EOF
	}

	event := m.events[m.current]
	m.current++
	return event, nil
}

func TestGRPCProvider_ListResource(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
		ctx:    context.Background(),
	}

	// Create a mock stream client that will return resource events
	mockStream := &mockListResourceStreamClient{
		events: []*proto.ListResource_Event{
			{
				DisplayName: "Test Resource 1",
				Identity: &proto.ResourceIdentityData{
					IdentityData: &proto.DynamicValue{
						Msgpack: []byte("\x81\xa7id_attr\xa4id-1"),
					},
				},
			},
			{
				DisplayName: "Test Resource 2",
				Identity: &proto.ResourceIdentityData{
					IdentityData: &proto.DynamicValue{
						Msgpack: []byte("\x81\xa7id_attr\xa4id-2"),
					},
				},
				ResourceObject: &proto.DynamicValue{
					Msgpack: []byte("\x81\xadresource_attr\xa5value"),
				},
			},
		},
	}

	client.EXPECT().ListResource(
		gomock.Any(),
		gomock.Any(),
	).Return(mockStream, nil)

	// Create the request
	configVal := cty.ObjectVal(map[string]cty.Value{
		"config": cty.ObjectVal(map[string]cty.Value{
			"filter_attr": cty.StringVal("filter-value"),
			"nested_filter": cty.ObjectVal(map[string]cty.Value{
				"nested_attr": cty.StringVal("value"),
			}),
		}),
	})
	request := providers.ListResourceRequest{
		TypeName:              "list",
		Config:                configVal,
		IncludeResourceObject: true,
		Limit:                 100,
	}

	resp := p.ListResource(request)
	checkDiags(t, resp.Diagnostics)

	data := resp.Result.AsValueMap()
	if _, ok := data["data"]; !ok {
		t.Fatal("Expected 'data' key in result")
	}
	// Verify that we received both events
	if len(data["data"].AsValueSlice()) != 2 {
		t.Fatalf("Expected 2 resources, got %d", len(data["data"].AsValueSlice()))
	}
	results := data["data"].AsValueSlice()

	// Verify first event
	displayName := results[0].GetAttr("display_name")
	if displayName.AsString() != "Test Resource 1" {
		t.Errorf("Expected DisplayName 'Test Resource 1', got '%s'", displayName.AsString())
	}

	expectedId1 := cty.ObjectVal(map[string]cty.Value{
		"id_attr": cty.StringVal("id-1"),
	})

	identity := results[0].GetAttr("identity")
	if !identity.RawEquals(expectedId1) {
		t.Errorf("Expected Identity %#v, got %#v", expectedId1, identity)
	}

	// ResourceObject should be null for the first event as it wasn't provided
	resourceObject := results[0].GetAttr("state")
	if !resourceObject.IsNull() {
		t.Errorf("Expected ResourceObject to be null, got %#v", resourceObject)
	}

	// Verify second event
	displayName = results[1].GetAttr("display_name")
	if displayName.AsString() != "Test Resource 2" {
		t.Errorf("Expected DisplayName 'Test Resource 2', got '%s'", displayName.AsString())
	}

	expectedId2 := cty.ObjectVal(map[string]cty.Value{
		"id_attr": cty.StringVal("id-2"),
	})
	identity = results[1].GetAttr("identity")
	if !identity.RawEquals(expectedId2) {
		t.Errorf("Expected Identity %#v, got %#v", expectedId2, identity)
	}

	expectedResource := cty.ObjectVal(map[string]cty.Value{
		"resource_attr": cty.StringVal("value"),
	})
	resourceObject = results[1].GetAttr("state")
	if !resourceObject.RawEquals(expectedResource) {
		t.Errorf("Expected ResourceObject %#v, got %#v", expectedResource, resourceObject)
	}
}

func TestGRPCProvider_ListResource_Error(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
		ctx:    context.Background(),
	}

	// Test case where the provider returns an error
	client.EXPECT().ListResource(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, fmt.Errorf("provider error"))

	configVal := cty.ObjectVal(map[string]cty.Value{
		"config": cty.ObjectVal(map[string]cty.Value{
			"filter_attr": cty.StringVal("filter-value"),
			"nested_filter": cty.ObjectVal(map[string]cty.Value{
				"nested_attr": cty.StringVal("value"),
			}),
		}),
	})
	request := providers.ListResourceRequest{
		TypeName: "list",
		Config:   configVal,
	}

	resp := p.ListResource(request)
	checkDiagsHasError(t, resp.Diagnostics)
}

func TestGRPCProvider_ListResource_Diagnostics(t *testing.T) {
	configVal := cty.ObjectVal(map[string]cty.Value{
		"config": cty.ObjectVal(map[string]cty.Value{
			"filter_attr": cty.StringVal("filter-value"),
			"nested_filter": cty.ObjectVal(map[string]cty.Value{
				"nested_attr": cty.StringVal("value"),
			}),
		}),
	})
	request := providers.ListResourceRequest{
		TypeName: "list",
		Config:   configVal,
		Limit:    100,
	}

	testCases := []struct {
		name          string
		events        []*proto.ListResource_Event
		expectedCount int
		expectedDiags int
		expectedWarns int // subset of expectedDiags
	}{
		{
			"no events",
			[]*proto.ListResource_Event{},
			0,
			0,
			0,
		},
		{
			"single event no diagnostics",
			[]*proto.ListResource_Event{
				{
					DisplayName: "Test Resource",
					Identity: &proto.ResourceIdentityData{
						IdentityData: &proto.DynamicValue{
							Msgpack: []byte("\x81\xa7id_attr\xa4id-1"),
						},
					},
				},
			},
			1,
			0,
			0,
		},
		{
			"event with warning",
			[]*proto.ListResource_Event{
				{
					DisplayName: "Test Resource",
					Identity: &proto.ResourceIdentityData{
						IdentityData: &proto.DynamicValue{
							Msgpack: []byte("\x81\xa7id_attr\xa4id-1"),
						},
					},
					Diagnostic: []*proto.Diagnostic{
						{
							Severity: proto.Diagnostic_WARNING,
							Summary:  "Test warning",
							Detail:   "Warning detail",
						},
					},
				},
			},
			1,
			1,
			1,
		},
		{
			"only a warning",
			[]*proto.ListResource_Event{
				{
					Diagnostic: []*proto.Diagnostic{
						{
							Severity: proto.Diagnostic_WARNING,
							Summary:  "Test warning",
							Detail:   "Warning detail",
						},
					},
				},
			},
			0,
			1,
			1,
		},
		{
			"only an error",
			[]*proto.ListResource_Event{
				{
					Diagnostic: []*proto.Diagnostic{
						{
							Severity: proto.Diagnostic_ERROR,
							Summary:  "Test error",
							Detail:   "Error detail",
						},
					},
				},
			},
			0,
			1,
			0,
		},
		{
			"event with error",
			[]*proto.ListResource_Event{
				{
					DisplayName: "Test Resource",
					Identity: &proto.ResourceIdentityData{
						IdentityData: &proto.DynamicValue{
							Msgpack: []byte("\x81\xa7id_attr\xa4id-1"),
						},
					},
					Diagnostic: []*proto.Diagnostic{
						{
							Severity: proto.Diagnostic_ERROR,
							Summary:  "Test error",
							Detail:   "Error detail",
						},
					},
				},
			},
			0,
			1,
			0,
		},
		{
			"multiple events mixed diagnostics",
			[]*proto.ListResource_Event{
				{
					DisplayName: "Resource 1",
					Identity: &proto.ResourceIdentityData{
						IdentityData: &proto.DynamicValue{
							Msgpack: []byte("\x81\xa7id_attr\xa4id-1"),
						},
					},
					Diagnostic: []*proto.Diagnostic{
						{
							Severity: proto.Diagnostic_WARNING,
							Summary:  "Warning 1",
							Detail:   "Warning detail 1",
						},
					},
				},
				{
					DisplayName: "Resource 2",
					Identity: &proto.ResourceIdentityData{
						IdentityData: &proto.DynamicValue{
							Msgpack: []byte("\x81\xa7id_attr\xa4id-2"),
						},
					},
				},
				{
					DisplayName: "Resource 3",
					Identity: &proto.ResourceIdentityData{
						IdentityData: &proto.DynamicValue{
							Msgpack: []byte("\x81\xa7id_attr\xa4id-3"),
						},
					},
					Diagnostic: []*proto.Diagnostic{
						{
							Severity: proto.Diagnostic_ERROR,
							Summary:  "Error 1",
							Detail:   "Error detail 1",
						},
						{
							Severity: proto.Diagnostic_WARNING,
							Summary:  "Warning 2",
							Detail:   "Warning detail 2",
						},
					},
				},
				{ // This event will never be reached
					DisplayName: "Resource 4",
					Identity: &proto.ResourceIdentityData{
						IdentityData: &proto.DynamicValue{
							Msgpack: []byte("\x81\xa7id_attr\xa4id-4"),
						},
					},
				},
			},
			2,
			3,
			2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := mockProviderClient(t)
			p := &GRPCProvider{
				client: client,
				ctx:    context.Background(),
			}

			mockStream := &mockListResourceStreamClient{
				events: tc.events,
			}

			client.EXPECT().ListResource(
				gomock.Any(),
				gomock.Any(),
			).Return(mockStream, nil)

			resp := p.ListResource(request)

			result := resp.Result.AsValueMap()
			nResults := result["data"].LengthInt()
			if nResults != tc.expectedCount {
				t.Fatalf("Expected %d results, got %d", tc.expectedCount, nResults)
			}

			nDiagnostics := len(resp.Diagnostics)
			if nDiagnostics != tc.expectedDiags {
				t.Fatalf("Expected %d diagnostics, got %d", tc.expectedDiags, nDiagnostics)
			}

			nWarnings := len(resp.Diagnostics.Warnings())
			if nWarnings != tc.expectedWarns {
				t.Fatalf("Expected %d warnings, got %d", tc.expectedWarns, nWarnings)
			}
		})
	}
}

func TestGRPCProvider_ListResource_Limit(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
		ctx:    context.Background(),
	}

	// Create a mock stream client that will return resource events
	mockStream := &mockListResourceStreamClient{
		events: []*proto.ListResource_Event{
			{
				DisplayName: "Test Resource 1",
				Identity: &proto.ResourceIdentityData{
					IdentityData: &proto.DynamicValue{
						Msgpack: []byte("\x81\xa7id_attr\xa4id-1"),
					},
				},
			},
			{
				DisplayName: "Test Resource 2",
				Identity: &proto.ResourceIdentityData{
					IdentityData: &proto.DynamicValue{
						Msgpack: []byte("\x81\xa7id_attr\xa4id-2"),
					},
				},
			},
			{
				DisplayName: "Test Resource 3",
				Identity: &proto.ResourceIdentityData{
					IdentityData: &proto.DynamicValue{
						Msgpack: []byte("\x81\xa7id_attr\xa4id-3"),
					},
				},
			},
		},
	}

	client.EXPECT().ListResource(
		gomock.Any(),
		gomock.Any(),
	).Return(mockStream, nil)

	// Create the request
	configVal := cty.ObjectVal(map[string]cty.Value{
		"config": cty.ObjectVal(map[string]cty.Value{
			"filter_attr": cty.StringVal("filter-value"),
			"nested_filter": cty.ObjectVal(map[string]cty.Value{
				"nested_attr": cty.StringVal("value"),
			}),
		}),
	})
	request := providers.ListResourceRequest{
		TypeName: "list",
		Config:   configVal,
		Limit:    2,
	}

	resp := p.ListResource(request)
	checkDiags(t, resp.Diagnostics)

	data := resp.Result.AsValueMap()
	if _, ok := data["data"]; !ok {
		t.Fatal("Expected 'data' key in result")
	}
	// Verify that we received both events
	if len(data["data"].AsValueSlice()) != 2 {
		t.Fatalf("Expected 2 resources, got %d", len(data["data"].AsValueSlice()))
	}
	results := data["data"].AsValueSlice()

	// Verify that we received both events
	if len(results) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(results))
	}
}

func TestGRPCProvider_GenerateResourceConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}
	client.EXPECT().GenerateResourceConfig(
		gomock.Any(),
		gomock.Cond[any](func(x any) bool {
			req := x.(*proto.GenerateResourceConfig_Request)
			if req.TypeName != "resource" {
				return false
			}
			if req.State == nil {
				t.Log("GenerateResourceConfig state is nil")
				return false
			}
			return true
		}),
	).Return(&proto.GenerateResourceConfig_Response{
		Config: &proto.DynamicValue{
			Msgpack: []byte("\x81\xa4attr\xa3bar"),
		},
	}, nil)
	resp := p.GenerateResourceConfig(providers.GenerateResourceConfigRequest{
		TypeName: "resource",
		State: cty.ObjectVal(map[string]cty.Value{
			"computed": cty.StringVal("computed"),
			"attr":     cty.StringVal("foo"),
		}),
	})
	checkDiags(t, resp.Diagnostics)
}
