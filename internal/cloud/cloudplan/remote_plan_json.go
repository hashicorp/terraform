// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloudplan

import (
	"github.com/hashicorp/terraform/internal/plans"
)

// RemotePlanJSON is a wrapper struct that associates a pre-baked JSON plan with
// several pieces of metadata that can't be derived directly from the JSON
// contents and must instead be discovered from a tfe.Run or tfe.Plan. The
// wrapper is useful for moving data between the Cloud backend (which is the
// only thing able to fetch the JSON and determine values for the metadata) and
// the command.ShowCommand and views.Show interface (which need to have all of
// this information together).
type RemotePlanJSON struct {
	// The raw bytes of json we got from the API.
	JSONBytes []byte
	// Indicates whether the json bytes are the "redacted json plan" format, or
	// the unredacted stable "external json plan" format. These formats are
	// actually very different under the hood; the redacted one can be decoded
	// directly into a jsonformat.Plan struct and is intended for formatting a
	// plan for human consumption, while the unredacted one matches what is
	// returned by the jsonplan.Marshal() function, cannot be directly decoded
	// into a public type (it's actually a jsonplan.plan struct), and will
	// generally be spat back out verbatim.
	Redacted bool
	// Normal/destroy/refresh. Required by (jsonformat.Renderer).RenderHumanPlan.
	Mode plans.Mode
	// Unchanged/errored. Required by (jsonformat.Renderer).RenderHumanPlan.
	Qualities []plans.Quality
	// A human-readable header with a link to view the associated run in the
	// HCP Terraform UI.
	RunHeader string
	// A human-readable footer with information relevant to the likely next
	// actions for this plan.
	RunFooter string
}
