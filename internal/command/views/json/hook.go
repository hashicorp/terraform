package json

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
)

type Hook interface {
	HookType() MessageType
	String() string
}

// ApplyStart: triggered by PreApply hook
type applyStart struct {
	Resource   ResourceAddr `json:"resource"`
	Action     ChangeAction `json:"action"`
	IDKey      string       `json:"id_key,omitempty"`
	IDValue    string       `json:"id_value,omitempty"`
	actionVerb string
}

var _ Hook = (*applyStart)(nil)

func (h *applyStart) HookType() MessageType {
	return MessageApplyStart
}

func (h *applyStart) String() string {
	var id string
	if h.IDKey != "" && h.IDValue != "" {
		id = fmt.Sprintf(" [%s=%s]", h.IDKey, h.IDValue)
	}
	return fmt.Sprintf("%s: %s...%s", h.Resource.Addr, h.actionVerb, id)
}

func NewApplyStart(addr addrs.AbsResourceInstance, action plans.Action, idKey string, idValue string) Hook {
	hook := &applyStart{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		IDKey:      idKey,
		IDValue:    idValue,
		actionVerb: startActionVerb(action),
	}

	return hook
}

// ApplyProgress: currently triggered by a timer started on PreApply. In
// future, this might also be triggered by provider progress reporting.
type applyProgress struct {
	Resource   ResourceAddr `json:"resource"`
	Action     ChangeAction `json:"action"`
	Elapsed    float64      `json:"elapsed_seconds"`
	actionVerb string
	elapsed    time.Duration
}

var _ Hook = (*applyProgress)(nil)

func (h *applyProgress) HookType() MessageType {
	return MessageApplyProgress
}

func (h *applyProgress) String() string {
	return fmt.Sprintf("%s: Still %s... [%s elapsed]", h.Resource.Addr, h.actionVerb, h.elapsed)
}

func NewApplyProgress(addr addrs.AbsResourceInstance, action plans.Action, elapsed time.Duration) Hook {
	return &applyProgress{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		Elapsed:    elapsed.Seconds(),
		actionVerb: progressActionVerb(action),
		elapsed:    elapsed,
	}
}

// ApplyComplete: triggered by PostApply hook
type applyComplete struct {
	Resource   ResourceAddr `json:"resource"`
	Action     ChangeAction `json:"action"`
	IDKey      string       `json:"id_key,omitempty"`
	IDValue    string       `json:"id_value,omitempty"`
	Elapsed    float64      `json:"elapsed_seconds"`
	actionNoun string
	elapsed    time.Duration
}

var _ Hook = (*applyComplete)(nil)

func (h *applyComplete) HookType() MessageType {
	return MessageApplyComplete
}

func (h *applyComplete) String() string {
	var id string
	if h.IDKey != "" && h.IDValue != "" {
		id = fmt.Sprintf(" [%s=%s]", h.IDKey, h.IDValue)
	}
	return fmt.Sprintf("%s: %s complete after %s%s", h.Resource.Addr, h.actionNoun, h.elapsed, id)
}

func NewApplyComplete(addr addrs.AbsResourceInstance, action plans.Action, idKey, idValue string, elapsed time.Duration) Hook {
	return &applyComplete{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		IDKey:      idKey,
		IDValue:    idValue,
		Elapsed:    elapsed.Seconds(),
		actionNoun: actionNoun(action),
		elapsed:    elapsed,
	}
}

// ApplyErrored: triggered by PostApply hook on failure. This will be followed
// by diagnostics when the apply finishes.
type applyErrored struct {
	Resource   ResourceAddr `json:"resource"`
	Action     ChangeAction `json:"action"`
	Elapsed    float64      `json:"elapsed_seconds"`
	actionNoun string
	elapsed    time.Duration
}

var _ Hook = (*applyErrored)(nil)

func (h *applyErrored) HookType() MessageType {
	return MessageApplyErrored
}

func (h *applyErrored) String() string {
	return fmt.Sprintf("%s: %s errored after %s", h.Resource.Addr, h.actionNoun, h.elapsed)
}

func NewApplyErrored(addr addrs.AbsResourceInstance, action plans.Action, elapsed time.Duration) Hook {
	return &applyErrored{
		Resource:   newResourceAddr(addr),
		Action:     changeAction(action),
		Elapsed:    elapsed.Seconds(),
		actionNoun: actionNoun(action),
		elapsed:    elapsed,
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
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		// This is not currently possible to reach, as we receive separate
		// passes for create and delete
		return "Replacing"
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
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		// This is not currently possible to reach, as we receive separate
		// passes for create and delete
		return "replacing"
	default:
		return "applying"
	}
}

// Convert the subset of plans.Action values we expect to receive into a
// noun for the applyComplete and applyErrored hook messages. This will be
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
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		// This is not currently possible to reach, as we receive separate
		// passes for create and delete
		return "Replacement"
	default:
		return "Apply"
	}
}
