package resources

import "code.cloudfoundry.org/cli/cf/models"

type PaginatedQuotaResources struct {
	Resources []QuotaResource
}

type QuotaResource struct {
	Resource
	Entity models.QuotaResponse
}

func (resource QuotaResource) ToFields() models.QuotaFields {
	appInstanceLimit := UnlimitedAppInstances
	if resource.Entity.AppInstanceLimit != "" {
		i, err := resource.Entity.AppInstanceLimit.Int64()
		if err == nil {
			appInstanceLimit = int(i)
		}
	}

	return models.QuotaFields{
		GUID:                    resource.Metadata.GUID,
		Name:                    resource.Entity.Name,
		MemoryLimit:             resource.Entity.MemoryLimit,
		InstanceMemoryLimit:     resource.Entity.InstanceMemoryLimit,
		RoutesLimit:             resource.Entity.RoutesLimit,
		ServicesLimit:           resource.Entity.ServicesLimit,
		NonBasicServicesAllowed: resource.Entity.NonBasicServicesAllowed,
		AppInstanceLimit:        appInstanceLimit,
		ReservedRoutePorts:      resource.Entity.ReservedRoutePorts,
	}
}
