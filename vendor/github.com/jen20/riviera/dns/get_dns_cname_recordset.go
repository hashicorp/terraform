package dns

import "github.com/jen20/riviera/azure"

type GetCNAMERecordSetResponse struct {
	ID           string             `mapstructure:"id"`
	Name         string             `mapstructure:"name"`
	Location     string             `mapstructure:"location"`
	Tags         map[string]*string `mapstructure:"tags"`
	TTL          *int               `mapstructure:"TTL"`
	CNAMERecord  CNAMERecord        `mapstructure:"CNAMERecord"`
}

type GetCNAMERecordSet struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ZoneName          string `json:"-"`
}

func (command GetCNAMERecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "CNAME", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetCNAMERecordSetResponse{}
		},
	}
}
