// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type GraphNodeLockable interface {
	// LockedBy returns the address of the lock for this node.
	LockedBy() addrs.Lock

	// Lock locks the node.
	Lock(ctx EvalContext)

	// Unlock unlocks the node.
	Unlock(ctx EvalContext)
}

func transformLock(g *Graph, c *configs.Config) error {
	// If we have no config then there can be no lock. TODO: maybe not
	if c == nil {
		return nil
	}

	// Transform all the children. We must do this first because
	// we can reference module outputs and they must show up in the
	// reference map.
	for _, cc := range c.Children {
		if err := transformLock(g, cc); err != nil {
			return err
		}
	}

	for _, l := range c.Module.Locks {
		addr := addrs.Lock{Name: l.Name}

		node := &nodeExpandLock{
			Addr:   addr,
			Module: c.Path,
			Config: l,
		}

		log.Printf("[TRACE] LockTransformer: adding lock %s as %T", l.Name, node)
		g.Add(node)
		// for _, n := range g.Vertices() {
		// 	if n, ok := n.(*NodeAbstractResource); ok {
		// 		n.ResourceAddr().
		// 	}
		// }
	}

	return nil
}
