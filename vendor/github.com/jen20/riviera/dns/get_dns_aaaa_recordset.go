package dns

import "github.com/jen20/riviera/azure"

type GetAAAARecordSetResponse struct {
	ID          string             `mapstructure:"id"`
	Name        string             `mapstructure:"name"`
	Location    string             `mapstructure:"location"`
	Tags        map[string]*string `mapstructure:"tags"`
	TTL         *int               `mapstructure:"TTL"`
	AAAARecords []AAAARecord       `mapstructure:"AAAARecords"`
}

type GetAAAARecordSet struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ZoneName          string `json:"-"`
}

func (command GetAAAARecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "AAAA", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetAAAARecordSetResponse{}
		},
	}
}
