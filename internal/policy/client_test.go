// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	gproto "google.golang.org/protobuf/proto"

	"github.com/hashicorp/terraform/internal/lang/marks"
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

func TestClientSetupRelationships(t *testing.T) {
	defs := []*proto.RelationshipDefinition{{
		ProviderSource: "registry.terraform.io/hashicorp/test",
		Name:           "attached",
		KeyResource:    "test_a",
		ValueResource:  "test_b",
	}}
	usages := []*proto.RelationshipUsage{{
		ProviderSource: "registry.terraform.io/hashicorp/test",
		Type:           "test_a",
		Hops:           []*proto.RelationshipHop{{Relationship: "attached"}},
	}}

	var gotReq *proto.PolicySetupRequest
	c := &client{
		client: &stubPolicyClient{
			setupFn: func(req *proto.PolicySetupRequest) (*proto.PolicySetupResponse, error) {
				gotReq = req
				return &proto.PolicySetupResponse{RelationshipDefinitions: defs, RelationshipUsages: usages}, nil
			},
		},
	}

	resp := c.Setup(t.Context(), SetupRequest{SourceLocations: []string{"./policies"}})
	if resp.Diagnostics.HasErrors() {
		t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
	}
	if !gotReq.GetClientCapabilities().GetRelationships() {
		t.Fatal("expected the client to announce the relationships capability")
	}
	got := c.Relationships()
	if len(got.Definitions) != 1 || got.Definitions[0] != defs[0] || len(got.Usages) != 1 || got.Usages[0] != usages[0] {
		t.Fatalf("unexpected relationships: %v", got)
	}
	if len(resp.Relationships.Definitions) != 1 || len(resp.Relationships.Usages) != 1 {
		t.Fatalf("unexpected response relationships: %v", resp.Relationships)
	}
}

func TestClientEvaluateRelated(t *testing.T) {
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

	related := []*proto.RelatedMembers{{
		Relationship: "attached",
		Completeness: proto.Completeness_PLAN_COMPLETENESS,
		Members:      []*proto.RelatedMember{{Address: "test_b.x", Type: "test_b"}},
	}}
	c.EvaluateResource(t.Context(), EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target:  "test_a",
		Meta:    &proto.PolicyEvaluateResourceRequest_ResourceMetadata{Address: "test_a.y"},
		Related: related,
	})
	if gotReq == nil {
		t.Fatal("expected EvaluateResource RPC to be called")
	}
	if len(gotReq.Related) != 1 || !gproto.Equal(gotReq.Related[0], related[0]) {
		t.Fatalf("unexpected related: %v", gotReq.Related)
	}
}

