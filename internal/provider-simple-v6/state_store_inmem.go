// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package simple

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

const inMemStoreName = "simple6_inmem"

// InMemStoreSingle allows 'storing' state in memory for the purpose of testing.
//
// "Single" reflects the fact that this implementation does not use any global scope vars
// in its implementation, unlike the current inmem backend. HOWEVER, you can test whether locking
// blocks multiple clients trying to access the same state at once by creating multiple instances
// of backend.Backend that wrap the same provider.Interface instance.
type InMemStoreSingle struct {
	states stateMap
	locks  lockMap

	chunkSize int
}

var _ providers.StateStoreChunkSizeSetter = &InMemStoreSingle{}

func stateStoreInMemGetSchema() providers.Schema {
	return providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"lock_id": {
					Type:        cty.String,
					Optional:    true,
					Description: "initializes the state in a locked configuration",
				},
			},
		},
	}
}

func (m *InMemStoreSingle) ValidateStateStoreConfig(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
	var resp providers.ValidateStateStoreConfigResponse

	attrs := req.Config.AsValueMap()

	// This is completely arbitrary validation included here to avoid this method being empty. It is not here for a purpose,
	// but could be used if an E2E test wants to trigger a validation error.
	if v, ok := attrs["lock_id"]; ok && !v.IsNull() {
		cutoff := cty.NumberVal(big.NewFloat(3))
		if v.Length().LessThan(cutoff) == cty.True {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("when set, the attribute \"lock_id\" must have a length equal or greater than %s", cutoff.AsString()))
			return resp
		}
	}

	return resp
}

func (m *InMemStoreSingle) ConfigureStateStore(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
	resp := providers.ConfigureStateStoreResponse{}

	m.states.Lock()
	defer m.states.Unlock()

	// set the default client lock info per the test config
	configVal := req.Config
	if v := configVal.GetAttr("lock_id"); !v.IsNull() {
		m.locks.lock(backend.DefaultStateName, v.AsString())
	}

	// We need to return a suggested chunk size; use the value suggested by Core
	resp.Capabilities.ChunkSize = req.Capabilities.ChunkSize
	return resp
}

func (m *InMemStoreSingle) ReadStateBytes(req providers.ReadStateBytesRequest) providers.ReadStateBytesResponse {
	resp := providers.ReadStateBytesResponse{}

	v, ok := m.states.m[req.StateId]
	if !ok {
		// Does not exist, so return no bytes

		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"State doesn't exist",
			fmt.Sprintf("The %q state does not exist", req.StateId),
		))
		return resp
	}

	resp.Bytes = v
	return resp
}

func (m *InMemStoreSingle) WriteStateBytes(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
	resp := providers.WriteStateBytesResponse{}

	if m.states.m == nil {
		m.states.m = make(map[string][]byte, 1)
	}

	m.states.m[req.StateId] = req.Bytes

	return resp
}

func (m *InMemStoreSingle) LockState(req providers.LockStateRequest) providers.LockStateResponse {
	resp := providers.LockStateResponse{}

	lockIdBytes, err := uuid.GenerateUUID()
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error creating random lock uuid: %w", err))
		return resp
	}

	lockId := string(lockIdBytes)
	returnedLockId, err := m.locks.lock(req.StateId, lockId)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
	}

	resp.LockId = string(returnedLockId)
	return resp
}

func (m *InMemStoreSingle) UnlockState(req providers.UnlockStateRequest) providers.UnlockStateResponse {
	resp := providers.UnlockStateResponse{}

	err := m.locks.unlock(req.StateId, req.LockId)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error when unlocking state %q: %w", req.StateId, err))
		return resp
	}

	return resp
}

func (m *InMemStoreSingle) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	m.states.Lock()
	defer m.states.Unlock()

	resp := providers.GetStatesResponse{}

	var stateIds []string

	for s := range m.states.m {
		stateIds = append(stateIds, s)
	}

	sort.Strings(stateIds)
	resp.States = stateIds
	return resp
}

func (m *InMemStoreSingle) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	m.states.Lock()
	defer m.states.Unlock()

	resp := providers.DeleteStateResponse{}

	if req.StateId == backend.DefaultStateName || req.StateId == "" {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("can't delete default state"))
		return resp
	}

	delete(m.states.m, req.StateId)
	return resp
}

func (m *InMemStoreSingle) SetStateStoreChunkSize(typeName string, size int) {
	if typeName != inMemStoreName {
		// If we hit this code it suggests someone's refactoring the PSS implementations used for testing
		panic(fmt.Sprintf("calling code tried to set the state store size on %s state store but the request reached the %s store implementation.",
			typeName,
			inMemStoreName,
		))
	}

	m.chunkSize = size
}

type stateMap struct {
	sync.Mutex
	m map[string][]byte // key=state id, value=state
}

type lockMap struct {
	sync.Mutex
	m map[string]string // key=state id, value=lock_id
}

func (l *lockMap) lock(name string, lockId string) (string, error) {
	l.Lock()
	defer l.Unlock()

	lock, ok := l.m[name]
	if ok {
		// Error; lock already exists for that state id
		return "", fmt.Errorf("state %q is already locked with lock id %q", name, lock)
	}

	if l.m == nil {
		l.m = make(map[string]string, 1)
	}

	l.m[name] = lockId

	return lockId, nil
}

func (l *lockMap) unlock(name, id string) error {
	l.Lock()
	defer l.Unlock()

	lockId, ok := l.m[name]

	if !ok {
		return errors.New("state not locked")
	}

	if id != lockId {
		return fmt.Errorf("invalid lock id: state %q was locked with lock id %q, but tried to unlock with lock id %q",
			name,
			lockId,
			id,
		)
	}

	delete(l.m, name)
	return nil
}
