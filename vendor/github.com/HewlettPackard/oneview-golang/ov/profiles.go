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

// Package ov
package ov

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// FirmwareOption structure for firware settings
type FirmwareOption struct {
	FirmwareOptionv200
	ForceInstallFirmware bool          `json:"forceInstallFirmware,omitempty"` // "forceInstallFirmware": false,
	FirmwareBaselineUri  utils.Nstring `json:"firmwareBaselineUri,omitempty"`  // "firmwareBaselineUri": null,
	ManageFirmware       bool          `json:"manageFirmware,omitempty"`       // "manageFirmware": false
}

// BootModeOption mode option
type BootModeOption struct {
	ManageMode    bool          `json:"manageMode,omitempty"`    // "manageMode": true,
	Mode          string        `json:"mode,omitempty"`          // "mode": "BIOS",
	PXEBootPolicy utils.Nstring `json:"pxeBootPolicy,omitempty"` // "pxeBootPolicy": null
}

// BootManagement management
type BootManagement struct {
	ManageBoot bool     `json:"manageBoot,omitempty"` // "manageBoot": true,
	Order      []string `json:"order,omitempty"`      // "order": ["CD","USB","HardDisk","PXE"]
}

// BiosSettings structure
type BiosSettings struct {
	ID    string `json:"id,omitempty"`    // id
	Value string `json:"value,omitempty"` // value
}

// BiosOption - bios options
type BiosOption struct {
	ManageBios         bool           `json:"manageBios,omitempty"`         // "manageBios": false,
	OverriddenSettings []BiosSettings `json:"overriddenSettings,omitempty"` // "overriddenSettings": []
}

// ServerProfile - server profile object for ov
type ServerProfile struct {
	ServerProfilev200
	*ServerProfilev300
	Affinity              string              `json:"affinity,omitempty"`         // "affinity": "Bay",
	AssociatedServer      utils.Nstring       `json:"associatedServer,omitempty"` // "associatedServer": null,
	Bios                  *BiosOption         `json:"bios,omitempty"`             // "bios": {	},
	Boot                  BootManagement      `json:"boot,omitempty"`             // "boot": { },
	BootMode              BootModeOption      `json:"bootMode,omitempty"`         // "bootMode": {},
	Category              string              `json:"category,omitempty"`         // "category": "server-profiles",
	Connections           []Connection        `json:"connections,omitempty"`
	Description           string              `json:"description,omitempty"`           // "description": "Docker Machine Bay 16",
	Created               string              `json:"created,omitempty"`               // "created": "20150831T154835.250Z",
	ETAG                  string              `json:"eTag,omitempty"`                  // "eTag": "1441036118675/8"
	EnclosureBay          int                 `json:"enclosureBay,omitempty"`          // "enclosureBay": 16,
	EnclosureGroupURI     utils.Nstring       `json:"enclosureGroupUri,omitempty"`     // "enclosureGroupUri": "/rest/enclosure-groups/56ad0069-8362-42fd-b4e3-f5c5a69af039",
	EnclosureURI          utils.Nstring       `json:"enclosureUri,omitempty"`          // "enclosureUri": "/rest/enclosures/092SN51207RR",
	Firmware              FirmwareOption      `json:"firmware,omitempty"`              // "firmware": { },
	HideUnusedFlexNics    bool                `json:"hideUnusedFlexNics,omitempty"`    // "hideUnusedFlexNics": false,
	InProgress            bool                `json:"inProgress,omitempty"`            // "inProgress": false,
	LocalStorage          LocalStorageOptions `json:"localStorage,omitempty"`          // "localStorage": {},
	MACType               string              `json:"macType,omitempty"`               // "macType": "Physical",
	Modified              string              `json:"modified,omitempty"`              // "modified": "20150902T175611.657Z",
	Name                  string              `json:"name,omitempty"`                  // "name": "Server_Profile_scs79",
	SanStorage            SanStorageOptions   `json:"sanStorage,omitempty"`            // "sanStorage": {},
	SerialNumber          utils.Nstring       `json:"serialNumber,omitempty"`          // "serialNumber": "2M25090RMW",
	SerialNumberType      string              `json:"serialNumberType,omitempty"`      // "serialNumberType": "Physical",
	ServerHardwareTypeURI utils.Nstring       `json:"serverHardwareTypeUri,omitempty"` // "serverHardwareTypeUri": "/rest/server-hardware-types/DB7726F7-F601-4EA8-B4A6-D1EE1B32C07C",
	ServerHardwareURI     utils.Nstring       `json:"serverHardwareUri,omitempty"`     // "serverHardwareUri": "/rest/server-hardware/30373237-3132-4D32-3235-303930524D57",
	State                 string              `json:"state,omitempty"`                 // "state": "Normal",
	Status                string              `json:"status,omitempty"`                // "status": "Critical",
	TaskURI               utils.Nstring       `json:"taskUri,omitempty"`               // "taskUri": "/rest/tasks/6F0DF438-7D30-41A2-A36D-62AB866BC7E8",
	Type                  string              `json:"type,omitempty"`                  // 	Type               string `json:"type,omitempty"`	// "type": "ServerProfileV4",
	URI                   utils.Nstring       `json:"uri,omitempty"`                   // "uri": "/rest/server-profiles/9979b3a4-646a-4c3e-bca6-80ca0b403a93",
	UUID                  utils.Nstring       `json:"uuid,omitempty"`                  // "uuid": "30373237-3132-4D32-3235-303930524D57",
	WWNType               string              `json:"wwnType,omitempty"`               // "wwnType": "Physical",
}

