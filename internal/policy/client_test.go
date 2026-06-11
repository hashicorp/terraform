// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/grpc"

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
	ctx := t.Context()

	tests := []struct {
		name       string
		attrs      cty.Value
		priorAttrs cty.Value

		// an optional function to override the default evaluateResourceFn
		evaluateResourceFn func(*proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error)

		// assertResponse is a helper function for each case to further assert the response of an evaluation
		assertResponse func(*testing.T, *callback.MockRegistry, *proto.PolicyEvaluateResourceRequest, EvaluationResponse)
	}{
		{
			name:       "nil attrs and prior attrs",
			attrs:      cty.NilVal,
			priorAttrs: cty.NilVal,
			assertResponse: func(t *testing.T, registry *callback.MockRegistry, req *proto.PolicyEvaluateResourceRequest, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != AllowResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
				}
				if len(resp.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
				}
				if req == nil {
					t.Fatal("expected request, got nil")
				}
			},
		},
		{
			name:       "non-nil attrs and prior attrs",
			attrs:      cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("test")}),
			priorAttrs: cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("prior")}),
			assertResponse: func(t *testing.T, registry *callback.MockRegistry, req *proto.PolicyEvaluateResourceRequest, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != AllowResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
				}
				if len(resp.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
				}
			},
		},
		{
			name:       "transforms diagnostics from response",
			attrs:      cty.NilVal,
			priorAttrs: cty.NilVal,
			evaluateResourceFn: func(req *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
				return &proto.PolicyEvaluateResourceResponse{
					Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
					PolicyDetails: []*proto.PolicyEvaluationDetail{{
						Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostics: []*proto.Diagnostic{{
							Severity: proto.Severity_WARNING,
							Summary:  "policy warning",
							Detail:   "transformed warning detail",
							Result: &proto.DiagnosticResult{
								Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
							},
						}},
					}},
				}, nil
			},
			assertResponse: func(t *testing.T, registry *callback.MockRegistry, req *proto.PolicyEvaluateResourceRequest, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != DenyResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, DenyResult)
				}
				if len(resp.Diagnostics) != 1 {
					t.Fatalf("unexpected diagnostics count: got %d, want 1", len(resp.Diagnostics))
				}

				diag := resp.Diagnostics[0]
				if diag.Severity() != tfdiags.Warning {
					t.Fatalf("unexpected diagnostic severity: got %s, want %s", diag.Severity(), tfdiags.Warning)
				}
				desc := diag.Description()
				if desc.Summary != "policy warning" {
					t.Fatalf("unexpected diagnostic summary: got %q, want %q", desc.Summary, "policy warning")
				}
				if desc.Detail != "transformed warning detail" {
					t.Fatalf("unexpected diagnostic detail: got %q, want %q", desc.Detail, "transformed warning detail")
				}

				extra := tfdiags.ExtraInfo[*PolicyExtra](diag)
				expectedExtra := &PolicyExtra{
					Severity: hcl.DiagWarning,
					Result:   DenyResult,
					Policy: Policy{
						Result: DenyResult,
						Range:  &hcl.Range{},
					},
				}
				if diff := cmp.Diff(extra, expectedExtra); diff != "" {
					t.Fatalf("unexpected diagnostic extra: %s", diff)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotReq *proto.PolicyEvaluateResourceRequest
			registry := &callback.MockRegistry{NextIDValue: 23}
			c := &client{
				client: &stubPolicyClient{
					evaluateResourceFn: func(req *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
						gotReq = req

						// assert that the evaluation id is registered with the callback registry
						_, ok := registry.FunctionsStore[req.EvaluationId]
						if !ok {
							t.Fatalf("expected evaluation id %d to be registered", req.EvaluationId)
						}

						if test.evaluateResourceFn != nil {
							return test.evaluateResourceFn(req)
						}
						return &proto.PolicyEvaluateResourceResponse{
							Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT,
						}, nil
					},
				},
				callbackRegistry: registry,
			}

			resp := c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
				Target:     "test_resource",
				Attrs:      test.attrs,
				PriorAttrs: test.priorAttrs,
			})

			test.assertResponse(t, registry, gotReq, resp)
			if gotReq == nil {
				t.Fatal("expected EvaluateResource RPC to be called")
			}
			if gotReq.EvaluationId == 0 {
				t.Fatal("expected non-zero evaluation id")
			}

			// assert the registry functions that should have been called
			if !registry.NextIDCalled {
				t.Fatal("expected callback registry NextID to be called")
			}
			if !registry.RegisterCalled {
				t.Fatal("expected callback registry Register to be called")
			}
			if !registry.UnregisterCalled {
				t.Fatal("expected callback registry Unregister to be called")
			}

			// after the evaluation, the callback registry should have been cleaned up
			_, ok := registry.FunctionsStore[gotReq.EvaluationId]
			if ok {
				t.Fatalf("expected evaluation id %d to be unregistered", gotReq.EvaluationId)
			}
		})
	}
}

