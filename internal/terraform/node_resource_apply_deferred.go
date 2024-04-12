// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "github.com/hashicorp/terraform/internal/addrs"

// nodeApplyableDeferredInstance is a node that represents a deferred instance
// in the apply graph. This node is targetable and helps maintain the correct
// ordering of the apply graph.
//
// Note, that it does not implement Execute, as deferred instances are not
// executed during the apply phase.
type nodeApplyableDeferredInstance struct {
	*NodeAbstractResourceInstance
}

// nodeApplyableDeferredPartialInstance is a node that represents a deferred
// partial instance in the apply graph. This simply adds a method  to get the
// partial address on top of the regular behaviour of
// nodeApplyableDeferredInstance.
type nodeApplyableDeferredPartialInstance struct {
	*nodeApplyableDeferredInstance

	PartialAddr addrs.PartialExpandedResource
}
