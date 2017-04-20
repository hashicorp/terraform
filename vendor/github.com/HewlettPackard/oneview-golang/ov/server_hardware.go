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
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// HardwareState
type HardwareState int

const (
	H_UNKNOWN HardwareState = 1 + iota
	H_ADDING
	H_NOPROFILE_APPLIED
	H_MONITORED
	H_UNMANAGED
	H_REMOVING
	H_REMOVE_FAILED
	H_REMOVED
	H_APPLYING_PROFILE
	H_PROFILE_APPLIED
	H_REMOVING_PROFILE
	H_PROFILE_ERROR
	H_UNSUPPORTED
	H_UPATING_FIRMWARE
)

var hardwarestates = [...]string{
	"Unknown",          // not initialized
	"Adding",           // server being added
	"NoProfileApplied", // server successfully added
	"Monitored",        // server being monitored
	"Unmanaged",        // discovered a supported server
	"Removing",         // server being removed
	"RemoveFailed",     // unsuccessful server removal
	"Removed",          // server successfully removed
	"ApplyingProfile",  // profile being applied to server
	"ProfileApplied",   // profile successfully applied
	"RemovingProfile",  // profile being removed
	"ProfileError",     // unsuccessful profile apply or removal
	"Unsupported",      // server model or version not currently supported by the appliance
	"UpdatingFirmware", // server firmware update in progress
}

func (h HardwareState) String() string { return hardwarestates[h-1] }
func (h HardwareState) Equal(s string) bool {
	return (strings.ToUpper(s) == strings.ToUpper(h.String()))
}

// ServerHardware get server hardware from ov
type ServerHardware struct {
	ServerHardwarev200
	AssetTag              string        `json:"assetTag,omitempty"`              // "assetTag": "[Unknown]",
	Category              string        `json:"category,omitempty"`              // "category": "server-hardware",
	Created               string        `json:"created,omitempty"`               // "created": "2015-08-14T21:02:01.537Z",
	Description           utils.Nstring `json:"description,omitempty"`           // "description": null,
	ETAG                  string        `json:"eTag,omitempty"`                  // "eTag": "1441147370086",
	FormFactor            string        `json:"formFactor,omitempty"`            // "formFactor": "HalfHeight",
	LicensingIntent       string        `json:"licensingIntent,omitempty"`       // "licensingIntent": "OneView",
	LocationURI           utils.Nstring `json:"locationUri,omitempty"`           // "locationUri": "/rest/enclosures/092SN51207RR",
	MemoryMb              int           `json:"memoryMb,omitempty"`              // "memoryMb": 262144,
	Model                 string        `json:"model,omitempty"`                 // "model": "ProLiant BL460c Gen9",
	Modified              string        `json:"modified,omitempty"`              // "modified": "2015-09-01T22:42:50.086Z",
	MpFirwareVersion      string        `json:"mpFirmwareVersion,omitempty"`     // "mpFirmwareVersion": "2.03 Nov 07 2014",
	MpModel               string        `json:"mpModel,omitempty"`               // "mpModel": "iLO4",
	Name                  string        `json:"name,omitempty"`                  // "name": "se05, bay 16",
	PartNumber            string        `json:"partNumber,omitempty"`            // "partNumber": "727021-B21",
	Position              int           `json:"position,omitempty"`              // "position": 16,
	PowerLock             bool          `json:"powerLock,omitempty"`             // "powerLock": false,
	PowerState            string        `json:"powerState,omitempty"`            // "powerState": "Off",
	ProcessorCoreCount    int           `json:"processorCoreCount,omitempty"`    // "processorCoreCount": 14,
	ProcessorCount        int           `json:"processorCount,omitempty"`        // "processorCount": 2,
	ProcessorSpeedMhz     int           `json:"processorSpeedMhz,omitempty"`     // "processorSpeedMhz": 2300,
	ProcessorType         string        `json:"processorType,omitempty"`         // "processorType": "Intel(R) Xeon(R) CPU E5-2695 v3 @ 2.30GHz",
	RefreshState          string        `json:"refreshState,omitempty"`          // "refreshState": "NotRefreshing",
	RomVersion            string        `json:"romVersion,omitempty"`            // "romVersion": "I36 11/03/2014",
	SerialNumber          utils.Nstring `json:"serialNumber,omitempty"`          // "serialNumber": "2M25090RMW",
	ServerGroupURI        utils.Nstring `json:"serverGroupUri,omitempty"`        // "serverGroupUri": "/rest/enclosure-groups/56ad0069-8362-42fd-b4e3-f5c5a69af039",
	ServerHardwareTypeURI utils.Nstring `json:"serverHardwareTypeUri,omitempty"` // "serverHardwareTypeUri": "/rest/server-hardware-types/DB7726F7-F601-4EA8-B4A6-D1EE1B32C07C",
	ServerProfileURI      utils.Nstring `json:"serverProfileUri,omitempty"`      // "serverProfileUri": "/rest/server-profiles/9979b3a4-646a-4c3e-bca6-80ca0b403a93",
	ShortModel            string        `json:"shortModel,omitempty"`            // "shortModel": "BL460c Gen9",
	State                 string        `json:"state,omitempty"`                 // "state": "ProfileApplied",
	StateReason           string        `json:"stateReason,omitempty"`           // "stateReason": "NotApplicable",
	Status                string        `json:"status,omitempty"`                // "status": "Warning",
	Type                  string        `json:"type,omitempty"`                  // "type": "server-hardware-3",
	URI                   utils.Nstring `json:"uri,omitempty"`                   // "uri": "/rest/server-hardware/30373237-3132-4D32-3235-303930524D57",
	UUID                  utils.Nstring `json:"uuid,omitempty"`                  // "uuid": "30373237-3132-4D32-3235-303930524D57",
	VirtualSerialNumber   utils.Nstring `json:"VirtualSerialNumber,omitempty"`   // "virtualSerialNumber": "",
	VirtualUUID           string        `json:"virtualUuid,omitempty"`           // "virtualUuid": "00000000-0000-0000-0000-000000000000"
	// v1 properties
	MpDnsName   string `json:"mpDnsName,omitempty"`   // "mpDnsName": "ILO2M25090RMW",
	MpIpAddress string `json:"mpIpAddress,omitempty"` // make this private to force calls to GetIloIPAddress() "mpIpAddress": "172.28.3.136",
	// extra client struct
	Client *OVClient
}

