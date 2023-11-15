// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import "github.com/hashicorp/terraform/internal/collections"

type LocalValue struct {
	Name string
}

func (LocalValue) referenceableSigil()   {}
func (LocalValue) inStackConfigSigil()   {}
func (LocalValue) inStackInstanceSigil() {}

func (v LocalValue) String() string {
	return "local." + v.Name
}

func (v LocalValue) UniqueKey() collections.UniqueKey[LocalValue] {
	return v
}

// A LocalValue is its own [collections.UniqueKey].
func (LocalValue) IsUniqueKey(LocalValue) {}

// ConfigLocalValue places a [LocalValue] in the context of a particular [Stack].
type ConfigLocalValue = InStackConfig[LocalValue]

// AbsLocalValue places a [LocalValue] in the context of a particular [StackInstance].
type AbsLocalValue = InStackInstance[LocalValue]
