package terraform

import "github.com/hashicorp/terraform/addrs"

// instanceExpander is implemented by nodes that causes instances to be
// registered in the instances.Expander.
// This is used to determine during apply whether a node is required to be in
// the graph, by checking if it has any requiresInstanceExpansion dependents.
// This prevents unnecessary nodes from being evaluated, and if the module is
// being removed, we may not be able to evaluate the expansion at all.
type instanceExpander interface {
	expandsInstances() addrs.Module
}

// requiresInstanceExpansion is implemented by nodes that require their address
// be previously registered in the instances.Expander in order to evaluate.
type requiresInstanceExpansion interface {
	requiresExpansion()
}
