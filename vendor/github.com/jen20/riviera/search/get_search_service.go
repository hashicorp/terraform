package search

import "github.com/jen20/riviera/azure"

type GetSearchServiceResponse struct {
	ID                *string            `mapstructure:"id"`
	Name              string             `mapstructure:"name"`
	ResourceGroupName string             `mapstructure:"-"`
	Location          string             `mapstructure:"location"`
	Tags              map[string]*string `mapstructure:"tags"`
	Sku               *Sku               `mapstructure:"sku"`
	ReplicaCount      *int               `mapstructure:"replicaCount"`
	PartitionCount    *int               `mapstructure:"partitionCount"`
	Status            *string            `mapstructure:"status"`
	StatusDetails     *string            `mapstructure:"statusDetails"`
	ProvisioningState *string            `mapstructure:"provisioningState"`
}

type GetSearchService struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s GetSearchService) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: searchServiceDefaultURLPath(s.ResourceGroupName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetSearchServiceResponse{}
		},
	}
}
