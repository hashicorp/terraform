// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

// MockHook is an implementation of Hook that can be used for tests.
// It records all of its function calls.
type MockHook struct {
	sync.Mutex

	PreApplyCalled       bool
	PreApplyAddr         addrs.AbsResourceInstance
	PreApplyGen          addrs.DeposedKey
	PreApplyAction       plans.Action
	PreApplyPriorState   cty.Value
	PreApplyPlannedState cty.Value
	PreApplyReturn       HookAction
	PreApplyError        error

	PostApplyCalled      bool
	PostApplyAddr        addrs.AbsResourceInstance
	PostApplyGen         addrs.DeposedKey
	PostApplyNewState    cty.Value
	PostApplyError       error
	PostApplyReturn      HookAction
	PostApplyReturnError error
	PostApplyFn          func(addrs.AbsResourceInstance, addrs.DeposedKey, cty.Value, error) (HookAction, error)

	PreDiffCalled        bool
	PreDiffAddr          addrs.AbsResourceInstance
	PreDiffGen           addrs.DeposedKey
	PreDiffPriorState    cty.Value
	PreDiffProposedState cty.Value
	PreDiffReturn        HookAction
	PreDiffError         error

	PostDiffCalled       bool
	PostDiffAddr         addrs.AbsResourceInstance
	PostDiffGen          addrs.DeposedKey
	PostDiffAction       plans.Action
	PostDiffPriorState   cty.Value
	PostDiffPlannedState cty.Value
	PostDiffReturn       HookAction
	PostDiffError        error

	PreProvisionInstanceCalled bool
	PreProvisionInstanceAddr   addrs.AbsResourceInstance
	PreProvisionInstanceState  cty.Value
	PreProvisionInstanceReturn HookAction
	PreProvisionInstanceError  error

	PostProvisionInstanceCalled bool
	PostProvisionInstanceAddr   addrs.AbsResourceInstance
	PostProvisionInstanceState  cty.Value
	PostProvisionInstanceReturn HookAction
	PostProvisionInstanceError  error

	PreProvisionInstanceStepCalled          bool
	PreProvisionInstanceStepAddr            addrs.AbsResourceInstance
	PreProvisionInstanceStepProvisionerType string
	PreProvisionInstanceStepReturn          HookAction
	PreProvisionInstanceStepError           error

	PostProvisionInstanceStepCalled          bool
	PostProvisionInstanceStepAddr            addrs.AbsResourceInstance
	PostProvisionInstanceStepProvisionerType string
	PostProvisionInstanceStepErrorArg        error
	PostProvisionInstanceStepReturn          HookAction
	PostProvisionInstanceStepError           error

	ProvisionOutputCalled          bool
	ProvisionOutputAddr            addrs.AbsResourceInstance
	ProvisionOutputProvisionerType string
	ProvisionOutputMessage         string

	PreRefreshCalled     bool
	PreRefreshAddr       addrs.AbsResourceInstance
	PreRefreshGen        addrs.DeposedKey
	PreRefreshPriorState cty.Value
	PreRefreshReturn     HookAction
	PreRefreshError      error

	PostRefreshCalled     bool
	PostRefreshAddr       addrs.AbsResourceInstance
	PostRefreshGen        addrs.DeposedKey
	PostRefreshPriorState cty.Value
	PostRefreshNewState   cty.Value
	PostRefreshReturn     HookAction
	PostRefreshError      error

	PreImportStateCalled bool
	PreImportStateAddr   addrs.AbsResourceInstance
	PreImportStateID     string
	PreImportStateReturn HookAction
	PreImportStateError  error

	PostImportStateCalled    bool
	PostImportStateAddr      addrs.AbsResourceInstance
	PostImportStateNewStates []providers.ImportedResource
	PostImportStateReturn    HookAction
	PostImportStateError     error

	PrePlanImportCalled bool
	PrePlanImportAddr   addrs.AbsResourceInstance
	PrePlanImportReturn HookAction
	PrePlanImportError  error

	PostPlanImportAddr   addrs.AbsResourceInstance
	PostPlanImportCalled bool
	PostPlanImportReturn HookAction
	PostPlanImportError  error

	PreApplyImportCalled bool
	PreApplyImportAddr   addrs.AbsResourceInstance
	PreApplyImportReturn HookAction
	PreApplyImportError  error

	PostApplyImportCalled bool
	PostApplyImportAddr   addrs.AbsResourceInstance
	PostApplyImportReturn HookAction
	PostApplyImportError  error

	PreEphemeralOpCalled      bool
	PreEphemeralOpAddr        addrs.AbsResourceInstance
	PreEphemeralOpReturn      HookAction
	PreEphemeralOpReturnError error

	PostEphemeralOpCalled      bool
	PostEphemeralOpAddr        addrs.AbsResourceInstance
	PostEphemeralOpError       error
	PostEphemeralOpReturn      HookAction
	PostEphemeralOpReturnError error

	StoppingCalled bool

	PostStateUpdateCalled bool
	PostStateUpdateState  *states.State
	PostStateUpdateReturn HookAction
	PostStateUpdateError  error
}

