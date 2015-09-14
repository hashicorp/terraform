package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	//
	// RemoteCache, if true, will set the result to only be the cache
	// and not backed by any real durable storage.
	RemotePath      string
	RemoteCacheOnly bool
	RemoteRefresh   bool

	// BackupPath is the path where the backup will be placed. If not set,
	// it is assumed to be the path where the state is stored locally
	// plus the DefaultBackupExtension.
	BackupPath string
}

// StateResult is the result of calling State and holds various different
// State implementations so they can be accessed directly.
type StateResult struct {
	// State is the final outer state that should be used for all
	// _real_ reads/writes.
	//
	// StatePath is the local path where the state will be stored or
	// cached, no matter whether State is local or remote.
	State     state.State
	StatePath string

	// Local and Remote are the local/remote state implementations, raw
	// and unwrapped by any backups. The paths here are the paths where
	// these state files would be saved.
	Local      *state.LocalState
	LocalPath  string
	Remote     *state.CacheState
	RemotePath string
}

// State returns the proper state.State implementation to represent the
// current environment.
//
// localPath is the path to where state would be if stored locally.
// dataDir is the path to the local data directory where the remote state
// cache would be stored.
func State(opts *StateOpts) (*StateResult, error) {
	result := new(StateResult)

	// Get the remote state cache path
	if opts.RemotePath != "" {
		result.RemotePath = opts.RemotePath

		var remote *state.CacheState
		if opts.RemoteCacheOnly {
			// Setup the in-memory state
			ls := &state.LocalState{Path: opts.RemotePath}
			if err := ls.RefreshState(); err != nil {
				return nil, err
			}
			is := &state.InmemState{}
			is.WriteState(ls.State())

			// Setupt he remote state, cache-only, and refresh it so that
			// we have access to the state right away.
			remote = &state.CacheState{
				Cache:   ls,
				Durable: is,
			}
			if err := remote.RefreshState(); err != nil {
				return nil, err
			}
		} else {
			if _, err := os.Stat(opts.RemotePath); err == nil {
				// We have a remote state, initialize that.
				remote, err = remoteStateFromPath(
					opts.RemotePath,
					opts.RemoteRefresh)
				if err != nil {
					return nil, err
				}
			}
		}

		if remote != nil {
			result.State = remote
			result.StatePath = opts.RemotePath
			result.Remote = remote
		}
	}

	// Do we have a local state?
	if opts.LocalPath != "" {
		local := &state.LocalState{
			Path:    opts.LocalPath,
			PathOut: opts.LocalPathOut,
		}

		// Always store it in the result even if we're not using it
		result.Local = local
		result.LocalPath = local.Path
		if local.PathOut != "" {
			result.LocalPath = local.PathOut
		}

		err := local.RefreshState()
		if err == nil {
			if result.State != nil && !result.State.State().Empty() {
				if !local.State().Empty() {
					// We already have a remote state... that is an error.
					return nil, fmt.Errorf(
						"Remote state found, but state file '%s' also present.",
						opts.LocalPath)
				}

				// Empty state
				local = nil
			}
		}
		if err != nil {
			return nil, errwrap.Wrapf(
				"Error reading local state: {{err}}", err)
		}

		if local != nil {
			result.State = local
			result.StatePath = opts.LocalPath
			if opts.LocalPathOut != "" {
				result.StatePath = opts.LocalPathOut
			}
		}
	}

	// If we have a result, make sure to back it up
	if result.State != nil {
		backupPath := result.StatePath + DefaultBackupExtension
		if opts.BackupPath != "" {
			backupPath = opts.BackupPath
		}

		if backupPath != "-" {
			result.State = &state.BackupState{
				Real: result.State,
				Path: backupPath,
			}
		}
	}

	// Return whatever state we have
	return result, nil
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
		Path: resultPath + DefaultBackupExtension,
	}

	return result, resultPath, nil
}

func remoteState(
	local *terraform.State,
	localPath string, refresh bool) (*state.CacheState, error) {
	// If there is no remote settings, it is an error
	if local.Remote == nil {
		return nil, fmt.Errorf("Remote state cache has no remote info")
	}

	// Initialize the remote client based on the local state
	client, err := remote.NewClient(strings.ToLower(local.Remote.Type), local.Remote.Config)
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
		// All the results below can be safely ignored since it means the
		// pull was successful in some way. Noop = nothing happened.
		// Init = both are empty. UpdateLocal = local state was older and
		// updated.
		//
		// We don't have to do anything, the pull was successful.
		case state.CacheRefreshNoop:
		case state.CacheRefreshInit:
		case state.CacheRefreshUpdateLocal:

		// Our local state has a higher serial number than remote, so we
		// want to explicitly sync the remote side with our local so that
		// the remote gets the latest serial number.
		case state.CacheRefreshLocalNewer:
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
			return nil, fmt.Errorf(
				"Unknown refresh result: %s", cache.RefreshResult())
		}
	}

	return cache, nil
}

func remoteStateFromPath(path string, refresh bool) (*state.CacheState, error) {
	// First create the local state for the path
	local := &state.LocalState{Path: path}
	if err := local.RefreshState(); err != nil {
		return nil, err
	}
	localState := local.State()

	return remoteState(localState, path, refresh)
}
