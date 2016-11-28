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

// Connectionv200 server profile object for ov
type Connectionv200 struct {
	AllocatedVFs int    `json:"allocatedVFs,omitempty"` // allocatedVFs The number of virtual functions allocated to this connection. This value will be null. integer read only
	RequestedVFs string `json:"requestedVFs,omitempty"` // requestedVFs This value can be "Auto" or 0. string
}