func TestClientEvaluateProvider(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name               string
		attrs              cty.Value
		evaluateProviderFn func(*proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error)
		assertResponse     func(*testing.T, EvaluationResponse)
	}{
		{
			name:  "nil attrs",
			attrs: cty.NilVal,
			assertResponse: func(t *testing.T, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != AllowResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
				}
				if len(resp.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
				}
			},
		},
		{
			name:  "unknown attrs",
			attrs: cty.UnknownVal(cty.EmptyObject),
			assertResponse: func(t *testing.T, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != AllowResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
				}
				if len(resp.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
				}
			},
		},
		{
			name:  "non-nil attrs",
			attrs: cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("test")}),
			assertResponse: func(t *testing.T, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != AllowResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
				}
				if len(resp.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
				}
			},
		},
		{
			name:  "transforms diagnostics from response",
			attrs: cty.NilVal,
			evaluateProviderFn: func(req *proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error) {
				return &proto.PolicyEvaluateProviderResponse{
					Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
					PolicyDetails: []*proto.PolicyEvaluationDetail{{
						Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostics: []*proto.Diagnostic{{
							Severity: proto.Severity_WARNING,
							Summary:  "policy warning",
							Detail:   "transformed warning detail",
							Result: &proto.DiagnosticResult{
								Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
							},
						}},
					}},
				}, nil
			},
			assertResponse: func(t *testing.T, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != DenyResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, DenyResult)
				}
				if len(resp.Diagnostics) != 1 {
					t.Fatalf("unexpected diagnostics count: got %d, want 1", len(resp.Diagnostics))
				}

				diag := resp.Diagnostics[0]
				if diag.Severity() != tfdiags.Warning {
					t.Fatalf("unexpected diagnostic severity: got %s, want %s", diag.Severity(), tfdiags.Warning)
				}
				desc := diag.Description()
				if desc.Summary != "policy warning" {
					t.Fatalf("unexpected diagnostic summary: got %q, want %q", desc.Summary, "policy warning")
				}
				if desc.Detail != "transformed warning detail" {
					t.Fatalf("unexpected diagnostic detail: got %q, want %q", desc.Detail, "transformed warning detail")
				}

				extra := tfdiags.ExtraInfo[*PolicyExtra](diag)
				expectedExtra := &PolicyExtra{
					Severity: hcl.DiagWarning,
					Result:   DenyResult,
					Policy: Policy{
						Result: DenyResult,
						Range:  &hcl.Range{},
					},
				}
				if diff := cmp.Diff(extra, expectedExtra); diff != "" {
					t.Fatalf("unexpected diagnostic extra: %s", diff)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotReq *proto.PolicyEvaluateProviderRequest
			c := &client{
				client: &stubPolicyClient{
					evaluateProviderFn: func(req *proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error) {
						gotReq = req
						if test.evaluateProviderFn != nil {
							return test.evaluateProviderFn(req)
						}
						return &proto.PolicyEvaluateProviderResponse{
							Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT,
						}, nil
					},
				},
				callbackRegistry: callback.NewRegistry(),
			}

			resp := c.EvaluateProvider(ctx, EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]{
				Target: "test_provider",
				Attrs:  test.attrs,
			})

			test.assertResponse(t, resp)
			if gotReq == nil {
				t.Fatal("expected EvaluateProvider RPC to be called")
			}
			if gotReq.ProviderType != "test_provider" {
				t.Fatalf("unexpected provider type: got %q, want %q", gotReq.ProviderType, "test_provider")
			}
		})
	}
}

