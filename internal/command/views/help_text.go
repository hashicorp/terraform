// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

// CommandHelper defines the interface for providing help text and prompts for commands.
type CommandHelper interface {
	HelpPrompt(command string)
	FullHelpText(text string)
}

// HelpHuman should be embedded in human view implementations to add methods
// for displaying help prompts and full help text.
type HelpHuman struct {
	view *View
}

func (v *HelpHuman) HelpPrompt(command string) {
	v.view.HelpPrompt(command)
}

func (v *HelpHuman) FullHelpText(text string) {
	v.view.FullHelpText(text)
}
