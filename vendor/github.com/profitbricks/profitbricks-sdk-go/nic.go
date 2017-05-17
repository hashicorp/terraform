package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Nic struct {
	Id         string                     `json:"id,omitempty"`
	Type_      string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties NicProperties              `json:"properties,omitempty"`
	Entities   *NicEntities               `json:"entities,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type NicProperties struct {
	Name           string   `json:"name,omitempty"`
	Mac            string   `json:"mac,omitempty"`
	Ips            []string `json:"ips,omitempty"`
	Dhcp           bool     `json:"dhcp"`
	Lan            int      `json:"lan,omitempty"`
	FirewallActive bool     `json:"firewallActive,omitempty"`
	Nat            bool     `json:"nat,omitempty"`
}

type NicEntities struct {
	Firewallrules *FirewallRules `json:"firewallrules,omitempty"`
}

type Nics struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []Nic        `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

type NicCreateRequest struct {
	NicProperties `json:"properties"`
}

// ListNics returns a Nics struct collection
func ListNics(dcid, srvid string) Nics {
	path := nic_col_path(dcid, srvid) + `?depth=` + Depth
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toNics(do(req))
}

// CreateNic creates a nic on a server
// from a jason []byte and returns a Instance struct
func CreateNic(dcid string, srvid string, request Nic) Nic {
	obj, _ := json.Marshal(request)
	path := nic_col_path(dcid, srvid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)
	return toNic(do(req))
}

// GetNic pulls data for the nic where id = srvid returns a Instance struct
func GetNic(dcid, srvid, nicid string) Nic {
	path := nic_path(dcid, srvid, nicid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toNic(do(req))
}

// PatchNic partial update of nic properties passed in as jason []byte
// Returns Instance struct
func PatchNic(dcid string, srvid string, nicid string, obj NicProperties) Nic {
	jason := []byte(MkJson(obj))
	path := nic_path(dcid, srvid, nicid)
	url := mk_url(path)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jason))
	req.Header.Add("Content-Type", PatchHeader)
	return toNic(do(req))
}

// DeleteNic deletes the nic where id=nicid and returns a Resp struct
func DeleteNic(dcid, srvid, nicid string) Resp {
	path := nic_path(dcid, srvid, nicid)
	return is_delete(path)
}

func toNic(resp Resp) Nic {
	var obj Nic
	json.Unmarshal(resp.Body, &obj)
	obj.Response = string(resp.Body)
	obj.Headers = &resp.Headers
	obj.StatusCode = resp.StatusCode
	return obj
}

func toNics(resp Resp) Nics {
	var col Nics
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
