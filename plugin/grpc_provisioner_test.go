package plugin

import (
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/config/hcl2shim"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/zclconf/go-cty/cty"

	mockproto "github.com/hashicorp/terraform/plugin/mock_proto"
)

var _ provisioners.Interface = (*GRPCProvisioner)(nil)

var (
	equateEmpty   = cmpopts.EquateEmpty()
	typeComparer  = cmp.Comparer(cty.Type.Equals)
	valueComparer = cmp.Comparer(cty.Value.RawEquals)
)

func mockProvisionerClient(t *testing.T) *mockproto.MockProvisionerClient {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProvisionerClient(ctrl)

	// we always need a GetSchema method
	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
	).Return(provisionerProtoSchema(), nil)

	return client
}

func provisionerProtoSchema() *proto.GetProvisionerSchema_Response {
	return &proto.GetProvisionerSchema_Response{
		Provisioner: &proto.Schema{
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
	}
}

func TestGRPCProvisioner_GetSchema(t *testing.T) {
	p := &GRPCProvisioner{
		client: mockProvisionerClient(t),
	}

	resp := p.GetSchema()
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvisioner_ValidateProvisionerConfig(t *testing.T) {
	client := mockProvisionerClient(t)
	p := &GRPCProvisioner{
		client: client,
	}

	client.EXPECT().ValidateProvisionerConfig(
		gomock.Any(),
		gomock.Any(),
	).Return(&proto.ValidateProvisionerConfig_Response{}, nil)

	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"attr": "value"})
	resp := p.ValidateProvisionerConfig(provisioners.ValidateProvisionerConfigRequest{Config: cfg})
	checkDiags(t, resp.Diagnostics)
}

func TestGRPCProvisioner_ProvisionResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mockproto.NewMockProvisionerClient(ctrl)

	// we always need a GetSchema method
	client.EXPECT().GetSchema(
		gomock.Any(),
		gomock.Any(),
	).Return(provisionerProtoSchema(), nil)

	stream := mockproto.NewMockProvisioner_ProvisionResourceClient(ctrl)
	stream.EXPECT().Recv().Return(&proto.ProvisionResource_Response{
		Output: "provisioned",
	}, io.EOF)

	client.EXPECT().ProvisionResource(
		gomock.Any(),
		gomock.Any(),
	).Return(stream, nil)

	p := &GRPCProvisioner{
		client: client,
	}

	rec := &provisionRecorder{}

	resp := p.ProvisionResource(provisioners.ProvisionResourceRequest{
		Config: cty.ObjectVal(map[string]cty.Value{
			"attr": cty.StringVal("value"),
		}),
		Connection: cty.EmptyObjectVal,
		UIOutput:   rec,
	})

	if resp.Diagnostics.HasErrors() {
		t.Fatal(resp.Diagnostics.Err())
	}

	if len(rec.output) == 0 || rec.output[0] != "provisioned" {
		t.Fatalf("expected %q, got %q", []string{"provisioned"}, rec.output)
	}
}

type provisionRecorder struct {
	output []string
}

func (r *provisionRecorder) Output(s string) {
	r.output = append(r.output, s)
}

func TestGRPCProvisioner_Stop(t *testing.T) {
	client := mockProvisionerClient(t)
	p := &GRPCProvisioner{
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
