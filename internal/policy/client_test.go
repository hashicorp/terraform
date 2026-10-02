// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/grpc"
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
	beginRunFn         func(*proto.BeginRunRequest) (*proto.BeginRunResponse, error)
	reportInstancesFn  func(*proto.ReportInstancesRequest) (*proto.ReportInstancesResponse, error)
	finishRunFn        func(*proto.FinishRunRequest) (*proto.FinishRunResponse, error)
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

func (s *stubPolicyClient) BeginRun(ctx context.Context, req *proto.BeginRunRequest, _ ...grpc.CallOption) (*proto.BeginRunResponse, error) {
	return s.beginRunFn(req)
}

func (s *stubPolicyClient) ReportInstances(ctx context.Context, req *proto.ReportInstancesRequest, _ ...grpc.CallOption) (*proto.ReportInstancesResponse, error) {
	return s.reportInstancesFn(req)
}

func (s *stubPolicyClient) FinishRun(ctx context.Context, req *proto.FinishRunRequest, _ ...grpc.CallOption) (*proto.FinishRunResponse, error) {
	return s.finishRunFn(req)
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

func TestClientSetupRelationshipsCapability(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name     string
		response *proto.PolicySetupResponse
		want     bool
	}{
		{
			name:     "server capabilities absent",
			response: &proto.PolicySetupResponse{},
			want:     false,
		},
		{
			name: "server does not announce relationships",
			response: &proto.PolicySetupResponse{
				ServerCapabilities: &proto.PolicySetupResponse_ServerCapabilities{},
			},
			want: false,
		},
		{
			name: "server announces relationships",
			response: &proto.PolicySetupResponse{
				ServerCapabilities: &proto.PolicySetupResponse_ServerCapabilities{Relationships: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq *proto.PolicySetupRequest
			c := &client{
				client: &stubPolicyClient{
					setupFn: func(req *proto.PolicySetupRequest) (*proto.PolicySetupResponse, error) {
						gotReq = req
						return tt.response, nil
					},
				},
			}

			if c.RelationshipsSupported() {
				t.Fatal("expected relationships to be unsupported before Setup")
			}

			resp := c.Setup(ctx, SetupRequest{SourceLocations: []string{"./policies"}})
			if resp.Diagnostics.HasErrors() {
				t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
			}
			if gotReq == nil {
				t.Fatal("expected a setup request to be sent")
			}
			if !gotReq.GetClientCapabilities().GetRelationships() {
				t.Fatal("expected the client to announce the relationships capability")
			}
			if got := c.RelationshipsSupported(); got != tt.want {
				t.Fatalf("unexpected RelationshipsSupported: got %t, want %t", got, tt.want)
			}
		})
	}
}

func TestClientSetupRelationshipsCapability_error(t *testing.T) {
	c := &client{
		client: &stubPolicyClient{
			setupFn: func(req *proto.PolicySetupRequest) (*proto.PolicySetupResponse, error) {
				return nil, errors.New("boom")
			},
		},
	}

	resp := c.Setup(t.Context(), SetupRequest{})
	if !resp.Diagnostics.HasErrors() {
		t.Fatal("expected setup error diagnostics")
	}
	if c.RelationshipsSupported() {
		t.Fatal("expected relationships to be unsupported after a failed Setup")
	}
}

func TestClientEvaluateRunID(t *testing.T) {
	for _, runID := range []string{"", "8c5d2b5e-6f43-4b0e-9a8e-1f2d3c4b5a69"} {
		t.Run(fmt.Sprintf("run_id=%q", runID), func(t *testing.T) {
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

			meta := &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
				ProviderType:   "test",
				Address:        "module.child.test_resource.a[0]",
				ProviderSource: "registry.terraform.io/hashicorp/test",
			}
			c.EvaluateResource(t.Context(), EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
				Target: "test_resource",
				Meta:   meta,
				RunID:  runID,
			})
			if gotReq == nil {
				t.Fatal("expected EvaluateResource RPC to be called")
			}
			if gotReq.RunId != runID {
				t.Fatalf("unexpected run_id: got %q, want %q", gotReq.RunId, runID)
			}
			if !gproto.Equal(gotReq.Metadata, meta) {
				t.Fatalf("unexpected metadata: got %v, want %v", gotReq.Metadata, meta)
			}
		})
	}
}

