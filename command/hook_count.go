package command

import (
	"sync"

	"github.com/hashicorp/terraform/terraform"
)

// CountHook is a hook that counts the number of resources
// added, removed, changed during the course of an apply.
type CountHook struct {
	Added   int
	Changed int
	Removed int

	pending map[string]countHookAction

	sync.Mutex
	terraform.NilHook
}

type countHookAction byte

const (
	countHookActionAdd countHookAction = iota
	countHookActionChange
	countHookActionRemove
)

func (h *CountHook) Reset() {
	h.Lock()
	defer h.Unlock()

	h.pending = nil
	h.Added = 0
	h.Changed = 0
	h.Removed = 0
}

func (h *CountHook) PreApply(
	id string,
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	if h.pending == nil {
		h.pending = make(map[string]countHookAction)
	}

	action := countHookActionChange
	if d.Destroy {
		action = countHookActionRemove
	} else if s.ID == "" {
		action = countHookActionAdd
	}

	h.pending[id] = action

	return terraform.HookActionContinue, nil
}

func (h *CountHook) PostApply(
	id string,
	s *terraform.ResourceState,
	e error) (terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	if h.pending != nil {
		if a, ok := h.pending[id]; ok {
			delete(h.pending, id)

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
