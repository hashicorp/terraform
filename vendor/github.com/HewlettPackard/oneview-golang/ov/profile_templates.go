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

// Package ov -
package ov

import (
	"encoding/json"
	"fmt"

	"github.com/HewlettPackard/oneview-golang/liboneview"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/docker/machine/libmachine/log"
)

// introduced in v200 for oneview, allows for an easier method
// to build the profiles for servers and associate them.
// we don't operate on a new struct, we simply use the ServerProfile struct

// ProfileTemplatesNotSupported - determine these functions are supported
func (c *OVClient) ProfileTemplatesNotSupported() bool {
	var currentversion liboneview.Version
	var asc liboneview.APISupport
	currentversion = currentversion.CalculateVersion(c.APIVersion, 108) // force icsp to 108 version since icsp version doesn't matter
	asc = asc.NewByName("profile_templates.go")
	if !asc.IsSupported(currentversion) {
		log.Debugf("ProfileTemplates client version not supported: %+v", currentversion)
		return true
	}
	return false
}

// IsProfileTemplates - returns true when we should use GetProfileTemplate...
func (c *OVClient) IsProfileTemplates() bool {
	return !c.ProfileTemplatesNotSupported()
}

// get a server profile template by name
func (c *OVClient) GetProfileTemplateByName(name string) (ServerProfile, error) {
	var (
		profile ServerProfile
	)
	// v2 way to get ServerProfile
	if c.IsProfileTemplates() {
		profiles, err := c.GetProfileTemplates(fmt.Sprintf("name matches '%s'", name), "name:asc")
		if profiles.Total > 0 {
			return profiles.Members[0], err
		} else {
			return profile, err
		}
	} else {

		// v1 way to get a ServerProfile
		profiles, err := c.GetProfiles(fmt.Sprintf("name matches '%s'", name), "name:asc")
		if profiles.Total > 0 {
			return profiles.Members[0], err
		} else {
			return profile, err
		}
	}

}

// get a server profiles
func (c *OVClient) GetProfileTemplates(filter string, sort string) (ServerProfileList, error) {
	var (
		uri      = "/rest/server-profile-templates"
		q        map[string]interface{}
		profiles ServerProfileList
	)
	q = make(map[string]interface{})
	if filter != "" {
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
		return profiles, err
	}

	log.Debugf("GetProfileTemplates %s", data)
	if err := json.Unmarshal([]byte(data), &profiles); err != nil {
		return profiles, err
	}
	return profiles, nil
}

func (c *OVClient) CreateProfileTemplate(serverProfileTemplate ServerProfile) error {
	log.Infof("Initializing creation of server profile template for %s.", serverProfileTemplate.Name)
	var (
		uri = "/rest/server-profile-templates"
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, serverProfileTemplate)
	log.Debugf("task -> %+v", t)
	_, err := c.RestAPICall(rest.POST, uri, serverProfileTemplate)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new server profile template request: %s", err)
		return err
	}

	return nil
}

func (c *OVClient) DeleteProfileTemplate(name string) error {
	var (
		serverProfileTemplate ServerProfile
		err                   error
		t                     *Task
		uri                   string
	)

	serverProfileTemplate, err = c.GetProfileTemplateByName(name)
	if err != nil {
		return err
	}
	if serverProfileTemplate.Name != "" {
		t = t.NewProfileTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", serverProfileTemplate.URI, serverProfileTemplate)
		log.Debugf("task -> %+v", t)
		uri = serverProfileTemplate.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		_, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete server profile template request: %s", err)
			t.TaskIsDone = true
			return err
		}

		return nil
	} else {
		log.Infof("ServerProfileTemplate could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *OVClient) UpdateProfileTemplate(serverProfileTemplate ServerProfile) error {
	log.Infof("Initializing update of server profile template for %s.", serverProfileTemplate.Name)
	var (
		uri = serverProfileTemplate.URI.String()
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, serverProfileTemplate)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, serverProfileTemplate)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update server profile template request: %s", err)
		return err
	}

	log.Debugf("Response update ServerProfileTemplate %s", data)
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
