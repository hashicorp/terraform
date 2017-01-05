package api

import (
	"bytes"
	"encoding/json"
	"fmt"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . UserProvidedServiceInstanceRepository

type UserProvidedServiceInstanceRepository interface {
	Create(name, drainURL string, routeServiceURL string, params map[string]interface{}) (apiErr error)
	Update(serviceInstanceFields models.ServiceInstanceFields) (apiErr error)
	GetSummaries() (models.UserProvidedServiceSummary, error)
}

type CCUserProvidedServiceInstanceRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCCUserProvidedServiceInstanceRepository(config coreconfig.Reader, gateway net.Gateway) (repo CCUserProvidedServiceInstanceRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CCUserProvidedServiceInstanceRepository) Create(name, drainURL string, routeServiceURL string, params map[string]interface{}) (apiErr error) {
	path := "/v2/user_provided_service_instances"

	jsonBytes, err := json.Marshal(models.UserProvidedService{
		Name:            name,
		Credentials:     params,
		SpaceGUID:       repo.config.SpaceFields().GUID,
		SysLogDrainURL:  drainURL,
		RouteServiceURL: routeServiceURL,
	})

	if err != nil {
		apiErr = fmt.Errorf("%s: %s", "Error parsing response", err.Error())
		return
	}

	return repo.gateway.CreateResource(repo.config.APIEndpoint(), path, bytes.NewReader(jsonBytes))
}

func (repo CCUserProvidedServiceInstanceRepository) Update(serviceInstanceFields models.ServiceInstanceFields) (apiErr error) {
	path := fmt.Sprintf("/v2/user_provided_service_instances/%s", serviceInstanceFields.GUID)

	reqBody := models.UserProvidedService{
		Credentials:     serviceInstanceFields.Params,
		SysLogDrainURL:  serviceInstanceFields.SysLogDrainURL,
		RouteServiceURL: serviceInstanceFields.RouteServiceURL,
	}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		apiErr = fmt.Errorf("%s: %s", "Error parsing response", err.Error())
		return
	}

	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, bytes.NewReader(jsonBytes))
}

func (repo CCUserProvidedServiceInstanceRepository) GetSummaries() (models.UserProvidedServiceSummary, error) {
	path := fmt.Sprintf("%s/v2/user_provided_service_instances", repo.config.APIEndpoint())

	model := models.UserProvidedServiceSummary{}

	apiErr := repo.gateway.GetResource(path, &model)
	if apiErr != nil {
		return models.UserProvidedServiceSummary{}, apiErr
	}

	return model, nil
}
