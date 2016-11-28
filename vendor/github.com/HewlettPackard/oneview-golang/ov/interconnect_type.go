package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type InterconnectType struct {
	Category                 string                 `json:"category,omitempty"`                 // "category": "interconnect-types",
	Created                  string                 `json:"created,omitempty"`                  // "created": "20150831T154835.250Z",
	Description              string                 `json:"description,omitempty"`              // "description": "Interconnect Type 1",
	DownlinkCount            int                    `json:"downlinkCount,omitempty"`            // "downlinkCount": 2,
	DownlinkPortCapability   DownlinkPortCapability `json:"downlinkPortCapability,omitempty"`   // "downlinkPortCapability": {...},
	ETAG                     string                 `json:"eTag,omitempty"`                     // "eTag": "1441036118675/8",
	InterconnectCapabilities InterconnectCapability `json:"interconnectCapabilities,omitempty"` // "interconnectCapabilities": {...},
	MaximumFirmwareVersion   string                 `json:"maximumFirmwareVersion,omitempty"`   // "maximumFirmwareVersion": "3.0.0",
	MinimumFirmwareVersion   string                 `json:"minimumFirmwareVersion,omitempty"`   // "minimumFirmwareVersion": "2.0.0",
	Modified                 string                 `json:"modified,omitempty"`                 // "modified": "20150831T154835.250Z",
	Name                     utils.Nstring          `json:"name,omitempty"`                     // "name": null,
	PartNumber               string                 `json:"partNumber,omitempty"`               // "partNumber": "572018-B21",
	PortInfos                []PortInfo             `json:"portInfos,omitempty"`                // "portInfos": {...},
	State                    string                 `json:"state,omitempty"`                    // "state": "Normal",
	Status                   string                 `json:"status,omitempty"`                   // "status": "Critical",
	Type                     string                 `json:"type,omitempty"`                     // "type": "interconnect-typeV3",
	UnsupportedCapabilities  []string               `json:"unsupportedCapabilities,omitempty"`  // "unsupportedCapabilities": [],
	URI                      utils.Nstring          `json:"uri,omitempty"`                      // "uri": "/rest/interconnect-types/9d31081c-e010-4005-bf0b-e64b0ca04af5"
}

type DownlinkPortCapability struct {
	Category           utils.Nstring          `json:"category,omitempty"`           // "category": null,
	Created            string                 `json:"created,omitempty"`            // "created": "20150831T154835.250Z",
	Description        string                 `json:"description,omitempty"`        // "description": "Downlink Port Capability",
	DownlinkSubPorts   map[string]interface{} `json:"downlinkSubPorts,omitempty"`   // "downlinkSubPorts": null,
	ETAG               string                 `json:"eTag,omitempty"`               // "eTag": "1441036118675/8",
	MaxBandwidthInGbps int                    `json:"maxBandwidthInGbps,omitempty"` // "maxBandwidthInGbps": 10,
	Modified           string                 `json:"modified,omitempty"`           // "modified": "20150831T154835.250Z",
	Name               utils.Nstring          `json:"name,omitempty"`               // "name": null,
	PortCapabilities   []string               `json:"portCapabilities,omitempty"`   //"portCapabilites":  ["ConnectionReservation","FibreChannel","ConnectionDeployment"],
	State              string                 `json:"state,omitempty"`              // "state": "Normal",
	Status             string                 `json:"status,omitempty"`             // "status": "Critical",
	TotalSubPort       int                    `json:"totalSubPort,omitempty"`       // "totalSubPort": 1,
	Type               string                 `json:"type,omitempty"`               // "type": "downlink-port-capability",
	URI                utils.Nstring          `json:"uri,omitempty"`                // "uri": "null"
}

type InterconnectCapability struct {
	Capabilities       []string `json:"capabilities,omitempty"`       // "capabilities": ["Ethernet"],
	MaxBandwidthInGbps int      `json:"maxBandwidthInGbps,omitempty"` // "maxBandwidthInGbps": 10,
}

type PortInfo struct {
	DownlinkCapable  bool          `json:"downlinkCapable,omitempty"` // "downlinkCapable": true,
	PairedPortName   utils.Nstring `json:"pairedPortName,omitempty"`  // "pairedPortName": null,
	PortCapabilities []string      `json:"portCapabilites,omitempty"` // "portCapabilities":  ["ConnectionReservation","FibreChannel","ConnectionDeployment"],
	PortName         string        `json:"portName,omitempty"`        // "portName": "4",
	PortNumber       int           `json:"portNumber,omitempty"`      // "portNumber": 20,
	UplinkCapable    bool          `json:"uplinkCapable,omitempty"`   // "uplinkCapable": true,
}

type InterconnectTypeList struct {
	Total       int                `json:"total,omitempty"`       // "total": 1,
	Count       int                `json:"count,omitempty"`       // "count": 1,
	Start       int                `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring      `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring      `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring      `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []InterconnectType `json:"members,omitempty"`     // "members":[]
}

func (c *OVClient) GetInterconnectTypeByName(name string) (InterconnectType, error) {
	var (
		interconnectType InterconnectType
	)
	interconnectTypes, err := c.GetInterconnectTypes(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if interconnectTypes.Total > 0 {
		return interconnectTypes.Members[0], err
	} else {
		return interconnectType, err
	}
}

func (c *OVClient) GetInterconnectTypeByUri(uri utils.Nstring) (InterconnectType, error) {
	var (
		interconnectType InterconnectType
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri.String(), nil)
	if err != nil {
		return interconnectType, err
	}
	log.Debugf("GetInterconnectType %s", data)
	if err := json.Unmarshal([]byte(data), &interconnectType); err != nil {
		return interconnectType, err
	}
	return interconnectType, nil
}

func (c *OVClient) GetInterconnectTypes(filter string, sort string) (InterconnectTypeList, error) {
	var (
		uri               = "/rest/interconnect-types"
		q                 map[string]interface{}
		interconnectTypes InterconnectTypeList
	)
	q = make(map[string]interface{})
	if len(filter) > 0 {
		q["filter"] = filter
	}

	if sort != "" {
		q["sort"] = sort
	}

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	// Setup query
	if len(q) > 0 {
		c.SetQueryString(q)
	}
	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return interconnectTypes, err
	}

	log.Debugf("GetInterconnectTypes %s", data)
	if err := json.Unmarshal([]byte(data), &interconnectTypes); err != nil {
		return interconnectTypes, err
	}
	return interconnectTypes, nil
}
