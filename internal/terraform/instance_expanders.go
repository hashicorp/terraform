// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
)

// graphNodeExpandsInstances is implemented by nodes that causes instances to
// be registered in the instances.Expander.
type graphNodeExpandsInstances interface {
	expandsInstances()
}

// forEachModuleInstance is a helper to deal with the common need of doing
// some action for every dynamic module instance associated with a static
// module path.
//
// Many of our plan graph nodes represent configuration constructs that need
// to produce a dynamic subgraph based on the expansion of whatever module
// they are declared inside, and this helper deals with enumerating those
// dynamic addresses so that callers can just focus on building a graph node
// for each one and registering it in the subgraph.
//
// Both of the two callbacks will be called for each instance or set of
// unknown instances. knownCb receives fully-known instance addresses,
// while unknownCb receives partially-expanded addresses. Callers typically
// create a different graph node type in each callback, because
// partially-expanded prefixes conceptually represent an infinite set of
// possible module instance addresses and therefore need quite different
// treatment than a single concrete module instance address.
func forEachModuleInstance(insts *instances.Expander, modAddr addrs.Module, includeOverrides bool, knownCb func(addrs.ModuleInstance), unknownCb func(addrs.PartialExpandedModule)) {
	for _, instAddr := range insts.ExpandModule(modAddr, includeOverrides) {
		knownCb(instAddr)
	}
	for _, instsAddr := range insts.UnknownModuleInstances(modAddr, includeOverrides) {
		unknownCb(instsAddr)
	}
}
