// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
)

func listResourceIdentities(stackState *stackstate.State, identitySchemas map[addrs.Provider]map[string]providers.IdentitySchema) ([]*stacks.ListResourceIdentities_Resource, error) {
	resourceIdentities := make([]*stacks.ListResourceIdentities_Resource, 0)

	// A non-existent stack state has no resource identities
	if stackState == nil {
		return resourceIdentities, nil
	}

	for ci := range stackState.AllComponentInstances() {
		componentIdentities := stackState.IdentitiesForComponent(ci)
		for ri, src := range componentIdentities {
			// We skip resources without identity JSON
			if len(src.IdentityJSON) == 0 {
				continue
			}

			providerAddrs := addrs.ImpliedProviderForUnqualifiedType(ri.ResourceInstance.Resource.Resource.ImpliedProvider())

			identitySchema, ok := identitySchemas[providerAddrs]
			if !ok {
				return nil, status.Errorf(codes.InvalidArgument, "provider %s could not be found in the identity schema", providerAddrs)
			}

			resourceType := ri.ResourceInstance.Resource.Resource.Type
			schema, ok := identitySchema[resourceType]
			if !ok {
				return nil, status.Errorf(codes.InvalidArgument, "resource %s could not be found in the identity schema", ri)
			}
			if src.IdentitySchemaVersion != uint64(schema.Version) {
				return nil, status.Errorf(codes.InvalidArgument, "resource %s has an invalid identity schema version, please update the provider or refresh the state", ri)
			}
			ty := schema.Body.ImpliedType()

			identity, err := ctyjson.Unmarshal(src.IdentityJSON, ty)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to unmarshal identity JSON for resource %s: %s", ri, err)
			}

			identityRaw, err := plans.NewDynamicValue(identity, ty)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to create dynamic value for identity for resource %s: %s", ri, err)
			}
			stacksIdentityRaw := stacks.NewDynamicValue(identityRaw, []cty.Path{})

			resourceIdentities = append(resourceIdentities, &stacks.ListResourceIdentities_Resource{
				ComponentAddr:         ci.Item.Component.String(),
				ComponentInstanceAddr: ci.Item.String(),
				ResourceInstanceAddr:  ri.String(),
				ResourceIdentity:      stacksIdentityRaw,
			})
		}
	}

	return resourceIdentities, nil
}
