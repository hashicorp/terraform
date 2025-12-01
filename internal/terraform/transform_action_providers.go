// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
)

// This is a simpler version of the provider transformer
// This only runs during apply; during plan/eval the providers are attached to the trigger nodes (only)

// I want this to happen after the regular action node provider transformation so we can grab the ProvidedBy() from the action and append that to
// the resource.
// Does that make sense? Is that possible? /shrug

type ActionProviderTransformer struct {
	Config  *configs.Config
	Changes *plans.ChangesSrc
}

func (t *ActionProviderTransformer) Transform(g *Graph) error {
	// map of resource nodes
	// map of resolved providers
	// walk the actions, get the resource and provider, make the connection

	resourceNodes := addrs.MakeMap[addrs.ConfigResource, []GraphNodeConfigResource]()
	resourceInstanceNodes := addrs.MakeMap[addrs.AbsResourceInstance, []GraphNodeResourceInstance]()
	m := providerVertexMap(g)

	for _, node := range g.Vertices() {
		if rin, ok := node.(GraphNodeResourceInstance); ok {
			resourceInstanceNodes.Put(rin.ResourceInstanceAddr(), append(resourceInstanceNodes.Get(rin.ResourceInstanceAddr()), rin))
		}
		rn, ok := node.(GraphNodeConfigResource)
		if !ok {
			continue
		}
		// We ignore any instances that _also_ implement
		// GraphNodeResourceInstance, since in the unlikely event that they
		// do exist we'd probably end up creating cycles by connecting them.
		if _, ok := node.(GraphNodeResourceInstance); ok {
			continue
		}
		rAddr := rn.ResourceAddr()
		resourceNodes.Put(rAddr, append(resourceNodes.Get(rAddr), rn))
	}

	for _, action := range t.Changes.ActionInvocations {
		ap := action.ProviderAddr
		target, ok := m[ap.String()]
		if !ok {
			panic("WHY")
		}

		lat, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger)
		if !ok {
			continue
		}
		resource := lat.TriggeringResourceAddr

		// I'm not sure if we need resource nodes during apply?
		v, ok := resourceNodes.GetOk(resource.ConfigResource())
		if !ok {
			panic("unpossible")
		}
		for _, node := range v {
			g.Connect(dag.BasicEdge(node, target))
		}

		vi, ok := resourceInstanceNodes.GetOk(resource)
		if !ok {
			panic("unpossible")
		}
		for _, node := range vi {
			g.Connect(dag.BasicEdge(node, target))
		}
	}

	return nil
}