func TestClientRelationshipRPCs(t *testing.T) {
	ctx := t.Context()
	errBoom := errors.New("boom")

	t.Run("BeginRun", func(t *testing.T) {
		req := &proto.BeginRunRequest{
			RunId:   "run",
			Stage:   proto.EvaluationStage_PLAN_EVALUATION_STAGE,
			Runtime: proto.RunRuntime_CLI_RUN_RUNTIME,
		}
		want := &proto.BeginRunResponse{Spec: &proto.CollectionSpec{
			Types: []*proto.TypeSpec{{ProviderSource: "registry.terraform.io/hashicorp/test", Type: "test_resource"}},
		}}
		var gotReq *proto.BeginRunRequest
		c := &client{client: &stubPolicyClient{
			beginRunFn: func(r *proto.BeginRunRequest) (*proto.BeginRunResponse, error) {
				gotReq = r
				return want, nil
			},
		}}
		got, err := c.BeginRun(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if gotReq != req {
			t.Fatalf("request was not passed through: %v", gotReq)
		}
		if got != want {
			t.Fatalf("response was not passed through: %v", got)
		}

		c = &client{client: &stubPolicyClient{
			beginRunFn: func(r *proto.BeginRunRequest) (*proto.BeginRunResponse, error) { return nil, errBoom },
		}}
		if _, err := c.BeginRun(ctx, req); !errors.Is(err, errBoom) {
			t.Fatalf("expected error %q, got %v", errBoom, err)
		}
	})

	t.Run("ReportInstances", func(t *testing.T) {
		req := &proto.ReportInstancesRequest{
			RunId:   "run",
			Records: []*proto.InstanceRecord{{Address: "test_resource.a"}},
		}
		want := &proto.ReportInstancesResponse{}
		var gotReq *proto.ReportInstancesRequest
		c := &client{client: &stubPolicyClient{
			reportInstancesFn: func(r *proto.ReportInstancesRequest) (*proto.ReportInstancesResponse, error) {
				gotReq = r
				return want, nil
			},
		}}
		got, err := c.ReportInstances(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if gotReq != req {
			t.Fatalf("request was not passed through: %v", gotReq)
		}
		if got != want {
			t.Fatalf("response was not passed through: %v", got)
		}

		c = &client{client: &stubPolicyClient{
			reportInstancesFn: func(r *proto.ReportInstancesRequest) (*proto.ReportInstancesResponse, error) { return nil, errBoom },
		}}
		if _, err := c.ReportInstances(ctx, req); !errors.Is(err, errBoom) {
			t.Fatalf("expected error %q, got %v", errBoom, err)
		}
	})

	t.Run("FinishRun", func(t *testing.T) {
		req := &proto.FinishRunRequest{RunId: "run"}
		want := &proto.FinishRunResponse{}
		var gotReq *proto.FinishRunRequest
		c := &client{client: &stubPolicyClient{
			finishRunFn: func(r *proto.FinishRunRequest) (*proto.FinishRunResponse, error) {
				gotReq = r
				return want, nil
			},
		}}
		got, err := c.FinishRun(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if gotReq != req {
			t.Fatalf("request was not passed through: %v", gotReq)
		}
		if got != want {
			t.Fatalf("response was not passed through: %v", got)
		}

		c = &client{client: &stubPolicyClient{
			finishRunFn: func(r *proto.FinishRunRequest) (*proto.FinishRunResponse, error) { return nil, errBoom },
		}}
		if _, err := c.FinishRun(ctx, req); !errors.Is(err, errBoom) {
			t.Fatalf("expected error %q, got %v", errBoom, err)
		}
	})
}

func TestMockClientRelationships(t *testing.T) {
	ctx := t.Context()
	m := NewTestMockClient(t)

	var _ RelationshipsClient = m
	if m.RelationshipsSupported() {
		t.Fatal("expected the mock client not to support relationships by default")
	}
	m.RelationshipsSupportedResponse = true
	if !m.RelationshipsSupported() {
		t.Fatal("expected the mock client to support relationships")
	}

	resp, err := m.BeginRun(ctx, &proto.BeginRunRequest{RunId: "run"})
	if err != nil || resp == nil {
		t.Fatalf("expected an empty response, got %v, %v", resp, err)
	}
	if !m.BeginRunCalled || m.BeginRunRequest.GetRunId() != "run" {
		t.Fatalf("BeginRun was not recorded: %v", m.BeginRunRequest)
	}

	for _, id := range []string{"a", "b"} {
		if _, err := m.ReportInstances(ctx, &proto.ReportInstancesRequest{RunId: id}); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
	if len(m.ReportInstancesRequests) != 2 || m.ReportInstancesRequests[1].GetRunId() != "b" {
		t.Fatalf("ReportInstances calls were not recorded: %v", m.ReportInstancesRequests)
	}

	m.FinishRunErr = errors.New("boom")
	if _, err := m.FinishRun(ctx, &proto.FinishRunRequest{RunId: "run"}); err == nil {
		t.Fatal("expected FinishRunErr to be returned")
	}
	if !m.FinishRunCalled || m.FinishRunRequest.GetRunId() != "run" {
		t.Fatalf("FinishRun was not recorded: %v", m.FinishRunRequest)
	}

	for _, target := range []string{"a", "b"} {
		m.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{Target: target})
	}
	if len(m.EvaluateRequests) != 2 || m.EvaluateRequests[0].Target != "a" || m.EvaluateRequest.Target != "b" {
		t.Fatalf("EvaluateResource calls were not recorded: %v", m.EvaluateRequests)
	}
}
