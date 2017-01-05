package resources

import "code.cloudfoundry.org/cli/cf/models"

type UserResource struct {
	Resource
	Entity UserEntity
}

type UserEntity struct {
	Name  string `json:"username,omitempty"`
	Admin bool
}

type UAAUserResources struct {
	Resources []struct {
		ID       string
		Username string
	}
}

func (resource UserResource) ToFields() models.UserFields {
	return models.UserFields{
		GUID:     resource.Metadata.GUID,
		IsAdmin:  resource.Entity.Admin,
		Username: resource.Entity.Name,
	}
}

type UAAUserResourceEmail struct {
	Value string `json:"value"`
}

type UAAUserResourceName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

type UAAUserResource struct {
	Username string                 `json:"userName"`
	Password string                 `json:"password"`
	Name     UAAUserResourceName    `json:"name,omitempty"`
	Emails   []UAAUserResourceEmail `json:"emails,omitempty"`
}

func NewUAAUserResource(username, password string) UAAUserResource {
	return UAAUserResource{
		Username: username,
		Emails:   []UAAUserResourceEmail{{Value: username}},
		Password: password,
		Name: UAAUserResourceName{
			GivenName:  username,
			FamilyName: username,
		},
	}
}

type UAAUserFields struct {
	ID string
}
