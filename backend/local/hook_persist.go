package local

import (
	"fmt"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

type PersistHook struct {
	terraform.NilHook
	Context *terraform.Context
	State   state.State
}

func (h *PersistHook) PreApply(
	instInfo *terraform.InstanceInfo,
	instState *terraform.InstanceState,
	diff *terraform.InstanceDiff) (terraform.HookAction, error) {

	if h.State == nil {
		return terraform.HookActionContinue, nil
	}

	recoveryLogWriter, existsWriter := h.State.(state.RecoveryLogWriter)

	if !existsWriter {
		fmt.Printf("State does not support 'RecoveryLogWriter' functional.\n" +
			"Please make sure what you set AWS remote backend as backend. Currently, only this backend is supported.\n")
		return terraform.HookActionContinue, nil
	}

	_id := getId(diff)
	if _id != "" {
		fmt.Printf("Resource processing. ID: %s\n", _id)
		state.GetGlobalInstancesStatusLogger().Add(_id, instInfo, instState, recoveryLogWriter)
	} else {
		fmt.Printf("Id is empty. Added deferred handler.\n")
		diff.IdChangedHandler = func(info *terraform.InstanceInfo, _state *terraform.InstanceState, s state.RecoveryLogWriter) terraform.OnIdChangedHandler {
			return func(id string) {
				fmt.Printf("Deferred handler performed with id: %s\n", id)
				state.GetGlobalInstancesStatusLogger().Add(id, info, _state, s)
			}
		}(instInfo, instState, recoveryLogWriter)

		state.GetGlobalInstancesStatusLogger().SetLostResource(instInfo, instState, diff, recoveryLogWriter)
	}

	return terraform.HookActionContinue, nil
}

func (h *PersistHook) PostApply(instInfo *terraform.InstanceInfo, instState *terraform.InstanceState, err error) (terraform.HookAction, error) {
	if recoveryLogWriter, existsWriter := h.State.(state.RecoveryLogWriter); existsWriter {
		if h.State != nil {
			h.State.WriteState(h.Context.State())
			h.State.PersistState()
			state.GetGlobalInstancesStatusLogger().Remove(instState, instInfo, recoveryLogWriter)
		}
	} else {
		fmt.Printf("State does not support 'RecoveryLogWriter' functional.\n" +
			"Please make sure what you set AWS remote backend as backend. Currently, only this backend is supported.\n")
	}
	return terraform.HookActionContinue, nil
}

func getId(diff *terraform.InstanceDiff) string {
	if diff == nil && diff.Attributes == nil {
		return ""
	}
	nameAttr, ok := diff.GetAttribute("name")
	if !ok {
		return ""
	}
	if nameAttr.NewComputed {
		return ""
	}
	name := nameAttr.New
	return name
}
