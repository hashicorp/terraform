package dns

import "github.com/jen20/riviera/azure"

type CNAMERecord struct {
	CNAME string `json:"cname" mapstructure:"cname"`
}

type CreateCNAMERecordSetResponse struct {
	ID           string             `mapstructure:"id"`
	Name         string             `mapstructure:"name"`
	Location     string             `mapstructure:"location"`
	Tags         map[string]*string `mapstructure:"tags"`
	TTL          *int               `mapstructure:"TTL"`
	CNAMERecord  CNAMERecord        `mapstructure:"CNAMERecord"`
}

type CreateCNAMERecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	CNAMERecord       CNAMERecord        `json:"CNAMERecord"`
}

func (command CreateCNAMERecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "CNAME", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateCNAMERecordSetResponse{}
		},
	}
}
