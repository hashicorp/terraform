package plugin

import (
	"log"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform/helper/schema"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/plugin/convert"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/msgpack"
	context "golang.org/x/net/context"
)

// NewGRPCProvisionerServerShim wraps a terraform.ResourceProvisioner in a
// proto.ProvisionerServer implementation. If the provided provisioner is not a
// *schema.Provisioner, this will return nil,
func NewGRPCProvisionerServerShim(p terraform.ResourceProvisioner) *GRPCProvisionerServer {
	sp, ok := p.(*schema.Provisioner)
	if !ok {
		return nil
	}
	return &GRPCProvisionerServer{
		provisioner: sp,
	}
}

type GRPCProvisionerServer struct {
	provisioner *schema.Provisioner
}

func (s *GRPCProvisionerServer) GetSchema(_ context.Context, req *proto.GetProvisionerSchema_Request) (*proto.GetProvisionerSchema_Response, error) {
	resp := &proto.GetProvisionerSchema_Response{}

	resp.Provisioner = &proto.Schema{
		Block: convert.ConfigSchemaToProto(schema.InternalMap(s.provisioner.Schema).CoreConfigSchema()),
	}

	return resp, nil
}

func (s *GRPCProvisionerServer) ValidateProvisionerConfig(_ context.Context, req *proto.ValidateProvisionerConfig_Request) (*proto.ValidateProvisionerConfig_Response, error) {
	resp := &proto.ValidateProvisionerConfig_Response{}

	cfgSchema := schema.InternalMap(s.provisioner.Schema).CoreConfigSchema()

	configVal, err := msgpack.Unmarshal(req.Config.Msgpack, cfgSchema.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	config := terraform.NewResourceConfigShimmed(configVal, cfgSchema)

	warns, errs := s.provisioner.Validate(config)
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, convert.WarnsAndErrsToProto(warns, errs))

	return resp, nil
}

// stringMapFromValue converts a cty.Value to a map[stirng]string.
// This will panic if the val is not a cty.Map(cty.String).
func stringMapFromValue(val cty.Value) map[string]string {
	m := map[string]string{}
	if val.IsNull() || !val.IsKnown() {
		return m
	}

	for it := val.ElementIterator(); it.Next(); {
		ak, av := it.Element()
		name := ak.AsString()

		if !av.IsKnown() || av.IsNull() {
			continue
		}

		av, _ = ctyconvert.Convert(av, cty.String)
		m[name] = av.AsString()
	}

	return m
}

// uiOutput implements the terraform.UIOutput interface to adapt the grpc
// stream to the legacy Provisioner.Apply method.
type uiOutput struct {
	srv proto.Provisioner_ProvisionResourceServer
}

func (o uiOutput) Output(s string) {
	err := o.srv.Send(&proto.ProvisionResource_Response{
		Output: toValidUTF8(s, string(utf8.RuneError)),
	})
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}
}

func (s *GRPCProvisionerServer) ProvisionResource(req *proto.ProvisionResource_Request, srv proto.Provisioner_ProvisionResourceServer) error {
	// We send back a diagnostics over the stream if there was a
	// provisioner-side problem.
	srvResp := &proto.ProvisionResource_Response{}

	cfgSchema := schema.InternalMap(s.provisioner.Schema).CoreConfigSchema()
	cfgVal, err := msgpack.Unmarshal(req.Config.Msgpack, cfgSchema.ImpliedType())
	if err != nil {
		srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, err)
		srv.Send(srvResp)
		return nil
	}
	resourceConfig := terraform.NewResourceConfigShimmed(cfgVal, cfgSchema)

	connVal, err := msgpack.Unmarshal(req.Connection.Msgpack, cty.Map(cty.String))
	if err != nil {
		srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, err)
		srv.Send(srvResp)
		return nil
	}

	conn := stringMapFromValue(connVal)

	instanceState := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: conn,
		},
		Meta: make(map[string]interface{}),
	}

	err = s.provisioner.Apply(uiOutput{srv}, instanceState, resourceConfig)
	if err != nil {
		srvResp.Diagnostics = convert.AppendProtoDiag(srvResp.Diagnostics, err)
		srv.Send(srvResp)
	}
	return nil
}

func (s *GRPCProvisionerServer) Stop(_ context.Context, req *proto.Stop_Request) (*proto.Stop_Response, error) {
	resp := &proto.Stop_Response{}

	err := s.provisioner.Stop()
	if err != nil {
		resp.Error = err.Error()
	}

	return resp, nil
}

// FIXME: backported from go1.13 strings package, remove once terraform is
//        using go >= 1.13
// ToValidUTF8 returns a copy of the string s with each run of invalid UTF-8 byte sequences
// replaced by the replacement string, which may be empty.
func toValidUTF8(s, replacement string) string {
	var b strings.Builder

	for i, c := range s {
		if c != utf8.RuneError {
			continue
		}

		_, wid := utf8.DecodeRuneInString(s[i:])
		if wid == 1 {
			b.Grow(len(s) + len(replacement))
			b.WriteString(s[:i])
			s = s[i:]
			break
		}
	}

	// Fast path for unchanged input
	if b.Cap() == 0 { // didn't call b.Grow above
		return s
	}

	invalid := false // previous byte was from an invalid UTF-8 sequence
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			i++
			invalid = false
			b.WriteByte(c)
			continue
		}
		_, wid := utf8.DecodeRuneInString(s[i:])
		if wid == 1 {
			i++
			if !invalid {
				invalid = true
				b.WriteString(replacement)
			}
			continue
		}
		invalid = false
		b.WriteString(s[i : i+wid])
		i += wid
	}

	return b.String()
}