func TestClientEvaluateModule(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name             string
		evaluateModuleFn func(*proto.PolicyEvaluateModuleRequest) (*proto.PolicyEvaluateModuleResponse, error)
		assertResponse   func(*testing.T, EvaluationResponse)
	}{
		{
			name: "allow response",
			assertResponse: func(t *testing.T, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != AllowResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
				}
				if len(resp.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
				}
			},
		},
		{
			name: "transforms diagnostics from response",
			evaluateModuleFn: func(req *proto.PolicyEvaluateModuleRequest) (*proto.PolicyEvaluateModuleResponse, error) {
				return &proto.PolicyEvaluateModuleResponse{
					Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
					PolicyDetails: []*proto.PolicyEvaluationDetail{{
						Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostics: []*proto.Diagnostic{{
							Severity: proto.Severity_WARNING,
							Summary:  "policy warning",
							Detail:   "transformed warning detail",
							Result: &proto.DiagnosticResult{
								Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
							},
						}},
					}},
				}, nil
			},
			assertResponse: func(t *testing.T, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != DenyResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, DenyResult)
				}
				if len(resp.Diagnostics) != 1 {
					t.Fatalf("unexpected diagnostics count: got %d, want 1", len(resp.Diagnostics))
				}

				diag := resp.Diagnostics[0]
				if diag.Severity() != tfdiags.Warning {
					t.Fatalf("unexpected diagnostic severity: got %s, want %s", diag.Severity(), tfdiags.Warning)
				}
				desc := diag.Description()
				if desc.Summary != "policy warning" {
					t.Fatalf("unexpected diagnostic summary: got %q, want %q", desc.Summary, "policy warning")
				}
				if desc.Detail != "transformed warning detail" {
					t.Fatalf("unexpected diagnostic detail: got %q, want %q", desc.Detail, "transformed warning detail")
				}

				extra := tfdiags.ExtraInfo[*PolicyExtra](diag)
				expectedExtra := &PolicyExtra{
					Severity: hcl.DiagWarning,
					Result:   DenyResult,
					Policy: Policy{
						Result: DenyResult,
						Range:  &hcl.Range{},
					},
				}
				if diff := cmp.Diff(extra, expectedExtra); diff != "" {
					t.Fatalf("unexpected diagnostic extra: %s", diff)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotReq *proto.PolicyEvaluateModuleRequest
			c := &client{
				client: &stubPolicyClient{
					evaluateModuleFn: func(req *proto.PolicyEvaluateModuleRequest) (*proto.PolicyEvaluateModuleResponse, error) {
						gotReq = req
						if test.evaluateModuleFn != nil {
							return test.evaluateModuleFn(req)
						}
						return &proto.PolicyEvaluateModuleResponse{
							Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT,
						}, nil
					},
				},
				callbackRegistry: callback.NewRegistry(),
			}

			resp := c.EvaluateModule(ctx, EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]{
				Target: "./child",
			})

			test.assertResponse(t, resp)
			if gotReq == nil {
				t.Fatal("expected EvaluateModule RPC to be called")
			}
			if gotReq.ModuleSource != "./child" {
				t.Fatalf("unexpected module source: got %q, want %q", gotReq.ModuleSource, "./child")
			}
		})
	}
}
