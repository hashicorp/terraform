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

type GoldenImage struct {
	ArtifactBundleCategory string              `json:"artifactBundleCategory,omitempty"` // "artifactBundleCategory": "artifact-category",
	BuildPlanCategory      string              `json:"buildPlanCategory,omitempty"`      // "buildPlanCategory": "buildplan-category",
	BuildPlanName          string              `json:"buildPlanName,omitempty"`          // "buildPlanName": "build plan 1",
	BuildPlanURI           utils.Nstring       `json:"buildPlanUri,omitempty"`           // "buildPlanUri": "/rest/build-planns/9b8f7ec0-52b3-475e-84f4-c4eac51c2c20"
	BundleName             string              `json:"bundleName,omitempty"`             // "bundleName": "bundle 1",
	BundleURI              utils.Nstring       `json:"bundleURI,omitempty"`              // "bundleURI": "/rest/artifact-bundles/9b8f7ec0-52b3-475e-84f4-c4eac51c2c20"
	Category               string              `json:"category,omitempty"`               // "category": "golden-images",
	CheckSum               string              `json:"checkSum,omitempty"`               // "checkSum": "1",
	Created                string              `json:"created,omitempty"`                // "created": "20150831T154835.250Z",
	DependentArtifacts     []DependentArtifact `json:"dependentArtifacts,omitempty"`     // "dependentArtifacts": [],
	Description            utils.Nstring       `json:"description,omitempty"`            // "description": "Golden Image 1",
	ETAG                   string              `json:"eTag,omitempty"`                   // "eTag": "1441036118675/8",
	ID                     string              `json:"id,omitempty"`                     // "id": "1",
	ImageCapture           bool                `json:"imageCapture,omitempty"`           // "imageCapture": true,
	ImportedFromBundle     bool                `json:"importedFromBundle,omitempty"`     // "importedFromBundle": true,
	Modified               string              `json:"modified,omitempty"`               // "modified": "20150831T154835.250Z",
	Name                   string              `json:"name,omitempty"`                   // "name": "Golden Image 1",
	OSVolumeCategory       string              `json:"osVolumeCategory,omitempty"`       // "osVolumeCategory": "os-volumes",
	OSVolumeName           string              `json:"osVolumeName,omitempty"`           // "osVolumeName": "os volume 1",
	OSVolumeURI            utils.Nstring       `json:"osVolumeURI,omitempty"`            // "osVolumeURI": "/rest/os-volumes/1234",
	ReadOnly               bool                `json:"readOnly,omitempty"`               // "readOnly": true,
	Size                   int                 `json:"size,omitempty"`                   // "size": 50
	State                  string              `json:"state,omitempty"`                  // "state": "Normal",
	Status                 string              `json:"status,omitempty"`                 // "status": "Critical",
	Type                   string              `json:"type,omitempty"`                   // "type": "GoldenImage",
	URI                    utils.Nstring       `json:"uri,omitempty"`                    // "uri": "/rest/golden-images/e2f0031b-52bd-4223-9ac1-d91cb519d548",
}

type GoldenImageList struct {
	Total       int           `json:"total,omitempty"`       // "total": 1,
	Count       int           `json:"count,omitempty"`       // "count": 1,
	Start       int           `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/golden-images?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []GoldenImage `json:"members,omitempty"`     // "members":[]
}

type DependentArtifact struct {
	Name string        `json:"name,omitempty"` // "name": "dependent artifact 1",
	URI  utils.Nstring `json:"uri,omitempty"`  // "uri": "/rest/artifact-bundles/9b8f7ec0-52b3-475e-84f4-c4eac51c2c20",
}

func (c *I3SClient) GetGoldenImageByName(name string) (GoldenImage, error) {
	var (
		goldenImage GoldenImage
	)
	goldenImages, err := c.GetGoldenImages(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if goldenImages.Total > 0 {
		return goldenImages.Members[0], err
	} else {
		return goldenImage, err
	}
}

func (c *I3SClient) GetGoldenImages(filter string, sort string) (GoldenImageList, error) {
	var (
		uri          = "/rest/golden-images"
		q            map[string]interface{}
		goldenImages GoldenImageList
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
		return goldenImages, err
	}

	log.Debugf("GetGoldenImages %s", data)
	if err := json.Unmarshal([]byte(data), &goldenImages); err != nil {
		return goldenImages, err
	}
	return goldenImages, nil
}

func (c *I3SClient) CreateGoldenImage(goldenImage GoldenImage) error {
	log.Infof("Initializing creation of goldenImage for %s.", goldenImage.Name)
	var (
		uri = "/rest/golden-images"
		t   *Task
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, goldenImage)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, goldenImage)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new golden image request: %s", err)
		return err
	}

	log.Debugf("Response New GoldenImage %s", data)
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

func (c *I3SClient) DeleteGoldenImage(name string) error {
	var (
		goldenImage GoldenImage
		err         error
		t           *Task
		uri         string
	)

	goldenImage, err = c.GetGoldenImageByName(name)
	if err != nil {
		return err
	}
	if goldenImage.Name != "" {
		t = t.NewTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", goldenImage.URI, goldenImage)
		log.Debugf("task -> %+v", t)
		uri = goldenImage.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		data, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete golden image request: %s", err)
			t.TaskIsDone = true
			return err
		}

		log.Debugf("Response delete golden image %s", data)
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
		log.Infof("GoldenImage could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *I3SClient) UpdateGoldenImage(goldenImage GoldenImage) error {
	log.Infof("Initializing update of golden image for %s.", goldenImage.Name)
	var (
		uri = goldenImage.URI.String()
		t   *Task
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, goldenImage)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, goldenImage)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update golden image request: %s", err)
		return err
	}

	log.Debugf("Response update Golden Image %s", data)
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
