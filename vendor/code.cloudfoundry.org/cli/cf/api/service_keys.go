package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . ServiceKeyRepository

type ServiceKeyRepository interface {
	CreateServiceKey(serviceKeyGUID string, keyName string, params map[string]interface{}) error
	ListServiceKeys(serviceKeyGUID string) ([]models.ServiceKey, error)
	GetServiceKey(serviceKeyGUID string, keyName string) (models.ServiceKey, error)
	DeleteServiceKey(serviceKeyGUID string) error
}

type CloudControllerServiceKeyRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerServiceKeyRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerServiceKeyRepository) {
	return CloudControllerServiceKeyRepository{
		config:  config,
		gateway: gateway,
	}
}

func (c CloudControllerServiceKeyRepository) CreateServiceKey(instanceGUID string, keyName string, params map[string]interface{}) error {
	path := "/v2/service_keys"

	request := models.ServiceKeyRequest{
		Name:                keyName,
		ServiceInstanceGUID: instanceGUID,
		Params:              params,
	}
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}

	err = c.gateway.CreateResource(c.config.APIEndpoint(), path, bytes.NewReader(jsonBytes))

	if httpErr, ok := err.(errors.HTTPError); ok {
		switch httpErr.ErrorCode() {
		case errors.ServiceKeyNameTaken:
			return errors.NewModelAlreadyExistsError("Service key", keyName)
		case errors.UnbindableService:
			return errors.NewUnbindableServiceError()
		default:
			return errors.New(httpErr.Error())
		}
	}

	return nil
}

func (c CloudControllerServiceKeyRepository) ListServiceKeys(instanceGUID string) ([]models.ServiceKey, error) {
	path := fmt.Sprintf("/v2/service_instances/%s/service_keys", instanceGUID)

	return c.listServiceKeys(path)
}

func (c CloudControllerServiceKeyRepository) GetServiceKey(instanceGUID string, keyName string) (models.ServiceKey, error) {
	path := fmt.Sprintf("/v2/service_instances/%s/service_keys?q=%s", instanceGUID, url.QueryEscape("name:"+keyName))

	serviceKeys, err := c.listServiceKeys(path)
	if err != nil || len(serviceKeys) == 0 {
		return models.ServiceKey{}, err
	}

	return serviceKeys[0], nil
}

func (c CloudControllerServiceKeyRepository) listServiceKeys(path string) ([]models.ServiceKey, error) {
	serviceKeys := []models.ServiceKey{}
	err := c.gateway.ListPaginatedResources(
		c.config.APIEndpoint(),
		path,
		resources.ServiceKeyResource{},
		func(resource interface{}) bool {
			serviceKey := resource.(resources.ServiceKeyResource).ToModel()
			serviceKeys = append(serviceKeys, serviceKey)
			return true
		})

	if err != nil {
		if httpErr, ok := err.(errors.HTTPError); ok && httpErr.ErrorCode() == errors.NotAuthorized {
			return []models.ServiceKey{}, errors.NewNotAuthorizedError()
		}
		return []models.ServiceKey{}, err
	}

	return serviceKeys, nil
}

func (c CloudControllerServiceKeyRepository) DeleteServiceKey(serviceKeyGUID string) error {
	path := fmt.Sprintf("/v2/service_keys/%s", serviceKeyGUID)
	return c.gateway.DeleteResource(c.config.APIEndpoint(), path)
}
