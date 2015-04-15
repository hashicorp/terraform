package state

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

// CacheState is an implementation of the state interfaces that uses
// a StateReadWriter for a local cache.
type CacheState struct {
	Cache   CacheStateCache
	Durable CacheStateDurable

	refreshResult CacheRefreshResult
	state         *terraform.State
}

// StateReader impl.
func (s *CacheState) State() *terraform.State {
	return s.state.DeepCopy()
}

// WriteState will write and persist the state to the cache.
//
// StateWriter impl.
func (s *CacheState) WriteState(state *terraform.State) error {
	if err := s.Cache.WriteState(state); err != nil {
		return err
	}

	s.state = state
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
	// Refresh the durable state
	if err := s.Durable.RefreshState(); err != nil {
		return err
	}

	// Refresh the cached state
	if err := s.Cache.RefreshState(); err != nil {
		return err
	}

	// Handle the matrix of cases that can happen when comparing these
	// two states.
	cached := s.Cache.State()
	durable := s.Durable.State()
	switch {
	case cached == nil && durable == nil:
		// Initialized
		s.refreshResult = CacheRefreshInit
	case cached != nil && durable == nil:
		// Cache is newer than remote. Not a big deal, user can just
		// persist to get correct state.
		s.refreshResult = CacheRefreshLocalNewer
	case cached == nil && durable != nil:
		// Cache should be updated since the remote is set but cache isn't
		s.refreshResult = CacheRefreshUpdateLocal
	case durable.Serial < cached.Serial:
		// Cache is newer than remote. Not a big deal, user can just
		// persist to get correct state.
		s.refreshResult = CacheRefreshLocalNewer
	case durable.Serial > cached.Serial:
		// Cache should be updated since the remote is newer
		s.refreshResult = CacheRefreshUpdateLocal
	case durable.Serial == cached.Serial:
		// They're supposedly equal, verify.
		if cached.Equal(durable) {
			// Hashes are the same, everything is great
			s.refreshResult = CacheRefreshNoop
			break
		}

		// This is very bad. This means we have two state files that
		// have the same serial but have a different hash. We can't
		// reconcile this. The most likely cause is parallel apply
		// operations.
		s.refreshResult = CacheRefreshConflict

		// Return early so we don't updtae the state
		return nil
	default:
		panic("unhandled cache refresh state")
	}

	if s.refreshResult == CacheRefreshUpdateLocal {
		if err := s.Cache.WriteState(durable); err != nil {
			s.refreshResult = CacheRefreshNoop
			return err
		}
		if err := s.Cache.PersistState(); err != nil {
			s.refreshResult = CacheRefreshNoop
			return err
		}

		cached = durable
	}

	s.state = cached

	return nil
}

// RefreshResult returns the result of the last refresh.
func (s *CacheState) RefreshResult() CacheRefreshResult {
	return s.refreshResult
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
	StateRefresher
}

// CacheStateDurable is the meta-interface that must be implemented for
// the durable storage for CacheState.
type CacheStateDurable interface {
	StateReader
	StateWriter
	StatePersister
	StateRefresher
}

// CacheRefreshResult is used to explain the result of the previous
// RefreshState for a CacheState.
type CacheRefreshResult int

const (
	// CacheRefreshNoop indicates nothing has happened,
	// but that does not indicate an error. Everything is
	// just up to date. (Push/Pull)
	CacheRefreshNoop CacheRefreshResult = iota

	// CacheRefreshInit indicates that there is no local or
	// remote state, and that the state was initialized
	CacheRefreshInit

	// CacheRefreshUpdateLocal indicates the local state
	// was updated. (Pull)
	CacheRefreshUpdateLocal

	// CacheRefreshUpdateRemote indicates the remote state
	// was updated. (Push)
	CacheRefreshUpdateRemote

	// CacheRefreshLocalNewer means the pull was a no-op
	// because the local state is newer than that of the
	// server. This means a Push should take place. (Pull)
	CacheRefreshLocalNewer

	// CacheRefreshRemoteNewer means the push was a no-op
	// because the remote state is newer than that of the
	// local state. This means a Pull should take place.
	// (Push)
	CacheRefreshRemoteNewer

	// CacheRefreshConflict means that the push or pull
	// was a no-op because there is a conflict. This means
	// there are multiple state definitions at the same
	// serial number with different contents. This requires
	// an operator to intervene and resolve the conflict.
	// Shame on the user for doing concurrent apply.
	// (Push/Pull)
	CacheRefreshConflict
)

func (sc CacheRefreshResult) String() string {
	switch sc {
	case CacheRefreshNoop:
		return "Local and remote state in sync"
	case CacheRefreshInit:
		return "Local state initialized"
	case CacheRefreshUpdateLocal:
		return "Local state updated"
	case CacheRefreshUpdateRemote:
		return "Remote state updated"
	case CacheRefreshLocalNewer:
		return "Local state is newer than remote state, push required"
	case CacheRefreshRemoteNewer:
		return "Remote state is newer than local state, pull required"
	case CacheRefreshConflict:
		return "Local and remote state conflict, manual resolution required"
	default:
		return fmt.Sprintf("Unknown state change type: %d", sc)
	}
}

// SuccessfulPull is used to clasify the CacheRefreshResult for
// a refresh operation. This is different by operation, but can be used
// to determine a proper exit code.
func (sc CacheRefreshResult) SuccessfulPull() bool {
	switch sc {
	case CacheRefreshNoop:
		return true
	case CacheRefreshInit:
		return true
	case CacheRefreshUpdateLocal:
		return true
	case CacheRefreshLocalNewer:
		return false
	case CacheRefreshConflict:
		return false
	default:
		return false
	}
}
