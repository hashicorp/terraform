// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NewRemoteGRPC returns a remote state manager (remote.State) containing
// an implementation of remote.Client that allows Terraform to interact with
// a provider implementing pluggable state storage.
//
// The remote.Client implementation's methods invoke the provider's RPC
// methods to perform tasks like reading in state, locking, etc.
//
// NewRemoteGRPC requires these arguments to create the remote.Client:
// 1) the provider interface, needed to call gRPC methods
// 2) the name of the state storage implementation in the provider
// 3) the name of the state/the active workspace
func NewRemoteGRPC(provider providers.Interface, typeName string, stateId string) statemgr.Full {
	mgr := &State{
		Client: &grpcClient{
			provider: provider,
			typeName: typeName,
			stateId:  stateId,
		},
	}
	return mgr
}

var (
	_ Client       = &grpcClient{}
	_ ClientLocker = &grpcClient{}
)

// grpcClient acts like a client to enable the State state manager
// to communicate with a provider that implements pluggable state
// storage via gRPC.
//
// The calling code needs to provide information about the store's name
// and the name of the state (i.e. CE workspace) to use, as these are
// arguments required in gRPC requests.
type grpcClient struct {
	provider providers.Interface
	typeName string // the state storage implementation's name
	stateId  string
}

// Get invokes the ReadStateBytes gRPC method in the plugin protocol
// and returns a copy of the downloaded state data.
//
// Implementation of remote.Client
func (g *grpcClient) Get() (*Payload, error) {
	// TODO - replace with method implementation added to main branch
	req := providers.ReadStateBytesRequest{
		TypeName: g.typeName,
		StateId:  g.stateId,
	}
	resp := g.provider.ReadStateBytes(req)

	if len(resp.Bytes) == 0 {
		// No state to return
		return nil, resp.Diagnostics.Err()
	}

	payload := &Payload{
		Data: resp.Bytes,
		MD5:  []byte("foobar"),
	}
	return payload, resp.Diagnostics.Err()
}

// Put invokes the WriteStateBytes gRPC method in the plugin protocol
// and to transfer state data to the remote location.
//
// Implementation of remote.Client
func (g *grpcClient) Put(state []byte) error {
	// TODO - replace with method implementation added to main branch
	req := providers.WriteStateBytesRequest{
		TypeName: g.typeName,
		StateId:  g.stateId,
	}
	resp := g.provider.WriteStateBytes(req)

	return resp.Diagnostics.Err()
}

// Delete invokes the DeleteState gRPC method in the plugin protocol
// to delete a named state in the remote location.
//
// NOTE: this is included to fulfil an interface, but deletion of
// workspaces is actually achieved through the backend.Backend
// interface's DeleteWorkspace method.
//
// Implementation of remote.Client
func (g *grpcClient) Delete() tfdiags.Diagnostics {
	req := providers.DeleteStateRequest{
		TypeName: g.typeName,
		StateId:  g.stateId,
	}
	resp := g.provider.DeleteState(req)
	return resp.Diagnostics
}

// Lock invokes the LockState gRPC method in the plugin protocol
// to lock a named state in the remote location.
//
// Implementation of remote.Client
func (g *grpcClient) Lock(*statemgr.LockInfo) (string, error) {
	panic("not implemented yet")
}

// Unlock invokes the UnlockState gRPC method in the plugin protocol
// to release a named lock on a specific state in the remote location.
//
// Implementation of remote.Client
func (g *grpcClient) Unlock(id string) error {
	panic("not implemented yet")
}
