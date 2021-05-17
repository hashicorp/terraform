package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
)

// The StateLocker view is used to display locking/unlocking status messages
// if the state lock process takes longer than expected.
type StateLocker interface {
	Locking()
	Unlocking()
}

// NewStateLocker returns an initialized StateLocker implementation for the given ViewType.
func NewStateLocker(vt arguments.ViewType, view *View) StateLocker {
	switch vt {
	case arguments.ViewHuman:
		return &StateLockerHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// StateLockerHuman is an implementation of StateLocker which prints status to
// a terminal.
type StateLockerHuman struct {
	view *View
}

var _ StateLocker = (*StateLockerHuman)(nil)

func (v *StateLockerHuman) Locking() {
	v.view.streams.Println("Acquiring state lock. This may take a few moments...")
}

func (v *StateLockerHuman) Unlocking() {
	v.view.streams.Println("Releasing state lock. This may take a few moments...")
}
