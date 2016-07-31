package storage

import "github.com/jen20/riviera/azure"

type UpdateStorageAccountTypeResponse struct {
	AccountType *string `mapstructure:"accountType"`
}

type UpdateStorageAccountType struct {
	Name              string  `json:"-"`
	ResourceGroupName string  `json:"-"`
	AccountType       *string `json:"accountType,omitempty"`
}

func (command UpdateStorageAccountType) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PATCH",
		URLPathFunc: storageDefaultURLPathFunc(command.ResourceGroupName, command.Name),
		ResponseTypeFunc: func() interface{} {
			return &UpdateStorageAccountTypeResponse{}
		},
	}
}
