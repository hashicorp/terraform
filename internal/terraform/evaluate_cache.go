// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// moduleCallValues caches the values of module calls which can no longer
// change during the current graph walk, so that references to a module call
// don't each need to rebuild its value from the output values of all of its
// instances.
//
// The value of a module call is final once the expansion of the call is known
// and every output value of every instance has been published, because
// neither can be set more than once during a walk.
//
// A nil *moduleCallValues is valid, and caches nothing.
type moduleCallValues struct {
	mu     sync.RWMutex
	values map[string]cty.Value
}

func newModuleCallValues() *moduleCallValues {
	return &moduleCallValues{
		values: make(map[string]cty.Value),
	}
}

func (c *moduleCallValues) get(addr addrs.AbsModuleCall) (cty.Value, bool) {
	if c == nil {
		return cty.NilVal, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.values[addr.String()]
	return val, ok
}

// put records the final value of the given module call.
func (c *moduleCallValues) put(addr addrs.AbsModuleCall, val cty.Value) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[addr.String()] = val
}
