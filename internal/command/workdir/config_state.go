// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"
)

// ConfigState describes a configuration block, and is used to make that config block stateful.
type ConfigState[T any] interface {
	Empty() bool
	Config(*configschema.Block) (cty.Value, error)
	SetConfig(cty.Value, *configschema.Block) error
	ForPlan(*configschema.Block, string) (*plans.Backend, error)
	DeepCopy() *T
}
