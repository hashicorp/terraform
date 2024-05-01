// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// nodeEphemeralResourceClose is the node type for closing the previously-opened
// instances of a particular ephemeral resource.
//
// Although ephemeral resource instances will always all get closed once a
// graph walk has completed anyway, the inclusion of explicit nodes for this
// allows closing ephemeral resource instances more promptly after all work
// that uses them has been completed, rather than always just waiting until
// the end of the graph walk.
//
// This is scoped to config-level resources rather than dynamic resource
// instances as a concession to allow using the same node type in both the plan
// and apply graphs, where the former only deals in whole resources while the
// latter contains individual instances.
type nodeEphemeralResourceClose struct {
	addr addrs.ConfigResource
}

func (n *nodeEphemeralResourceClose) Name() string {
	return n.addr.String() + " (close)"
}
