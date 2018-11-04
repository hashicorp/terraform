package plugin

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	mockproto "github.com/hashicorp/terraform/plugin/mock_proto"
	"github.com/hashicorp/terraform/plugin/proto"
)

var _ providers.Interface = (*GRPCProvider)(nil)

func mockProviderClient(t *testing.T) *mockproto.MockProviderClient {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProviderClient(ctrl)

	// we always need a GetSchema method
	client.EXPECT().GetSchema(
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
	}
}

func TestGRPCProvider_GetSchema(t *testing.T) {
	p := &GRPCProvider{
		client: mockProviderClient(t),
	}

	resp := p.GetSchema()
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
	resp := p.PrepareProviderConfig(providers.PrepareProviderConfigRequest{Config: cfg})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_ValidateResourceTypeConfig(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().ValidateResourceTypeConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateResourceTypeConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"attr": "value"})
	resp := p.ValidateResourceTypeConfig(providers.ValidateResourceTypeConfigRequest{
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
	resp := p.ValidateDataSourceConfig(providers.ValidateDataSourceConfigRequest{
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

func TestGRPCProvider_Configure(t *testing.T) {
	client := mockProviderClient(t)
	p := &GRPCProvider{
		client: client,
	}

	client.EXPECT().Configure(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.Configure_Response{}, nil)

	resp := p.Configure(providers.ConfigureRequest{
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("foo"),
		}),
	})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvider_Stop(t *testing.T) {
	client := mockProviderClient(t)
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
