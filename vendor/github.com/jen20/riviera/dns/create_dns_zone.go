package dns

import "github.com/jen20/riviera/azure"

type CreateDNSZone struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
}

func (command CreateDNSZone) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsZoneDefaultURLPathFunc(command.ResourceGroupName, command.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
