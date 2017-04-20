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

type OSVolume struct {
	Category             string              `json:"category,omitempty"`             // "category": "os-volumes",
	Created              string              `json:"created,omitempty"`              // "created": "20150831T154835.250Z",
	DependentArtifacts   []DependentArtifact `json:"dependentArtifacts,omitempty"`   // "dependentArtifacts": [],
	DeploymentClusterUri utils.Nstring       `json:"deploymentClusterUri,omitempty"` // "deploymentClusterUri": "",
	Description          string              `json:"description,omitempty"`          // "description": "OS Volume 1",
	ETAG                 string              `json:"eTag,omitempty"`                 // "eTag": "1441036118675/8",
	GoldenVolumeUri      utils.Nstring       `json:"goldenVolumeUri,omitempty"`      // "goldenVolumeUri": "",
	Modified             string              `json:"modified,omitempty"`             // "modified": "20150831T154835.250Z",
	Name                 string              `json:"name,omitempty"`                 // "name": "OS Volume1 1",
	OEVolumeIQN          string              `json:"oeVolumeIQN,omitempty"`          // "",
	OEVolumeID           string              `json:"oeVolumeId,omitempty"`           // "",
	OEVolumeIp           string              `json:"oeVolumeIp,omitempty"`           // "",
	Size                 int                 `json:"size,omitempty"`                 // 99,
	State                string              `json:"state,omitempty"`                // "state": "Normal",
	StatelessServerUri   utils.Nstring       `json:"statelessServerUri,omitempty"`   // "statelessServerUri": "",
	Status               string              `json:"status,omitempty"`               // "status": "Critical",
	Type                 string              `json:"type,omitempty"`                 // "type": "OSVolume",
	URI                  utils.Nstring       `json:"uri,omitempty"`                  // "uri": "/rest/os-volumes/31e5dcba-b8ac-4f64-bbaa-7a4474f11994"
}

type OSVolumeList struct {
	Total       int           `json:"total,omitempty"`       // "total": 1,
	Count       int           `json:"count,omitempty"`       // "count": 1,
	Start       int           `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []OSVolume    `json:"members,omitempty"`     // "members":[]
}

func (c *I3SClient) GetOSVolumeByName(name string) (OSVolume, error) {
	var (
		osVolume OSVolume
	)
	osVolumes, err := c.GetOSVolumes(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if err != nil {
		return osVolume, err
	}
	if osVolumes.Total > 0 {
		return osVolumes.Members[0], err
	} else {
		return osVolume, err
	}
}

func (c *I3SClient) GetOSVolumes(filter string, sort string) (OSVolumeList, error) {
	var (
		uri       = "/rest/os-volumes"
		q         map[string]interface{}
		osVolumes OSVolumeList
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
		return osVolumes, err
	}

	log.Debugf("GetOSVolumes %s", data)
	if err := json.Unmarshal([]byte(data), &osVolumes); err != nil {
		return osVolumes, err
	}
	return osVolumes, nil
}
