// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Import is the view interface for the import command.
type Import interface {
	// Diagnostics renders diagnostics (warnings and errors).
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt directs the user to the help command for additional info.
	HelpPrompt()

	// Success renders the import success message.
	Success()

	// MissingResourceConfig renders an error when the target resource
	// is not defined in the configuration, including example configuration.
	MissingResourceConfig(addr, modulePath, resourceType, resourceName string)

	// InvalidAddressReference renders a reference to documentation about
	// valid resource address syntax.
	InvalidAddressReference()
}

// NewImport returns an initialized Import implementation for human-readable
// output.
func NewImport(view *View) Import {
	return &ImportHuman{view: view}
}

// ImportHuman is the human-readable implementation of the Import view.
type ImportHuman struct {
	view *View
}

var _ Import = (*ImportHuman)(nil)

func (v *ImportHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *ImportHuman) HelpPrompt() {
	v.view.HelpPrompt("import")
}

func (v *ImportHuman) Success() {
	v.view.streams.Print(v.view.colorize.Color("[reset][green]\n" + importSuccessMsg))
}

func (v *ImportHuman) MissingResourceConfig(addr, modulePath, resourceType, resourceName string) {
	v.view.streams.Eprint(v.view.colorize.Color(fmt.Sprintf(
		importMissingResourceFmt,
		addr, modulePath, resourceType, resourceName,
	)))
}

func (v *ImportHuman) InvalidAddressReference() {
	v.view.streams.Print(importInvalidAddressReference)
}

const importSuccessMsg = `Import successful!

The resources that were imported are shown above. These resources are now in
your Terraform state and will henceforth be managed by Terraform.
`

const importMissingResourceFmt = `[reset][bold][red]Error:[reset][bold] resource address %q does not exist in the configuration.[reset]

Before importing this resource, please create its configuration in %s. For example:

resource %q %q {
  # (resource arguments)
}
`

const importInvalidAddressReference = `For information on valid syntax, see:
https://developer.hashicorp.com/terraform/cli/state/resource-addressing
`
