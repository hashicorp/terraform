package dns

import "github.com/jen20/riviera/azure"

type AAAARecord struct {
	IPv6Address string `json:"ipv6Address" mapstructure:"ipv6Address"`
}

type CreateAAAARecordSetResponse struct {
	ID          string             `mapstructure:"id"`
	Name        string             `mapstructure:"name"`
	Location    string             `mapstructure:"location"`
	Tags        map[string]*string `mapstructure:"tags"`
	TTL         *int               `mapstructure:"TTL"`
	AAAARecords []AAAARecord       `mapstructure:"AAAARecords"`
}

type CreateAAAARecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	AAAARecords       []AAAARecord       `json:"AAAARecords"`
}

func (command CreateAAAARecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "AAAA", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateAAAARecordSetResponse{}
		},
	}
}
