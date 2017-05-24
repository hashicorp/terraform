package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type Datacenter struct {
	Id         string                     `json:"id,omitempty"`
	Type_      string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties DatacenterProperties       `json:"properties,omitempty"`
	Entities   DatacenterEntities         `json:"entities,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type DatacenterElementMetadata struct {
	CreatedDate      time.Time `json:"createdDate,omitempty"`
	CreatedBy        string    `json:"createdBy,omitempty"`
	Etag             string    `json:"etag,omitempty"`
	LastModifiedDate time.Time `json:"lastModifiedDate,omitempty"`
	LastModifiedBy   string    `json:"lastModifiedBy,omitempty"`
	State            string    `json:"state,omitempty"`
}

type DatacenterProperties struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Version     int32  `json:"version,omitempty"`
}

type DatacenterEntities struct {
	Servers       *Servers       `json:"servers,omitempty"`
	Volumes       *Volumes       `json:"volumes,omitempty"`
	Loadbalancers *Loadbalancers `json:"loadbalancers,omitempty"`
	Lans          *Lans          `json:"lans,omitempty"`
}

type Datacenters struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Datacenter `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

func ListDatacenters() Datacenters {
	path := dc_col_path()
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toDataCenters(resp)
}

func CreateDatacenter(dc Datacenter) Datacenter {
	obj, _ := json.Marshal(dc)
	path := dc_col_path()
	url := mk_url(path)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)

	return toDataCenter(do(req))
}

func CompositeCreateDatacenter(datacenter Datacenter) Datacenter {
	obj, _ := json.Marshal(datacenter)
	path := dc_col_path()
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)
	return toDataCenter(do(req))
}

func GetDatacenter(dcid string) Datacenter {
	path := dc_path(dcid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toDataCenter(do(req))
}

func PatchDatacenter(dcid string, obj DatacenterProperties) Datacenter {
	jason_patch := []byte(MkJson(obj))
	path := dc_path(dcid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jason_patch))
	req.Header.Add("Content-Type", PatchHeader)
	return toDataCenter(do(req))
}

func DeleteDatacenter(dcid string) Resp {
	path := dc_path(dcid)
	return is_delete(path)
}

func toDataCenter(resp Resp) Datacenter {
	var dc Datacenter
	json.Unmarshal(resp.Body, &dc)
	dc.Response = string(resp.Body)
	dc.Headers = &resp.Headers
	dc.StatusCode = resp.StatusCode
	return dc
}

func toDataCenters(resp Resp) Datacenters {
	var col Datacenters
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
