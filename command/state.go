package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

// StateOpts are options to get the state for a command.
type StateOpts struct {
	// LocalPath is the path where the state is stored locally.
	//
	// LocalPathOut is the path where the local state will be saved. If this
	// isn't set, it will be saved back to LocalPath.
	LocalPath    string
	LocalPathOut string

	// RemotePath is the path where the remote state cache would be.
	RemotePath string

	// BackupPath is the path where the backup will be placed. If not set,
	// it is assumed to be the path where the state is stored locally
	// plus the DefaultBackupExtension.
	BackupPath string
}

// State returns the proper state.State implementation to represent the
// current environment.
//
// localPath is the path to where state would be if stored locally.
// dataDir is the path to the local data directory where the remote state
// cache would be stored.
func State(opts *StateOpts) (state.State, string, error) {
	var result state.State
	var resultPath string

	// Get the remote state cache path
	if opts.RemotePath != "" {
		if _, err := os.Stat(opts.RemotePath); err == nil {
			// We have a remote state, initialize that.
			result, err = remoteStateFromPath(opts.RemotePath)
			if err != nil {
				return nil, "", err
			}
			resultPath = opts.RemotePath
		}
	}

	// Do we have a local state?
	if opts.LocalPath != "" {
		local := &state.LocalState{
			Path:    opts.LocalPath,
			PathOut: opts.LocalPathOut,
		}
		err := local.RefreshState()
		if err != nil {
			isNotExist := false
			errwrap.Walk(err, func(e error) {
				if !isNotExist && os.IsNotExist(e) {
					isNotExist = true
				}
			})
			if isNotExist {
				err = nil
			}
		} else {
			if result != nil {
				if !local.State().Empty() {
					// We already have a remote state... that is an error.
					return nil, "", fmt.Errorf(
						"Remote state found, but state file '%s' also present.",
						opts.LocalPath)
				}

				// Empty state
				local = nil
			}
		}
		if err != nil {
			return nil, "", errwrap.Wrapf(
				"Error reading local state: {{err}}", err)
		}

		if local != nil {
			result = local
			resultPath = opts.LocalPath
			if opts.LocalPathOut != "" {
				resultPath = opts.LocalPathOut
			}
		}
	}

	// If we have a result, make sure to back it up
	if result != nil {
		backupPath := resultPath + DefaultBackupExtention
		if opts.BackupPath != "" {
			backupPath = opts.BackupPath
		}

		result = &state.BackupState{
			Real: result,
			Path: backupPath,
		}
	}

	// Return whatever state we have
	return result, resultPath, nil
}

// StateFromPlan gets our state from the plan.
func StateFromPlan(
	localPath string, plan *terraform.Plan) (state.State, string, error) {
	var result state.State
	resultPath := localPath
	if plan != nil && plan.State != nil &&
		plan.State.Remote != nil && plan.State.Remote.Type != "" {
		var err error

		// It looks like we have a remote state in the plan, so
		// we have to initialize that.
		resultPath = filepath.Join(DefaultDataDir, DefaultStateFilename)
		result, err = remoteState(plan.State, resultPath, false)
		if err != nil {
			return nil, "", err
		}
	}

	if result == nil {
		local := &state.LocalState{Path: resultPath}
		local.SetState(plan.State)
		result = local
	}

	// If we have a result, make sure to back it up
	result = &state.BackupState{
		Real: result,
		Path: resultPath + DefaultBackupExtention,
	}

	return result, resultPath, nil
}

func remoteState(
	local *terraform.State,
	localPath string, refresh bool) (state.State, error) {
	// If there is no remote settings, it is an error
	if local.Remote == nil {
		return nil, fmt.Errorf("Remote state cache has no remote info")
	}

	// Initialize the remote client based on the local state
	client, err := remote.NewClient(local.Remote.Type, local.Remote.Config)
	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf(
			"Error initializing remote driver '%s': {{err}}",
			local.Remote.Type), err)
	}

	// Create the remote client
	durable := &remote.State{Client: client}

	// Create the cached client
	cache := &state.CacheState{
		Cache:   &state.LocalState{Path: localPath},
		Durable: durable,
	}

	if refresh {
		// Refresh the cache
		if err := cache.RefreshState(); err != nil {
			return nil, errwrap.Wrapf(
				"Error reloading remote state: {{err}}", err)
		}
		switch cache.RefreshResult() {
		case state.CacheRefreshNoop:
		case state.CacheRefreshInit:
		case state.CacheRefreshLocalNewer:
		case state.CacheRefreshUpdateLocal:
			// Write our local state out to the durable storage to start.
			if err := cache.WriteState(local); err != nil {
				return nil, errwrap.Wrapf(
					"Error preparing remote state: {{err}}", err)
			}
			if err := cache.PersistState(); err != nil {
				return nil, errwrap.Wrapf(
					"Error preparing remote state: {{err}}", err)
			}
		default:
			return nil, errwrap.Wrapf(
				"Error initilizing remote state: {{err}}", err)
		}
	}

	return cache, nil
}

func remoteStateFromPath(path string) (state.State, error) {
	// First create the local state for the path
	local := &state.LocalState{Path: path}
	if err := local.RefreshState(); err != nil {
		return nil, err
	}
	localState := local.State()

	return remoteState(localState, path, true)
}
