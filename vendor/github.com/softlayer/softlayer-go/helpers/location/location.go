/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package location

import (
	"fmt"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

// GetLocationByName returns a Location that matches the provided name, or an
// error if no matching Location can be found.
//
// If you need to access a datacenter's unique properties, use
// GetDatacenterByName instead
func GetLocationByName(sess *session.Session, name string, args ...interface{}) (datatypes.Location, error) {
	var mask string
	if len(args) > 0 {
		mask = args[0].(string)
	}

	locs, err := services.GetLocationService(sess).
		Mask(mask).
		Filter(filter.New(filter.Path("name").Eq(name)).Build()).
		GetDatacenters()

	if err != nil {
		return datatypes.Location{}, err
	}

	// An empty filtered result set does not raise an error
	if len(locs) == 0 {
		return datatypes.Location{}, fmt.Errorf("No locations found with name of %s", name)
	}

	return locs[0], nil
}

// GetDatacenterByName returns a Location_Datacenter that matches the provided
// name, or an error if no matching datacenter can be found.
//
// Note that unless you need to access datacenter-specific properties
// (backendHardwareRouters, etc.), it is more efficient to use
// GetLocationByName, since GetDatacenterByName requires an extra call to the
// API
func GetDatacenterByName(sess *session.Session, name string, args ...interface{}) (datatypes.Location_Datacenter, error) {
	var mask string
	if len(args) > 0 {
		mask = args[0].(string)
	}

	// SoftLayer does not provide a direct path to retrieve a list of "Location_Datacenter"
	// objects. Location_Datacenter.getDatacenters() actually returns a list of "Location"
	// objects, which do not have datacenter-specific properties populated.  So we do this
	// in two passes

	// First get the Location which matches the name
	location, err := GetLocationByName(sess, name, "mask[id]")

	if err != nil {
		return datatypes.Location_Datacenter{}, nil
	}

	// Now get the Datacenter record itself.
	return services.GetLocationDatacenterService(sess).
		Id(*location.Id).
		Mask(mask).
		GetObject()
}
