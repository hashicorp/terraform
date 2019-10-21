package swift

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
)

const (
	objectEnvPrefix = "env-"
	delimiter       = "/"
)

func (b *Backend) Workspaces() ([]string, error) {
	client := &RemoteClient{
		client:           b.client,
		container:        b.container,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		expireSecs:       b.expireSecs,
		lockState:        b.lock,
	}

	// List our container objects
	objectNames, err := client.ListObjectsNames(objectEnvPrefix, delimiter)

	if err != nil {
		return nil, err
	}

	// Find the envs, we use a map since we can get duplicates with
	// path suffixes.
	envs := map[string]struct{}{}
	for _, object := range objectNames {
		object = strings.TrimPrefix(object, objectEnvPrefix)
		object = strings.TrimSuffix(object, delimiter)

		// Ignore objects that still contain a "/"
		// as we dont store states in subdirectories
		if idx := strings.Index(object, delimiter); idx >= 0 {
			continue
		}

		// swift is eventually consistent, thus a deleted object may
		// be listed in objectList. To ensure consistency, we query
		// each object  with a "newest" arg set to true
		payload, err := client.get(b.objectName(object))
		if err != nil {
			return nil, err
		}
		if payload == nil {
			// object doesn't exist anymore. skipping.
			continue
		}

		envs[object] = struct{}{}
	}

	result := make([]string, 1, len(envs)+1)
	result[0] = backend.DefaultStateName

	for k, _ := range envs {
		result = append(result, k)
	}

	return result, nil
}

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	client := &RemoteClient{
		client:           b.client,
		container:        b.container,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		expireSecs:       b.expireSecs,
		objectName:       b.objectName(name),
		lockState:        b.lock,
	}

	// Delete our object
	err := client.Delete()

	return err
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	if name == "" {
		return nil, fmt.Errorf("missing state name")
	}

	client := &RemoteClient{
		client:           b.client,
		container:        b.container,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		expireSecs:       b.expireSecs,
		objectName:       b.objectName(name),
		lockState:        b.lock,
	}

	var stateMgr state.State = &remote.State{Client: client}

	// If we're not locking, disable it
	if !b.lock {
		stateMgr = &state.LockDisabled{Inner: stateMgr}
	}

	// Check to see if this state already exists.
	// If we're trying to force-unlock a state, we can't take the lock before
	// fetching the state. If the state doesn't exist, we have to assume this
	// is a normal create operation, and take the lock at that point.
	//
	// If we need to force-unlock, but for some reason the state no longer
	// exists, the user will have to use openstack tools to manually fix the
	// situation.
	existing, err := b.Workspaces()
	if err != nil {
		return nil, err
	}

	exists := false
	for _, s := range existing {
		if s == name {
			exists = true
			break
		}
	}

	// We need to create the object so it's listed by States.
	if !exists {
		// the default state always exists
		if name == backend.DefaultStateName {
			return stateMgr, nil
		}

		// Grab a lock, we use this to write an empty state if one doesn't
		// exist already. We have to write an empty state as a sentinel value
		// so States() knows it exists.
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockId, err := stateMgr.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock state in Swift: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockId); err != nil {
				return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockId, err)
			}

			return parent
		}

		// Grab the value
		if err := stateMgr.RefreshState(); err != nil {
			err = lockUnlock(err)
			return nil, err
		}

		// If we have no state, we have to create an empty state
		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(states.NewState()); err != nil {
				err = lockUnlock(err)
				return nil, err
			}
			if err := stateMgr.PersistState(); err != nil {
				err = lockUnlock(err)
				return nil, err
			}
		}

		// Unlock, the state should now be initialized
		if err := lockUnlock(nil); err != nil {
			return nil, err
		}
	}

	return stateMgr, nil
}

func (b *Backend) objectName(name string) string {
	if name != backend.DefaultStateName {
		name = fmt.Sprintf("%s%s/%s", objectEnvPrefix, name, b.stateName)
	} else {
		name = b.stateName
	}

	return name
}

const errStateUnlock = `
Error unlocking Swift state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
The Swift backend acquires a lock during initialization to ensure
the minimum required keys are prepared.
`
