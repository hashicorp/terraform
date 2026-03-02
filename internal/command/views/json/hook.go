// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
)

type Hook interface {
	HookType() MessageType
	String() string
}

// operationStart: triggered by Pre{Apply,EphemeralOp} hook
type operationStart struct {
	Resource   ResourceAddr `json:"resource"`
	Action     ChangeAction `json:"action"`
	IDKey      string       `json:"id_key,omitempty"`
	IDValue    string       `json:"id_value,omitempty"`
	actionVerb string
	msgType    MessageType
}

var _ Hook = (*operationStart)(nil)

func (h *operationStart) HookType() MessageType {
	return h.msgType
}

func (h *operationStart) String() string {
	var id string
	if h.IDKey != "" && h.IDValue != "" {
		id = fmt.Sprintf(" [%s=%s]", h.IDKey, h.IDValue)
	}
	return fmt.Sprintf("%s: %s...%s", h.Resource.Addr, h.actionVerb, id)
}

func NewApplyStart(addr addrs.AbsResourceInstance, action plans.Action, idKey string, idValue string) Hook {
	hook := &operationStart{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		IDKey:      idKey,
		IDValue:    idValue,
		actionVerb: startActionVerb(action),
		msgType:    MessageApplyStart,
	}

	return hook
}

func NewEphemeralOpStart(addr addrs.AbsResourceInstance, action plans.Action) Hook {
	hook := &operationStart{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		actionVerb: startActionVerb(action),
		msgType:    MessageEphemeralOpStart,
	}

	return hook
}

// operationProgress: currently triggered by a timer started on Pre{Apply,EphemeralOp}. In
// future, this might also be triggered by provider progress reporting.
type operationProgress struct {
	Resource   ResourceAddr `json:"resource"`
	Action     ChangeAction `json:"action"`
	Elapsed    float64      `json:"elapsed_seconds"`
	actionVerb string
	elapsed    time.Duration
	msgType    MessageType
}

var _ Hook = (*operationProgress)(nil)

func (h *operationProgress) HookType() MessageType {
	return h.msgType
}

func (h *operationProgress) String() string {
	return fmt.Sprintf("%s: Still %s... [%s elapsed]", h.Resource.Addr, h.actionVerb, h.elapsed)
}

func NewApplyProgress(addr addrs.AbsResourceInstance, action plans.Action, elapsed time.Duration) Hook {
	return &operationProgress{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		Elapsed:    elapsed.Seconds(),
		actionVerb: progressActionVerb(action),
		elapsed:    elapsed,
		msgType:    MessageApplyProgress,
	}
}

func NewEphemeralOpProgress(addr addrs.AbsResourceInstance, action plans.Action, elapsed time.Duration) Hook {
	return &operationProgress{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		Elapsed:    elapsed.Seconds(),
		actionVerb: progressActionVerb(action),
		elapsed:    elapsed,
		msgType:    MessageEphemeralOpProgress,
	}
}

// operationComplete: triggered by PostApply hook
type operationComplete struct {
	Resource   ResourceAddr `json:"resource"`
	Action     ChangeAction `json:"action"`
	IDKey      string       `json:"id_key,omitempty"`
	IDValue    string       `json:"id_value,omitempty"`
	Elapsed    float64      `json:"elapsed_seconds"`
	actionNoun string
	elapsed    time.Duration
	msgType    MessageType
}

var _ Hook = (*operationComplete)(nil)

func (h *operationComplete) HookType() MessageType {
	return h.msgType
}

func (h *operationComplete) String() string {
	var id string
	if h.IDKey != "" && h.IDValue != "" {
		id = fmt.Sprintf(" [%s=%s]", h.IDKey, h.IDValue)
	}
	return fmt.Sprintf("%s: %s complete after %s%s", h.Resource.Addr, h.actionNoun, h.elapsed, id)
}

func NewApplyComplete(addr addrs.AbsResourceInstance, action plans.Action, idKey, idValue string, elapsed time.Duration) Hook {
	return &operationComplete{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		IDKey:      idKey,
		IDValue:    idValue,
		Elapsed:    elapsed.Seconds(),
		actionNoun: actionNoun(action),
		elapsed:    elapsed,
		msgType:    MessageApplyComplete,
	}
}

