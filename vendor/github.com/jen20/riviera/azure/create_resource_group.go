package azure

type CreateResourceGroupResponse struct {
	ID                *string `mapstructure:"id"`
	Name              *string `mapstructure:"name"`
	Location          *string `mapstructure:"location"`
	ProvisioningState *string `mapstructure:"provisioningState"`
}

type CreateResourceGroup struct {
	Name     string             `json:"-"`
	Location string             `json:"-" riviera:"location"`
	Tags     map[string]*string `json:"-" riviera:"tags"`
}

func (command CreateResourceGroup) APIInfo() APIInfo {
	return APIInfo{
		APIVersion:  resourceGroupAPIVersion,
		Method:      "PUT",
		URLPathFunc: resourceGroupDefaultURLFunc(command.Name),
		ResponseTypeFunc: func() interface{} {
			return &CreateResourceGroupResponse{}
		},
	}
}
