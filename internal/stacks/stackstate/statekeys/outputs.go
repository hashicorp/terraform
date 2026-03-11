// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

var (
	_ Key = Output{}
)

type Output struct {
	OutputAddr stackaddrs.OutputValue
}

func parseOutput(s string) (Key, error) {
	addrRaw, ok := finalKeyField(s)
	if !ok {
		return nil, fmt.Errorf("unsupported extra field in component instance key")
	}
	addr, diags := stackaddrs.ParseAbsOutputValueStr(addrRaw)
	if diags.HasErrors() {
		return nil, fmt.Errorf("output key has invalid output address %q", addrRaw)
	}
	if !addr.Stack.IsRoot() {
		return nil, fmt.Errorf("output key was for non-root stack %q", addrRaw)
	}

	return Output{
		OutputAddr: addr.Item,
	}, nil
}

func (o Output) KeyType() KeyType {
	return OutputType
}

func (o Output) rawSuffix() string {
	return o.OutputAddr.String()
}
