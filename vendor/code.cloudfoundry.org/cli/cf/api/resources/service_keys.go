package resources

import "code.cloudfoundry.org/cli/cf/models"

type ServiceKeyResource struct {
	Resource
	Entity ServiceKeyEntity
}

type ServiceKeyEntity struct {
	Name                string                 `json:"name"`
	ServiceInstanceGUID string                 `json:"service_instance_guid"`
	ServiceInstanceURL  string                 `json:"service_instance_url"`
	Credentials         map[string]interface{} `json:"credentials"`
}

func (resource ServiceKeyResource) ToFields() models.ServiceKeyFields {
	return models.ServiceKeyFields{
		Name: resource.Entity.Name,
		URL:  resource.Metadata.URL,
		GUID: resource.Metadata.GUID,
	}
}

func (resource ServiceKeyResource) ToModel() models.ServiceKey {
	return models.ServiceKey{
		Fields: models.ServiceKeyFields{
			Name: resource.Entity.Name,
			GUID: resource.Metadata.GUID,
			URL:  resource.Metadata.URL,

			ServiceInstanceGUID: resource.Entity.ServiceInstanceGUID,
			ServiceInstanceURL:  resource.Entity.ServiceInstanceURL,
		},
		Credentials: resource.Entity.Credentials,
	}
}
