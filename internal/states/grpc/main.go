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
	req := providers.LockStateRequest{
		TypeName:  g.typeName,
		StateId:   g.stateId,
		Operation: info.Operation,
	}
	resp := g.provider.LockState(req)
	return resp.LockId, resp.Diagnostics.Err()
}

func (g *grpcStateManager) Unlock(id string) error {
	req := providers.UnlockStateRequest{
		TypeName: g.typeName,
		StateId:  g.stateId,
		LockId:   id,
	}
	resp := g.provider.UnlockState(req)
	return resp.Diagnostics.Err()
}

func (g *grpcStateManager) State() *states.State {
	g.mu.Lock()
	defer g.mu.Unlock()

	// TODO: Return a deep copy of the state value from internal, in-memory store here
	return nil
}

func (g *grpcStateManager) WriteState(state *states.State) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// TODO: write to internal, in-memory store here
	return nil
}

func (g *grpcStateManager) RefreshState() error {
	// No ReadState method implemented on the provider yet
	return nil
}

func (g *grpcStateManager) PersistState(foobar *schemarepo.Schemas) error {
	// No WriteState method implemented on the provider yet
	return nil
}

func (g *grpcStateManager) GetRootOutputValues(ctx context.Context) (map[string]*states.OutputValue, error) {
	if err := g.RefreshState(); err != nil {
		return nil, fmt.Errorf("Failed to load state: %s", err)
	}

	state := g.State()
	if state == nil {
		state = states.NewState()
	}

	return state.RootOutputValues, nil
}
