package dns

import "github.com/jen20/riviera/azure"

type DeleteDNSZone struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (command DeleteDNSZone) ApiInfo() azure.ApiInfo {
	return azure.ApiInfo{
		ApiVersion:         apiVersion,
		Method:             "DELETE",
		URLPathFunc:        dnsZoneDefaultURLPathFunc(command.ResourceGroupName, command.Name),
		SkipArmBoilerplate: true,
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
