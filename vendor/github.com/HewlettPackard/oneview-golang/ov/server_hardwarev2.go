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
	"strings"

	"github.com/HewlettPackard/oneview-golang/liboneview"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// introduced in v200 for oneview, new v2 hardware attributes
// such as support for different types of ilo connections

// IsHardwareSchemaV2 - true when we are using v2, false for v1
func (c *OVClient) IsHardwareSchemaV2() bool {
	var currentversion liboneview.Version
	var asc liboneview.APISupport
	currentversion = currentversion.CalculateVersion(c.APIVersion, 108) // force icsp to 108 version since icsp version doesn't matter
	asc = asc.NewByName("server_hardwarev2.go")
	if asc.IsSupported(currentversion) {
		log.Debugf("IsHardwareSchemaV2 is supported: %+v", currentversion)
		return true
	}
	return false
}

// ServerHardwarev200 get server hardware from ov
// mphostinfo is private to force calls to GetIloIPAddress
type ServerHardwarev200 struct {
	IntelligentProvisioningVersion string              `json:"intelligentProvisioningVersion,omitempty"` // intelligentProvisioningVersion The installed version of the Intelligent Provisioning embedded server provisioning tool. string
	MpHostInfo                     *MpHostInfov200     `json:"mpHostInfo,omitempty"`                     // The host name and IP address information for the Management Processor that resides on this server.
	MpState                        string              `json:"mpState,omitempty"`                        //  Indicates the current state of the management processor.
	PortMap                        *PortMapv200        `json:"portMap,omitempty"`                        //  A list of adapters/slots, their ports and attributes. This information is available for blade servers but not rack servers.
	ServerSettings                 *ServerSettingsv200 `json:"serverSettings,omitempty"`                 //  Indicates the current settings on the server and state of these settings.
	Signature                      *Signaturev200      `json:"signature,omitempty"`                      // Data representing the current configuration or 'signature' of the server.
}

// TODO: needs a type
// PhysicalServerMPState
// Values
//     OK
//     Reset
//     Resetting

// MpHostInfov200 -
type MpHostInfov200 struct {
	MpHostName  string            `json:"mpHostName,omitempty"`    // mpHostName The host name of the Management Processor. string
	MpIPAddress []MpIPAddressv200 `json:"mpIpAddresses,omitempty"` // The list of IP addresses and corresponding type information for the Management Processor.
}

// MpIPAddressv200 -
type MpIPAddressv200 struct {
	Address string `json:"address,omitempty"` // address An IP address for the Management Processor. string
	Type    string `json:"type,omitempty"`    // type The type of IP address. The following are useful values for the IP address type: Static - Static IP address configuration; DHCP - Dynamic host configuration protocol; SLAAC - Stateless address autoconfiguration (IPv6); LinkLocal - Link-local address (IPv6);
}

// MpIPTypev200 Type constant
type MpIPTypev200 int

// const block
const (
	MpDHCP MpIPTypev200 = 1 + iota
	MpLinkLocal
	MpLinkLocalRequired
	MpLookup
	MpSlaaC
	MpStatic
	MpUndefined
)

var mpiptypevlist = [...]string{
	"DHCP",
	"LinkLocal",
	"LinkLocal_Required",
	"Lookup",
	"SLAAC",
	"Static",
	"Undefined",
}

// String helper for MpIPTypev200
func (o MpIPTypev200) String() string { return mpiptypevlist[o-1] }

// Equal helper for MpIPTypev200
func (o MpIPTypev200) Equal(s string) bool {
	return (strings.ToUpper(s) == strings.ToUpper(o.String()))
}

// PortMapv200 -
type PortMapv200 struct {
	DeviceSlots []DeviceSlotsv200 `json:"deviceSlots,omitempty"` //  List of each slot found on the server hardware. read only
}

// DeviceSlotsv200 -
type DeviceSlotsv200 struct {
	DeviceName    string             `json:"deviceName,omitempty"`    // deviceName The name or model of the adapter. string read only
	DeviceNumber  int                `json:"deviceNumber,omitempty"`  // deviceNumber The device number of a blade server in an enclosure. integer read only
	Location      string             `json:"location,omitempty"`      // location The location of the adapter in the server. Lom indicates LAN on motherboard, Flb is for FlexibleLOM for blades and Mezz is for Mezzanine adapters.
	PhysicalPorts []PhysicalPortv200 `json:"physicalPorts,omitempty"` // physicalPorts        //
	SlotNumber    int                `json:"slotNumber,omitempty"`    // slotNumber The slot number of the adapter on the server hardware within its specified location. integer read only
}

// PhysicalPortv200 -
type PhysicalPortv200 struct {
	InterconnectPort         int               `json:"interconnectPort,omitempty"`         // interconnectPort The downlink port number on the interconnect that hosts network connections for this adapter port. If the adapter port is not connected to an interconnect downlink port, the value will be 0. integer read only
	InterconnectURI          utils.Nstring     `json:"interconnectUri,omitempty"`          // interconnectUri The URI of the interconnect that hosts network connections for this adapter port. string read only
	MAC                      string            `json:"mac,omitempty"`                      // mac Physical mac address of this physical port. string read only
	PhysicalInterconnectPort int               `json:"physicalInterconnectPort,omitempty"` // physicalInterconnectPort The downlink port number on the interconnect to which the adapter port is physically connected. If the adapter port is not connected to an interconnect downlink port, the value will be 0. integer read only
	PhysicalInterconnectURI  utils.Nstring     `json:"physicalInterconnectUri,omitempty"`  // physicalInterconnectUri The URI of the interconnect to which the adapter port is physically connected. string read only
	PortNumber               int               `json:"portNumber,omitempty"`               // portNumber Physical port number of the adapter. integer read only
	Type                     string            `json:"type,omitempty"`                     // type Physical port type. Values include Ethernet and FibreChannel. Physical Server Port Type read only
	VirtualPorts             []VirtualPortv200 `json:"virtualPorts,omitempty"`             // For Flex-capable devices, a list of FlexNICs defined on the server. array of Server Fabric Virtual Port read only
	WWN                      string            `json:"wwn,omitempty"`                      // wwn The world wide name of this physical port. string read only
}

