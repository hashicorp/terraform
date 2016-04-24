package dns

import "github.com/jen20/riviera/azure"

type GetNSRecordSetResponse struct {
	ID        string             `mapstructure:"id"`
	Name      string             `mapstructure:"name"`
	Location  string             `mapstructure:"location"`
	Tags      map[string]*string `mapstructure:"tags"`
	TTL       *int               `mapstructure:"TTL"`
	NSRecords []NSRecord         `mapstructure:"NSRecords"`
}

type GetNSRecordSet struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ZoneName          string `json:"-"`
}

func (command GetNSRecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "NS", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetNSRecordSetResponse{}
		},
	}
}
