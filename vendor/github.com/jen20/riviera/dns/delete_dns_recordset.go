package dns

import "github.com/jen20/riviera/azure"

type DeleteRecordSet struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ZoneName          string `json:"-"`
	RecordSetType     string `json:"-"`
}

func (command DeleteRecordSet) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "DELETE",
		URLPathFunc: dnsRecordSetDefaultURLPathFunc(command.ResourceGroupName, command.ZoneName, command.RecordSetType, command.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
