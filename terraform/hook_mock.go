package terraform

// MockHook is an implementation of Hook that can be used for tests.
// It records all of its function calls.
type MockHook struct {
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
