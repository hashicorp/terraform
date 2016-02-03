package sql

import "github.com/jen20/riviera/azure"

type GetDatabaseResponse struct {
	ID                             *string            `mapstructure:"id"`
	Name                           *string            `mapstructure:"name"`
	Location                       *string            `mapstructure:"location"`
	Tags                           *map[string]string `mapstructure:"tags"`
	DatabaseID                     *string            `mapstructure:"databaseId"`
	DatabaseName                   *string            `mapstructure:"databaseName"`
	Edition                        *string            `mapstructure:"edition"`
	ServiceLevelObjective          *string            `mapstructure:"serviceLevelObjective"`
	MaxSizeInBytes                 *string            `mapstructure:"maxSizeInBytes"`
	CreationDate                   *string            `mapstructure:"creationDate"`
	CurrentServiceLevelObjectiveID *string            `mapstructure:"currentServiceLevelObjectiveId"`
	RequestedServiceObjectiveID    *string            `mapstructure:"requestedServiceObjectiveId"`
	DefaultSecondaryLocation       *string            `mapstructure:"defaultSecondaryLocation"`
	Encryption                     *string            `mapstructure:"encryption"`
}

type GetDatabase struct {
	Name              string `json:"-"`
	ServerName        string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s GetDatabase) ApiInfo() azure.ApiInfo {
	return azure.ApiInfo{
		ApiVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: sqlDatabaseDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetDatabaseResponse{}
		},
	}
}
