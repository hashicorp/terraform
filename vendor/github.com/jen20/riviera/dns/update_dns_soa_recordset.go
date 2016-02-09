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

type UpdateSOARecordSetResponse struct {
	ID        string             `mapstructure:"id"`
	Name      string             `mapstructure:"name"`
	Location  string             `mapstructure:"location"`
	Tags      map[string]*string `mapstructure:"tags"`
	TTL       *int               `mapstructure:"TTL"`
	SOARecord SOARecord          `mapstructure:"SOARecord"`
}

type UpdateSOARecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	SOARecord         SOARecord          `json:"SOARecord"`
}

func (command UpdateSOARecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PATCH",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "SOA", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &UpdateSOARecordSetResponse{}
		},
	}
}
