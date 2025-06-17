// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// ConfigState describes a configuration block, and is used to make that config block stateful.
type ConfigState interface {
	Empty() bool
	Config(*configschema.Block) (cty.Value, error)
	SetConfig(cty.Value, *configschema.Block) error
}

// DeepCopier implementations can return deep copies of themselves for use elsewhere
// without mutating the original value.
type DeepCopier[T any] interface {
	DeepCopy() *T
}

// Planner implementations can return a representation of their data that's
// appropriate for storing in a plan file.
type Planner[T any] interface {
	Plan(*configschema.Block, string) (*T, error)
}
