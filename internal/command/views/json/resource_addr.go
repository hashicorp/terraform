// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
)

type ResourceAddr struct {
	Addr                 string                  `json:"addr"`
	Module               string                  `json:"module"`
	Resource             string                  `json:"resource"`
	ImpliedProvider      string                  `json:"implied_provider"`
	ResourceType         string                  `json:"resource_type"`
	ResourceName         string                  `json:"resource_name"`
	ResourceKey          ctyjson.SimpleJSONValue `json:"resource_key"`
	IsResourceKeyUnknown bool                    `json:"is_resource_key_unknown,omitempty"`
}

func newResourceAddr(addr addrs.AbsResourceInstance) ResourceAddr {
	resourceKey := ctyjson.SimpleJSONValue{Value: cty.NilVal}
	isResourceKeyUnknown := false
	if addr.Resource.Key != nil {
		if keyValue := addr.Resource.Key.Value(); keyValue.IsWhollyKnown() {
			resourceKey.Value = keyValue
		} else {
			// Deferred resources use WildcardKey to indicate their resource key is unknown
			isResourceKeyUnknown = true
		}
	}
	return ResourceAddr{
		Addr:                 addr.String(),
		Module:               addr.Module.String(),
		Resource:             addr.Resource.String(),
		ImpliedProvider:      addr.Resource.Resource.ImpliedProvider(),
		ResourceType:         addr.Resource.Resource.Type,
		ResourceName:         addr.Resource.Resource.Name,
		ResourceKey:          resourceKey,
		IsResourceKeyUnknown: isResourceKeyUnknown,
	}
}