func NewEphemeralOpComplete(addr addrs.AbsResourceInstance, action plans.Action, elapsed time.Duration) Hook {
	return &operationComplete{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		Elapsed:    elapsed.Seconds(),
		actionNoun: actionNoun(action),
		elapsed:    elapsed,
		msgType:    MessageEphemeralOpComplete,
	}
}

// operationErrored: triggered by PostApply hook on failure. This will be followed
// by diagnostics when the apply finishes.
type operationErrored struct {
	Resource   ResourceAddr `json:"resource"`
	Action     ChangeAction `json:"action"`
	Elapsed    float64      `json:"elapsed_seconds"`
	actionNoun string
	elapsed    time.Duration
	msgType    MessageType
}

var _ Hook = (*operationErrored)(nil)

func (h *operationErrored) HookType() MessageType {
	return h.msgType
}

func (h *operationErrored) String() string {
	return fmt.Sprintf("%s: %s errored after %s", h.Resource.Addr, h.actionNoun, h.elapsed)
}

func NewApplyErrored(addr addrs.AbsResourceInstance, action plans.Action, elapsed time.Duration) Hook {
	return &operationErrored{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		Elapsed:    elapsed.Seconds(),
		actionNoun: actionNoun(action),
		elapsed:    elapsed,
		msgType:    MessageApplyErrored,
	}
}

func NewEphemeralOpErrored(addr addrs.AbsResourceInstance, action plans.Action, elapsed time.Duration) Hook {
	return &operationErrored{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		Elapsed:    elapsed.Seconds(),
		actionNoun: actionNoun(action),
		elapsed:    elapsed,
		msgType:    MessageEphemeralOpErrored,
	}
}

// ProvisionStart: triggered by PreProvisionInstanceStep hook
type provisionStart struct {
	Resource    ResourceAddr `json:"resource"`
	Provisioner string       `json:"provisioner"`
}

var _ Hook = (*provisionStart)(nil)

func (h *provisionStart) HookType() MessageType {
	return MessageProvisionStart
}

func (h *provisionStart) String() string {
	return fmt.Sprintf("%s: Provisioning with '%s'...", h.Resource.Addr, h.Provisioner)
}

func NewProvisionStart(addr addrs.AbsResourceInstance, provisioner string) Hook {
	return &provisionStart{
		Resource:    newResourceAddr(addr),
		Provisioner: provisioner,
	}
}

// ProvisionProgress: triggered by ProvisionOutput hook
type provisionProgress struct {
	Resource    ResourceAddr `json:"resource"`
	Provisioner string       `json:"provisioner"`
	Output      string       `json:"output"`
}

var _ Hook = (*provisionProgress)(nil)

func (h *provisionProgress) HookType() MessageType {
	return MessageProvisionProgress
}

func (h *provisionProgress) String() string {
	return fmt.Sprintf("%s: (%s): %s", h.Resource.Addr, h.Provisioner, h.Output)
}

func NewProvisionProgress(addr addrs.AbsResourceInstance, provisioner string, output string) Hook {
	return &provisionProgress{
		Resource:    newResourceAddr(addr),
		Provisioner: provisioner,
		Output:      output,
	}
}

// ProvisionComplete: triggered by PostProvisionInstanceStep hook
type provisionComplete struct {
	Resource    ResourceAddr `json:"resource"`
	Provisioner string       `json:"provisioner"`
}

var _ Hook = (*provisionComplete)(nil)

func (h *provisionComplete) HookType() MessageType {
	return MessageProvisionComplete
}

func (h *provisionComplete) String() string {
	return fmt.Sprintf("%s: (%s) Provisioning complete", h.Resource.Addr, h.Provisioner)
}

func NewProvisionComplete(addr addrs.AbsResourceInstance, provisioner string) Hook {
	return &provisionComplete{
		Resource:    newResourceAddr(addr),
		Provisioner: provisioner,
	}
}

