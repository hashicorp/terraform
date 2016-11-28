package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type ServerHardwareType struct {
	Category    string        `json:"category,omitempty"`    // "category": "server-hardware",
	Created     string        `json:"created,omitempty"`     // "created": "20150831T154835.250Z",
	Description string        `json:"description,omitempty"` // "description": "ServerHardware",
	ETAG        string        `json:"eTag,omitempty"`        // "eTag": "1441036118675/8",
	Modified    string        `json:"modified,omitempty"`    // "modified": "20150831T154835.250Z",
	Name        string        `json:"name,omitempty"`        // "name": "ServerHardware 1",
	Type        string        `json:"type,omitempty"`        // "type": "server-hardware-type-4",
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/server-hardware-types/e2f0031b-52bd-4223-9ac1-d91cb519d548"
}

type ServerHardwareTypeList struct {
	Total       int                  `json:"total,omitempty"`       // "total": 1,
	Count       int                  `json:"count,omitempty"`       // "count": 1,
	Start       int                  `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring        `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring        `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring        `json:"uri,omitempty"`         // "uri": "/rest/server-hardware-types?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []ServerHardwareType `json:"members,omitempty"`     // "members":[]
}

func (c *OVClient) GetServerHardwareTypeByName(name string) (ServerHardwareType, error) {
	var (
		serverHardwareType ServerHardwareType
	)
	serverHardwareTypes, err := c.GetServerHardwareTypes(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if serverHardwareTypes.Total > 0 {
		return serverHardwareTypes.Members[0], err
	} else {
		return serverHardwareType, fmt.Errorf("Could not find Server Hardware Type: %s", name)
	}
}

func (c *OVClient) GetServerHardwareTypeByUri(uri utils.Nstring) (ServerHardwareType, error) {
	var (
		serverHardwareType ServerHardwareType
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri.String(), nil)
	if err != nil {
		return serverHardwareType, err
	}
	log.Debugf("GetServerHardwareType %s", data)
	if err := json.Unmarshal([]byte(data), &serverHardwareType); err != nil {
		return serverHardwareType, err
	}
	return serverHardwareType, nil
}

func (c *OVClient) GetServerHardwareTypes(filter string, sort string) (ServerHardwareTypeList, error) {
	var (
		uri                 = "/rest/server-hardware-types"
		q                   map[string]interface{}
		serverHardwareTypes ServerHardwareTypeList
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
		return serverHardwareTypes, err
	}

	log.Debugf("GetServerHardwareTypes %s", data)
	if err := json.Unmarshal([]byte(data), &serverHardwareTypes); err != nil {
		return serverHardwareTypes, err
	}
	return serverHardwareTypes, nil
}
