package sql

import "github.com/jen20/riviera/azure"

type GetServerResponse struct {
	ID                         *string             `mapstructure:"id"`
	Name                       *string             `mapstructure:"name"`
	Location                   *string             `mapstructure:"location"`
	Tags                       *map[string]*string `mapstructure:"tags"`
	Kind                       *string             `mapstructure:"kind"`
	FullyQualifiedDomainName   *string             `mapstructure:"fullyQualifiedDomainName"`
	AdministratorLogin         *string             `mapstructure:"administratorLogin"`
	AdministratorLoginPassword *string             `mapstructure:"administratorLoginPassword"`
	ExternalAdministratorLogin *string             `mapstructure:"externalAdministratorLogin"`
	ExternalAdministratorSid   *string             `mapstructure:"externalAdministratorSid"`
	Version                    *string             `mapstructure:"version"`
	State                      *string             `mapstructure:"state"`
}

type GetServer struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s GetServer) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: sqlServerDefaultURLPath(s.ResourceGroupName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetServerResponse{}
		},
	}
}
