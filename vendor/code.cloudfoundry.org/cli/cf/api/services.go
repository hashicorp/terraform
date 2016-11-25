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

//go:generate counterfeiter . ServiceRepository

type ServiceRepository interface {
	PurgeServiceOffering(offering models.ServiceOffering) error
	GetServiceOfferingByGUID(serviceGUID string) (offering models.ServiceOffering, apiErr error)
	FindServiceOfferingsByLabel(name string) (offering models.ServiceOfferings, apiErr error)
	FindServiceOfferingByLabelAndProvider(name, provider string) (offering models.ServiceOffering, apiErr error)

	FindServiceOfferingsForSpaceByLabel(spaceGUID, name string) (offering models.ServiceOfferings, apiErr error)

	GetAllServiceOfferings() (offerings models.ServiceOfferings, apiErr error)
	GetServiceOfferingsForSpace(spaceGUID string) (offerings models.ServiceOfferings, apiErr error)
	FindInstanceByName(name string) (instance models.ServiceInstance, apiErr error)
	PurgeServiceInstance(instance models.ServiceInstance) error
	CreateServiceInstance(name, planGUID string, params map[string]interface{}, tags []string) (apiErr error)
	UpdateServiceInstance(instanceGUID, planGUID string, params map[string]interface{}, tags []string) (apiErr error)
	RenameService(instance models.ServiceInstance, newName string) (apiErr error)
	DeleteService(instance models.ServiceInstance) (apiErr error)
	FindServicePlanByDescription(planDescription resources.ServicePlanDescription) (planGUID string, apiErr error)
	ListServicesFromBroker(brokerGUID string) (services []models.ServiceOffering, err error)
	ListServicesFromManyBrokers(brokerGUIDs []string) (services []models.ServiceOffering, err error)
	GetServiceInstanceCountForServicePlan(v1PlanGUID string) (count int, apiErr error)
	MigrateServicePlanFromV1ToV2(v1PlanGUID, v2PlanGUID string) (changedCount int, apiErr error)
}

type CloudControllerServiceRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerServiceRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerServiceRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerServiceRepository) GetServiceOfferingByGUID(serviceGUID string) (models.ServiceOffering, error) {
	offering := new(resources.ServiceOfferingResource)
	apiErr := repo.gateway.GetResource(repo.config.APIEndpoint()+fmt.Sprintf("/v2/services/%s", serviceGUID), offering)
	serviceOffering := offering.ToFields()
	return models.ServiceOffering{ServiceOfferingFields: serviceOffering}, apiErr
}

func (repo CloudControllerServiceRepository) GetServiceOfferingsForSpace(spaceGUID string) (models.ServiceOfferings, error) {
	return repo.getServiceOfferings(fmt.Sprintf("/v2/spaces/%s/services", spaceGUID))
}

func (repo CloudControllerServiceRepository) FindServiceOfferingsForSpaceByLabel(spaceGUID, name string) (offerings models.ServiceOfferings, err error) {
	offerings, err = repo.getServiceOfferings(fmt.Sprintf("/v2/spaces/%s/services?q=%s", spaceGUID, url.QueryEscape("label:"+name)))

	if httpErr, ok := err.(errors.HTTPError); ok && httpErr.ErrorCode() == errors.BadQueryParameter {
		offerings, err = repo.findServiceOfferingsByPaginating(spaceGUID, name)
	}

	if err == nil && len(offerings) == 0 {
		err = errors.NewModelNotFoundError("Service offering", name)
	}

	return
}

func (repo CloudControllerServiceRepository) findServiceOfferingsByPaginating(spaceGUID, label string) (offerings models.ServiceOfferings, apiErr error) {
	offerings, apiErr = repo.GetServiceOfferingsForSpace(spaceGUID)
	if apiErr != nil {
		return
	}

	matchingOffering := models.ServiceOfferings{}

	for _, offering := range offerings {
		if offering.Label == label {
			matchingOffering = append(matchingOffering, offering)
		}
	}
	return matchingOffering, nil
}

func (repo CloudControllerServiceRepository) GetAllServiceOfferings() (models.ServiceOfferings, error) {
	return repo.getServiceOfferings("/v2/services")
}

