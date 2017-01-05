package organizations

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

//go:generate counterfeiter . OrganizationRepository

type OrganizationRepository interface {
	ListOrgs(limit int) ([]models.Organization, error)
	GetManyOrgsByGUID(orgGUIDs []string) (orgs []models.Organization, apiErr error)
	FindByName(name string) (org models.Organization, apiErr error)
	Create(org models.Organization) (apiErr error)
	Rename(orgGUID string, name string) (apiErr error)
	Delete(orgGUID string) (apiErr error)
	SharePrivateDomain(orgGUID string, domainGUID string) (apiErr error)
	UnsharePrivateDomain(orgGUID string, domainGUID string) (apiErr error)
}

type CloudControllerOrganizationRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerOrganizationRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerOrganizationRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerOrganizationRepository) ListOrgs(limit int) ([]models.Organization, error) {
	orgs := []models.Organization{}
	err := repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		"/v2/organizations?order-by=name",
		resources.OrganizationResource{},
		func(resource interface{}) bool {
			if orgResource, ok := resource.(resources.OrganizationResource); ok {
				orgs = append(orgs, orgResource.ToModel())
				return limit == 0 || len(orgs) < limit
			}
			return false
		},
	)
	return orgs, err
}

func (repo CloudControllerOrganizationRepository) GetManyOrgsByGUID(orgGUIDs []string) (orgs []models.Organization, err error) {
	for _, orgGUID := range orgGUIDs {
		url := fmt.Sprintf("%s/v2/organizations/%s", repo.config.APIEndpoint(), orgGUID)
		orgResource := resources.OrganizationResource{}
		err = repo.gateway.GetResource(url, &orgResource)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, orgResource.ToModel())
	}
	return
}

func (repo CloudControllerOrganizationRepository) FindByName(name string) (org models.Organization, apiErr error) {
	found := false
	apiErr = repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/organizations?q=%s&inline-relations-depth=1", url.QueryEscape("name:"+strings.ToLower(name))),
		resources.OrganizationResource{},
		func(resource interface{}) bool {
			org = resource.(resources.OrganizationResource).ToModel()
			found = true
			return false
		})

	if apiErr == nil && !found {
		apiErr = errors.NewModelNotFoundError("Organization", name)
	}

	return
}

func (repo CloudControllerOrganizationRepository) Create(org models.Organization) (apiErr error) {
	data := fmt.Sprintf(`{"name":"%s"`, org.Name)
	if org.QuotaDefinition.GUID != "" {
		data = data + fmt.Sprintf(`, "quota_definition_guid":"%s"`, org.QuotaDefinition.GUID)
	}
	data = data + "}"
	return repo.gateway.CreateResource(repo.config.APIEndpoint(), "/v2/organizations", strings.NewReader(data))
}

func (repo CloudControllerOrganizationRepository) Rename(orgGUID string, name string) (apiErr error) {
	url := fmt.Sprintf("/v2/organizations/%s", orgGUID)
	data := fmt.Sprintf(`{"name":"%s"}`, name)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), url, strings.NewReader(data))
}

func (repo CloudControllerOrganizationRepository) Delete(orgGUID string) (apiErr error) {
	url := fmt.Sprintf("/v2/organizations/%s?recursive=true", orgGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), url)
}

func (repo CloudControllerOrganizationRepository) SharePrivateDomain(orgGUID string, domainGUID string) error {
	url := fmt.Sprintf("/v2/organizations/%s/private_domains/%s", orgGUID, domainGUID)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), url, nil)
}

func (repo CloudControllerOrganizationRepository) UnsharePrivateDomain(orgGUID string, domainGUID string) error {
	url := fmt.Sprintf("/v2/organizations/%s/private_domains/%s", orgGUID, domainGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), url)
}
