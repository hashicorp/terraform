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

type ArtifactsBundle struct {
	ArtifactsCount     int                      `json:"artifactsCount,omitempty"`     // "artifactsCount": 1,
	ArtifactsBundleID  string                   `json:"artifactsbundleID,omitempty"`  // "artifactsbundleID": "id",
	BackupService      bool                     `json:"backupService,omitempty"`      // "backupService": true,
	BuildPlans         []BuildPlanArtifact      `json:"buildPlans,omitempty"`         // "buildPlans": [],
	Category           string                   `json:"category,omitempty"`           // "category": "artifacts-bundle"
	CheckSum           string                   `json:"checkSum,omitempty"`           // "checkSum": "0",
	Created            string                   `json:"created,omitempty"`            // "20150831T154835.250Z",
	DeploymentPlans    []DeploymentPlanArtifact `json:"deploymentPlans,omitempty"`    // "deploymentPlans": [],
	Description        string                   `json:"description,omitempty"`        // "description": "Artifacts Bundle 1",
	DownloadURI        utils.Nstring            `json:"downloadURI,omitempty"`        // "downloadURI": "",
	ETAG               string                   `json:"eTag,omitempty"`               // "eTag": "1441036118675/8",
	GoldenImage        []GoldenImageArtifact    `json:"goldenimage,omitempty"`        // "goldenimage": [],
	ImportBundle       bool                     `json:"importbundle,omitempty"`       // "importbundle": true,
	LastBackUpDownload string                   `json:"lastBackUpDownload,omitempty"` // "lastBackUpDownload": "20150831T154835.250Z",
	Modified           string                   `json:"modified,omitempty"`           // "modified": "20150831T154835.250Z",
	Name               string                   `json:"name,omitempty"`               // "name": "Artifacts Bundle 1",
	PlanScripts        []PlanScriptArtifact     `json:"planScripts,omitempty"`        // "planScripts": [],
	ReadOnly           bool                     `json:"readOnly,omitempty"`           // "readOnly": true,
	RecoverBundle      bool                     `json:"recoverBundle,omitempty"`      // "recoverBundle": true,
	Size               int                      `json:"size,omitempty"`               // "size": 99,
	State              string                   `json:"state,omitempty"`              // "state": "Normal",
	Status             string                   `json:"status,omitempty"`             // "status": "Critical",
	Type               string                   `json:"type,omitempty"`               // "type": "ArtifactsBundle",
	URI                utils.Nstring            `json:"uri,omitempty"`                // "uri": "/rest/artifact-bundles/31e5dcba-b8ac-4f64-bbaa-7a4474f11994"
}

type BuildPlanArtifact struct {
	BPID           string `json:"bpID,omitempty"`           // "bpID": "build plan id",
	BuildPlanName  string `json:"buildPlanName,omitempty"`  // "buildPlanName": "build plan 1",
	Description    string `json:"description,omitempty"`    // "description": "build plan 1",
	PlanScriptName string `json:"planScriptName,omitempty"` // "planScriptName": "plan script 1",
	ReadOnly       bool   `json:"readOnly,omitempty"`       // "readOnly": true,
}

type DeploymentPlanArtifact struct {
	DeploymentPlanName string `json:"deploymentplanName,omitempty"` // "deploymentplanName": "deploy plan 1"
	Description        string `json:"description,omitempty"`        // "description": "deploy plan 1",
	DPId               string `json:"dpId,omitempty"`               // "dpId": "deploy plan id",
	GoldenImageName    string `json:"goldenImageName,omitempty"`    // "goldenImageName": "golden image 1",
	OEBPName           string `json:"oebpName,omitempty"`           // "oebpName": "oebp name 1",
	ReadOnly           bool   `json:"readOnly,omitempty"`           // "readOnly": true,
}

type GoldenImageArtifact struct {
	Description     string `json:"description,omitempty"`     // "description": "golden image 1",
	GIID            string `json:"giId,omitempty"`            // "giId": "golden image id",
	GoldenImageName string `json:"goldenimageName,omitempty"` // "goldenimageName": "golden image 1",
	ReadOnly        bool   `json:"readOnly,omitempty"`        // "readOnly": true,
}

type PlanScriptArtifact struct {
	Description    string `json:"description,omitempty"`    // "description": "plan script 1",
	PlanScriptName string `json:"planScriptName,omitempty"` // "planScriptName": "plan script 1",
	PSID           string `json:"psID,omitempty"`           // "psID": "plan script id",
	ReadOnly       bool   `json:"readOnly,omitempty"`       // "readOnly", false,
}

