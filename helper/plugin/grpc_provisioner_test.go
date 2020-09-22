package plugin

import (
	"testing"
	"unicode/utf8"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform/helper/schema"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	mockproto "github.com/hashicorp/terraform/plugin/mock_proto"
	"github.com/hashicorp/terraform/terraform"
	context "golang.org/x/net/context"
)

var _ proto.ProvisionerServer = (*GRPCProvisionerServer)(nil)

type validUTF8Matcher string

func (m validUTF8Matcher) Matches(x interface{}) bool {
	resp := x.(*proto.ProvisionResource_Response)
	return utf8.Valid([]byte(resp.Output))
}

func (m validUTF8Matcher) String() string {
	return string(m)
}

func mockProvisionerServer(t *testing.T, c *gomock.Controller) *mockproto.MockProvisioner_ProvisionResourceServer {
	server := mockproto.NewMockProvisioner_ProvisionResourceServer(c)

	server.EXPECT().Send(
		validUTF8Matcher("check for valid utf8"),
	).Return(nil)

	return server
}

// ensure that a provsioner cannot return invalid utf8 which isn't allowed in
// the grpc protocol.
func TestProvisionerInvalidUTF8(t *testing.T) {
	p := &schema.Provisioner{
		ConnSchema: map[string]*schema.Schema{
			"foo": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		Schema: map[string]*schema.Schema{
			"foo": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},

		ApplyFunc: func(ctx context.Context) error {
			out := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)
			out.Output("invalid \xc3\x28\n")
			return nil
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mockProvisionerServer(t, ctrl)
	cfg := &proto.DynamicValue{
		Msgpack: []byte("\x81\xa3foo\x01"),
	}
	conn := &proto.DynamicValue{
		Msgpack: []byte("\x81\xa3foo\xa4host"),
	}
	provisionerServer := NewGRPCProvisionerServerShim(p)
	req := &proto.ProvisionResource_Request{
		Config:     cfg,
		Connection: conn,
	}

	if err := provisionerServer.ProvisionResource(req, srv); err != nil {
		t.Fatal(err)
	}
}