// TODO : implement const for type on PhysicalPortv200
// Ethernet Ethernet network
// FibreChannel Fibre Channel network
// InfiniBand

// VirtualPortv200 -
type VirtualPortv200 struct {
	CurrentAllocatedVirtualFunctionCount int    `json:"currentAllocatedVirtualFunctionCount,omitempty"` // currentAllocatedVirtualFunctionCount The number of virtual functions presently allocated to this virtual port. integer read only
	MAC                                  string `json:"mac,omitempty"`                                  // mac The mac address assigned to this virtual port. string read only
	PortFunction                         string `json:"portFunction,omitempty"`                         // portFunction The function identifier for this FlexNIC, such as a, b, c or d. string read only
	PortNumber                           int    `json:"portNumber,omitempty"`                           // portNumber The port number assigned to this virtual port. integer read only
	WWNN                                 string `json:"WWNN,omitempty"`                                 // wwnn The world wide node name assigned to this virtual port. string read only
	WWPN                                 string `json:"WWPN,omitempty"`                                 // wwpn The world wide port name assigned to this virtual port. string read only
}

// ServerSettingsv200 -
type ServerSettingsv200 struct {
	FirmwareAndDriversInstallState *FirmwareAndDriversInstallStatev200 `json:"firmwareAndDriversInstallState,omitempty"` //  firmwareAndDriversInstallState The current installation status details of the firmware and/or OS drivers on the server.
	HPSmartUpdateToolStatus        *HpSmartUpdateToolStatusv200        `json:"hpSmartUpdateToolStatus,omitempty"`        // The status of HP Smart Update Tool installed on the server.
}

// FirmwareAndDriversInstallStatev200 -
type FirmwareAndDriversInstallStatev200 struct {
	InstallState            string `json:"installState,omitempty"`            // installState The installation state information of the firmware and/or OS drivers on the server.
	InstalledStateTimestamp string `json:"installedStateTimestamp,omitempty"` // The timestamp information indicating the time when the install state value got updated. string
}

//TODO: const for FirmwareInstallState
// ActivateFailed Indicates activation of one or more smart components failed on the server.
// Activated Indicates all the smart components from the SPP bundle specified in the firmware and driver baseline settings are installed and activated on the server.
// Activating Indicates HP Smart Update Tool is activating the installed smart components.
// InstallFailed Indicates HP Smart Update Tool has failed installing one or more smart components.
// InstalledPendingReboot Indicates HP Smart Update Tool has completed installing the smart components, however the server needs to be rebooted for the updates to take effect.
// Installing Indicates HP Smart Update Tool is installing the smart components from the staged location.
// Pending Indicates the firmware and/or OS driver baseline settings have been applied to the server hardware and will take effect when HP Smart Update tool updates firmware and/or OS driver components based on these settings.
// StageFailed Indicates HP Smart Update Tool has failed to stage the smart components from the SPP bundle specified in the firmware and/or OS driver baseline settings.
// Staged Indicates HP Smart Update Tool has completed staging the smart components from the SPP bundle specified in the firmware and/or OS driver baseline settings.
// Staging Indicates HP Smart Update Tool is staging the firmware and/or OS driver smart components from the SPP bundle specified in the firmware and/or OS driver baseline settings.
// Uninitialized Indicates the current firmware and/or OS driver settings have been cleared. No components will be updated on the server.
// Unknown Indicates the server failed to return the current settings, therefore, the actual values might not be current.

type HpSmartUpdateToolStatusv200 struct {
	HPSUTInstallState string `json:"hpSUTInstallState,omitempty"` // hpSUTInstallState HpSUT Install State Enum
	InstallState      string `json:"installState,omitempty"`      // installState HP Smart Update Tool's installed state on the server indicating whether it is installed, not installed or unknown.
	LastOperationTime string `json:"lastOperationTime,omitempty"` // lastOperationTime The timestamp indicating when HP Smart Update Tool was active or running a firmware update operation. string
	Mode              string `json:"mode,omitempty"`              // mode The current run mode configured for HP Smart Update Tool installed on the server. string
	ServiceState      string `json:"serviceState,omitempty"`      // serviceState The state of the HP Smart Update Tool Service running on the server operating system. string
	Version           string `json:"version,omitempty"`           // version The current version of the HP Smart Update Tool installed on the server. string
}

// Signaturev200 -
type Signaturev200 struct {
	PersonalityChecksum int `json:"personalityChecksum,omitempty"` // A calculated checksum of the server 'personality,' based on the defined connections and server identifiers. integer read only
	ServerHwChecksum    int `json:"serverHwChecksum,omitempty"`    // A calculated checksum of the server hardware, based on the hardware components installed in the server. integer read only
}
