// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

var _ Client = (*MockClient)(nil)

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
}

func (p *MockClient) Setup(ctx context.Context, req SetupRequest) (resp SetupResponse) {
	p.mu.Lock()
	p.SetupCalled = true
	p.SetupRequest = req
	fn := p.SetupFn
	fixed := p.SetupResponse
	p.mu.Unlock()

	if fn != nil {
		return fn(ctx, req)
	}
	if fixed != nil {
		return *fixed
	}
	return resp
}

func (p *MockClient) EvaluateResource(ctx context.Context, r EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) (resp EvaluationResponse) {
	p.mu.Lock()
	p.EvaluateCalled = true
	p.EvaluateRequest = r
	fn := p.EvaluateFn
	fixed := p.EvaluateResponse
	p.mu.Unlock()

	if fn != nil {
		return fn(ctx, r)
	}
	if fixed != nil {
		return *fixed
	}
	return resp
}

func (p *MockClient) EvaluateProvider(ctx context.Context, r EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) (resp EvaluationResponse) {
	p.mu.Lock()
	p.EvaluateProviderCalled = true
	p.EvaluateProviderRequest = r
	fn := p.EvaluateProviderFn
	fixed := p.EvaluateProviderResponse
	p.mu.Unlock()

	if fn != nil {
		return fn(ctx, r)
	}
	if fixed != nil {
		return *fixed
	}
	return resp
}

func (p *MockClient) EvaluateModule(ctx context.Context, r EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) (resp EvaluationResponse) {
	p.mu.Lock()
	p.EvaluateModuleCalled = true
	p.EvaluateModuleRequest = r
	fn := p.EvaluateModuleFn
	fixed := p.EvaluateModuleResponse
	p.mu.Unlock()

	if fn != nil {
		return fn(ctx, r)
	}
	if fixed != nil {
		return *fixed
	}
	return resp
}

func (p *MockClient) Stop() {
	p.mu.Lock()
	p.StopCalled = true
	fn := p.StopFn
	p.mu.Unlock()

	if fn != nil {
		fn()
	}
}
