package azure

type GetResourceGroupResponse struct {
	ID                *string             `mapstructure:"id"`
	Name              *string             `mapstructure:"name"`
	Location          *string             `mapstructure:"location"`
	ProvisioningState *string             `mapstructure:"provisioningState"`
	Tags              *map[string]*string `mapstructure:"tags"`
}

type GetResourceGroup struct {
	Name string `json:"-"`
}

func (command GetResourceGroup) APIInfo() APIInfo {
	return APIInfo{
		APIVersion:  resourceGroupAPIVersion,
		Method:      "GET",
		URLPathFunc: resourceGroupDefaultURLFunc(command.Name),
		ResponseTypeFunc: func() interface{} {
			return &GetResourceGroupResponse{}
		},
	}
}
