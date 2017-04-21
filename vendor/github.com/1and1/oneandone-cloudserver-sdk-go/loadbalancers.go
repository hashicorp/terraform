package oneandone

import (
	"net/http"
)

type LoadBalancer struct {
	ApiPtr
	idField
	Name                  string             `json:"name,omitempty"`
	Description           string             `json:"description,omitempty"`
	State                 string             `json:"state,omitempty"`
	CreationDate          string             `json:"creation_date,omitempty"`
	Ip                    string             `json:"ip,omitempty"`
	HealthCheckTest       string             `json:"health_check_test,omitempty"`
	HealthCheckInterval   int                `json:"health_check_interval"`
	HealthCheckPath       string             `json:"health_check_path,omitempty"`
	HealthCheckPathParser string             `json:"health_check_path_parser,omitempty"`
	Persistence           bool               `json:"persistence"`
	PersistenceTime       int                `json:"persistence_time"`
	Method                string             `json:"method,omitempty"`
	Rules                 []LoadBalancerRule `json:"rules,omitempty"`
	ServerIps             []ServerIpInfo     `json:"server_ips,omitempty"`
	Datacenter            *Datacenter        `json:"datacenter,omitempty"`
	CloudPanelId          string             `json:"cloudpanel_id,omitempty"`
}

type LoadBalancerRule struct {
	idField
	Protocol     string `json:"protocol,omitempty"`
	PortBalancer uint16 `json:"port_balancer"`
	PortServer   uint16 `json:"port_server"`
	Source       string `json:"source,omitempty"`
}

type LoadBalancerRequest struct {
	Name                  string             `json:"name,omitempty"`
	Description           string             `json:"description,omitempty"`
	DatacenterId          string             `json:"datacenter_id,omitempty"`
	HealthCheckTest       string             `json:"health_check_test,omitempty"`
	HealthCheckInterval   *int               `json:"health_check_interval"`
	HealthCheckPath       string             `json:"health_check_path,omitempty"`
	HealthCheckPathParser string             `json:"health_check_path_parser,omitempty"`
	Persistence           *bool              `json:"persistence"`
	PersistenceTime       *int               `json:"persistence_time"`
	Method                string             `json:"method,omitempty"`
	Rules                 []LoadBalancerRule `json:"rules,omitempty"`
}

// GET /load_balancers
func (api *API) ListLoadBalancers(args ...interface{}) ([]LoadBalancer, error) {
	url, err := processQueryParams(createUrl(api, loadBalancerPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []LoadBalancer{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /load_balancers
func (api *API) CreateLoadBalancer(request *LoadBalancerRequest) (string, *LoadBalancer, error) {
	url := createUrl(api, loadBalancerPathSegment)
	result := new(LoadBalancer)
	err := api.Client.Post(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return "", nil, err
	}
	result.api = api
	return result.Id, result, nil
}

// GET /load_balancers/{id}
func (api *API) GetLoadBalancer(lb_id string) (*LoadBalancer, error) {
	url := createUrl(api, loadBalancerPathSegment, lb_id)
	result := new(LoadBalancer)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /load_balancers/{id}
func (api *API) DeleteLoadBalancer(lb_id string) (*LoadBalancer, error) {
	url := createUrl(api, loadBalancerPathSegment, lb_id)
	result := new(LoadBalancer)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /load_balancers/{id}
func (api *API) UpdateLoadBalancer(lb_id string, request *LoadBalancerRequest) (*LoadBalancer, error) {
	url := createUrl(api, loadBalancerPathSegment, lb_id)
	result := new(LoadBalancer)
	err := api.Client.Put(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /load_balancers/{id}/server_ips
func (api *API) ListLoadBalancerServerIps(lb_id string) ([]ServerIpInfo, error) {
	result := []ServerIpInfo{}
	url := createUrl(api, loadBalancerPathSegment, lb_id, "server_ips")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GET /load_balancers/{id}/server_ips/{id}
func (api *API) GetLoadBalancerServerIp(lb_id string, ip_id string) (*ServerIpInfo, error) {
	result := new(ServerIpInfo)
	url := createUrl(api, loadBalancerPathSegment, lb_id, "server_ips", ip_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /load_balancers/{id}/server_ips
func (api *API) AddLoadBalancerServerIps(lb_id string, ip_ids []string) (*LoadBalancer, error) {
	result := new(LoadBalancer)
	request := serverIps{
		ServerIps: ip_ids,
	}
	url := createUrl(api, loadBalancerPathSegment, lb_id, "server_ips")
	err := api.Client.Post(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /load_balancers/{id}/server_ips/{id}
func (api *API) DeleteLoadBalancerServerIp(lb_id string, ip_id string) (*LoadBalancer, error) {
	result := new(LoadBalancer)
	url := createUrl(api, loadBalancerPathSegment, lb_id, "server_ips", ip_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /load_balancers/{load_balancer_id}/rules
func (api *API) ListLoadBalancerRules(lb_id string) ([]LoadBalancerRule, error) {
	result := []LoadBalancerRule{}
	url := createUrl(api, loadBalancerPathSegment, lb_id, "rules")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /load_balancers/{load_balancer_id}/rules
func (api *API) AddLoadBalancerRules(lb_id string, lb_rules []LoadBalancerRule) (*LoadBalancer, error) {
	result := new(LoadBalancer)
	data := struct {
		Rules []LoadBalancerRule `json:"rules"`
	}{lb_rules}
	url := createUrl(api, loadBalancerPathSegment, lb_id, "rules")
	err := api.Client.Post(url, &data, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /load_balancers/{load_balancer_id}/rules/{rule_id}
func (api *API) GetLoadBalancerRule(lb_id string, rule_id string) (*LoadBalancerRule, error) {
	result := new(LoadBalancerRule)
	url := createUrl(api, loadBalancerPathSegment, lb_id, "rules", rule_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /load_balancers/{load_balancer_id}/rules/{rule_id}
func (api *API) DeleteLoadBalancerRule(lb_id string, rule_id string) (*LoadBalancer, error) {
	result := new(LoadBalancer)
	url := createUrl(api, loadBalancerPathSegment, lb_id, "rules", rule_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

func (lb *LoadBalancer) GetState() (string, error) {
	in, err := lb.api.GetLoadBalancer(lb.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
