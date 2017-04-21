package oneandone

import "net/http"

type Datacenter struct {
	idField
	CountryCode string `json:"country_code,omitempty"`
	Location    string `json:"location,omitempty"`
}

// GET /datacenters
func (api *API) ListDatacenters(args ...interface{}) ([]Datacenter, error) {
	url, err := processQueryParams(createUrl(api, datacenterPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []Datacenter{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GET /datacenters/{datacenter_id}
func (api *API) GetDatacenter(dc_id string) (*Datacenter, error) {
	result := new(Datacenter)
	url := createUrl(api, datacenterPathSegment, dc_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return result, nil
}
