// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import "github.com/hashicorp/terraform/internal/collections"

type InputVariable struct {
	Name string
}

func (InputVariable) referenceableSigil()   {}
func (InputVariable) inStackConfigSigil()   {}
func (InputVariable) inStackInstanceSigil() {}

func (v InputVariable) String() string {
	return "var." + v.Name
}

func (v InputVariable) UniqueKey() collections.UniqueKey[InputVariable] {
	return v
}

// An InputVariable is its own [collections.UniqueKey].
func (InputVariable) IsUniqueKey(InputVariable) {}

// ConfigInputVariable places an [InputVariable] in the context of a particular [Stack].
type ConfigInputVariable = InStackConfig[InputVariable]

// AbsInputVariable places an [InputVariable] in the context of a particular [StackInstance].
type AbsInputVariable = InStackInstance[InputVariable]
