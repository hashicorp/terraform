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
	gproto "google.golang.org/protobuf/proto"

	"github.com/hashicorp/terraform/internal/configs/configschema"
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
	trees := []*proto.RelationshipTree{{
		ResourceType:   "test_a",
		ProviderSource: "registry.terraform.io/hashicorp/test",
		Relationships: []*proto.Relationship{{
			Name:           "attached",
			KeyType:        "test_a",
			ValueType:      "test_b",
			ProviderSource: "registry.terraform.io/hashicorp/test",
		}},
	}}

	var gotReq *proto.PolicySetupRequest
	c := &client{
		client: &stubPolicyClient{
			setupFn: func(req *proto.PolicySetupRequest) (*proto.PolicySetupResponse, error) {
				gotReq = req
				return &proto.PolicySetupResponse{
					ServerCapabilities: &proto.PolicySetupResponse_ServerCapabilities{
						Relationships:           trees,
						GetresourcesIdentifiers: true,
					},
				}, nil
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
	if len(got.Trees) != 1 || got.Trees[0] != trees[0] {
		t.Fatalf("unexpected relationships: %v", got)
	}
	if !c.identifiers.Load() {
		t.Fatal("expected the client to keep the getresources_identifiers capability")
	}
}

// testObject returns a planned update of test_resource.NAME in generation 1.
func testObject(name string, attrs, prior cty.Value) *callback.Object {
	return &callback.Object{
		Generation: 1,
		Address:    "test_resource." + name,
		Type:       "test_resource",
		Metadata: &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
			Address:   "test_resource." + name,
			Operation: proto.Operation_UPDATE,
		},
		Attrs:      attrs,
		PriorAttrs: prior,
	}
}

func TestClientEvaluateRelationships(t *testing.T) {
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

	val := cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("a")})
	subject := testObject("a", val, cty.NilVal)
	relations := []*proto.Relation{{
		Relationship: "attached",
		Completeness: proto.Completeness_PLAN_COMPLETENESS,
		Resources:    map[string]*proto.RelatedResource{"x": {Operation: proto.Operation_CREATE}},
	}}
	c.EvaluateResource(t.Context(), EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target:        "test_resource",
		Attrs:         CtyToPolicyValue(val),
		Object:        subject,
		Meta:          subject.Metadata,
		Relationships: relations,
	})
	if gotReq == nil {
		t.Fatal("expected EvaluateResource RPC to be called")
	}
	if len(gotReq.Relationships) != 1 || !gproto.Equal(gotReq.Relationships[0], relations[0]) {
		t.Fatalf("unexpected relationships: %v", gotReq.Relationships)
	}
	if gotReq.Identifier != subject.Identifier() {
		t.Fatalf("wrong identifier %q, want %q", gotReq.Identifier, subject.Identifier())
	}

	// The subject's values are those of its identifier.
	resp := c.GetResourcesByIdentifier(t.Context(), []string{gotReq.Identifier})
	if len(resp.Diagnostics) != 0 || len(resp.Resources) != 1 {
		t.Fatalf("unexpected response: %v", resp)
	}
	if !gproto.Equal(resp.Resources[0].Attrs, gotReq.Attrs) || resp.Resources[0].PriorAttrs != nil {
		t.Fatalf("wrong values of the subject: %v", resp.Resources[0])
	}
}

