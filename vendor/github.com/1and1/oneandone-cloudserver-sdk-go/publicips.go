package oneandone

import "net/http"

type PublicIp struct {
	idField
	typeField
	IpAddress    string      `json:"ip,omitempty"`
	AssignedTo   *assignedTo `json:"assigned_to,omitempty"`
	ReverseDns   string      `json:"reverse_dns,omitempty"`
	IsDhcp       *bool       `json:"is_dhcp,omitempty"`
	State        string      `json:"state,omitempty"`
	SiteId       string      `json:"site_id,omitempty"`
	CreationDate string      `json:"creation_date,omitempty"`
	Datacenter   *Datacenter `json:"datacenter,omitempty"`
	ApiPtr
}

type assignedTo struct {
	Identity
	typeField
}

const (
	IpTypeV4 = "IPV4"
	IpTypeV6 = "IPV6"
)

// GET /public_ips
func (api *API) ListPublicIps(args ...interface{}) ([]PublicIp, error) {
	url, err := processQueryParams(createUrl(api, publicIpPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []PublicIp{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /public_ips
func (api *API) CreatePublicIp(ip_type string, reverse_dns string, datacenter_id string) (string, *PublicIp, error) {
	res := new(PublicIp)
	url := createUrl(api, publicIpPathSegment)
	req := struct {
		DatacenterId string `json:"datacenter_id,omitempty"`
		ReverseDns   string `json:"reverse_dns,omitempty"`
		Type         string `json:"type,omitempty"`
	}{DatacenterId: datacenter_id, ReverseDns: reverse_dns, Type: ip_type}
	err := api.Client.Post(url, &req, &res, http.StatusCreated)
	if err != nil {
		return "", nil, err
	}
	res.api = api
	return res.Id, res, nil
}

// GET /public_ips/{id}
func (api *API) GetPublicIp(ip_id string) (*PublicIp, error) {
	result := new(PublicIp)
	url := createUrl(api, publicIpPathSegment, ip_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /public_ips/{id}
func (api *API) DeletePublicIp(ip_id string) (*PublicIp, error) {
	result := new(PublicIp)
	url := createUrl(api, publicIpPathSegment, ip_id)
	err := api.Client.Delete(url, nil, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /public_ips/{id}
func (api *API) UpdatePublicIp(ip_id string, reverse_dns string) (*PublicIp, error) {
	result := new(PublicIp)
	url := createUrl(api, publicIpPathSegment, ip_id)
	req := struct {
		ReverseDns string `json:"reverse_dns,omitempty"`
	}{reverse_dns}
	err := api.Client.Put(url, &req, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

func (ip *PublicIp) GetState() (string, error) {
	in, err := ip.api.GetPublicIp(ip.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
