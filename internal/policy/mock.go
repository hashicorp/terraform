// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

var _ Client = (*MockClient)(nil)
var _ RelationshipsClient = (*MockClient)(nil)

// MockClient implements the Client interface, but mocks out all the
// calls for testing purposes.
type MockClient struct {
	mu sync.Mutex

	// Setup method tracking
	SetupCalled   bool
	SetupResponse *SetupResponse
	SetupRequest  SetupRequest
	SetupFn       func(context.Context, SetupRequest) SetupResponse

	// Evaluate method tracking
	EvaluateCalled   bool
	EvaluateResponse *EvaluationResponse
	EvaluateRequest  EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]
	// EvaluateRequests records every EvaluateResource request in call order.
	EvaluateRequests []EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]
	EvaluateFn       func(context.Context, EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) EvaluationResponse

	// EvaluateProvider method tracking
	EvaluateProviderCalled   bool
	EvaluateProviderResponse *EvaluationResponse
	EvaluateProviderRequest  EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]
	EvaluateProviderFn       func(context.Context, EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) EvaluationResponse

	// EvaluateModule method tracking
	EvaluateModuleCalled   bool
	EvaluateModuleResponse *EvaluationResponse
	EvaluateModuleRequest  EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]
	EvaluateModuleFn       func(context.Context, EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) EvaluationResponse

	// Stop method tracking
	StopCalled bool
	StopFn     func()

	// RelationshipsSupportedResponse is returned by RelationshipsSupported.
	// The default (false) means the mock behaves like a policy plugin without
	// the relationships capability.
	RelationshipsSupportedResponse bool

	// BeginRun method tracking
	BeginRunCalled   bool
	BeginRunRequest  *proto.BeginRunRequest
	BeginRunResponse *proto.BeginRunResponse
	BeginRunErr      error
	BeginRunFn       func(context.Context, *proto.BeginRunRequest) (*proto.BeginRunResponse, error)

	// ReportInstances method tracking; every request is recorded in call order.
	ReportInstancesRequests []*proto.ReportInstancesRequest
	ReportInstancesResponse *proto.ReportInstancesResponse
	ReportInstancesErr      error
	ReportInstancesFn       func(context.Context, *proto.ReportInstancesRequest) (*proto.ReportInstancesResponse, error)

	// FinishRun method tracking
	FinishRunCalled   bool
	FinishRunRequest  *proto.FinishRunRequest
	FinishRunResponse *proto.FinishRunResponse
	FinishRunErr      error
	FinishRunFn       func(context.Context, *proto.FinishRunRequest) (*proto.FinishRunResponse, error)
}

func (p *MockClient) beginWrite() func() {
	p.mu.Lock()
	return p.mu.Unlock
}

func (p *MockClient) Setup(ctx context.Context, req SetupRequest) (resp SetupResponse) {
	defer p.beginWrite()()

	p.SetupCalled = true
	p.SetupRequest = req
	if p.SetupFn != nil {
		return p.SetupFn(ctx, req)
	}

	if p.SetupResponse != nil {
		return *p.SetupResponse
	}

	return resp
}

func (p *MockClient) EvaluateResource(ctx context.Context, r EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) (resp EvaluationResponse) {
	defer p.beginWrite()()

	p.EvaluateCalled = true
	p.EvaluateRequest = r
	p.EvaluateRequests = append(p.EvaluateRequests, r)
	if p.EvaluateFn != nil {
		return p.EvaluateFn(ctx, r)
	}

	if p.EvaluateResponse != nil {
		return *p.EvaluateResponse
	}

	return resp
}

func (p *MockClient) EvaluateProvider(ctx context.Context, r EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) (resp EvaluationResponse) {
	defer p.beginWrite()()

	p.EvaluateProviderCalled = true
	p.EvaluateProviderRequest = r
	if p.EvaluateProviderFn != nil {
		return p.EvaluateProviderFn(ctx, r)
	}

	if p.EvaluateProviderResponse != nil {
		return *p.EvaluateProviderResponse
	}

	return resp
}

func (p *MockClient) EvaluateModule(ctx context.Context, r EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) (resp EvaluationResponse) {
	defer p.beginWrite()()

	p.EvaluateModuleCalled = true
	p.EvaluateModuleRequest = r
	if p.EvaluateModuleFn != nil {
		return p.EvaluateModuleFn(ctx, r)
	}

	if p.EvaluateModuleResponse != nil {
		return *p.EvaluateModuleResponse
	}

	return resp
}

func (p *MockClient) Stop() {
	defer p.beginWrite()()
	p.StopCalled = true
	if p.StopFn != nil {
		p.StopFn()
	}
}

func (p *MockClient) RelationshipsSupported() bool {
	defer p.beginWrite()()
	return p.RelationshipsSupportedResponse
}

func (p *MockClient) BeginRun(ctx context.Context, req *proto.BeginRunRequest) (*proto.BeginRunResponse, error) {
	defer p.beginWrite()()

	p.BeginRunCalled = true
	p.BeginRunRequest = req
	if p.BeginRunFn != nil {
		return p.BeginRunFn(ctx, req)
	}
	if p.BeginRunErr != nil {
		return nil, p.BeginRunErr
	}
	if p.BeginRunResponse != nil {
		return p.BeginRunResponse, nil
	}
	return &proto.BeginRunResponse{}, nil
}

func (p *MockClient) ReportInstances(ctx context.Context, req *proto.ReportInstancesRequest) (*proto.ReportInstancesResponse, error) {
	defer p.beginWrite()()

	p.ReportInstancesRequests = append(p.ReportInstancesRequests, req)
	if p.ReportInstancesFn != nil {
		return p.ReportInstancesFn(ctx, req)
	}
	if p.ReportInstancesErr != nil {
		return nil, p.ReportInstancesErr
	}
	if p.ReportInstancesResponse != nil {
		return p.ReportInstancesResponse, nil
	}
	return &proto.ReportInstancesResponse{}, nil
}

func (p *MockClient) FinishRun(ctx context.Context, req *proto.FinishRunRequest) (*proto.FinishRunResponse, error) {
	defer p.beginWrite()()

	p.FinishRunCalled = true
	p.FinishRunRequest = req
	if p.FinishRunFn != nil {
		return p.FinishRunFn(ctx, req)
	}
	if p.FinishRunErr != nil {
		return nil, p.FinishRunErr
	}
	if p.FinishRunResponse != nil {
		return p.FinishRunResponse, nil
	}
	return &proto.FinishRunResponse{}, nil
}
