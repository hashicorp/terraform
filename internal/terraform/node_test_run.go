// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/moduletest"
)

type NodeTestRun struct {
	file *moduletest.File
	run  *moduletest.Run
}

func (n *NodeTestRun) Run() *moduletest.Run {
	return n.run
}

func (n *NodeTestRun) File() *moduletest.File {
	return n.file
}

// GraphNodeReferencer
func (n *NodeTestRun) References() []*addrs.Reference {
	var result []*addrs.Reference
	// If we have a config then we prefer to use that.
	if c := n.run.Config; c != nil {
		refs, _ := n.run.GetReferences()
		result = append(result, refs...)
		// for _, expr := range c.Variables {
		// 	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, expr)
		// }
	}
	// if c := n.Config; c != nil {
	// 	result = append(result, n.DependsOn()...)

	// 	if n.Schema == nil {
	// 		// Should never happen, but we'll log if it does so that we can
	// 		// see this easily when debugging.
	// 		log.Printf("[WARN] no schema is attached to %s, so config references cannot be detected", n.Name())
	// 	}

	// 	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
	// 	result = append(result, refs...)
	// 	refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
	// 	result = append(result, refs...)

	// 	for _, expr := range c.TriggersReplacement {
	// 		refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, expr)
	// 		result = append(result, refs...)
	// 	}

	// 	// ReferencesInBlock() requires a schema
	// 	if n.Schema != nil {
	// 		refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, c.Config, n.Schema)
	// 		result = append(result, refs...)
	// 	}

	// 	if c.Managed != nil {
	// 		if c.Managed.Connection != nil {
	// 			refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, c.Managed.Connection.Config, connectionBlockSupersetSchema)
	// 			result = append(result, refs...)
	// 		}

	// 		for _, p := range c.Managed.Provisioners {
	// 			if p.When != configs.ProvisionerWhenCreate {
	// 				continue
	// 			}
	// 			if p.Connection != nil {
	// 				refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, p.Connection.Config, connectionBlockSupersetSchema)
	// 				result = append(result, refs...)
	// 			}

	// 			schema := n.ProvisionerSchemas[p.Type]
	// 			if schema == nil {
	// 				log.Printf("[WARN] no schema for provisioner %q is attached to %s, so provisioner block references cannot be detected", p.Type, n.Name())
	// 			}
	// 			refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, p.Config, schema)
	// 			result = append(result, refs...)
	// 		}
	// 	}

	// 	for _, check := range c.Preconditions {
	// 		refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, check.Condition)
	// 		result = append(result, refs...)
	// 		refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, check.ErrorMessage)
	// 		result = append(result, refs...)
	// 	}
	// 	for _, check := range c.Postconditions {
	// 		refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, check.Condition)
	// 		result = append(result, refs...)
	// 		refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, check.ErrorMessage)
	// 		result = append(result, refs...)
	// 	}
	// }

	return result
}
