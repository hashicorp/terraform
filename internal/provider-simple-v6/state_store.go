// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// simple provider a minimal provider implementation for testing
package simple

import (
	"sort"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states/remote"
)

// we keep the states and locks in package-level variables, so that they can be
// accessed from multiple instances of the backend. This better emulates
// backend instances accessing a single remote data store.
var (
	states stateMap
)

func init() {
	Reset()
}

// Reset clears out all existing state and lock data.
// This is used to initialize the package during init, as well as between
// tests.
func Reset() {
	states = stateMap{
		m: map[string]*remote.State{
			"default": {}, // start with the default workspace existing by default
		},
	}
}

type stateMap struct {
	sync.Mutex
	m map[string]*remote.State
}

func (s simple) ValidateStateStoreConfig(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
	// At this moment there is nothing to configure for the simple provider,
	// so we will happily return without taking any action
	return providers.ValidateStateStoreConfigResponse{}
}

func (s simple) ConfigureStateStore(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
	// At this moment there is nothing to configure for the simple provider,
	// so we will happily return without taking any action
	return providers.ConfigureStateStoreResponse{}
}

func (s simple) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	states.Lock()
	defer states.Unlock()

	var workspaces []string

	for s := range states.m {
		workspaces = append(workspaces, s)
	}

	sort.Strings(workspaces)

	if len(workspaces) == 0 {
		resp := providers.GetStatesResponse{}
		resp.Diagnostics = resp.Diagnostics.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No existing workspaces",
			Detail: `Use the "terraform workspace" command to create and select a new workspace.
If the backend already contains existing workspaces, you may need to update
the backend configuration.`,
		})
		return resp
	}

	return providers.GetStatesResponse{
		States:      workspaces,
		Diagnostics: nil,
	}
}

func (s simple) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	states.Lock()
	defer states.Unlock()

	resp := providers.DeleteStateResponse{}
	if req.StateId == backend.DefaultStateName || req.StateId == "" {
		resp.Diagnostics = resp.Diagnostics.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot delete the default state",
			Detail:   "The default state cannot be deleted by Terraform",
		})
		return resp
	}

	delete(states.m, req.StateId)
	return resp
}
