// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Import interface {
	// MissingResourceBlockError is called when a resource block is missing in the configuration.
	// The command should instead prepare a diagnostic for this error, but our diagnostics printer
	// currently doesn't support including a code example in the detail.
	//
	// Original code comment from command/import.go:
	//
	// This is not a diagnostic because currently our diagnostics printer
	// doesn't support having a code example in the detail, and there's
	// a code example in this message.
	// TODO: Improve the diagnostics printer so we can use it for this
	// message.
	MissingResourceBlockError(addr addrs.AbsResourceInstance, modulePath string, resource addrs.Resource)

	// InvalidAddressHelp shows help for the user when they've provided an invalid resource address.
	InvalidAddressHelp()

	// The import has completed successfully
	Success()

	Diagnostics(diags tfdiags.Diagnostics)
}

// NewImport returns an initialized Import implementation for the given ViewType.
func NewImport(vt arguments.ViewType, view *View) Import {
	switch vt {
	case arguments.ViewHuman:
		return &ImportHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The ImportHuman implementation renders diagnostics in a human-readable form.
type ImportHuman struct {
	view *View
}

var _ Import = (*ImportHuman)(nil)

func (v *ImportHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *ImportHuman) MissingResourceBlockError(addr addrs.AbsResourceInstance, modulePath string, resource addrs.Resource) {
	msg := `[reset][bold][red]Error:[reset][bold] resource address %q does not exist in the configuration.[reset]

Before importing this resource, please create its configuration in %s. For example:

resource %q %q {
  # (resource arguments)
}
`
	// Print to stderr
	v.view.streams.Eprintln(
		v.view.colorize.Color(
			fmt.Sprintf(
				msg,
				addr, modulePath, resource.Type, resource.Name,
			),
		),
	)
}

func (v *ImportHuman) InvalidAddressHelp() {
	v.view.streams.Println(`For information on valid syntax, see:
https://developer.hashicorp.com/terraform/cli/state/resource-addressing`)
}

func (v *ImportHuman) Success() {
	v.view.streams.Println(
		v.view.colorize.Color(`[reset][green]
Import successful!

The resources that were imported are shown above. These resources are now in
your Terraform state and will henceforth be managed by Terraform.
`),
	)
}
