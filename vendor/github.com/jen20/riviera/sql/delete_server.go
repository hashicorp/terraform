package sql

import "github.com/jen20/riviera/azure"

type DeleteServer struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s DeleteServer) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "DELETE",
		URLPathFunc: sqlServerDefaultURLPath(s.ResourceGroupName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
