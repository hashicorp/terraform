package terraform

// MockHook is an implementation of Hook that can be used for tests.
// It records all of its function calls.
type MockHook struct {
	PreDiffCalled bool
	PreDiffId     string
	PreDiffState  *ResourceState
	PreDiffReturn HookAction
	PreDiffError  error

	PostDiffCalled bool
	PostDiffId     string
	PostDiffDiff   *ResourceDiff
	PostDiffReturn HookAction
	PostDiffError  error

	PostRefreshCalled bool
	PostRefreshId     string
	PostRefreshState  *ResourceState
	PostRefreshReturn HookAction
	PostRefreshError  error

	PreRefreshCalled bool
	PreRefreshId     string
	PreRefreshState  *ResourceState
	PreRefreshReturn HookAction
	PreRefreshError  error
}

func (h *MockHook) PreDiff(n string, s *ResourceState) (HookAction, error) {
	h.PreDiffCalled = true
	h.PreDiffId = n
	h.PreDiffState = s
	return h.PreDiffReturn, h.PreDiffError
}

func (h *MockHook) PostDiff(n string, d *ResourceDiff) (HookAction, error) {
	h.PostDiffCalled = true
	h.PostDiffId = n
	h.PostDiffDiff = d
	return h.PostDiffReturn, h.PostDiffError
}

func (h *MockHook) PreRefresh(n string, s *ResourceState) (HookAction, error) {
	h.PreRefreshCalled = true
	h.PreRefreshId = n
	h.PreRefreshState = s
	return h.PreRefreshReturn, h.PreRefreshError
}

func (h *MockHook) PostRefresh(n string, s *ResourceState) (HookAction, error) {
	h.PostRefreshCalled = true
	h.PostRefreshId = n
	h.PostRefreshState = s
	return h.PostRefreshReturn, h.PostRefreshError
}