// GetIloIPAddress - Use MpIpAddress for v1 and
// For v2 check MpHostInfo is not nil , loop through MpHostInfo.MpIPAddress[],
// and return the first nonzero address
func (h ServerHardware) GetIloIPAddress() string {
	if h.Client.IsHardwareSchemaV2() {
		if h.MpHostInfo != nil {
			log.Debug("working on getting IloIPAddress from MpHostInfo using v2")
			for _, MpIPObj := range h.MpHostInfo.MpIPAddress {
				if len(MpIPObj.Address) > 0 &&
					(MpDHCP.Equal(MpIPObj.Type) ||
						MpStatic.Equal(MpIPObj.Type) ||
						MpUndefined.Equal(MpIPObj.Type)) {
					return MpIPObj.Address
				}
			}
		}
	} else {
		log.Debug("working on getting IloIPAddress from MpIpAddress")
		return h.MpIpAddress
	}
	return ""
}

// server hardware list, simillar to ServerProfileList with a TODO
type ServerHardwareList struct {
	Type        string           `json:"type,omitempty"`        // "type": "server-hardware-list-3",
	Category    string           `json:"category,omitempty"`    // "category": "server-hardware",
	Count       int              `json:"count,omitempty"`       // "count": 15,
	Created     string           `json:"created,omitempty"`     // "created": "2015-09-08T04:58:21.489Z",
	ETAG        string           `json:"eTag,omitempty"`        // "eTag": "1441688301489",
	Modified    string           `json:"modified,omitempty"`    // "modified": "2015-09-08T04:58:21.489Z",
	NextPageURI utils.Nstring    `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	PrevPageURI utils.Nstring    `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	Start       int              `json:"start,omitempty"`       // "start": 0,
	Total       int              `json:"total,omitempty"`       // "total": 15,
	URI         string           `json:"uri,omitempty"`         // "uri": "/rest/server-hardware?sort=name:asc&filter=serverHardwareTypeUri=%27/rest/server-hardware-types/DB7726F7-F601-4EA8-B4A6-D1EE1B32C07C%27&filter=serverGroupUri=%27/rest/enclosure-groups/56ad0069-8362-42fd-b4e3-f5c5a69af039%27&start=0&count=100"
	Members     []ServerHardware `json:"members,omitempty"`     //"members":[]
}

