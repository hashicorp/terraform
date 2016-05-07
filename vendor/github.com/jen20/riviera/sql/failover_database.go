package sql

import "github.com/jen20/riviera/azure"

type FailoverDatabase struct {
	DatabaseName      string `json:"-"`
	ServerName        string `json:"-"`
	ResourceGroupName string `json:"-"`
	LinkID            string `json:"-"`
}

func (s FailoverDatabase) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "POST",
		URLPathFunc: sqlDatabaseFailoverUnplanned(s.ResourceGroupName, s.ServerName, s.DatabaseName, s.LinkID),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
		HasBodyOverride: true,
	}
}
