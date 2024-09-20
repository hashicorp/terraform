// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

var (
	_ Key = Variable{}
)

type Variable struct {
	VariableAddr stackaddrs.InputVariable
}

func parseVariable(s string) (Key, error) {
	addrRaw, ok := finalKeyField(s)
	if !ok {
		return nil, fmt.Errorf("unsupported extra field in component instance key")
	}
	addr, diags := stackaddrs.ParseAbsInputVariableStr(addrRaw)
	if diags.HasErrors() {
		return nil, fmt.Errorf("variable key has invalid output address %q", addrRaw)
	}
	if !addr.Stack.IsRoot() {
		return nil, fmt.Errorf("variable key was for non-root stack %q", addrRaw)
	}

	return Variable{
		VariableAddr: addr.Item,
	}, nil
}

func (v Variable) KeyType() KeyType {
	return VariableType
}

func (v Variable) rawSuffix() string {
	return v.VariableAddr.String()
}
