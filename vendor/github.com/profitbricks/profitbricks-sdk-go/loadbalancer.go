package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Loadbalancer struct {
	Id         string                     `json:"id,omitempty"`
	Type_      string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties LoadbalancerProperties     `json:"properties,omitempty"`
	Entities   LoadbalancerEntities       `json:"entities,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type LoadbalancerProperties struct {
	Name string `json:"name,omitempty"`
	Ip   string `json:"ip,omitempty"`
	Dhcp bool   `json:"dhcp,omitempty"`
}

type LoadbalancerEntities struct {
	Balancednics *BalancedNics `json:"balancednics,omitempty"`
}

type BalancedNics struct {
	Id    string `json:"id,omitempty"`
	Type_ string `json:"type,omitempty"`
	Href  string `json:"href,omitempty"`
	Items []Nic  `json:"items,omitempty"`
}

type Loadbalancers struct {
	Id    string         `json:"id,omitempty"`
	Type_ string         `json:"type,omitempty"`
	Href  string         `json:"href,omitempty"`
	Items []Loadbalancer `json:"items,omitempty"`

	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

type LoablanacerCreateRequest struct {
	LoadbalancerProperties `json:"properties"`
}

// Listloadbalancers returns a Collection struct
// for loadbalancers in the Datacenter
func ListLoadbalancers(dcid string) Loadbalancers {
	path := lbal_col_path(dcid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toLoadbalancers(do(req))
}

// Createloadbalancer creates a loadbalancer in the datacenter
//from a jason []byte and returns a Instance struct
func CreateLoadbalancer(dcid string, request Loadbalancer) Loadbalancer {
	obj, _ := json.Marshal(request)
	path := lbal_col_path(dcid)
	url := mk_url(path)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)
	return toLoadbalancer(do(req))
}

// GetLoadbalancer pulls data for the Loadbalancer
// where id = lbalid returns a Instance struct
func GetLoadbalancer(dcid, lbalid string) Loadbalancer {
	path := lbal_path(dcid, lbalid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toLoadbalancer(do(req))
}

func PatchLoadbalancer(dcid string, lbalid string, obj LoadbalancerProperties) Loadbalancer {
	jason := []byte(MkJson(obj))
	path := lbal_path(dcid, lbalid)
	url := mk_url(path)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jason))
	req.Header.Add("Content-Type", PatchHeader)
	return toLoadbalancer(do(req))
}

func DeleteLoadbalancer(dcid, lbalid string) Resp {
	path := lbal_path(dcid, lbalid)
	return is_delete(path)
}

func ListBalancedNics(dcid, lbalid string) Nics {
	path := balnic_col_path(dcid, lbalid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toNics(do(req))
}

func AssociateNic(dcid string, lbalid string, nicid string) Nic {
	sm := map[string]string{"id": nicid}
	jason := []byte(MkJson(sm))
	path := balnic_col_path(dcid, lbalid)
	url := mk_url(path)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jason))
	req.Header.Add("Content-Type", FullHeader)
	return toNic(do(req))
}

func GetBalancedNic(dcid, lbalid, balnicid string) Nic {
	path := balnic_path(dcid, lbalid, balnicid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toNic(do(req))
}

func DeleteBalancedNic(dcid, lbalid, balnicid string) Resp {
	path := balnic_path(dcid, lbalid, balnicid)
	return is_delete(path)
}

func toLoadbalancer(resp Resp) Loadbalancer {
	var server Loadbalancer
	json.Unmarshal(resp.Body, &server)
	server.Response = string(resp.Body)
	server.Headers = &resp.Headers
	server.StatusCode = resp.StatusCode
	return server
}

func toLoadbalancers(resp Resp) Loadbalancers {
	var col Loadbalancers
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
