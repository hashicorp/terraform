package dns

import "github.com/jen20/riviera/azure"

type GetARecordSetResponse struct {
	ID       string             `mapstructure:"id"`
	Name     string             `mapstructure:"name"`
	Location string             `mapstructure:"location"`
	Tags     map[string]*string `mapstructure:"tags"`
	TTL      *int               `mapstructure:"TTL"`
	ARecords []ARecord          `mapstructure:"ARecords"`
}

type GetARecordSet struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ZoneName          string `json:"-"`
}

func (command GetARecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "A", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetARecordSetResponse{}
		},
	}
}
