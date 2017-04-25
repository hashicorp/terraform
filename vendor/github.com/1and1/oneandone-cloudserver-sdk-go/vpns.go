package oneandone

import "net/http"

type VPN struct {
	Identity
	descField
	typeField
	CloudPanelId string      `json:"cloudpanel_id,omitempty"`
	CreationDate string      `json:"creation_date,omitempty"`
	State        string      `json:"state,omitempty"`
	IPs          []string    `json:"ips,omitempty"`
	Datacenter   *Datacenter `json:"datacenter,omitempty"`
	ApiPtr
}

type configZipFile struct {
	Base64String string `json:"config_zip_file"`
}

// GET /vpns
func (api *API) ListVPNs(args ...interface{}) ([]VPN, error) {
	url, err := processQueryParams(createUrl(api, vpnPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []VPN{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for _, vpn := range result {
		vpn.api = api
	}
	return result, nil
}

// POST /vpns
func (api *API) CreateVPN(name string, description string, datacenter_id string) (string, *VPN, error) {
	res := new(VPN)
	url := createUrl(api, vpnPathSegment)
	req := struct {
		Name         string `json:"name"`
		Description  string `json:"description,omitempty"`
		DatacenterId string `json:"datacenter_id,omitempty"`
	}{Name: name, Description: description, DatacenterId: datacenter_id}
	err := api.Client.Post(url, &req, &res, http.StatusAccepted)
	if err != nil {
		return "", nil, err
	}
	res.api = api
	return res.Id, res, nil
}

// GET /vpns/{vpn_id}
func (api *API) GetVPN(vpn_id string) (*VPN, error) {
	result := new(VPN)
	url := createUrl(api, vpnPathSegment, vpn_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /vpns/{vpn_id}
func (api *API) ModifyVPN(vpn_id string, name string, description string) (*VPN, error) {
	result := new(VPN)
	url := createUrl(api, vpnPathSegment, vpn_id)
	req := struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	}{Name: name, Description: description}
	err := api.Client.Put(url, &req, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /vpns/{vpn_id}
func (api *API) DeleteVPN(vpn_id string) (*VPN, error) {
	result := new(VPN)
	url := createUrl(api, vpnPathSegment, vpn_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /vpns/{vpn_id}/configuration_file
// Returns VPN configuration files (in a zip arhive) as a base64 encoded string
func (api *API) GetVPNConfigFile(vpn_id string) (string, error) {
	result := new(configZipFile)
	url := createUrl(api, vpnPathSegment, vpn_id, "configuration_file")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return "", err
	}

	return result.Base64String, nil
}

func (vpn *VPN) GetState() (string, error) {
	in, err := vpn.api.GetVPN(vpn.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
