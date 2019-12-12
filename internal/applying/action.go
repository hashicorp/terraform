package applying

import (
	"context"
	"fmt"
	"sync"

	multierror "github.com/hashicorp/go-multierror"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/instances"
	"github.com/hashicorp/terraform/internal/schemas"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// action is an interface representing actions that can be taken during an
// apply operation. Actions are the nodes in our apply graph, while the edges
// are the dependency relationships between them, with the dependency arrows
// pointing from the depending action to the action it depends on.
type action interface {
	// Name returns a concise name for the action to be used primarily in
	// debug traces.
	Name() string

	// Execute is responsible for doing whatever the action represents, and
	// returning error diagnostics if it's unable to complete the operation
	// successfully.
	//
	// If Execute returns errors, the graph walk is halted early.
	//
	// It may also optionally return warnings if something possibly-concerning
	// happened but the action completed anyway.
	Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics
}

// actionData is used to transfer certain shared state between actions.
//
// The methods of this type should be minimal, low-level manipulation code,
// with most interesting user-level behavior implemented in the actions
// themselves where we have better context to return good error messages, etc.
//
// actionData methods are safe to call concurrently and concurrent calls are
// the common case due to graph traversal concurrency.
type actionData struct {
	state             *states.SyncState
	schemas           *schemas.Schemas
	providerInstances map[string]providers.Interface
	dependencies      Dependencies
	expander          *instances.Expander

	mu sync.RWMutex
}

func newActionData(deps Dependencies, schemas *schemas.Schemas, state *states.State) *actionData {
	return &actionData{
		state:             state.SyncWrapper(),
		schemas:           schemas,
		providerInstances: map[string]providers.Interface{},
		dependencies:      deps,
		expander:          instances.NewExpander(),
	}
}

func (d *actionData) State() *states.SyncState {
	return d.state
}

func (d *actionData) StartProviderInstance(addr addrs.Provider) (providers.Interface, error) {
	inst, err := d.dependencies.ResourceProvider(addr.LegacyString(), "<unused>")
	if err != nil {
		return nil, err
	}

	return inst, nil
}

func (d *actionData) SetConfiguredProviderInstance(configAddr addrs.AbsProviderConfig, inst providers.Interface) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// absolute provider configurations are not hashable because they contain
	// a slice, so we'll use string serializations for our map keys.
	key := configAddr.String()

	if _, exists := d.providerInstances[key]; exists {
		// Should never happen; if we get here then that suggests the graph
		// has duplicate actions or missing dependency edges.
		panic(fmt.Sprintf("provider configuration %s already has a registered provider instance", key))
	}

	d.providerInstances[key] = inst
}

func (d *actionData) CloseProviderInstance(configAddr addrs.AbsProviderConfig) error {
	key := configAddr.String()
	inst, exists := d.providerInstances[key]
	if !exists {
		// Should never happen; if we get here then that suggests the graph
		// has duplicate actions or missing dependency edges.
		panic(fmt.Sprintf("provider configuration %s has no registered provider instance", key))
	}

	err := inst.Close()
	if err != nil {
		return err
	}
	delete(d.providerInstances, key)
	return nil
}

// close is intended to be called by the code orchestrating the walk once the
// walk is done, and is not part of the API intended to be consumed by
// action implementations.
func (d *actionData) close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// If we have any provider instances still running then we'll try to
	// clean them all up.
	var err error
	for _, inst := range d.providerInstances {
		instErr := inst.Close()
		if instErr != nil {
			err = multierror.Append(instErr)
		}
	}

	return err
}
