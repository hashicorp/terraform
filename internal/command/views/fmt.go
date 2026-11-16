// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Fmt is the view for the `terraform fmt` command.
type Fmt interface {
	// Print the path of a file that requires formatting.
	// This being invoked doesn't mean the file was actually modified; -write=false would stop that.
	FileFormatted(path string)

	// Print the diff of a file that requires formatting.
	// This being invoked doesn't mean the file was actually modified; -write=false would stop that.
	Diff(diff []byte)

	// Print the formatted content of a file.
	// This being invoked doesn't mean the file was actually modified; -write=false would stop that.
	FormattedContent(data string)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewFmt(viewType arguments.ViewType, view *View) Fmt {
	switch viewType {
	case arguments.ViewHuman:
		return &FmtHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

type FmtHuman struct {
	view *View
}

var _ Fmt = (*FmtHuman)(nil)

func (v *FmtHuman) FileFormatted(path string) {
	v.view.streams.Println(path)
}

func (v *FmtHuman) Diff(diff []byte) {
	v.view.streams.Print(string(diff))
}

func (v *FmtHuman) FormattedContent(data string) {
	v.view.streams.Print(data)
}

func (v *FmtHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
