// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
)

// StackCall represents a call to an embedded stack. This is essentially the
// address of a "stack" block in the configuration, before it's been fully
// expanded into zero or more instances.
type StackCall struct {
	Name string
}

func (StackCall) referenceableSigil()   {}
func (StackCall) inStackConfigSigil()   {}
func (StackCall) inStackInstanceSigil() {}

func (c StackCall) String() string {
	return "stack." + c.Name
}

func (c StackCall) UniqueKey() collections.UniqueKey[StackCall] {
	return stackCallUniqueKey(c.String())
}

type stackCallUniqueKey string

// IsUniqueKey implements collections.UniqueKey.
func (stackCallUniqueKey) IsUniqueKey(StackCall) {}

// ConfigStackCall represents a static stack call inside a particular [Stack].
type ConfigStackCall = InStackConfig[StackCall]

// AbsStackCall represents an instance of a stack call inside a particular
// [StackInstance[.
type AbsStackCall = InStackInstance[StackCall]

func AbsStackCallInstance(call AbsStackCall, key addrs.InstanceKey) StackInstance {
	ret := make(StackInstance, len(call.Stack), len(call.Stack)+1)
	copy(ret, call.Stack)
	return append(ret, StackInstanceStep{
		Name: call.Item.Name,
		Key:  key,
	})
}
