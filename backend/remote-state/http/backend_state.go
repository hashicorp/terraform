package http

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

const (
	stateFileSuffix = ".tfstate"
	lockFileSuffix  = ".tflock"
)

// States get a list of all states.
func (b *Backend) States() ([]string, error) {
	var result []string
	client := &RemoteClient{
		client:   b.client,
		address:  b.address,
		username: b.username,
		password: b.password,
	}

	resp, err := client.Get()
	if err != nil {
		return nil, err
	}
	// Read in the body
	buff := string(resp.Data)
	// Get all data and make it a slice.
	// The REST API should return all the files from the root of the project name.
	// This means we get all the state files of a project
	// ex. buff := "default.tfstate default.tflock foo.tfstate bar.tfstate"
	buff = strings.Replace(buff, ",", " ", -1)
	// make it a slice so we can search and loop thru it
	buffSlice := strings.Fields(buff)
	// tracks whether or not we've seen the default state
	foundDefaultState := false
	for _, el := range buffSlice {
		// not a TF state file, skip
		extName := filepath.Ext(el)
		if extName != stateFileSuffix {
			continue
		}
		// basename of the file with extension
		fName := filepath.Base(el)
		// remove the extension to get file name, add to list of results
		file := fName[:len(fName)-len(extName)]
		result = append(result, file)
		// we found the default state file
		if file == backend.DefaultStateName {
			foundDefaultState = true
		}
	}
	// did not encounter the default state, add it now
	if !foundDefaultState {
		return append(result, backend.DefaultStateName), nil
	}
	return result, nil
}

// DeleteState deletes a state file
func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}
	client, err := b.remoteClient(name)
	if err != nil {
		return err
	}

	return client.Delete()
}

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, errors.New("missing state name")
	}
	client := &RemoteClient{
		client:        b.client,
		address:       b.statePath(name),
		updateMethod:  b.updateMethod,
		lockAddress:   b.lockPath(name),
		unlockAddress: b.lockPath(name),
		lockMethod:    b.lockMethod,
		unlockMethod:  b.unlockMethod,
		username:      b.username,
		password:      b.password,
	}
	return client, nil
}

// State reads the state file
func (b *Backend) State(name string) (state.State, error) {
	client, err := b.remoteClient(name)
	if err != nil {
		return nil, err
	}

	stateMgr := &remote.State{Client: client}

	// Grab the value
	if err := stateMgr.RefreshState(); err != nil {
		return nil, err
	}

	// If we have no state, we have to create an empty state
	if v := stateMgr.State(); v == nil {
		// take a lock on this state while we write it
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockID, err := client.Lock(lockInfo)

		if err != nil {
			return nil, fmt.Errorf("failed to lock http state: %s", err)
		}

		// Local helper function so we can call it multiple places
		lockUnlock := func(parent error) error {
			if err := stateMgr.Unlock(lockID); err != nil {
				return fmt.Errorf(strings.TrimSpace(errStateUnlock), lockID, err)
			}
			return parent
		}

		if err := stateMgr.WriteState(terraform.NewState()); err != nil {
			return nil, lockUnlock(err)
		}
		if err := stateMgr.PersistState(); err != nil {
			return nil, lockUnlock(err)
		}

		// Unlock, the state should now be initialized
		if err := lockUnlock(nil); err != nil {
			return nil, err
		}
	}
	return stateMgr, nil
}

// Construct the path of the state file based on named state
func (b *Backend) statePath(name string) string {
	paths := []string{b.address, "/", name, stateFileSuffix}
	var buf bytes.Buffer
	for _, p := range paths {
		buf.WriteString(p)
	}
	path := buf.String()

	return path
}

// Construct the path of the lock file based on named state
func (b *Backend) lockPath(name string) string {
	paths := []string{b.address, "/", name, lockFileSuffix}
	var buf bytes.Buffer
	for _, p := range paths {
		buf.WriteString(p)
	}
	path := buf.String()

	return path
}

const errStateUnlock = `
Error unlocking http state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
`
