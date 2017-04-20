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

// LogicalDrive logical drive options
type LogicalDrive struct {
	Bootable  bool   `json:"bootable"`            // "bootable": true,
	RaidLevel string `json:"raidLevel,omitempty"` // "raidLevel": "RAID1"
}

// LocalStorageOptions -
type LocalStorageOptions struct { // "localStorage": {
	LocalStorageSettingsV3
	ManageLocalStorage bool           `json:"manageLocalStorage,omitempty"` // "manageLocalStorage": true,
	LogicalDrives      []LogicalDrive `json:"logicalDrives,omitempty"`      // "logicalDrives": [],
	Initialize         bool           `json:"initialize,omitempty"`         // 				"initialize": true
}

// Clone local storage
func (c LocalStorageOptions) Clone() LocalStorageOptions {
	return LocalStorageOptions{
		ManageLocalStorage: c.ManageLocalStorage,
		LogicalDrives:      c.LogicalDrives,
		Initialize:         c.Initialize,
	}
}

// StoragePath storage path host-to-target paths
//Use GET /rest/storage-systems/{arrayid}/managedPorts?query="expectedNetworkUri EQ '/rest/fc-networks/{netowrk-id}'"
//to retrieve the storage targets for the associated network.
type StoragePath struct {
	ConnectionID      int      `json:"connectionId,omitempty"`      // connectionId (required), The ID of the connection associated with this storage path. Use GET /rest/server-profiles/available-networks to retrieve the list of available networks.
	IsEnabled         bool     `json:"isEnabled"`                   // isEnabled (required), Identifies whether the storage path is enabled.
	Status            string   `json:"status,omitempty"`            // status (read only), The overall health status of the storage path.
	StorageTargetType string   `json:"storageTargetType,omitempty"` // storageTargetType ('Auto', 'TargetPorts')
	StorageTargets    []string `json:"storageTargets,omitempty"`    // only set when storageTargetType is TargetPorts
}

// Clone -
func (c StoragePath) Clone() StoragePath {
	return StoragePath{
		ConnectionID:      c.ConnectionID,
		IsEnabled:         c.IsEnabled,
		StorageTargetType: c.StorageTargetType,
		StorageTargets:    c.StorageTargets,
	}
}

// VolumeAttachment volume attachment
type VolumeAttachment struct {
	VolumeAttachmentV3
	ID                             int           `json:"id,omitempty"`                             // id, The ID of the attached storage volume.
	LUN                            string        `json:"lun,omitempty"`                            // lun, The logical unit number.
	LUNType                        string        `json:"lunType,omitempty"`                        // lunType(required), The logical unit number type: Auto or Manual.
	Permanent                      bool          `json:"permanent"`                                // permanent, If true, indicates that the volume will persist when the profile is deleted. If false, then the volume will be deleted when the profile is deleted.
	State                          string        `json:"state,omitempty"`                          // state(read only), current state of the attachment
	Status                         string        `json:"status,omitempty"`                         // status(read only), The current status of the attachment.
	StoragePaths                   []StoragePath `json:"storagePaths,omitempty"`                   // A list of host-to-target path associations.
	VolumeDescription              string        `json:"volumeDescription,omitempty"`              // The description of the storage volume.
	VolumeName                     string        `json:"volumeName,omitempty"`                     // The name of the volume. This attribute is required when creating a volume.
	VolumeProvisionType            string        `json:"volumeProvisionType,omitempty"`            // The provisioning type of the new volume: Thin or Thick. This attribute is required when creating a volume.
	VolumeProvisionedCapacityBytes string        `json:"volumeProvisionedCapacityBytes,omitempty"` // The requested provisioned capacity of the storage volume in bytes. This attribute is required when creating a volume.
	VolumeShareable                bool          `json:"volumeShareable"`                          // Identifies whether the storage volume is shared or private. If false, then the volume will be private. If true, then the volume will be shared. This attribute is required when creating a volume.
	VolumeStoragePoolURI           utils.Nstring `json:"volumeStoragePoolUri,omitempty"`           // The URI of the storage pool associated with this volume attachment's volume. Use GET /rest/server-profiles/available-storage-systems to retrieve the URI of the storage pool associated with a volume.
	VolumeStorageSystemURI         utils.Nstring `json:"volumeStorageSystemUri,omitempty"`         // The URI of the storage system associated with this volume attachment. Use GET /rest/server-profiles/available-storage-systems to retrieve the URI of the storage system associated with a volume.
	VolumeURI                      utils.Nstring `json:"volumenUri,omitempty"`                     // The URI of the storage volume associated with this volume attachment. Use GET /rest/server-profiles/available-storage-systems to retrieve the URIs of available storage volumes.
}

