package sql

import "github.com/jen20/riviera/azure"

type CreateOrUpdateServerResponse struct {
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

type CreateOrUpdateServer struct {
	Name                       string             `json:"-"`
	ResourceGroupName          string             `json:"-"`
	Location                   string             `json:"-" riviera:"location"`
	Tags                       map[string]*string `json:"-" riviera:"tags"`
	AdministratorLogin         *string            `json:"administratorLogin,omitempty"`
	AdministratorLoginPassword *string            `json:"administratorLoginPassword,omitempty"`
	Version                    *string            `json:"version,omitempty"`
}

func (s CreateOrUpdateServer) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: sqlServerDefaultURLPath(s.ResourceGroupName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateOrUpdateServerResponse{}
		},
	}
}
