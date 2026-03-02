// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package states

import (
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states"
)

type TestRunState struct {
	// Run and RestoreState represent the run block to use to either destroy
	// or restore the state to. If RestoreState is false, then the state will
	// destroyed, if true it will be restored to the config of the relevant
	// run block.
	Run          *moduletest.Run
	RestoreState bool

	// Manifest is the underlying state manifest for this state.
	Manifest *TestRunManifest

	// State is the actual state.
	State *states.State

	// Backend is the backend where this state should be saved upon test
	// completion.
	Backend backend.Backend
}
