// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	gproto "google.golang.org/protobuf/proto"

	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

type stubPolicyClient struct {
	proto.PolicyClient

	setupFn            func(*proto.PolicySetupRequest) (*proto.PolicySetupResponse, error)
	evaluateResourceFn func(*proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error)
	evaluateProviderFn func(*proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error)
	evaluateModuleFn   func(*proto.PolicyEvaluateModuleRequest) (*proto.PolicyEvaluateModuleResponse, error)
}

func (s *stubPolicyClient) Setup(ctx context.Context, req *proto.PolicySetupRequest, _ ...grpc.CallOption) (*proto.PolicySetupResponse, error) {
	return s.setupFn(req)
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
		attrs      PolicyValue
		priorAttrs PolicyValue

		// an optional function to override the default evaluateResourceFn
		evaluateResourceFn func(*proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error)

		// assertResponse is a helper function for each case to further assert the response of an evaluation
		assertResponse func(*testing.T, *callback.MockRegistry, *proto.PolicyEvaluateResourceRequest, EvaluationResponse)
	}{
		{
			name:       "nil attrs and prior attrs",
			attrs:      PolicyValue{Raw: cty.NilVal},
			priorAttrs: PolicyValue{Raw: cty.NilVal},
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
			name: "non-nil attrs and prior attrs",
			attrs: PolicyValue{
				Raw:           cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("test")}),
				RedactedPaths: []cty.Path{cty.GetAttrPath("secret")},
			},
			priorAttrs: PolicyValue{
				Raw: cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("prior")}),
			},
			assertResponse: func(t *testing.T, registry *callback.MockRegistry, req *proto.PolicyEvaluateResourceRequest, resp EvaluationResponse) {
				t.Helper()
				if resp.Overall != AllowResult {
					t.Fatalf("unexpected result: got %s, want %s", resp.Overall, AllowResult)
				}
				if len(resp.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
				}

				want := &proto.AttributePath{Steps: []*proto.AttributePath_Step{{
					Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "secret"},
				}}}
				if len(req.Attrs.RedactedPaths) != 1 || !gproto.Equal(req.Attrs.RedactedPaths[0], want) {
					t.Fatalf("unexpected redacted paths: %#v", req.Attrs.RedactedPaths)
				}
			},
		},
		{
			name:       "transforms diagnostics from response",
			attrs:      PolicyValue{Raw: cty.NilVal},
			priorAttrs: PolicyValue{Raw: cty.NilVal},
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
		attrs              PolicyValue
		evaluateProviderFn func(*proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error)
		assertResponse     func(*testing.T, EvaluationResponse)
	}{
		{
			name:  "nil attrs",
			attrs: PolicyValue{Raw: cty.NilVal},
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
			attrs: PolicyValue{Raw: cty.UnknownVal(cty.EmptyObject)},
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
			attrs: PolicyValue{Raw: cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("test")})},
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
			attrs: PolicyValue{Raw: cty.NilVal},
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

