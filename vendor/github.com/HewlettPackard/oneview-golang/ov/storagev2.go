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

	"github.com/HewlettPackard/oneview-golang/utils"
)

// LocalStorageSettingsV3 -
type LocalStorageSettingsV3 struct { // "localStorage": {
	LocalStorageSettingsV4
	Controllers []LocalStorageEmbeddedController `json:"controllers,omitempty"` //  The list of embedded local storage controllers.
}

// LocalStorageEmbeddedController -
type LocalStorageEmbeddedController struct {
	LocalStorageEmbeddedControllerV4
	ImportConfiguration bool             `json:"importConfiguration,omitempty"` // importConfiguration Determines if the logical drives in the current configuration should be imported. Boolean
	Initialize          bool             `json:"initialize,omitempty"`          // initialize Determines if the controller should be initialized before configuration. Boolean
	LogicalDrives       []LogicalDriveV3 `json:"logicalDrives,omitempty"`       // LogicalDriveV3 The list of logical drives associated with the controller.
	Managed             bool             `json:"managed,omitempty"`             // managed Determines if the specific controller is managed by OneView. Boolean
	Mode                string           `json:"mode,omitempty"`                // mode Determines the mode of operation of the controller. The controller mode can be changed between RAID and HBA mode when supported by the selected server hardware type. string
	SlotNumber          string           `json:"slotNumber,omitempty"`          // slotNumber The PCI slot number used by the controller. This value will always be set to "0", as only the embedded controller is supported in the current version. string
}

// LogicalDriveV3 -
type LogicalDriveV3 struct {
	LogicalDriveV4
	Bootable          bool   `json:"bootable,omitempty"`          // bootable Indicates if the logical drive is bootable or not. Boolean
	DriveName         string `json:"driveName,omitempty"`         // driveName The name of the logical drive. string
	DriveNumber       int    `json:"driveNumber,omitempty"`       // driveNumber The number assigned to the logical drive by HP SSA. This value is read-only and gets automatically populated once the logical drive is created. integer read only
	DriveTechnology   string `json:"driveTechnology,omitempty"`   // driveTechnology Defines the interface type for drives that will be used to build the logical drive. Supported values depend on the local storage capabilities of the selected server hardware type. string
	NumPhysicalDrives int    `json:"numPhysicalDrives,omitempty"` // numPhysicalDrives The number of physical drives to be used to build the logical drive. The provided values must be consistent with the selected RAID level and cannot exceed the maximum supported number of drives for the selected server hardware type. integer
	RaidLevel         string `json:"raidLevel,omitempty"`         // raidLevel The RAID level of the logical drive. Supported values depend on the local storage capabilities of the selected server hardware type. string
}

// SanStorageV3 -
type SanStorageV3 struct {
	HostOSType        string               `json:"hostOSType,omitempty"`        // hostOSType The operating system type of the host. To retrieve the list of supported host OS types, issue a REST Get request using the /rest/storage-systems/host-types API. string required
	ManageSanStorage  bool                 `json:"manageSanStorage"`            // manageSanStorage Identifies whether SAN storage is managed in this profile. Boolean required
	VolumeAttachments []VolumeAttachmentV2 `json:"volumeAttachments,omitempty"` // The list of storage volume attachments.
}

// VolumeAttachmentV2 The list of storage volume attachments.
type VolumeAttachmentV2 struct {
	ID                             int             `json:"id,omitempty"`                             // id The ID of the attached storage volume. integer
	LUN                            string          `json:"lun,omitempty"`                            // lun The logical unit number. string
	LUNType                        string          `json:"lunType,omitempty"`                        // lunType The logical unit number type: Auto or Manual. string required
	Permanent                      bool            `json:"permanent"`                                // permanent If true, indicates that the volume will persist when the profile is deleted. If false, then the volume will be deleted when the profile is deleted. Boolean
	State                          string          `json:"state,omitempty"`                          //state The current state of the attachment. VolumeAttachmentStateV2 read only
	Status                         string          `json:"status,omitempty"`                         // status The current status of the attachment. string read only
	StoragePaths                   []StoragePathV2 `json:"storagePaths,omitempty"`                   // A list of host-to-target path associations.
	VolumeDescription              string          `json:"volumeDescription,omitempty"`              // volumeDescription The description of the storage volume. string
	VolumeName                     string          `json:"volumeName,omitempty"`                     // volumeName The name of the volume. This attribute is required when creating a volume. string
	VolumeProvisionType            string          `json:"volumeProvisionType,omitempty"`            // volumeProvisionType The provisioning type of the new volume: Thin or Thick. This attribute is required when creating a volume. string
	VolumeProvisionedCapacityBytes string          `json:"volumeProvisionedCapacityBytes,omitempty"` // volumeProvisionedCapacityBytes The requested provisioned capacity of the storage volume in bytes. This attribute is required when creating a volume. string
	VolumeShareable                bool            `json:"volumeShareable"`                          // volumeShareable Identifies whether the storage volume is shared or private. If false, then the volume will be private. If true, then the volume will be shared. This attribute is required when creating a volume. Boolean
	VolumeStoragePoolURI           utils.Nstring   `json:"volumeStoragePoolUri,omitempty"`           // volumeStoragePoolUri The URI of the storage pool associated with this volume attachment's volume. Use GET /rest/server-profiles/available-storage-systems to retrieve the URI of the storage pool associated with a volume. string
	VolumeStorageSystemURI         utils.Nstring   `json:"volumeStorageSystemUri,omitempty"`         // volumeStorageSystemUri The URI of the storage system associated with this volume attachment. Use GET /rest/server-profiles/available-storage-systems to retrieve the URI of the storage system associated with a volume. string Format URI
	VolumeURI                      utils.Nstring   `json:"volumeUri,omitempty"`                      // volumeUri The URI of the storage volume associated with this volume attachment. Use GET /rest/server-profiles/available-storage-systems to retrieve the URIs of available storage volumes. string Format URI
}

