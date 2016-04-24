package sql

import "github.com/jen20/riviera/azure"

type DeleteFirewallRule struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ServerName        string `json:"-"`
}

func (s DeleteFirewallRule) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "DELETE",
		URLPathFunc: sqlServerFirewallDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
