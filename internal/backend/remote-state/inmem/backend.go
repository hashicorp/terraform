// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmem

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	statespkg "github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
		m: map[string]*statemgr.LockInfo{},
	}
}

// New creates a new backend for Inmem remote state.
func New() backend.Backend {
	if os.Getenv("TF_INMEM_TEST") != "" {
		// We use a different schema for testing. This isn't user facing unless they
		// dig into the code.
		fmt.Println("TF_INMEM_TEST is set: Using test schema for the inmem backend")

		return &Backend{
			Base: backendbase.Base{
				Schema: &configschema.Block{
					Attributes: testSchemaAttrs(),
				},
			},
		}
	}

	// Default schema that's user-facing
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: defaultSchemaAttrs,
			},
		},
	}
}

var defaultSchemaAttrs = map[string]*configschema.Attribute{
	"lock_id": {
		Type:        cty.String,
		Optional:    true,
		Description: "initializes the state in a locked configuration",
	},
}

func testSchemaAttrs() map[string]*configschema.Attribute {
	var newSchema = make(map[string]*configschema.Attribute)
	maps.Copy(newSchema, defaultSchemaAttrs)

	// Append test-specific parts of schema
	newSchema["test_nesting_single"] = &configschema.Attribute{
		Description: "An attribute that contains nested attributes, where nesting mode is NestingSingle",
		NestedType: &configschema.Object{
			Nesting: configschema.NestingSingle,
			Attributes: map[string]*configschema.Attribute{
				"child": {
					Type:        cty.String,
					Optional:    true,
					Description: "A nested attribute inside the parent attribute `test_nesting_single`",
				},
			},
		},
	}
	return newSchema
}

type Backend struct {
	backendbase.Base
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	states.Lock()
	defer states.Unlock()

	defaultClient := &RemoteClient{
		Name: backend.DefaultStateName,
	}

	states.m[backend.DefaultStateName] = &remote.State{
		Client: defaultClient,
	}

	// set the default client lock info per the test config
	if v := configVal.GetAttr("lock_id"); !v.IsNull() {
		info := statemgr.NewLockInfo()
		info.ID = v.AsString()
		info.Operation = "test"
		info.Info = "test config"

		locks.lock(backend.DefaultStateName, info)
	}

	return nil
}

func (b *Backend) Workspaces() ([]string, error) {
	states.Lock()
	defer states.Unlock()

	var workspaces []string

	for s := range states.m {
		workspaces = append(workspaces, s)
	}

	sort.Strings(workspaces)
	return workspaces, nil
}

func (b *Backend) DeleteWorkspace(name string, _ bool) error {
	states.Lock()
	defer states.Unlock()

	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	delete(states.m, name)
	return nil
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
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
		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockID, err := s.Lock(lockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to lock inmem state: %s", err)
		}
		defer s.Unlock(lockID)

		// If we have no state, we have to create an empty state
		if v := s.State(); v == nil {
			if err := s.WriteState(statespkg.NewState()); err != nil {
				return nil, err
			}
			if err := s.PersistState(nil); err != nil {
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
	m map[string]*statemgr.LockInfo
}

func (l *lockMap) lock(name string, info *statemgr.LockInfo) (string, error) {
	l.Lock()
	defer l.Unlock()

	lockInfo := l.m[name]
	if lockInfo != nil {
		lockErr := &statemgr.LockError{
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

	lockErr := &statemgr.LockError{
		Info: &statemgr.LockInfo{},
	}

	if id != lockInfo.ID {
		lockErr.Err = errors.New("invalid lock id")
		*lockErr.Info = *lockInfo
		return lockErr
	}

	delete(l.m, name)
	return nil
}
