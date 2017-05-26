package oneandone

import (
	"net/http"
)

type PrivateNetwork struct {
	Identity
	descField
	CloudpanelId   string      `json:"cloudpanel_id,omitempty"`
	NetworkAddress string      `json:"network_address,omitempty"`
	SubnetMask     string      `json:"subnet_mask,omitempty"`
	State          string      `json:"state,omitempty"`
	SiteId         string      `json:"site_id,omitempty"`
	CreationDate   string      `json:"creation_date,omitempty"`
	Servers        []Identity  `json:"servers,omitempty"`
	Datacenter     *Datacenter `json:"datacenter,omitempty"`
	ApiPtr
}

type PrivateNetworkRequest struct {
	Name           string `json:"name,omitempty"`
	Description    string `json:"description,omitempty"`
	DatacenterId   string `json:"datacenter_id,omitempty"`
	NetworkAddress string `json:"network_address,omitempty"`
	SubnetMask     string `json:"subnet_mask,omitempty"`
}

// GET /private_networks
func (api *API) ListPrivateNetworks(args ...interface{}) ([]PrivateNetwork, error) {
	url, err := processQueryParams(createUrl(api, privateNetworkPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []PrivateNetwork{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /private_networks
func (api *API) CreatePrivateNetwork(request *PrivateNetworkRequest) (string, *PrivateNetwork, error) {
	result := new(PrivateNetwork)
	url := createUrl(api, privateNetworkPathSegment)
	err := api.Client.Post(url, &request, &result, http.StatusAccepted)
	if err != nil {
		return "", nil, err
	}
	result.api = api
	return result.Id, result, nil
}

// GET /private_networks/{id}
func (api *API) GetPrivateNetwork(pn_id string) (*PrivateNetwork, error) {
	result := new(PrivateNetwork)
	url := createUrl(api, privateNetworkPathSegment, pn_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /private_networks/{id}
func (api *API) UpdatePrivateNetwork(pn_id string, request *PrivateNetworkRequest) (*PrivateNetwork, error) {
	result := new(PrivateNetwork)
	url := createUrl(api, privateNetworkPathSegment, pn_id)
	err := api.Client.Put(url, &request, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /private_networks/{id}
func (api *API) DeletePrivateNetwork(pn_id string) (*PrivateNetwork, error) {
	result := new(PrivateNetwork)
	url := createUrl(api, privateNetworkPathSegment, pn_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /private_networks/{id}/servers
func (api *API) ListPrivateNetworkServers(pn_id string) ([]Identity, error) {
	result := []Identity{}
	url := createUrl(api, privateNetworkPathSegment, pn_id, "servers")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /private_networks/{id}/servers
func (api *API) AttachPrivateNetworkServers(pn_id string, sids []string) (*PrivateNetwork, error) {
	result := new(PrivateNetwork)
	req := servers{
		Servers: sids,
	}
	url := createUrl(api, privateNetworkPathSegment, pn_id, "servers")
	err := api.Client.Post(url, &req, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /private_networks/{id}/servers/{id}
func (api *API) GetPrivateNetworkServer(pn_id string, server_id string) (*Identity, error) {
	result := new(Identity)
	url := createUrl(api, privateNetworkPathSegment, pn_id, "servers", server_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /private_networks/{id}/servers/{id}
func (api *API) DetachPrivateNetworkServer(pn_id string, pns_id string) (*PrivateNetwork, error) {
	result := new(PrivateNetwork)
	url := createUrl(api, privateNetworkPathSegment, pn_id, "servers", pns_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

func (pn *PrivateNetwork) GetState() (string, error) {
	in, err := pn.api.GetPrivateNetwork(pn.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
