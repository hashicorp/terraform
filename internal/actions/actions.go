// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package actions

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// Actions keeps track of action declarations accessible to the context.
// It is used to plan and execute actions in the context of a Terraform configuration.
type Actions struct {
	// Must hold this lock when accessing all fields after this one.
	mu sync.Mutex

	actionInstances        addrs.Map[addrs.AbsActionInstance, ActionData]
	partialExpandedActions addrs.Map[addrs.PartialExpandedAction, ActionData]
}

func NewActions() *Actions {
	return &Actions{
		actionInstances:        addrs.MakeMap[addrs.AbsActionInstance, ActionData](),
		partialExpandedActions: addrs.MakeMap[addrs.PartialExpandedAction, ActionData](),
	}
}

type ActionData struct {
	ConfigValue  cty.Value
	ProviderAddr addrs.AbsProviderConfig
}

func (a *Actions) AddActionInstance(addr addrs.AbsActionInstance, configValue cty.Value, providerAddr addrs.AbsProviderConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.actionInstances.Has(addr) {
		panic("action instance already exists: " + addr.String())
	}

	a.actionInstances.Put(addr, ActionData{
		ConfigValue:  configValue,
		ProviderAddr: providerAddr,
	})
}

func (a *Actions) GetActionInstance(addr addrs.AbsActionInstance) (*ActionData, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, ok := a.actionInstances.GetOk(addr)

	if !ok {
		return nil, false
	}

	return &data, true
}

func (a *Actions) GetActionInstanceKeys(addr addrs.AbsAction) []addrs.AbsActionInstance {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := []addrs.AbsActionInstance{}
	for _, data := range a.actionInstances.Elements() {
		if data.Key.ContainingAction().Equal(addr) {
			result = append(result, data.Key)
		}
	}

	return result
}

func (a *Actions) AddPartialExpandedAction(addr addrs.PartialExpandedAction, configValue cty.Value, providerAddr addrs.AbsProviderConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.partialExpandedActions.Has(addr) {
		panic("action instance already exists: " + addr.String())
	}

	a.partialExpandedActions.Put(addr, ActionData{
		ConfigValue:  configValue,
		ProviderAddr: providerAddr,
	})
}

func (a *Actions) GetPartialExpandedAction(addr addrs.PartialExpandedAction) (*ActionData, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, ok := a.partialExpandedActions.GetOk(addr)

	if !ok {
		return nil, false
	}

	return &data, true
}
