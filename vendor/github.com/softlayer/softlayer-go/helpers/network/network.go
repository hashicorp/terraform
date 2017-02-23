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

package network

import (
	"fmt"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

// GetNadcLbVipByName Get a virtual ip address by name attached to a load balancer
// appliance like the Netscaler VPX. In the case of some load balancer appliances
// looking up the virtual ip address by name is necessary since they don't get
// assigned an id.
func GetNadcLbVipByName(sess *session.Session, nadcId int, vipName string, mask ...string) (*datatypes.Network_LoadBalancer_VirtualIpAddress, error) {
	service := services.GetNetworkApplicationDeliveryControllerService(sess)

	service = service.
		Id(nadcId)

	if len(mask) > 0 {
		service = service.Mask(mask[0])
	}

	vips, err := service.GetLoadBalancers()

	if err != nil {
		return nil, fmt.Errorf("Error getting NADC load balancers: %s", err)
	}

	for _, vip := range vips {
		if *vip.Name == vipName {
			return &vip, nil
		}
	}

	return nil, fmt.Errorf("Could not find any VIPs for NADC %d matching name %s", nadcId, vipName)
}

// GetNadcLbVipServiceByName Get a load balancer service by name attached to a load balancer
// appliance like the Netscaler VPX. In the case of some load balancer appliances
// looking up the virtual ip address by name is necessary since they don't get
// assigned an id.
func GetNadcLbVipServiceByName(
	sess *session.Session, nadcId int, vipName string, serviceName string, mask ...string,
) (*datatypes.Network_LoadBalancer_Service, error) {
	vipMask := "id,name,services[name,destinationIpAddress,destinationPort,weight,healthCheck,connectionLimit]"

	if len(mask) != 0 {
		vipMask = mask[0]
	}

	vip, err := GetNadcLbVipByName(sess, nadcId, vipName, vipMask)
	if err != nil {
		return nil, err
	}

	for _, service := range vip.Services {
		if *service.Name == serviceName {
			return &service, nil
		}
	}

	return nil, fmt.Errorf(
		"Could not find service %s in VIP %s for load balancer %d",
		serviceName, vipName, nadcId)
}

// GetOsTypeByName retrieves an object of type SoftLayer_Network_Storage_Iscsi_OS_Type.
// To order block storage, OS type is required as a mandatory input.
// GetOsTypeByName helps in getting the OS id and keyName
// Examples:
// id:6   name: Hyper-V  keyName: HYPER_V
// id:12  name: Linux    keyName: LINUX
// id:22  name: VMWare   keyName: VMWARE
// id:30  name: Xen      keyName: XEN
func GetOsTypeByName(sess *session.Session, name string, args ...interface{}) (datatypes.Network_Storage_Iscsi_OS_Type, error) {
	var mask string
	if len(args) > 0 {
		mask = args[0].(string)
	}

	osTypes, err := services.GetNetworkStorageIscsiOSTypeService(sess).
		Mask(mask).
		Filter(filter.New(filter.Path("name").Eq(name)).Build()).
		GetAllObjects()

	if err != nil {
		return datatypes.Network_Storage_Iscsi_OS_Type{}, err
	}

	// An empty filtered result set does not raise an error
	if len(osTypes) == 0 {
		return datatypes.Network_Storage_Iscsi_OS_Type{}, fmt.Errorf("No OS type found with name of %s", name)
	}

	return osTypes[0], nil
}
