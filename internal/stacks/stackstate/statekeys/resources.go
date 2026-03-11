// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/states"
)

// ResourceInstanceObject represents state keys for resource instance objects.
type ResourceInstanceObject struct {
	ResourceInstance stackaddrs.AbsResourceInstance
	DeposedKey       states.DeposedKey
}

func parseResourceInstanceObject(s string) (Key, error) {
	componentInstAddrRaw, s := cutKeyField(s)
	resourceInstAddrRaw, s := cutKeyField(s)
	deposedRaw, ok := finalKeyField(s)
	if !ok {
		return nil, fmt.Errorf("unsupported extra field in resource instance object key")
	}
	componentInstAddr, diags := stackaddrs.ParseAbsComponentInstanceStr(componentInstAddrRaw)
	if diags.HasErrors() {
		return nil, fmt.Errorf("resource instance object key has invalid component instance address %q", componentInstAddrRaw)
	}
	resourceInstAddr, diags := addrs.ParseAbsResourceInstanceStr(resourceInstAddrRaw)
	if diags.HasErrors() {
		return nil, fmt.Errorf("resource instance object key has invalid resource instance address %q", resourceInstAddrRaw)
	}
	var deposedKey states.DeposedKey
	if deposedRaw != "cur" {
		var err error
		deposedKey, err = states.ParseDeposedKey(deposedRaw)
		if err != nil {
			return nil, fmt.Errorf("resource instance object key has invalid deposed key %q", deposedRaw)
		}
	} else {
		deposedKey = states.NotDeposed
	}
	return ResourceInstanceObject{
		ResourceInstance: stackaddrs.AbsResourceInstance{
			Component: componentInstAddr,
			Item:      resourceInstAddr,
		},
		DeposedKey: deposedKey,
	}, nil
}

func (k ResourceInstanceObject) KeyType() KeyType {
	return ResourceInstanceObjectType
}

func (k ResourceInstanceObject) rawSuffix() string {
	var b rawKeyBuilder
	b.AppendField(k.ResourceInstance.Component.String())
	b.AppendField(k.ResourceInstance.Item.String())
	if k.DeposedKey != states.NotDeposed {
		// A valid deposed key is always eight hex digits, and never
		// contains a comma so we can write it unquoted.
		b.AppendField(string(k.DeposedKey))
	} else {
		b.AppendField("cur") // short for "current"
	}
	return b.Raw()
}
