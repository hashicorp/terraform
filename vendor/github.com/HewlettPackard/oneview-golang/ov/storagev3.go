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

type LocalStorageSettingsV4 struct {
	SasLogicalJBODs []LogicalJbod `json:"sasLogicalJBODs,omitempty"` // "sasLogicalJBODs": [],
}

type LocalStorageEmbeddedControllerV4 struct {
	DeviceSlot string `json:"deviceSlot,omitempty"` // "deviceSlot": "Mezz 1",
}

type LogicalDriveV4 struct {
	Name             string `json:"name,omitempty"`             // "name": "logical drive name",
	SasLogicalJBODId int    `json:"sasLogicalJBODId,omitempty"` // "sasLogicalJBODId": 1,
}

type LogicalJbod struct {
	DeviceSlot        string        `json:"deviceSlot,omitempty"`        // "deviceSlot": "Mezz 1",
	DriveMaxSizeGB    int           `json:"driveMaxSizeGB,omitempty"`    // "driveMaxSizeGB": 100,
	DriveMinSizeGB    int           `json:"driveMinSizeGB,omitempty"`    // "driveMinSizeGB": 10,
	DriveTechnology   string        `json:"driveTechnology,omitempty"`   // "driveTechnology": "SasHdd",
	ID                int           `json:"id,omitempty"`                // "id": 1,
	Name              string        `json:"name,omitempty"`              // "name": "logical jbod 1",
	NumPhysicalDrives int           `json:"numPhysicalDrives,omitempty"` // "numPhyricalDrives": 1,
	SasLogicalJBODUri utils.Nstring `json:"sasLogicalJBODUri,omitempty"` // "sasLogicalJBODUri": nil
	Status            string        `json:"status,omitempty"`            // "status": "OK",
}

type VolumeAttachmentV3 struct {
	IsBootVolume bool `json:"isBootVolume,omitempty"` // "isBootVolume": true,
}
