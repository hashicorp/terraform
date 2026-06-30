// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"sync"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
)

// streamingPolicyResults is a plans.PolicyResult that renders each result to
// the operation view as soon as it is produced and retains nothing, bounding
// memory regardless of plan size. It is the workspace (CLI) sink; stacks
// continues to use the buffered *plans.PolicyResults.
// policyResultStreamer is the minimal view capability the streaming sink
// needs: rendering a single policy result immediately. Both views.Operation
// (plan and apply) and views.Init (init) satisfy it.
type policyResultStreamer interface {
	StreamPolicyResult(addr string, result plans.PolicyEvaluation)
}

type streamingPolicyResults struct {
	view policyResultStreamer
	mu   sync.Mutex // the graph walk / installer calls Add* concurrently
}

var _ plans.PolicyResult = (*streamingPolicyResults)(nil)

// NewStreamingPolicyResults returns a streaming policy-results sink backed by
// the given view. Every AddResource/AddModule/AddProvider call is rendered
// immediately and dropped, so nothing is retained.
func NewStreamingPolicyResults(view policyResultStreamer) plans.PolicyResult {
	return &streamingPolicyResults{view: view}
}

func (s *streamingPolicyResults) AddResource(addr addrs.AbsResourceInstance, result policy.EvaluationResponse, config *configs.Resource) {
	var rng hcl.Range
	if config != nil {
		rng = config.DeclRange
	}
	s.emit(addr.String(), result, rng)
}

func (s *streamingPolicyResults) AddModule(addr addrs.Module, result policy.EvaluationResponse, config *configs.ModuleCall) {
	var rng hcl.Range
	if config != nil {
		rng = config.DeclRange
	}
	s.emit(addr.String(), result, rng)
}

func (s *streamingPolicyResults) AddProvider(addr addrs.AbsProviderConfig, result policy.EvaluationResponse, config *configs.Provider) {
	var rng hcl.Range
	if config != nil {
		rng = config.DeclRange
	}
	s.emit(addr.String(), result, rng)
}

func (s *streamingPolicyResults) emit(addr string, result policy.EvaluationResponse, rng hcl.Range) {
	if result.Empty() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.view.StreamPolicyResult(addr, plans.PolicyEvaluation{
		EvaluationResponse: result,
		ConfigDeclRange:    rng,
	})
}
