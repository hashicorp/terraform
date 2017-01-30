package terraform

import (
	"sync/atomic"
)

// stopHook is a private Hook implementation that Terraform uses to
// signal when to stop or cancel actions.
type stopHook struct {
	stop uint32
}

func (h *stopHook) PreApply(*InstanceInfo, *InstanceState, *InstanceDiff) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostApply(*InstanceInfo, *InstanceState, error) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreDiff(*InstanceInfo, *InstanceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostDiff(*InstanceInfo, *InstanceDiff) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreProvisionResource(*InstanceInfo, *InstanceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostProvisionResource(*InstanceInfo, *InstanceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreProvision(*InstanceInfo, string) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostProvision(*InstanceInfo, string, error) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) ProvisionOutput(*InstanceInfo, string, string) {
}

func (h *stopHook) PreRefresh(*InstanceInfo, *InstanceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostRefresh(*InstanceInfo, *InstanceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreImportState(*InstanceInfo, string) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostImportState(*InstanceInfo, []*InstanceState) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostStateUpdate(*State) (HookAction, error) {
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
