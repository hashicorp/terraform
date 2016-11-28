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

// create an API support functions for each method that has new changes.
// Check if support should be used in test cases or functions to determine
// if a certain behavior is required for a given functional context.
//
// For exmple:
//
// If we are making a call to profile_templates.go we should only use this on V2
// 1. ask if profile_templates needs supprt checks (use APISupport.Get)
// 2. if profile_templates needs support check, find out if the current lib Version
//    will support profile_templates or not : APISupport.IsSupported
// als see profile_templates.go -> ProfileTemplatesNotSupported

// APISupport
type APISupport int

// Methods that require support
const (
	C_PROFILE_TEMPLATES APISupport = 1 + iota
	C_SERVER_HARDWAREV2
	C_NONE
)

// apisupportlist - real names of things
var apisupportlist = [...]string{
	"profile_templates.go", // different way to get server templates
	"server_hardwarev2.go", // different way to get ilo ip
	"No Support Check Required",
}

// NewByName - returns a new APISupport by name
func (o APISupport) NewByName(name string) APISupport {
	return o.New(o.Get(name))
}

// New - returns a new APISupport object
func (o APISupport) New(i int) APISupport {
	var asc APISupport
	asc = APISupport(i)
	return asc
}

// IsSupported - given the current Version is there api support?
func (o APISupport) IsSupported(v Version) bool {
	switch o {
	case C_SERVER_HARDWAREV2:
		return (API_VER2 == v) || (API_VER_UNKNOWN == v) // adding unkonw to assume this is the latest
	case C_PROFILE_TEMPLATES:
		return (API_VER2 == v) || (API_VER_UNKNOWN == v) // lets assume this is the latest
	default:
		return true
	}
}

// Integer get the int value for APISupport
func (o APISupport) Integer() int { return int(o) }

// String helper for state
func (o APISupport) String() string { return apisupportlist[o] }

// Equal helper for state
func (o APISupport) Equal(s string) bool {
	return (strings.ToUpper(s) == strings.ToUpper(o.String()))
}

// HasCheck - used to determine if we have to make an api verification check
func (o APISupport) HasCheck(s string) bool {
	for _, sc := range apisupportlist {
		if sc == s {
			return true
		}
	}
	return false
}

// Get - get an APISupport from string, returns C_NONE if not found
func (o APISupport) Get(s string) int {
	for i, sc := range apisupportlist {
		if sc == s {
			return i + 1
		}
	}
	return len(apisupportlist)
}
