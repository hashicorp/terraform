package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
)

// NodeDisabledProvider represents a provider that is disabled. A disabled
// provider does nothing. It exists to properly set inheritance information
// for child providers.
type NodeDisabledProvider struct {
	*NodeAbstractProvider
}

var (
	_ GraphNodeSubPath        = (*NodeDisabledProvider)(nil)
	_ RemovableIfNotTargeted  = (*NodeDisabledProvider)(nil)
	_ GraphNodeReferencer     = (*NodeDisabledProvider)(nil)
	_ GraphNodeProvider       = (*NodeDisabledProvider)(nil)
	_ GraphNodeAttachProvider = (*NodeDisabledProvider)(nil)
	_ dag.GraphNodeDotter     = (*NodeDisabledProvider)(nil)
)

func (n *NodeDisabledProvider) Name() string {
	return fmt.Sprintf("%s (disabled)", n.NodeAbstractProvider.Name())
}