func TestClientSetupEntitlement(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name        string
		entitlement *Entitlement
		want        *proto.PolicySetupRequest_Entitlement
	}{
		{
			name:        "nil entitlement is not serialized",
			entitlement: nil,
			want:        nil,
		},
		{
			name: "entitlement is mapped onto the proto request",
			entitlement: &Entitlement{
				Host:  "app.terraform.io",
				Token: "secret",
				Org:   "hashicorp",
			},
			want: &proto.PolicySetupRequest_Entitlement{
				Host:  "app.terraform.io",
				Token: "secret",
				Org:   "hashicorp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq *proto.PolicySetupRequest
			c := &client{
				client: &stubPolicyClient{
					setupFn: func(req *proto.PolicySetupRequest) (*proto.PolicySetupResponse, error) {
						gotReq = req
						return &proto.PolicySetupResponse{}, nil
					},
				},
			}

			resp := c.Setup(ctx, SetupRequest{
				SourceLocations: []string{"./policies"},
				Entitlement:     tt.entitlement,
			})
			if resp.Diagnostics.HasErrors() {
				t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
			}
			if gotReq == nil {
				t.Fatal("expected a setup request to be sent")
			}

			got := gotReq.Entitlement
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil entitlement, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected entitlement, got nil")
			}
			if got.Host != tt.want.Host || got.Token != tt.want.Token || got.Org != tt.want.Org {
				t.Fatalf("unexpected entitlement: got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isPluginCrashError tests
// ---------------------------------------------------------------------------

// TestIsPluginCrashError covers the full decision matrix for the transport-
// level crash classifier. Each sub-test documents the scenario it guards.
func TestIsPluginCrashError(t *testing.T) {
	t.Run("nil_error_returns_false", func(t *testing.T) {
		if isPluginCrashError(nil) {
			t.Error("expected false for nil error")
		}
	})

	t.Run("io_EOF_returns_true", func(t *testing.T) {
		// io.EOF indicates the server closed the connection mid-stream,
		// which is what happens on a segfault or OOM kill.
		if !isPluginCrashError(io.EOF) {
			t.Error("expected true for io.EOF")
		}
	})

	t.Run("grpc_unavailable_returns_true", func(t *testing.T) {
		// Unavailable is the canonical gRPC code when the remote process
		// dies and the connection can no longer be used.
		err := grpcstatus.Error(grpccodes.Unavailable, "connection refused")
		if !isPluginCrashError(err) {
			t.Errorf("expected true for Unavailable, got false")
		}
	})

	t.Run("grpc_unavailable_broken_pipe_returns_true", func(t *testing.T) {
		err := grpcstatus.Error(grpccodes.Unavailable, "transport: write tcp ... broken pipe")
		if !isPluginCrashError(err) {
			t.Error("expected true for Unavailable/broken-pipe")
		}
	})

	t.Run("grpc_internal_eof_returns_true", func(t *testing.T) {
		// go-grpc wraps connection-loss errors as Internal when the framer
		// encounters an unexpected EOF on the wire.
		err := grpcstatus.Error(grpccodes.Internal, "EOF")
		if !isPluginCrashError(err) {
			t.Error("expected true for Internal/EOF")
		}
	})

	t.Run("grpc_internal_broken_pipe_returns_true", func(t *testing.T) {
		err := grpcstatus.Error(grpccodes.Internal, "transport: broken pipe")
		if !isPluginCrashError(err) {
			t.Error("expected true for Internal/broken pipe")
		}
	})

	t.Run("grpc_internal_connection_reset_returns_true", func(t *testing.T) {
		err := grpcstatus.Error(grpccodes.Internal, "read tcp: connection reset by peer")
		if !isPluginCrashError(err) {
			t.Error("expected true for Internal/connection reset")
		}
	})

	t.Run("grpc_internal_unexpected_eof_returns_true", func(t *testing.T) {
		err := grpcstatus.Error(grpccodes.Internal, "unexpected EOF")
		if !isPluginCrashError(err) {
			t.Error("expected true for Internal/unexpected EOF")
		}
	})

	t.Run("grpc_internal_other_message_returns_false", func(t *testing.T) {
		// Internal with an unrelated message must not be classified as a crash.
		err := grpcstatus.Error(grpccodes.Internal, "server panicked with unknown error")
		if isPluginCrashError(err) {
			t.Error("expected false for Internal with unrelated message")
		}
	})

	t.Run("grpc_unknown_eof_returns_true", func(t *testing.T) {
		err := grpcstatus.Error(grpccodes.Unknown, "EOF")
		if !isPluginCrashError(err) {
			t.Error("expected true for Unknown/EOF")
		}
	})

	t.Run("grpc_unknown_connection_reset_returns_true", func(t *testing.T) {
		err := grpcstatus.Error(grpccodes.Unknown, "connection reset by peer")
		if !isPluginCrashError(err) {
			t.Error("expected true for Unknown/connection reset")
		}
	})

	t.Run("grpc_unknown_other_message_returns_false", func(t *testing.T) {
		err := grpcstatus.Error(grpccodes.Unknown, "unexpected policy result")
		if isPluginCrashError(err) {
			t.Error("expected false for Unknown with unrelated message")
		}
	})

	t.Run("grpc_deadline_exceeded_returns_false", func(t *testing.T) {
		// DeadlineExceeded is handled by the per-call timeout path, not
		// as a crash.
		err := grpcstatus.Error(grpccodes.DeadlineExceeded, "deadline exceeded")
		if isPluginCrashError(err) {
			t.Error("expected false for DeadlineExceeded")
		}
	})

	t.Run("grpc_canceled_returns_false", func(t *testing.T) {
		// Canceled is produced when the client cancels the context (parent
		// cancellation path), not a plugin crash.
		err := grpcstatus.Error(grpccodes.Canceled, "context canceled")
		if isPluginCrashError(err) {
			t.Error("expected false for Canceled")
		}
	})

	t.Run("non_grpc_broken_pipe_returns_true", func(t *testing.T) {
		// Raw net errors wrapped by the transport layer — not a gRPC status.
		err := fmt.Errorf("write unix /tmp/plugin.sock->@: write: broken pipe")
		if !isPluginCrashError(err) {
			t.Error("expected true for raw broken pipe error")
		}
	})

	t.Run("non_grpc_connection_reset_returns_true", func(t *testing.T) {
		err := fmt.Errorf("read tcp 127.0.0.1:12345->127.0.0.1:54321: connection reset by peer")
		if !isPluginCrashError(err) {
			t.Error("expected true for raw connection reset error")
		}
	})

	t.Run("non_grpc_EOF_string_returns_true", func(t *testing.T) {
		err := fmt.Errorf("read unix: EOF")
		if !isPluginCrashError(err) {
			t.Error("expected true for raw EOF string error")
		}
	})

	t.Run("non_grpc_unrelated_error_returns_false", func(t *testing.T) {
		err := fmt.Errorf("permission denied")
		if isPluginCrashError(err) {
			t.Error("expected false for unrelated non-gRPC error")
		}
	})

	t.Run("case_insensitive_eof_returns_true", func(t *testing.T) {
		// Some transport implementations capitalise differently.
		err := grpcstatus.Error(grpccodes.Internal, "Unexpected EOF reading framer")
		if !isPluginCrashError(err) {
			t.Error("expected true for case-insensitive EOF match")
		}
	})
}

// ---------------------------------------------------------------------------
// transportErrorDiags tests
// ---------------------------------------------------------------------------

// TestTransportErrorDiags verifies the structure of the diagnostic produced
// for a transport-level crash.
func TestTransportErrorDiags(t *testing.T) {
	err := grpcstatus.Error(grpccodes.Unavailable, "connection refused")
	diags := transportErrorDiags("aws_instance", err)

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}

	d := diags[0]
	if d.Severity != proto.Severity_ERROR {
		t.Errorf("severity = %v, want ERROR", d.Severity)
	}
	if d.Summary != "Policy plugin crashed" {
		t.Errorf("summary = %q, want \"Policy plugin crashed\"", d.Summary)
	}
	if !strings.Contains(d.Detail, "aws_instance") {
		t.Errorf("detail %q does not mention resource label", d.Detail)
	}
	if !strings.Contains(d.Detail, pluginCrashDiagMsg) {
		t.Errorf("detail %q does not contain pluginCrashDiagMsg sentinel", d.Detail)
	}
	if !strings.Contains(d.Detail, err.Error()) {
		t.Errorf("detail %q does not contain original error message", d.Detail)
	}
}

// ---------------------------------------------------------------------------
// EvaluateResource plugin-crash path tests
// ---------------------------------------------------------------------------

// TestClientEvaluateResource_PluginCrash_Unavailable verifies that when the
// gRPC transport returns Unavailable (the most common crash signal), the client
// returns a PolicyErrorResult with a "Policy plugin crashed" diagnostic rather
// than a generic "Failed to evaluate Terraform Policy" diagnostic.
func TestClientEvaluateResource_PluginCrash_Unavailable(t *testing.T) {
	ctx := t.Context()

	crashErr := grpcstatus.Error(grpccodes.Unavailable, "transport: connection refused")

	c := &client{
		client: &stubPolicyClient{
			evaluateResourceFn: func(_ *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
				return nil, crashErr
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target: "aws_instance",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected diagnostics, got none")
	}
	d := resp.Diagnostics[0]
	desc := d.Description()
	if desc.Summary != "Policy plugin crashed" {
		t.Errorf("summary = %q, want \"Policy plugin crashed\"", desc.Summary)
	}
	if !strings.Contains(desc.Detail, pluginCrashDiagMsg) {
		t.Errorf("detail %q does not contain pluginCrashDiagMsg", desc.Detail)
	}
}

// TestClientEvaluateResource_PluginCrash_EOF verifies that an io.EOF from the
// gRPC layer (plugin killed mid-stream) is classified as a crash.
func TestClientEvaluateResource_PluginCrash_EOF(t *testing.T) {
	ctx := t.Context()

	c := &client{
		client: &stubPolicyClient{
			evaluateResourceFn: func(_ *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
				return nil, io.EOF
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target: "aws_vpc",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected diagnostics, got none")
	}
	d := resp.Diagnostics[0]
	if d.Description().Summary != "Policy plugin crashed" {
		t.Errorf("summary = %q, want \"Policy plugin crashed\"", d.Description().Summary)
	}
}

// TestClientEvaluateResource_PluginCrash_InternalEOF verifies that
// grpc Internal/EOF is classified as a crash and not as a generic error.
func TestClientEvaluateResource_PluginCrash_InternalEOF(t *testing.T) {
	ctx := t.Context()

	c := &client{
		client: &stubPolicyClient{
			evaluateResourceFn: func(_ *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
				return nil, grpcstatus.Error(grpccodes.Internal, "EOF")
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target: "aws_s3_bucket",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	d := resp.Diagnostics[0]
	if d.Description().Summary != "Policy plugin crashed" {
		t.Errorf("summary = %q, want \"Policy plugin crashed\"", d.Description().Summary)
	}
}

// TestClientEvaluateResource_PluginCrash_BrokenPipe verifies that a raw
// broken-pipe error (non-gRPC status) is classified as a crash.
func TestClientEvaluateResource_PluginCrash_BrokenPipe(t *testing.T) {
	ctx := t.Context()

	c := &client{
		client: &stubPolicyClient{
			evaluateResourceFn: func(_ *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
				return nil, fmt.Errorf("write: broken pipe")
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target: "aws_security_group",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected diagnostics, got none")
	}
	if resp.Diagnostics[0].Description().Summary != "Policy plugin crashed" {
		t.Errorf("unexpected summary: %q", resp.Diagnostics[0].Description().Summary)
	}
}

// TestClientEvaluateResource_NonCrashError_KeepsGenericMessage verifies that
// a non-transport error (e.g. grpc PermissionDenied) still produces the
// generic "Failed to evaluate Terraform Policy" diagnostic and is not
// misclassified as a crash.
func TestClientEvaluateResource_NonCrashError_KeepsGenericMessage(t *testing.T) {
	ctx := t.Context()

	c := &client{
		client: &stubPolicyClient{
			evaluateResourceFn: func(_ *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
				return nil, grpcstatus.Error(grpccodes.PermissionDenied, "access denied")
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target: "aws_instance",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected diagnostics, got none")
	}
	d := resp.Diagnostics[0]
	if d.Description().Summary == "Policy plugin crashed" {
		t.Errorf("PermissionDenied should not be classified as a crash; got summary %q", d.Description().Summary)
	}
	if d.Description().Summary != "Failed to evaluate Terraform Policy" {
		t.Errorf("unexpected summary: %q", d.Description().Summary)
	}
}

// ---------------------------------------------------------------------------
// EvaluateProvider plugin-crash path tests
// ---------------------------------------------------------------------------

// TestClientEvaluateProvider_PluginCrash_Unavailable verifies that the
// provider evaluation path also classifies transport failures correctly.
func TestClientEvaluateProvider_PluginCrash_Unavailable(t *testing.T) {
	ctx := t.Context()

	c := &client{
		client: &stubPolicyClient{
			evaluateProviderFn: func(_ *proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error) {
				return nil, grpcstatus.Error(grpccodes.Unavailable, "server unavailable")
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateProvider(ctx, EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]{
		Target: "aws",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if resp.Diagnostics[0].Description().Summary != "Policy plugin crashed" {
		t.Errorf("unexpected summary %q for provider crash", resp.Diagnostics[0].Description().Summary)
	}
}

// TestClientEvaluateProvider_NonCrashError_KeepsGenericMessage verifies that
// a non-transport error on the provider path keeps the generic message.
func TestClientEvaluateProvider_NonCrashError_KeepsGenericMessage(t *testing.T) {
	ctx := t.Context()

	c := &client{
		client: &stubPolicyClient{
			evaluateProviderFn: func(_ *proto.PolicyEvaluateProviderRequest) (*proto.PolicyEvaluateProviderResponse, error) {
				return nil, grpcstatus.Error(grpccodes.Unimplemented, "not implemented")
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateProvider(ctx, EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]{
		Target: "aws",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if resp.Diagnostics[0].Description().Summary != "Failed to evaluate Terraform Policy" {
		t.Errorf("unexpected summary %q", resp.Diagnostics[0].Description().Summary)
	}
}

// ---------------------------------------------------------------------------
// EvaluateModule plugin-crash path tests
// ---------------------------------------------------------------------------

// TestClientEvaluateModule_PluginCrash_ConnectionReset verifies that a
// connection-reset error on the module evaluation path is classified as a crash.
func TestClientEvaluateModule_PluginCrash_ConnectionReset(t *testing.T) {
	ctx := t.Context()

	c := &client{
		client: &stubPolicyClient{
			evaluateModuleFn: func(_ *proto.PolicyEvaluateModuleRequest) (*proto.PolicyEvaluateModuleResponse, error) {
				return nil, fmt.Errorf("read tcp: connection reset by peer")
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateModule(ctx, EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]{
		Target: "./child",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if resp.Diagnostics[0].Description().Summary != "Policy plugin crashed" {
		t.Errorf("unexpected summary %q", resp.Diagnostics[0].Description().Summary)
	}
}

// TestClientEvaluateModule_NonCrashError_KeepsGenericMessage verifies that a
// non-transport error on the module path keeps the generic message.
func TestClientEvaluateModule_NonCrashError_KeepsGenericMessage(t *testing.T) {
	ctx := t.Context()

	c := &client{
		client: &stubPolicyClient{
			evaluateModuleFn: func(_ *proto.PolicyEvaluateModuleRequest) (*proto.PolicyEvaluateModuleResponse, error) {
				return nil, grpcstatus.Error(grpccodes.NotFound, "policy set not found")
			},
		},
		callbackRegistry: callback.NewRegistry(),
	}

	resp := c.EvaluateModule(ctx, EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]{
		Target: "./child",
	})

	if resp.Overall != PolicyErrorResult {
		t.Fatalf("Overall = %v, want PolicyErrorResult", resp.Overall)
	}
	if resp.Diagnostics[0].Description().Summary != "Failed to evaluate Terraform Policy" {
		t.Errorf("unexpected summary %q", resp.Diagnostics[0].Description().Summary)
	}
}
