package dns

import "github.com/jen20/riviera/azure"

type ARecord struct {
	IPv4Address string `json:"ipv4Address" mapstructure:"ipv4Address"`
}

type CreateARecordSetResponse struct {
	ID       string             `mapstructure:"id"`
	Name     string             `mapstructure:"name"`
	Location string             `mapstructure:"location"`
	Tags     map[string]*string `mapstructure:"tags"`
	TTL      *int               `mapstructure:"TTL"`
	ARecords []ARecord          `mapstructure:"ARecords"`
}

type CreateARecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	ARecords          []ARecord          `json:"ARecords"`
}

func (command CreateARecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "A", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateARecordSetResponse{}
		},
	}
}
