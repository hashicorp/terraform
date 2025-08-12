// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package actions

import (
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/providers"
)

// Actions keeps track of action declarations accessible to the context.
// It is used to plan and execute actions in the context of a Terraform configuration.
type Actions struct {
	// Must hold this lock when accessing all fields after this one.
	mu sync.Mutex

	actionInstances addrs.Map[addrs.AbsActionInstance, ActionData]
}

func NewActions() *Actions {
	return &Actions{
		actionInstances: addrs.MakeMap[addrs.AbsActionInstance, ActionData](),
	}
}

type ActionData struct {
	Config       hcl.Body
	Schema       *providers.ActionSchema
	KeyData      instances.RepetitionData
	ProviderAddr addrs.AbsProviderConfig
}

func (a *Actions) AddActionInstance(addr addrs.AbsActionInstance, providerAddr addrs.AbsProviderConfig, config hcl.Body, schema *providers.ActionSchema, keyData instances.RepetitionData) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.actionInstances.Has(addr) {
		panic("action instance already exists: " + addr.String())
	}

	a.actionInstances.Put(addr, ActionData{
		Config:       config,
		Schema:       schema,
		KeyData:      keyData,
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