func (repo CloudControllerServiceRepository) getServiceOfferings(path string) ([]models.ServiceOffering, error) {
	var offerings []models.ServiceOffering
	apiErr := repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		path,
		resources.ServiceOfferingResource{},
		func(resource interface{}) bool {
			if so, ok := resource.(resources.ServiceOfferingResource); ok {
				offerings = append(offerings, so.ToModel())
			}
			return true
		})

	return offerings, apiErr
}

func (repo CloudControllerServiceRepository) FindInstanceByName(name string) (instance models.ServiceInstance, apiErr error) {
	path := fmt.Sprintf("%s/v2/spaces/%s/service_instances?return_user_provided_service_instances=true&q=%s&inline-relations-depth=1", repo.config.APIEndpoint(), repo.config.SpaceFields().GUID, url.QueryEscape("name:"+name))

	responseJSON := new(resources.PaginatedServiceInstanceResources)
	apiErr = repo.gateway.GetResource(path, responseJSON)
	if apiErr != nil {
		return
	}

	if len(responseJSON.Resources) == 0 {
		apiErr = errors.NewModelNotFoundError("Service instance", name)
		return
	}

	instanceResource := responseJSON.Resources[0]
	instance = instanceResource.ToModel()

	if instanceResource.Entity.ServicePlan.Metadata.GUID != "" {
		resource := &resources.ServiceOfferingResource{}
		path = fmt.Sprintf("%s/v2/services/%s", repo.config.APIEndpoint(), instanceResource.Entity.ServicePlan.Entity.ServiceOfferingGUID)
		apiErr = repo.gateway.GetResource(path, resource)
		instance.ServiceOffering = resource.ToFields()
	}

	return
}

func (repo CloudControllerServiceRepository) CreateServiceInstance(name, planGUID string, params map[string]interface{}, tags []string) (err error) {
	path := "/v2/service_instances?accepts_incomplete=true"
	request := models.ServiceInstanceCreateRequest{
		Name:      name,
		PlanGUID:  planGUID,
		SpaceGUID: repo.config.SpaceFields().GUID,
		Params:    params,
		Tags:      tags,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}

	err = repo.gateway.CreateResource(repo.config.APIEndpoint(), path, bytes.NewReader(jsonBytes))

	if httpErr, ok := err.(errors.HTTPError); ok && httpErr.ErrorCode() == errors.ServiceInstanceNameTaken {
		serviceInstance, findInstanceErr := repo.FindInstanceByName(name)

		if findInstanceErr == nil && serviceInstance.ServicePlan.GUID == planGUID {
			return errors.NewModelAlreadyExistsError("Service", name)
		}
	}

	return
}

func (repo CloudControllerServiceRepository) UpdateServiceInstance(instanceGUID, planGUID string, params map[string]interface{}, tags []string) (err error) {
	path := fmt.Sprintf("/v2/service_instances/%s?accepts_incomplete=true", instanceGUID)
	request := models.ServiceInstanceUpdateRequest{
		PlanGUID: planGUID,
		Params:   params,
		Tags:     tags,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}

	err = repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, bytes.NewReader(jsonBytes))

	return
}

func (repo CloudControllerServiceRepository) RenameService(instance models.ServiceInstance, newName string) (apiErr error) {
	body := fmt.Sprintf(`{"name":"%s"}`, newName)
	path := fmt.Sprintf("/v2/service_instances/%s?accepts_incomplete=true", instance.GUID)

	if instance.IsUserProvided() {
		path = fmt.Sprintf("/v2/user_provided_service_instances/%s", instance.GUID)
	}
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, strings.NewReader(body))
}

func (repo CloudControllerServiceRepository) DeleteService(instance models.ServiceInstance) (apiErr error) {
	if len(instance.ServiceBindings) > 0 || len(instance.ServiceKeys) > 0 {
		return errors.NewServiceAssociationError()
	}
	path := fmt.Sprintf("/v2/service_instances/%s?%s", instance.GUID, "accepts_incomplete=true")
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}

func (repo CloudControllerServiceRepository) PurgeServiceOffering(offering models.ServiceOffering) error {
	url := fmt.Sprintf("/v2/services/%s?purge=true", offering.GUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), url)
}

