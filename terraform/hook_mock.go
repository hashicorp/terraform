package terraform

// MockHook is an implementation of Hook that can be used for tests.
// It records all of its function calls.
type MockHook struct {
	PreApplyCalled bool
	PreApplyInfo   *InstanceInfo
	PreApplyDiff   *InstanceDiff
	PreApplyState  *InstanceState
	PreApplyReturn HookAction
	PreApplyError  error

	PostApplyCalled      bool
	PostApplyInfo        *InstanceInfo
	PostApplyState       *InstanceState
	PostApplyError       error
	PostApplyReturn      HookAction
	PostApplyReturnError error

	PreDiffCalled bool
	PreDiffInfo   *InstanceInfo
	PreDiffState  *InstanceState
	PreDiffReturn HookAction
	PreDiffError  error

	PostDiffCalled bool
	PostDiffInfo   *InstanceInfo
	PostDiffDiff   *InstanceDiff
	PostDiffReturn HookAction
	PostDiffError  error

	PreProvisionResourceCalled bool
	PreProvisionResourceInfo   *InstanceInfo
	PreProvisionInstanceState  *InstanceState
	PreProvisionResourceReturn HookAction
	PreProvisionResourceError  error

	PostProvisionResourceCalled bool
	PostProvisionResourceInfo   *InstanceInfo
	PostProvisionInstanceState  *InstanceState
	PostProvisionResourceReturn HookAction
	PostProvisionResourceError  error

	PreProvisionCalled        bool
	PreProvisionInfo          *InstanceInfo
	PreProvisionProvisionerId string
	PreProvisionReturn        HookAction
	PreProvisionError         error

	PostProvisionCalled        bool
	PostProvisionInfo          *InstanceInfo
	PostProvisionProvisionerId string
	PostProvisionReturn        HookAction
	PostProvisionError         error

	ProvisionOutputCalled        bool
	ProvisionOutputInfo          *InstanceInfo
	ProvisionOutputProvisionerId string
	ProvisionOutputMessage       string

	PostRefreshCalled bool
	PostRefreshInfo   *InstanceInfo
	PostRefreshState  *InstanceState
	PostRefreshReturn HookAction
	PostRefreshError  error

	PreRefreshCalled bool
	PreRefreshInfo   *InstanceInfo
	PreRefreshState  *InstanceState
	PreRefreshReturn HookAction
	PreRefreshError  error

	PostStateUpdateCalled bool
	PostStateUpdateState  *State
	PostStateUpdateReturn HookAction
	PostStateUpdateError  error
}

func (h *MockHook) PreApply(n *InstanceInfo, s *InstanceState, d *InstanceDiff) (HookAction, error) {
	h.PreApplyCalled = true
	h.PreApplyInfo = n
	h.PreApplyDiff = d
	h.PreApplyState = s
	return h.PreApplyReturn, h.PreApplyError
}

func (h *MockHook) PostApply(n *InstanceInfo, s *InstanceState, e error) (HookAction, error) {
	h.PostApplyCalled = true
	h.PostApplyInfo = n
	h.PostApplyState = s
	h.PostApplyError = e
	return h.PostApplyReturn, h.PostApplyReturnError
}

func (h *MockHook) PreDiff(n *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.PreDiffCalled = true
	h.PreDiffInfo = n
	h.PreDiffState = s
	return h.PreDiffReturn, h.PreDiffError
}

func (h *MockHook) PostDiff(n *InstanceInfo, d *InstanceDiff) (HookAction, error) {
	h.PostDiffCalled = true
	h.PostDiffInfo = n
	h.PostDiffDiff = d
	return h.PostDiffReturn, h.PostDiffError
}

func (h *MockHook) PreProvisionResource(n *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.PreProvisionResourceCalled = true
	h.PreProvisionResourceInfo = n
	h.PreProvisionInstanceState = s
	return h.PreProvisionResourceReturn, h.PreProvisionResourceError
}

func (h *MockHook) PostProvisionResource(n *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.PostProvisionResourceCalled = true
	h.PostProvisionResourceInfo = n
	h.PostProvisionInstanceState = s
	return h.PostProvisionResourceReturn, h.PostProvisionResourceError
}

func (h *MockHook) PreProvision(n *InstanceInfo, provId string) (HookAction, error) {
	h.PreProvisionCalled = true
	h.PreProvisionInfo = n
	h.PreProvisionProvisionerId = provId
	return h.PreProvisionReturn, h.PreProvisionError
}

func (h *MockHook) PostProvision(n *InstanceInfo, provId string) (HookAction, error) {
	h.PostProvisionCalled = true
	h.PostProvisionInfo = n
	h.PostProvisionProvisionerId = provId
	return h.PostProvisionReturn, h.PostProvisionError
}

func (h *MockHook) ProvisionOutput(
	n *InstanceInfo,
	provId string,
	msg string) {
	h.ProvisionOutputCalled = true
	h.ProvisionOutputInfo = n
	h.ProvisionOutputProvisionerId = provId
	h.ProvisionOutputMessage = msg
}

func (h *MockHook) PreRefresh(n *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.PreRefreshCalled = true
	h.PreRefreshInfo = n
	h.PreRefreshState = s
	return h.PreRefreshReturn, h.PreRefreshError
}

func (h *MockHook) PostRefresh(n *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.PostRefreshCalled = true
	h.PostRefreshInfo = n
	h.PostRefreshState = s
	return h.PostRefreshReturn, h.PostRefreshError
}

func (h *MockHook) PostStateUpdate(s *State) (HookAction, error) {
	h.PostStateUpdateCalled = true
	h.PostStateUpdateState = s
	return h.PostStateUpdateReturn, h.PostStateUpdateError
}
