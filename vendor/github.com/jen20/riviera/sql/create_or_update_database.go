package sql

import "github.com/jen20/riviera/azure"

type CreateOrUpdateDatabaseResponse struct {
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

type CreateOrUpdateDatabase struct {
	Name                          string             `json:"-"`
	ResourceGroupName             string             `json:"-"`
	ServerName                    string             `json:"-"`
	Location                      string             `json:"-" riviera:"location"`
	Tags                          map[string]*string `json:"-" riviera:"tags"`
	Edition                       *string            `json:"edition,omitempty"`
	Collation                     *string            `json:"collation,omitempty"`
	MaxSizeBytes                  *string            `json:"maxSizeBytes,omitempty"`
	RequestedServiceObjectiveName *string            `json:"requestedServiceObjectiveName,omitempty"`
	RequestedServiceObjectiveID   *string            `json:"requestedServiceObjectiveId,omitempty"`
	CreateMode                    *string            `json:"createMode,omitempty"`
	SourceDatabaseID              *string            `json:"sourceDatabaseId,omitempty"`
	SourceDatabaseDeletionDate    *string            `json:"sourceDatabaseDeletionDate,omitempty"`
	RestorePointInTime            *string            `json:"restorePointInTime,omitempty"`
	ElasticPoolName               *string            `json:"elasticPoolName,omitempty"`
}

func (s CreateOrUpdateDatabase) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: sqlDatabaseDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateOrUpdateDatabaseResponse{}
		},
	}
}
