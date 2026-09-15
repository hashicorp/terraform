// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Get is the view used for the `get` command.
type Get interface {
	ModuleInstallationLogger
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewGet(viewType arguments.ViewType, view *View) Get {
	switch viewType {
	case arguments.ViewHuman:
		return &GetHuman{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

// GetHuman renders module installation progress and diagnostics for direct consumption.
type GetHuman struct {
	view *View
}

var _ Get = (*GetHuman)(nil)

func (v *GetHuman) LogModuleDownload(message string) {
	v.view.streams.Println(message)
}

func (v *GetHuman) LogModuleInstallation(message string) {
	v.view.streams.Println(message)
}

func (v *GetHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
