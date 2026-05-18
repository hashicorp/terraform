// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/grpc"
	gproto "google.golang.org/protobuf/proto"

	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

type stubPolicyClient struct {
	proto.PolicyClient

	evaluateResourceFn func(*proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error)
	evaluateProviderFn func(*proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error)
	evaluateModuleFn   func(*proto.PolicyEvaluateModuleRequest) (*proto.PolicyEvaluateModuleResponse, error)
}

func (s *stubPolicyClient) EvaluateResource(ctx context.Context, req *proto.PolicyEvaluateResourceRequest, _ ...grpc.CallOption) (*proto.PolicyEvaluateResourceResponse, error) {
	return s.evaluateResourceFn(req)
}

func (s *stubPolicyClient) EvaluateProvider(ctx context.Context, req *proto.PolicyEvaluateProviderRequest, _ ...grpc.CallOption) (*proto.PolicyEvaluateProviderResponse, error) {
	return s.evaluateProviderFn(req)
}

func (s *stubPolicyClient) EvaluateModule(ctx context.Context, req *proto.PolicyEvaluateModuleRequest, _ ...grpc.CallOption) (*proto.PolicyEvaluateModuleResponse, error) {
	return s.evaluateModuleFn(req)
}

func TestClientEvaluate(t *testing.T) {
	ctx := context.Background()

	var gotReq *proto.PolicyEvaluateResourceRequest
	c := &client{
		client: &stubPolicyClient{
			evaluateResourceFn: func(req *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
				gotReq = req
				return &proto.PolicyEvaluateResourceResponse{
					Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT,
				}, nil
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.Evaluate(ctx, EvaluationRequest[*proto.ResourceMetadata]{
		Target: "test_resource",
		Attrs: PolicyValue{
			Raw:           cty.ObjectVal(map[string]cty.Value{"secret": cty.StringVal("x")}),
			RedactedPaths: []cty.Path{cty.GetAttrPath("secret")},
		},
		PriorAttrs: PolicyValue{Raw: cty.NilVal},
	})

	if resp.Overall != AllowResult {
		t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
	}
	if len(resp.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
	}
	if gotReq == nil {
		t.Fatal("expected EvaluateResource RPC to be called")
	}
	if gotReq.EvaluationId == 0 {
		t.Fatal("expected non-zero evaluation id")
	}
	want := &proto.Path{Steps: []*proto.Path_Step{{
		Selector: &proto.Path_Step_AttributeName{AttributeName: "secret"},
	}}}
	if len(gotReq.Attrs.RedactedPaths) != 1 || !gproto.Equal(gotReq.Attrs.RedactedPaths[0], want) {
		t.Fatalf("unexpected redacted paths: %#v", gotReq.Attrs.RedactedPaths)
	}
}

func TestClientEvaluateProvider(t *testing.T) {
	ctx := context.Background()

	var gotReq *proto.PolicyEvaluateProviderRequest
	c := &client{
		client: &stubPolicyClient{
			evaluateProviderFn: func(req *proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error) {
				gotReq = req
				return &proto.PolicyEvaluateProviderResponse{
					Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT,
				}, nil
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateProvider(ctx, EvaluationRequest[*proto.ProviderMetadata]{
		Target: "test_provider",
		Attrs: PolicyValue{
			Raw:           cty.ObjectVal(map[string]cty.Value{"token": cty.StringVal("x")}),
			RedactedPaths: []cty.Path{cty.GetAttrPath("token")},
		},
	})

	if resp.Overall != AllowResult {
		t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
	}
	if len(resp.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
	}
	if gotReq == nil {
		t.Fatal("expected EvaluateProvider RPC to be called")
	}
	if gotReq.ProviderType != "test_provider" {
		t.Fatalf("unexpected provider type: got %q, want %q", gotReq.ProviderType, "test_provider")
	}
	want := &proto.Path{Steps: []*proto.Path_Step{{
		Selector: &proto.Path_Step_AttributeName{AttributeName: "token"},
	}}}
	if len(gotReq.Attrs.RedactedPaths) != 1 || !gproto.Equal(gotReq.Attrs.RedactedPaths[0], want) {
		t.Fatalf("unexpected redacted paths: %#v", gotReq.Attrs.RedactedPaths)
	}
}

func TestClientEvaluateModule(t *testing.T) {
	ctx := context.Background()

	var gotReq *proto.PolicyEvaluateModuleRequest
	c := &client{
		client: &stubPolicyClient{
			evaluateModuleFn: func(req *proto.PolicyEvaluateModuleRequest) (*proto.PolicyEvaluateModuleResponse, error) {
				gotReq = req
				return &proto.PolicyEvaluateModuleResponse{
					Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT,
				}, nil
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateModule(ctx, EvaluationRequest[*proto.ModuleMetadata]{
		Target: "./child",
	})

	if resp.Overall != AllowResult {
		t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
	}
	if len(resp.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
	}
	if gotReq == nil {
		t.Fatal("expected EvaluateModule RPC to be called")
	}
	if gotReq.ModuleSource != "./child" {
		t.Fatalf("unexpected module source: got %q, want %q", gotReq.ModuleSource, "./child")
	}
}
