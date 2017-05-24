package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type IpBlock struct {
	Id         string                     `json:"id,omitempty"`
	Type_      string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties IpBlockProperties          `json:"properties,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type IpBlockProperties struct {
	Ips      []string `json:"ips,omitempty"`
	Location string   `json:"location,omitempty"`
	Size     int      `json:"size,omitempty"`
}

type IpBlocks struct {
	Id         string       `json:"id,omitempty"`
	Type_      string       `json:"type,omitempty"`
	Href       string       `json:"href,omitempty"`
	Items      []IpBlock    `json:"items,omitempty"`
	Response   string       `json:"Response,omitempty"`
	Headers    *http.Header `json:"headers,omitempty"`
	StatusCode int          `json:"headers,omitempty"`
}

// ListIpBlocks
func ListIpBlocks() IpBlocks {
	path := ipblock_col_path()
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toIpBlocks(do(req))
}

func ReserveIpBlock(request IpBlock) IpBlock {
	obj, _ := json.Marshal(request)
	path := ipblock_col_path()
	url := mk_url(path)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)
	return toIpBlock(do(req))
}
func GetIpBlock(ipblockid string) IpBlock {
	path := ipblock_path(ipblockid)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	return toIpBlock(do(req))
}

func ReleaseIpBlock(ipblockid string) Resp {
	path := ipblock_path(ipblockid)
	return is_delete(path)
}

func toIpBlock(resp Resp) IpBlock {
	var obj IpBlock
	json.Unmarshal(resp.Body, &obj)
	obj.Response = string(resp.Body)
	obj.Headers = &resp.Headers
	obj.StatusCode = resp.StatusCode
	return obj
}

func toIpBlocks(resp Resp) IpBlocks {
	var col IpBlocks
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
