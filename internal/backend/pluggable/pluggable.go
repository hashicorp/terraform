// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pluggable

import (
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/pluggable/chunks"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// NewPluggable returns a Pluggable. A Pluggable fulfils the
// backend.Backend interface and allows management of state via
// a state store implemented in the provider that's within the Pluggable.
//
// These are the assumptions about that
// provider:
// * The provider implements at least one state store.
// * The provider has already been fully configured before using NewPluggable.
//
// The state store could also be configured prior to using NewPluggable,
// but preferably it will be configured via the Pluggable,
// using the relevant backend.Backend methods.
//
// By wrapping a configured provider in a Pluggable we allow calling code
// to use the provider's gRPC methods when interacting with state.
func NewPluggable(p providers.Interface, typeName string) (*Pluggable, error) {
	if p == nil {
		return nil, errors.New("Attempted to initialize pluggable state with a nil provider interface. This is a bug in Terraform and should be reported")
	}
	if typeName == "" {
		return nil, errors.New("Attempted to initialize pluggable state with an empty string identifier for the state store. This is a bug in Terraform and should be reported")
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
	if len(schemaResp.StateStores) == 0 {
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

// ProviderSchema returns the schema for the provider implementing the state store.
//
// This isn't part of the backend.Backend interface but is needed in calling code.
// When it's used the backend.Backend will need to be cast to a Pluggable.
func (p *Pluggable) ProviderSchema() *configschema.Block {
	schemaResp := p.provider.GetProviderSchema()
	return schemaResp.Provider.Body
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
	var diags tfdiags.Diagnostics
	req := providers.ConfigureStateStoreRequest{
		TypeName: p.typeName,
		Config:   config,
		Capabilities: providers.StateStoreClientCapabilities{
			// The core binary will always request the default chunk size from the provider to start
			ChunkSize: chunks.DefaultStateStoreChunkSize,
		},
	}
	resp := p.provider.ConfigureStateStore(req)
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return diags
	}

	// Validate the returned value from chunk size negotiation
	chunkSize := resp.Capabilities.ChunkSize
	if chunkSize == 0 || chunkSize > chunks.MaxStateStoreChunkSize {
		diags = diags.Append(fmt.Errorf("Failed to negotiate acceptable chunk size. "+
			"Expected size > 0 and <= %d bytes, provider wants %d bytes",
			chunks.MaxStateStoreChunkSize, chunkSize,
		))
		return diags
	}

	// Negotiated chunk size is valid, so set it in the provider server
	// that will use the value for future RPCs to read/write state.
	cs := p.provider.(providers.StateStoreChunkSizeSetter)
	cs.SetStateStoreChunkSize(p.typeName, int(chunkSize))
	log.Printf("[TRACE] Pluggable.Configure: negotiated a chunk size of %v when configuring state store %s",
		chunkSize,
		p.typeName,
	)

	return resp.Diagnostics
}

// Workspaces returns a list of all states/CE workspaces that the backend.Backend
// can find, given how it is configured. For example returning a list of differently
// -named files in a blob storage service.
//
// Workspace implements backend.Backend
func (p *Pluggable) Workspaces() ([]string, tfdiags.Diagnostics) {
	req := providers.GetStatesRequest{
		TypeName: p.typeName,
	}
	resp := p.provider.GetStates(req)

	return resp.States, resp.Diagnostics
}

// DeleteWorkspace deletes the state file for the named workspace.
// The state storage provider is expected to return error diagnostics
// if the workspace doesn't exist or it is unable to be deleted.
//
// DeleteWorkspace implements backend.Backend
func (p *Pluggable) DeleteWorkspace(workspace string, force bool) tfdiags.Diagnostics {
	req := providers.DeleteStateRequest{
		TypeName: p.typeName,
		StateId:  workspace,
	}
	resp := p.provider.DeleteState(req)
	return resp.Diagnostics
}

// StateMgr returns a state manager that uses gRPC to communicate with the
// state storage provider to interact with state.
//
// StateMgr implements backend.Backend
func (p *Pluggable) StateMgr(workspace string) (statemgr.Full, tfdiags.Diagnostics) {
	// repackages the provider's methods inside a state manager,
	// to be passed to the calling code that expects a statemgr.Full
	return remote.NewRemoteGRPC(p.provider, p.typeName, workspace), nil
}
