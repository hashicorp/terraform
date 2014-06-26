package terraform

// MockHook is an implementation of Hook that can be used for tests.
// It records all of its function calls.
type MockHook struct {
	PostRefreshCalled bool
	PostRefreshState  *ResourceState
	PostRefreshReturn HookAction
	PostRefreshError  error

	PreRefreshCalled bool
	PreRefreshState  *ResourceState
	PreRefreshReturn HookAction
	PreRefreshError  error
}

func (h *MockHook) PreRefresh(s *ResourceState) (HookAction, error) {
	h.PreRefreshCalled = true
	h.PreRefreshState = s
	return h.PreRefreshReturn, h.PreRefreshError
}

func (h *MockHook) PostRefresh(s *ResourceState) (HookAction, error) {
	h.PostRefreshCalled = true
	h.PostRefreshState = s
	return h.PostRefreshReturn, h.PostRefreshError
}
