package spaces

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . SpaceRepository

type SpaceRepository interface {
	ListSpaces(func(models.Space) bool) error
	ListSpacesFromOrg(orgGUID string, spaceFunc func(models.Space) bool) error
	FindByName(name string) (space models.Space, apiErr error)
	FindByNameInOrg(name, orgGUID string) (space models.Space, apiErr error)
	Create(name string, orgGUID string, spaceQuotaGUID string) (space models.Space, apiErr error)
	Rename(spaceGUID, newName string) (apiErr error)
	SetAllowSSH(spaceGUID string, allow bool) (apiErr error)
	Delete(spaceGUID string) (apiErr error)
}

type CloudControllerSpaceRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerSpaceRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerSpaceRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerSpaceRepository) ListSpaces(callback func(models.Space) bool) error {
	return repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/organizations/%s/spaces?order-by=name&inline-relations-depth=1", repo.config.OrganizationFields().GUID),
		resources.SpaceResource{},
		func(resource interface{}) bool {
			return callback(resource.(resources.SpaceResource).ToModel())
		})
}

func (repo CloudControllerSpaceRepository) ListSpacesFromOrg(orgGUID string, callback func(models.Space) bool) error {
	return repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/organizations/%s/spaces?order-by=name&inline-relations-depth=1", orgGUID),
		resources.SpaceResource{},
		func(resource interface{}) bool {
			return callback(resource.(resources.SpaceResource).ToModel())
		})
}

func (repo CloudControllerSpaceRepository) FindByName(name string) (space models.Space, apiErr error) {
	return repo.FindByNameInOrg(name, repo.config.OrganizationFields().GUID)
}

func (repo CloudControllerSpaceRepository) FindByNameInOrg(name, orgGUID string) (space models.Space, apiErr error) {
	foundSpace := false
	apiErr = repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/organizations/%s/spaces?q=%s&inline-relations-depth=1", orgGUID, url.QueryEscape("name:"+strings.ToLower(name))),
		resources.SpaceResource{},
		func(resource interface{}) bool {
			space = resource.(resources.SpaceResource).ToModel()
			foundSpace = true
			return false
		})

	if !foundSpace {
		apiErr = errors.NewModelNotFoundError("Space", name)
	}

	return
}

func (repo CloudControllerSpaceRepository) Create(name, orgGUID, spaceQuotaGUID string) (models.Space, error) {
	var space models.Space
	path := "/v2/spaces?inline-relations-depth=1"

	bodyMap := map[string]string{"name": name, "organization_guid": orgGUID}
	if spaceQuotaGUID != "" {
		bodyMap["space_quota_definition_guid"] = spaceQuotaGUID
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return models.Space{}, err
	}

	resource := new(resources.SpaceResource)
	err = repo.gateway.CreateResource(repo.config.APIEndpoint(), path, strings.NewReader(string(body)), resource)
	if err != nil {
		return models.Space{}, err
	}
	space = resource.ToModel()
	return space, nil
}

func (repo CloudControllerSpaceRepository) Rename(spaceGUID, newName string) (apiErr error) {
	path := fmt.Sprintf("/v2/spaces/%s", spaceGUID)
	body := fmt.Sprintf(`{"name":"%s"}`, newName)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, strings.NewReader(body))
}

func (repo CloudControllerSpaceRepository) SetAllowSSH(spaceGUID string, allow bool) (apiErr error) {
	path := fmt.Sprintf("/v2/spaces/%s", spaceGUID)
	body := fmt.Sprintf(`{"allow_ssh":%t}`, allow)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, strings.NewReader(body))
}

func (repo CloudControllerSpaceRepository) Delete(spaceGUID string) (apiErr error) {
	path := fmt.Sprintf("/v2/spaces/%s?recursive=true", spaceGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}
