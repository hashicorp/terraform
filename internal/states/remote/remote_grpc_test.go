// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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

// Testing grpcClient's Put method, via the state manager made using a grpcClient.
// The PersistState method on a state manager calls the Put method of the underlying client.
func Test_grpcClient_Put(t *testing.T) {
	typeName := "foo_bar" // state store 'bar' in provider 'foo'
	stateId := "production"

	// State with 1 output
	s := states.NewState()
	s.SetOutputValue(addrs.AbsOutputValue{
		Module:      addrs.RootModuleInstance,
		OutputValue: addrs.OutputValue{Name: "foo"},
	}, cty.StringVal("bar"), false)

	t.Run("state manager made using grpcClient writes the expected state", func(t *testing.T) {
		provider := testing_provider.MockProvider{
			// Mock a provider and internal state store that
			// have both been configured
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,

			// Check values received by the provider from the Put method.
			WriteStateBytesFn: func(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
				if req.TypeName != typeName || req.StateId != stateId {
					t.Fatalf("expected provider WriteStateBytes method to receive TypeName %q and StateId %q, instead got TypeName %q and StateId %q",
						typeName,
						stateId,
						req.TypeName,
						req.StateId)
				}

				r := bytes.NewReader(req.Bytes)
				reqState, err := statefile.Read(r)
				if err != nil {
					t.Fatal(err)
				}
				if reqState.State.String() != s.String() {
					t.Fatalf("wanted state %s got %s", s.String(), reqState.State.String())
				}
				return providers.WriteStateBytesResponse{
					// no diags
				}
			},
		}

		// This package will be consumed in a statemgr.Full, so we test using NewRemoteGRPC
		// and invoke the method on that interface that uses Put.
		c := NewRemoteGRPC(&provider, typeName, stateId)

		// Set internal state value that will be persisted.
		c.WriteState(s)

		// Test PersistState, which uses Put.
		err := c.PersistState(nil)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	})

	t.Run("state manager made using grpcClient returns expected error from error diagnostic", func(t *testing.T) {
		expectedErr := "error forced from test"
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  expectedErr,
			Detail:   expectedErr,
		})
		provider := testing_provider.MockProvider{
			// Mock a provider and internal state store that
			// have both been configured
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,

			// Force an error diagnostic
			WriteStateBytesFn: func(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
				return providers.WriteStateBytesResponse{
					Diagnostics: diags,
				}
			},
		}

		// This package will be consumed in a statemgr.Full, so we test using NewRemoteGRPC
		// and invoke the method on that interface that uses Get.
		c := NewRemoteGRPC(&provider, typeName, stateId)

		// Set internal state value that will be persisted.
		c.WriteState(s)

		// Test PersistState, which uses Put.
		err := c.PersistState(nil)
		if err == nil {
			t.Fatalf("expected error but got none")
		}
		if !strings.Contains(err.Error(), expectedErr) {
			t.Fatalf("expected error to contain %q, but got: %s", expectedErr, err.Error())
		}
	})

	t.Run("grpcClient refuses zero-byte writes", func(t *testing.T) {
		provider := testing_provider.MockProvider{
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,
			WriteStateBytesFn: func(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
				t.Fatal("expected WriteStateBytes not to be called for zero-byte payload")
				return providers.WriteStateBytesResponse{}
			},
		}

		client := &grpcClient{
			provider: &provider,
			typeName: typeName,
			stateId:  stateId,
		}

		diags := client.Put(nil)
		if !diags.HasErrors() {
			t.Fatalf("expected diagnostics when attempting to write zero bytes")
		}
		if provider.WriteStateBytesCalled {
			t.Fatalf("provider WriteStateBytes should not be called")
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

// Testing grpcClient's Lock method.
// The Lock method on a state manager calls the Lock method of the underlying client.
func Test_grpcClient_Lock(t *testing.T) {
	typeName := "foo_bar" // state store 'bar' in provider 'foo'
	stateId := "production"
	operation := "apply"
	lockInfo := &statemgr.LockInfo{
		Operation: operation,
		// This is sufficient when locking via PSS
	}

	t.Run("state manager made using grpcClient sends expected values to Lock method", func(t *testing.T) {
		expectedLockId := "id-from-mock"
		provider := testing_provider.MockProvider{
			// Mock a provider and internal state store that
			// have both been configured
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,

			// Check values received by the provider from the Lock method.
			LockStateFn: func(req providers.LockStateRequest) providers.LockStateResponse {
				if req.TypeName != typeName || req.StateId != stateId || req.Operation != operation {
					t.Fatalf("expected provider ReadStateBytes method to receive TypeName %q, StateId %q, and Operation %q, instead got TypeName %q, StateId %q, and Operation %q",
						typeName,
						stateId,
						operation,
						req.TypeName,
						req.StateId,
						req.Operation,
					)
				}
				return providers.LockStateResponse{
					LockId: expectedLockId,
				}
			},
		}

		// This package will be consumed in a statemgr.Full, so we test using NewRemoteGRPC
		// and invoke the method on that interface that uses Lock.
		c := NewRemoteGRPC(&provider, typeName, stateId)

		lockId, err := c.Lock(lockInfo)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if !provider.LockStateCalled {
			t.Fatal("expected remote grpc state manager's Lock method to call Lock method on underlying provider, but it has not been called")
		}
		if lockId != expectedLockId {
			t.Fatalf("unexpected lock id returned, wanted %q, got %q", expectedLockId, lockId)
		}
	})

	t.Run("state manager made using grpcClient returns expected error from Lock method's error diagnostic", func(t *testing.T) {
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

			// Force return of an error.
			LockStateResponse: providers.LockStateResponse{
				Diagnostics: diags,
			},
		}

		// This package will be consumed in a statemgr.Full, so we test using NewRemoteGRPC
		// and invoke the method on that interface that uses Lock.
		c := NewRemoteGRPC(&provider, typeName, stateId)

		_, err := c.Lock(lockInfo)
		if !provider.LockStateCalled {
			t.Fatal("expected remote grpc state manager's Lock method to call Lock method on underlying provider, but it has not been called")
		}
		if err == nil {
			t.Fatal("expected error but got none")
		}
		expectedMsg := "error forced from test"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Fatalf("expected error to include %q but got: %s", expectedMsg, err)
		}
	})

	t.Run("state manager made using grpcClient currently swallows warning diagnostics returned from the Lock method", func(t *testing.T) {
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "warning forced from test",
			Detail:   "warning forced from test",
		})

		provider := testing_provider.MockProvider{
			// Mock a provider and internal state store that
			// have both been configured
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,

			// Force return of a warning.
			LockStateResponse: providers.LockStateResponse{
				Diagnostics: diags,
			},
		}

		c := NewRemoteGRPC(&provider, typeName, stateId)

		_, err := c.Lock(lockInfo)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// The warning is swallowed by the Lock method.
		// The Locker interface should be updated to allow use of diagnostics instead of errors,
		// and this test should be updated.
	})
}

