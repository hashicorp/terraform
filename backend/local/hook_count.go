package local

import (
	"strings"
	"sync"

	"github.com/hashicorp/terraform/terraform"
)

// CountHook is a hook that counts the number of resources
// added, removed, changed during the course of an apply.
type CountHook struct {
	Added   int
	Changed int
	Removed int

	ToAdd          int
	ToChange       int
	ToRemove       int
	ToRemoveAndAdd int

	pending map[string]countHookAction

	sync.Mutex
	terraform.NilHook
}

func (h *CountHook) Reset() {
	h.Lock()
	defer h.Unlock()

	h.pending = nil
	h.Added = 0
	h.Changed = 0
	h.Removed = 0
}

func (h *CountHook) PreApply(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	if d.Empty() {
		return terraform.HookActionContinue, nil
	}

	if h.pending == nil {
		h.pending = make(map[string]countHookAction)
	}

	action := countHookActionChange
	if d.GetDestroy() {
		action = countHookActionRemove
	} else if s.ID == "" {
		action = countHookActionAdd
	}

	h.pending[n.HumanId()] = action

	return terraform.HookActionContinue, nil
}

func (h *CountHook) PostApply(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState,
	e error) (terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	if h.pending != nil {
		if a, ok := h.pending[n.HumanId()]; ok {
			delete(h.pending, n.HumanId())

			if e == nil {
				switch a {
				case countHookActionAdd:
					h.Added += 1
				case countHookActionChange:
					h.Changed += 1
				case countHookActionRemove:
					h.Removed += 1
				}
			}
		}
	}

	return terraform.HookActionContinue, nil
}

func (h *CountHook) PostDiff(
	n *terraform.InstanceInfo, d *terraform.InstanceDiff) (
	terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	// We don't count anything for data sources
	if strings.HasPrefix(n.Id, "data.") {
		return terraform.HookActionContinue, nil
	}

	switch d.ChangeType() {
	case terraform.DiffDestroyCreate:
		h.ToRemoveAndAdd += 1
	case terraform.DiffCreate:
		h.ToAdd += 1
	case terraform.DiffDestroy:
		h.ToRemove += 1
	case terraform.DiffUpdate:
		h.ToChange += 1
	}

	return terraform.HookActionContinue, nil
}
