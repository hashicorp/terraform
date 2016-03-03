package dns

import "github.com/jen20/riviera/azure"

type NSRecord struct {
	NSDName string `json:"nsdname" mapstructure:"nsdname"`
}

type CreateNSRecordSetResponse struct {
	ID        string             `mapstructure:"id"`
	Name      string             `mapstructure:"name"`
	Location  string             `mapstructure:"location"`
	Tags      map[string]*string `mapstructure:"tags"`
	TTL       *int               `mapstructure:"TTL"`
	NSRecords []NSRecord         `mapstructure:"NSRecords"`
}

type CreateNSRecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	NSRecords         []NSRecord         `json:"NSRecords"`
}

func (command CreateNSRecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "NS", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateNSRecordSetResponse{}
		},
	}
}
