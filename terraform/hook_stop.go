package terraform

import (
	"sync"
)

// stopHook is a private Hook implementation that Terraform uses to
// signal when to stop or cancel actions.
type stopHook struct {
	sync.Mutex

	// This should be incremented for every thing that can be stopped.
	// When this is zero, a stopper can assume that everything is properly
	// stopped.
	count int

	// This channel should be closed when it is time to stop
	ch chan struct{}

	serial    int
	stoppedCh chan<- struct{}
}

func (h *stopHook) PreApply(string, *ResourceState, *ResourceDiff) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostApply(string, *ResourceState) (HookAction, error) {
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
	select {
	case <-h.ch:
		h.stoppedCh <- struct{}{}
		return HookActionHalt, nil
	default:
		return HookActionContinue, nil
	}
}

// reset should be called within the lock context
func (h *stopHook) reset() {
	h.ch = make(chan struct{})
	h.count = 0
	h.serial += 1
	h.stoppedCh = nil
}

func (h *stopHook) ref() int {
	h.Lock()
	defer h.Unlock()
	h.count++
	return h.serial
}

func (h *stopHook) unref(s int) {
	h.Lock()
	defer h.Unlock()
	if h.serial == s {
		h.count--
	}
}