type ArtifactsBundleList struct {
	Total       int               `json:"total,omitempty"`       // "total": 1,
	Count       int               `json:"count,omitempty"`       // "count": 1,
	Start       int               `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring     `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring     `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring     `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []ArtifactsBundle `json:"members,omitempty"`     // "members":[]
}

type InputArtifactsBundle struct {
	BuildPlans         []InputArtifacts `json:"buildPlans,omitempty"`         // "buildPlans": [],
	DeploymentClusters []string         `json:"deploymentClusters,omitempty"` // "deploymentClusters": [],
	DeploymentGroups   []string         `json:"deploymentGroups,omitempty"`   // "deploymentGroups": [],
	DeploymentPlans    []InputArtifacts `json:"deploymentPlans,omitempty"`    // "deploymentPlans": [],
	Description        string           `json:"description,omitempty"`        // "description": "Artifacts Bundle 1",
	GoldenImages       []InputArtifacts `json:"goldenImages,omitempty"`       // "goldenImages": [],
	GoldenVolumes      []string         `json:"goldenVolumes,omitempty"`      // "goldenVolumes": [],
	I3SAppliance       []string         `json:"i3sAppliance,omitempty"`       // "i3sAppliance": [],
	Name               string           `json:"name,omitempty"`               // "name": "Artifacts Bundle 1",
	OEVolumes          []string         `json:"oeVolumes,omitempty"`          // "oeVolumes": [],
	PlanScripts        []InputArtifacts `json:"planScripts,omitempty"`        // "planScripts": [],
	StatelessServers   []string         `json:"statelessServers,omitempty"`   // "statelessServers": [],
}

type InputArtifacts struct {
	ReadOnly    bool          `json:"readOnly,omitempty"`    // "readyOnly": false,
	ResourceUri utils.Nstring `json:"resourceUri,omitempty"` // "resourceUri": "",
}

func (c *I3SClient) GetArtifactsBundleByName(name string) (ArtifactsBundle, error) {
	var (
		artifactsBundle ArtifactsBundle
	)
	artifactsBundles, err := c.GetArtifactsBundles(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if err != nil {
		return artifactsBundle, err
	}
	if artifactsBundles.Total > 0 {
		return artifactsBundles.Members[0], err
	} else {
		return artifactsBundle, err
	}
}

func (c *I3SClient) GetArtifactsBundles(filter string, sort string) (ArtifactsBundleList, error) {
	var (
		uri              = "/rest/artifact-bundles"
		q                map[string]interface{}
		artifactsBundles ArtifactsBundleList
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
		return artifactsBundles, err
	}

	log.Debugf("GetArtifactsBundles %s", data)
	if err := json.Unmarshal([]byte(data), &artifactsBundles); err != nil {
		return artifactsBundles, err
	}
	return artifactsBundles, nil
}

func (c *I3SClient) CreateArtifactsBundle(artifactsBundle InputArtifactsBundle) error {
	log.Infof("Initializing creation of artifactsBundle for %s.", artifactsBundle.Name)
	var (
		uri = "/rest/artifact-bundles"
		t   *Task
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, artifactsBundle)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, artifactsBundle)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new artifacts bundle request: %s", err)
		return err
	}

	log.Debugf("Response New ArtifactsBundle %s", data)
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

func (c *I3SClient) DeleteArtifactsBundle(name string) error {
	var (
		artifactsBundle ArtifactsBundle
		err             error
		t               *Task
		uri             string
	)

	artifactsBundle, err = c.GetArtifactsBundleByName(name)
	if err != nil {
		return err
	}
	if artifactsBundle.Name != "" {
		t = t.NewTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", artifactsBundle.URI, artifactsBundle)
		log.Debugf("task -> %+v", t)
		uri = artifactsBundle.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		data, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete artifactsBundle request: %s", err)
			t.TaskIsDone = true
			return err
		}

		log.Debugf("Response delete artifactsBundle %s", data)
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
	} else {
		log.Infof("ArtifactsBundle could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *I3SClient) UpdateArtifactsBundle(artifactsBundle ArtifactsBundle) error {
	log.Infof("Initializing update of artifacts bundle for %s.", artifactsBundle.Name)
	var (
		uri = artifactsBundle.URI.String()
		t   *Task
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, artifactsBundle)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, artifactsBundle)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update artifacts bundle request: %s", err)
		return err
	}

	log.Debugf("Response update ArtifactsBundle %s", data)
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
