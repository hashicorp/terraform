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

import "github.com/HewlettPackard/oneview-golang/utils"

// firmware additional properties introduced in 200
// "FirmwareOnly" - Updates the firmware without powering down the server hardware using using HP Smart Update Tools.
// "FirmwareAndOSDrivers" - Updates the firmware and OS drivers without powering down the server hardware using HP Smart Update Tools.
// "FirmwareOnlyOfflineMode" - Manages the firmware through HP OneView. Selecting this option requires the server hardware to be powered down.
type FirmwareOptionv200 struct {
	FirmwareInstallType string `json:"firmwareInstallType,omitempty"` // Specifies the way a Service Pack for ProLiant (SPP) is installed. This field is used if the 'manageFirmware' field is true. Possible values are
}

// ServerProfilev200 - v200 changes to ServerProfile
type ServerProfilev200 struct {
	TemplateCompliance       string        `json:"templateCompliance,omitempty"`       // v2 Compliant, NonCompliant, Unknown
	ServerProfileTemplateURI utils.Nstring `json:"serverProfileTemplateUri,omitempty"` // undocmented option
}
