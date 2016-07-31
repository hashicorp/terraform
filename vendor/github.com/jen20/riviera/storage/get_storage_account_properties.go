package storage

import "github.com/jen20/riviera/azure"

type GetStorageAccountPropertiesResponse struct {
	ID               *string `mapstructure:"id"`
	Name             *string `mapstructure:"name"`
	Location         *string `mapstructure:"location"`
	AccountType      *string `mapstructure:"accountType"`
	PrimaryEndpoints *struct {
		Blob  *string `mapstructure:"blob"`
		Queue *string `mapstructure:"queue"`
		Table *string `mapstructure:"table"`
		File  *string `mapstructure:"file"`
	} `mapstructure:"primaryEndpoints"`
	PrimaryLocation     *string `mapstructure:"primaryLocation"`
	StatusOfPrimary     *string `mapstructure:"statusOfPrimary"`
	LastGeoFailoverTime *string `mapstructure:"lastGeoFailoverTime"`
	SecondaryLocation   *string `mapstructure:"secondaryLocation"`
	StatusOfSecondary   *string `mapstructure:"statusOfSecondary"`
	SecondaryEndpoints  *struct {
		Blob  *string `mapstructure:"blob"`
		Queue *string `mapstructure:"queue"`
		Table *string `mapstructure:"table"`
	} `mapstructure:"secondaryEndpoints"`
	CreationTime *string `mapstructure:"creationTime"`
	CustomDomain *struct {
		Name *string `mapstructure:"name"`
	} `mapstructure:"customDomain"`
}

type GetStorageAccountProperties struct {
	Name              string `json:"-"`
	ResourceGroupName string `json:"-"`
}

func (s GetStorageAccountProperties) APIInfo() azure.APIInfo {
	return azure.APIInfo{
		APIVersion:  apiVersion,
		Method:      "GET",
		URLPathFunc: storageDefaultURLPathFunc(s.ResourceGroupName, s.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetStorageAccountPropertiesResponse{}
		},
	}
}
