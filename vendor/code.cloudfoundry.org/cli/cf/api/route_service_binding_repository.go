package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . RouteServiceBindingRepository

type RouteServiceBindingRepository interface {
	Bind(instanceGUID, routeGUID string, userProvided bool, parameters string) error
	Unbind(instanceGUID, routeGUID string, userProvided bool) error
}

type CloudControllerRouteServiceBindingRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerRouteServiceBindingRepository(config coreconfig.Reader, gateway net.Gateway) CloudControllerRouteServiceBindingRepository {
	return CloudControllerRouteServiceBindingRepository{
		config:  config,
		gateway: gateway,
	}
}

func (repo CloudControllerRouteServiceBindingRepository) Bind(
	instanceGUID string,
	routeGUID string,
	userProvided bool,
	opaqueParams string,
) error {
	var rs io.ReadSeeker
	if opaqueParams != "" {
		opaqueJSON := json.RawMessage(opaqueParams)
		s := struct {
			Parameters *json.RawMessage `json:"parameters"`
		}{
			&opaqueJSON,
		}

		jsonBytes, err := json.Marshal(s)
		if err != nil {
			return err
		}

		rs = bytes.NewReader(jsonBytes)
	} else {
		rs = strings.NewReader("")
	}

	return repo.gateway.UpdateResourceSync(
		repo.config.APIEndpoint(),
		getPath(instanceGUID, routeGUID, userProvided),
		rs,
	)
}

func (repo CloudControllerRouteServiceBindingRepository) Unbind(instanceGUID, routeGUID string, userProvided bool) error {
	path := getPath(instanceGUID, routeGUID, userProvided)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}

func getPath(instanceGUID, routeGUID string, userProvided bool) string {
	var resource string
	if userProvided {
		resource = "user_provided_service_instances"
	} else {
		resource = "service_instances"
	}

	return fmt.Sprintf("/v2/%s/%s/routes/%s", resource, instanceGUID, routeGUID)
}
