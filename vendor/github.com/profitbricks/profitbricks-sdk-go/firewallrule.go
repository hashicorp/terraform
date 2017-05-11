package profitbricks

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type FirewallRule struct {
	Id         string                     `json:"id,omitempty"`
	Type_      string                     `json:"type,omitempty"`
	Href       string                     `json:"href,omitempty"`
	Metadata   *DatacenterElementMetadata `json:"metadata,omitempty"`
	Properties FirewallruleProperties     `json:"properties,omitempty"`
	Response   string                     `json:"Response,omitempty"`
	Headers    *http.Header               `json:"headers,omitempty"`
	StatusCode int                        `json:"headers,omitempty"`
}

type FirewallruleProperties struct {
	Name           string `json:"name,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	SourceMac      string `json:"sourceMac,omitempty"`
	SourceIp       string `json:"sourceIp,omitempty"`
	TargetIp       string `json:"targetIp,omitempty"`
	IcmpCode       interface{}    `json:"icmpCode,omitempty"`
	IcmpType       interface{}    `json:"icmpType,omitempty"`
	PortRangeStart interface{}    `json:"portRangeStart,omitempty"`
	PortRangeEnd   interface{}    `json:"portRangeEnd,omitempty"`
}

type FirewallRules struct {
	Id         string         `json:"id,omitempty"`
	Type_      string         `json:"type,omitempty"`
	Href       string         `json:"href,omitempty"`
	Items      []FirewallRule `json:"items,omitempty"`
	Response   string         `json:"Response,omitempty"`
	Headers    *http.Header   `json:"headers,omitempty"`
	StatusCode int            `json:"headers,omitempty"`
}

func ListFirewallRules(dcId string, serverid string, nicId string) FirewallRules {
	path := fwrule_col_path(dcId, serverid, nicId)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toFirewallRules(resp)
}

func GetFirewallRule(dcid string, srvid string, nicId string, fwId string) FirewallRule {
	path := fwrule_path(dcid, srvid, nicId, fwId)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", FullHeader)
	resp := do(req)
	return toFirewallRule(resp)
}

func CreateFirewallRule(dcid string, srvid string, nicId string, fw FirewallRule) FirewallRule {
	path := fwrule_col_path(dcid, srvid, nicId)
	url := mk_url(path) + `?depth=` + Depth
	obj, _ := json.Marshal(fw)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(obj))
	req.Header.Add("Content-Type", FullHeader)
	return toFirewallRule(do(req))
}

func PatchFirewallRule(dcid string, srvid string, nicId string, fwId string, obj FirewallruleProperties) FirewallRule {
	jason_patch := []byte(MkJson(obj))
	path := fwrule_path(dcid, srvid, nicId, fwId)
	url := mk_url(path) + `?depth=` + Depth
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(jason_patch))
	req.Header.Add("Content-Type", PatchHeader)
	return toFirewallRule(do(req))
}

func DeleteFirewallRule(dcid string, srvid string, nicId string, fwId string) Resp {
	path := fwrule_path(dcid, srvid, nicId, fwId)
	return is_delete(path)
}

func toFirewallRule(resp Resp) FirewallRule {
	var dc FirewallRule
	json.Unmarshal(resp.Body, &dc)
	dc.Response = string(resp.Body)
	dc.Headers = &resp.Headers
	dc.StatusCode = resp.StatusCode
	return dc
}

func toFirewallRules(resp Resp) FirewallRules {
	var col FirewallRules
	json.Unmarshal(resp.Body, &col)
	col.Response = string(resp.Body)
	col.Headers = &resp.Headers
	col.StatusCode = resp.StatusCode
	return col
}
