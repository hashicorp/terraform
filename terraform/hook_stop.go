package terraform

import (
	"sync/atomic"
)

// stopHook is a private Hook implementation that Terraform uses to
// signal when to stop or cancel actions.
type stopHook struct {
	stop uint32
}

func (h *stopHook) PreApply(string, *ResourceState, *ResourceDiff) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostApply(string, *ResourceState, error) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreDiff(string, *ResourceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostDiff(string, *ResourceDiff) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreRefresh(string, *ResourceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostRefresh(string, *ResourceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) hook() (HookAction, error) {
	if h.Stopped() {
		return HookActionHalt, nil
	}

	return HookActionContinue, nil
}

// reset should be called within the lock context
func (h *stopHook) Reset() {
	atomic.StoreUint32(&h.stop, 0)
}

func (h *stopHook) Stop() {
	atomic.StoreUint32(&h.stop, 1)
}

func (h *stopHook) Stopped() bool {
	return atomic.LoadUint32(&h.stop) == 1
}
