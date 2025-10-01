// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Testing grpcClient's Get method, via the state manager made using a grpcClient.
// The RefreshState method on a state manager calls the Get method of the underlying client.
func Test_grpcClient_Get(t *testing.T) {
	typeName := "foo_bar" // state store 'bar' in provider 'foo'
	stateId := "production"
	stateString := `{
    "version": 4,
    "terraform_version": "0.13.0",
    "serial": 0,
    "lineage": "",
    "outputs": {
        "foo": {
            "value": "bar",
            "type": "string"
        }
    }
}`

	t.Run("state manager made using grpcClient returns expected state", func(t *testing.T) {
		provider := testing_provider.MockProvider{
			// Mock a provider and internal state store that
			// have both been configured
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,

			// Check values received by the provider from the Get method.
			ReadStateBytesFn: func(req providers.ReadStateBytesRequest) providers.ReadStateBytesResponse {
				if req.TypeName != typeName || req.StateId != stateId {
					t.Fatalf("expected provider ReadStateBytes method to receive TypeName %q and StateId %q, instead got TypeName %q and StateId %q",
						typeName,
						stateId,
						req.TypeName,
						req.StateId)
				}
				return providers.ReadStateBytesResponse{
					Bytes: []byte(stateString),
					// no diags
				}
			},
		}

		// This package will be consumed in a statemgr.Full, so we test using NewRemoteGRPC
		// and invoke the method on that interface that uses Get.
		c := NewRemoteGRPC(&provider, typeName, stateId)

		err := c.RefreshState() // Calls Get
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if !provider.ReadStateBytesCalled {
			t.Fatal("expected remote grpc state manager's RefreshState method to, via Get, call ReadStateBytes method on underlying provider, but it has not been called")
		}
		s := c.State()
		v, ok := s.RootOutputValues["foo"]
		if !ok {
			t.Fatal("state manager doesn't contain the state returned by the mock")
		}
		if v.Value.AsString() != "bar" {
			t.Fatal("state manager doesn't contain the correct output value in the state")
		}
	})

	t.Run("state manager made using grpcClient returns expected error from error diagnostic", func(t *testing.T) {
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "error forced from test",
			Detail:   "error forced from test",
		})
		provider := testing_provider.MockProvider{
			// Mock a provider and internal state store that
			// have both been configured
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,

			// Force an error diagnostic
			ReadStateBytesFn: func(req providers.ReadStateBytesRequest) providers.ReadStateBytesResponse {
				return providers.ReadStateBytesResponse{
					// we don't expect state to accompany an error, but this test shows that
					// if an error us present amy state returned is ignored.
					Bytes:       []byte(stateString),
					Diagnostics: diags,
				}
			},
		}

		// This package will be consumed in a statemgr.Full, so we test using NewRemoteGRPC
		// and invoke the method on that interface that uses Get.
		c := NewRemoteGRPC(&provider, typeName, stateId)

		err := c.RefreshState() // Calls Get
		if err == nil {
			t.Fatal("expected an error but got none")
		}

		if !provider.ReadStateBytesCalled {
			t.Fatal("expected remote grpc state manager's RefreshState method to, via Get, call ReadStateBytes method on underlying provider, but it has not been called")
		}
		s := c.State()
		if s != nil {
			t.Fatalf("expected refresh to fail due to error diagnostic, but state has been refreshed: %s", s.String())
		}
	})
}

// Testing grpcClient's Delete method.
// This method is needed to implement the remote.Client interface, but
// this is not invoked by the remote state manager (remote.State) that
// will contain the client.
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

	diags := c.Delete()
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}

	if !provider.DeleteStateCalled {
		t.Fatal("expected Delete method to call DeleteState method on underlying provider, but it has not been called")
	}
}
