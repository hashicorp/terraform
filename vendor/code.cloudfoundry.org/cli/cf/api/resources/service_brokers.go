package resources

import "code.cloudfoundry.org/cli/cf/models"

type ServiceBrokerResource struct {
	Resource
	Entity ServiceBrokerEntity
}

type ServiceBrokerEntity struct {
	GUID     string
	Name     string
	Password string `json:"auth_password"`
	Username string `json:"auth_username"`
	URL      string `json:"broker_url"`
}

func (resource ServiceBrokerResource) ToFields() (fields models.ServiceBroker) {
	fields.Name = resource.Entity.Name
	fields.GUID = resource.Metadata.GUID
	fields.URL = resource.Entity.URL
	fields.Username = resource.Entity.Username
	fields.Password = resource.Entity.Password
	return
}
