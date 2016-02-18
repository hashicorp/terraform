package search

import "github.com/jen20/riviera/azure"

type Sku struct {
	Name string `json:"name" mapstructure:"name"`
}

type CreateOrUpdateSearchServiceResponse struct {
	ID                 *string             `mapstructure:"id"`
	Name               *string             `mapstructure:"name"`
	Location           *string             `mapstructure:"location"`
	Tags               *map[string]*string `mapstructure:"tags"`
	Sku                *Sku                `mapstructure:"sku"`
	ReplicaCount       *int                `mapstructure:"replicaCount"`
	PartitionCount     *int                `mapstructure:"partitionCount"`
	Status             *string             `mapstructure:"status"`
	StatusDetails      *string             `mapstructure:"statusDetails"`
	ProvisioningStatus *string             `mapstructure:"provisioningStatus"`
}

type CreateOrUpdateSearchService struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	Sku               Sku                `json:"-" riviera:"sku"`
	ReplicaCount      *int               `json:"replicaCount,omitempty"`
	PartitionCount    *int               `json:"partitionCount,omitempty"`
}

func (s CreateOrUpdateSearchService) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: searchServiceDefaultURLPath(s.ResourceGroupName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateOrUpdateSearchServiceResponse{}
		},
	}
}
