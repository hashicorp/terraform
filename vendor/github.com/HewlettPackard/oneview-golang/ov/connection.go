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
	"github.com/HewlettPackard/oneview-golang/utils"
)

// BootTarget -
type BootTarget struct {
	ArrayWWPN string `json:"arrayWwpn,omitempty"` // arrayWwpn(string,required),The wwpn of the target device that provides access to the Boot Volume. This value must contain 16 HEX digits.
	LUN       string `json:"lun,omitempty"`       // lun(string,required), The LUN of the Boot Volume presented by the target device. This value can be either 1 to 3 decimal digits in the range 0 to 255, or 13 to 16 hex digits with no other characters
}

// BootOption -
type BootOption struct {
	BootOptionV3
	Priority string       `json:"priority,omitempty"` // priority(const_string), indicates the boot priority for this device. PXE and Fibre Channel connections are treated separately; an Ethernet connection and a Fibre Channel connection can both be marked as Primary. The 'order' attribute controls ordering among the different device types.
	Targets  []BootTarget `json:"targets,omitempty"`  // targets {BootTarget}
}

// Connection server profile object for ov
type Connection struct {
	Connectionv200
	AllocatedMbps    int           `json:"allocatedMbps,omitempty"`    // allocatedMbps(int:read), The transmit throughput (mbps) currently allocated to this connection. When Fibre Channel connections are set to Auto for requested bandwidth, the value can be set to -2000 to indicate that the actual value is unknown until OneView is able to negotiate the actual speed.
	Boot             BootOption    `json:"boot,omitempty"`             // boot {}
	DeploymentStatus string        `json:"deploymentStatus,omitempty"` // deploymentStatus(const_string:read), The deployment status of the connection. The value can be 'Undefined', 'Reserved', or 'Deployed'.
	FunctionType     string        `json:"functionType,omitempty"`     // functionType(const_string),  Type of function required for the connection. functionType cannot be modified after the connection is created. 'Ethernet', 'FibreChannel'
	ID               int           `json:"id,omitempty"`               // id(int), A unique identifier for this connection. When creating or editing a profile, an id is automatically assigned if the attribute is omitted or 0 is specified. When editing a profile, a connection is created if the id does not identify an existing connection.
	InterconnectURI  utils.Nstring `json:"interconnectUri,omitempty"`  // interconnectUri(Nstring:read), The interconnectUri associated with the connection.
	MAC              utils.Nstring `json:"mac,omitempty"`              // mac(Nstring), The MAC address that is currently programmed on the FlexNic. The value can be a virtual MAC, user defined MAC or physical MAC read from the device. It cannot be modified after the connection is created.
	MacType          string        `json:"macType,omitempty"`          // macType(const_string), Physical, UserDefined, Virtual
	MaximumMbps      int           `json:"maximumMbps,omitempty"`      // maximumMbps(int:read),  Maximum transmit throughput (mbps) allowed on this connection. The value is limited by the maximum throughput of the network link and maximumBandwidth of the selected network (networkUri). For Fibre Channel connections, the value is limited to the same value as the allocatedMbps.
	Name             string        `json:"name,omitempty"`             // name(string), A string used to identify the respective connection. The connection name is case insensitive, limited to 63 characters and must be unique within the profile.
	NetworkURI       utils.Nstring `json:"networkUri,omitempty"`       // networkUri(Nstring, required), Identifies the network or network set to be connected. Use GET /rest/server-profiles/available-networks to retrieve the list of available Ethernet networks, Fibre Channel networks and network sets that are available along with their respective ports.
	PortID           string        `json:"portId,omitempty"`           // portId(string), Identifies the port (FlexNIC) used for this connection, for example 'Flb 1:1-a'. The port can be automatically selected by specifying 'Auto', 'None', or a physical port when creating or editing the connection. If 'Auto' is specified, a port that provides access to the selected network (networkUri) will be selected. A physical port (e.g. 'Flb 1:2') can be specified if the choice of a specific FlexNIC on the physical port is not important. If 'None' is specified, the connection will not be configured on the server hardware. When omitted, portId defaults to 'Auto'. Use /rest/server-profiles/profile-ports to retrieve the list of available ports.
	RequestedMbps    string        `json:"requestedMbps,omitempty"`    // requestedMbps(string), The transmit throughput (mbps) that should be allocated to this connection. For FlexFabric connections, this value must not exceed the maximum bandwidth of the selected network (networkUri). If omitted, this value defaults to the typical bandwidth value of the selected network. The sum of the requestedBW values for the connections (FlexNICs) on an adapter port cannot exceed the capacity of the network link. For Virtual Connect Fibre Channel connections, the available discrete values are based on the adapter and the Fibre Channel interconnect module. Use GET /rest/server-profiles/profile-ports to retrieve the list of available ports and the acceptable bandwidth values for the ports.
	WWNN             utils.Nstring `json:"wwnn,omitempty"`             // wwnn(Nstring), The node WWN address that is currently programmed on the FlexNic. The value can be a virtual WWNN, user defined WWNN or physical WWNN read from the device. It cannot be modified after the connection is created.
	WWPN             utils.Nstring `json:"wwpn,omitempty"`             // wwpn(Nstring), The port WWN address that is currently programmed on the FlexNIC. The value can be a virtual WWPN, user defined WWPN or the physical WWPN read from the device. It cannot be modified after the connection is created.
	WWPNType         string        `json:"wwpnType,omitempty"`         // wwpnType(const_string), Physical, UserDefined, Virtual
}

// Clone clone connection
func (c Connection) Clone() Connection {
	return Connection{
		Boot:          c.Boot,
		FunctionType:  c.FunctionType,
		ID:            c.ID,
		MacType:       c.MacType,
		Name:          c.Name,
		NetworkURI:    c.NetworkURI,
		PortID:        c.PortID,
		RequestedMbps: c.RequestedMbps,
		WWPNType:      c.WWPNType,
	}
}
