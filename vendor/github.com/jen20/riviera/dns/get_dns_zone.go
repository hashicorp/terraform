package dns

import "github.com/jen20/riviera/azure"

type GetDNSZoneResponse struct {
	ID                    *string             `mapstructure:"id"`
	Name                  *string             `mapstructure:"name"`
	Location              *string             `mapstructure:"location"`
	Tags                  *map[string]*string `mapstructure:"tags"`
	NumberOfRecordSets    *string             `mapstructure:"numberOfRecordSets"`
	MaxNumberOfRecordSets *string             `mapstructure:"maxNumberOfRecordSets"`
}

type GetDNSZone struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s GetDNSZone) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: dnsZoneDefaultURLPathFunc(s.ResourceGroupName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetDNSZoneResponse{}
		},
	}
}
