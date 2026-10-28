// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateIdentities interface {
	Identities(identities map[string]json.RawMessage)
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewStateIdentities(viewType arguments.ViewType, view *View) StateIdentities {
	switch viewType {
	case arguments.ViewJSON:
		return &StateIdentitiesJSON{
			view:       NewJSONStaticView[StateIdentitiesOutput](view),
			legacyView: view,
		}
	case arguments.ViewHuman:
		return &StateIdentitiesHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

type StateIdentitiesJSON struct {
	view *JSONStaticView[StateIdentitiesOutput]

	// TODO: Remove and report diagnostics as JSON
	legacyView *View
}

// TODO: Change to struct w/ FormatVersion & Diagnostics
type StateIdentitiesOutput map[string]json.RawMessage

func (v *StateIdentitiesJSON) Identities(identities map[string]json.RawMessage) {
	v.view.Print(identities)
}

func (v *StateIdentitiesJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.legacyView.Diagnostics(diags)
}

type StateIdentitiesHuman struct {
	view *View
}

func (v *StateIdentitiesHuman) Identities(identities map[string]json.RawMessage) {
	panic("not implemented")
}

func (v *StateIdentitiesHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