// ProvisionErrored: triggered by PostProvisionInstanceStep hook on failure.
// This will be followed by diagnostics when the apply finishes.
type provisionErrored struct {
	Resource    ResourceAddr `json:"resource"`
	Provisioner string       `json:"provisioner"`
}

var _ Hook = (*provisionErrored)(nil)

func (h *provisionErrored) HookType() MessageType {
	return MessageProvisionErrored
}

func (h *provisionErrored) String() string {
	return fmt.Sprintf("%s: (%s) Provisioning errored", h.Resource.Addr, h.Provisioner)
}

func NewProvisionErrored(addr addrs.AbsResourceInstance, provisioner string) Hook {
	return &provisionErrored{
		Resource:    newResourceAddr(addr),
		Provisioner: provisioner,
	}
}

// RefreshStart: triggered by PreRefresh hook
type refreshStart struct {
	Resource ResourceAddr `json:"resource"`
	IDKey    string       `json:"id_key,omitempty"`
	IDValue  string       `json:"id_value,omitempty"`
}

var _ Hook = (*refreshStart)(nil)

func (h *refreshStart) HookType() MessageType {
	return MessageRefreshStart
}

func (h *refreshStart) String() string {
	var id string
	if h.IDKey != "" && h.IDValue != "" {
		id = fmt.Sprintf(" [%s=%s]", h.IDKey, h.IDValue)
	}
	return fmt.Sprintf("%s: Refreshing state...%s", h.Resource.Addr, id)
}

func NewRefreshStart(addr addrs.AbsResourceInstance, idKey, idValue string) Hook {
	return &refreshStart{
		Resource: newResourceAddr(addr),
		IDKey:    idKey,
		IDValue:  idValue,
	}
}

// RefreshComplete: triggered by PostRefresh hook
type refreshComplete struct {
	Resource ResourceAddr `json:"resource"`
	IDKey    string       `json:"id_key,omitempty"`
	IDValue  string       `json:"id_value,omitempty"`
}

var _ Hook = (*refreshComplete)(nil)

func (h *refreshComplete) HookType() MessageType {
	return MessageRefreshComplete
}

func (h *refreshComplete) String() string {
	var id string
	if h.IDKey != "" && h.IDValue != "" {
		id = fmt.Sprintf(" [%s=%s]", h.IDKey, h.IDValue)
	}
	return fmt.Sprintf("%s: Refresh complete%s", h.Resource.Addr, id)
}

func NewRefreshComplete(addr addrs.AbsResourceInstance, idKey, idValue string) Hook {
	return &refreshComplete{
		Resource: newResourceAddr(addr),
		IDKey:    idKey,
		IDValue:  idValue,
	}
}

// ActionStart: triggered by StartAction hook
type actionStart struct {
	Action           ActionAddr              `json:"action"`
	LifecycleTrigger *lifecycleActionTrigger `json:"lifecycle,omitempty"`
	InvokeTrigger    *invokeActionTrigger    `json:"invoke,omitempty"`
}

type lifecycleActionTrigger struct {
	TriggeringResource ResourceAddr `json:"resource"`
	TriggerIndex       int          `json:"trigger_index"`
	ActionsIndex       int          `json:"actions_index"`
	TriggerEvent       string       `json:"trigger_event"`
}

type invokeActionTrigger struct{}

var _ Hook = (*actionStart)(nil)

func (h *actionStart) HookType() MessageType {
	return MessageActionStart
}

func (h *actionStart) String() string {
	switch {
	case h.LifecycleTrigger != nil:
		return fmt.Sprintf("%s.trigger[%d]: Action started: %s", h.LifecycleTrigger.TriggeringResource.Addr, h.LifecycleTrigger.TriggerIndex, h.Action.Addr)
	default:
		return fmt.Sprintf("Action started: %s", h.Action.Addr)
	}
}

