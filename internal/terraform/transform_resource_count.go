// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/providers"
)

// ResourceCountTransformer is a GraphTransformer that expands the count
// out for a specific resource.
//
// This assumes that the count is already interpolated.
type ResourceCountTransformer struct {
	Concrete ConcreteResourceInstanceNodeFunc
	Schema   *providers.Schema

	Addr          addrs.ConfigResource
	InstanceAddrs []addrs.AbsResourceInstance
}

func (t *ResourceCountTransformer) Transform(g *Graph) error {
	for idx, addr := range t.InstanceAddrs {
		abstract := NewNodeAbstractResourceInstance(addr)
		abstract.Schema = t.Schema
		abstract.expansionEnum = idx
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		log.Printf("[TRACE] ResourceCountTransformer: adding %s as %T", addr, node)
		g.Add(node)
	}
	return nil
}
