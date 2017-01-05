package models

import (
	"reflect"
	"strings"
	"time"
)

type Application struct {
	ApplicationFields
	Stack    *Stack
	Routes   []RouteSummary
	Services []ServicePlanSummary
}

func (model Application) HasRoute(route Route) bool {
	for _, boundRoute := range model.Routes {
		if boundRoute.GUID == route.GUID {
			return true
		}
	}
	return false
}

func (model Application) ToParams() AppParams {
	state := strings.ToUpper(model.State)
	params := AppParams{
		GUID:            &model.GUID,
		Name:            &model.Name,
		BuildpackURL:    &model.BuildpackURL,
		Command:         &model.Command,
		DiskQuota:       &model.DiskQuota,
		InstanceCount:   &model.InstanceCount,
		HealthCheckType: &model.HealthCheckType,
		Memory:          &model.Memory,
		State:           &state,
		SpaceGUID:       &model.SpaceGUID,
		EnvironmentVars: &model.EnvironmentVars,
		DockerImage:     &model.DockerImage,
	}

	if model.Stack != nil {
		params.StackGUID = &model.Stack.GUID
	}

	return params
}

type ApplicationFields struct {
	GUID                 string
	Name                 string
	BuildpackURL         string
	Command              string
	Diego                bool
	DetectedStartCommand string
	DiskQuota            int64 // in Megabytes
	EnvironmentVars      map[string]interface{}
	InstanceCount        int
	Memory               int64 // in Megabytes
	RunningInstances     int
	HealthCheckType      string
	HealthCheckTimeout   int
	State                string
	SpaceGUID            string
	StackGUID            string
	PackageUpdatedAt     *time.Time
	PackageState         string
	StagingFailedReason  string
	Buildpack            string
	DetectedBuildpack    string
	DockerImage          string
	EnableSSH            bool
	AppPorts             []int
}

const (
	ApplicationStateStopped  = "stopped"
	ApplicationStateStarted  = "started"
	ApplicationStateRunning  = "running"
	ApplicationStateCrashed  = "crashed"
	ApplicationStateFlapping = "flapping"
	ApplicationStateDown     = "down"
	ApplicationStateStarting = "starting"
)

type AppParams struct {
	BuildpackURL       *string
	Command            *string
	DiskQuota          *int64
	Domains            []string
	EnvironmentVars    *map[string]interface{}
	GUID               *string
	HealthCheckType    *string
	HealthCheckTimeout *int
	DockerImage        *string
	Diego              *bool
	EnableSSH          *bool
	Hosts              []string
	RoutePath          *string
	InstanceCount      *int
	Memory             *int64
	Name               *string
	NoHostname         *bool
	NoRoute            bool
	UseRandomRoute     bool
	UseRandomPort      bool
	Path               *string
	ServicesToBind     []string
	SpaceGUID          *string
	StackGUID          *string
	StackName          *string
	State              *string
	PackageUpdatedAt   *time.Time
	AppPorts           *[]int
	Routes             []ManifestRoute
}

func (app *AppParams) Merge(other *AppParams) {
	if other.AppPorts != nil {
		app.AppPorts = other.AppPorts
	}
	if other.BuildpackURL != nil {
		app.BuildpackURL = other.BuildpackURL
	}
	if other.Command != nil {
		app.Command = other.Command
	}
	if other.DiskQuota != nil {
		app.DiskQuota = other.DiskQuota
	}
	if other.DockerImage != nil {
		app.DockerImage = other.DockerImage
	}
	if other.Domains != nil {
		app.Domains = other.Domains
	}
	if other.EnableSSH != nil {
		app.EnableSSH = other.EnableSSH
	}
	if other.EnvironmentVars != nil {
		app.EnvironmentVars = other.EnvironmentVars
	}
	if other.GUID != nil {
		app.GUID = other.GUID
	}
	if other.HealthCheckType != nil {
		app.HealthCheckType = other.HealthCheckType
	}
	if other.HealthCheckTimeout != nil {
		app.HealthCheckTimeout = other.HealthCheckTimeout
	}
	if other.Hosts != nil {
		app.Hosts = other.Hosts
	}
	if other.InstanceCount != nil {
		app.InstanceCount = other.InstanceCount
	}
	if other.Memory != nil {
		app.Memory = other.Memory
	}
	if other.Name != nil {
		app.Name = other.Name
	}
	if other.Path != nil {
		app.Path = other.Path
	}
	if other.RoutePath != nil {
		app.RoutePath = other.RoutePath
	}
	if other.ServicesToBind != nil {
		app.ServicesToBind = other.ServicesToBind
	}
	if other.SpaceGUID != nil {
		app.SpaceGUID = other.SpaceGUID
	}
	if other.StackGUID != nil {
		app.StackGUID = other.StackGUID
	}
	if other.StackName != nil {
		app.StackName = other.StackName
	}
	if other.State != nil {
		app.State = other.State
	}

	app.NoRoute = app.NoRoute || other.NoRoute
	noHostBool := app.IsNoHostnameTrue() || other.IsNoHostnameTrue()
	app.NoHostname = &noHostBool
	app.UseRandomRoute = app.UseRandomRoute || other.UseRandomRoute
}

func (app *AppParams) IsEmpty() bool {
	noHostBool := false
	return reflect.DeepEqual(*app, AppParams{NoHostname: &noHostBool})
}

func (app *AppParams) IsHostEmpty() bool {
	return app.Hosts == nil || len(app.Hosts) == 0
}

func (app *AppParams) IsNoHostnameTrue() bool {
	if app.NoHostname == nil {
		return false
	}
	return *app.NoHostname
}
