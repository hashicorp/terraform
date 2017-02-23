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

package hardware

import (
	"fmt"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/helpers/location"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"regexp"
)

// GeRouterByName returns a Hardware that matches the provided hostname,
// or an error if no matching Hardware can be found.
// SoftLayer does not provide a direct path to retrieve a list of router
// objects. So, get Location_Datacenter object at first and get an array of
// router objects from the Datacenter
func GetRouterByName(sess *session.Session, hostname string, args ...interface{}) (datatypes.Hardware, error) {
	var mask string
	if len(args) > 0 {
		mask = args[0].(string)
	}

	r, _ := regexp.Compile("[A-Za-z]+[0-9]+$")
	dcName := r.FindString(hostname)
	if len(dcName) == 0 {
		return datatypes.Hardware{}, fmt.Errorf("Cannot get datacenter name from hostname %s", hostname)
	}

	datacenter, err := location.GetDatacenterByName(sess, dcName, "hardwareRouters[id,hostname]")
	if err != nil {
		return datatypes.Hardware{}, err
	}

	for _, router := range datacenter.HardwareRouters {
		if *router.Hostname == hostname {
			return services.GetHardwareService(sess).
				Id(*router.Id).
				Mask(mask).
				GetObject()
		}
	}

	return datatypes.Hardware{}, fmt.Errorf("No routers found with hostname of %s", hostname)
}
