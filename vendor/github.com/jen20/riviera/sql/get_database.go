package sql

import "github.com/jen20/riviera/azure"

type GetDatabaseResponse struct {
	ID                             *string             `mapstructure:"id"`
	Name                           *string             `mapstructure:"name"`
	Location                       *string             `mapstructure:"location"`
	Tags                           *map[string]*string `mapstructure:"tags"`
	Kind                           *string             `mapstructure:"kind"`
	DatabaseID                     *string             `mapstructure:"databaseId"`
	DatabaseName                   *string             `mapstructure:"databaseName"`
	Status                         *string             `mapstructure:"status"`
	Collation                      *string             `mapstructure:"collation"`
	Edition                        *string             `mapstructure:"edition"`
	ServiceLevelObjective          *string             `mapstructure:"serviceLevelObjective"`
	MaxSizeInBytes                 *string             `mapstructure:"maxSizeInBytes"`
	CreationDate                   *string             `mapstructure:"creationDate"`
	CurrentServiceLevelObjectiveID *string             `mapstructure:"currentServiceLevelObjectiveId"`
	RequestedServiceObjectiveID    *string             `mapstructure:"requestedServiceObjectiveId"`
	RequestedServiceObjectiveName  *string             `mapstructure:"requestedServiceObjectiveName"`
	DefaultSecondaryLocation       *string             `mapstructure:"defaultSecondaryLocation"`
	Encryption                     *string             `mapstructure:"encryption"`
	EarliestRestoreDate            *string             `mapstructure:"earliestRestoreDate"`
	ElasticPoolName                *string             `mapstructure:"elasticPoolName"`
	ContainmentState               *string             `mapstructure:"containmentState"`
}

type GetDatabase struct {
	Name              string `json:"-"`
	ServerName        string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s GetDatabase) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: sqlDatabaseDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetDatabaseResponse{}
		},
	}
}
