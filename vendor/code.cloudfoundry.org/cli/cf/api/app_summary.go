package api

import (
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

type ApplicationSummaries struct {
	Apps []ApplicationFromSummary
}

func (resource ApplicationSummaries) ToModels() (apps []models.ApplicationFields) {
	for _, application := range resource.Apps {
		apps = append(apps, application.ToFields())
	}
	return
}

type ApplicationFromSummary struct {
	GUID                 string
	Name                 string
	Routes               []RouteSummary
	Services             []ServicePlanSummary
	Diego                bool `json:"diego,omitempty"`
	RunningInstances     int  `json:"running_instances"`
	Memory               int64
	Instances            int
	DiskQuota            int64 `json:"disk_quota"`
	AppPorts             []int `json:"ports"`
	URLs                 []string
	EnvironmentVars      map[string]interface{} `json:"environment_json,omitempty"`
	HealthCheckTimeout   int                    `json:"health_check_timeout"`
	State                string
	DetectedStartCommand string     `json:"detected_start_command"`
	SpaceGUID            string     `json:"space_guid"`
	StackGUID            string     `json:"stack_guid"`
	Command              string     `json:"command"`
	PackageState         string     `json:"package_state"`
	PackageUpdatedAt     *time.Time `json:"package_updated_at"`
	Buildpack            string
}

func (resource ApplicationFromSummary) ToFields() (app models.ApplicationFields) {
	app = models.ApplicationFields{}
	app.GUID = resource.GUID
	app.Name = resource.Name
	app.Diego = resource.Diego
	app.State = strings.ToLower(resource.State)
	app.InstanceCount = resource.Instances
	app.DiskQuota = resource.DiskQuota
	app.RunningInstances = resource.RunningInstances
	app.Memory = resource.Memory
	app.SpaceGUID = resource.SpaceGUID
	app.StackGUID = resource.StackGUID
	app.PackageUpdatedAt = resource.PackageUpdatedAt
	app.PackageState = resource.PackageState
	app.DetectedStartCommand = resource.DetectedStartCommand
	app.HealthCheckTimeout = resource.HealthCheckTimeout
	app.BuildpackURL = resource.Buildpack
	app.Command = resource.Command
	app.AppPorts = resource.AppPorts
	app.EnvironmentVars = resource.EnvironmentVars

	return
}

func (resource ApplicationFromSummary) ToModel() models.Application {
	var app models.Application

	app.ApplicationFields = resource.ToFields()

	routes := []models.RouteSummary{}
	for _, route := range resource.Routes {
		routes = append(routes, route.ToModel())
	}
	app.Routes = routes

	services := []models.ServicePlanSummary{}
	for _, service := range resource.Services {
		services = append(services, service.ToModel())
	}

	app.Routes = routes
	app.Services = services

	return app
}

type RouteSummary struct {
	GUID   string
	Host   string
	Path   string
	Port   int
	Domain DomainSummary
}

func (resource RouteSummary) ToModel() (route models.RouteSummary) {
	domain := models.DomainFields{}
	domain.GUID = resource.Domain.GUID
	domain.Name = resource.Domain.Name
	domain.Shared = resource.Domain.OwningOrganizationGUID != ""

	route.GUID = resource.GUID
	route.Host = resource.Host
	route.Path = resource.Path
	route.Port = resource.Port
	route.Domain = domain
	return
}

func (resource ServicePlanSummary) ToModel() (route models.ServicePlanSummary) {
	route.GUID = resource.GUID
	route.Name = resource.Name
	return
}

type DomainSummary struct {
	GUID                   string
	Name                   string
	OwningOrganizationGUID string
}

//go:generate counterfeiter . AppSummaryRepository

type AppSummaryRepository interface {
	GetSummariesInCurrentSpace() (apps []models.Application, apiErr error)
	GetSummary(appGUID string) (summary models.Application, apiErr error)
}

type CloudControllerAppSummaryRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerAppSummaryRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerAppSummaryRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerAppSummaryRepository) GetSummariesInCurrentSpace() ([]models.Application, error) {
	resources := new(ApplicationSummaries)

	path := fmt.Sprintf("%s/v2/spaces/%s/summary", repo.config.APIEndpoint(), repo.config.SpaceFields().GUID)
	err := repo.gateway.GetResource(path, resources)
	if err != nil {
		return []models.Application{}, err
	}

	apps := make([]models.Application, len(resources.Apps))
	for i, resource := range resources.Apps {
		apps[i] = resource.ToModel()
	}

	return apps, nil
}

func (repo CloudControllerAppSummaryRepository) GetSummary(appGUID string) (summary models.Application, apiErr error) {
	path := fmt.Sprintf("%s/v2/apps/%s/summary", repo.config.APIEndpoint(), appGUID)
	summaryResponse := new(ApplicationFromSummary)
	apiErr = repo.gateway.GetResource(path, summaryResponse)
	if apiErr != nil {
		return
	}

	summary = summaryResponse.ToModel()

	return
}
