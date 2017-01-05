package api

import (
	"fmt"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

type ServiceInstancesSummaries struct {
	Apps             []ServiceInstanceSummaryApp
	ServiceInstances []ServiceInstanceSummary `json:"services"`
}

func (resource ServiceInstancesSummaries) ToModels() []models.ServiceInstance {
	var instances []models.ServiceInstance
	for _, instanceSummary := range resource.ServiceInstances {
		applicationNames := resource.findApplicationNamesForInstance(instanceSummary.Name)

		planSummary := instanceSummary.ServicePlan
		servicePlan := models.ServicePlanFields{}
		servicePlan.Name = planSummary.Name
		servicePlan.GUID = planSummary.GUID

		offeringSummary := planSummary.ServiceOffering
		serviceOffering := models.ServiceOfferingFields{}
		serviceOffering.Label = offeringSummary.Label
		serviceOffering.Provider = offeringSummary.Provider
		serviceOffering.Version = offeringSummary.Version

		instance := models.ServiceInstance{}
		instance.Name = instanceSummary.Name
		instance.LastOperation.Type = instanceSummary.LastOperation.Type
		instance.LastOperation.State = instanceSummary.LastOperation.State
		instance.LastOperation.Description = instanceSummary.LastOperation.Description
		instance.ApplicationNames = applicationNames
		instance.ServicePlan = servicePlan
		instance.ServiceOffering = serviceOffering

		instances = append(instances, instance)
	}

	return instances
}

func (resource ServiceInstancesSummaries) findApplicationNamesForInstance(instanceName string) []string {
	var applicationNames []string
	for _, app := range resource.Apps {
		for _, name := range app.ServiceNames {
			if name == instanceName {
				applicationNames = append(applicationNames, app.Name)
			}
		}
	}

	return applicationNames
}

type ServiceInstanceSummaryApp struct {
	Name         string
	ServiceNames []string `json:"service_names"`
}

type LastOperationSummary struct {
	Type        string `json:"type"`
	State       string `json:"state"`
	Description string `json:"description"`
}

type ServiceInstanceSummary struct {
	Name          string
	LastOperation LastOperationSummary `json:"last_operation"`
	ServicePlan   ServicePlanSummary   `json:"service_plan"`
}

type ServicePlanSummary struct {
	Name            string
	GUID            string
	ServiceOffering ServiceOfferingSummary `json:"service"`
}

type ServiceOfferingSummary struct {
	Label    string
	Provider string
	Version  string
}

//go:generate counterfeiter . ServiceSummaryRepository

type ServiceSummaryRepository interface {
	GetSummariesInCurrentSpace() ([]models.ServiceInstance, error)
}

type CloudControllerServiceSummaryRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerServiceSummaryRepository(config coreconfig.Reader, gateway net.Gateway) CloudControllerServiceSummaryRepository {
	return CloudControllerServiceSummaryRepository{
		config:  config,
		gateway: gateway,
	}
}

func (repo CloudControllerServiceSummaryRepository) GetSummariesInCurrentSpace() ([]models.ServiceInstance, error) {
	var instances []models.ServiceInstance
	path := fmt.Sprintf("%s/v2/spaces/%s/summary", repo.config.APIEndpoint(), repo.config.SpaceFields().GUID)
	resource := new(ServiceInstancesSummaries)

	err := repo.gateway.GetResource(path, resource)
	if err != nil {
		return nil, err
	}

	instances = resource.ToModels()

	return instances, nil
}
