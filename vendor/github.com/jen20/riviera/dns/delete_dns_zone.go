package dns

import "github.com/jen20/riviera/azure"

type DeleteDNSZone struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (command DeleteDNSZone) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "DELETE",
		URLPathFunc: dnsZoneDefaultURLPathFunc(command.ResourceGroupName, command.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
