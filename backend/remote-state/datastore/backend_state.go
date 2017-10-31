package datastore

import (
	"context"
	"fmt"
	"sort"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/datastore"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

const errStateUnlock = `Error unlocking Datastore state. Lock ID: %s

Error: %s

You may have to force-unlock this state in order to use it again.
The Datastore backend acquires a lock during initialization to ensure
the minimum required key/values are prepared.`

// State returns the current state for this workspace. This state may
// not be loaded locally: the proper APIs should be called on state.State
// to load the state. If the state.State is a state.Locker, it's up to the
// caller to call Lock and Unlock as needed.
//
// If the named state doesn't exist it will be created. The "default" state
// is always assumed to exist.
func (b *Backend) State(name string) (state.State, error) {
	s := &remote.State{Client: newRemoteClient(b.ds, b.ns, name)}

	// We only initialise state so it shows up in calls to b.States().
	// b.States() knows the default state always exists, so no need to
	// initialise it.
	if name == backend.DefaultStateName {
		return s, nil
	}

	return s, initState(s)
}

func initState(s state.State) error {
	info := state.NewLockInfo()
	info.Operation = "init"
	id, err := s.Lock(info)
	if err != nil {
		return fmt.Errorf("failed to lock state for initialisation: %v", err)
	}

	unlock := func(why error) error {
		if err := s.Unlock(id); err != nil {
			return fmt.Errorf(errStateUnlock, id, err)
		}
		return why
	}

	// This state exists. No need to initialise it.
	if err := s.RefreshState(); err != nil || s.State() != nil {
		return unlock(err)
	}

	if err := s.WriteState(terraform.NewState()); err != nil {
		return unlock(err)
	}
	if err := s.PersistState(); err != nil {
		return unlock(err)
	}

	return unlock(nil)
}

// DeleteState removes the named state if it exists. It is an error
// to delete the default state.
//
// DeleteState does not prevent deleting a state that is in use. It is the
// responsibility of the caller to hold a Lock on the state when calling
// this method.
func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("cannot delete default state")
	}
	if err := newRemoteClient(b.ds, b.ns, name).Delete(); err != nil {
		return fmt.Errorf("cannot delete state: %v", err)
	}
	return nil
}

// States returns a list of configured named states.
func (b *Backend) States() ([]string, error) {
	states := []string{backend.DefaultStateName}
	e := &entityState{}
	for i := b.ds.Run(context.TODO(), datastore.NewQuery(kindTerraformState).Namespace(b.ns).KeysOnly()); ; {
		k, err := i.Next(e)
		if err == iterator.Done {
			sort.Strings(states[1:])
			return states, nil
		}
		if err != nil {
			return nil, fmt.Errorf("cannot query workspaces from Google Datastore: %v", err)
		}
		if k.Name == backend.DefaultStateName {
			continue
		}
		states = append(states, k.Name)
	}
}
