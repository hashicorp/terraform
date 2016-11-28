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

type PlanScript struct {
	Category           string              `json:"category,omitempty"`           // "category": "PlanScript",
	Content            string              `json:"content,omitempty"`            // "content": "f"
	Created            string              `json:"created,omitempty"`            // "created": "20150831T154835.250Z",
	CustomAttributes   string              `json:"customAttributes,omitempty"`   // "customAttributes": "",
	DependentArtifacts []DependentArtifact `json:"dependentArtifacts,omitempty"` // "dependentArtifacts": [],
	Description        string              `json:"description,omitempty"`        // "description": "PlanScript 1",
	ETAG               string              `json:"eTag,omitempty"`               // "eTag": "1441036118675/8",
	HPProvided         bool                `json:"hpProvided,omitempty"`         // "hpProvided": false,
	ID                 string              `json:"id,omitempty"`                 // "id": "1",
	Modified           string              `json:"modified,omitempty"`           // "modified": "20150831T154835.250Z",
	Name               string              `json:"name,omitempty"`               // "name": "PlanScript 1",
	PlanType           string              `json:"planType,omitempty"`           // "planType": "Deploy",
	State              string              `json:"state,omitempty"`              // "state": "Normal",
	Status             string              `json:"status,omitempty"`             // "status": "Critical",
	Type               string              `json:"type,omitempty"`               // "type": "OEDeploymentPlan",
	URI                utils.Nstring       `json:"uri,omitempty"`                // "uri": "/rest/plan-scripts/31e5dcba-b8ac-4f64-bbaa-7a4474f11994"
}

type PlanScriptList struct {
	Total       int           `json:"total,omitempty"`       // "total": 1,
	Count       int           `json:"count,omitempty"`       // "count": 1,
	Start       int           `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []PlanScript  `json:"members,omitempty"`     // "members":[]
}

func (c *I3SClient) GetPlanScriptByName(name string) (PlanScript, error) {
	var (
		planScript PlanScript
	)
	planScripts, err := c.GetPlanScripts(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if err != nil {
		return planScript, err
	}
	if planScripts.Total > 0 {
		return planScripts.Members[0], err
	} else {
		return planScript, err
	}
}

func (c *I3SClient) GetPlanScripts(filter string, sort string) (PlanScriptList, error) {
	var (
		uri         = "/rest/plan-scripts"
		q           map[string]interface{}
		planScripts PlanScriptList
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
		return planScripts, err
	}

	log.Debugf("GetPlanScripts %s", data)
	if err := json.Unmarshal([]byte(data), &planScripts); err != nil {
		return planScripts, err
	}
	return planScripts, nil
}

func (c *I3SClient) CreatePlanScript(planScript PlanScript) error {
	log.Infof("Initializing creation of plan script for %s.", planScript.Name)
	var (
		uri                 = "/rest/plan-scripts"
		attemptedPlanScript *PlanScript
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.POST, uri, planScript)
	if err != nil {
		log.Errorf("Error submitting new plan script request: %s", err)
		return err
	}

	log.Debugf("Response New Plan Script %s", data)
	if err := json.Unmarshal([]byte(data), &attemptedPlanScript); err != nil {
		log.Errorf("Error with task un-marshal: %s", err)
		return err
	}

	if attemptedPlanScript.URI == "" {
		return fmt.Errorf("PlanScript not succesfully created")
	}

	return nil
}

func (c *I3SClient) DeletePlanScript(name string) error {
	var (
		planScript PlanScript
		err        error
		uri        string
	)

	planScript, err = c.GetPlanScriptByName(name)
	if err != nil {
		return err
	}
	if planScript.Name != "" {
		log.Debugf("REST : %s \n %+v\n", planScript.URI, planScript)
		uri = planScript.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			return err
		}
		_, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete plan script request: %s", err)
			return err
		}
	} else {
		log.Infof("Plan script could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *I3SClient) UpdatePlanScript(planScript PlanScript) error {
	log.Infof("Initializing update of plan script for %s.", planScript.Name)
	var (
		uri = planScript.URI.String()
		t   *Task
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, planScript)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, planScript)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update plan script request: %s", err)
		return err
	}

	log.Debugf("Response update PlanScript %s", data)
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