// VolumeAttachmentStateV2 -
type VolumeAttachmentStateV2 int

// Constants for VolumeAttachmentStateV2
const (
	VAAttachFailed VolumeAttachmentStateV2 = 1 + iota
	VAAttached
	VAAttaching
	VACreating
	VADeleteFailed
	VADeleted
	VADeleting
	VAReserveFailed
	VAReserved
	VAReserving
	VAUpdating
	VAUserDeleted
	VAVolumeCreateFailed
	VAVolumeCreating
)

var volumeattachmentstatev2list = [...]string{
	"AttachFailed",       // An error occurred attempting to attach the volume in HP OneView or 3Par.
	"Attached",           // The volume attachment has been created or updated in HP OneView and 3Par.
	"Attaching",          // The volume attachment on an assigned profile is being created or updated in HP OneView and 3Par.
	"Creating",           // The volume attachment is being created.
	"DeleteFailed",       // An error occurred removing the volume attachment from HP OneView or 3Par.
	"Deleted",            // The volume attachment has been deleted.
	"Deleting",           // The volume attachment is being removed from HP OneView and 3Par.
	"ReserveFailed",      // An error occurred creating or updating he volume attachment as 'unassigned'.
	"Reserved",           // The volume attachment has been created or updated as 'unassigned' in HP OneView and 3Par.
	"Reserving",          // The volume attachment on an unassigned profile is being created or updated as 'unassigned'.
	"Updating",           // The volume attachment is being updated.
	"UserDeleted",        // The volume attachment was deleted via the HP OneView REST API or in 3Par without editing the profile.
	"VolumeCreateFailed", // An error occurred attempting to create the volume in HP OneView and 3Par.
	"VolumeCreating",     // The volume is being created in HP OneView and 3Par.
}

// String - helper
func (o VolumeAttachmentStateV2) String() string { return powerstates[o-1] }

// Equal - helper
func (o VolumeAttachmentStateV2) Equal(s string) bool {
	return (strings.ToUpper(s) == strings.ToUpper(o.String()))
}

// StoragePathV2 - A list of host-to-target path associations.
type StoragePathV2 struct {
	ConnectionID      int      `json:"connectionId,omitempty"`      // connectionId The ID of the connection associated with this storage path. Use GET /rest/server-profiles/available-networks to retrieve the list of available networks. integer required
	IsEnabled         bool     `json:"isEnabled,omitempty"`         // isEnabled Identifies whether the storage path is enabled. Boolean required
	Status            string   `json:"status,omitempty"`            // status The overall health status of the storage path. string read only
	StorageTargetType string   `json:"storageTargetType,omitempty"` // storageTargetType If set to Auto, the storage system will automatically identify the storage targets. In this case, set the storageTargets field to an empty array. If set to TargetPorts, the storage targets can be manually specified in the storageTargets field using comma-separated strings.
	StorageTargets    []string `json:"storageTargets,omitempty"`    // The WWPNs (World Wide Port Names) of the targets on the storage system. If storageTargetType is set to Auto, the storage system will automatically select the target ports, in which case the storageTargets field is not needed and should be set to an empty array. If storageTargetType is set to TargetPorts, then the the storageTargets field should be an array of comma-separated strings representing the WWPNs intended to be used to connect with the storage system. Use GET /rest/storage-systems/{arrayid}/managedPorts?query="expectedNetworkUri EQ '/rest/fc-networks/{netowrk-id}'" to retrieve the storage targets for the associated network.
}
