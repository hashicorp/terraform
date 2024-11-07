// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin6

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockproto "github.com/hashicorp/terraform/internal/plugin6/mock_proto"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
)

var _ providers.Interface = (*GRPCProvider)(nil)

var (
	equateEmpty   = cmpopts.EquateEmpty()
	typeComparer  = cmp.Comparer(cty.Type.Equals)
	valueComparer = cmp.Comparer(cty.Value.RawEquals)
)

func mockProviderClient(t *testing.T) *mockproto.MockProviderClient {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	// we always need a GetSchema method
	client.EXPECT().GetProviderSchema(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(providerProtoSchema(), nil)

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
			"resource": {
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
		DataSourceSchemas: map[string]*proto.Schema{
			"data": {
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
		ServerCapabilities: &proto.ServerCapabilities{
			GetProviderSchemaOptional: true,
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

	client.EXPECT().GetProviderSchema(
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

	client.EXPECT().GetProviderSchema(
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

func TestGRPCProvider_PrepareProviderConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ValidateProviderConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateProviderConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"attr": "value"})
	resp := p.ValidateProviderConfig(providers.ValidateProviderConfigRequest{Config: cfg})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_ValidateResourceConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ValidateResourceConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateResourceConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"attr": "value"})
	resp := p.ValidateResourceConfig(providers.ValidateResourceConfigRequest{
		TypeName: "resource",
		Config:   cfg,
	})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_ValidateDataResourceConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ValidateDataResourceConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateDataResourceConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"attr": "value"})
	resp := p.ValidateDataResourceConfig(providers.ValidateDataResourceConfigRequest{
		TypeName: "data",
		Config:   cfg,
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

func TestGRPCProvider_Configure(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ConfigureProvider(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ConfigureProvider_Response{}, nil)

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

	client.EXPECT().StopProvider(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.StopProvider_Response{}, nil)

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
