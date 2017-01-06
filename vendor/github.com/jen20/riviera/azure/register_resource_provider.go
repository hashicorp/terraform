package azure

import "fmt"

type RegisterResourceProviderResponse struct {
	ID                *string `mapstructure:"id"`
	Namespace         *string `mapstructure:"namespace"`
	RegistrationState *string `mapstructure:"registrationState"`
	ApplicationID     *string `mapstructure:"applicationId"`
}

type RegisterResourceProvider struct {
	Namespace string `json:"-"`
}

func (command RegisterResourceProvider) APIInfo() APIInfo {
	return APIInfo{
		APIVersion: resourceGroupAPIVersion,
		Method:     "POST",
		URLPathFunc: func() string {
			return fmt.Sprintf("providers/%s/register", command.Namespace)
		},
		ResponseTypeFunc: func() interface{} {
			return &RegisterResourceProviderResponse{}
		},
	}
}
