// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/states"
)

type ListStates = collections.Map[addrs.List, []*states.ResourceInstanceObjectSrc]

type QueryRunner struct {
	State addrs.Map[addrs.AbsResourceInstance, []*states.ResourceInstanceObjectSrc]
	View  QueryViews
}

type QueryViews interface {
	List(ListStates)
	Resource(addrs.AbsResourceInstance, *states.ResourceInstanceObjectSrc)
}
