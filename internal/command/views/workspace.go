// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The WorkspaceList view is used for the `workspace list` subcommand.
type WorkspaceList interface {
	List(selected string, list []string, diags tfdiags.Diagnostics)
}
