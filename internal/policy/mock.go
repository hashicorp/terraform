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
	EvaluateRequest  EvaluationRequest[*proto.ResourceMetadata]
	EvaluateFn       func(context.Context, EvaluationRequest[*proto.ResourceMetadata]) EvaluationResponse

	// EvaluateProvider method tracking
	EvaluateProviderCalled   bool
	EvaluateProviderResponse *EvaluationResponse
	EvaluateProviderRequest  EvaluationRequest[*proto.ProviderMetadata]
	EvaluateProviderFn       func(context.Context, EvaluationRequest[*proto.ProviderMetadata]) EvaluationResponse

	// EvaluateModule method tracking
	EvaluateModuleCalled   bool
	EvaluateModuleResponse *EvaluationResponse
	EvaluateModuleRequest  EvaluationRequest[*proto.ModuleMetadata]
	EvaluateModuleFn       func(context.Context, EvaluationRequest[*proto.ModuleMetadata]) EvaluationResponse

	// Stop method tracking
	StopCalled bool
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

func (p *MockClient) Evaluate(ctx context.Context, r EvaluationRequest[*proto.ResourceMetadata]) (resp EvaluationResponse) {
	defer p.beginWrite()()

	p.EvaluateCalled = true
	p.EvaluateRequest = r
	if p.EvaluateFn != nil {
		return p.EvaluateFn(ctx, r)
	}

	if p.EvaluateResponse != nil {
		return *p.EvaluateResponse
	}

	return resp
}

func (p *MockClient) EvaluateProvider(ctx context.Context, r EvaluationRequest[*proto.ProviderMetadata]) (resp EvaluationResponse) {
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

func (p *MockClient) EvaluateModule(ctx context.Context, r EvaluationRequest[*proto.ModuleMetadata]) (resp EvaluationResponse) {
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
}
