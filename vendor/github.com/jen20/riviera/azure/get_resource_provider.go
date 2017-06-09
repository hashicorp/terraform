package azure

import "fmt"

type GetResourceProviderResponse struct {
	ID                *string `mapstructure:"id"`
	Namespace         *string `mapstructure:"namespace"`
	RegistrationState *string `mapstructure:"registrationState"`
}

type GetResourceProvider struct {
	Namespace string `json:"-"`
}

func (command GetResourceProvider) APIInfo() APIInfo {
	return APIInfo{
		APIVersion: resourceGroupAPIVersion,
		Method:     "GET",
		URLPathFunc: func() string {
			return fmt.Sprintf("providers/%s", command.Namespace)
		},
		ResponseTypeFunc: func() interface{} {
			return &GetResourceProviderResponse{}
		},
	}
}
