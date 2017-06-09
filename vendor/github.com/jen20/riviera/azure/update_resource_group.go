package azure

type UpdateResourceGroupResponse struct {
	ID                *string             `mapstructure:"id"`
	Name              *string             `mapstructure:"name"`
	Location          *string             `mapstructure:"location"`
	ProvisioningState *string             `mapstructure:"provisioningState"`
	Tags              *map[string]*string `mapstructure:"tags"`
}

type UpdateResourceGroup struct {
	Name string             `json:"-"`
	Tags map[string]*string `json:"-" riviera:"tags"`
}

func (command UpdateResourceGroup) APIInfo() APIInfo {
	return APIInfo{
		APIVersion:  resourceGroupAPIVersion,
		Method:      "PATCH",
		URLPathFunc: resourceGroupDefaultURLFunc(command.Name),
		ResponseTypeFunc: func() interface{} {
			return &UpdateResourceGroupResponse{}
		},
	}
}
