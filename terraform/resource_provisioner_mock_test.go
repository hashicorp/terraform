package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/provisioners"
)

func TestMockResourceProvisioner_impl(t *testing.T) {
	var _ ResourceProvisioner = new(MockResourceProvisioner)
}

// simpleMockProvisioner returns a MockProvisioner that is pre-configured
// with schema for its own config, with the same content as returned by
// function simpleTestSchema.
//
// For most reasonable uses the returned provisioner must be registered in a
// componentFactory under the name "test". Use simpleMockComponentFactory
// to obtain a pre-configured componentFactory containing the result of
// this function along with simpleMockProvider, both registered as "test".
//
// The returned provisioner has no other behaviors by default, but the caller
// may modify it in order to stub any other required functionality, or modify
// the default schema stored in the field GetSchemaReturn. Each new call to
// simpleTestProvisioner produces entirely new instances of all of the nested
// objects so that callers can mutate without affecting mock objects.
func simpleMockProvisioner() *MockProvisioner {
	return &MockProvisioner{
		GetSchemaResponse: provisioners.GetSchemaResponse{
			Provisioner: simpleTestSchema(),
		},
	}
}