// Clone clone volume attachment for submits
func (c VolumeAttachment) Clone() VolumeAttachment {
	var sp []StoragePath
	for _, s := range c.StoragePaths {
		sp = append(sp, s.Clone())
	}
	return VolumeAttachment{
		ID:                             c.ID,
		LUN:                            c.LUN,
		LUNType:                        c.LUNType,
		Permanent:                      c.Permanent,
		StoragePaths:                   sp,
		VolumeDescription:              c.VolumeDescription,
		VolumeName:                     c.VolumeName,
		VolumeProvisionType:            c.VolumeProvisionType,
		VolumeProvisionedCapacityBytes: c.VolumeProvisionedCapacityBytes,
		VolumeShareable:                c.VolumeShareable,
		VolumeStoragePoolURI:           c.VolumeStoragePoolURI,
		VolumeStorageSystemURI:         c.VolumeStorageSystemURI,
		VolumeURI:                      c.VolumeURI,
	}
}

// SanStorageOptions pecify san storage
// No San
// 		"sanStorage": {
// 				"volumeAttachments": [],
// 				"manageSanStorage": false
// 		},
type SanStorageOptions struct { // sanStorage
	SanStorageV3
	HostOSType            string             `json:"hostOSType,omitempty"`            // hostOSType(required),  The operating system type of the host. To retrieve the list of supported host OS types, issue a REST Get request using the /rest/storage-systems/host-types API.
	ManageSanStorage      bool               `json:"manageSanStorage"`                // manageSanStorage(required),  Identifies whether SAN storage is managed in this profile.
	VolumeAttachments     []VolumeAttachment `json:"volumeAttachments,omitempty"`     // volumeAttachments, The list of storage volume attachments. array of Volume Attachment
	SerialNumber          string             `json:"serialNumber,omitempty"`          // serialNumber (searchable) A 10-byte value that is exposed to the Operating System as the server hardware's Serial Number. The value can be a virtual serial number, user defined serial number or physical serial number read from the server's ROM. It cannot be modified after the profile is created.
	SerialNumberType      string             `json:"serialNumberType,omitempty"`      // serialNumberType (searchable) Specifies the type of Serial Number and UUID to be programmed into the server ROM. The value can be 'Virtual', 'UserDefined', or 'Physical'. The serialNumberType defaults to 'Virtual' when serialNumber or uuid are not specified. It cannot be modified after the profile is created.
	ServerHardwareTypeURI utils.Nstring      `json:"serverHardwareTypeUri,omitempty"` // serverHardwareTypeUri Identifies the server hardware type for which the Server Profile was designed. The serverHardwareTypeUri is determined when the profile is created and cannot be modified. Use GET /rest/server-hardware-types to retrieve the list of server hardware types.
	ServerHardwareURI     utils.Nstring      `json:"serverHardwareUri,omitempty"`     // serverHardwareUri Identifies the server hardware to which the server profile is currently assigned, if applicable. Use GET /rest/server-profiles/available-targets to retrieve the list of available servers.
	State                 string             `json:"state,omitempty"`                 // state (searchable, readonly) Current State of this Server Profile
	Status                string             `json:"status,omitempty"`                // status (searchable, readonly) Overall health status of this Server Profile
	TaskURI               utils.Nstring      `json:"taskUri,omitempty"`               // taskUri (read only) URI of the task currently executing or most recently executed on this server profile.
	Type                  string             `json:"type,omitempty"`                  // type (read only) Identifies the resource type. This field should always be set to 'ServerProfileV4'.
	URI                   utils.Nstring      `json:"uri,omitempty"`                   // uri (searchable, readonly) URI of this Server Profile. The URI is automatically generated when the server profile is created and cannot be modified.
	UUID                  string             `json:"uuid,omitempty"`                  // uuid (searchable) A 36-byte value that is exposed to the Operating System as the server hardware's UUID. The value can be a virtual uuid, user defined uuid or physical uuid read from the server's ROM. It cannot be modified after the profile is created.
	WWNType               string             `json:"wwnType,omitempty"`               // wwnType (searchable) Specifies the type of WWN address to be programmed into the IO devices. The value can be 'Virtual' or 'Physical'. It cannot be modified after the profile is created.
}

// Clone clone local storage for submitting
func (c SanStorageOptions) Clone() SanStorageOptions {
	var va []VolumeAttachment
	for _, v := range c.VolumeAttachments {
		va = append(va, v.Clone())
	}
	return SanStorageOptions{
		HostOSType:        c.HostOSType,
		ManageSanStorage:  c.ManageSanStorage,
		VolumeAttachments: va,
	}
}
