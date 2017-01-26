package cfapi

import (
	"fmt"

	"code.cloudfoundry.org/cli/cf/api"
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
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
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// CCServicePlan -
type CCServicePlan struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"label,omitempty"`
}

// CCService -
type CCService struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"label"`
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
func (sm *ServiceManager) CreateServiceInstance(name string, servicePlanID string, spaceID string) (serviceInstance CCServiceInstance, err error) {
	/*
		payload := map[string]interface{}{"name": name, "service_plan_guid": servicePlanID, "space_guid": spaceID}

		body, err := json.Marshal(payload)
		if err != nil {
			return
		}

		resource := CCServiceInstanceResource{}
		if err = sm.ccGateway.CreateResource(sm.apiEndpoint, "/v2/service_instances", bytes.NewReader(body), &resource); err != nil {
			return
		}

		serviceInstance = resource.Entity
		serviceInstance.ID = resource.Metadata.GUID
	*/
	return
}

// ReadServiceInstance -
func (sm *ServiceManager) ReadServiceInstance(serviceInstanceID string) (serviceInstance CCServiceInstance, err error) {
	/*
		resource := &CCServiceInstanceResource{}
		err = sm.ccGateway.GetResource(
			fmt.Sprintf("%s/v2/service_instances/%s", sm.apiEndpoint, serviceInstanceID), &resource)

		serviceInstance = resource.Entity
		serviceInstance.ID = resource.Metadata.GUID
	*/
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