// Testing grpcClient's Unlock method.
// The Unlock method on a state manager calls the Unlock method of the underlying client.
func Test_grpcClient_Unlock(t *testing.T) {
	typeName := "foo_bar" // state store 'bar' in provider 'foo'
	stateId := "production"

	t.Run("state manager made using grpcClient sends expected values to Unlock method", func(t *testing.T) {
		expectedLockId := "id-from-mock"
		provider := testing_provider.MockProvider{
			// Mock a provider and internal state store that
			// have both been configured
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,

			// Check values received by the provider from the Lock method.
			UnlockStateFn: func(req providers.UnlockStateRequest) providers.UnlockStateResponse {
				if req.TypeName != typeName || req.StateId != stateId || req.LockId != expectedLockId {
					t.Fatalf("expected provider ReadStateBytes method to receive TypeName %q, StateId %q, and LockId %q, instead got TypeName %q, StateId %q, and LockId %q",
						typeName,
						stateId,
						expectedLockId,
						req.TypeName,
						req.StateId,
						req.LockId,
					)
				}
				return providers.UnlockStateResponse{}
			},
		}

		// This package will be consumed in a statemgr.Full, so we test using NewRemoteGRPC
		// and invoke the method on that interface that uses Unlock.
		c := NewRemoteGRPC(&provider, typeName, stateId)

		err := c.Unlock(expectedLockId)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if !provider.UnlockStateCalled {
			t.Fatal("expected remote grpc state manager's Unlock method to call Unlock method on underlying provider, but it has not been called")
		}

	})

	t.Run("state manager made using grpcClient returns expected error from Unlock method's error diagnostic", func(t *testing.T) {
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

			// Force return of an error.
			UnlockStateResponse: providers.UnlockStateResponse{
				Diagnostics: diags,
			},
		}

		// This package will be consumed in a statemgr.Full, so we test using NewRemoteGRPC
		// and invoke the method on that interface that uses Unlock.
		c := NewRemoteGRPC(&provider, typeName, stateId)

		err := c.Unlock("foobar") // argument used here isn't important in this test
		if !provider.UnlockStateCalled {
			t.Fatal("expected remote grpc state manager's Unlock method to call Unlock method on underlying provider, but it has not been called")
		}
		if err == nil {
			t.Fatal("expected error but got none")
		}
		expectedMsg := "error forced from test"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Fatalf("expected error to include %q but got: %s", expectedMsg, err)
		}
	})

	t.Run("state manager made using grpcClient currently swallows warning diagnostics returned from the Unlock method", func(t *testing.T) {
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "warning forced from test",
			Detail:   "warning forced from test",
		})

		provider := testing_provider.MockProvider{
			// Mock a provider and internal state store that
			// have both been configured
			ConfigureProviderCalled:   true,
			ConfigureStateStoreCalled: true,

			// Force return of a warning.
			UnlockStateResponse: providers.UnlockStateResponse{
				Diagnostics: diags,
			},
		}

		c := NewRemoteGRPC(&provider, typeName, stateId)

		err := c.Unlock("foobar") // argument used here isn't important in this test
		if !provider.UnlockStateCalled {
			t.Fatal("expected remote grpc state manager's Unlock method to call Unlock method on underlying provider, but it has not been called")
		}
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// The warning is swallowed by the Unlock method.
		// The Locker interface should be updated to allow use of diagnostics instead of errors,
		// and this test should be updated.
	})
}
