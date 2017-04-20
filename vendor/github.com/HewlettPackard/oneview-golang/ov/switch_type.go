package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type SwitchType struct {
	Name string        `json:"name,omitempty"` // "name": "Ethernet Network 1",
	URI  utils.Nstring `json:"uri"`            //"uri": "/rest/switch-types/a2bc8f42-8bb8-4560-b80f-6c3c0e0d66e0"
}

type SwitchTypeList struct {
	Total       int           `json:"total,omitempty"`       // "total": 1,
	Count       int           `json:"count,omitempty"`       // "count": 1,
	Start       int           `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []SwitchType  `json:"members,omitempty"`     // "members":[]
}

func (c *OVClient) GetSwitchTypeByName(name string) (SwitchType, error) {
	var (
		switchType SwitchType
	)
	switchTypes, err := c.GetSwitchTypes(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if switchTypes.Total > 0 {
		return switchTypes.Members[0], err
	} else {
		return switchType, err
	}
}

func (c *OVClient) GetSwitchTypes(filter string, sort string) (SwitchTypeList, error) {
	var (
		uri         = "/rest/switch-types"
		q           map[string]interface{}
		switchTypes SwitchTypeList
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
		return switchTypes, err
	}

	log.Debugf("GetSwitchTypes %s", data)
	if err := json.Unmarshal([]byte(data), &switchTypes); err != nil {
		return switchTypes, err
	}
	return switchTypes, nil
}