// Each object is encoded once per session: the first registration of an
// identifier wins, and getresources returns identifiers only when the plugin
// accepts them.
func TestClientObjects(t *testing.T) {
	ctx := t.Context()
	marked := cty.ObjectVal(map[string]cty.Value{
		"name":   cty.StringVal("a"),
		"secret": cty.StringVal("s").Mark(marks.Sensitive),
	})
	obj := testObject("a", marked, cty.NilVal)
	unmarked, _ := marked.UnmarkDeep()
	todayResult, err := msgpack.Marshal(unmarked, cty.DynamicPseudoType)
	if err != nil {
		t.Fatal(err)
	}

	for _, identifiers := range []bool{false, true} {
		t.Run(fmt.Sprintf("getresources_identifiers=%t", identifiers), func(t *testing.T) {
			var reqs []*proto.PolicyEvaluateResourceRequest
			registry := callback.NewRegistry()
			c := &client{
				client: &stubPolicyClient{
					setupFn: func(*proto.PolicySetupRequest) (*proto.PolicySetupResponse, error) {
						return &proto.PolicySetupResponse{
							ServerCapabilities: &proto.PolicySetupResponse_ServerCapabilities{GetresourcesIdentifiers: identifiers},
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

			id, err := c.RegisterObject(obj)
			if err != nil {
				t.Fatal(err)
			}
			// Another object with the same identifier stands for the first
			// one's values.
			other := testObject("a", cty.EmptyObjectVal, cty.NilVal)
			c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
				Target: "test_resource",
				Attrs:  CtyToPolicyValue(cty.EmptyObjectVal),
				Object: other,
			})
			want, err := resourceAttributesToProto(CtyToPolicyValue(marked))
			if err != nil {
				t.Fatal(err)
			}
			if reqs[0].Identifier != id || !gproto.Equal(reqs[0].Attrs, want) {
				t.Errorf("wrong subject\ngot:  %v %v\nwant: %v %v", reqs[0].Identifier, reqs[0].Attrs, id, want)
			}
			// The subject has no prior object, so its prior attrs are
			// inline.
			if wantPrior, _ := resourceAttributesToProto(CtyToPolicyValue(cty.EmptyObjectVal)); !gproto.Equal(reqs[0].PriorAttrs, wantPrior) {
				t.Errorf("wrong prior attrs %v", reqs[0].PriorAttrs)
			}

			evalID := registry.NextID()
			registry.Register(evalID, callback.Functions{
				GetResources: func(context.Context, string, cty.Value) ([]callback.Resource, bool, error) {
					return []callback.Resource{{Value: marked, Object: obj}}, false, nil
				},
			})
			nullAttrs, _ := msgpack.Marshal(cty.NullVal(cty.DynamicPseudoType), cty.DynamicPseudoType)
			server := &callback.Server{Registry: registry, Objects: &c.objectTable}
			resources, err := server.GetResources(ctx, &proto.GetResourcesRequest{EvaluationRequestId: evalID, Type: "test_resource", Attributes: nullAttrs})
			if err != nil {
				t.Fatal(err)
			}
			if !identifiers {
				if len(resources.Identifiers) != 0 || len(resources.Results) != 1 || !slices.Equal(resources.Results[0], todayResult) {
					t.Errorf("wrong getresources response: %v", resources)
				}
				return
			}
			if len(resources.Results) != 0 || !slices.Equal(resources.Identifiers, []string{id}) {
				t.Errorf("wrong getresources response: %v", resources)
			}
			byID, err := server.GetResourcesByIdentifier(ctx, &proto.GetResourcesByIdentifierRequest{EvaluationRequestId: evalID, Identifiers: []string{id}})
			if err != nil {
				t.Fatal(err)
			}
			if len(byID.Resources) != 1 || byID.Resources[0].Identifier != id || byID.Resources[0].Type != "test_resource" ||
				!gproto.Equal(byID.Resources[0].Metadata, obj.Metadata) || !gproto.Equal(byID.Resources[0].Attrs, want) {
				t.Errorf("wrong resolved object: %v", byID)
			}
		})
	}
}

// An object's encoding redacts the paths that the schema declares
// sensitive, whether it's first encoded for a subject or for a related
// resource; a subject's inline values do too.
func TestClientObjectSchemaSensitive(t *testing.T) {
	ctx := t.Context()
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"name":   {Type: cty.String, Optional: true},
			"secret": {Type: cty.String, Optional: true, Sensitive: true},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"nested": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"key": {Type: cty.String, Optional: true, Sensitive: true},
					},
				},
			},
		},
	}
	val := cty.ObjectVal(map[string]cty.Value{
		"name":   cty.StringVal("a").Mark(marks.Sensitive),
		"secret": cty.StringVal("s"),
		"nested": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"key": cty.StringVal("k")})}).Mark(marks.Sensitive),
	})
	pathString := func(path *proto.AttributePath) string {
		var ret string
		for _, step := range path.Steps {
			switch s := step.Selector.(type) {
			case *proto.AttributePath_Step_AttributeName:
				ret += "." + s.AttributeName
			case *proto.AttributePath_Step_ElementKeyInt:
				ret += fmt.Sprintf("[%d]", s.ElementKeyInt)
			}
		}
		return ret[1:]
	}
	redacted := func(attrs *proto.ResourceAttributes) []string {
		var ret []string
		for _, path := range attrs.RedactedPaths {
			ret = append(ret, pathString(path))
		}
		return ret
	}

	sends := func(subjectFirst bool) (subject, prior, member *proto.ResourceAttributes) {
		var req *proto.PolicyEvaluateResourceRequest
		c := &client{
			client: &stubPolicyClient{
				evaluateResourceFn: func(r *proto.PolicyEvaluateResourceRequest) (*proto.PolicyEvaluateResourceResponse, error) {
					req = r
					return &proto.PolicyEvaluateResourceResponse{Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT}, nil
				},
			},
			callbackRegistry: callback.NewRegistry(),
		}
		obj := testObject("a", val, cty.NilVal)
		obj.Schema = schema
		evaluate := func() {
			c.EvaluateResource(ctx, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
				Target:     "test_resource",
				Attrs:      CtyToPolicyValue(val),
				PriorAttrs: CtyToPolicyValue(val),
				Object:     obj,
			})
		}
		if subjectFirst {
			evaluate()
		}
		id, err := c.RegisterObject(obj)
		if err != nil {
			t.Fatal(err)
		}
		if !subjectFirst {
			evaluate()
		}
		resp := c.GetResourcesByIdentifier(ctx, []string{id})
		return req.Attrs, req.PriorAttrs, resp.Resources[0].Attrs
	}

	subjectFirst, prior, member := sends(true)
	memberFirst, _, _ := sends(false)
	for _, got := range [][]string{redacted(memberFirst), redacted(prior), redacted(member)} {
		if want := redacted(subjectFirst); !slices.Equal(got, want) {
			t.Fatalf("wrong redacted paths\ngot:  %v\nwant: %v", got, want)
		}
	}
	for _, path := range []string{"name", "secret", "nested", "nested[0].key"} {
		if !slices.Contains(redacted(subjectFirst), path) {
			t.Errorf("redacted paths %v don't include %s", redacted(subjectFirst), path)
		}
	}
}