var _ Hook = (*MockHook)(nil)

func (h *MockHook) PreApply(id HookResourceIdentity, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreApplyCalled = true
	h.PreApplyAddr = id.Addr
	h.PreApplyGen = dk
	h.PreApplyAction = action
	h.PreApplyPriorState = priorState
	h.PreApplyPlannedState = plannedNewState
	return h.PreApplyReturn, h.PreApplyError
}

func (h *MockHook) PostApply(id HookResourceIdentity, dk addrs.DeposedKey, newState cty.Value, err error) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostApplyCalled = true
	h.PostApplyAddr = id.Addr
	h.PostApplyGen = dk
	h.PostApplyNewState = newState
	h.PostApplyError = err

	if h.PostApplyFn != nil {
		return h.PostApplyFn(id.Addr, dk, newState, err)
	}

	return h.PostApplyReturn, h.PostApplyReturnError
}

func (h *MockHook) PreDiff(id HookResourceIdentity, dk addrs.DeposedKey, priorState, proposedNewState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreDiffCalled = true
	h.PreDiffAddr = id.Addr
	h.PreDiffGen = dk
	h.PreDiffPriorState = priorState
	h.PreDiffProposedState = proposedNewState
	return h.PreDiffReturn, h.PreDiffError
}

func (h *MockHook) PostDiff(id HookResourceIdentity, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostDiffCalled = true
	h.PostDiffAddr = id.Addr
	h.PostDiffGen = dk
	h.PostDiffAction = action
	h.PostDiffPriorState = priorState
	h.PostDiffPlannedState = plannedNewState
	return h.PostDiffReturn, h.PostDiffError
}

func (h *MockHook) PreProvisionInstance(id HookResourceIdentity, state cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreProvisionInstanceCalled = true
	h.PreProvisionInstanceAddr = id.Addr
	h.PreProvisionInstanceState = state
	return h.PreProvisionInstanceReturn, h.PreProvisionInstanceError
}

func (h *MockHook) PostProvisionInstance(id HookResourceIdentity, state cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostProvisionInstanceCalled = true
	h.PostProvisionInstanceAddr = id.Addr
	h.PostProvisionInstanceState = state
	return h.PostProvisionInstanceReturn, h.PostProvisionInstanceError
}

func (h *MockHook) PreProvisionInstanceStep(id HookResourceIdentity, typeName string) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreProvisionInstanceStepCalled = true
	h.PreProvisionInstanceStepAddr = id.Addr
	h.PreProvisionInstanceStepProvisionerType = typeName
	return h.PreProvisionInstanceStepReturn, h.PreProvisionInstanceStepError
}

