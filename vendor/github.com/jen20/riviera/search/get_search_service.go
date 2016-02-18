package search

import "github.com/jen20/riviera/azure"

type GetSearchServiceResponse struct {
	ID                *string            `mapstructure:"id"`
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	Sku               *Sku               `json:"sku,omitempty"`
	ReplicaCount      *string            `json:"replicaCount,omitempty"`
	PartitionCount    *string            `json:"partitionCount,omitempty"`
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
