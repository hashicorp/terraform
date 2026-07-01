// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ui

import "github.com/hashicorp/cli"

// WrappedUi wraps the primary output cli.Ui, and redirects Warn calls to Output
// calls. This ensures that warnings are sent to stdout, and are properly
// serialized within the stdout stream.
//
// This behaviour matches the behaviour of the new views package,
// which also sends warnings to stdout.
//
// For more context, see: https://github.com/hashicorp/terraform/pull/27057
type WrappedUi struct {
	cli.Ui
}

func (u *WrappedUi) Warn(msg string) {
	u.Ui.Output(msg)
}
