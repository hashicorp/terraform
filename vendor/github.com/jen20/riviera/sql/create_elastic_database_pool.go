package sql

import "github.com/jen20/riviera/azure"

type CreateElasticDatabasePool struct {
	Name              string             `json:"-"`
	ServerName        string             `json:"-"`
	ResourceGroupName string             `json:"-"`
	Location          string             `json:"-" riviera:"location"`
	Tags              map[string]*string `json:"-" riviera:"tags"`
	Edition           *string            `json:"edition,omitempty"`
	DTU               *string            `json:"dtu,omitempty"`
	StorageMB         *string            `json:"storageMB,omitempty"`
	DatabaseDTUMin    *string            `json:"databaseDtuMin,omitempty"`
	DatabaseDTUMax    *string            `json:"databaseDtuMax,omitempty"`
}

func (s CreateElasticDatabasePool) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "PUT",
		URLPathFunc: sqlElasticPoolDefaultURLPath(s.ResourceGroupName, s.ServerName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return nil
		},
	}
}
