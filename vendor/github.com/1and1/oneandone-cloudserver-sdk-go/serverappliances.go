package oneandone

import "net/http"

type ServerAppliance struct {
	Identity
	typeField
	OsInstallBase string      `json:"os_installation_base,omitempty"`
	OsFamily      string      `json:"os_family,omitempty"`
	Os            string      `json:"os,omitempty"`
	OsVersion     string      `json:"os_version,omitempty"`
	Version       string      `json:"version,omitempty"`
	MinHddSize    int         `json:"min_hdd_size"`
	Architecture  interface{} `json:"os_architecture"`
	Licenses      interface{} `json:"licenses,omitempty"`
	Categories    []string    `json:"categories,omitempty"`
	//	AvailableDatacenters []string  `json:"available_datacenters,omitempty"`
	ApiPtr
}

// GET /server_appliances
func (api *API) ListServerAppliances(args ...interface{}) ([]ServerAppliance, error) {
	url, err := processQueryParams(createUrl(api, serverAppliancePathSegment), args...)
	if err != nil {
		return nil, err
	}
	res := []ServerAppliance{}
	err = api.Client.Get(url, &res, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range res {
		res[index].api = api
	}
	return res, nil
}

// GET /server_appliances/{id}
func (api *API) GetServerAppliance(sa_id string) (*ServerAppliance, error) {
	res := new(ServerAppliance)
	url := createUrl(api, serverAppliancePathSegment, sa_id)
	err := api.Client.Get(url, &res, http.StatusOK)
	if err != nil {
		return nil, err
	}
	//	res.api = api
	return res, nil
}
