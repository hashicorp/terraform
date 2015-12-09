package azure

import (
	"github.com/Azure/azure-sdk-for-go/arm/network"
)

// makeNetworkSubResourceRef is a helper method which, given the string ID of
// a resource, returns the network.SubResource which references it:
func makeNetworkSubResourceRef(id string) *network.SubResource {
	i := id

	return &network.SubResource{
		ID: &i,
	}
}

// makeNetworkSubResourcesListRef is a helper method which; given the string
// ID's of a collections of resources, returns the *[]network.SubResource
// whose elements reference them.
func makeNetworkSubResourcesListRef(ids []string) *[]network.SubResource {
	res := []network.SubResource{}

	for _, id := range ids {
		res = append(res, *makeNetworkSubResourceRef(id))
	}

	return &res
}
