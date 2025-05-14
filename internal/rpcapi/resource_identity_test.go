// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResourceIdentity(t *testing.T) {
	for name, tc := range map[string]struct {
		state *stackstate.State

		expected    []*stacks.ListResourceIdentities_Resource
		expectedErr error
	}{
		"nil state": {
			state:    nil,
			expected: []*stacks.ListResourceIdentities_Resource{},
		},
		"empty state": {
			state:    stackstate.NewStateBuilder().Build(),
			expected: []*stacks.ListResourceIdentities_Resource{},
		},
		"resource with no identity": {
			state: stackstate.NewStateBuilder().
				AddResourceInstance(stackstate.NewResourceInstanceBuilder().
					SetAddr(mustAbsResourceInstanceObject(t, "component.self.testing_resource.hello")).
					SetProviderAddr(mustDefaultRootProvider("testing")).
					SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]any{
							"id":    "moved",
							"value": "moved",
						}),
					})).
				Build(),
			expected: []*stacks.ListResourceIdentities_Resource{},
		},

		"resource with identity": {
			state: stackstate.NewStateBuilder().
				AddResourceInstance(stackstate.NewResourceInstanceBuilder().
					SetAddr(mustAbsResourceInstanceObject(t, "component.self.testing_resource.hello")).
					SetProviderAddr(mustDefaultRootProvider("testing")).
					SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]any{
							"id":    "moved",
							"value": "moved",
						}),
						IdentityJSON: mustMarshalJSONAttrs(map[string]any{
							"id": "hello",
						}),
					})).
				Build(),
			expected: []*stacks.ListResourceIdentities_Resource{
				{
					ComponentAddr:         "component.self",
					ComponentInstanceAddr: "component.self",
					ResourceInstanceAddr:  "testing_resource.hello",
					ResourceIdentity: &stacks.DynamicValue{
						Msgpack: []byte("\x81\xa2id\xa5hello"),
					},
				},
			},
		},

		"resource with identity and newer identity version": {
			state: stackstate.NewStateBuilder().
				AddResourceInstance(stackstate.NewResourceInstanceBuilder().
					SetAddr(mustAbsResourceInstanceObject(t, "component.self.testing_resource.hello")).
					SetProviderAddr(mustDefaultRootProvider("testing")).
					SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]any{
							"id":    "moved",
							"value": "moved",
						}),
						IdentityJSON: mustMarshalJSONAttrs(map[string]any{
							"id": "hello",
						}),
						IdentitySchemaVersion: 2,
					})).
				Build(),
			expectedErr: status.Errorf(codes.InvalidArgument, "resource testing_resource.hello has an invalid identity schema version, please update the provider or refresh the state"),
		},
	} {
		t.Run(name, func(t *testing.T) {

			identitySchemas := map[addrs.Provider]map[string]providers.IdentitySchema{
				addrs.NewDefaultProvider("testing"): {
					"testing_resource": {
						Version: 0,
						Body: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"id": {
									Type:     cty.String,
									Required: true,
								},
							},
							Nesting: configschema.NestingSingle,
						},
					},
				},
			}

			actual, err := listResourceIdentities(tc.state, identitySchemas)

			if tc.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tc.expectedErr)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}

				if len(actual) != len(tc.expected) {
					t.Errorf("expected %d resources, got %d", len(tc.expected), len(actual))
				}

				for index, expected := range tc.expected {
					actual := actual[index]

					if actual.ComponentAddr != expected.ComponentAddr {
						t.Errorf("expected component address %s, got %s", expected.ComponentAddr, actual.ComponentAddr)
					}

					if actual.ComponentInstanceAddr != expected.ComponentInstanceAddr {
						t.Errorf("expected component instance address %s, got %s", expected.ComponentInstanceAddr, actual.ComponentInstanceAddr)
					}

					if actual.ResourceInstanceAddr != expected.ResourceInstanceAddr {
						t.Errorf("expected resource instance address %s, got %s", expected.ResourceInstanceAddr, actual.ResourceInstanceAddr)
					}

					if string(actual.ResourceIdentity.Msgpack) != string(expected.ResourceIdentity.Msgpack) {
						t.Errorf("expected resource identity %v, got %v", expected.ResourceIdentity, actual.ResourceIdentity)
					}
				}
			}
		})
	}
}
