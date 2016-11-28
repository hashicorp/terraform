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
	"errors"
	"fmt"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/docker/machine/libmachine/log"
)

// OVClient - wrapper class for ov api's
type OVClient struct {
	rest.Client
}

// new Client
func (c *OVClient) NewOVClient(user string, password string, domain string, endpoint string, sslverify bool, apiversion int) *OVClient {
	return &OVClient{
		rest.Client{
			User:       user,
			Password:   password,
			Domain:     domain,
			Endpoint:   endpoint,
			SSLVerify:  sslverify,
			APIVersion: apiversion,
			APIKey:     "none",
		},
	}
}

// Create machine
func (c *OVClient) CreateMachine(host_name string, server_template string) (err error) {
	var (
		pt       *PowerTask
		bladep   ServerProfile
		blade    ServerHardware
		template ServerProfile
	)
	// check if the profile exist with host_name
	if bladep, err = c.GetProfileByName(host_name); err != nil {
		log.Errorf("Error unable to get blade by name: %s", err)
		return err
	}
	if bladep.ServerHardwareURI != "" {
		// Template already exist, power it on and continue
		// Power on the server profile if it exist
		if blade, err = c.GetServerHardware(bladep.ServerHardwareURI); err != nil {
			log.Errorf("Error in getting server hardware from uri, %s", err)
			return err
		}
		pt = pt.NewPowerTask(blade)
		if err = pt.PowerExecutor(P_OFF); err != nil {
			log.Errorf("Unable to power off blade, %s, Error: %s", blade.Name, err)
			return err
		}
		return nil
	}

	// check for a server profile template name, if it doesn't exist, exit
	if template, err = c.GetProfileTemplateByName(server_template); err != nil {
		log.Errorf("Error unable to get template by name (%s): %s", server_template, err)
		return err
	}
	if template.Name != server_template {
		return errors.New(fmt.Sprintf("Error template name not found, %s.", server_template))
	}

	// get the template : uri ?? not sure where used

	// get available hardware
	log.Debugf("*** GetAvailableHardware")
	blade, err = c.GetAvailableHardware(template.ServerHardwareTypeURI, template.EnclosureGroupURI)
	if err != nil {
		log.Errorf("Error with getting available hardware: %s", err)
		return err
	}

	// load that available hardware
	blade, err = c.GetServerHardware(blade.URI)
	if err != nil {
		return err
	}

	log.Debugf("*** Blade => %+v", blade)
	log.Debugf("client 3 *******---> %+v", blade.Client.APIKey)
	// now we have a server_hardware object...
	// Power off the blade, so we can provision the server
	pt = pt.NewPowerTask(blade)
	if err = pt.PowerExecutor(P_OFF); err != nil {
		log.Errorf("Unable to power off blade, %s, Error: %s", blade.Name, err)
		return err
	}

	// create a template with the new blade
	if err = c.CreateProfileFromTemplate(host_name, template, blade); err != nil {
		log.Errorf("Error creating a new profile from template: %s", err)
		return err
	}
	// matching_profiles['members'].first ?? not sure where used
	return nil
}
