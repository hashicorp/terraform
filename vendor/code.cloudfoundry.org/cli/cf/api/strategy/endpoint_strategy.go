package strategy

import "github.com/blang/semver"

type EndpointStrategy struct {
	EventsEndpointStrategy
	DomainsEndpointStrategy
}

func NewEndpointStrategy(versionString string) EndpointStrategy {
	version, err := semver.Make(versionString)
	if err != nil {
		version, _ = semver.Make("0.0.0")
	}

	strategy := EndpointStrategy{
		EventsEndpointStrategy:  eventsEndpointStrategy{},
		DomainsEndpointStrategy: domainsEndpointStrategy{},
	}

	v210, _ := semver.Make("2.1.0")
	if version.GTE(v210) {
		strategy.EventsEndpointStrategy = globalEventsEndpointStrategy{}
		strategy.DomainsEndpointStrategy = separatedDomainsEndpointStrategy{}
	}

	return strategy
}
