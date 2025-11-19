package terraform

import "github.com/hashicorp/terraform/internal/addrs"

// This is a simpler version of the provider transformer, which only runs during apply
// I want this to happen after the regular action node provider transformation so we can grab the ProvidedBy() from the action and append that to
// the resource
type GraphNodeActionsProviderConsumer interface {
	GraphNodeModulePath

	// Provider() returns the Provider FQN for the node.
	ActionsProviders() (providers []addrs.Provider)

	// Add a resolved action provider address for this resource.
	AddActionProvider(addrs.AbsProviderConfig)
}
