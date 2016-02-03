package sql

import "github.com/jen20/riviera/azure"

type GetFirewallRuleResponse struct {
	ID             *string `mapstructure:"id"`
	Name           *string `mapstructure:"name"`
	Location       *string `mapstructure:"location"`
	StartIpAddress *string `json:"startIpAddress,omitempty"`
	EndIpAddress   *string `json:"endIpAddress,omitempty"`
}

type GetFirewallRule struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
	ServerName        string `json:"-"`
}

func (s GetFirewallRule) ApiInfo() azure.ApiInfo {
	return azure.ApiInfo{
		ApiVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: sqlServerFirewallDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetFirewallRuleResponse{}
		},
	}
}
