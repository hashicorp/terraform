package resources

import (
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf/models"
)

type PaginatedApplicationResources struct {
	Resources []ApplicationResource
}

type AppRouteEntity struct {
	Host   string
	Domain struct {
		Resource
		Entity struct {
			Name string
		}
	}
}

type AppRouteResource struct {
	Resource
	Entity AppRouteEntity
}

type IntegrityFields struct {
	Sha1 string `json:"sha1"`
	Size int64  `json:"size"`
}

type AppFileResource struct {
	Sha1 string `json:"sha1"`
	Size int64  `json:"size"`
	Path string `json:"fn"`
	Mode string `json:"mode"`
}

type ApplicationResource struct {
	Resource
	Entity ApplicationEntity
}

type ApplicationEntity struct {
	Name                 *string                 `json:"name,omitempty"`
	Command              *string                 `json:"command,omitempty"`
	DetectedStartCommand *string                 `json:"detected_start_command,omitempty"`
	State                *string                 `json:"state,omitempty"`
	SpaceGUID            *string                 `json:"space_guid,omitempty"`
	Instances            *int                    `json:"instances,omitempty"`
	Memory               *int64                  `json:"memory,omitempty"`
	DiskQuota            *int64                  `json:"disk_quota,omitempty"`
	StackGUID            *string                 `json:"stack_guid,omitempty"`
	Stack                *StackResource          `json:"stack,omitempty"`
	Routes               *[]AppRouteResource     `json:"routes,omitempty"`
	Buildpack            *string                 `json:"buildpack,omitempty"`
	DetectedBuildpack    *string                 `json:"detected_buildpack,omitempty"`
	EnvironmentJSON      *map[string]interface{} `json:"environment_json,omitempty"`
	HealthCheckType      *string                 `json:"health_check_type,omitempty"`
	HealthCheckTimeout   *int                    `json:"health_check_timeout,omitempty"`
	PackageState         *string                 `json:"package_state,omitempty"`
	StagingFailedReason  *string                 `json:"staging_failed_reason,omitempty"`
	Diego                *bool                   `json:"diego,omitempty"`
	DockerImage          *string                 `json:"docker_image,omitempty"`
	EnableSSH            *bool                   `json:"enable_ssh,omitempty"`
	PackageUpdatedAt     *time.Time              `json:"package_updated_at,omitempty"`
	AppPorts             *[]int                  `json:"ports,omitempty"`
}

func (resource AppRouteResource) ToFields() (route models.RouteSummary) {
	route.GUID = resource.Metadata.GUID
	route.Host = resource.Entity.Host
	return
}

func (resource AppRouteResource) ToModel() (route models.RouteSummary) {
	route.GUID = resource.Metadata.GUID
	route.Host = resource.Entity.Host
	route.Domain.GUID = resource.Entity.Domain.Metadata.GUID
	route.Domain.Name = resource.Entity.Domain.Entity.Name
	return
}

func (resource AppFileResource) ToIntegrityFields() IntegrityFields {
	return IntegrityFields{
		Sha1: resource.Sha1,
		Size: resource.Size,
	}
}

func NewApplicationEntityFromAppParams(app models.AppParams) ApplicationEntity {
	entity := ApplicationEntity{
		Buildpack:          app.BuildpackURL,
		Name:               app.Name,
		SpaceGUID:          app.SpaceGUID,
		Instances:          app.InstanceCount,
		Memory:             app.Memory,
		DiskQuota:          app.DiskQuota,
		StackGUID:          app.StackGUID,
		Command:            app.Command,
		HealthCheckType:    app.HealthCheckType,
		HealthCheckTimeout: app.HealthCheckTimeout,
		DockerImage:        app.DockerImage,
		Diego:              app.Diego,
		EnableSSH:          app.EnableSSH,
		PackageUpdatedAt:   app.PackageUpdatedAt,
		AppPorts:           app.AppPorts,
	}

	if app.State != nil {
		state := strings.ToUpper(*app.State)
		entity.State = &state
	}

	if app.EnvironmentVars != nil && *app.EnvironmentVars != nil {
		entity.EnvironmentJSON = app.EnvironmentVars
	}

	return entity
}

func (resource ApplicationResource) ToFields() (app models.ApplicationFields) {
	entity := resource.Entity
	app.GUID = resource.Metadata.GUID

	if entity.Name != nil {
		app.Name = *entity.Name
	}
	if entity.Memory != nil {
		app.Memory = *entity.Memory
	}
	if entity.DiskQuota != nil {
		app.DiskQuota = *entity.DiskQuota
	}
	if entity.Instances != nil {
		app.InstanceCount = *entity.Instances
	}
	if entity.State != nil {
		app.State = strings.ToLower(*entity.State)
	}
	if entity.EnvironmentJSON != nil {
		app.EnvironmentVars = *entity.EnvironmentJSON
	}
	if entity.SpaceGUID != nil {
		app.SpaceGUID = *entity.SpaceGUID
	}
	if entity.DetectedStartCommand != nil {
		app.DetectedStartCommand = *entity.DetectedStartCommand
	}
	if entity.Command != nil {
		app.Command = *entity.Command
	}
	if entity.PackageState != nil {
		app.PackageState = *entity.PackageState
	}
	if entity.StagingFailedReason != nil {
		app.StagingFailedReason = *entity.StagingFailedReason
	}
	if entity.DockerImage != nil {
		app.DockerImage = *entity.DockerImage
	}
	if entity.Buildpack != nil {
		app.Buildpack = *entity.Buildpack
	}
	if entity.DetectedBuildpack != nil {
		app.DetectedBuildpack = *entity.DetectedBuildpack
	}
	if entity.HealthCheckType != nil {
		app.HealthCheckType = *entity.HealthCheckType
	}
	if entity.Diego != nil {
		app.Diego = *entity.Diego
	}
	if entity.EnableSSH != nil {
		app.EnableSSH = *entity.EnableSSH
	}
	if entity.PackageUpdatedAt != nil {
		app.PackageUpdatedAt = entity.PackageUpdatedAt
	}

	return
}

func (resource ApplicationResource) ToModel() (app models.Application) {
	app.ApplicationFields = resource.ToFields()

	entity := resource.Entity
	if entity.Stack != nil {
		app.Stack = entity.Stack.ToFields()
	}

	if entity.Routes != nil {
		for _, routeResource := range *entity.Routes {
			app.Routes = append(app.Routes, routeResource.ToModel())
		}
	}

	return
}
