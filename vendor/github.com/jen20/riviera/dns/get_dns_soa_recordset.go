package dns

import "github.com/jen20/riviera/azure"

type GetSOARecordSetResponse struct {
	ID        string             `mapstructure:"id"`
	Name      string             `mapstructure:"name"`
	Location  string             `mapstructure:"location"`
	Tags      map[string]*string `mapstructure:"tags"`
	TTL       *int               `mapstructure:"TTL"`
	SOARecord SOARecord          `mapstructure:"SOARecord"`
}

type GetSOARecordSet struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ZoneName          string `json:"-"`
}

func (command GetSOARecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "SOA", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetSOARecordSetResponse{}
		},
	}
}
