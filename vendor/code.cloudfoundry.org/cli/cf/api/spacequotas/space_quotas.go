package spacequotas

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . SpaceQuotaRepository

type SpaceQuotaRepository interface {
	FindByName(name string) (quota models.SpaceQuota, apiErr error)
	FindByOrg(guid string) (quota []models.SpaceQuota, apiErr error)
	FindByGUID(guid string) (quota models.SpaceQuota, apiErr error)
	FindByNameAndOrgGUID(spaceQuotaName string, orgGUID string) (quota models.SpaceQuota, apiErr error)

	AssociateSpaceWithQuota(spaceGUID string, quotaGUID string) error
	UnassignQuotaFromSpace(spaceGUID string, quotaGUID string) error

	// CRUD ahoy
	Create(quota models.SpaceQuota) error
	Update(quota models.SpaceQuota) error
	Delete(quotaGUID string) error
}

type CloudControllerSpaceQuotaRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerSpaceQuotaRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerSpaceQuotaRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerSpaceQuotaRepository) findAllWithPath(path string) ([]models.SpaceQuota, error) {
	var quotas []models.SpaceQuota
	apiErr := repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		path,
		resources.SpaceQuotaResource{},
		func(resource interface{}) bool {
			if qr, ok := resource.(resources.SpaceQuotaResource); ok {
				quotas = append(quotas, qr.ToModel())
			}
			return true
		})
	return quotas, apiErr
}

func (repo CloudControllerSpaceQuotaRepository) FindByName(name string) (quota models.SpaceQuota, apiErr error) {
	return repo.FindByNameAndOrgGUID(name, repo.config.OrganizationFields().GUID)
}

func (repo CloudControllerSpaceQuotaRepository) FindByNameAndOrgGUID(spaceQuotaName string, orgGUID string) (models.SpaceQuota, error) {
	quotas, apiErr := repo.FindByOrg(orgGUID)
	if apiErr != nil {
		return models.SpaceQuota{}, apiErr
	}

	for _, quota := range quotas {
		if quota.Name == spaceQuotaName {
			return quota, nil
		}
	}

	apiErr = errors.NewModelNotFoundError("Space Quota", spaceQuotaName)
	return models.SpaceQuota{}, apiErr
}

func (repo CloudControllerSpaceQuotaRepository) FindByOrg(guid string) ([]models.SpaceQuota, error) {
	path := fmt.Sprintf("/v2/organizations/%s/space_quota_definitions", guid)
	quotas, apiErr := repo.findAllWithPath(path)
	if apiErr != nil {
		return nil, apiErr
	}
	return quotas, nil
}

func (repo CloudControllerSpaceQuotaRepository) FindByGUID(guid string) (quota models.SpaceQuota, apiErr error) {
	quotas, apiErr := repo.FindByOrg(repo.config.OrganizationFields().GUID)
	if apiErr != nil {
		return
	}

	for _, quota := range quotas {
		if quota.GUID == guid {
			return quota, nil
		}
	}

	apiErr = errors.NewModelNotFoundError("Space Quota", guid)
	return models.SpaceQuota{}, apiErr
}

func (repo CloudControllerSpaceQuotaRepository) Create(quota models.SpaceQuota) error {
	path := "/v2/space_quota_definitions"
	return repo.gateway.CreateResourceFromStruct(repo.config.APIEndpoint(), path, quota)
}

func (repo CloudControllerSpaceQuotaRepository) Update(quota models.SpaceQuota) error {
	path := fmt.Sprintf("/v2/space_quota_definitions/%s", quota.GUID)
	return repo.gateway.UpdateResourceFromStruct(repo.config.APIEndpoint(), path, quota)
}

func (repo CloudControllerSpaceQuotaRepository) AssociateSpaceWithQuota(spaceGUID string, quotaGUID string) error {
	path := fmt.Sprintf("/v2/space_quota_definitions/%s/spaces/%s", quotaGUID, spaceGUID)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, strings.NewReader(""))
}

func (repo CloudControllerSpaceQuotaRepository) UnassignQuotaFromSpace(spaceGUID string, quotaGUID string) error {
	path := fmt.Sprintf("/v2/space_quota_definitions/%s/spaces/%s", quotaGUID, spaceGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}

func (repo CloudControllerSpaceQuotaRepository) Delete(quotaGUID string) (apiErr error) {
	path := fmt.Sprintf("/v2/space_quota_definitions/%s", quotaGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}
