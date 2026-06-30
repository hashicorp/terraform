// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"iter"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/policy"
)

type PolicyResult interface {
	AddResource(addr addrs.AbsResourceInstance, result policy.EvaluationResponse, config *configs.Resource)
	AddModule(addr addrs.Module, result policy.EvaluationResponse, config *configs.ModuleCall)
	AddProvider(addr addrs.AbsProviderConfig, result policy.EvaluationResponse, config *configs.Provider)
}

// PolicyResults represents the results of policy evaluation of resources, modules, and providers for a single plan.
type PolicyResults struct {
	mu   *sync.Mutex
	set  addrs.Map[addrs.AbsResourceInstance, PolicyEvaluation]
	pset addrs.Map[addrs.AbsProviderConfig, PolicyEvaluation]
	mset addrs.Map[addrs.Module, PolicyEvaluation]
}

// *PolicyResults is the buffered implementation of PolicyResult.
var _ PolicyResult = (*PolicyResults)(nil)

// AsPolicyResult adapts a concrete *PolicyResults to the PolicyResult
// interface, converting a nil pointer into a true nil interface. Use this at
// every concrete->interface boundary so callers' `!= nil` checks stay correct
// and never see a typed-nil.
func AsPolicyResult(pr *PolicyResults) PolicyResult {
	if pr == nil {
		return nil
	}
	return pr
}

// PolicyEvaluation holds the result of a policy evaluation for a single resource, module, or provider.
type PolicyEvaluation struct {
	EvaluationResponse policy.EvaluationResponse
	ConfigDeclRange    hcl.Range
}

func NewPolicyResults() *PolicyResults {
	return &PolicyResults{
		mu:   &sync.Mutex{},
		set:  addrs.MakeMap[addrs.AbsResourceInstance, PolicyEvaluation](),
		pset: addrs.MakeMap[addrs.AbsProviderConfig, PolicyEvaluation](),
		mset: addrs.MakeMap[addrs.Module, PolicyEvaluation](),
	}
}

func (pr *PolicyResults) AddResource(addr addrs.AbsResourceInstance, result policy.EvaluationResponse, config *configs.Resource) {
	// Don't add empty results
	if result.Empty() {
		return
	}
	pr.mu.Lock()
	defer pr.mu.Unlock()
	var rng hcl.Range
	if config != nil {
		rng = config.DeclRange
	}
	pr.set.Put(addr, PolicyEvaluation{EvaluationResponse: result, ConfigDeclRange: rng})
}

func (pr *PolicyResults) AddProvider(addr addrs.AbsProviderConfig, result policy.EvaluationResponse, config *configs.Provider) {
	// Don't add empty results
	if result.Empty() {
		return
	}
	pr.mu.Lock()
	defer pr.mu.Unlock()
	var rng hcl.Range
	if config != nil {
		rng = config.DeclRange
	}
	pr.pset.Put(addr, PolicyEvaluation{EvaluationResponse: result, ConfigDeclRange: rng})
}

func (pr *PolicyResults) AddModule(addr addrs.Module, result policy.EvaluationResponse, config *configs.ModuleCall) {
	// Don't add empty results
	if result.Empty() {
		return
	}
	pr.mu.Lock()
	defer pr.mu.Unlock()
	var rng hcl.Range
	if config != nil {
		rng = config.DeclRange
	}
	pr.mset.Put(addr, PolicyEvaluation{EvaluationResponse: result, ConfigDeclRange: rng})
}

func (pr *PolicyResults) Iter() iter.Seq2[string, PolicyEvaluation] {
	return func(yield func(string, PolicyEvaluation) bool) {
		for k, v := range pr.set.Iter() {
			if !yield(k.String(), v) {
				return
			}
		}
		for k, v := range pr.pset.Iter() {
			if !yield(k.String(), v) {
				return
			}
		}
		for k, v := range pr.mset.Iter() {
			if !yield(k.String(), v) {
				return
			}
		}
	}
}

func (pr *PolicyResults) Len() int {
	if pr == nil {
		return 0
	}
	return pr.set.Len() + pr.pset.Len() + pr.mset.Len()
}