func TestObjectIdentifier(t *testing.T) {
	obj := callback.Object{Generation: 1, Address: "test_resource.a"}
	if obj.Identifier() != obj.Identifier() {
		t.Fatal("the identifier of an object isn't stable")
	}
	for name, other := range map[string]callback.Object{
		"generation":  {Generation: 2, Address: "test_resource.a"},
		"old object":  {Generation: 1, OldObject: true, Address: "test_resource.a"},
		"deposed key": {Generation: 1, DeposedKey: "00000001", Address: "test_resource.a"},
		"address":     {Generation: 1, Address: "test_resource.b"},
	} {
		if other.Identifier() == obj.Identifier() {
			t.Errorf("identifiers of objects that differ in %s are equal", name)
		}
	}
	// The values aren't part of the identifier.
	withValues := obj
	withValues.Attrs = cty.EmptyObjectVal
	if withValues.Identifier() != obj.Identifier() {
		t.Error("the identifier of an object depends on its values")
	}
}

func TestClientGetResourcesByIdentifier(t *testing.T) {
	var c client
	a := testObject("a", cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("a")}), cty.NilVal)
	b := testObject("b", cty.NilVal, cty.ObjectVal(map[string]cty.Value{"name": cty.StringVal("b")}))
	b.Metadata.Operation = proto.Operation_DELETE
	for _, obj := range []*callback.Object{a, b} {
		if _, err := c.RegisterObject(obj); err != nil {
			t.Fatal(err)
		}
	}

	resp := c.GetResourcesByIdentifier(t.Context(), []string{b.Identifier(), a.Identifier()})
	if len(resp.Diagnostics) != 0 || len(resp.Resources) != 2 {
		t.Fatalf("unexpected response: %v", resp)
	}
	for i, obj := range []*callback.Object{b, a} {
		res := resp.Resources[i]
		attrs := res.Attrs
		if obj == b {
			if res.Attrs != nil {
				t.Errorf("a DELETE has attrs: %v", res)
			}
			attrs = res.PriorAttrs
		} else if res.PriorAttrs != nil {
			t.Errorf("an object without prior attrs has prior attrs: %v", res)
		}
		val, err := msgpack.Unmarshal(attrs.Raw, cty.DynamicPseudoType)
		if err != nil {
			t.Fatal(err)
		}
		if res.Identifier != obj.Identifier() || res.Metadata.Address != obj.Address || val.GetAttr("name").AsString() != obj.Address[len("test_resource."):] {
			t.Errorf("resource %d is %v, want %s", i, res, obj.Address)
		}
	}

	unknown := (&callback.Object{Generation: 1, Address: "test_resource.c"}).Identifier()
	resp = c.GetResourcesByIdentifier(t.Context(), []string{a.Identifier(), unknown})
	if len(resp.Resources) != 0 || len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Severity != proto.Severity_ERROR {
		t.Fatalf("wrong response for an unknown identifier: %v", resp)
	}
}
