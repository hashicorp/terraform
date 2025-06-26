package grpc_statemgr

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

var (
	_ statemgr.Full = &grpcStateManager{}
)

// NewGrpcStateManager takes in a provider that implements states storage
// and returns a state manager implementation that allows calling code to
// use the provider's state management-related methods.
//
// Requires being passed:
// 1) the name of the state storage implementation
// 2) the name of the state/the active workspace
func NewGrpcStateManager(provider providers.Interface, typeName string, stateId string) statemgr.Full {
	return &grpcStateManager{
		provider: provider,
		typeName: typeName,
		stateId:  stateId,
	}
}

type grpcStateManager struct {
	provider providers.Interface
	typeName string // the state storage implementation's name
	stateId  string

	mu sync.Mutex
	// TODO
	// We need fields here similar to the remote state manager; we need to
	// have in memory state that can be updated repeatedly and eventually persisted
	// via a call to PersistState.
}

func (g *grpcStateManager) Lock(info *statemgr.LockInfo) (string, error) {
	fmt.Printf("in Lock method of state manager from pluggable state store %q in provider %q", g.typeName, g.provider)
	// TODO (SarahFrench/radeksimko): - implement Lock RPC in protocol, provider interface, and use here

	// req := providers.LockStateRequest{
	// 	TypeName:  g.typeName,
	// 	StateId:   g.stateId,
	// 	Operation: info.Operation,
	// }
	// resp := g.provider.LockState(req)
	// return resp.LockId, resp.Diagnostics.Err()

	return "", nil
}

func (g *grpcStateManager) Unlock(id string) error {
	fmt.Printf("in Unlock method of state manager from pluggable state store %q in provider %q, trying to unlock %s", g.typeName, g.provider, id)

	// TODO (SarahFrench/radeksimko): - implement Unlock RPC in protocol, provider interface, and use here

	// req := providers.UnlockStateRequest{
	// 	TypeName: g.typeName,
	// 	StateId:  g.stateId,
	// 	LockId:   id,
	// }
	// resp := g.provider.UnlockState(req)
	// return resp.Diagnostics.Err()

	return nil
}

func (g *grpcStateManager) State() *states.State {
	fmt.Printf("in State method of state manager from pluggable state store %q in provider %q", g.typeName, g.provider)

	g.mu.Lock()
	defer g.mu.Unlock()

	// TODO (SarahFrench/radeksimko): Return a deep copy of the state value from internal, in-memory store here
	return nil
}

func (g *grpcStateManager) WriteState(state *states.State) error {
	fmt.Printf("in WriteState method of state manager from pluggable state store %q in provider %q", g.typeName, g.provider)

	g.mu.Lock()
	defer g.mu.Unlock()

	// TODO (SarahFrench/radeksimko): write to internal, in-memory store here
	return nil
}

func (g *grpcStateManager) RefreshState() error {
	fmt.Printf("in RefreshState method of state manager from pluggable state store %q in provider %q", g.typeName, g.provider)

	// TODO (SarahFrench/radeksimko): - implement ReadState RPC in protocol, provider interface, and use here
	return nil
}

func (g *grpcStateManager) PersistState(foobar *schemarepo.Schemas) error {
	fmt.Printf("in PersistState method of state manager from pluggable state store %q in provider %q", g.typeName, g.provider)

	// TODO (SarahFrench/radeksimko): - implement WriteState RPC in protocol, provider interface, and use here
	return nil
}

func (g *grpcStateManager) GetRootOutputValues(ctx context.Context) (map[string]*states.OutputValue, error) {
	fmt.Printf("in GetRootOutputValues method of state manager from pluggable state store %q in provider %q", g.typeName, g.provider)

	if err := g.RefreshState(); err != nil {
		return nil, fmt.Errorf("Failed to load state: %s", err)
	}

	state := g.State()
	if state == nil {
		state = states.NewState()
	}

	return state.RootOutputValues, nil
}
