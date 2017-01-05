package strategy

import "code.cloudfoundry.org/cli/cf/api/resources"

//go:generate counterfeiter . EventsEndpointStrategy

type EventsEndpointStrategy interface {
	EventsURL(appGUID string, limit int64) string
	EventsResource() resources.EventResource
}

type eventsEndpointStrategy struct{}

func (s eventsEndpointStrategy) EventsURL(appGUID string, limit int64) string {
	return buildURL(v2("apps", appGUID, "events"), params{
		resultsPerPage: limit,
	})
}

func (s eventsEndpointStrategy) EventsResource() resources.EventResource {
	return resources.EventResourceOldV2{}
}

type globalEventsEndpointStrategy struct{}

func (s globalEventsEndpointStrategy) EventsURL(appGUID string, limit int64) string {
	return buildURL(v2("events"), params{
		resultsPerPage: limit,
		orderDirection: "desc",
		q:              map[string]string{"actee": appGUID},
	})
}

func (s globalEventsEndpointStrategy) EventsResource() resources.EventResource {
	return resources.EventResourceNewV2{}
}
