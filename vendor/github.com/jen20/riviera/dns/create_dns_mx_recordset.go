package dns

import "github.com/jen20/riviera/azure"

type MXRecord struct {
	Preference string `json:"preference" mapstructure:"preference"` //*Why* is this a string in the API?!
	Exchange   string `json:"exchange" mapstructure:"exchange"`
}

type CreateMXRecordSetResponse struct {
	ID        string             `mapstructure:"id"`
	Name      string             `mapstructure:"name"`
	Location  string             `mapstructure:"location"`
	Tags      map[string]*string `mapstructure:"tags"`
	TTL       *int               `mapstructure:"TTL"`
	MXRecords []MXRecord         `mapstructure:"MXRecords"`
}

type CreateMXRecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	MXRecords         []MXRecord         `json:"MXRecords"`
}

func (command CreateMXRecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "MX", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateMXRecordSetResponse{}
		},
	}
}
