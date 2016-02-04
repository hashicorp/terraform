package sql

import "github.com/jen20/riviera/azure"

type DeleteDatabase struct {
	Name              string `json:"-"`
	ServerName        string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s DeleteDatabase) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "DELETE",
		URLPathFunc: sqlDatabaseDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