func (h *MockHook) PostProvisionInstanceStep(id HookResourceIdentity, typeName string, err error) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostProvisionInstanceStepCalled = true
	h.PostProvisionInstanceStepAddr = id.Addr
	h.PostProvisionInstanceStepProvisionerType = typeName
	h.PostProvisionInstanceStepErrorArg = err
	return h.PostProvisionInstanceStepReturn, h.PostProvisionInstanceStepError
}

func (h *MockHook) ProvisionOutput(id HookResourceIdentity, typeName string, line string) {
	h.Lock()
	defer h.Unlock()

	h.ProvisionOutputCalled = true
	h.ProvisionOutputAddr = id.Addr
	h.ProvisionOutputProvisionerType = typeName
	h.ProvisionOutputMessage = line
}

func (h *MockHook) PreRefresh(id HookResourceIdentity, dk addrs.DeposedKey, priorState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreRefreshCalled = true
	h.PreRefreshAddr = id.Addr
	h.PreRefreshGen = dk
	h.PreRefreshPriorState = priorState
	return h.PreRefreshReturn, h.PreRefreshError
}

func (h *MockHook) PostRefresh(id HookResourceIdentity, dk addrs.DeposedKey, priorState cty.Value, newState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostRefreshCalled = true
	h.PostRefreshAddr = id.Addr
	h.PostRefreshPriorState = priorState
	h.PostRefreshNewState = newState
	return h.PostRefreshReturn, h.PostRefreshError
}

func (h *MockHook) PreImportState(id HookResourceIdentity, importID string) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreImportStateCalled = true
	h.PreImportStateAddr = id.Addr
	h.PreImportStateID = importID
	return h.PreImportStateReturn, h.PreImportStateError
}

func (h *MockHook) PostImportState(id HookResourceIdentity, imported []providers.ImportedResource) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostImportStateCalled = true
	h.PostImportStateAddr = id.Addr
	h.PostImportStateNewStates = imported
	return h.PostImportStateReturn, h.PostImportStateError
}

func (h *MockHook) PrePlanImport(id HookResourceIdentity, importID string) (HookAction, error) {
	h.PrePlanImportCalled = true
	h.PrePlanImportAddr = id.Addr
	return h.PrePlanImportReturn, h.PrePlanImportError
}

func (h *MockHook) PostPlanImport(id HookResourceIdentity, imported []providers.ImportedResource) (HookAction, error) {
	h.PostPlanImportCalled = true
	h.PostPlanImportAddr = id.Addr
	return h.PostPlanImportReturn, h.PostPlanImportError
}

func (h *MockHook) PreApplyImport(id HookResourceIdentity, importing plans.ImportingSrc) (HookAction, error) {
	h.PreApplyImportCalled = true
	h.PreApplyImportAddr = id.Addr
	return h.PreApplyImportReturn, h.PreApplyImportError
}

func (h *MockHook) PostApplyImport(id HookResourceIdentity, importing plans.ImportingSrc) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostApplyImportCalled = true
	h.PostApplyImportAddr = id.Addr
	return h.PostApplyImportReturn, h.PostApplyImportError
}

func (h *MockHook) PreEphemeralOp(id HookResourceIdentity, action plans.Action) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreEphemeralOpCalled = true
	h.PreEphemeralOpAddr = id.Addr
	return h.PreEphemeralOpReturn, h.PreEphemeralOpReturnError
}

func (h *MockHook) PostEphemeralOp(id HookResourceIdentity, action plans.Action, opErr error) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostEphemeralOpCalled = true
	h.PostEphemeralOpAddr = id.Addr
	h.PostEphemeralOpError = opErr
	return h.PostEphemeralOpReturn, h.PostEphemeralOpReturnError
}

func (h *MockHook) Stopping() {
	h.Lock()
	defer h.Unlock()

	h.StoppingCalled = true
}

func (h *MockHook) PostStateUpdate(new *states.State) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostStateUpdateCalled = true
	h.PostStateUpdateState = new
	return h.PostStateUpdateReturn, h.PostStateUpdateError
}