// GetConnectionByName gets the connection from a profile with a given name
func (s ServerProfile) GetConnectionByName(name string) (Connection, error) {
	var connection Connection
	for _, c := range s.Connections {
		if c.Name == name {
			return c, nil
		}
	}
	return connection, errors.New("Error connection not found on server profile, please try a different connection name.")
}

// Clone server profile
func (s ServerProfile) Clone() ServerProfile {
	var ca []Connection
	for _, c := range s.Connections {
		ca = append(ca, c.Clone())
	}

	return ServerProfile{
		Affinity:           s.Affinity,
		Bios:               s.Bios,
		Boot:               s.Boot,
		BootMode:           s.BootMode,
		Connections:        ca,
		Description:        s.Description,
		EnclosureBay:       s.EnclosureBay,
		EnclosureGroupURI:  s.EnclosureGroupURI,
		EnclosureURI:       s.EnclosureURI,
		Firmware:           s.Firmware,
		HideUnusedFlexNics: s.HideUnusedFlexNics,
		LocalStorage:       s.LocalStorage.Clone(),
		MACType:            s.MACType,
		Name:               s.Name,
		SanStorage:         s.SanStorage.Clone(),
		SerialNumberType:   s.SerialNumberType,
		Type:               s.Type,
		WWNType:            s.WWNType,
	}
}

// ServerProfileList a list of ServerProfile objects
// TODO: missing properties, need to think how we can make a higher lvl structure like an OVList
// Then things like Members are inherited
type ServerProfileList struct {
	Total       int             `json:"total,omitempty"`       // "total": 1,
	Count       int             `json:"count,omitempty"`       // "count": 1,
	Start       int             `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring   `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring   `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring   `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=serialNumber%20matches%20%272M25090RMW%27&sort=name:asc"
	Members     []ServerProfile `json:"members,omitempty"`     // "members":[]
}

// get a server profile by name
func (c *OVClient) GetProfileByName(name string) (ServerProfile, error) {
	var (
		profile ServerProfile
	)
	profiles, err := c.GetProfiles(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if profiles.Total > 0 {
		return profiles.Members[0], err
	} else {
		return profile, err
	}
}

// GetProfileBySN  accepts serial number
func (c *OVClient) GetProfileBySN(serialnum string) (ServerProfile, error) {
	var (
		profile ServerProfile
	)
	profiles, err := c.GetProfiles(fmt.Sprintf("serialNumber matches '%s'", serialnum), "name:asc")
	if profiles.Total > 0 {
		return profiles.Members[0], err
	} else {
		return profile, err
	}
}

// GetProfiles - get a server profiles
func (c *OVClient) GetProfiles(filter string, sort string) (ServerProfileList, error) {
	var (
		uri      = "/rest/server-profiles"
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

	log.Debugf("GetProfiles %s", data)
	if err := json.Unmarshal([]byte(data), &profiles); err != nil {
		return profiles, err
	}
	return profiles, nil
}

// GetProfileByURI - get the profile from a uri
func (c *OVClient) GetProfileByURI(uri utils.Nstring) (ServerProfile, error) {
	var (
		profile ServerProfile
	)

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri.String(), nil)
	if err != nil {
		return profile, err
	}

	log.Debugf("GetProfileByURI %s", data)
	if err := json.Unmarshal([]byte(data), &profile); err != nil {
		return profile, err
	}
	return profile, nil
}

// SubmitNewProfile - submit new profile template
func (c *OVClient) SubmitNewProfile(p ServerProfile) (t *Task, err error) {
	log.Infof("Initializing creation of server profile for %s.", p.Name)
	var (
		uri = "/rest/server-profiles"
	// 	task = rest_api(:oneview, :post, '/rest/server-profiles', { 'body' => new_template_profile })
	)
	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, p)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, p)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new profile request: %s", err)
		return t, err
	}

	log.Debugf("Response NewProfile %s", data)
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		t.TaskIsDone = true
		log.Errorf("Error with task un-marshal: %s", err)
		return t, err
	}

	return t, err
}

