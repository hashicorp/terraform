package dns

import "github.com/jen20/riviera/azure"

type TXTRecord struct {
	Value string `json:"value" mapstructure:"value"`
}

type CreateTXTRecordSetResponse struct {
	ID         string             `mapstructure:"id"`
	Name       string             `mapstructure:"name"`
	Location   string             `mapstructure:"location"`
	Tags       map[string]*string `mapstructure:"tags"`
	TTL        *int               `mapstructure:"TTL"`
	TXTRecords []TXTRecord        `mapstructure:"TXTRecords"`
}

type CreateTXTRecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	TXTRecords        []TXTRecord        `json:"TXTRecords"`
}

func (command CreateTXTRecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "TXT", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateTXTRecordSetResponse{}
		},
	}
}
