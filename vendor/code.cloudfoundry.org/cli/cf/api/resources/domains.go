package resources

import "code.cloudfoundry.org/cli/cf/models"

type DomainResource struct {
	Resource
	Entity DomainEntity
}

type DomainEntity struct {
	Name                   string `json:"name"`
	OwningOrganizationGUID string `json:"owning_organization_guid,omitempty"`
	SharedOrganizationsURL string `json:"shared_organizations_url,omitempty"`
	RouterGroupGUID        string `json:"router_group_guid,omitempty"`
	RouterGroupType        string `json:"router_group_type,omitempty"`
	Wildcard               bool   `json:"wildcard"`
}

func (resource DomainResource) ToFields() models.DomainFields {
	privateDomain := resource.Entity.SharedOrganizationsURL != "" || resource.Entity.OwningOrganizationGUID != ""
	return models.DomainFields{
		Name: resource.Entity.Name,
		GUID: resource.Metadata.GUID,
		OwningOrganizationGUID: resource.Entity.OwningOrganizationGUID,
		Shared:                 !privateDomain,
		RouterGroupGUID:        resource.Entity.RouterGroupGUID,
		RouterGroupType:        resource.Entity.RouterGroupType,
	}
}
