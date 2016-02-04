package sql

import "github.com/jen20/riviera/azure"

type CreateOrUpdateFirewallRuleResponse struct {
	ID             *string `mapstructure:"id"`
	Name           *string `mapstructure:"name"`
	Location       *string `mapstructure:"location"`
	StartIpAddress *string `json:"startIpAddress,omitempty"`
	EndIpAddress   *string `json:"endIpAddress,omitempty"`
}

type CreateOrUpdateFirewallRule struct {
	Name              string  `json:"-"`
	ResourceGroupName string  `json:"-"`
	ServerName        string  `json:"-"`
	StartIpAddress    *string `json:"startIpAddress,omitempty"`
	EndIpAddress      *string `json:"endIpAddress,omitempty"`
}

func (s CreateOrUpdateFirewallRule) ApiInfo() azure.ApiInfo {
	return azure.ApiInfo{
		ApiVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: sqlServerFirewallDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateOrUpdateFirewallRuleResponse{}
		},
	}
}
