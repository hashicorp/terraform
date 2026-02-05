// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Provenance is a data structure for managing instance keys and nodes within the graph.
type Provenance struct {
	Nodes *addrs.Map[addrs.Referenceable, *NodePlannableResourceInstance]
	cache map[addrs.Referenceable]*NodePlannableResourceInstance
	walkOperation
	Targets []addrs.Targetable

	Keys        addrs.Map[addrs.Referenceable, addrs.InstanceKey]
	EvalContext EvalContext

	Values map[string][]marks.SourceMark
}

func NewProvenance() *Provenance {
	return &Provenance{
		Values: make(map[string][]marks.SourceMark),
	}
}

func (d *Provenance) ExecuteResource(moduleAddr addrs.ModuleInstance, addr addrs.Resource) tfdiags.Diagnostics {
	scope := evalContextModuleInstance{Addr: moduleAddr}
	var diags tfdiags.Diagnostics

	evalInstance := func(instanceAddr addrs.ResourceInstance) tfdiags.Diagnostics {
		if _, ok := d.cache[instanceAddr]; ok {
			return nil
		}
		resourceInstance, ok := d.Nodes.GetOk(instanceAddr)
		if !ok {
			return tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Resource not found",
					fmt.Sprintf("The resource %s was not found.", instanceAddr),
				),
			}
		}

		diags := resourceInstance.Execute(d.EvalContext.withScope(scope), d.walkOperation)

		if diags.HasErrors() {
			return diags
		}

		d.cache[instanceAddr] = resourceInstance
		return nil
	}

	for referencedAddr, instanceKey := range d.Keys.Iter() {
		if referencedAddr != addr {
			continue
		}
		// Wildcard key, evaluate all instances
		if instanceKey == addrs.WildcardKey {
			for _, node := range d.Nodes.Elems {
				plannableAddr := node.Key.(addrs.ResourceInstance)
				if addr.Equal(plannableAddr.Resource) {
					diags = diags.Append(evalInstance(plannableAddr))
				}
			}
		} else {
			instanceAddr := addr.Instance(instanceKey)
			diags = diags.Append(evalInstance(instanceAddr))
		}
	}

	return nil
}

func (d *Provenance) Store(addr fmt.Stringer, value cty.Value) tfdiags.Diagnostics {
	key := addr.String()
	if _, ok := d.Values[key]; !ok {
		d.Values[key] = []marks.SourceMark{}
	}
	sourceMarks := marks.GetMarks[marks.SourceMark](value)
	d.Values[key] = append(d.Values[key], sourceMarks...)

	return nil
}
