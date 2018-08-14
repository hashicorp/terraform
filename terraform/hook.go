package terraform

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

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
	// PreApply and PostApply are called before and after an action for a
	// single instance is applied. The error argument in PostApply is the
	// error, if any, that was returned from the provider Apply call itself.
	PreApply(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error)
	PostApply(addr addrs.AbsResourceInstance, gen states.Generation, newState cty.Value, err error) (HookAction, error)

	// PreDiff and PostDiff are called before and after a provider is given
	// the opportunity to customize the proposed new state to produce the
	// planned new state.
	PreDiff(addr addrs.AbsResourceInstance, gen states.Generation, priorState, proposedNewState cty.Value) (HookAction, error)
	PostDiff(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error)

	// The provisioning hooks signal both the overall start end end of
	// provisioning for a particular instance and of each of the individual
	// configured provisioners for each instance. The sequence of these
	// for a given instance might look something like this:
	//
	//          PreProvisionInstance(aws_instance.foo[1], ...)
	//      PreProvisionInstanceStep(aws_instance.foo[1], "file")
	//     PostProvisionInstanceStep(aws_instance.foo[1], "file", nil)
	//      PreProvisionInstanceStep(aws_instance.foo[1], "remote-exec")
	//               ProvisionOutput(aws_instance.foo[1], "remote-exec", "Installing foo...")
	//               ProvisionOutput(aws_instance.foo[1], "remote-exec", "Configuring bar...")
	//     PostProvisionInstanceStep(aws_instance.foo[1], "remote-exec", nil)
	//         PostProvisionInstance(aws_instance.foo[1], ...)
	//
	// ProvisionOutput is called with output sent back by the provisioners.
	// This will be called multiple times as output comes in, with each call
	// representing one line of output. It cannot control whether the
	// provisioner continues running.
	PreProvisionInstance(addr addrs.AbsResourceInstance, state cty.Value) (HookAction, error)
	PostProvisionInstance(addr addrs.AbsResourceInstance, state cty.Value) (HookAction, error)
	PreProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string) (HookAction, error)
	PostProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string, err error) (HookAction, error)
	ProvisionOutput(addr addrs.AbsResourceInstance, typeName string, line string)

	// PreRefresh and PostRefresh are called before and after a single
	// resource state is refreshed, respectively.
	PreRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value) (HookAction, error)
	PostRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value, newState cty.Value) (HookAction, error)

	// PreImportState and PostImportState are called before and after
	// (respectively) each state import operation for a given resource address.
	PreImportState(addr addrs.AbsResourceInstance, importID string) (HookAction, error)
	PostImportState(addr addrs.AbsResourceInstance, imported []*states.ImportedObject) (HookAction, error)

	// PostStateUpdate is called each time the state is updated. It receives
	// a deep copy of the state, which it may therefore access freely without
	// any need for locks to protect from concurrent writes from the caller.
	PostStateUpdate(new *states.State) (HookAction, error)
}

// NilHook is a Hook implementation that does nothing. It exists only to
// simplify implementing hooks. You can embed this into your Hook implementation
// and only implement the functions you are interested in.
type NilHook struct{}

var _ Hook = (*NilHook)(nil)

func (*NilHook) PreApply(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostApply(addr addrs.AbsResourceInstance, gen states.Generation, newState cty.Value, err error) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PreDiff(addr addrs.AbsResourceInstance, gen states.Generation, priorState, proposedNewState cty.Value) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostDiff(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PreProvisionInstance(addr addrs.AbsResourceInstance, state cty.Value) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostProvisionInstance(addr addrs.AbsResourceInstance, state cty.Value) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PreProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string, err error) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) ProvisionOutput(addr addrs.AbsResourceInstance, typeName string, line string) {
}

func (*NilHook) PreRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value, newState cty.Value) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PreImportState(addr addrs.AbsResourceInstance, importID string) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostImportState(addr addrs.AbsResourceInstance, imported []*states.ImportedObject) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostStateUpdate(new *states.State) (HookAction, error) {
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