func NewActionStart(id terraform.HookActionIdentity) Hook {
	action := &actionStart{
		Action: newActionAddr(id.Addr),
	}

	switch trigger := id.ActionTrigger.(type) {
	case *plans.ResourceActionTrigger:
		action.LifecycleTrigger = &lifecycleActionTrigger{
			TriggeringResource: newResourceAddr(trigger.TriggeringResourceAddr),
			TriggerIndex:       trigger.ActionTriggerBlockIndex,
			ActionsIndex:       trigger.ActionsListIndex,
			TriggerEvent:       trigger.ActionTriggerEvent.String(),
		}
	case *plans.InvokeActionTrigger:
		action.InvokeTrigger = new(invokeActionTrigger)
	}

	return action
}

type actionProgress struct {
	Action           ActionAddr              `json:"action"`
	Message          string                  `json:"message"`
	LifecycleTrigger *lifecycleActionTrigger `json:"lifecycle,omitempty"`
	InvokeTrigger    *invokeActionTrigger    `json:"invoke,omitempty"`
}

var _ Hook = (*actionProgress)(nil)

func (h *actionProgress) HookType() MessageType {
	return MessageActionProgress
}

func (h *actionProgress) String() string {
	switch {
	case h.LifecycleTrigger != nil:
		return fmt.Sprintf("%s (%d): %s - %s", h.LifecycleTrigger.TriggeringResource.Addr, h.LifecycleTrigger.TriggerIndex, h.Action.Addr, h.Message)
	default:
		return fmt.Sprintf("%s - %s", h.Action.Addr, h.Message)
	}
}

func NewActionProgress(id terraform.HookActionIdentity, message string) Hook {
	action := &actionProgress{
		Action:  newActionAddr(id.Addr),
		Message: message,
	}

	switch trigger := id.ActionTrigger.(type) {
	case *plans.ResourceActionTrigger:
		action.LifecycleTrigger = &lifecycleActionTrigger{
			TriggeringResource: newResourceAddr(trigger.TriggeringResourceAddr),
			TriggerIndex:       trigger.ActionTriggerBlockIndex,
			ActionsIndex:       trigger.ActionsListIndex,
			TriggerEvent:       trigger.ActionTriggerEvent.String(),
		}
	case *plans.InvokeActionTrigger:
		action.InvokeTrigger = new(invokeActionTrigger)
	}

	return action
}

type actionComplete struct {
	Action           ActionAddr              `json:"action"`
	LifecycleTrigger *lifecycleActionTrigger `json:"lifecycle,omitempty"`
	InvokeTrigger    *invokeActionTrigger    `json:"invoke,omitempty"`
}

var _ Hook = (*actionComplete)(nil)

func (h *actionComplete) HookType() MessageType {
	return MessageActionComplete
}

func (h *actionComplete) String() string {
	switch {
	case h.LifecycleTrigger != nil:
		return fmt.Sprintf("%s (%d): Action complete: %s", h.LifecycleTrigger.TriggeringResource.Addr, h.LifecycleTrigger.TriggerIndex, h.Action.Addr)
	default:
		return fmt.Sprintf("Action complete: %s", h.Action.Addr)
	}
}

func NewActionComplete(id terraform.HookActionIdentity) Hook {
	action := &actionComplete{
		Action: newActionAddr(id.Addr),
	}

	switch trigger := id.ActionTrigger.(type) {
	case *plans.ResourceActionTrigger:
		action.LifecycleTrigger = &lifecycleActionTrigger{
			TriggeringResource: newResourceAddr(trigger.TriggeringResourceAddr),
			TriggerIndex:       trigger.ActionTriggerBlockIndex,
			ActionsIndex:       trigger.ActionsListIndex,
			TriggerEvent:       trigger.ActionTriggerEvent.String(),
		}
	case *plans.InvokeActionTrigger:
		action.InvokeTrigger = new(invokeActionTrigger)
	}

	return action
}

type actionErrored struct {
	Action           ActionAddr              `json:"action"`
	Error            string                  `json:"error"`
	LifecycleTrigger *lifecycleActionTrigger `json:"lifecycle,omitempty"`
	InvokeTrigger    *invokeActionTrigger    `json:"invoke,omitempty"`
}

var _ Hook = (*actionErrored)(nil)

func (h *actionErrored) HookType() MessageType {
	return MessageActionErrored
}

