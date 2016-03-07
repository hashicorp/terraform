package sql

import "github.com/jen20/riviera/azure"

type DeleteElasticDatabasePool struct {
	Name              string `json:"-"`
	ServerName        string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s DeleteElasticDatabasePool) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "DELETE",
		URLPathFunc: sqlElasticPoolDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
