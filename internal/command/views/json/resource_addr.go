// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
)

type ResourceAddr struct {
	Addr               string                  `json:"addr"`
	Module             string                  `json:"module"`
	Resource           string                  `json:"resource"`
	ImpliedProvider    string                  `json:"implied_provider"`
	ResourceType       string                  `json:"resource_type"`
	ResourceName       string                  `json:"resource_name"`
	ResourceKey        ctyjson.SimpleJSONValue `json:"resource_key"`
	ResourceKeyUnknown bool                    `json:"resource_key_unknown,omitempty"`
}

func newResourceAddr(addr addrs.AbsResourceInstance) ResourceAddr {
	resourceKey := ctyjson.SimpleJSONValue{Value: cty.NilVal}
	resourceKeyUnknown := false
	if addr.Resource.Key != nil {
		if keyValue := addr.Resource.Key.Value(); keyValue.IsWhollyKnown() {
			resourceKey.Value = keyValue
		} else {
			// Deferred resources use [addrs.WildcardKey] to indicate the resource key is unknown
			resourceKeyUnknown = true
		}
	}
	return ResourceAddr{
		Addr:               addr.String(),
		Module:             addr.Module.String(),
		Resource:           addr.Resource.String(),
		ImpliedProvider:    addr.Resource.Resource.ImpliedProvider(),
		ResourceType:       addr.Resource.Resource.Type,
		ResourceName:       addr.Resource.Resource.Name,
		ResourceKey:        resourceKey,
		ResourceKeyUnknown: resourceKeyUnknown,
	}
}