func (h *actionErrored) String() string {
	switch {
	case h.LifecycleTrigger != nil:
		return fmt.Sprintf("%s (%d): Action errored: %s - %s", h.LifecycleTrigger.TriggeringResource.Addr, h.LifecycleTrigger.TriggerIndex, h.Action.Addr, h.Error)
	default:
		return fmt.Sprintf("Action errored: %s - %s", h.Action.Addr, h.Error)
	}
}

func NewActionErrored(id terraform.HookActionIdentity, err error) Hook {
	action := &actionErrored{
		Action: newActionAddr(id.Addr),
		Error:  err.Error(),
	}

	switch trigger := id.ActionTrigger.(type) {
	case *plans.ResourceActionTrigger:
		action.LifecycleTrigger = &lifecycleActionTrigger{
			TriggeringResource: newResourceAddr(trigger.TriggeringResourceAddr),
			TriggerIndex:       trigger.ActionTriggerBlockIndex,
			ActionsIndex:       trigger.ActionsListIndex,
			TriggerEvent:       trigger.ActionTriggerEvent.String(),
		}
	case *plans.InvokeActionTrigger:
		action.InvokeTrigger = new(invokeActionTrigger)
	}

	return action
}

// Convert the subset of plans.Action values we expect to receive into a
// present-tense verb for the applyStart hook message.
func startActionVerb(action plans.Action) string {
	switch action {
	case plans.Create:
		return "Creating"
	case plans.Update:
		return "Modifying"
	case plans.Delete:
		return "Destroying"
	case plans.Read:
		return "Refreshing"
	case plans.CreateThenDelete, plans.DeleteThenCreate, plans.CreateThenForget:
		// This is not currently possible to reach, as we receive separate
		// passes for create and delete
		return "Replacing"
	case plans.Forget:
		return "Removing"
	case plans.Open:
		return "Opening"
	case plans.Renew:
		return "Renewing"
	case plans.Close:
		return "Closing"
	case plans.NoOp:
		// This should never be possible: a no-op planned change should not
		// be applied. We'll fall back to "Applying".
		fallthrough
	default:
		return "Applying"
	}
}

// Convert the subset of plans.Action values we expect to receive into a
// present-tense verb for the applyProgress hook message. This will be
// prefixed with "Still ", so it is lower-case.
func progressActionVerb(action plans.Action) string {
	switch action {
	case plans.Create:
		return "creating"
	case plans.Update:
		return "modifying"
	case plans.Delete:
		return "destroying"
	case plans.Read:
		return "refreshing"
	case plans.CreateThenDelete, plans.CreateThenForget, plans.DeleteThenCreate:
		// This is not currently possible to reach, as we receive separate
		// passes for create and delete
		return "replacing"
	case plans.Open:
		return "opening"
	case plans.Renew:
		return "renewing"
	case plans.Close:
		return "closing"
	case plans.Forget:
		// Removing a resource from state should not take very long. Fall back
		// to "applying" just in case, since the terminology "forgetting" is
		// meant to be internal to Terraform.
		fallthrough
	case plans.NoOp:
		// This should never be possible: a no-op planned change should not
		// be applied. We'll fall back to "applying".
		fallthrough
	default:
		return "applying"
	}
}

// Convert the subset of plans.Action values we expect to receive into a
// noun for the operationComplete and operationErrored hook messages. This will be
// combined into a phrase like "Creation complete after 1m4s".
func actionNoun(action plans.Action) string {
	switch action {
	case plans.Create:
		return "Creation"
	case plans.Update:
		return "Modifications"
	case plans.Delete:
		return "Destruction"
	case plans.Read:
		return "Refresh"
	case plans.CreateThenDelete, plans.DeleteThenCreate, plans.CreateThenForget:
		// This is not currently possible to reach, as we receive separate
		// passes for create and delete
		return "Replacement"
	case plans.Forget:
		return "Removal"
	case plans.Open:
		return "Opening"
	case plans.Renew:
		return "Renewal"
	case plans.Close:
		return "Closing"
	case plans.NoOp:
		// This should never be possible: a no-op planned change should not
		// be applied. We'll fall back to "Apply".
		fallthrough
	default:
		return "Apply"
	}
}
