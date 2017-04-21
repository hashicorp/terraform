package oneandone

import (
	"net/http"
)

type MonitoringPolicy struct {
	ApiPtr
	idField
	Name         string               `json:"name,omitempty"`
	Description  string               `json:"description,omitempty"`
	State        string               `json:"state,omitempty"`
	Default      *int                 `json:"default,omitempty"`
	CreationDate string               `json:"creation_date,omitempty"`
	Email        string               `json:"email,omitempty"`
	Agent        bool                 `json:"agent"`
	Servers      []Identity           `json:"servers,omitempty"`
	Thresholds   *MonitoringThreshold `json:"thresholds,omitempty"`
	Ports        []MonitoringPort     `json:"ports,omitempty"`
	Processes    []MonitoringProcess  `json:"processes,omitempty"`
	CloudPanelId string               `json:"cloudpanel_id,omitempty"`
}

type MonitoringThreshold struct {
	Cpu          *MonitoringLevel `json:"cpu,omitempty"`
	Ram          *MonitoringLevel `json:"ram,omitempty"`
	Disk         *MonitoringLevel `json:"disk,omitempty"`
	Transfer     *MonitoringLevel `json:"transfer,omitempty"`
	InternalPing *MonitoringLevel `json:"internal_ping,omitempty"`
}

type MonitoringLevel struct {
	Warning  *MonitoringValue `json:"warning,omitempty"`
	Critical *MonitoringValue `json:"critical,omitempty"`
}

type MonitoringValue struct {
	Value int  `json:"value"`
	Alert bool `json:"alert"`
}

type MonitoringPort struct {
	idField
	Protocol          string `json:"protocol,omitempty"`
	Port              int    `json:"port"`
	AlertIf           string `json:"alert_if,omitempty"`
	EmailNotification bool   `json:"email_notification"`
}

type MonitoringProcess struct {
	idField
	Process           string `json:"process,omitempty"`
	AlertIf           string `json:"alert_if,omitempty"`
	EmailNotification bool   `json:"email_notification"`
}

// GET /monitoring_policies
func (api *API) ListMonitoringPolicies(args ...interface{}) ([]MonitoringPolicy, error) {
	url, err := processQueryParams(createUrl(api, monitorPolicyPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []MonitoringPolicy{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /monitoring_policies
func (api *API) CreateMonitoringPolicy(mp *MonitoringPolicy) (string, *MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	url := createUrl(api, monitorPolicyPathSegment)
	err := api.Client.Post(url, &mp, &result, http.StatusCreated)
	if err != nil {
		return "", nil, err
	}
	result.api = api
	return result.Id, result, nil
}

// GET /monitoring_policies/{id}
func (api *API) GetMonitoringPolicy(mp_id string) (*MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	url := createUrl(api, monitorPolicyPathSegment, mp_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /monitoring_policies/{id}
func (api *API) DeleteMonitoringPolicy(mp_id string) (*MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	url := createUrl(api, monitorPolicyPathSegment, mp_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /monitoring_policies/{id}
func (api *API) UpdateMonitoringPolicy(mp_id string, mp *MonitoringPolicy) (*MonitoringPolicy, error) {
	url := createUrl(api, monitorPolicyPathSegment, mp_id)
	result := new(MonitoringPolicy)
	err := api.Client.Put(url, &mp, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /monitoring_policies/{id}/ports
func (api *API) ListMonitoringPolicyPorts(mp_id string) ([]MonitoringPort, error) {
	result := []MonitoringPort{}
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "ports")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /monitoring_policies/{id}/ports
func (api *API) AddMonitoringPolicyPorts(mp_id string, mp_ports []MonitoringPort) (*MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	data := struct {
		Ports []MonitoringPort `json:"ports"`
	}{mp_ports}
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "ports")
	err := api.Client.Post(url, &data, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /monitoring_policies/{id}/ports/{id}
func (api *API) GetMonitoringPolicyPort(mp_id string, port_id string) (*MonitoringPort, error) {
	result := new(MonitoringPort)
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "ports", port_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /monitoring_policies/{id}/ports/{id}
func (api *API) DeleteMonitoringPolicyPort(mp_id string, port_id string) (*MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "ports", port_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /monitoring_policies/{id}/ports/{id}
func (api *API) ModifyMonitoringPolicyPort(mp_id string, port_id string, mp_port *MonitoringPort) (*MonitoringPolicy, error) {
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "ports", port_id)
	result := new(MonitoringPolicy)
	req := struct {
		Ports *MonitoringPort `json:"ports"`
	}{mp_port}
	err := api.Client.Put(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /monitoring_policies/{id}/processes
func (api *API) ListMonitoringPolicyProcesses(mp_id string) ([]MonitoringProcess, error) {
	result := []MonitoringProcess{}
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "processes")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /monitoring_policies/{id}/processes
func (api *API) AddMonitoringPolicyProcesses(mp_id string, mp_procs []MonitoringProcess) (*MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	request := struct {
		Processes []MonitoringProcess `json:"processes"`
	}{mp_procs}
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "processes")
	err := api.Client.Post(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /monitoring_policies/{id}/processes/{id}
func (api *API) GetMonitoringPolicyProcess(mp_id string, proc_id string) (*MonitoringProcess, error) {
	result := new(MonitoringProcess)
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "processes", proc_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /monitoring_policies/{id}/processes/{id}
func (api *API) DeleteMonitoringPolicyProcess(mp_id string, proc_id string) (*MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "processes", proc_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /monitoring_policies/{id}/processes/{id}
func (api *API) ModifyMonitoringPolicyProcess(mp_id string, proc_id string, mp_proc *MonitoringProcess) (*MonitoringPolicy, error) {
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "processes", proc_id)
	result := new(MonitoringPolicy)
	req := struct {
		Processes *MonitoringProcess `json:"processes"`
	}{mp_proc}
	err := api.Client.Put(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /monitoring_policies/{id}/servers
func (api *API) ListMonitoringPolicyServers(mp_id string) ([]Identity, error) {
	result := []Identity{}
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "servers")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /monitoring_policies/{id}/servers
func (api *API) AttachMonitoringPolicyServers(mp_id string, sids []string) (*MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	request := servers{
		Servers: sids,
	}
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "servers")
	err := api.Client.Post(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /monitoring_policies/{id}/servers/{id}
func (api *API) GetMonitoringPolicyServer(mp_id string, ser_id string) (*Identity, error) {
	result := new(Identity)
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "servers", ser_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /monitoring_policies/{id}/servers/{id}
func (api *API) RemoveMonitoringPolicyServer(mp_id string, ser_id string) (*MonitoringPolicy, error) {
	result := new(MonitoringPolicy)
	url := createUrl(api, monitorPolicyPathSegment, mp_id, "servers", ser_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

func (mp *MonitoringPolicy) GetState() (string, error) {
	in, err := mp.api.GetMonitoringPolicy(mp.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
