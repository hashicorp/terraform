// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	encJson "encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/moduleref"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Modules interface {
	// Display renders the list of module entries.
	Display(manifest moduleref.Manifest) int

	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewModules(vt arguments.ViewType, view *View) Modules {
	switch vt {
	case arguments.ViewJSON:
		return &ModulesJSON{view: view}
	case arguments.ViewHuman:
		return &ModulesHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type ModulesHuman struct {
	view *View
}

var _ Modules = (*ModulesHuman)(nil)

func (v *ModulesHuman) Display(manifest moduleref.Manifest) int {
	return 0
}

func (v *ModulesHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type ModulesJSON struct {
	view *View
}

var _ Modules = (*ModulesHuman)(nil)

func (v *ModulesJSON) Display(manifest moduleref.Manifest) int {
	var bytes []byte
	var err error
	if bytes, err = encJson.Marshal(manifest); err != nil {
		v.view.streams.Eprintf("error marshalling manifest: %v", err)
		return 1
	}

	v.view.streams.Println(string(bytes))
	return 0
}

func (v *ModulesJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
