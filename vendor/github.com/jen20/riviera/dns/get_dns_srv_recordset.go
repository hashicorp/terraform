package dns

import "github.com/jen20/riviera/azure"

type GetSRVRecordSetResponse struct {
	ID         string             `mapstructure:"id"`
	Name       string             `mapstructure:"name"`
	Location   string             `mapstructure:"location"`
	Tags       map[string]*string `mapstructure:"tags"`
	TTL        *int               `mapstructure:"TTL"`
	SRVRecords []SRVRecord        `mapstructure:"SRVRecords"`
}

type GetSRVRecordSet struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ZoneName          string `json:"-"`
}

func (command GetSRVRecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "SRV", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetSRVRecordSetResponse{}
		},
	}
}
