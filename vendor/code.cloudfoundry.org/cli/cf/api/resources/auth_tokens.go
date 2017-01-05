package resources

import "code.cloudfoundry.org/cli/cf/models"

type PaginatedAuthTokenResources struct {
	Resources []AuthTokenResource
}

type AuthTokenResource struct {
	Resource
	Entity AuthTokenEntity
}

type AuthTokenEntity struct {
	Label    string
	Provider string
}

func (resource AuthTokenResource) ToFields() (authToken models.ServiceAuthTokenFields) {
	authToken.GUID = resource.Metadata.GUID
	authToken.Label = resource.Entity.Label
	authToken.Provider = resource.Entity.Provider
	return
}
