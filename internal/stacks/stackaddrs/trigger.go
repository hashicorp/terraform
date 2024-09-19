// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"github.com/hashicorp/terraform/internal/collections"
)

// Trigger is the address of a "trigger" block within a stack config.
type Trigger struct {
	Name string
}

func (Trigger) referenceableSigil()   {}
func (Trigger) inStackConfigSigil()   {}
func (Trigger) inStackInstanceSigil() {}

func (c Trigger) String() string {
	return "trigger." + c.Name
}

func (c Trigger) UniqueKey() collections.UniqueKey[Trigger] {
	return c
}

// A Trigger is its own [collections.UniqueKey].
func (Trigger) IsUniqueKey(Trigger) {}

// ConfigTrigger places a [Trigger] in the context of a particular [Stack].
type ConfigTrigger = InStackConfig[Trigger]
