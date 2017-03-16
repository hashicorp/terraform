package profitbricks

import (
	"encoding/json"
	"net/http"
)

type Location struct {
	Id         string                    `json:"id,omitempty"`
	Type_      string                    `json:"type,omitempty"`
	Href       string                    `json:"href,omitempty"`
	Metadata   DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties Properties                `json:"properties,omitempty"`
	Response   string                    `json:"Response,omitempty"`
	Headers    *http.Header              `json:"headers,omitempty"`
	StatusCode int                       `json:"headers,omitempty"`
}

type Locations struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Location   `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

type Properties struct {
	Name     string `json:"name,omitempty"`
	Features []string `json:"features,omitempty"`
}

// ListLocations returns location collection data
func ListLocations() Locations {
	url := mk_url(location_col_path()) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toLocations(do(req))
}

// GetLocation returns location data
func GetLocation(locid string) Location {
	url := mk_url(location_path(locid)) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toLocation(do(req))
}

func toLocation(resp Resp) Location {
	var obj Location
	json.Unmarshal(resp.Body, &obj)
	obj.Response = string(resp.Body)
	obj.Headers = &resp.Headers
	obj.StatusCode = resp.StatusCode
	return obj
}

func toLocations(resp Resp) Locations {
	var col Locations
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
