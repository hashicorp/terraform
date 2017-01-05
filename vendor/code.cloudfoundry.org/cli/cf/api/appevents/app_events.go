package appevents

import (
	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/api/strategy"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . Repository

type Repository interface {
	RecentEvents(appGUID string, limit int64) ([]models.EventFields, error)
}

type CloudControllerAppEventsRepository struct {
	config   coreconfig.Reader
	gateway  net.Gateway
	strategy strategy.EndpointStrategy
}

func NewCloudControllerAppEventsRepository(config coreconfig.Reader, gateway net.Gateway, strategy strategy.EndpointStrategy) CloudControllerAppEventsRepository {
	return CloudControllerAppEventsRepository{
		config:   config,
		gateway:  gateway,
		strategy: strategy,
	}
}

func (repo CloudControllerAppEventsRepository) RecentEvents(appGUID string, limit int64) ([]models.EventFields, error) {
	count := int64(0)
	events := make([]models.EventFields, 0, limit)
	apiErr := repo.listEvents(appGUID, limit, func(eventField models.EventFields) bool {
		count++
		events = append(events, eventField)
		return count < limit
	})

	return events, apiErr
}

func (repo CloudControllerAppEventsRepository) listEvents(appGUID string, limit int64, cb func(models.EventFields) bool) error {
	return repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		repo.strategy.EventsURL(appGUID, limit),
		repo.strategy.EventsResource(),

		func(resource interface{}) bool {
			return cb(resource.(resources.EventResource).ToFields())
		})
}
