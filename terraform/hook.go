package terraform

// HookAction is an enum of actions that can be taken as a result of a hook
// callback. This allows you to modify the behavior of Terraform at runtime.
type HookAction byte

const (
	// HookActionContinue continues with processing as usual.
	HookActionContinue HookAction = iota

	// HookActionHalt halts immediately: no more hooks are processed
	// and the action that Terraform was about to take is cancelled.
	HookActionHalt
)

// Hook is the interface that must be implemented to hook into various
// parts of Terraform, allowing you to inspect or change behavior at runtime.
//
// There are MANY hook points into Terraform. If you only want to implement
// some hook points, but not all (which is the likely case), then embed the
// NilHook into your struct, which implements all of the interface but does
// nothing. Then, override only the functions you want to implement.
type Hook interface {
	// PreApply and PostApply are called before and after a single
	// resource is applied. The error argument in PostApply is the
	// error, if any, that was returned from the provider Apply call itself.
	PreApply(*InstanceInfo, *InstanceState, *InstanceDiff) (HookAction, error)
	PostApply(*InstanceInfo, *InstanceState, error) (HookAction, error)

	// PreDiff and PostDiff are called before and after a single resource
	// resource is diffed.
	PreDiff(*InstanceInfo, *InstanceState) (HookAction, error)
	PostDiff(*InstanceInfo, *InstanceDiff) (HookAction, error)

	// Provisioning hooks
	//
	// All should be self-explanatory. ProvisionOutput is called with
	// output sent back by the provisioners. This will be called multiple
	// times as output comes in, but each call should represent a line of
	// output. The ProvisionOutput method cannot control whether the
	// hook continues running.
	PreProvisionResource(*InstanceInfo, *InstanceState) (HookAction, error)
	PostProvisionResource(*InstanceInfo, *InstanceState) (HookAction, error)
	PreProvision(*InstanceInfo, string) (HookAction, error)
	PostProvision(*InstanceInfo, string, error) (HookAction, error)
	ProvisionOutput(*InstanceInfo, string, string)

	// PreRefresh and PostRefresh are called before and after a single
	// resource state is refreshed, respectively.
	PreRefresh(*InstanceInfo, *InstanceState) (HookAction, error)
	PostRefresh(*InstanceInfo, *InstanceState) (HookAction, error)

	// PostStateUpdate is called after the state is updated.
	PostStateUpdate(*State) (HookAction, error)

	// PreImportState and PostImportState are called before and after
	// a single resource's state is being improted.
	PreImportState(*InstanceInfo, string) (HookAction, error)
	PostImportState(*InstanceInfo, []*InstanceState) (HookAction, error)
}

// NilHook is a Hook implementation that does nothing. It exists only to
// simplify implementing hooks. You can embed this into your Hook implementation
// and only implement the functions you are interested in.
type NilHook struct{}

func (*NilHook) PreApply(*InstanceInfo, *InstanceState, *InstanceDiff) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostApply(*InstanceInfo, *InstanceState, error) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PreDiff(*InstanceInfo, *InstanceState) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostDiff(*InstanceInfo, *InstanceDiff) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PreProvisionResource(*InstanceInfo, *InstanceState) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostProvisionResource(*InstanceInfo, *InstanceState) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PreProvision(*InstanceInfo, string) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostProvision(*InstanceInfo, string, error) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) ProvisionOutput(
	*InstanceInfo, string, string) {
}

func (*NilHook) PreRefresh(*InstanceInfo, *InstanceState) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostRefresh(*InstanceInfo, *InstanceState) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PreImportState(*InstanceInfo, string) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostImportState(*InstanceInfo, []*InstanceState) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostStateUpdate(*State) (HookAction, error) {
	return HookActionContinue, nil
}

// handleHook turns hook actions into panics. This lets you use the
// panic/recover mechanism in Go as a flow control mechanism for hook
// actions.
func handleHook(a HookAction, err error) {
	if err != nil {
		// TODO: handle errors
	}

	switch a {
	case HookActionContinue:
		return
	case HookActionHalt:
		panic(HookActionHalt)
	}
}
