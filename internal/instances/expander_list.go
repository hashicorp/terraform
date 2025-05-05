// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package instances

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// SetQueryListSingle records that the given query list inside the given module
// does not use any repetition arguments and is therefore a singleton.
func (e *Expander) SetQueryListSingle(moduleAddr addrs.ModuleInstance, listAddr addrs.List) {
	e.setQueryListExpansion(moduleAddr, listAddr, expansionSingleVal)
}

// SetQueryListCount records that the given query list inside the given module
// uses the "count" repetition argument, with the given value.
func (e *Expander) SetQueryListCount(moduleAddr addrs.ModuleInstance, listAddr addrs.List, count int) {
	e.setQueryListExpansion(moduleAddr, listAddr, expansionCount(count))
}

// SetQueryListCountUnknown records that the given query list inside the given
// module uses the "count" repetition argument but its value isn't yet known.
func (e *Expander) SetQueryListCountUnknown(moduleAddr addrs.ModuleInstance, listAddr addrs.List) {
	e.setQueryListExpansion(moduleAddr, listAddr, expansionDeferredIntKey)
}

// SetQueryListForEach records that the given query list inside the given module
// uses the "for_each" repetition argument, with the given map value.
//
// In the configuration language the for_each argument can also accept a set.
// It's the caller's responsibility to convert that into an identity map before
// calling this method.
func (e *Expander) SetQueryListForEach(moduleAddr addrs.ModuleInstance, listAddr addrs.List, mapping map[string]cty.Value) {
	e.setQueryListExpansion(moduleAddr, listAddr, expansionForEach(mapping))
}

// SetQueryListForEachUnknown records that the given query list inside the given
// module uses the "for_each" repetition argument, but the map keys aren't
// known yet.
func (e *Expander) SetQueryListForEachUnknown(moduleAddr addrs.ModuleInstance, listAddr addrs.List) {
	e.setQueryListExpansion(moduleAddr, listAddr, expansionDeferredStringKey)
}

func (e *Expander) setQueryListExpansion(parentAddr addrs.ModuleInstance, listAddr addrs.List, exp expansion) {
	e.mu.Lock()
	defer e.mu.Unlock()

	mod, known := e.findModule(parentAddr)
	if !known {
		panic(fmt.Sprintf("can't register expansion in %s where path includes unknown expansion", parentAddr))
	}
	if _, exists := mod.queryLists[listAddr]; exists {
		panic(fmt.Sprintf("expansion already registered for %s", listAddr))
	}
	mod.queryLists[listAddr] = exp
}

// GetListInstanceRepetitionData returns an object describing the values
// that should be available for each.key, each.value, and count.index within
// the definition block for the given list instance.
func (e *Expander) GetListInstanceRepetitionData(addr addrs.ListInstance) RepetitionData {
	e.mu.RLock()
	defer e.mu.RUnlock()

	parentMod, known := e.findModule(addrs.RootModuleInstance)
	if !known {
		// If we're nested inside something unexpanded then we don't even
		// know what type of expansion we're doing.
		return TotallyUnknownRepetitionData
	}
	exp, ok := parentMod.queryLists[addr.List]
	if !ok {
		panic(fmt.Sprintf("no expansion has been registered for %s", addr.List))
	}
	return exp.repetitionData(addr.Key)
}

func (e *Expander) ListInstanceKeys(addr addrs.List) (keyType addrs.InstanceKeyType, knownKeys []addrs.InstanceKey, unknownKeys bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	parentMod, known := e.findModule(addrs.RootModuleInstance)
	if !known {
		// If we're nested inside something unexpanded then we don't even
		// know yet what kind of instance key to expect. (The caller might
		// be able to infer this itself using configuration info, though.)
		return addrs.UnknownKeyType, nil, true
	}
	exp, ok := parentMod.queryLists[addr]
	if !ok {
		panic(fmt.Sprintf("no expansion has been registered for %s", addr))
	}
	return exp.instanceKeys()
}

func (e *Expander) ToInstanceKeys(addr addrs.List) (keyType addrs.InstanceKeyType, knownKeys []addrs.InstanceKey, unknownKeys bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	parentMod, known := e.findModule(addrs.RootModuleInstance)
	if !known {
		// If we're nested inside something unexpanded then we don't even
		// know yet what kind of instance key to expect. (The caller might
		// be able to infer this itself using configuration info, though.)
		return addrs.UnknownKeyType, nil, true
	}
	exp, ok := parentMod.queryLists[addr]
	if !ok {
		panic(fmt.Sprintf("no expansion has been registered for %s", addr))
	}
	return exp.instanceKeys()
}
