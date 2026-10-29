// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Unlock is the view used for the `unlock` command.
type Unlock interface {
	// The user didn't force the unlock and didn't respond with "yes" to the confirmation prompt.
	ForceUnlockCancelled(diags tfdiags.Diagnostics)

	// The unlock was attempted and succeeded.
	UnlockSuccessful(diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewUnlock(viewType arguments.ViewType, view *View) Unlock {
	switch viewType {
	case arguments.ViewHuman:
		return &UnlockHuman{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

type UnlockHuman struct {
	view *View
}

var _ Unlock = (*UnlockHuman)(nil)

func (v *UnlockHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *UnlockHuman) ForceUnlockCancelled(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)

	v.view.streams.Println("Cancelled force-unlock.")
}

func (v *UnlockHuman) UnlockSuccessful(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)

	outputUnlockSuccess := `
[reset][bold][green]Terraform state has been successfully unlocked![reset][green]

The state has been unlocked, and Terraform commands should now be able to
obtain a new lock on the remote state.
`

	v.view.streams.Println(v.view.colorize.Color(strings.TrimSpace(outputUnlockSuccess)))
}
