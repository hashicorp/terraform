package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type OSDeploymentPlan struct {
	AdditionalParameters    []CustomAttribute `json:"additionalParameters,omitempty"`    // "additionalParameters": [],
	Architecture            string            `json:"architecture,omitempty"`            // "architecture": "",
	Category                string            `json:"category,omitempty"`                // "category": "os-deployment-plans",
	Created                 string            `json:"created,omitempty"`                 // "created": "20150831T154835.250Z",
	DeploymentAppliance     string            `json:"deploymentAppliance,omitempty"`     // "deploymentAppliance": "ivp6:port",
	DeploymentApplianceIpv4 string            `json:"deploymentApplianceIpv4,omitempty"` // "deploymentApplianceIpv4": "255.255.255.0"
	DeploymentType          string            `json:"deploymentType,omitempty"`          // "deploymentType": "i3s",
	Description             utils.Nstring     `json:"description,omitempty"`             // "description": "",
	ETAG                    string            `json:"eTag,omitempty"`                    // "eTag": "1441036118675/8",
	Id                      string            `json:"id,omitempty"`                      // "id": "ca7d7f3d-668a-4f56-9f83-ab491470a50a",
	Modified                string            `json:"modified,omitempty"`                // "modified": "20150831T154835.250Z",
	Name                    string            `json:"name,omitempty"`                    // "name": "osdp 1",
	NativePlanUri           string            `json:"nativePlanUri,omitempty"`           // "nativePlanUri": "/rest/deployment-plans/611f051a-3588-4b42-bf93-9fc70e034710",
	OsType                  string            `json:"osType,omitempty"`                  // "osType": "",
	OsdpSize                string            `json:"osdpSize,omitempty"`                // "osdpSize": "10GB",
	State                   string            `json:"state,omitempty"`                   // "state": "Normal",
	Status                  string            `json:"status,omitempty"`                  // "status": "Critical",
	Type                    string            `json:"type,omitempty"`                    // "type": "Osdp",
	URI                     utils.Nstring     `json:"uri,omitempty"`                     // "uri": "/rest/os-deployment-plans/ca7d7f3d-668a-4f56-9f83-ab491470a50a"
}

type OSDeploymentPlanList struct {
	Total       int                `json:"total,omitempty"`       // "total": 1,
	Count       int                `json:"count,omitempty"`       // "count": 1,
	Start       int                `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring      `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring      `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring      `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []OSDeploymentPlan `json:"members,omitempty"`     // "members":[]
}

type CustomAttribute struct {
	CaConstraints string `json:"caConstraints,omitempty"` // "caConstraints": "{}",
	CaEditable    bool   `json:"caEditable,omitempty"`    // "caEditable": true,
	CaID          string `json:"caId,omitempty"`          // "caId": "23436361-3a69-4ff0-9b97-85a2deb10822",
	CaType        string `json:"caType,omitempty"`        // "caType": "string",
	Description   string `json:"description,omitempty"`   // "description": "",
	Name          string `json:"name,omitempty"`          // "name": "attribute_name",
	Value         string `json:"value,omitempty"`         // "value": "attribute_value",
}

// get an os deployment plan with uri
func (c *OVClient) GetOSDeploymentPlan(uri utils.Nstring) (OSDeploymentPlan, error) {

	var osDeploymentPlan OSDeploymentPlan
	// refresh login

	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	// rest call
	data, err := c.RestAPICall(rest.GET, uri.String(), nil)
	if err != nil {
		return osDeploymentPlan, err
	}

	log.Debugf("GetOSDeploymentPlan %s", data)
	if err := json.Unmarshal([]byte(data), &osDeploymentPlan); err != nil {
		return osDeploymentPlan, err
	}
	return osDeploymentPlan, nil
}

func (c *OVClient) GetOSDeploymentPlanByName(name string) (OSDeploymentPlan, error) {
	var (
		osdp OSDeploymentPlan
	)
	osdps, err := c.GetOSDeploymentPlans(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if osdps.Total > 0 {
		return osdps.Members[0], err
	} else {
		return osdp, err
	}
}

func (c *OVClient) GetOSDeploymentPlans(filter string, sort string) (OSDeploymentPlanList, error) {
	var (
		uri   = "/rest/os-deployment-plans/"
		q     map[string]interface{}
		osdps OSDeploymentPlanList
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
		return osdps, err
	}

	log.Debugf("GetOSDeploymentPlans %s", data)
	if err := json.Unmarshal([]byte(data), &osdps); err != nil {
		return osdps, err
	}
	return osdps, nil
}
