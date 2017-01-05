package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
	"github.com/google/go-querystring/query"
)

//go:generate counterfeiter . RouteRepository

type RouteRepository interface {
	ListRoutes(cb func(models.Route) bool) (apiErr error)
	ListAllRoutes(cb func(models.Route) bool) (apiErr error)
	Find(host string, domain models.DomainFields, path string, port int) (route models.Route, apiErr error)
	Create(host string, domain models.DomainFields, path string, port int, useRandomPort bool) (createdRoute models.Route, apiErr error)
	CheckIfExists(host string, domain models.DomainFields, path string) (found bool, apiErr error)
	CreateInSpace(host, path, domainGUID, spaceGUID string, port int, randomPort bool) (createdRoute models.Route, apiErr error)
	Bind(routeGUID, appGUID string) (apiErr error)
	Unbind(routeGUID, appGUID string) (apiErr error)
	Delete(routeGUID string) (apiErr error)
}

type CloudControllerRouteRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerRouteRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerRouteRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerRouteRepository) ListRoutes(cb func(models.Route) bool) (apiErr error) {
	return repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/spaces/%s/routes?inline-relations-depth=1", repo.config.SpaceFields().GUID),
		resources.RouteResource{},
		func(resource interface{}) bool {
			return cb(resource.(resources.RouteResource).ToModel())
		})
}

func (repo CloudControllerRouteRepository) ListAllRoutes(cb func(models.Route) bool) (apiErr error) {
	return repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/routes?q=organization_guid:%s&inline-relations-depth=1", repo.config.OrganizationFields().GUID),
		resources.RouteResource{},
		func(resource interface{}) bool {
			return cb(resource.(resources.RouteResource).ToModel())
		})
}

func normalizedPath(path string) string {
	if path != "" && !strings.HasPrefix(path, `/`) {
		return `/` + path
	}

	return path
}

func queryStringForRouteSearch(host, guid, path string, port int) string {
	args := []string{
		fmt.Sprintf("host:%s", host),
		fmt.Sprintf("domain_guid:%s", guid),
	}

	if path != "" {
		args = append(args, fmt.Sprintf("path:%s", normalizedPath(path)))
	}

	if port != 0 {
		args = append(args, fmt.Sprintf("port:%d", port))
	}

	return strings.Join(args, ";")
}

func (repo CloudControllerRouteRepository) Find(host string, domain models.DomainFields, path string, port int) (models.Route, error) {
	var route models.Route
	queryString := queryStringForRouteSearch(host, domain.GUID, path, port)

	q := struct {
		Query                string `url:"q"`
		InlineRelationsDepth int    `url:"inline-relations-depth"`
	}{queryString, 1}

	opt, _ := query.Values(q)

	found := false
	apiErr := repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/routes?%s", opt.Encode()),
		resources.RouteResource{},
		func(resource interface{}) bool {
			keepSearching := true
			route = resource.(resources.RouteResource).ToModel()
			if doesNotMatchVersionSpecificAttributes(route, path, port) {
				return keepSearching
			}

			found = true
			return !keepSearching
		})

	if apiErr == nil && !found {
		apiErr = errors.NewModelNotFoundError("Route", host)
	}

	return route, apiErr
}

func doesNotMatchVersionSpecificAttributes(route models.Route, path string, port int) bool {
	return normalizedPath(route.Path) != normalizedPath(path) || route.Port != port
}

func (repo CloudControllerRouteRepository) Create(host string, domain models.DomainFields, path string, port int, useRandomPort bool) (createdRoute models.Route, apiErr error) {
	return repo.CreateInSpace(host, path, domain.GUID, repo.config.SpaceFields().GUID, port, useRandomPort)
}

func (repo CloudControllerRouteRepository) CheckIfExists(host string, domain models.DomainFields, path string) (bool, error) {
	path = normalizedPath(path)

	u, err := url.Parse(repo.config.APIEndpoint())
	if err != nil {
		return false, err
	}

	u.Path = fmt.Sprintf("/v2/routes/reserved/domain/%s/host/%s", domain.GUID, host)
	if path != "" {
		q := u.Query()
		q.Set("path", path)
		u.RawQuery = q.Encode()
	}

	var rawResponse interface{}
	err = repo.gateway.GetResource(u.String(), &rawResponse)
	if err != nil {
		if _, ok := err.(*errors.HTTPNotFoundError); ok {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (repo CloudControllerRouteRepository) CreateInSpace(host, path, domainGUID, spaceGUID string, port int, randomPort bool) (models.Route, error) {
	path = normalizedPath(path)

	body := struct {
		Host       string `json:"host,omitempty"`
		Path       string `json:"path,omitempty"`
		Port       int    `json:"port,omitempty"`
		DomainGUID string `json:"domain_guid"`
		SpaceGUID  string `json:"space_guid"`
	}{host, path, port, domainGUID, spaceGUID}

	data, err := json.Marshal(body)
	if err != nil {
		return models.Route{}, err
	}

	q := struct {
		GeneratePort         bool `url:"generate_port,omitempty"`
		InlineRelationsDepth int  `url:"inline-relations-depth"`
	}{randomPort, 1}

	opt, _ := query.Values(q)
	uriFragment := "/v2/routes?" + opt.Encode()

	resource := new(resources.RouteResource)
	err = repo.gateway.CreateResource(
		repo.config.APIEndpoint(),
		uriFragment,
		bytes.NewReader(data),
		resource,
	)
	if err != nil {
		return models.Route{}, err
	}

	return resource.ToModel(), nil
}

func (repo CloudControllerRouteRepository) Bind(routeGUID, appGUID string) (apiErr error) {
	path := fmt.Sprintf("/v2/apps/%s/routes/%s", appGUID, routeGUID)
	return repo.gateway.UpdateResource(repo.config.APIEndpoint(), path, nil)
}

func (repo CloudControllerRouteRepository) Unbind(routeGUID, appGUID string) (apiErr error) {
	path := fmt.Sprintf("/v2/apps/%s/routes/%s", appGUID, routeGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}

func (repo CloudControllerRouteRepository) Delete(routeGUID string) (apiErr error) {
	path := fmt.Sprintf("/v2/routes/%s", routeGUID)
	return repo.gateway.DeleteResource(repo.config.APIEndpoint(), path)
}
