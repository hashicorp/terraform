package dns

import "github.com/jen20/riviera/azure"

type SRVRecord struct {
	Priority int    `json:"priority" mapstructure:"priority"`
	Weight   int    `json:"weight" mapstructure:"weight"`
	Port     int    `json:"port" mapstructure:"port"`
	Target   string `json:"target" mapstructure:"target"`
}

type CreateSRVRecordSetResponse struct {
	ID         string             `mapstructure:"id"`
	Name       string             `mapstructure:"name"`
	Location   string             `mapstructure:"location"`
	Tags       map[string]*string `mapstructure:"tags"`
	TTL        *int               `mapstructure:"TTL"`
	SRVRecords []SRVRecord        `mapstructure:"SRVRecords"`
}

type CreateSRVRecordSet struct {
	Name              string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	ZoneName          string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	TTL               int                `json:"TTL"`
	SRVRecords        []SRVRecord        `json:"SRVRecords"`
}

func (command CreateSRVRecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, "SRV", command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateSRVRecordSetResponse{}
		},
	}
}
