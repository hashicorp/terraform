package storage

import "github.com/jen20/riviera/azure"

type CustomDomain struct {
	Name             *string `json:"name" mapstructure:"name"`
	UseSubDomainName *bool   `json:"useSubDomainName,omitempty" mapstructure:"useSubdomainName"`
}

type UpdateStorageAccountCustomDomainResponse struct {
	CustomDomain CustomDomain `mapstructure:"customDomain"`
}

type UpdateStorageAccountCustomDomain struct {
	Name              string       `json:"-"`
	ResourceGroupName string       `json:"-"`
	CustomDomain      CustomDomain `json:"customDomain"`
}

func (command UpdateStorageAccountCustomDomain) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PATCH",
		URLPathFunc: storageDefaultURLPathFunc(command.ResourceGroupName, command.Name),
		ResponseTypeFunc: func() interface{} {
			return &UpdateStorageAccountCustomDomainResponse{}
		},
	}
}
