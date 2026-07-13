// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateMigrate interface {
	Log(message string, params ...any)
	Diagnostics(diags tfdiags.Diagnostics)

	ProviderInstaller
	Spacer // The `state migrate` command logs empty lines to space-out different sections of human-readable output
}

func NewStateMigrate(viewType arguments.ViewType, view *View) StateMigrate {
	switch viewType {
	case arguments.ViewHuman:
		return &StateMigrateHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

var (
	_ StateMigrate      = (*StateMigrateHuman)(nil)
	_ ProviderInstaller = (*StateMigrateHuman)(nil)
	_ Spacer            = (*StateMigrateHuman)(nil)
)

type StateMigrateHuman struct {
	view *View
}

func (s *StateMigrateHuman) Diagnostics(diags tfdiags.Diagnostics) {
	s.view.Diagnostics(diags)
}

func (s *StateMigrateHuman) Log(message string, params ...any) {
	s.view.streams.Println(fmt.Sprintf(message, params...))
}

// Implements Spacer
func (s *StateMigrateHuman) Spacer() {
	s.view.Spacer()
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) LogInitMessage(code InitMessageCode, params ...any) {
	msg, ok := MessageRegistry[code]
	if !ok {
		panic("missing message for InstallingProviderMessage init message code")
	}
	s.Log(msg.HumanValue, params...)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) Output(code InitMessageCode, params ...any) {
	msg, ok := MessageRegistry[code]
	if !ok {
		panic("missing message for InstallingProviderMessage init message code")
	}
	s.Log(msg.HumanValue, params...)
}

// Implements ProviderInstaller interface.
func (s *StateMigrateHuman) PrepareMessage(code InitMessageCode, params ...any) string {
	message, ok := MessageRegistry[code]
	if !ok {
		// display the message code as fallback if not found in the message registry
		return string(code)
	}

	if message.HumanValue == "" {
		// no need to apply colorization if the message is empty
		return message.HumanValue
	}

	return s.view.colorize.Color(strings.TrimSpace(fmt.Sprintf(message.HumanValue, params...)))
}
