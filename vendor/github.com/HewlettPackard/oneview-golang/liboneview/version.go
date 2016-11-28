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

// Package liboneview -
package liboneview

import "strings"

// The sum of the API versions give a unique API combined version
const (
	Ver1    = 228 // OV api version (120) + ICSP api version (108)
	Ver2    = 308 // OV api version (200) + ICSP api version (108)
	Unknown = -1
)

// Driver Version
type Version int

// Supported versions
const (
	API_VER1        Version = Ver1
	API_VER2        Version = Ver2
	API_VER_UNKNOWN Version = Unknown
)

// verstringlist - String list description of supported drivers
var verstringlist = [...]string{
	"HP OneView 120,HP ICSP 108",
	"HP OneView 200,HP ICSP 108",
	"Unknown",
}

// verintlist - Integer list description of supported drivers
var verintlist = [...]int{
	Ver1,
	Ver2,
	Unknown,
}

func (o Version) EqualV(v Version) bool { return (int(o) == v.Integer()) }

// String helper for state
func (o Version) Integer() int { return int(o) }

// String helper for state
func (o Version) String() string {
	for i, ver := range verintlist {
		if ver == int(o) {
			return verstringlist[i]
		}
	}
	return verstringlist[len(verstringlist)-1]
}

// Equal helper for state
func (o Version) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(o.String())) }

type verMap map[int]bool

var validVersion verMap

// init the version mapping
func init() {
	validVersion = map[int]bool{
		Ver1: true,
		Ver2: true,
	}
}

// IsVersionValid -  tests if the combination of OV and ICSP REST APIs are compatible for this driver
func IsVersionValid(ver int) bool {
	return validVersion[ver]
}

// CalculateVersion - calculate the current version
func (o Version) CalculateVersion(ovversion int, icspversion int) Version {
	var cver int
	cver = ovversion + icspversion
	for _, ver := range verintlist {
		if ver == cver {
			return Version(cver)
		}
	}
	return Version(Unknown)
}
