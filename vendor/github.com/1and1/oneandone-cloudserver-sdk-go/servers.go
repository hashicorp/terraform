package oneandone

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
)

type Server struct {
	ApiPtr
	Identity
	descField
	CloudPanelId  string           `json:"cloudpanel_id,omitempty"`
	CreationDate  string           `json:"creation_date,omitempty"`
	FirstPassword string           `json:"first_password,omitempty"`
	Datacenter    *Datacenter      `json:"datacenter,omitempty"`
	Status        *Status          `json:"status,omitempty"`
	Hardware      *Hardware        `json:"hardware,omitempty"`
	Image         *Identity        `json:"image,omitempty"`
	Dvd           *Identity        `json:"dvd,omitempty"`
	MonPolicy     *Identity        `json:"monitoring_policy,omitempty"`
	Snapshot      *ServerSnapshot  `json:"snapshot,omitempty"`
	Ips           []ServerIp       `json:"ips,omitempty"`
	PrivateNets   []Identity       `json:"private_networks,omitempty"`
	Alerts        *ServerAlerts    `json:"-"`
	AlertsRaw     *json.RawMessage `json:"alerts,omitempty"`
}

type Hardware struct {
	Vcores            int     `json:"vcore,omitempty"`
	CoresPerProcessor int     `json:"cores_per_processor"`
	Ram               float32 `json:"ram"`
	Hdds              []Hdd   `json:"hdds,omitempty"`
	FixedInsSizeId    string  `json:"fixed_instance_size_id,omitempty"`
	ApiPtr
}

type ServerHdds struct {
	Hdds []Hdd `json:"hdds,omitempty"`
}

type Hdd struct {
	idField
	Size   int  `json:"size,omitempty"`
	IsMain bool `json:"is_main,omitempty"`
	ApiPtr
}

type serverDeployImage struct {
	idField
	Password string    `json:"password,omitempty"`
	Firewall *Identity `json:"firewall_policy,omitempty"`
}

type ServerIp struct {
	idField
	typeField
	Ip            string     `json:"ip,omitempty"`
	ReverseDns    string     `json:"reverse_dns,omitempty"`
	Firewall      *Identity  `json:"firewall_policy,omitempty"`
	LoadBalancers []Identity `json:"load_balancers,omitempty"`
	ApiPtr
}

type ServerIpInfo struct {
	idField           // IP id
	Ip         string `json:"ip,omitempty"`
	ServerName string `json:"server_name,omitempty"`
}

type ServerSnapshot struct {
	idField
	CreationDate string `json:"creation_date,omitempty"`
	DeletionDate string `json:"deletion_date,omitempty"`
}

type ServerAlerts struct {
	AlertSummary []serverAlertSummary
	AlertDetails *serverAlertDetails
}

type serverAlertSummary struct {
	countField
	typeField
}

type serverAlertDetails struct {
	Criticals []ServerAlert `json:"critical,omitempty"`
	Warnings  []ServerAlert `json:"warning,omitempty"`
}

type ServerAlert struct {
	typeField
	descField
	Date string `json:"date"`
}

type ServerRequest struct {
	Name               string   `json:"name,omitempty"`
	Description        string   `json:"description,omitempty"`
	Hardware           Hardware `json:"hardware"`
	ApplianceId        string   `json:"appliance_id,omitempty"`
	Password           string   `json:"password,omitempty"`
	PowerOn            bool     `json:"power_on"`
	FirewallPolicyId   string   `json:"firewall_policy_id,omitempty"`
	IpId               string   `json:"ip_id,omitempty"`
	LoadBalancerId     string   `json:"load_balancer_id,omitempty"`
	MonitoringPolicyId string   `json:"monitoring_policy_id,omitempty"`
	DatacenterId       string   `json:"datacenter_id,omitempty"`
	SSHKey             string   `json:"rsa_key,omitempty"`
}

type ServerAction struct {
	Action string `json:"action,omitempty"`
	Method string `json:"method,omitempty"`
}

type FixedInstanceInfo struct {
	Identity
	Hardware *Hardware `json:"hardware,omitempty"`
	ApiPtr
}

