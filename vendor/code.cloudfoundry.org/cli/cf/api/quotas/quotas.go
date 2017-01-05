package quotas

import (
	"fmt"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . QuotaRepository

type QuotaRepository interface {
	FindAll() (quotas []models.QuotaFields, apiErr error)
	FindByName(name string) (quota models.QuotaFields, apiErr error)

	AssignQuotaToOrg(orgGUID, quotaGUID string) error

	// CRUD ahoy
	Create(quota models.QuotaFields) error
	Update(quota models.QuotaFields) error
	Delete(quotaGUID string) error
}

type CloudControllerQuotaRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerQuotaRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerQuotaRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerQuotaRepository) findAllWithPath(path string) ([]models.QuotaFields, error) {
	var quotas []models.QuotaFields
	apiErr := repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		path,
		resources.QuotaResource{},
		func(resource interface{}) bool {
			if qr, ok := resource.(resources.QuotaResource); ok {
				quotas = append(quotas, qr.ToFields())
			}
			return true
		})
	return quotas, apiErr
}

func (repo CloudControllerQuotaRepository) FindAll() (quotas []models.QuotaFields, apiErr error) {
	return repo.findAllWithPath("/v2/quota_definitions")
}

func (repo CloudControllerQuotaRepository) FindByName(name string) (quota models.QuotaFields, apiErr error) {
	path := fmt.Sprintf("/v2/quota_definitions?q=%s", url.QueryEscape("name:"+name))
	quotas, apiErr := repo.findAllWithPath(path)
	if apiErr != nil {
		return
	}

	if len(quotas) == 0 {
		apiErr = errors.NewModelNotFoundError("Quota", name)
		return
	}

	quota = quotas[0]
	return
}

func (repo CloudControllerQuotaRepository) Create(quota models.QuotaFields) error {
	return repo.gateway.CreateResourceFromStruct(repo.config.APIEndpoint(), "/v2/quota_definitions", quota)
}

func (repo CloudControllerQuotaRepository) Update(quota models.QuotaFields) error {
	path := fmt.Sprintf("/v2/quota_definitions/%s", quota.GUID)
	return repo.gateway.UpdateResourceFromStruct(repo.config.APIEndpoint(), path, quota)
}

func (repo CloudControllerQuotaRepository) AssignQuotaToOrg(orgGUID, quotaGUID string) (apiErr error) {
	path := fmt.Sprintf("/v2/organizations/%s", orgGUID)
	data := fmt.Sprintf(`{"quota_definition_guid":"%s"}`, quotaGUID)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, strings.NewReader(data))
}

func (repo CloudControllerQuotaRepository) Delete(quotaGUID string) (apiErr error) {
	path := fmt.Sprintf("/v2/quota_definitions/%s", quotaGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}
