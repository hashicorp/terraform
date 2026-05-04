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

// PolicyResults represents the results of policy evaluation of resources and providers for a single plan.
type PolicyResults struct {
	mu *sync.Mutex
	// Diagnostics holds diagnostics not tied to any policy
	Diagnostics policy.Diagnostics
	set         addrs.Map[addrs.AbsResourceInstance, PolicyEvaluation]
	pset        addrs.Map[addrs.AbsProviderConfig, PolicyEvaluation]
	mset        addrs.Map[addrs.Module, PolicyEvaluation]
}

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
	// Don't add implicitly allowed resources
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
	// Don't add implicitly allowed providers
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
	// Don't add implicitly allowed modules
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
