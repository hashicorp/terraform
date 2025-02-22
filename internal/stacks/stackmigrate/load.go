// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/backend/remote"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/workdir"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Loader struct {
	ConfigurationPath string
	BackendStatePath  string
	Workspace         string
	Discovery         *disco.Disco
}

// LoadState loads a state from the given configPath. The configuration at configPath
// must have been initialized via `terraform init` before calling this function.
// The backend state is loaded from backendStatePath. For local backends, there
// is no backend state file, so this can be an empty string.
func (l *Loader) LoadState() (*states.State, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	state := states.NewState()
	backendInit.Init(l.Discovery)

	// First, we'll load the "backend state". This should have been initialised
	// by the `terraform init` command, and contains the configuration for the
	// backend that we're using.
	var backendState *workdir.BackendStateFile
	var err error
	st := &clistate.LocalState{Path: l.BackendStatePath}
	// If the backend state file is not provided, RefreshState will
	// return nil error and State will be empty.
	// In this case, we assume that we're using a local backend.
	if err := st.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading backend state: %s", err))
		return state, diags
	}
	backendState = st.State()

	// Now that we have the backend state, we can initialise the backend itself
	// based on what we had from the `terraform init` command.
	var backend backend.Backend
	var backendConfig cty.Value
	if backendState == nil { // local backend
		backend = local.New()
		backendConfig = cty.ObjectVal(map[string]cty.Value{
			"path":          cty.StringVal(fmt.Sprintf("%s/%s", l.ConfigurationPath, "terraform.tfstate")),
			"workspace_dir": cty.StringVal(l.ConfigurationPath),
		})
	} else {
		initFn := backendInit.Backend(backendState.Backend.Type)
		if initFn == nil {
			diags = diags.Append(fmt.Errorf("unknown backend type %q", backendState.Backend.Type))
			return state, diags
		}

		backend = initFn()
		schema := backend.ConfigSchema()
		config, err := backendState.Backend.Config(schema)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to decode current backend config",
				fmt.Sprintf("The backend configuration created by the most recent run of \"terraform init\" could not be decoded: %s. The configuration may have been initialized by an earlier version that used an incompatible configuration structure. Run \"terraform init -reconfigure\" to force re-initialization of the backend.", err),
			))
			return state, diags
		}

		var moreDiags tfdiags.Diagnostics
		backendConfig, moreDiags = backend.PrepareConfig(config)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return state, diags
		}

		// it's safe to ignore terraform version conflict between the local and remote environments.
		if backendR, ok := backend.(*remote.Remote); ok {
			backendR.IgnoreVersionConflict()
		}
	}

	// Now that we have the backend and its configuration, we can configure it.
	moreDiags := backend.Configure(backendConfig)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return state, diags
	}

	// The backend is initialised and configured, so now we can load the state
	// from the backend.
	stateManager, err := backend.StateMgr(l.Workspace)
	if err != nil {
		diags = diags.Append(fmt.Errorf("error loading state: %s", err))
		return state, diags
	}

	// We'll lock the backend here to ensure that we don't have any concurrent
	// operations on the state. If this fails, we'll return an error and the
	// user should retry the migration later when nothing is currently updating
	// the state.
	id, err := stateManager.Lock(statemgr.NewLockInfo())
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to lock state", fmt.Sprintf("The state is currently locked by another operation: %s. Please retry the migration later.", err)))
		return state, diags
	}
	if err := stateManager.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading state: %s", err))
		return state, diags
	}
	state = stateManager.State()

	// Remember to unlock the state when we're done.
	if err := stateManager.Unlock(id); err != nil {
		// If we couldn't unlock the state, we'll warn about that but the
		// migration can actually continue.
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Warning, "Failed to unlock state", fmt.Sprintf("The state was successfully loaded but could not be unlocked: %s. The migration can continue but the state many need to be unlocked manually.", err)))
	}

	return state, diags
}
