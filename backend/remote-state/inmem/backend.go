package inmem

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

// we keep the states and locks in package-level variables, so that they can be
// accessed from multiple instances of the backend. This better emulates
// backend instances accessing a single remote data store.
var (
	states stateMap
	locks  lockMap
)

func init() {
	Reset()
}

// Reset clears out all existing state and lock data.
// This is used to initialize the package during init, as well as between
// tests.
func Reset() {
	states = stateMap{
		m: map[string]*remote.State{},
	}

	locks = lockMap{
		m: map[string]*state.LockInfo{},
	}
}

// New creates a new backend for Inmem remote state.
func New() backend.Backend {
	// Set the schema
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"lock_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "initializes the state in a locked configuration",
			},
		},
	}
	backend := &Backend{Backend: s}
	backend.Backend.ConfigureFunc = backend.configure
	return backend
}

type Backend struct {
	*schema.Backend
}

func (b *Backend) configure(ctx context.Context) error {
	states.Lock()
	defer states.Unlock()

	defaultClient := &RemoteClient{
		Name: backend.DefaultStateName,
	}

	states.m[backend.DefaultStateName] = &remote.State{
		Client: defaultClient,
	}

	// set the default client lock info per the test config
	data := schema.FromContextBackendConfig(ctx)
	if v, ok := data.GetOk("lock_id"); ok && v.(string) != "" {
		info := state.NewLockInfo()
		info.ID = v.(string)
		info.Operation = "test"
		info.Info = "test config"

		locks.lock(backend.DefaultStateName, info)
	}

	return nil
}

func (b *Backend) States() ([]string, error) {
	states.Lock()
	defer states.Unlock()

	var workspaces []string

	for s := range states.m {
		workspaces = append(workspaces, s)
	}

	sort.Strings(workspaces)
	return workspaces, nil
}

func (b *Backend) DeleteState(name string) error {
	states.Lock()
	defer states.Unlock()

	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	delete(states.m, name)
	return nil
}

func (b *Backend) State(name string) (state.State, error) {
	states.Lock()
	defer states.Unlock()

	s := states.m[name]
	if s == nil {
		s = &remote.State{
			Client: &RemoteClient{
				Name: name,
			},
		}
		states.m[name] = s

		// to most closely replicate other implementations, we are going to
		// take a lock and create a new state if it doesn't exist.
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "init"
		lockID, err := s.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock inmem state: %s", err)
		}
		defer s.Unlock(lockID)

		// If we have no state, we have to create an empty state
		if v := s.State(); v == nil {
			if err := s.WriteState(terraform.NewState()); err != nil {
				return nil, err
			}
			if err := s.PersistState(); err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}

type stateMap struct {
	sync.Mutex
	m map[string]*remote.State
}

// Global level locks for inmem backends.
type lockMap struct {
	sync.Mutex
	m map[string]*state.LockInfo
}

func (l *lockMap) lock(name string, info *state.LockInfo) (string, error) {
	l.Lock()
	defer l.Unlock()

	lockInfo := l.m[name]
	if lockInfo != nil {
		lockErr := &state.LockError{
			Info: lockInfo,
		}

		lockErr.Err = errors.New("state locked")
		// make a copy of the lock info to avoid any testing shenanigans
		*lockErr.Info = *lockInfo
		return "", lockErr
	}

	info.Created = time.Now().UTC()
	l.m[name] = info

	return info.ID, nil
}

func (l *lockMap) unlock(name, id string) error {
	l.Lock()
	defer l.Unlock()

	lockInfo := l.m[name]

	if lockInfo == nil {
		return errors.New("state not locked")
	}

	lockErr := &state.LockError{
		Info: &state.LockInfo{},
	}

	if id != lockInfo.ID {
		lockErr.Err = errors.New("invalid lock id")
		*lockErr.Info = *lockInfo
		return lockErr
	}

	delete(l.m, name)
	return nil
}
