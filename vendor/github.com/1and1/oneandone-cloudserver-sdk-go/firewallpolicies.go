package oneandone

import (
	"net/http"
)

type FirewallPolicy struct {
	Identity
	descField
	DefaultPolicy uint8                `json:"default"`
	CloudpanelId  string               `json:"cloudpanel_id,omitempty"`
	CreationDate  string               `json:"creation_date,omitempty"`
	State         string               `json:"state,omitempty"`
	Rules         []FirewallPolicyRule `json:"rules,omitempty"`
	ServerIps     []ServerIpInfo       `json:"server_ips,omitempty"`
	ApiPtr
}

type FirewallPolicyRule struct {
	idField
	Protocol string `json:"protocol,omitempty"`
	PortFrom *int   `json:"port_from,omitempty"`
	PortTo   *int   `json:"port_to,omitempty"`
	SourceIp string `json:"source,omitempty"`
}

type FirewallPolicyRequest struct {
	Name        string               `json:"name,omitempty"`
	Description string               `json:"description,omitempty"`
	Rules       []FirewallPolicyRule `json:"rules,omitempty"`
}

// GET /firewall_policies
func (api *API) ListFirewallPolicies(args ...interface{}) ([]FirewallPolicy, error) {
	url, err := processQueryParams(createUrl(api, firewallPolicyPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []FirewallPolicy{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /firewall_policies
func (api *API) CreateFirewallPolicy(fp_data *FirewallPolicyRequest) (string, *FirewallPolicy, error) {
	result := new(FirewallPolicy)
	url := createUrl(api, firewallPolicyPathSegment)
	err := api.Client.Post(url, &fp_data, &result, http.StatusAccepted)
	if err != nil {
		return "", nil, err
	}
	result.api = api
	return result.Id, result, nil
}

// GET /firewall_policies/{id}
func (api *API) GetFirewallPolicy(fp_id string) (*FirewallPolicy, error) {
	result := new(FirewallPolicy)
	url := createUrl(api, firewallPolicyPathSegment, fp_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil

}

// DELETE /firewall_policies/{id}
func (api *API) DeleteFirewallPolicy(fp_id string) (*FirewallPolicy, error) {
	result := new(FirewallPolicy)
	url := createUrl(api, firewallPolicyPathSegment, fp_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /firewall_policies/{id}
func (api *API) UpdateFirewallPolicy(fp_id string, fp_new_name string, fp_new_desc string) (*FirewallPolicy, error) {
	result := new(FirewallPolicy)
	data := FirewallPolicyRequest{
		Name:        fp_new_name,
		Description: fp_new_desc,
	}
	url := createUrl(api, firewallPolicyPathSegment, fp_id)
	err := api.Client.Put(url, &data, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /firewall_policies/{id}/server_ips
func (api *API) ListFirewallPolicyServerIps(fp_id string) ([]ServerIpInfo, error) {
	result := []ServerIpInfo{}
	url := createUrl(api, firewallPolicyPathSegment, fp_id, "server_ips")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GET /firewall_policies/{id}/server_ips/{id}
func (api *API) GetFirewallPolicyServerIp(fp_id string, ip_id string) (*ServerIpInfo, error) {
	result := new(ServerIpInfo)
	url := createUrl(api, firewallPolicyPathSegment, fp_id, "server_ips", ip_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /firewall_policies/{id}/server_ips
func (api *API) AddFirewallPolicyServerIps(fp_id string, ip_ids []string) (*FirewallPolicy, error) {
	result := new(FirewallPolicy)
	request := serverIps{
		ServerIps: ip_ids,
	}

	url := createUrl(api, firewallPolicyPathSegment, fp_id, "server_ips")
	err := api.Client.Post(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /firewall_policies/{id}/server_ips/{id}
func (api *API) DeleteFirewallPolicyServerIp(fp_id string, ip_id string) (*FirewallPolicy, error) {
	result := new(FirewallPolicy)
	url := createUrl(api, firewallPolicyPathSegment, fp_id, "server_ips", ip_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /firewall_policies/{id}/rules
func (api *API) ListFirewallPolicyRules(fp_id string) ([]FirewallPolicyRule, error) {
	result := []FirewallPolicyRule{}
	url := createUrl(api, firewallPolicyPathSegment, fp_id, "rules")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /firewall_policies/{id}/rules
func (api *API) AddFirewallPolicyRules(fp_id string, fp_rules []FirewallPolicyRule) (*FirewallPolicy, error) {
	result := new(FirewallPolicy)
	data := struct {
		Rules []FirewallPolicyRule `json:"rules"`
	}{fp_rules}
	url := createUrl(api, firewallPolicyPathSegment, fp_id, "rules")
	err := api.Client.Post(url, &data, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /firewall_policies/{id}/rules/{id}
func (api *API) GetFirewallPolicyRule(fp_id string, rule_id string) (*FirewallPolicyRule, error) {
	result := new(FirewallPolicyRule)
	url := createUrl(api, firewallPolicyPathSegment, fp_id, "rules", rule_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /firewall_policies/{id}/rules/{id}
func (api *API) DeleteFirewallPolicyRule(fp_id string, rule_id string) (*FirewallPolicy, error) {
	result := new(FirewallPolicy)
	url := createUrl(api, firewallPolicyPathSegment, fp_id, "rules", rule_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

func (fp *FirewallPolicy) GetState() (string, error) {
	in, err := fp.api.GetFirewallPolicy(fp.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