func (repo CloudControllerServiceRepository) PurgeServiceInstance(instance models.ServiceInstance) error {
	url := fmt.Sprintf("/v2/service_instances/%s?purge=true", instance.GUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), url)
}

func (repo CloudControllerServiceRepository) FindServiceOfferingsByLabel(label string) (models.ServiceOfferings, error) {
	path := fmt.Sprintf("/v2/services?q=%s", url.QueryEscape("label:"+label))
	offerings, apiErr := repo.getServiceOfferings(path)

	if apiErr != nil {
		return models.ServiceOfferings{}, apiErr
	} else if len(offerings) == 0 {
		apiErr = errors.NewModelNotFoundError("Service offering", label)
		return models.ServiceOfferings{}, apiErr
	}

	return offerings, apiErr
}

func (repo CloudControllerServiceRepository) FindServiceOfferingByLabelAndProvider(label, provider string) (models.ServiceOffering, error) {
	path := fmt.Sprintf("/v2/services?q=%s", url.QueryEscape("label:"+label+";provider:"+provider))
	offerings, apiErr := repo.getServiceOfferings(path)

	if apiErr != nil {
		return models.ServiceOffering{}, apiErr
	} else if len(offerings) == 0 {
		apiErr = errors.NewModelNotFoundError("Service offering", label+" "+provider)
		return models.ServiceOffering{}, apiErr
	}

	return offerings[0], apiErr
}

func (repo CloudControllerServiceRepository) FindServicePlanByDescription(planDescription resources.ServicePlanDescription) (string, error) {
	path := fmt.Sprintf("/v2/services?inline-relations-depth=1&q=%s",
		url.QueryEscape("label:"+planDescription.ServiceLabel+";provider:"+planDescription.ServiceProvider))

	offerings, err := repo.getServiceOfferings(path)
	if err != nil {
		return "", err
	}

	for _, serviceOfferingResource := range offerings {
		for _, servicePlanResource := range serviceOfferingResource.Plans {
			if servicePlanResource.Name == planDescription.ServicePlanName {
				return servicePlanResource.GUID, nil
			}
		}
	}

	return "", errors.NewModelNotFoundError("Plan", planDescription.String())
}

func (repo CloudControllerServiceRepository) ListServicesFromManyBrokers(brokerGUIDs []string) ([]models.ServiceOffering, error) {
	brokerGUIDsString := strings.Join(brokerGUIDs, ",")
	services := []models.ServiceOffering{}

	err := repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/services?q=%s", url.QueryEscape("service_broker_guid IN "+brokerGUIDsString)),
		resources.ServiceOfferingResource{},
		func(resource interface{}) bool {
			if service, ok := resource.(resources.ServiceOfferingResource); ok {
				services = append(services, service.ToModel())
			}
			return true
		})
	return services, err
}

func (repo CloudControllerServiceRepository) ListServicesFromBroker(brokerGUID string) (offerings []models.ServiceOffering, err error) {
	err = repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/services?q=%s", url.QueryEscape("service_broker_guid:"+brokerGUID)),
		resources.ServiceOfferingResource{},
		func(resource interface{}) bool {
			if offering, ok := resource.(resources.ServiceOfferingResource); ok {
				offerings = append(offerings, offering.ToModel())
			}
			return true
		})
	return
}

func (repo CloudControllerServiceRepository) MigrateServicePlanFromV1ToV2(v1PlanGUID, v2PlanGUID string) (changedCount int, apiErr error) {
	path := fmt.Sprintf("/v2/service_plans/%s/service_instances", v1PlanGUID)
	body := strings.NewReader(fmt.Sprintf(`{"service_plan_guid":"%s"}`, v2PlanGUID))
	response := new(resources.ServiceMigrateV1ToV2Response)

	apiErr = repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, body, response)
	if apiErr != nil {
		return
	}

	changedCount = response.ChangedCount
	return
}

func (repo CloudControllerServiceRepository) GetServiceInstanceCountForServicePlan(v1PlanGUID string) (count int, apiErr error) {
	path := fmt.Sprintf("%s/v2/service_plans/%s/service_instances?results-per-page=1", repo.config.APIEndpoint(), v1PlanGUID)
	response := new(resources.PaginatedServiceInstanceResources)
	apiErr = repo.gateway.GetResource(path, response)
	count = response.TotalResults
	return
}
