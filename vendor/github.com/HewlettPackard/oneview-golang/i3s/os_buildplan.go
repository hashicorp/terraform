/*
(c) Copyright [2015] Hewlett Packard Enterprise Development LP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package i3s

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type OSBuildPlan struct {
	BuildPlanID        string              `json:"buildPlanid,omitempty"`        // "buildPlanid": "1",
	BuildSteps         []BuildStep         `json:"buildStep,omitempty"`          // "buildStep": [],
	Category           string              `json:"category,omitempty"`           // "category": "oe-deployment-plan",
	Created            string              `json:"created,omitempty"`            // "created": "20150831T154835.250Z",
	CustomAttributes   []CustomAttribute   `json:"customAttributes,omitempty"`   // "customAttributes": [],
	DependentArtifacts []DependentArtifact `json:"dependentArtifacts,omitempty"` // "dependentArtifacts": [],
	Description        string              `json:"description,omitempty"`        // "description": "Deployment Plan 1",
	OEBuildPlanETAG    string              `json:"eTag,omitempty"`               // "eTag": "1234",
	ETAG               string              `json:"etag,omitempty"`               // "etag": "1441036118675/8",
	HPProvided         bool                `json:"hpProvided,omitempty"`         // "hpProvided": false,
	Modified           string              `json:"modified,omitempty"`           // "modified": "20150831T154835.250Z",
	Name               string              `json:"name,omitempty"`               // "name": "Deployment Plan 1",
	OEBuildPlanType    string              `json:"oeBuildPlanType,omitempty"`    // "oeBuildPlanType": "type 1",
	Status             string              `json:"status,omitempty"`             // "status": "Critical",
	Type               string              `json:"type,omitempty"`               // "type": "OEDeploymentPlan",
	URI                utils.Nstring       `json:"uri,omitempty"`                // "uri": "/rest/deployment-plans/31e5dcba-b8ac-4f64-bbaa-7a4474f11994"
}

type OSBuildPlanList struct {
	Total       int           `json:"total,omitempty"`       // "total": 1,
	Count       int           `json:"count,omitempty"`       // "count": 1,
	Start       int           `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []OSBuildPlan `json:"members,omitempty"`     // "members":[]
}

type BuildStep struct {
	Parameters     string        `json:"parameters,omitempty"`     // "parameters": "anystring",
	PlanScriptName string        `json:"planScriptName,omitempty"` // "planScriptName": "script 1",
	PlanScriptURI  utils.Nstring `json:"planScriptUri,omitempty"`  // "planScriptUri": "/rest/plan-scripts/7dcd507c-09c4-48d0-b265-67f006b728ca",
	SerialNumber   string        `json:"serialNumber,omitempty"`   // "serialNumber": "1",
}

func (c *I3SClient) GetOSBuildPlanByName(name string) (OSBuildPlan, error) {
	var (
		osBuildPlan OSBuildPlan
	)
	osBuildPlans, err := c.GetOSBuildPlans(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if err != nil {
		return osBuildPlan, err
	}
	if osBuildPlans.Total > 0 {
		return osBuildPlans.Members[0], err
	} else {
		return osBuildPlan, err
	}
}

func (c *I3SClient) GetOSBuildPlans(filter string, sort string) (OSBuildPlanList, error) {
	var (
		uri          = "/rest/build-plans"
		q            map[string]interface{}
		osBuildPlans OSBuildPlanList
	)
	q = make(map[string]interface{})
	if len(filter) > 0 {
		q["filter"] = filter
	}

	if sort != "" {
		q["sort"] = sort
	}

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	// Setup query
	if len(q) > 0 {
		c.SetQueryString(q)
	}

	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return osBuildPlans, err
	}

	log.Debugf("GetOSBuildPlans %s", data)
	if err := json.Unmarshal([]byte(data), &osBuildPlans); err != nil {
		return osBuildPlans, err
	}
	return osBuildPlans, nil
}

func (c *I3SClient) CreateOSBuildPlan(osBuildPlan OSBuildPlan) error {
	log.Infof("Initializing creation of osBuildPlan for %s.", osBuildPlan.Name)
	var (
		uri = "/rest/build-plans"
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	log.Debugf("REST : %s \n %+v\n", uri, osBuildPlan)
	_, err := c.RestAPICall(rest.POST, uri, osBuildPlan)
	if err != nil {
		log.Errorf("Error submitting new os build plan request: %s", err)
		return err
	}

	return nil
}

func (c *I3SClient) DeleteOSBuildPlan(name string) error {
	var (
		osBuildPlan OSBuildPlan
		err         error
		uri         string
	)

	osBuildPlan, err = c.GetOSBuildPlanByName(name)
	if err != nil {
		return err
	}
	if osBuildPlan.Name != "" {
		log.Debugf("REST : %s \n %+v\n", osBuildPlan.URI, osBuildPlan)
		uri = osBuildPlan.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			return err
		}
		_, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete os build plan request: %s", err)
			return err
		}

		return nil
	} else {
		log.Infof("OS Build Plan could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *I3SClient) UpdateOSBuildPlan(osBuildPlan OSBuildPlan) error {
	log.Infof("Initializing update of os build plan for %s.", osBuildPlan.Name)
	var (
		uri = osBuildPlan.URI.String()
		t   *Task
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, osBuildPlan)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, osBuildPlan)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update os buildplan request: %s", err)
		return err
	}

	log.Debugf("Response update os build plan %s", data)
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		t.TaskIsDone = true
		log.Errorf("Error with task un-marshal: %s", err)
		return err
	}

	err = t.Wait()
	if err != nil {
		return err
	}

	return nil
}
