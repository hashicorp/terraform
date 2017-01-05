package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	. "code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . BuildpackRepository

type BuildpackRepository interface {
	FindByName(name string) (buildpack models.Buildpack, apiErr error)
	ListBuildpacks(func(models.Buildpack) bool) error
	Create(name string, position *int, enabled *bool, locked *bool) (createdBuildpack models.Buildpack, apiErr error)
	Delete(buildpackGUID string) (apiErr error)
	Update(buildpack models.Buildpack) (updatedBuildpack models.Buildpack, apiErr error)
}

type CloudControllerBuildpackRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerBuildpackRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerBuildpackRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerBuildpackRepository) ListBuildpacks(cb func(models.Buildpack) bool) error {
	return repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		buildpacksPath,
		resources.BuildpackResource{},
		func(resource interface{}) bool {
			return cb(resource.(resources.BuildpackResource).ToFields())
		})
}

func (repo CloudControllerBuildpackRepository) FindByName(name string) (buildpack models.Buildpack, apiErr error) {
	foundIt := false
	apiErr = repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("%s?q=%s", buildpacksPath, url.QueryEscape("name:"+name)),
		resources.BuildpackResource{},
		func(resource interface{}) bool {
			buildpack = resource.(resources.BuildpackResource).ToFields()
			foundIt = true
			return false
		})

	if !foundIt {
		apiErr = errors.NewModelNotFoundError("Buildpack", name)
	}
	return
}

func (repo CloudControllerBuildpackRepository) Create(name string, position *int, enabled *bool, locked *bool) (createdBuildpack models.Buildpack, apiErr error) {
	entity := resources.BuildpackEntity{Name: name, Position: position, Enabled: enabled, Locked: locked}
	body, err := json.Marshal(entity)
	if err != nil {
		apiErr = fmt.Errorf("%s: %s", T("Could not serialize information"), err.Error())
		return
	}

	resource := new(resources.BuildpackResource)
	apiErr = repo.gateway.CreateResource(repo.config.APIEndpoint(), buildpacksPath, bytes.NewReader(body), resource)
	if apiErr != nil {
		return
	}

	createdBuildpack = resource.ToFields()
	return
}

func (repo CloudControllerBuildpackRepository) Delete(buildpackGUID string) (apiErr error) {
	path := fmt.Sprintf("%s/%s", buildpacksPath, buildpackGUID)
	apiErr = repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
	return
}

func (repo CloudControllerBuildpackRepository) Update(buildpack models.Buildpack) (updatedBuildpack models.Buildpack, apiErr error) {
	path := fmt.Sprintf("%s/%s", buildpacksPath, buildpack.GUID)

	entity := resources.BuildpackEntity{
		Name:     buildpack.Name,
		Position: buildpack.Position,
		Enabled:  buildpack.Enabled,
		Key:      "",
		Filename: "",
		Locked:   buildpack.Locked,
	}

	body, err := json.Marshal(entity)
	if err != nil {
		apiErr = fmt.Errorf("%s: %s", T("Could not serialize updates."), err.Error())
		return
	}

	resource := new(resources.BuildpackResource)
	apiErr = repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, bytes.NewReader(body), resource)
	if apiErr != nil {
		return
	}

	updatedBuildpack = resource.ToFields()
	return
}

const buildpacksPath = "/v2/buildpacks"
