package cfapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

// ServiceManager -
type ServiceManager struct {
	config    coreconfig.Reader
	ccGateway net.Gateway

	apiEndpoint string

	repo api.ServiceRepository
}

// CCServiceInstance -
type CCServiceInstance struct {
	Name            string   `json:"name"`
	SpaceGUID       string   `json:"space_guid"`
	ServicePlanGUID string   `json:"service_plan_guid"`
	Tags            []string `json:"tags,omitempty"`
}

// CCServiceInstanceResource -
type CCServiceInstanceResource struct {
	Metadata resources.Metadata `json:"metadata"`
	Entity   CCServiceInstance  `json:"entity"`
}

// CCServiceInstanceUpdateRequest -
type CCServiceInstanceUpdateRequest struct {
	Name            string                 `json:"name"`
	ServicePlanGUID string                 `json:"service_plan_guid"`
	Params          map[string]interface{} `json:"parameters,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
}

// CCUserProvidedService -
type CCUserProvidedService struct {
	Name            string                 `json:"name"`
	SpaceGUID       string                 `json:"space_guid"`
	SyslogDrainURL  string                 `json:"syslog_drain_url,omitempty"`
	RouteServiceURL string                 `json:"route_service_url,omitempty"`
	Credentials     map[string]interface{} `json:"credentials,omitempty"`
}

// CCUserProvidedServiceResource -
type CCUserProvidedServiceResource struct {
	Metadata resources.Metadata    `json:"metadata"`
	Entity   CCUserProvidedService `json:"entity"`
}

// CCUserProvidedServiceUpdateRequest -
type CCUserProvidedServiceUpdateRequest struct {
	Name            string                 `json:"name"`
	ServicePlanGUID string                 `json:"service_plan_guid"`
	SyslogDrainURL  string                 `json:"syslog_drain_url,omitempty"`
	RouteServiceURL string                 `json:"route_service_url,omitempty"`
	Credentials     map[string]interface{} `json:"credentials,omitempty"`
}

// NewServiceManager -
func NewServiceManager(config coreconfig.Reader, ccGateway net.Gateway) (sm *ServiceManager, err error) {

	sm = &ServiceManager{
		config:      config,
		ccGateway:   ccGateway,
		apiEndpoint: config.APIEndpoint(),
		repo:        api.NewCloudControllerServiceRepository(config, ccGateway),
	}

	return
}

// CreateServiceInstance -
func (sm *ServiceManager) CreateServiceInstance(name string, servicePlanID string, spaceID string, params map[string]interface{}, tags []string) (id string, err error) {

	path := "/v2/service_instances?accepts_incomplete=true"
	request := models.ServiceInstanceCreateRequest{
		Name:      name,
		PlanGUID:  servicePlanID,
		SpaceGUID: spaceID,
		Params:    params,
		Tags:      tags,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return
	}

	resource := CCServiceInstanceResource{}
	err = sm.ccGateway.CreateResource(sm.apiEndpoint, path, bytes.NewReader(jsonBytes), &resource)

	id = resource.Metadata.GUID
	return
}

// UpdateServiceInstance -
func (sm *ServiceManager) UpdateServiceInstance(serviceInstanceID string, name string, servicePlanID string, params map[string]interface{}, tags []string) (serviceInstance CCServiceInstance, err error) {

	path := fmt.Sprintf("/v2/service_instances/%s?accepts_incomplete=true", serviceInstanceID)
	request := CCServiceInstanceUpdateRequest{
		Name:            name,
		ServicePlanGUID: servicePlanID,
		Params:          params,
		Tags:            tags,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return
	}

	resource := CCServiceInstance{}
	err = sm.ccGateway.UpdateResource(sm.apiEndpoint, path, bytes.NewReader(jsonBytes), &resource)

	return
}

// ReadServiceInstance -
func (sm *ServiceManager) ReadServiceInstance(serviceInstanceID string) (serviceInstance CCServiceInstance, err error) {

	path := fmt.Sprintf("%s/v2/service_instances/%s", sm.apiEndpoint, serviceInstanceID)
	resource := CCServiceInstanceResource{}
	err = sm.ccGateway.GetResource(path, &resource)
	if err != nil {
		return
	}

	serviceInstance = resource.Entity

	return
}

// FindServiceInstance -
func (sm *ServiceManager) FindServiceInstance(name string, spaceID string) (serviceInstance CCServiceInstance, err error) {

	path := fmt.Sprintf("/v2/spaces/%s/service_instances?return_user_provided_service_instances=true&q=%s&inline-relations-depth=1",
		spaceID, url.QueryEscape("name:"+name))

	var found bool

	apiErr := sm.ccGateway.ListPaginatedResources(
		sm.apiEndpoint,
		path,
		CCServiceInstanceResource{},
		func(resource interface{}) bool {
			if sp, ok := resource.(CCServiceInstanceResource); ok {
				serviceInstance = sp.Entity // there should 1 or 0 instances in the space with that name
				found = true
				return false
			}
			return true

		})

	if apiErr != nil {
		switch apiErr.(type) {
		case *errors.HTTPNotFoundError:
			err = errors.NewModelNotFoundError("Space", spaceID)
		default:
			err = apiErr
		}
	} else {
		if !found {
			err = errors.NewModelNotFoundError("ServiceInstance", name)
		}
	}

	return

}

// DeleteServiceInstance -
func (sm *ServiceManager) DeleteServiceInstance(serviceInstanceID string) (err error) {

	err = sm.ccGateway.DeleteResource(sm.apiEndpoint, fmt.Sprintf("/v2/service_instances/%s", serviceInstanceID))

	return

}

// CreateUserProvidedService -
func (sm *ServiceManager) CreateUserProvidedService(name string, spaceID string, credentials map[string]interface{}, syslogDrainURL string, routeServiceURL string) (id string, err error) {

	path := "/v2/user_provided_service_instances"
	request := models.UserProvidedService{
		Name:            name,
		SpaceGUID:       spaceID,
		Credentials:     credentials,
		SysLogDrainURL:  syslogDrainURL,
		RouteServiceURL: routeServiceURL,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return
	}

	ups := CCUserProvidedServiceResource{}
	err = sm.ccGateway.CreateResource(sm.apiEndpoint, path, bytes.NewReader(jsonBytes), &ups)

	id = ups.Metadata.GUID

	return
}

// ReadUserProvidedService -
func (sm *ServiceManager) ReadUserProvidedService(serviceInstanceID string) (ups CCUserProvidedService, err error) {

	path := fmt.Sprintf("%s/v2/user_provided_service_instances/%s", sm.apiEndpoint, serviceInstanceID)
	resource := CCUserProvidedServiceResource{}
	err = sm.ccGateway.GetResource(path, &resource)
	if err != nil {
		return
	}

	ups = resource.Entity

	return
}

// UpdateUserProvidedService -
func (sm *ServiceManager) UpdateUserProvidedService(serviceInstanceID string, name string, credentials map[string]interface{},
	syslogDrainURL string, routeServiceURL string) (ups CCUserProvidedService, err error) {

	path := fmt.Sprintf("/v2/user_provided_service_instances/%s", serviceInstanceID)
	request := CCUserProvidedServiceUpdateRequest{
		Name:            name,
		Credentials:     credentials,
		SyslogDrainURL:  syslogDrainURL,
		RouteServiceURL: routeServiceURL,
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return
	}

	ups = CCUserProvidedService{}
	err = sm.ccGateway.UpdateResource(sm.apiEndpoint, path, bytes.NewReader(jsonBytes), &ups)

	return
}

// DeleteUserProvidedService -
func (sm *ServiceManager) DeleteUserProvidedService(serviceInstanceID string) (err error) {

	err = sm.ccGateway.DeleteResource(sm.apiEndpoint, fmt.Sprintf("/v2/user_provided_service_instances/%s", serviceInstanceID))

	return

}

// FindServicePlanID -
func (sm *ServiceManager) FindServicePlanID(service string, plan string) (id string, err error) {

	var offeredPlans []string
	var servicePlans []models.ServicePlanFields

	err = sm.ccGateway.ListPaginatedResources(
		sm.apiEndpoint,
		fmt.Sprintf("/v2/services/%s/service_plans", service),
		resources.ServicePlanResource{},
		func(resource interface{}) bool {
			if sp, ok := resource.(resources.ServicePlanResource); ok {
				servicePlans = append(servicePlans, sp.ToFields())
			}
			return true
		})
	if err != nil {
		return
	}

	for _, v := range servicePlans {
		if v.Name == plan {
			id = v.GUID
		}
		offeredPlans = append(offeredPlans, v.Name)
	}
	if len(id) == 0 {
		err = fmt.Errorf("Plan %s does not exist in service %s (%s)", plan, service, offeredPlans)
	}

	return
}

// FindSpaceService -
func (sm *ServiceManager) FindSpaceService(label string, spaceID string) (offering models.ServiceOffering, err error) {

	var offerings models.ServiceOfferings
	var count int

	offerings, err = sm.repo.FindServiceOfferingsForSpaceByLabel(spaceID, label)
	count = len(offerings)

	switch {
	case count < 1:
		err = fmt.Errorf("Service %s not found in space %s", label, spaceID)
	case count > 1:
		err = fmt.Errorf("Too many %s Services in space %s", label, spaceID)
	}

	offering = offerings[0]

	return
}

// FindServiceByName -
func (sm *ServiceManager) FindServiceByName(label string) (offering models.ServiceOffering, err error) {

	var offerings models.ServiceOfferings
	var count int

	offerings, err = sm.repo.FindServiceOfferingsByLabel(label)
	count = len(offerings)

	switch {
	case count < 1:
		err = fmt.Errorf("Service %s not found", label)
	case count > 1:
		err = fmt.Errorf("Too many %s Services", label)
	}

	offering = offerings[0]

	return
}
