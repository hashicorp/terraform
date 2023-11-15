// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// Targetable is the stacks analog to [addrs.Targetable], representing something
// that can be "targeted" inside a stack configuration.
type Targetable interface {
	targetableSigil()
}

// ComponentTargetable is an adapter type that makes everything that's
// targetable in the main Terraform language also targetable through a
// component instance when in a stack configuration.
//
// To represent targeting an entire component, place [addrs.RootModuleInstance]
// in field Item to describe targeting the component's root module.
type ComponentTargetable[T addrs.Targetable] struct {
	Component AbsComponentInstance
	Item      T
}

func (ComponentTargetable[T]) targetableSigil() {}
