package sql

import "github.com/jen20/riviera/azure"

type CreateOrUpdateFirewallRuleResponse struct {
	ID             *string `mapstructure:"id"`
	Name           *string `mapstructure:"name"`
	Location       *string `mapstructure:"location"`
	StartIPAddress *string `json:"startIpAddress,omitempty"`
	EndIPAddress   *string `json:"endIpAddress,omitempty"`
}

type CreateOrUpdateFirewallRule struct {
	Name              string  `json:"-"`
	ResourceGroupName string  `json:"-"`
	ServerName        string  `json:"-"`
	StartIPAddress    *string `json:"startIpAddress,omitempty"`
	EndIPAddress      *string `json:"endIpAddress,omitempty"`
}

func (s CreateOrUpdateFirewallRule) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: sqlServerFirewallDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateOrUpdateFirewallRuleResponse{}
		},
	}
}
