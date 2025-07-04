package pluggable

import (
	"errors"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NewPluggable returns an instance of the backend.Backend interface that
// contains a provider interface. These are the assumptions about that
// provider:
//
// * The provider implements at least one state store.
// * The provider has already been configured before using NewPluggable.
//
// The state store could also be configured prior to using NewPluggable,
// or it could be configured using the relevant backend.Backend methods.
//
// By wrapping a configured provider in a Pluggable we allow calling code
// to use the provider's gRPC methods when interacting with state.
func NewPluggable(p providers.Interface, typeName string) (backend.Backend, error) {

	if p == nil {
		return nil, errors.New("Attempted to initialize pluggable state with a nil provider interface. This is a bug in Terraform and should be reported")
	}
	if typeName == "" {
		return nil, errors.New("Attempted to initialize pluggable state with an empty string identifier for the state store name. This is a bug in Terraform and should be reported")
	}

	return &Pluggable{
		provider: p,
		typeName: typeName,
	}, nil
}

var _ backend.Backend = &Pluggable{}

type Pluggable struct {
	provider providers.Interface
	typeName string
}

// ConfigSchema returns the schema for the state store implementation
// name provided when the Pluggable was constructed.
//
// ConfigSchema implements backend.Backend
func (p *Pluggable) ConfigSchema() *configschema.Block {
	schemaResp := p.provider.GetProviderSchema()
	if schemaResp.StateStores == nil {
		// No state stores
		return nil
	}
	val, ok := schemaResp.StateStores[p.typeName]
	if !ok {
		// Cannot find state store with that type
		return nil
	}

	// State store type exists
	return val.Body
}

// PrepareConfig validates configuration for the state store in
// the state storage provider. The configuration sent from Terraform core
// will not include any values from environment variables; it is the
// provider's responsibility to access any environment variables
// to get the complete set of configuration prior to validating it.
//
// PrepareConfig implements backend.Backend
func (p *Pluggable) PrepareConfig(config cty.Value) (cty.Value, tfdiags.Diagnostics) {
	req := providers.ValidateStateStoreConfigRequest{
		TypeName: p.typeName,
		Config:   config,
	}
	resp := p.provider.ValidateStateStoreConfig(req)
	return config, resp.Diagnostics
}

// Configure configures the state store in the state storage provider.
// Calling code is expected to have already validated the config using
// the PrepareConfig method.
//
// It is the provider's responsibility to access any environment variables
// set by the user to get the complete set of configuration.
//
// Configure implements backend.Backend
func (p *Pluggable) Configure(config cty.Value) tfdiags.Diagnostics {
	req := providers.ConfigureStateStoreRequest{
		TypeName: p.typeName,
		Config:   config,
	}
	resp := p.provider.ConfigureStateStore(req)
	return resp.Diagnostics
}

// Workspaces returns a list of all states/CE workspaces that the backend.Backend
// can find, given how it is configured. For example returning a list of differently
// -named files in a blob storage service.
//
// Workspace implements backend.Backend
func (p *Pluggable) Workspaces() ([]string, error) {
	req := providers.GetStatesRequest{
		TypeName: p.typeName,
	}
	resp := p.provider.GetStates(req)

	return resp.States, resp.Diagnostics.Err()
}

// DeleteWorkspace deletes the state file for the named workspace.
// The state storage provider is expected to return error diagnostics
// if the workspace doesn't exist or it is unable to be deleted.
//
// DeleteWorkspace implements backend.Backend
func (p *Pluggable) DeleteWorkspace(workspace string, force bool) error {
	req := providers.DeleteStateRequest{
		TypeName: p.typeName,
		StateId:  workspace,
	}
	resp := p.provider.DeleteState(req)
	return resp.Diagnostics.Err()
}

// StateMgr returns a state manager that uses gRPC to communicate with the
// state storage provider to interact with state.
//
// StateMgr implements backend.Backend
func (p *Pluggable) StateMgr(workspace string) (statemgr.Full, error) {
	// repackages the provider's methods inside a state manager,
	// to be passed to the calling code that expects a statemgr.Full
	return remote.NewRemoteGRPC(p.provider, p.typeName, workspace), nil
}
