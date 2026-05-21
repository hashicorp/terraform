// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateMigrate interface {
	Log(message string, params ...any)
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewStateMigrate(viewType arguments.ViewType, view *View) StateMigrate {
	switch viewType {
	case arguments.ViewHuman:
		return &StateMigrateHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

type StateMigrateHuman struct {
	view *View
}

func (s *StateMigrateHuman) Diagnostics(diags tfdiags.Diagnostics) {
	s.view.Diagnostics(diags)
}

func (s *StateMigrateHuman) Log(message string, params ...any) {
	s.view.streams.Print(fmt.Sprintf(message, params...))
}
