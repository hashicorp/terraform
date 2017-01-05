package api

import (
	"bytes"
	"encoding/json"
	"fmt"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . ServiceBindingRepository

type ServiceBindingRepository interface {
	Create(instanceGUID string, appGUID string, paramsMap map[string]interface{}) error
	Delete(instance models.ServiceInstance, appGUID string) (bool, error)
	ListAllForService(instanceGUID string) ([]models.ServiceBindingFields, error)
}

type CloudControllerServiceBindingRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerServiceBindingRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerServiceBindingRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerServiceBindingRepository) Create(instanceGUID, appGUID string, paramsMap map[string]interface{}) error {
	path := "/v2/service_bindings"
	request := models.ServiceBindingRequest{
		AppGUID:             appGUID,
		ServiceInstanceGUID: instanceGUID,
		Params:              paramsMap,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}

	return repo.gateway.CreateResource(repo.config.APIEndpoint(), path, bytes.NewReader(jsonBytes))
}

func (repo CloudControllerServiceBindingRepository) Delete(instance models.ServiceInstance, appGUID string) (bool, error) {
	var path string
	for _, binding := range instance.ServiceBindings {
		if binding.AppGUID == appGUID {
			path = binding.URL
			break
		}
	}

	if path == "" {
		return false, nil
	}

	return true, repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}

func (repo CloudControllerServiceBindingRepository) ListAllForService(instanceGUID string) ([]models.ServiceBindingFields, error) {
	serviceBindings := []models.ServiceBindingFields{}
	err := repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/service_instances/%s/service_bindings", instanceGUID),
		resources.ServiceBindingResource{},
		func(resource interface{}) bool {
			if serviceBindingResource, ok := resource.(resources.ServiceBindingResource); ok {
				serviceBindings = append(serviceBindings, serviceBindingResource.ToFields())
			}
			return true
		},
	)
	return serviceBindings, err
}
