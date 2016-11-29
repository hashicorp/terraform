package storage

import "github.com/jen20/riviera/azure"

type UpdateStorageAccountTagsResponse struct {
	AccountType *string `mapstructure:"accountType"`
}

type UpdateStorageAccountTags struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
}

func (command UpdateStorageAccountTags) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PATCH",
		URLPathFunc: storageDefaultURLPathFunc(command.ResourceGroupName, command.Name),
		ResponseTypeFunc: func() interface{} {
			return &UpdateStorageAccountTypeResponse{}
		},
	}
}
