package applications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	. "code.cloudfoundry.org/cli/cf/i18n"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . Repository

type Repository interface {
	Create(params models.AppParams) (createdApp models.Application, apiErr error)
	GetApp(appGUID string) (models.Application, error)
	Read(name string) (app models.Application, apiErr error)
	ReadFromSpace(name string, spaceGUID string) (app models.Application, apiErr error)
	Update(appGUID string, params models.AppParams) (updatedApp models.Application, apiErr error)
	Delete(appGUID string) (apiErr error)
	ReadEnv(guid string) (*models.Environment, error)
	CreateRestageRequest(guid string) (apiErr error)
}

type CloudControllerRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerRepository) Create(params models.AppParams) (models.Application, error) {
	appResource := resources.NewApplicationEntityFromAppParams(params)
	data, err := json.Marshal(appResource)
	if err != nil {
		return models.Application{}, fmt.Errorf("%s: %s", T("Failed to marshal JSON"), err.Error())
	}

	resource := new(resources.ApplicationResource)
	err = repo.gateway.CreateResource(repo.config.APIEndpoint(), "/v2/apps", bytes.NewReader(data), resource)
	if err != nil {
		return models.Application{}, err
	}

	return resource.ToModel(), nil
}

func (repo CloudControllerRepository) GetApp(appGUID string) (app models.Application, apiErr error) {
	path := fmt.Sprintf("%s/v2/apps/%s", repo.config.APIEndpoint(), appGUID)
	appResources := new(resources.ApplicationResource)

	apiErr = repo.gateway.GetResource(path, appResources)
	if apiErr != nil {
		return
	}

	app = appResources.ToModel()
	return
}

func (repo CloudControllerRepository) Read(name string) (app models.Application, apiErr error) {
	return repo.ReadFromSpace(name, repo.config.SpaceFields().GUID)
}

func (repo CloudControllerRepository) ReadFromSpace(name string, spaceGUID string) (app models.Application, apiErr error) {
	path := fmt.Sprintf("%s/v2/spaces/%s/apps?q=%s&inline-relations-depth=1", repo.config.APIEndpoint(), spaceGUID, url.QueryEscape("name:"+name))
	appResources := new(resources.PaginatedApplicationResources)
	apiErr = repo.gateway.GetResource(path, appResources)
	if apiErr != nil {
		return
	}

	if len(appResources.Resources) == 0 {
		apiErr = errors.NewModelNotFoundError("App", name)
		return
	}

	res := appResources.Resources[0]
	app = res.ToModel()
	return
}

func (repo CloudControllerRepository) Update(appGUID string, params models.AppParams) (updatedApp models.Application, apiErr error) {
	appResource := resources.NewApplicationEntityFromAppParams(params)
	data, err := json.Marshal(appResource)
	if err != nil {
		return models.Application{}, fmt.Errorf("%s: %s", T("Failed to marshal JSON"), err.Error())
	}

	path := fmt.Sprintf("/v2/apps/%s?inline-relations-depth=1", appGUID)
	resource := new(resources.ApplicationResource)
	apiErr = repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, bytes.NewReader(data), resource)
	if apiErr != nil {
		return
	}

	updatedApp = resource.ToModel()
	return
}

func (repo CloudControllerRepository) Delete(appGUID string) (apiErr error) {
	path := fmt.Sprintf("/v2/apps/%s?recursive=true", appGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}

func (repo CloudControllerRepository) ReadEnv(guid string) (*models.Environment, error) {
	var (
		err error
	)

	path := fmt.Sprintf("%s/v2/apps/%s/env", repo.config.APIEndpoint(), guid)
	appResource := models.NewEnvironment()

	err = repo.gateway.GetResource(path, appResource)
	if err != nil {
		return &models.Environment{}, err
	}

	return appResource, err
}

func (repo CloudControllerRepository) CreateRestageRequest(guid string) error {
	path := fmt.Sprintf("/v2/apps/%s/restage", guid)
	return repo.gateway.CreateResource(repo.config.APIEndpoint(), path, strings.NewReader(""), nil)
}
