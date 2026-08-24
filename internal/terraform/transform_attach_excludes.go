// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
)

type GraphNodeAttachExcludes interface {
	GraphNodeConfigResource
	AttachExcludes(excludes []addrs.Targetable)
}

type AttachExcludesTransformer struct {
	// Excludes are the resources to exclude from the plan graph.
	Excludes []addrs.Targetable
}

func (t AttachExcludesTransformer) Transform(g *Graph) error {
	if len(t.Excludes) == 0 {
		return nil
	}

	for v := range g.VerticesSeq() {
		resourceNode, ok := v.(GraphNodeAttachExcludes)
		if !ok {
			continue
		}

		// Only managed resources can be excluded directly
		if resourceNode.ResourceAddr().Resource.Mode != addrs.ManagedResourceMode {
			continue
		}

		log.Printf("[TRACE] AttachExcludesTransformer: attaching exclude addresses to %s", v.Name())
		resourceNode.AttachExcludes(t.Excludes)
	}

	return nil
}
