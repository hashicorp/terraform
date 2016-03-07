package dns

import "github.com/jen20/riviera/azure"

type GetTXTRecordSetResponse struct {
	ID         string             `mapstructure:"id"`
	Name       string             `mapstructure:"name"`
	Location   string             `mapstructure:"location"`
	Tags       map[string]*string `mapstructure:"tags"`
	TTL        *int               `mapstructure:"TTL"`
	TXTRecords []TXTRecord        `mapstructure:"TXTRecords"`
}

type GetTXTRecordSet struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ZoneName          string `json:"-"`
}

func (command GetTXTRecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "TXT", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetTXTRecordSetResponse{}
		},
	}
}
