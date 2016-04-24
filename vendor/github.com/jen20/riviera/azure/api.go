package azure

import "fmt"

const resourceGroupAPIVersion = "2015-01-01"

func resourceGroupDefaultURLFunc(resourceGroupName string) func() string {
	return func() string {
		return fmt.Sprintf("resourceGroups/%s", resourceGroupName)
	}
}
