// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views/json"
)

// The StateLocker view is used to display locking/unlocking status messages
// if the state lock process takes longer than expected.
//
// NOTE: A state locker view is always a secondary view during a command,
// so it will not need to embed the JSONOutputVersionLogger interface.
type StateLocker interface {
	Locking()
	Unlocking()
}

// NewStateLocker returns an initialized StateLocker implementation for the given ViewType.
func NewStateLocker(vt arguments.ViewType, view *View) StateLocker {
	switch vt {
	case arguments.ViewHuman:
		return &StateLockerHuman{view: view}
	case arguments.ViewJSON:
		return &StateLockerJSON{view: NewJSONView(view)}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// StateLockerHuman is an implementation of StateLocker which prints status to
// a terminal.
type StateLockerHuman struct {
	view *View
}

var (
	_ StateLocker = (*StateLockerHuman)(nil)
	_ StateLocker = (*StateLockerJSON)(nil)
)

func (v *StateLockerHuman) Locking() {
	v.view.streams.Println("Acquiring state lock. This may take a few moments...")
}

func (v *StateLockerHuman) Unlocking() {
	v.view.streams.Println("Releasing state lock. This may take a few moments...")
}

// StateLockerJSON is an implementation of StateLocker which prints the state lock status
// to a terminal in machine-readable JSON form.
type StateLockerJSON struct {
	view *JSONView
}

func (v *StateLockerJSON) Locking() {
	message := "Acquiring state lock. This may take a few moments..."

	v.view.log.Info(
		message,
		"type", json.MessageStateLockAcquire,
	)
}

func (v *StateLockerJSON) Unlocking() {
	message := "Releasing state lock. This may take a few moments..."

	v.view.log.Info(
		message,
		"type", json.MessageStateLockRelease,
	)
}
