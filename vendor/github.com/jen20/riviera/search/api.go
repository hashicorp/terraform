package search

import "fmt"

const apiVersion = "2015-08-19"
const apiProvider = "Microsoft.Search"

func searchServiceDefaultURLPath(resourceGroupName, serviceName string) func() string {
	return func() string {
		return fmt.Sprintf("resourceGroups/%s/providers/%s/searchServices/%s", resourceGroupName, apiProvider, serviceName)
	}
}
