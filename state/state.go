package state

import (
	"github.com/hashicorp/terraform/terraform"
)

// State is the collection of all state interfaces.
type State interface {
	StateReader
	StateWriter
	StateRefresher
	StatePersister
}

// StateReader is the interface for things that can return a state. Retrieving
// the state here must not error. Loading the state fresh (an operation that
// can likely error) should be implemented by RefreshState. If a state hasn't
// been loaded yet, it is okay for State to return nil.
type StateReader interface {
	State() *terraform.State
}

// StateWriter is the interface that must be implemented by something that
// can write a state. Writing the state can be cached or in-memory, as
// full persistence should be implemented by StatePersister.
type StateWriter interface {
	WriteState(*terraform.State) error
}

// StateRefresher is the interface that is implemented by something that
// can load a state. This might be refreshing it from a remote location or
// it might simply be reloading it from disk.
type StateRefresher interface {
	RefreshState() error
}

// StatePersister is implemented to truly persist a state. Whereas StateWriter
// is allowed to perhaps be caching in memory, PersistState must write the
// state to some durable storage.
type StatePersister interface {
	PersistState() error
}
