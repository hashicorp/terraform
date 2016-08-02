package storage

import "github.com/jen20/riviera/azure"

type DeleteStorageAccount struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (command DeleteStorageAccount) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "DELETE",
		URLPathFunc: storageDefaultURLPathFunc(command.ResourceGroupName, command.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
