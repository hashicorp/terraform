package state

import (
	"github.com/hashicorp/terraform/terraform"
)

// CacheState is an implementation of the state interfaces that uses
// a StateReadWriter for a local cache.
type CacheState struct {
	Cache   CacheStateCache
	Durable CacheStateDurable

	state *terraform.State
}

// StateReader impl.
func (s *CacheState) State() *terraform.State {
	return s.state
}

// WriteState will write and persist the state to the cache.
//
// StateWriter impl.
func (s *CacheState) WriteState(state *terraform.State) error {
	if err := s.Cache.WriteState(state); err != nil {
		return err
	}

	return s.Cache.PersistState()
}

// RefreshState will refresh both the cache and the durable states. It
// can return a myriad of errors (defined at the top of this file) depending
// on potential conflicts that can occur while doing this.
//
// If the durable state is newer than the local cache, then the local cache
// will be replaced with the durable.
//
// StateRefresher impl.
func (s *CacheState) RefreshState() error {
	return nil
}

// PersistState takes the local cache, assuming it is newer than the remote
// state, and persists it to the durable storage. If you want to challenge the
// assumption that the local state is the latest, call a RefreshState prior
// to this.
//
// StatePersister impl.
func (s *CacheState) PersistState() error {
	if err := s.Durable.WriteState(s.state); err != nil {
		return err
	}

	return s.Durable.PersistState()
}

// CacheStateCache is the meta-interface that must be implemented for
// the cache for the CacheState.
type CacheStateCache interface {
	StateReader
	StateWriter
	StatePersister
}

// CacheStateDurable is the meta-interface that must be implemented for
// the durable storage for CacheState.
type CacheStateDurable interface {
	StateWriter
	StatePersister
}
