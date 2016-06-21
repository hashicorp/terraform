package storage

import "fmt"

const apiVersion = "2015-06-15"
const apiProvider = "Microsoft.Storage"

func storageDefaultURLPathFunc(resourceGroupName, storageAccountName string) func() string {
	return func() string {
		return fmt.Sprintf("resourceGroups/%s/providers/%s/storageAccounts/%s", resourceGroupName, apiProvider, storageAccountName)
	}
}
