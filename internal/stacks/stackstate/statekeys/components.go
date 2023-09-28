// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

type ComponentInstance struct {
	ComponentInstanceAddr stackaddrs.AbsComponentInstance
}

func parseComponentInstance(s string) (Key, error) {
	addrRaw, ok := finalKeyField(s)
	if !ok {
		return nil, fmt.Errorf("unsupported extra field in component instance key")
	}
	addr, diags := stackaddrs.ParseAbsComponentInstanceStr(addrRaw)
	if diags.HasErrors() {
		return nil, fmt.Errorf("component instance key has invalid component instance address %q", addrRaw)
	}
	return ComponentInstance{
		ComponentInstanceAddr: addr,
	}, nil
}

func (k ComponentInstance) KeyType() KeyType {
	return ComponentInstanceType
}

func (k ComponentInstance) rawSuffix() string {
	return k.ComponentInstanceAddr.String()
}
