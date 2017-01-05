package api

import (
	"fmt"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/models"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . ServicePlanRepository

type ServicePlanRepository interface {
	Search(searchParameters map[string]string) ([]models.ServicePlanFields, error)
	Update(models.ServicePlanFields, string, bool) error
	ListPlansFromManyServices(serviceGUIDs []string) ([]models.ServicePlanFields, error)
}

type CloudControllerServicePlanRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerServicePlanRepository(config coreconfig.Reader, gateway net.Gateway) CloudControllerServicePlanRepository {
	return CloudControllerServicePlanRepository{
		config:  config,
		gateway: gateway,
	}
}

func (repo CloudControllerServicePlanRepository) Update(servicePlan models.ServicePlanFields, serviceGUID string, public bool) error {
	return repo.gateway.UpdateResource(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/service_plans/%s", servicePlan.GUID),
		strings.NewReader(fmt.Sprintf(`{"public":%t}`, public)),
	)
}

func (repo CloudControllerServicePlanRepository) ListPlansFromManyServices(serviceGUIDs []string) ([]models.ServicePlanFields, error) {
	serviceGUIDsString := strings.Join(serviceGUIDs, ",")
	plans := []models.ServicePlanFields{}

	err := repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		fmt.Sprintf("/v2/service_plans?q=%s", url.QueryEscape("service_guid IN "+serviceGUIDsString)),
		resources.ServicePlanResource{},
		func(resource interface{}) bool {
			if plan, ok := resource.(resources.ServicePlanResource); ok {
				plans = append(plans, plan.ToFields())
			}
			return true
		})
	return plans, err
}

func (repo CloudControllerServicePlanRepository) Search(queryParams map[string]string) (plans []models.ServicePlanFields, err error) {
	err = repo.gateway.ListPaginatedResources(
		repo.config.APIEndpoint(),
		combineQueryParametersWithURI("/v2/service_plans", queryParams),
		resources.ServicePlanResource{},
		func(resource interface{}) bool {
			if sp, ok := resource.(resources.ServicePlanResource); ok {
				plans = append(plans, sp.ToFields())
			}
			return true
		})
	return
}

func combineQueryParametersWithURI(uri string, queryParams map[string]string) string {
	if len(queryParams) == 0 {
		return uri
	}

	params := []string{}
	for key, value := range queryParams {
		params = append(params, url.QueryEscape(key+":"+value))
	}

	return uri + "?q=" + strings.Join(params, "%3B")
}
