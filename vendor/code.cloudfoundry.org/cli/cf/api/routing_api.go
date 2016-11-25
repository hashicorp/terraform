package api

import (
	"fmt"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

type routingAPIRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

//go:generate counterfeiter . RoutingAPIRepository

type RoutingAPIRepository interface {
	ListRouterGroups(cb func(models.RouterGroup) bool) (apiErr error)
}

func NewRoutingAPIRepository(config coreconfig.Reader, gateway net.Gateway) RoutingAPIRepository {
	return routingAPIRepository{
		config:  config,
		gateway: gateway,
	}
}

func (r routingAPIRepository) ListRouterGroups(cb func(models.RouterGroup) bool) (apiErr error) {
	routerGroups := models.RouterGroups{}
	endpoint := fmt.Sprintf("%s/v1/router_groups", r.config.RoutingAPIEndpoint())
	apiErr = r.gateway.GetResource(endpoint, &routerGroups)
	if apiErr != nil {
		return apiErr
	}

	for _, router := range routerGroups {
		if cb(router) == false {
			return
		}
	}
	return
}
