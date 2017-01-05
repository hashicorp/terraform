package api

import (
	"bytes"
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

//go:generate counterfeiter . ServiceBrokerRepository

type ServiceBrokerRepository interface {
	ListServiceBrokers(callback func(models.ServiceBroker) bool) error
	FindByName(name string) (serviceBroker models.ServiceBroker, apiErr error)
	FindByGUID(guid string) (serviceBroker models.ServiceBroker, apiErr error)
	Create(name, url, username, password, spaceGUID string) (apiErr error)
	Update(serviceBroker models.ServiceBroker) (apiErr error)
	Rename(guid, name string) (apiErr error)
	Delete(guid string) (apiErr error)
}

type CloudControllerServiceBrokerRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerServiceBrokerRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerServiceBrokerRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerServiceBrokerRepository) ListServiceBrokers(callback func(models.ServiceBroker) bool) error {
	return repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		"/v2/service_brokers",
		resources.ServiceBrokerResource{},
		func(resource interface{}) bool {
			callback(resource.(resources.ServiceBrokerResource).ToFields())
			return true
		})
}

func (repo CloudControllerServiceBrokerRepository) FindByName(name string) (serviceBroker models.ServiceBroker, apiErr error) {
	foundBroker := false
	apiErr = repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/service_brokers?q=%s", url.QueryEscape("name:"+name)),
		resources.ServiceBrokerResource{},
		func(resource interface{}) bool {
			serviceBroker = resource.(resources.ServiceBrokerResource).ToFields()
			foundBroker = true
			return false
		})

	if !foundBroker && (apiErr == nil) {
		apiErr = errors.NewModelNotFoundError("Service Broker", name)
	}

	return
}
func (repo CloudControllerServiceBrokerRepository) FindByGUID(guid string) (serviceBroker models.ServiceBroker, apiErr error) {
	broker := new(resources.ServiceBrokerResource)
	apiErr = repo.gateway.GetResource(repo.config.APIEndpoint()+fmt.Sprintf("/v2/service_brokers/%s", guid), broker)
	serviceBroker = broker.ToFields()
	return
}

func (repo CloudControllerServiceBrokerRepository) Create(name, url, username, password, spaceGUID string) error {
	path := "/v2/service_brokers"
	args := struct {
		Name      string `json:"name"`
		URL       string `json:"broker_url"`
		Username  string `json:"auth_username"`
		Password  string `json:"auth_password"`
		SpaceGUID string `json:"space_guid,omitempty"`
	}{
		name,
		url,
		username,
		password,
		spaceGUID,
	}
	bs, err := json.Marshal(args)
	if err != nil {
		return err
	}
	return repo.gateway.CreateResource(repo.config.APIEndpoint(), path, bytes.NewReader(bs))
}

func (repo CloudControllerServiceBrokerRepository) Update(serviceBroker models.ServiceBroker) (apiErr error) {
	path := fmt.Sprintf("/v2/service_brokers/%s", serviceBroker.GUID)
	body := fmt.Sprintf(
		`{"broker_url":"%s","auth_username":"%s","auth_password":"%s"}`,
		serviceBroker.URL, serviceBroker.Username, serviceBroker.Password,
	)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, strings.NewReader(body))
}

func (repo CloudControllerServiceBrokerRepository) Rename(guid, name string) (apiErr error) {
	path := fmt.Sprintf("/v2/service_brokers/%s", guid)
	body := fmt.Sprintf(`{"name":"%s"}`, name)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, strings.NewReader(body))
}

func (repo CloudControllerServiceBrokerRepository) Delete(guid string) (apiErr error) {
	path := fmt.Sprintf("/v2/service_brokers/%s", guid)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}