// GET /servers
func (api *API) ListServers(args ...interface{}) ([]Server, error) {
	url, err := processQueryParams(createUrl(api, serverPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []Server{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for _, s := range result {
		s.api = api
		s.decodeRaws()
	}
	return result, nil
}

// POST /servers
func (api *API) CreateServer(request *ServerRequest) (string, *Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment)
	insert2map := func(hasht map[string]interface{}, key string, value string) {
		if key != "" && value != "" {
			hasht[key] = value
		}
	}
	req := make(map[string]interface{})
	hw := make(map[string]interface{})
	req["name"] = request.Name
	req["description"] = request.Description
	req["appliance_id"] = request.ApplianceId
	req["power_on"] = request.PowerOn
	insert2map(req, "password", request.Password)
	insert2map(req, "firewall_policy_id", request.FirewallPolicyId)
	insert2map(req, "ip_id", request.IpId)
	insert2map(req, "load_balancer_id", request.LoadBalancerId)
	insert2map(req, "monitoring_policy_id", request.MonitoringPolicyId)
	insert2map(req, "datacenter_id", request.DatacenterId)
	insert2map(req, "rsa_key", request.SSHKey)
	req["hardware"] = hw
	if request.Hardware.FixedInsSizeId != "" {
		hw["fixed_instance_size_id"] = request.Hardware.FixedInsSizeId
	} else {
		hw["vcore"] = request.Hardware.Vcores
		hw["cores_per_processor"] = request.Hardware.CoresPerProcessor
		hw["ram"] = request.Hardware.Ram
		hw["hdds"] = request.Hardware.Hdds
	}
	err := api.Client.Post(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return "", nil, err
	}
	result.api = api
	result.decodeRaws()
	return result.Id, result, nil
}

// This is a wraper function for `CreateServer` that returns the server's IP address and first password.
// The function waits at most `timeout` seconds for the server to be created.
// The initial `POST /servers` response does not contain the IP address, so we need to wait
// until the server is created.
func (api *API) CreateServerEx(request *ServerRequest, timeout int) (string, string, error) {
	id, server, err := api.CreateServer(request)
	if server != nil && err == nil {
		count := timeout / 5
		if request.PowerOn {
			err = api.WaitForState(server, "POWERED_ON", 5, count)
		} else {
			err = api.WaitForState(server, "POWERED_OFF", 5, count)
		}
		if err != nil {
			return "", "", err
		}
		server, err := api.GetServer(id)
		if server != nil && err == nil && server.Ips[0].Ip != "" {
			if server.FirstPassword != "" {
				return server.Ips[0].Ip, server.FirstPassword, nil
			}
			if request.Password != "" {
				return server.Ips[0].Ip, request.Password, nil
			}
			// should never reach here
			return "", "", errors.New("No server's password was found.")
		}
	}
	return "", "", err
}

// GET /servers/{id}
func (api *API) GetServer(server_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/fixed_instance_sizes
func (api *API) ListFixedInstanceSizes() ([]FixedInstanceInfo, error) {
	result := []FixedInstanceInfo{}
	url := createUrl(api, serverPathSegment, "fixed_instance_sizes")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// GET /servers/fixed_instance_sizes/{fixed_instance_size_id}
func (api *API) GetFixedInstanceSize(fis_id string) (*FixedInstanceInfo, error) {
	result := new(FixedInstanceInfo)
	url := createUrl(api, serverPathSegment, "fixed_instance_sizes", fis_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /servers/{id}
func (api *API) DeleteServer(server_id string, keep_ips bool) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id)
	pm := make(map[string]interface{}, 1)
	pm["keep_ips"] = keep_ips
	url = appendQueryParams(url, pm)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// PUT /servers/{id}
func (api *API) RenameServer(server_id string, new_name string, new_desc string) (*Server, error) {
	data := struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	}{Name: new_name, Description: new_desc}
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id)
	err := api.Client.Put(url, &data, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{server_id}/hardware
func (api *API) GetServerHardware(server_id string) (*Hardware, error) {
	result := new(Hardware)
	url := createUrl(api, serverPathSegment, server_id, "hardware")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /servers/{server_id}/hardware
func (api *API) UpdateServerHardware(server_id string, hardware *Hardware) (*Server, error) {
	var vc, cpp *int
	var ram *float32
	if hardware.Vcores > 0 {
		vc = new(int)
		*vc = hardware.Vcores
	}
	if hardware.CoresPerProcessor > 0 {
		cpp = new(int)
		*cpp = hardware.CoresPerProcessor
	}
	if big.NewFloat(float64(hardware.Ram)).Cmp(big.NewFloat(0)) != 0 {
		ram = new(float32)
		*ram = hardware.Ram
	}
	req := struct {
		VCores *int     `json:"vcore,omitempty"`
		Cpp    *int     `json:"cores_per_processor,omitempty"`
		Ram    *float32 `json:"ram,omitempty"`
		Flavor string   `json:"fixed_instance_size_id,omitempty"`
	}{VCores: vc, Cpp: cpp, Ram: ram, Flavor: hardware.FixedInsSizeId}

	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "hardware")
	err := api.Client.Put(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/hardware/hdds
func (api *API) ListServerHdds(server_id string) ([]Hdd, error) {
	result := []Hdd{}
	url := createUrl(api, serverPathSegment, server_id, "hardware/hdds")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /servers/{id}/hardware/hdds
func (api *API) AddServerHdds(server_id string, hdds *ServerHdds) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "hardware/hdds")
	err := api.Client.Post(url, &hdds, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/hardware/hdds/{id}
func (api *API) GetServerHdd(server_id string, hdd_id string) (*Hdd, error) {
	result := new(Hdd)
	url := createUrl(api, serverPathSegment, server_id, "hardware/hdds", hdd_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /servers/{id}/hardware/hdds/{id}
func (api *API) DeleteServerHdd(server_id string, hdd_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "hardware/hdds", hdd_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// PUT /servers/{id}/hardware/hdds/{id}
func (api *API) ResizeServerHdd(server_id string, hdd_id string, new_size int) (*Server, error) {
	data := Hdd{Size: new_size}
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "hardware/hdds", hdd_id)
	err := api.Client.Put(url, &data, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/image
func (api *API) GetServerImage(server_id string) (*Identity, error) {
	result := new(Identity)
	url := createUrl(api, serverPathSegment, server_id, "image")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// PUT /servers/{id}/image
func (api *API) ReinstallServerImage(server_id string, image_id string, password string, fp_id string) (*Server, error) {
	data := new(serverDeployImage)
	data.Id = image_id
	data.Password = password
	if fp_id != "" {
		fp := new(Identity)
		fp.Id = fp_id
		data.Firewall = fp
	}

	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "image")
	err := api.Client.Put(url, &data, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/ips
func (api *API) ListServerIps(server_id string) ([]ServerIp, error) {
	result := []ServerIp{}
	url := createUrl(api, serverPathSegment, server_id, "ips")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /servers/{id}/ips
func (api *API) AssignServerIp(server_id string, ip_type string) (*Server, error) {
	data := typeField{Type: ip_type}
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "ips")
	err := api.Client.Post(url, &data, &result, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/ips/{id}
func (api *API) GetServerIp(server_id string, ip_id string) (*ServerIp, error) {
	result := new(ServerIp)
	url := createUrl(api, serverPathSegment, server_id, "ips", ip_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /servers/{id}/ips/{id}
func (api *API) DeleteServerIp(server_id string, ip_id string, keep_ip bool) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "ips", ip_id)
	qm := make(map[string]interface{}, 1)
	qm["keep_ip"] = keep_ip
	url = appendQueryParams(url, qm)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /servers/{id}/status
func (api *API) GetServerStatus(server_id string) (*Status, error) {
	result := new(Status)
	url := createUrl(api, serverPathSegment, server_id, "status")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// PUT /servers/{id}/status/action (action = REBOOT)
func (api *API) RebootServer(server_id string, is_hardware bool) (*Server, error) {
	result := new(Server)
	request := ServerAction{}
	request.Action = "REBOOT"
	if is_hardware {
		request.Method = "HARDWARE"
	} else {
		request.Method = "SOFTWARE"
	}
	url := createUrl(api, serverPathSegment, server_id, "status", "action")
	err := api.Client.Put(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// PUT /servers/{id}/status/action (action = POWER_OFF)
func (api *API) ShutdownServer(server_id string, is_hardware bool) (*Server, error) {
	result := new(Server)
	request := ServerAction{}
	request.Action = "POWER_OFF"
	if is_hardware {
		request.Method = "HARDWARE"
	} else {
		request.Method = "SOFTWARE"
	}
	url := createUrl(api, serverPathSegment, server_id, "status", "action")
	err := api.Client.Put(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// PUT /servers/{id}/status/action (action = POWER_ON)
func (api *API) StartServer(server_id string) (*Server, error) {
	result := new(Server)
	request := ServerAction{}
	request.Action = "POWER_ON"
	url := createUrl(api, serverPathSegment, server_id, "status", "action")
	err := api.Client.Put(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/dvd
func (api *API) GetServerDvd(server_id string) (*Identity, error) {
	result := new(Identity)
	url := createUrl(api, serverPathSegment, server_id, "dvd")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /servers/{id}/dvd
func (api *API) EjectServerDvd(server_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "dvd")
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// PUT /servers/{id}/dvd
func (api *API) LoadServerDvd(server_id string, dvd_id string) (*Server, error) {
	request := Identity{}
	request.Id = dvd_id
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "dvd")
	err := api.Client.Put(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/private_networks
func (api *API) ListServerPrivateNetworks(server_id string) ([]Identity, error) {
	result := []Identity{}
	url := createUrl(api, serverPathSegment, server_id, "private_networks")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /servers/{id}/private_networks
func (api *API) AssignServerPrivateNetwork(server_id string, pn_id string) (*Server, error) {
	req := new(Identity)
	req.Id = pn_id
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "private_networks")
	err := api.Client.Post(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/private_networks/{id}
func (api *API) GetServerPrivateNetwork(server_id string, pn_id string) (*PrivateNetwork, error) {
	result := new(PrivateNetwork)
	url := createUrl(api, serverPathSegment, server_id, "private_networks", pn_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /servers/{id}/private_networks/{id}
func (api *API) RemoveServerPrivateNetwork(server_id string, pn_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "private_networks", pn_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{server_id}/ips/{ip_id}/load_balancers
func (api *API) ListServerIpLoadBalancers(server_id string, ip_id string) ([]Identity, error) {
	result := []Identity{}
	url := createUrl(api, serverPathSegment, server_id, "ips", ip_id, "load_balancers")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /servers/{server_id}/ips/{ip_id}/load_balancers
func (api *API) AssignServerIpLoadBalancer(server_id string, ip_id string, lb_id string) (*Server, error) {
	req := struct {
		LbId string `json:"load_balancer_id"`
	}{lb_id}
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "ips", ip_id, "load_balancers")
	err := api.Client.Post(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// DELETE /servers/{server_id}/ips/{ip_id}/load_balancers
func (api *API) UnassignServerIpLoadBalancer(server_id string, ip_id string, lb_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "ips", ip_id, "load_balancers", lb_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{server_id}/ips/{ip_id}/firewall_policy
func (api *API) GetServerIpFirewallPolicy(server_id string, ip_id string) (*Identity, error) {
	result := new(Identity)
	url := createUrl(api, serverPathSegment, server_id, "ips", ip_id, "firewall_policy")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// PUT /servers/{server_id}/ips/{ip_id}/firewall_policy
func (api *API) AssignServerIpFirewallPolicy(server_id string, ip_id string, fp_id string) (*Server, error) {
	req := idField{fp_id}
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "ips", ip_id, "firewall_policy")
	err := api.Client.Put(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// DELETE /servers/{server_id}/ips/{ip_id}/firewall_policy
func (api *API) UnassignServerIpFirewallPolicy(server_id string, ip_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "ips", ip_id, "firewall_policy")
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// GET /servers/{id}/snapshots
func (api *API) GetServerSnapshot(server_id string) (*ServerSnapshot, error) {
	result := new(ServerSnapshot)
	url := createUrl(api, serverPathSegment, server_id, "snapshots")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /servers/{id}/snapshots
func (api *API) CreateServerSnapshot(server_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "snapshots")
	err := api.Client.Post(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// PUT /servers/{server_id}/snapshots/{snapshot_id}
func (api *API) RestoreServerSnapshot(server_id string, snapshot_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "snapshots", snapshot_id)
	err := api.Client.Put(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// DELETE /servers/{server_id}/snapshots/{snapshot_id}
func (api *API) DeleteServerSnapshot(server_id string, snapshot_id string) (*Server, error) {
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "snapshots", snapshot_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

// POST /servers/{server_id}/clone
func (api *API) CloneServer(server_id string, new_name string, datacenter_id string) (*Server, error) {
	data := struct {
		Name         string `json:"name"`
		DatacenterId string `json:"datacenter_id,omitempty"`
	}{Name: new_name, DatacenterId: datacenter_id}
	result := new(Server)
	url := createUrl(api, serverPathSegment, server_id, "clone")
	err := api.Client.Post(url, &data, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	result.decodeRaws()
	return result, nil
}

func (s *Server) GetState() (string, error) {
	st, err := s.api.GetServerStatus(s.Id)
	if st == nil {
		return "", err
	}
	return st.State, err
}

func (server *Server) decodeRaws() {
	if server.AlertsRaw != nil {
		server.Alerts = new(ServerAlerts)
		var sad serverAlertDetails
		if err := json.Unmarshal(*server.AlertsRaw, &sad); err == nil {
			server.Alerts.AlertDetails = &sad
			return
		}
		var sams []serverAlertSummary
		if err := json.Unmarshal(*server.AlertsRaw, &sams); err == nil {
			server.Alerts.AlertSummary = sams
		}
	}
}
