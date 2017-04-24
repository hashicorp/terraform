package oneandone

import "net/http"

// Struct to describe a ISO image that can be used to boot a server.
//
// Values of this type describe ISO images that can be inserted into the servers virtual DVD drive.
//
//
type DvdIso struct {
	Identity
	OsFamily             string      `json:"os_family,omitempty"`
	Os                   string      `json:"os,omitempty"`
	OsVersion            string      `json:"os_version,omitempty"`
	Type                 string      `json:"type,omitempty"`
	AvailableDatacenters []string    `json:"available_datacenters,omitempty"`
	Architecture         interface{} `json:"os_architecture,omitempty"`
	ApiPtr
}

// GET /dvd_isos
func (api *API) ListDvdIsos(args ...interface{}) ([]DvdIso, error) {
	url, err := processQueryParams(createUrl(api, dvdIsoPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []DvdIso{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// GET /dvd_isos/{id}
func (api *API) GetDvdIso(dvd_id string) (*DvdIso, error) {
	result := new(DvdIso)
	url := createUrl(api, dvdIsoPathSegment, dvd_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}
