package dns

import "github.com/jen20/riviera/azure"

type SOARecord struct {
	Email       string `json:"email" mapstructure:"email"`
	ExpireTime  int    `json:"expireTime" mapstructure:"expireTime"`
	Host        string `json:"host" mapstructure:"host"`
	MinimumTTL  int    `json:"minimumTTL" mapstructure:"minimumTTL"`
	RefreshTime int    `json:"refreshTime" mapstructure:"refreshTime"`
	RetryTime   int    `json:"retryTime" mapstructure:"retryTime"`
}

type CreateSOARecordSetResponse struct {
	ID         string             `mapstructure:"id"`
	Name       string             `mapstructure:"name"`
	Location   string             `mapstructure:"location"`
	Tags       map[string]*string `mapstructure:"tags"`
	TTL        *int               `mapstructure:"TTL"`
	SOARecords []SOARecord        `mapstructure:"SOARecords"`
}

type CreateSOARecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	SOARecords        []SOARecord        `json:"SOARecords"`
}

func (command CreateSOARecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "SOA", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateSOARecordSetResponse{}
		},
	}
}
