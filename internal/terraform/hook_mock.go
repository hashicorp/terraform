package terraform

import (
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

// MockHook is an implementation of Hook that can be used for tests.
// It records all of its function calls.
type MockHook struct {
	sync.Mutex

	PreApplyCalled       bool
	PreApplyAddr         addrs.AbsResourceInstance
	PreApplyGen          states.Generation
	PreApplyAction       plans.Action
	PreApplyPriorState   cty.Value
	PreApplyPlannedState cty.Value
	PreApplyReturn       HookAction
	PreApplyError        error

	PostApplyCalled      bool
	PostApplyAddr        addrs.AbsResourceInstance
	PostApplyGen         states.Generation
	PostApplyNewState    cty.Value
	PostApplyError       error
	PostApplyReturn      HookAction
	PostApplyReturnError error
	PostApplyFn          func(addrs.AbsResourceInstance, states.Generation, cty.Value, error) (HookAction, error)

	PreDiffCalled        bool
	PreDiffAddr          addrs.AbsResourceInstance
	PreDiffGen           states.Generation
	PreDiffPriorState    cty.Value
	PreDiffProposedState cty.Value
	PreDiffReturn        HookAction
	PreDiffError         error

	PostDiffCalled       bool
	PostDiffAddr         addrs.AbsResourceInstance
	PostDiffGen          states.Generation
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
	PreRefreshGen        states.Generation
	PreRefreshPriorState cty.Value
	PreRefreshReturn     HookAction
	PreRefreshError      error

	PostRefreshCalled     bool
	PostRefreshAddr       addrs.AbsResourceInstance
	PostRefreshGen        states.Generation
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

	PostStateUpdateCalled bool
	PostStateUpdateState  *states.State
	PostStateUpdateReturn HookAction
	PostStateUpdateError  error
}

var _ Hook = (*MockHook)(nil)

func (h *MockHook) PreApply(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreApplyCalled = true
	h.PreApplyAddr = addr
	h.PreApplyGen = gen
	h.PreApplyAction = action
	h.PreApplyPriorState = priorState
	h.PreApplyPlannedState = plannedNewState
	return h.PreApplyReturn, h.PreApplyError
}

func (h *MockHook) PostApply(addr addrs.AbsResourceInstance, gen states.Generation, newState cty.Value, err error) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostApplyCalled = true
	h.PostApplyAddr = addr
	h.PostApplyGen = gen
	h.PostApplyNewState = newState
	h.PostApplyError = err

	if h.PostApplyFn != nil {
		return h.PostApplyFn(addr, gen, newState, err)
	}

	return h.PostApplyReturn, h.PostApplyReturnError
}

func (h *MockHook) PreDiff(addr addrs.AbsResourceInstance, gen states.Generation, priorState, proposedNewState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreDiffCalled = true
	h.PreDiffAddr = addr
	h.PreDiffGen = gen
	h.PreDiffPriorState = priorState
	h.PreDiffProposedState = proposedNewState
	return h.PreDiffReturn, h.PreDiffError
}

func (h *MockHook) PostDiff(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostDiffCalled = true
	h.PostDiffAddr = addr
	h.PostDiffGen = gen
	h.PostDiffAction = action
	h.PostDiffPriorState = priorState
	h.PostDiffPlannedState = plannedNewState
	return h.PostDiffReturn, h.PostDiffError
}

func (h *MockHook) PreProvisionInstance(addr addrs.AbsResourceInstance, state cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreProvisionInstanceCalled = true
	h.PreProvisionInstanceAddr = addr
	h.PreProvisionInstanceState = state
	return h.PreProvisionInstanceReturn, h.PreProvisionInstanceError
}

func (h *MockHook) PostProvisionInstance(addr addrs.AbsResourceInstance, state cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostProvisionInstanceCalled = true
	h.PostProvisionInstanceAddr = addr
	h.PostProvisionInstanceState = state
	return h.PostProvisionInstanceReturn, h.PostProvisionInstanceError
}

func (h *MockHook) PreProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreProvisionInstanceStepCalled = true
	h.PreProvisionInstanceStepAddr = addr
	h.PreProvisionInstanceStepProvisionerType = typeName
	return h.PreProvisionInstanceStepReturn, h.PreProvisionInstanceStepError
}

func (h *MockHook) PostProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string, err error) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostProvisionInstanceStepCalled = true
	h.PostProvisionInstanceStepAddr = addr
	h.PostProvisionInstanceStepProvisionerType = typeName
	h.PostProvisionInstanceStepErrorArg = err
	return h.PostProvisionInstanceStepReturn, h.PostProvisionInstanceStepError
}

func (h *MockHook) ProvisionOutput(addr addrs.AbsResourceInstance, typeName string, line string) {
	h.Lock()
	defer h.Unlock()

	h.ProvisionOutputCalled = true
	h.ProvisionOutputAddr = addr
	h.ProvisionOutputProvisionerType = typeName
	h.ProvisionOutputMessage = line
}

func (h *MockHook) PreRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreRefreshCalled = true
	h.PreRefreshAddr = addr
	h.PreRefreshGen = gen
	h.PreRefreshPriorState = priorState
	return h.PreRefreshReturn, h.PreRefreshError
}

func (h *MockHook) PostRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value, newState cty.Value) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostRefreshCalled = true
	h.PostRefreshAddr = addr
	h.PostRefreshPriorState = priorState
	h.PostRefreshNewState = newState
	return h.PostRefreshReturn, h.PostRefreshError
}

func (h *MockHook) PreImportState(addr addrs.AbsResourceInstance, importID string) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PreImportStateCalled = true
	h.PreImportStateAddr = addr
	h.PreImportStateID = importID
	return h.PreImportStateReturn, h.PreImportStateError
}

func (h *MockHook) PostImportState(addr addrs.AbsResourceInstance, imported []providers.ImportedResource) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostImportStateCalled = true
	h.PostImportStateAddr = addr
	h.PostImportStateNewStates = imported
	return h.PostImportStateReturn, h.PostImportStateError
}

func (h *MockHook) PostStateUpdate(new *states.State) (HookAction, error) {
	h.Lock()
	defer h.Unlock()

	h.PostStateUpdateCalled = true
	h.PostStateUpdateState = new
	return h.PostStateUpdateReturn, h.PostStateUpdateError
}
