package views

import (
	"fmt"

	"github.com/hashicorp/terraform/command/arguments"
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
		return &StateLockerHuman{View: *view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// StateLockerHuman is an implementation of StateLocker which prints status to
// a terminal.
type StateLockerHuman struct {
	View
}

var _ StateLocker = (*StateLockerHuman)(nil)

func (v *StateLockerHuman) Locking() {
	v.streams.Println("Acquiring state lock. This may take a few moments...")
}

func (v *StateLockerHuman) Unlocking() {
	v.streams.Println("Releasing state lock. This may take a few moments...")
}