// create profile from template
func (c *OVClient) CreateProfileFromTemplate(name string, template ServerProfile, blade ServerHardware) error {
	log.Debugf("TEMPLATE : %+v\n", template)
	var (
		new_template ServerProfile
		err          error
	)

	//GET on /rest/server-profile-templates/{id}new-profile
	if c.IsProfileTemplates() {
		log.Debugf("getting profile by URI %+v, v2", template.URI)
		new_template, err = c.GetProfileByURI(template.URI)
		if err != nil {
			return err
		}
		new_template.Type = "ServerProfileV5"
		new_template.ServerProfileTemplateURI = template.URI // create relationship
		log.Debugf("new_template -> %+v", new_template)
	} else {
		log.Debugf("get new_template from clone, v1")
		new_template = template.Clone()
	}
	new_template.ServerHardwareURI = blade.URI
	new_template.Description += " " + name
	new_template.Name = name

	t, err := c.SubmitNewProfile(new_template)
	if err != nil {
		return err
	}
	err = t.Wait()
	if err != nil {
		return err
	}
	return nil
}

// submit new profile template
func (c *OVClient) SubmitDeleteProfile(p ServerProfile) (t *Task, err error) {
	var (
		uri = p.URI.String()
	// 	task = rest_api(:oneview, :post, '/rest/server-profiles', { 'body' => new_template_profile })
	)
	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, p)
	log.Debugf("task -> %+v", t)
	if uri == "" {
		log.Warn("Unable to post delete, no uri found.")
		t.TaskIsDone = true
		return t, err
	}
	data, err := c.RestAPICall(rest.DELETE, uri, nil)
	if err != nil {
		log.Errorf("Error submitting new profile request: %s", err)
		t.TaskIsDone = true
		return t, err
	}

	log.Debugf("Response delete profile %s", data)
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		t.TaskIsDone = true
		log.Errorf("Error with task un-marshal: %s", err)
		return t, err
	}

	return t, err
}

// delete a profile, assign the server and remove the profile from the system
func (c *OVClient) DeleteProfile(name string) error {
	// get the profile for this server
	var (
		servernamemsg string
		server        ServerHardware
		profile       ServerProfile
		err           error
	)

	servernamemsg = "'no server'"
	profile, err = c.GetProfileByName(name)
	if err != nil {
		return err
	}

	if profile.Name != "" {
		if profile.ServerHardwareURI != "" {
			server, err = c.GetServerHardware(profile.ServerHardwareURI)
			if err != nil {
				log.Warnf("Problem getting server hardware, %s", err)
			} else {
				if server.Name != "" {
					servernamemsg = server.Name
				}
			}
		}
		log.Infof("Delete server profile %s from oneview, %s will be unassigned.", profile.Name, servernamemsg)

		// power off the server so that we can remove it
		if server.Name != "" {
			server.PowerOff()
		}

		// submit delete task
		t, err := c.SubmitDeleteProfile(profile)
		err = t.Wait()
		if err != nil {
			return err
		}

		// check for task execution

	} else {
		log.Infof("Profile could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *OVClient) UpdateServerProfile(p ServerProfile) error {
	log.Infof("Initializing update of server profile for %s.", p.Name)
	var (
		uri = p.URI.String()
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, p)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, p)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update server profile request: %s", err)
		return err
	}

	log.Debugf("Response update ServerProfile %s", data)
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
