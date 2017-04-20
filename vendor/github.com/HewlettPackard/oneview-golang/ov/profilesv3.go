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

package ov

import (
	"fmt"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type BootOptionV3 struct {
	BootTargetLun string `json:"bootTargetLun,omitempty"` // "bootTargetLun": "0",
	BootTargetName string `json:"bootTargetName,omitempty"` // "bootTargetName": "iqn.2015-02.com.hpe:iscsi.target",
	BootVolumeSource string `json:"bootVolumeSource,omitempty"` // "bootVolumeSource": "",
	ChapLevel string `json:"chapLevel,omitempty"` // "chapLevel": "None",
	ChapName  string  `json:"chapName,omitempty"` // "chapName": "chap name",
	ChapSecret string `json:"chapSecret,omitempty"` // "chapSecret": "super secret chap secret",
	FirstBootTargetIp string `json:"firstBootTargetIp,omitempty"` // "firtBootTargetIp": "10.0.0.50",
	FirstBootTargetPort string `json:"firstBootTargetPort,omitempty"` // "firstBootTargetPort": "8080",
	InitiatorGateway string `json:"initiatorGateway,omitempty"` // "initiatorGateway": "3260",
	InitiatorIp string `json:"initiatorIp,omitempty"` // "initiatorIp": "192.168.6.21",
	InitiatorName string `json:"initiatorName,omitempty"` // "initiatorName": "iqn.2015-02.com.hpe:oneview-vcgs02t012",
	InitiatorNameSource string `json:"initiatorNameSource,omitempty"` // "initiatorNameSource": "UserDefined"
	InitiatorSubnetMask string `json:"initiatorSubnetMask,omitempty"` // "initiatorSubnetMask": "255.255.240.0",
	InitiatorVlanId int `json:"initiatorVlanId,omitempty"` // "initiatorVlanId": 77,
	MutualChapName string `json:"mutualChapName,omitempty"` // "mutualChapName": "name of mutual chap",
	MutualChapSecret string `json:"mutualChapSecret,omitempty"` // "mutualChapSecret": "secret of mutual chap",
	SecondBootTargetIp string `json:"secondBootTargetIp,omitempty"` // "secondBootTargetIp": "10.0.0.51",
	SecondBootTargetPort string `json:"secondBootTargetPort,omitempty"` // "secondBootTargetPort": "78"
}

type ServerProfilev300 struct {
	IscsiInitiatorName  string `json:"iscsiInitiatorName,omitempty"` // "iscsiInitiatorName": "name of iscsi initiator name",
	IscsiInitiatorNameType string `json:"iscsiInitiatorNameType,omitempty"` // "iscsiInitiatorNameType": "UserDefined",
	OSDeploymentSettings OSDeploymentSettings `json:"osDeploymentSettings,omitempty"` // "osDeploymentSettings": {...},
}

type OSDeploymentSettings struct {
	OSCustomAttributes []OSCustomAttribute `json:"osCustomAttributes,omitempty"` // "osCustomAttributes": [],
	OSDeploymentPlanUri utils.Nstring `json:"osDeploymentPlanUri,omitempty"` // "osDeploymentPlanUri": nil,
	OSVolumeUri utils.Nstring `json:"osVolumeUri,omitempty"` // "osVolumeUri": nil,
}

type OSCustomAttribute struct {
	Name string `json:"name,omitempty"` // "name": "custom attribute 1",
	Value string `json:"value,omitempty"` // "value": "custom attribute value"
}

// create profile from template
func (c *OVClient) CreateProfileFromTemplateWithI3S(name string, template ServerProfile, blade ServerHardware) error {
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
		new_template.Type = "ServerProfileV6"
		new_template.ServerProfileTemplateURI = template.URI // create relationship
		log.Debugf("new_template -> %+v", new_template)
	} else {
		return fmt.Errorf("Can't use v1 with image streamer.")
	}

	new_template.ServerHardwareURI = blade.URI
	new_template.ServerHardwareTypeURI = blade.ServerHardwareTypeURI
	new_template.Description += " " + name
	new_template.Name = name
	for i := 0; i < len(new_template.Connections); i++ {
		new_template.Connections[i].Boot.Priority = "NotBootable"
	}
	//return fmt.Errorf("%+v", new_template.Connections)

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

// CustomizeServer - use customizeserver when working with an already provisioned server
// server apply deployment job
type CustomizeServer struct {
	ProfileName               string            // name of server
	OSDeploymentBuildPlan     string          // name of the OS build plan
	OSDeploymentAttributes    map[string]string // name value pairs for server custom attributes

}

// CustomizeServer - Customize Server
func (c *OVClient) CustomizeServer(cs CustomizeServer) error {
	s, err := c.GetProfileByName(cs.ProfileName)
	if err != nil || s.URI.IsNil() {
		return fmt.Errorf("Server not found\n %+v", err)
	}

	blade, err := c.GetServerHardware(s.ServerHardwareURI)
	if err != nil {
		return err
	}

	osDeploymentPlan, err := c.GetOSDeploymentPlanByName(cs.OSDeploymentBuildPlan)
	if err != nil || osDeploymentPlan.URI.IsNil(){
		return fmt.Errorf("osDeploymentPlan not found: %s\n %+v", cs.OSDeploymentBuildPlan, err)
	}

	serverDeploymentAttributes := make([]OSCustomAttribute, len(osDeploymentPlan.AdditionalParameters))
    for i := 0; i < len(osDeploymentPlan.AdditionalParameters); i++{
    	customAttribute := &osDeploymentPlan.AdditionalParameters[i]
    	if val, ok := cs.OSDeploymentAttributes[customAttribute.Name]; ok {
    		customAttribute.Value = val
    	}
    	serverDeploymentAttributes[i].Name = customAttribute.Name
    	serverDeploymentAttributes[i].Value = customAttribute.Value
    }

    s.OSDeploymentSettings.OSDeploymentPlanUri = osDeploymentPlan.URI
	s.OSDeploymentSettings.OSCustomAttributes = serverDeploymentAttributes

	s.Connections, err = c.ManageI3SConnections(s.Connections)
	if err != nil{
		return err
	}

	err = c.UpdateServerProfile(s)
	if err != nil {
		return err
	}

	err = blade.PowerOn()
	if err != nil {
		return err
	}

	return nil
}

func (c *OVClient) DeleteOSBuildPlanFromServer(profileName string) error {
	s, err := c.GetProfileByName(profileName)
	if err != nil || s.URI.IsNil() {
		return fmt.Errorf("Server not found\n %+v", err)
	}

	blade, err := c.GetServerHardware(s.ServerHardwareURI)
	if err != nil {
		return err
	}

	err = blade.PowerOff()
	if err != nil {
		return err
	}

	s.OSDeploymentSettings.OSDeploymentPlanUri = utils.NewNstring("")
	//s.OSDeploymentSettings.OSCustomAttributes = nil

	err = c.UpdateServerProfile(s)
	if err != nil {
		return err
	}

	return nil
}