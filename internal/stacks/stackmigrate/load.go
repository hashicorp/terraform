// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	backendinit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Load loads a state from the given configPath. The configuration at configPath
// must have been initialized via `terraform init` before calling this function.
func Load(configurationPath, backendStatePath, workspace string) (*states.State, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First, we'll load the "backend state". This should have been initialised
	// by the `terraform init` command, and contains the configuration for the
	// backend that we're using.

	backendStateManager := &clistate.LocalState{
		Path: backendStatePath,
	}
	if err := backendStateManager.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading backend state: %s", err))
		return states.NewState(), diags
	}
	backendState := backendStateManager.State()

	// Now that we have the backend state, we can initialise the backend itself
	// based on what we had from the `terraform init` command.

	var backend backend.Backend
	if backendState == nil {
		backend = local.New()
		moreDiags := backend.Configure(cty.ObjectVal(map[string]cty.Value{
			"path":          cty.StringVal(filepath.Join(configurationPath, local.DefaultStateFilename)),
			"workspace_dir": cty.StringVal(filepath.Join(configurationPath, local.DefaultWorkspaceDir)),
		}))
		diags = diags.Append(moreDiags)
		if diags.HasErrors() {
			return states.NewState(), diags
		}
	} else {
		f := backendinit.Backend(backendState.Backend.Type)
		if f == nil {
			diags = diags.Append(fmt.Errorf("unknown backend type %q", backendState.Backend.Type))
			return states.NewState(), diags
		}

		backend := f()
		schema := backend.ConfigSchema()
		config, err := backendState.Backend.Config(schema)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to decode current backend config",
				fmt.Sprintf("The backend configuration created by the most recent run of \"terraform init\" could not be decoded: %s. The configuration may have been initialized by an earlier version that used an incompatible configuration structure. Run \"terraform init -reconfigure\" to force re-initialization of the backend.", err),
			))
			return states.NewState(), diags
		}

		var moreDiags tfdiags.Diagnostics

		config, moreDiags = backend.PrepareConfig(config)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return states.NewState(), diags
		}

		moreDiags = backend.Configure(config)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return states.NewState(), diags
		}
	}

	// The backend is initialised and configured, so now we can load the state
	// from the backend.

	stateManager, err := backend.StateMgr(workspace)
	if err != nil {
		diags = diags.Append(fmt.Errorf("error loading state: %s", err))
		return states.NewState(), diags
	}

	// We'll lock the backend here to ensure that we don't have any concurrent
	// operations on the state. If this fails, we'll return an error and the
	// user should retry the migration later when nothing is currently updating
	// the state.

	id, err := stateManager.Lock(statemgr.NewLockInfo())
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to lock state", fmt.Sprintf("The state is currently locked by another operation: %s. Please retry the migration later.", err)))
		return states.NewState(), diags
	}
	if err := stateManager.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading state: %s", err))
		return states.NewState(), diags
	}
	state := stateManager.State()

	// Remember to unlock the state when we're done.

	if err := stateManager.Unlock(id); err != nil {
		// If we couldn't unlock the state, we'll warn about that but the
		// migration can actually continue.
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Warning, "Failed to unlock state", fmt.Sprintf("The state was successfully loaded but could not be unlocked: %s. The migration can continue but the state many need to be unlocked manually.", err)))
	}

	return state, diags
}
