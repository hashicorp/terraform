// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Logout interface {
	// There is no stored credential for the given host; nothing to log out from!
	NoCredentialsStored(host string)

	// Logout process impacts a credentials helper
	NotifyTokenRemovedFromCredentialsHelper(host, credsHelperType string)

	// Logout process impacts a file
	NotifyTokenRemovedFromFile(host, file string)

	// Successful logout message
	LogoutSuccess(host string)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewLogout(viewType arguments.ViewType, view *View) Logout {
	switch viewType {
	case arguments.ViewHuman:
		return &LogoutHuman{
			view: view,
		}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

type LogoutHuman struct {
	view *View
}

func (v *LogoutHuman) NoCredentialsStored(host string) {
	v.view.streams.Print(
		fmt.Sprintf(
			"No credentials for %s are stored.\n\n",
			host,
		),
	)
}

func (v *LogoutHuman) NotifyTokenRemovedFromCredentialsHelper(host, credsHelperType string) {
	v.view.streams.Print(
		fmt.Sprintf("Removing the stored credentials for %s from the configured\n%q credentials helper.\n\n", host, credsHelperType),
	)
}

func (v *LogoutHuman) NotifyTokenRemovedFromFile(host, file string) {
	v.view.streams.Print(
		fmt.Sprintf("Removing the stored credentials for %s from the following file:\n    %s\n\n", host, file),
	)
}

func (v *LogoutHuman) LogoutSuccess(host string) {
	v.view.streams.Print(
		v.view.colorize.Color(
			fmt.Sprintf("[green][bold]Success![reset] [bold]Terraform has removed the stored API token for %s.[reset]\n\n", host),
		),
	)
}

func (v *LogoutHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
