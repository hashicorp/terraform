package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Lan struct {
	Id         string                     `json:"id,omitempty"`
	Type_      string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties LanProperties              `json:"properties,omitempty"`
	Entities   *LanEntities               `json:"entities,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type LanProperties struct {
	Name   string      `json:"name,omitempty"`
	Public interface{} `json:"public,omitempty"`
}

type LanEntities struct {
	Nics *LanNics `json:"nics,omitempty"`
}

type LanNics struct {
	Id    string `json:"id,omitempty"`
	Type_ string `json:"type,omitempty"`
	Href  string `json:"href,omitempty"`
	Items []Nic  `json:"items,omitempty"`
}

type Lans struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Lan        `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

// ListLan returns a Collection for lans in the Datacenter
func ListLans(dcid string) Lans {
	path := lan_col_path(dcid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toLans(do(req))
}

// CreateLan creates a lan in the datacenter
// from a jason []byte and returns a Instance struct
func CreateLan(dcid string, request Lan) Lan {
	obj, _ := json.Marshal(request)
	path := lan_col_path(dcid)
	url := mk_url(path)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)
	return toLan(do(req))
}

// GetLan pulls data for the lan where id = lanid returns an Instance struct
func GetLan(dcid, lanid string) Lan {
	path := lan_path(dcid, lanid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toLan(do(req))
}

// PatchLan does a partial update to a lan using json from []byte jason
// returns a Instance struct
func PatchLan(dcid string, lanid string, obj LanProperties) Lan {
	jason := []byte(MkJson(obj))
	path := lan_path(dcid, lanid)
	url := mk_url(path)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jason))
	req.Header.Add("Content-Type", PatchHeader)
	return toLan(do(req))
}

// DeleteLan deletes a lan where id == lanid
func DeleteLan(dcid, lanid string) Resp {
	path := lan_path(dcid, lanid)
	return is_delete(path)
}

func toLan(resp Resp) Lan {
	var lan Lan
	json.Unmarshal(resp.Body, &lan)
	lan.Response = string(resp.Body)
	lan.Headers = &resp.Headers
	lan.StatusCode = resp.StatusCode
	return lan
}

func toLans(resp Resp) Lans {
	var col Lans
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
