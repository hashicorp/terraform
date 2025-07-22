// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"testing"

	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
)

// Testing grpcClient's Delete method.
// This method is needed to implement the remote.Client interface, but
// this is not invoked by the remote state manager (remote.State) that
// wil contain the client.
//
// In future we should remove the need for a Delete method in
// remote.Client, but for now it is implemented and tested.
func Test_grpcClient_Delete(t *testing.T) {
	typeName := "foo_bar" // state store 'bar' in provider 'foo'
	stateId := "production"

	provider := testing_provider.MockProvider{
		// Mock a provider and internal state store that
		// have both been configured
		ConfigureProviderCalled:   true,
		ConfigureStateStoreCalled: true,

		// Check values received by the provider from the Delete method.
		DeleteStateFn: func(req providers.DeleteStateRequest) providers.DeleteStateResponse {
			if req.TypeName != typeName || req.StateId != stateId {
				t.Fatalf("expected provider DeleteState method to receive TypeName %q and StateId %q, instead got TypeName %q and StateId %q",
					typeName,
					stateId,
					req.TypeName,
					req.StateId)
			}
			return providers.DeleteStateResponse{
				// no diags
			}
		},
	}

	// Delete isn't accessible via a statemgr.Full, so we don't use NewRemoteGRPC.
	// See comment above test for more information.
	c := grpcClient{
		provider: &provider,
		typeName: typeName,
		stateId:  stateId,
	}

	err := c.Delete()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !provider.DeleteStateCalled {
		t.Fatal("expected Delete method to call DeleteState method on underlying provider, but it has not been called")
	}
}