func TestClientValueIDs(t *testing.T) {
	ctx := t.Context()
	marked := cty.ObjectVal(map[string]cty.Value{
		"name":   cty.StringVal("a"),
		"secret": cty.StringVal("s").Mark(marks.Sensitive),
	})
	src := callback.ValueSource{Generation: 1, Kind: callback.KindPlanned, Address: "test_resource.a"}
	// The encodings before value IDs: an inline value, and a getresources
	// result.
	today, err := resourceAttributesToProto(CtyToPolicyValue(marked))
	if err != nil {
		t.Fatal(err)
	}
	unmarked, _ := marked.UnmarkDeep()
	todayResult, err := msgpack.Marshal(unmarked, cty.DynamicPseudoType)
	if err != nil {
		t.Fatal(err)
	}
	bytes := func(t *testing.T, m gproto.Message) []byte {
		t.Helper()
		b, err := gproto.MarshalOptions{Deterministic: true}.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	for _, ids := range []bool{false, true} {
		t.Run(fmt.Sprintf("resource_value_ids=%t", ids), func(t *testing.T) {
			var reqs []*proto.PolicyEvaluateResourceRequest
			registry := callback.NewRegistry()
			c := &client{
				client: &stubPolicyClient{
					setupFn: func(*proto.PolicySetupRequest) (*proto.PolicySetupResponse, error) {
						return &proto.PolicySetupResponse{
							ServerCapabilities: &proto.PolicySetupResponse_ServerCapabilities{ResourceValueIds: ids},
						}, nil
					},
					evaluateResourceFn: func(req *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
						reqs = append(reqs, req)
						return &proto.PolicyEvaluateResourceResponse{Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT}, nil
					},
				},
				callbackRegistry: registry,
			}
			c.Setup(ctx, SetupRequest{})

			// The second send of the source carries the first encoding, even
			// with another value. The missing prior value has no source.
			for _, val := range []cty.Value{marked, cty.EmptyObjectVal} {
				c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
					Target:           "test_resource",
					Attrs:            CtyToPolicyValue(val),
					AttrsSource:      &src,
					PriorAttrsSource: &src,
				})
			}
			wantInline := gproto.Clone(today).(*proto.ResourceAttributes)
			if ids {
				wantInline.Id = src.ID()
			}
			for i, req := range reqs {
				if got, want := bytes(t, req.Attrs), bytes(t, wantInline); !slices.Equal(got, want) {
					t.Errorf("request %d: wrong attrs\ngot:  %v\nwant: %v", i, req.Attrs, wantInline)
				}
				if req.PriorAttrs.Id != "" {
					t.Errorf("request %d: prior attrs without a value have ID %q", i, req.PriorAttrs.Id)
				}
			}

			member, err := c.ResourceAttributes(cty.EmptyObjectVal, src)
			if err != nil {
				t.Fatal(err)
			}
			id := registry.NextID()
			registry.Register(id, callback.Functions{
				GetResources: func(context.Context, string, cty.Value) ([]callback.Resource, bool, error) {
					return []callback.Resource{{Value: cty.EmptyObjectVal, Source: src}}, false, nil
				},
			})
			nullAttrs, _ := msgpack.Marshal(cty.NullVal(cty.DynamicPseudoType), cty.DynamicPseudoType)
			server := &callback.Server{Registry: registry, Values: &c.valueTable}
			resources, err := server.GetResources(ctx, &proto.GetResourcesRequest{EvaluationRequestId: id, Type: "test_resource", Attributes: nullAttrs})
			if err != nil {
				t.Fatal(err)
			}

			byRef := &proto.ResourceAttributes{Id: src.ID()}
			if !ids {
				if got, want := bytes(t, member), bytes(t, today); !slices.Equal(got, want) {
					t.Errorf("wrong member attrs\ngot:  %v\nwant: %v", member, today)
				}
				if len(resources.Values) != 0 || len(resources.Results) != 1 || !slices.Equal(resources.Results[0], todayResult) {
					t.Errorf("wrong getresources response: %v", resources)
				}
				return
			}
			if !gproto.Equal(member, byRef) {
				t.Errorf("wrong member attrs\ngot:  %v\nwant: %v", member, byRef)
			}
			if len(resources.Results) != 0 || len(resources.Values) != 1 || !gproto.Equal(resources.Values[0], byRef) {
				t.Errorf("wrong getresources response: %v", resources)
			}
			values, err := server.GetResourceValues(ctx, &proto.GetResourceValuesRequest{Ids: []string{src.ID()}})
			if err != nil {
				t.Fatal(err)
			}
			if got, want := bytes(t, values.Values[0]), bytes(t, wantInline); !slices.Equal(got, want) {
				t.Errorf("wrong resolved value\ngot:  %v\nwant: %v", values.Values[0], wantInline)
			}
		})
	}
}

func TestValueSourceID(t *testing.T) {
	src := callback.ValueSource{Generation: 1, Kind: callback.KindPrior, Address: "test_resource.a"}
	if src.ID() != src.ID() {
		t.Fatal("the ID of a source isn't stable")
	}
	for name, other := range map[string]callback.ValueSource{
		"generation":  {Generation: 2, Kind: callback.KindPrior, Address: "test_resource.a"},
		"kind":        {Generation: 1, Kind: callback.KindState, Address: "test_resource.a"},
		"deposed key": {Generation: 1, Kind: callback.KindPrior, DeposedKey: "00000001", Address: "test_resource.a"},
		"address":     {Generation: 1, Kind: callback.KindPrior, Address: "component.x.test_resource.a"},
	} {
		if other.ID() == src.ID() {
			t.Errorf("IDs of sources that differ in %s are equal", name)
		}
	}
}

func TestClientGetResourceValues(t *testing.T) {
	var c client
	c.ids.Store(true)
	a := callback.ValueSource{Kind: callback.KindState, Address: "test_resource.a"}
	b := callback.ValueSource{Kind: callback.KindState, Address: "test_resource.b"}
	for _, src := range []callback.ValueSource{a, b} {
		if _, err := c.ResourceAttributes(cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal(src.Address)}), src); err != nil {
			t.Fatal(err)
		}
	}

	values, err := c.GetResourceValues(t.Context(), []string{b.ID(), a.ID()})
	if err != nil {
		t.Fatal(err)
	}
	for i, src := range []callback.ValueSource{b, a} {
		val, err := msgpack.Unmarshal(values[i].Raw, cty.DynamicPseudoType)
		if err != nil {
			t.Fatal(err)
		}
		if values[i].Id != src.ID() || val.GetAttr("name").AsString() != src.Address {
			t.Errorf("value %d is %v, want the value of %s", i, values[i], src.Address)
		}
	}

	unknown := callback.ValueSource{Kind: callback.KindState, Address: "test_resource.c"}.ID()
	if _, err := c.GetResourceValues(t.Context(), []string{a.ID(), unknown}); status.Code(err) != codes.NotFound {
		t.Fatalf("wrong error for an unknown ID: %v", err)
	}
}
