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

// Package icsp -
package icsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// ICSPClient - wrapper class for icsp api's
type ICSPClient struct {
	rest.Client
}

// new Client
func (c *ICSPClient) NewICSPClient(user string, password string, domain string, endpoint string, sslverify bool, apiversion int) *ICSPClient {
	return &ICSPClient{
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

// CustomServerAttributes setup custom attributes to apply to a server
type CustomServerAttributes struct {
	Values map[string]string // Hash of strings for custom attributes
}

func (cs *CustomServerAttributes) New() *CustomServerAttributes {
	return &CustomServerAttributes{
		Values: make(map[string]string),
	}
}

func (cs *CustomServerAttributes) Set(key string, value string) {
	cs.Values[key] = value
}

func (cs *CustomServerAttributes) Get(key string) string {
	return cs.Values[key]
}

// CustomizeServer - use customizeserver when working with creating a new server
// server create if it's missing
// server apply deployment job
type CustomizeServer struct {
	HostName         string                  // provide a hostname to set
	SerialNumber     string                  // should be the serial number for the server
	ILoUser          string                  // should be the user name for ilo administration
	IloPassword      string                  // should be the ilo password to use
	IloIPAddress     string                  // PXE ip address for ilo
	IloPort          int                     // port number for ilo server
	OSBuildPlans     []string                // name of the OS build plan
	ServerProperties *CustomServerAttributes // name value pairs for server custom attributes
	PublicSlotID     int                     // the public interface that will be used to get public ipaddress
	PublicMAC        string                  // public connection name, overrides PublicSlotID
}

// PostApplyDeploymentJobs - post deployment task to update custom attributes with
// results of a job task that was executed on the server
func (c *ICSPClient) PostApplyDeploymentJobs(jt *JobTask, s Server, properties []string) error {
	// look at jobResultLogDetails, parse *=* strings
	job, err := c.GetJob(jt.JobURI)
	if err != nil {
		return err
	}
	// parses the provisioning log to set any attributes from output of the log
	for _, result := range job.JobResult {
		for _, line := range strings.Split(result.JobResultLogDetails, "\n") {
			r := regexp.MustCompile("(.*)=(.*)")
			if r.FindString(line) != "" {
				for _, property := range properties {
					a := r.FindStringSubmatch(line)
					if len(a) >= 3 && property == a[1] {
						s.SetCustomAttribute(a[1], "server", a[2])
					}
				}
			}
		}
	}

	// do netconfiguration for all interfaces on server s
	var netconfig NetConfig
	var emptyconfig utils.Nstring
	emptyconfig.Nil()
	// TODO: determine configuration options for network customization
	netconfig = NewNetConfig(emptyconfig, //s.HostName,
		emptyconfig, // workgroup utils.Nstring,
		emptyconfig, // domain utils.Nstring,
		emptyconfig, // winslist utils.Nstring,
		emptyconfig, // dnsnamelist utils.Nstring,
		emptyconfig) // dnssearchlist utils.Nstring)
	netconfig.AddAllDHCP(s.Interfaces, false, emptyconfig) // TODO: could use a option for ipv6
	s, err = netconfig.Save(s)
	if err != nil {
		return err
	}
	// place those strings into custom attributes
	s, err = c.SaveServer(s)
	if err != nil {
		return err
	}

	// apply os build plan customizations for netconfig
	buildplans := make([]string, 1)
	buildplans[0] = "ProLiant SW - Post Install Network Personalization"

	_, err = c.ApplyDeploymentJobs(buildplans, netconfig.GetPersonalityData(), s)
	if err != nil {
		return err
	}

	// update public_interface
	s, err = s.ReloadFull(c)
	if err != nil {
		return err
	}

	// get the existing mac address for public interface
	inet, err := s.GetPublicInterface()
	if err != nil {
		return err
	}
	pubinet, err := s.GetInterfaceFromMac(inet.MACAddr)
	// re-save interface to public_interface
	s, err = c.UpdatePublicInterfaceAttributes(s, pubinet)
	log.Debugf("Server settings s after post deploy -> %+v", s)
	return err
}

// UpdatePublicInterfaceAttributes - updates the server attributes with public interface
func (c *ICSPClient) UpdatePublicInterfaceAttributes(s Server, publicinterface Interface) (Server, error) {
	publicinterfacejson, err := json.Marshal(publicinterface)
	if err != nil {
		return s, err
	}
	// save the publicinterface into a custom attribute called public_interface
	s.SetCustomAttribute("public_interface", "server", fmt.Sprintf("%s", bytes.NewBuffer(publicinterfacejson)))
	s.SetCustomAttribute("interface", "server", publicinterface.Slot)

	// save it
	s, err = c.SaveServer(s)
	return s, err
}

// PreApplyDeploymentJobs - update public interface information with what is actively the public interface
func (c *ICSPClient) PreApplyDeploymentJobs(s Server, publicinterface Interface) (Server, error) {
	return c.UpdatePublicInterfaceAttributes(s, publicinterface)
}

// CustomizeServer - Customize Server
func (c *ICSPClient) CustomizeServer(cs CustomizeServer) error {
	s, err := c.GetServerBySerialNumber(cs.SerialNumber)
	if err != nil {
		return err
	}
	if s.SerialNumber != cs.SerialNumber {
		log.Infof("ICSP creating server for : %s", cs.IloIPAddress)
		if err := c.CreateServer(cs.ILoUser, cs.IloPassword, cs.IloIPAddress, cs.IloPort); err != nil {
			return err
		}
		// reload that server
		s, err = c.GetServerBySerialNumber(cs.SerialNumber)
		if err != nil {
			return err
		}
	} else {
		log.Infof("ICSP server was already created, %s, skipping", cs.HostName)
	}

	// verify that the server actually has a URI
	if s.URI.IsNil() {
		return fmt.Errorf("Server customization failure, unable to get a valid server from icsp for serial number: %s", cs.SerialNumber)
	}

	// verify that the server retrieved matches it's serial number
	if s.SerialNumber != cs.SerialNumber {
		return fmt.Errorf("Server customization failure, server serial numbers mismatch for %s.", cs.SerialNumber)
	}

	// handle getting interface name
	var publicinterface Interface
	if cs.PublicMAC != "" {
		publicinterface, err = s.GetInterfaceFromMac(cs.PublicMAC)
		if err != nil {
			return err
		}
	} else {
		publicinterface, err = s.GetInterface(cs.PublicSlotID)
		if err != nil {
			return err
		}
	}

	// save the server attributes to the server
	for k, v := range cs.ServerProperties.Values {
		// handle sepecial custom attributes
		// handle @server_name@ and replace for s.Name
		v = strings.Replace(v, "@server_name@", s.Name, -1)
		v = strings.Replace(v, "@interface@", publicinterface.Slot, -1)
		s.SetCustomAttribute(k, "server", v)
	}

	// save it
	newserver, err := c.SaveServer(s)
	if err != nil {
		return err
	}

	// call to capture the public_interface attribute
	newserver, err = c.PreApplyDeploymentJobs(newserver, publicinterface)
	if err != nil {
		return err
	}

	// apply the build Plan
	jt, err := c.ApplyDeploymentJobs(cs.OSBuildPlans, nil, newserver)
	if err != nil {
		return err
	}

	// use jt to get additional customizations we can use on the server custom attributes
	// TODO: this needs to be evaluated on usefull ness and proper way to pass up additional deployment information back to the server in icsp
	var findprops []string
	findprops = append(findprops, "public_ip")
	return c.PostApplyDeploymentJobs(jt, newserver, findprops)
}
