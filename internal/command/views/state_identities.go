// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type StateIdentities interface {
	Identities(identities map[string]json.RawMessage, diags tfdiags.Diagnostics)
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewStateIdentities(viewType arguments.ViewType, view *View) StateIdentities {
	switch viewType {
	case arguments.ViewJSON:
		return NewStateIdentitiesJSON(view)
	default:
		panic(fmt.Sprintf("unsupported view type: %s", viewType))
	}
}

type StateIdentitiesJSON struct {
	view          *JSONStaticView[StateIdentitiesOutput]
	formatVersion string
}

func NewStateIdentitiesJSON(view *View) *StateIdentitiesJSON {
	// FormatVersion represents the version of the json format and will be
	// incremented for any change to this format that requires changes to a
	// consuming parser.
	const formatVersion = "1.0"

	return &StateIdentitiesJSON{
		view:          NewJSONStaticView[StateIdentitiesOutput](view),
		formatVersion: formatVersion,
	}
}

type StateIdentitiesOutput struct {
	FormatVersion string                     `json:"format_version"`
	Identities    map[string]json.RawMessage `json:"identities"`
	Diagnostics   []*viewsjson.Diagnostic    `json:"diagnostics,omitempty"`
}

func (v *StateIdentitiesJSON) Identities(identities map[string]json.RawMessage, diags tfdiags.Diagnostics) {
	output := StateIdentitiesOutput{
		FormatVersion: v.formatVersion,
		Identities:    identities,
		Diagnostics:   v.view.prepareDiagnostics(diags),
	}
	v.view.Print(output)
}

func (v *StateIdentitiesJSON) Diagnostics(diags tfdiags.Diagnostics) {
	if len(diags) == 0 {
		return
	}

	output := StateIdentitiesOutput{
		FormatVersion: v.formatVersion,
		Diagnostics:   v.view.prepareDiagnostics(diags),
	}

	switch {
	case diags.HasErrors():
		v.view.Eprint(output)
	default:
		v.view.Print(output)
	}
}