// server hardware power off
func (s ServerHardware) PowerOff() error {
	var pt *PowerTask
	pt = pt.NewPowerTask(s)
	return pt.PowerExecutor(P_OFF)
}

// server hardware power on
func (s ServerHardware) PowerOn() error {
	var pt *PowerTask
	pt = pt.NewPowerTask(s)
	return pt.PowerExecutor(P_ON)
}

// get the power state
func (s ServerHardware) GetPowerState() (PowerState, error) {
	var pt *PowerTask
	pt = pt.NewPowerTask(s)
	if err := pt.GetCurrentPowerState(); err != nil {
		return P_UKNOWN, err
	}
	return pt.State, nil
}

// get a server hardware with uri
func (c *OVClient) GetServerHardware(uri utils.Nstring) (ServerHardware, error) {

	var hardware ServerHardware
	// refresh login

	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	// rest call
	data, err := c.RestAPICall(rest.GET, uri.String(), nil)
	if err != nil {
		return hardware, err
	}

	log.Debugf("GetServerHardware %s", data)
	if err := json.Unmarshal([]byte(data), &hardware); err != nil {
		return hardware, err
	}
	hardware.Client = c
	return hardware, nil
}

// get a server hardware with uri
func (c *OVClient) GetServerHardwareByName(name string) (ServerHardware, error) {

	var (
		serverHardware ServerHardware
	)

	filters := []string{fmt.Sprintf("name matches '%s'", name)}
	serverHardwareList, err := c.GetServerHardwareList(filters, "name:asc")
	if serverHardwareList.Total > 0 {
		serverHardwareList.Members[0].Client = c
		return serverHardwareList.Members[0], err
	} else {
		return serverHardware, err
	}
}

// get a server hardware with filters
func (c *OVClient) GetServerHardwareList(filters []string, sort string) (ServerHardwareList, error) {
	var (
		uri        = "/rest/server-hardware"
		q          map[string]interface{}
		serverlist ServerHardwareList
	)
	q = make(map[string]interface{})
	if len(filters) > 0 {
		q["filter"] = filters
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

	// parse the url and setup any query strings
	var Url *url.URL
	Url, err = url.Parse(utils.Sanatize(c.Endpoint))
	if err != nil {
		return serverlist, err
	}
	Url.Path += uri
	c.GetQueryString(Url)

	log.Debugf("GetServerHardwareList %s", data)
	if err := json.Unmarshal([]byte(data), &serverlist); err != nil {
		return serverlist, err
	}
	return serverlist, nil
}

// get available server
// blades = rest_api(:oneview, :get, "/rest/server-hardware?sort=name:asc&filter=serverHardwareTypeUri='#{server_hardware_type_uri}'&filter=serverGroupUri='#{enclosure_group_uri}'")
func (c *OVClient) GetAvailableHardware(hardwaretype_uri utils.Nstring, servergroup_uri utils.Nstring) (hw ServerHardware, err error) {
	var (
		hwlist ServerHardwareList
		f      = []string{"serverHardwareTypeUri='" + hardwaretype_uri.String() + "'",
			"serverGroupUri='" + servergroup_uri.String() + "'"}
	)
	if hwlist, err = c.GetServerHardwareList(f, "name:desc"); err != nil {
		return hw, err
	}
	if !(len(hwlist.Members) > 0) {
		return hw, errors.New("Error! No available blades that are compatible with the server profile!")
	}

	// pick an available blade
	for _, blade := range hwlist.Members {
		if H_NOPROFILE_APPLIED.Equal(blade.State) {
			hw = blade
			break
		}
	}
	if hw.Name == "" {
		return hw, errors.New("No more blades are available for provisioning!")
	}
	return hw, nil
}
