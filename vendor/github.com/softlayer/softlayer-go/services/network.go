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

/**
 * AUTOMATICALLY GENERATED CODE - DO NOT MODIFY
 */

package services

import (
	"fmt"
	"strings"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
)

// no documentation yet
type Network struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkService returns an instance of the Network SoftLayer service
func GetNetworkService(sess *session.Session) Network {
	return Network{Session: sess}
}

func (r Network) Id(id int) Network {
	r.Options.Id = &id
	return r
}

func (r Network) Mask(mask string) Network {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network) Filter(filter string) Network {
	r.Options.Filter = filter
	return r
}

func (r Network) Limit(limit int) Network {
	r.Options.Limit = &limit
	return r
}

func (r Network) Offset(offset int) Network {
	r.Options.Offset = &offset
	return r
}

// Provide a template containing the following properties to create a Network:
// * networkIdentifier
// * cidr
// * name
//
//
// The ``networkIdentifier`` must be an IP address within RFC 1918 blocks:
// * 192.168.0.0/16
// * 172.16.0.0/12
// * 10.0.0.0/8
// The ``cidr`` must be an integer between 16 and 24, inclusive. The ``networkIdentifier``/``cidr`` must represent a valid subnet specification. The ``name`` must not be empty, but otherwise can contain up to 50 characters of user specified information to identify the Network.
//
// The subnet specification of the Network bounds the IP address space which can be utilized and constrains the creation of Subnets within the Network.
//
// Example networkIdentifier/CIDR combinations:
// * 192.168.0.0/16
// * 192.168.0.0/17
// * 172.16.0.0/16
// * 172.31.0.0/16
// * 10.0.0.0/16
// * 10.255.0.0/16
func (r Network) CreateObject(templateObject *datatypes.Network) (resp datatypes.Network, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network", "createObject", params, &r.Options, &resp)
	return
}

// Creation of a Subnet is necessary prior to provisioning compute resources into a Network. In order to create a Subnet, both a [[SoftLayer_Network_Subnet|Subnet]] and [[SoftLayer_Network_Pod|Pod]] must be specified. The Pod determines where the Subnet will be available for use by compute resources.
//
// Provide a Subnet template containing the following properties:
// * networkIdentifier
// * cidr
// The ``networkIdentifier`` must represent an IP address within that specified by the Network. The ``cidr`` must be an integer between 24 and 29, inclusive, and represent a subnet size smaller than the Network's. The ``networkIdentifier``/``cidr`` must represent a valid subnet specification.
//
// Provide a Pod template containing the following property:
// * name
// The ``name`` must represent a valid Pod e.g. sjc01.pod02. See [[SoftLayer_Network_Pod (type)]] for more information.
//
// The following constraints apply to Subnet creation:
// * It must fit within the bounds of the Network.
// * It must be no larger than /24 and no smaller than /29.
// * Its size must not equal that of the Network. This implies that a fully
// utilized Network will have a minimum of two Subnets.
// * The Pod must support the ability to create Networks by having the
// SUPPORTS_CUSTOMER_DEFINED_NETWORK capability. See [[SoftLayer_Network_Pod/getCapabilities]].
func (r Network) CreateSubnet(subnet *datatypes.Network_Subnet, pod *datatypes.Network_Pod) (resp datatypes.Network_Subnet, err error) {
	params := []interface{}{
		subnet,
		pod,
	}
	err = r.Session.DoRequest("SoftLayer_Network", "createSubnet", params, &r.Options, &resp)
	return
}

// Remove the specified Network. This operation may only be completed if the Network has no Subnets. Attempting to remove a Network with subnets will result in an error.
func (r Network) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network", "deleteObject", nil, &r.Options, &resp)
	return
}

// ```Currently not supported. If attempted and if the subnet would be removed, an error will be presented.```
//
// Remove a Subnet from the Network. Specification of the Subnet to be removed may be done by specifying the ``ID`` property, or by specifying both the ``networkIdentifier`` and ``cidr`` properties on the Subnet template parameter. If the ``ID`` is provided, the ``networkIdentifier``/``cidr`` will be ignored.
//
// Subnets may only be removed when no compute resources are utilizing them.
func (r Network) DeleteSubnet(subnet *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnet,
	}
	err = r.Session.DoRequest("SoftLayer_Network", "deleteSubnet", params, &r.Options, &resp)
	return
}

// Modify either the ``name`` or ``notes`` properties of a Network.
func (r Network) EditObject(templateObject *datatypes.Network) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network", "editObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network) GetAllObjects() (resp []datatypes.Network, err error) {
	err = r.Session.DoRequest("SoftLayer_Network", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve The size of the Network specified in CIDR notation. Specified in conjunction with the ``networkIdentifier`` to describe the bounding subnet size for the Network. Required for creation. See [[SoftLayer_Network/createObject]] documentation for creation details.
func (r Network) GetCidr() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network", "getCidr", nil, &r.Options, &resp)
	return
}

// Retrieve A name for the Network. This is required during creation of a Network and is entirely user defined.
func (r Network) GetName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network", "getName", nil, &r.Options, &resp)
	return
}

// Retrieve The starting IP address of the Network. Specified in conjunction with the ``cidr`` property to specify the bounding IP address space for the Network. Required for creation. See [[SoftLayer_Network/createObject]] documentation for creation details.
func (r Network) GetNetworkIdentifier() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network", "getNetworkIdentifier", nil, &r.Options, &resp)
	return
}

// Retrieve Notes, or a description of the Network. This is entirely user defined.
func (r Network) GetNotes() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network", "getNotes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network) GetObject() (resp datatypes.Network, err error) {
	err = r.Session.DoRequest("SoftLayer_Network", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The Subnets within the Network. These represent the realized segments of the Network and reside within a [[SoftLayer_Network_Pod|Pod]]. A Subnet must be specified when provisioning a compute resource within a Network.
func (r Network) GetSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network", "getSubnets", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Application_Delivery_Controller data type models a single instance of an application delivery controller. Local properties are read only, except for a ''notes'' property, which can be used to describe your application delivery controller service. The type's relational properties provide more information to the service's function and login information to the controller's backend management if advanced view is enabled.
type Network_Application_Delivery_Controller struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerService returns an instance of the Network_Application_Delivery_Controller SoftLayer service
func GetNetworkApplicationDeliveryControllerService(sess *session.Session) Network_Application_Delivery_Controller {
	return Network_Application_Delivery_Controller{Session: sess}
}

func (r Network_Application_Delivery_Controller) Id(id int) Network_Application_Delivery_Controller {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller) Mask(mask string) Network_Application_Delivery_Controller {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller) Filter(filter string) Network_Application_Delivery_Controller {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller) Limit(limit int) Network_Application_Delivery_Controller {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller) Offset(offset int) Network_Application_Delivery_Controller {
	r.Options.Offset = &offset
	return r
}

// Create or add to an application delivery controller based load balancer service. The loadBalancer parameter must have its ''name'', ''type'', ''sourcePort'', and ''virtualIpAddress'' properties populated. Changes are reflected immediately in the application delivery controller.
func (r Network_Application_Delivery_Controller) CreateLiveLoadBalancer(loadBalancer *datatypes.Network_LoadBalancer_VirtualIpAddress) (resp bool, err error) {
	params := []interface{}{
		loadBalancer,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "createLiveLoadBalancer", params, &r.Options, &resp)
	return
}

// Remove a virtual IP address from an application delivery controller based load balancer. Only the ''name'' property in the loadBalancer parameter must be populated. Changes are reflected immediately in the application delivery controller.
func (r Network_Application_Delivery_Controller) DeleteLiveLoadBalancer(loadBalancer *datatypes.Network_LoadBalancer_VirtualIpAddress) (resp bool, err error) {
	params := []interface{}{
		loadBalancer,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "deleteLiveLoadBalancer", params, &r.Options, &resp)
	return
}

// Remove an entire load balancer service, including all virtual IP addresses, from and application delivery controller based load balancer. The ''name'' property the and ''name'' property within the ''vip'' property of the service parameter must be provided. Changes are reflected immediately in the application delivery controller.
func (r Network_Application_Delivery_Controller) DeleteLiveLoadBalancerService(service *datatypes.Network_LoadBalancer_Service) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		service,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "deleteLiveLoadBalancerService", params, &r.Options, &resp)
	return
}

// Edit an applications delivery controller record. Currently only a controller's notes property is editable.
func (r Network_Application_Delivery_Controller) EditObject(templateObject *datatypes.Network_Application_Delivery_Controller) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer customer account that owns an application delivery controller record.
func (r Network_Application_Delivery_Controller) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The average daily public bandwidth usage for the current billing cycle.
func (r Network_Application_Delivery_Controller) GetAverageDailyPublicBandwidthUsage() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getAverageDailyPublicBandwidthUsage", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller) GetBandwidthDataByDate(startDateTime *datatypes.Time, endDateTime *datatypes.Time, networkType *string) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
		networkType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getBandwidthDataByDate", params, &r.Options, &resp)
	return
}

// Use this method when needing a bandwidth image for a single application delivery controller. It will gather the correct input parameters for the generic graphing utility based on the date ranges
func (r Network_Application_Delivery_Controller) GetBandwidthImageByDate(startDateTime *datatypes.Time, endDateTime *datatypes.Time, networkType *string) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
		networkType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getBandwidthImageByDate", params, &r.Options, &resp)
	return
}

// Retrieve The billing item for a Application Delivery Controller.
func (r Network_Application_Delivery_Controller) GetBillingItem() (resp datatypes.Billing_Item_Network_Application_Delivery_Controller, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve Previous configurations for an Application Delivery Controller.
func (r Network_Application_Delivery_Controller) GetConfigurationHistory() (resp []datatypes.Network_Application_Delivery_Controller_Configuration_History, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getConfigurationHistory", nil, &r.Options, &resp)
	return
}

// Retrieve bandwidth graph by date.
func (r Network_Application_Delivery_Controller) GetCustomBandwidthDataByDate(graphData *datatypes.Container_Graph) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		graphData,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getCustomBandwidthDataByDate", params, &r.Options, &resp)
	return
}

// Retrieve The datacenter that the application delivery controller resides in.
func (r Network_Application_Delivery_Controller) GetDatacenter() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getDatacenter", nil, &r.Options, &resp)
	return
}

// Retrieve A brief description of an application delivery controller record.
func (r Network_Application_Delivery_Controller) GetDescription() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getDescription", nil, &r.Options, &resp)
	return
}

// Retrieve The date in which the license for this application delivery controller will expire.
func (r Network_Application_Delivery_Controller) GetLicenseExpirationDate() (resp datatypes.Time, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getLicenseExpirationDate", nil, &r.Options, &resp)
	return
}

// Get the graph image for an application delivery controller service based on the supplied graph type and metric.  The available graph types are: 'connections' and 'status', and the available metrics are: 'day', 'week' and 'month'.
//
// This method returns the raw binary image data.
func (r Network_Application_Delivery_Controller) GetLiveLoadBalancerServiceGraphImage(service *datatypes.Network_LoadBalancer_Service, graphType *string, metric *string) (resp []byte, err error) {
	params := []interface{}{
		service,
		graphType,
		metric,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getLiveLoadBalancerServiceGraphImage", params, &r.Options, &resp)
	return
}

// Retrieve The virtual IP address records that belong to an application delivery controller based load balancer.
func (r Network_Application_Delivery_Controller) GetLoadBalancers() (resp []datatypes.Network_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getLoadBalancers", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating that this Application Delivery Controller is a managed resource.
func (r Network_Application_Delivery_Controller) GetManagedResourceFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getManagedResourceFlag", nil, &r.Options, &resp)
	return
}

// Retrieve An application delivery controller's management ip address.
func (r Network_Application_Delivery_Controller) GetManagementIpAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getManagementIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The network VLAN that an application delivery controller resides on.
func (r Network_Application_Delivery_Controller) GetNetworkVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getNetworkVlan", nil, &r.Options, &resp)
	return
}

// Retrieve The network VLANs that an application delivery controller resides on.
func (r Network_Application_Delivery_Controller) GetNetworkVlans() (resp []datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getNetworkVlans", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Application_Delivery_Controller object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Application_Delivery_Controller service. You can only retrieve application delivery controllers that are associated with your SoftLayer customer account.
func (r Network_Application_Delivery_Controller) GetObject() (resp datatypes.Network_Application_Delivery_Controller, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The total public outbound bandwidth for the current billing cycle.
func (r Network_Application_Delivery_Controller) GetOutboundPublicBandwidthUsage() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getOutboundPublicBandwidthUsage", nil, &r.Options, &resp)
	return
}

// Retrieve The password used to connect to an application delivery controller's management interface when it is operating in advanced view mode.
func (r Network_Application_Delivery_Controller) GetPassword() (resp datatypes.Software_Component_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getPassword", nil, &r.Options, &resp)
	return
}

// Retrieve An application delivery controller's primary public IP address.
func (r Network_Application_Delivery_Controller) GetPrimaryIpAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getPrimaryIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The projected public outbound bandwidth for the current billing cycle.
func (r Network_Application_Delivery_Controller) GetProjectedPublicBandwidthUsage() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getProjectedPublicBandwidthUsage", nil, &r.Options, &resp)
	return
}

// Retrieve A network application controller's subnets. A subnet is a group of IP addresses
func (r Network_Application_Delivery_Controller) GetSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller) GetTagReferences() (resp []datatypes.Tag_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getTagReferences", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller) GetType() (resp datatypes.Network_Application_Delivery_Controller_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller) GetVirtualIpAddresses() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "getVirtualIpAddresses", nil, &r.Options, &resp)
	return
}

// Restore an application delivery controller's base configuration state. The configuration will be set to what it was when initially provisioned.
func (r Network_Application_Delivery_Controller) RestoreBaseConfiguration() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "restoreBaseConfiguration", nil, &r.Options, &resp)
	return
}

// Restore an application delivery controller's configuration state.
func (r Network_Application_Delivery_Controller) RestoreConfiguration(configurationHistoryId *int) (resp bool, err error) {
	params := []interface{}{
		configurationHistoryId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "restoreConfiguration", params, &r.Options, &resp)
	return
}

// Save an application delivery controller's configuration state. The notes property for this method is optional.
func (r Network_Application_Delivery_Controller) SaveCurrentConfiguration(notes *string) (resp datatypes.Network_Application_Delivery_Controller_Configuration_History, err error) {
	params := []interface{}{
		notes,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "saveCurrentConfiguration", params, &r.Options, &resp)
	return
}

// Update the the virtual IP address interface within an application delivery controller based load balancer identified by the ''name'' property in the loadBalancer parameter. You only need to set the properties in the loadBalancer parameter that you wish to change. Any virtual IP properties omitted or left empty are ignored. Changes are reflected immediately in the application delivery controller.
func (r Network_Application_Delivery_Controller) UpdateLiveLoadBalancer(loadBalancer *datatypes.Network_LoadBalancer_VirtualIpAddress) (resp bool, err error) {
	params := []interface{}{
		loadBalancer,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "updateLiveLoadBalancer", params, &r.Options, &resp)
	return
}

// Update the NetScaler VPX License.
//
// This service will create a transaction to update a NetScaler VPX License.  After the license is updated the load balancer will reboot in order to apply the newly issued license
//
// The load balancer will be unavailable during the reboot.
func (r Network_Application_Delivery_Controller) UpdateNetScalerLicense() (resp datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller", "updateNetScalerLicense", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Application_Delivery_Controller_Configuration_History data type models a single instance of a configuration history entry for an application delivery controller. The configuration history entries are used to support creating backups of an application delivery controller's configuration state in order to restore them later if needed.
type Network_Application_Delivery_Controller_Configuration_History struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerConfigurationHistoryService returns an instance of the Network_Application_Delivery_Controller_Configuration_History SoftLayer service
func GetNetworkApplicationDeliveryControllerConfigurationHistoryService(sess *session.Session) Network_Application_Delivery_Controller_Configuration_History {
	return Network_Application_Delivery_Controller_Configuration_History{Session: sess}
}

func (r Network_Application_Delivery_Controller_Configuration_History) Id(id int) Network_Application_Delivery_Controller_Configuration_History {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_Configuration_History) Mask(mask string) Network_Application_Delivery_Controller_Configuration_History {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_Configuration_History) Filter(filter string) Network_Application_Delivery_Controller_Configuration_History {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_Configuration_History) Limit(limit int) Network_Application_Delivery_Controller_Configuration_History {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_Configuration_History) Offset(offset int) Network_Application_Delivery_Controller_Configuration_History {
	r.Options.Offset = &offset
	return r
}

// deleteObject permanently removes a configuration history record
func (r Network_Application_Delivery_Controller_Configuration_History) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_Configuration_History", "deleteObject", nil, &r.Options, &resp)
	return
}

// Retrieve The application delivery controller that a configuration history record belongs to.
func (r Network_Application_Delivery_Controller_Configuration_History) GetController() (resp datatypes.Network_Application_Delivery_Controller, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_Configuration_History", "getController", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_Configuration_History) GetObject() (resp datatypes.Network_Application_Delivery_Controller_Configuration_History, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_Configuration_History", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerHealthAttributeService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerHealthAttributeService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute {
	return Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute) GetHealthCheck() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Check, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute", "getHealthCheck", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute) GetType() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute", "getType", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerHealthAttributeTypeService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerHealthAttributeTypeService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type {
	return Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type) GetAllObjects() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Health_Check struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerHealthCheckService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_Health_Check SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerHealthCheckService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_Health_Check {
	return Network_Application_Delivery_Controller_LoadBalancer_Health_Check{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_Health_Check {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_Health_Check {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_Health_Check {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_Health_Check {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_Health_Check {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) GetAttributes() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Check", "getAttributes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Check, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Check", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve Collection of scale load balancers that use this health check.
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) GetScaleLoadBalancers() (resp []datatypes.Scale_LoadBalancer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Check", "getScaleLoadBalancers", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) GetServices() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Check", "getServices", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check) GetType() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Check", "getType", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerHealthCheckTypeService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerHealthCheckTypeService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type {
	return Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type) GetAllObjects() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Routing_Method struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerRoutingMethodService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_Routing_Method SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerRoutingMethodService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_Routing_Method {
	return Network_Application_Delivery_Controller_LoadBalancer_Routing_Method{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Method) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_Routing_Method {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Method) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_Routing_Method {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Method) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_Routing_Method {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Method) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_Routing_Method {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Method) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_Routing_Method {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Method) GetAllObjects() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Routing_Method, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Routing_Method", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Method) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Routing_Method, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Routing_Method", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Routing_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerRoutingTypeService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_Routing_Type SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerRoutingTypeService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_Routing_Type {
	return Network_Application_Delivery_Controller_LoadBalancer_Routing_Type{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Type) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_Routing_Type {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Type) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_Routing_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Type) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_Routing_Type {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Type) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_Routing_Type {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Type) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_Routing_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Type) GetAllObjects() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Routing_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Routing_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Routing_Type) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Routing_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Routing_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Service struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerServiceService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_Service SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerServiceService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_Service {
	return Network_Application_Delivery_Controller_LoadBalancer_Service{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_Service {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_Service {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_Service {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_Service {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_Service {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) DeleteObject() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "deleteObject", nil, &r.Options, &resp)
	return
}

// Get the graph image for a load balancer service based on the supplied graph type and metric.  The available graph types are: 'connections' and 'status', and the available metrics are: 'day', 'week' and 'month'.
//
// This method returns the raw binary image data.
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) GetGraphImage(graphType *string, metric *string) (resp []byte, err error) {
	params := []interface{}{
		graphType,
		metric,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "getGraphImage", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) GetGroupReferences() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service_Group_CrossReference, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "getGroupReferences", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) GetGroups() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "getGroups", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) GetHealthCheck() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Check, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "getHealthCheck", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) GetHealthChecks() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Health_Check, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "getHealthChecks", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) GetIpAddress() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "getIpAddress", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) GetServiceGroup() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "getServiceGroup", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Service) ToggleStatus() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service", "toggleStatus", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Service_Group struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerServiceGroupService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_Service_Group SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerServiceGroupService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_Service_Group {
	return Network_Application_Delivery_Controller_LoadBalancer_Service_Group{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_Service_Group {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_Service_Group {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_Service_Group {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_Service_Group {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_Service_Group {
	r.Options.Offset = &offset
	return r
}

// Get the graph image for a load balancer service group based on the supplied graph type and metric.  The only available graph type currently is: 'connections', and the available metrics are: 'day', 'week' and 'month'.
//
// This method returns the raw binary image data.
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) GetGraphImage(graphType *string, metric *string) (resp []byte, err error) {
	params := []interface{}{
		graphType,
		metric,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "getGraphImage", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) GetRoutingMethod() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Routing_Method, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "getRoutingMethod", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) GetRoutingType() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Routing_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "getRoutingType", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) GetServiceReferences() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service_Group_CrossReference, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "getServiceReferences", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) GetServices() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "getServices", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) GetVirtualServer() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualServer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "getVirtualServer", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) GetVirtualServers() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualServer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "getVirtualServers", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_Service_Group) KickAllConnections() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_Service_Group", "kickAllConnections", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerVirtualIpAddressService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerVirtualIpAddressService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress {
	return Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress {
	r.Options.Offset = &offset
	return r
}

// Like any other API object, the load balancers can have their exposed properties edited by passing in a modified version of the object.  The load balancer object also can modify its services in this way.  Simply request the load balancer object you wish to edit, then modify the objects in the services array and pass the modified object to this function.  WARNING:  Services cannot be deleted in this manner, you must call deleteObject() on the service to physically remove them from the load balancer.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) EditObject(templateObject *datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve A virtual IP address's associated application delivery controller.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetApplicationDeliveryController() (resp datatypes.Network_Application_Delivery_Controller, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getApplicationDeliveryController", nil, &r.Options, &resp)
	return
}

// Retrieve A virtual IP address's associated application delivery controllers.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetApplicationDeliveryControllers() (resp []datatypes.Network_Application_Delivery_Controller, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getApplicationDeliveryControllers", nil, &r.Options, &resp)
	return
}

// Yields a list of the SSL/TLS encryption ciphers that are currently supported on this virtual IP address instance.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetAvailableSecureTransportCiphers() (resp []datatypes.Security_SecureTransportCipher, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getAvailableSecureTransportCiphers", nil, &r.Options, &resp)
	return
}

// Yields a list of the secure communication protocols that are currently supported on this virtual IP address instance. The list of supported ciphers for each protocol is culled to match availability.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetAvailableSecureTransportProtocols() (resp []datatypes.Security_SecureTransportProtocol, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getAvailableSecureTransportProtocols", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for the load balancer virtual IP. This is only valid when dedicatedFlag is false. This is an independent virtual IP, and if canceled, will only affect the associated virtual IP.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for the load balancing device housing the virtual IP. This billing item represents a device which could contain other virtual IPs. Caution should be taken when canceling. This is only valid when dedicatedFlag is true.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetDedicatedBillingItem() (resp datatypes.Billing_Item_Network_LoadBalancer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getDedicatedBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve Denotes whether the virtual IP is configured within a high availability cluster.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetHighAvailabilityFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getHighAvailabilityFlag", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetIpAddress() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetLoadBalancerHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getLoadBalancerHardware", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating that the load balancer is a managed resource.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetManagedResourceFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getManagedResourceFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The list of security ciphers enabled for this virtual IP address
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetSecureTransportCiphers() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress_SecureTransportCipher, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getSecureTransportCiphers", nil, &r.Options, &resp)
	return
}

// Retrieve The list of secure transport protocols enabled for this virtual IP address
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetSecureTransportProtocols() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress_SecureTransportProtocol, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getSecureTransportProtocols", nil, &r.Options, &resp)
	return
}

// Retrieve The SSL certificate currently associated with the VIP.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetSecurityCertificate() (resp datatypes.Security_Certificate, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getSecurityCertificate", nil, &r.Options, &resp)
	return
}

// Retrieve The SSL certificate currently associated with the VIP. Provides chosen certificate visibility to unprivileged users.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetSecurityCertificateEntry() (resp datatypes.Security_Certificate_Entry, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getSecurityCertificateEntry", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) GetVirtualServers() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualServer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "getVirtualServers", nil, &r.Options, &resp)
	return
}

// Start SSL acceleration on all SSL virtual services (those with a type of HTTPS). This action should be taken only after configuring an SSL certificate for the virtual IP.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) StartSsl() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "startSsl", nil, &r.Options, &resp)
	return
}

// Stop SSL acceleration on all SSL virtual services (those with a type of HTTPS).
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) StopSsl() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "stopSsl", nil, &r.Options, &resp)
	return
}

// Upgrades the connection limit on the Virtual IP to Address to the next, higher connection limit of the same product.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress) UpgradeConnectionLimit() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress", "upgradeConnectionLimit", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_VirtualServer struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkApplicationDeliveryControllerLoadBalancerVirtualServerService returns an instance of the Network_Application_Delivery_Controller_LoadBalancer_VirtualServer SoftLayer service
func GetNetworkApplicationDeliveryControllerLoadBalancerVirtualServerService(sess *session.Session) Network_Application_Delivery_Controller_LoadBalancer_VirtualServer {
	return Network_Application_Delivery_Controller_LoadBalancer_VirtualServer{Session: sess}
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) Id(id int) Network_Application_Delivery_Controller_LoadBalancer_VirtualServer {
	r.Options.Id = &id
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) Mask(mask string) Network_Application_Delivery_Controller_LoadBalancer_VirtualServer {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) Filter(filter string) Network_Application_Delivery_Controller_LoadBalancer_VirtualServer {
	r.Options.Filter = filter
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) Limit(limit int) Network_Application_Delivery_Controller_LoadBalancer_VirtualServer {
	r.Options.Limit = &limit
	return r
}

func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) Offset(offset int) Network_Application_Delivery_Controller_LoadBalancer_VirtualServer {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) DeleteObject() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualServer", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) GetObject() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualServer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualServer", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) GetRoutingMethod() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_Routing_Method, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualServer", "getRoutingMethod", nil, &r.Options, &resp)
	return
}

// Retrieve Collection of scale load balancers this virtual server applies to.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) GetScaleLoadBalancers() (resp []datatypes.Scale_LoadBalancer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualServer", "getScaleLoadBalancers", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) GetServiceGroups() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_Service_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualServer", "getServiceGroups", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) GetVirtualIpAddress() (resp datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualServer", "getVirtualIpAddress", nil, &r.Options, &resp)
	return
}

// Start SSL acceleration on all SSL virtual services (those with a type of HTTPS). This action should be taken only after configuring an SSL certificate for the virtual IP.
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) StartSsl() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualServer", "startSsl", nil, &r.Options, &resp)
	return
}

// Stop SSL acceleration on all SSL virtual services (those with a type of HTTPS).
func (r Network_Application_Delivery_Controller_LoadBalancer_VirtualServer) StopSsl() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualServer", "stopSsl", nil, &r.Options, &resp)
	return
}

// A SoftLayer_Network_Backbone represents a single backbone connection from SoftLayer to the public Internet, from the Internet to the SoftLayer private network, or a link that connects the private networks between SoftLayer's datacenters. The SoftLayer_Network_Backbone data type is a collection of data associated with one of those connections.
type Network_Backbone struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkBackboneService returns an instance of the Network_Backbone SoftLayer service
func GetNetworkBackboneService(sess *session.Session) Network_Backbone {
	return Network_Backbone{Session: sess}
}

func (r Network_Backbone) Id(id int) Network_Backbone {
	r.Options.Id = &id
	return r
}

func (r Network_Backbone) Mask(mask string) Network_Backbone {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Backbone) Filter(filter string) Network_Backbone {
	r.Options.Filter = filter
	return r
}

func (r Network_Backbone) Limit(limit int) Network_Backbone {
	r.Options.Limit = &limit
	return r
}

func (r Network_Backbone) Offset(offset int) Network_Backbone {
	r.Options.Offset = &offset
	return r
}

// Retrieve a list of all SoftLayer backbone connections. Use this method if you need all backbones or don't know the id number of a specific backbone.
func (r Network_Backbone) GetAllBackbones() (resp []datatypes.Network_Backbone, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone", "getAllBackbones", nil, &r.Options, &resp)
	return
}

// Retrieve a list of all SoftLayer backbone connections for a location name.
func (r Network_Backbone) GetBackbonesForLocationName(locationName *string) (resp []datatypes.Network_Backbone, err error) {
	params := []interface{}{
		locationName,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Backbone", "getBackbonesForLocationName", params, &r.Options, &resp)
	return
}

// Retrieve a graph of a SoftLayer backbone's last 24 hours of activity. getGraphImage returns a PNG image measuring 827 pixels by 293 pixels.
func (r Network_Backbone) GetGraphImage() (resp []byte, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone", "getGraphImage", nil, &r.Options, &resp)
	return
}

// Retrieve A backbone's status.
func (r Network_Backbone) GetHealth() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone", "getHealth", nil, &r.Options, &resp)
	return
}

// Retrieve Which of the SoftLayer datacenters a backbone is connected to.
func (r Network_Backbone) GetLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone", "getLocation", nil, &r.Options, &resp)
	return
}

// Retrieve A backbone's primary network component.
func (r Network_Backbone) GetNetworkComponent() (resp datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone", "getNetworkComponent", nil, &r.Options, &resp)
	return
}

// Retrieve an individual SoftLayer_Network_Backbone record. Use the getAllBackbones() method to retrieve a list of all SoftLayer network backbones.
func (r Network_Backbone) GetObject() (resp datatypes.Network_Backbone, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Backbone_Location_Dependent struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkBackboneLocationDependentService returns an instance of the Network_Backbone_Location_Dependent SoftLayer service
func GetNetworkBackboneLocationDependentService(sess *session.Session) Network_Backbone_Location_Dependent {
	return Network_Backbone_Location_Dependent{Session: sess}
}

func (r Network_Backbone_Location_Dependent) Id(id int) Network_Backbone_Location_Dependent {
	r.Options.Id = &id
	return r
}

func (r Network_Backbone_Location_Dependent) Mask(mask string) Network_Backbone_Location_Dependent {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Backbone_Location_Dependent) Filter(filter string) Network_Backbone_Location_Dependent {
	r.Options.Filter = filter
	return r
}

func (r Network_Backbone_Location_Dependent) Limit(limit int) Network_Backbone_Location_Dependent {
	r.Options.Limit = &limit
	return r
}

func (r Network_Backbone_Location_Dependent) Offset(offset int) Network_Backbone_Location_Dependent {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Backbone_Location_Dependent) GetAllObjects() (resp []datatypes.Network_Backbone_Location_Dependent, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone_Location_Dependent", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Backbone_Location_Dependent) GetDependentLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone_Location_Dependent", "getDependentLocation", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Backbone_Location_Dependent) GetObject() (resp datatypes.Network_Backbone_Location_Dependent, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone_Location_Dependent", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Backbone_Location_Dependent) GetSourceDependentsByName(locationName *string) (resp datatypes.Location, err error) {
	params := []interface{}{
		locationName,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Backbone_Location_Dependent", "getSourceDependentsByName", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Backbone_Location_Dependent) GetSourceLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Backbone_Location_Dependent", "getSourceLocation", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Bandwidth_Version1_Allotment class provides methods and data structures necessary to work with an array of hardware objects associated with a single Bandwidth Pooling.
type Network_Bandwidth_Version1_Allotment struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkBandwidthVersion1AllotmentService returns an instance of the Network_Bandwidth_Version1_Allotment SoftLayer service
func GetNetworkBandwidthVersion1AllotmentService(sess *session.Session) Network_Bandwidth_Version1_Allotment {
	return Network_Bandwidth_Version1_Allotment{Session: sess}
}

func (r Network_Bandwidth_Version1_Allotment) Id(id int) Network_Bandwidth_Version1_Allotment {
	r.Options.Id = &id
	return r
}

func (r Network_Bandwidth_Version1_Allotment) Mask(mask string) Network_Bandwidth_Version1_Allotment {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Bandwidth_Version1_Allotment) Filter(filter string) Network_Bandwidth_Version1_Allotment {
	r.Options.Filter = filter
	return r
}

func (r Network_Bandwidth_Version1_Allotment) Limit(limit int) Network_Bandwidth_Version1_Allotment {
	r.Options.Limit = &limit
	return r
}

func (r Network_Bandwidth_Version1_Allotment) Offset(offset int) Network_Bandwidth_Version1_Allotment {
	r.Options.Offset = &offset
	return r
}

// Create a allotment for servers to pool bandwidth and avoid overages in billing if they use more than there allocated bandwidth.
func (r Network_Bandwidth_Version1_Allotment) CreateObject(templateObject *datatypes.Network_Bandwidth_Version1_Allotment) (resp datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "createObject", params, &r.Options, &resp)
	return
}

// Edit a bandwidth allotment's local properties. Currently you may only change an allotment's name. Use the [[SoftLayer_Network_Bandwidth_Version1_Allotment::reassignServers|reassignServers()]] and [[SoftLayer_Network_Bandwidth_Version1_Allotment::unassignServers|unassignServers()]] methods to move servers in and out of your allotments.
func (r Network_Bandwidth_Version1_Allotment) EditObject(templateObject *datatypes.Network_Bandwidth_Version1_Allotment) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account associated with this virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The bandwidth allotment detail records associated with this virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetActiveDetails() (resp []datatypes.Network_Bandwidth_Version1_Allotment_Detail, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getActiveDetails", nil, &r.Options, &resp)
	return
}

// Retrieve The Application Delivery Controller contained within a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetApplicationDeliveryControllers() (resp []datatypes.Network_Application_Delivery_Controller, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getApplicationDeliveryControllers", nil, &r.Options, &resp)
	return
}

// Retrieve The average daily public bandwidth usage for the current billing cycle.
func (r Network_Bandwidth_Version1_Allotment) GetAverageDailyPublicBandwidthUsage() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getAverageDailyPublicBandwidthUsage", nil, &r.Options, &resp)
	return
}

// This method recurses through all servers on a Bandwidth Pool for 24 hour time span starting at a given date/time. To get the private data set for all servers on a Bandwidth Pool from midnight Feb 1st, 2008 to 23:59 on Feb 1st, you would pass a parameter of '02/01/2008 0:00'.  The ending date / time is calculated for you to prevent requesting data from the server for periods larger than 24 hours as this method requires processing a lot of data records and can get slow at times.
func (r Network_Bandwidth_Version1_Allotment) GetBackendBandwidthByHour(date *datatypes.Time) (resp []datatypes.Container_Network_Bandwidth_Version1_Usage, err error) {
	params := []interface{}{
		date,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBackendBandwidthByHour", params, &r.Options, &resp)
	return
}

// This method recurses through all servers on a Bandwidth Pool between the given start and end dates to retrieve public bandwidth data.
func (r Network_Bandwidth_Version1_Allotment) GetBackendBandwidthUse(startDate *datatypes.Time, endDate *datatypes.Time) (resp []datatypes.Network_Bandwidth_Version1_Usage_Detail, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBackendBandwidthUse", params, &r.Options, &resp)
	return
}

// Retrieve a collection of bandwidth data from an individual public or private network tracking object. Data is ideal if you with to employ your own traffic storage and graphing systems.
func (r Network_Bandwidth_Version1_Allotment) GetBandwidthForDateRange(startDate *datatypes.Time, endDate *datatypes.Time) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBandwidthForDateRange", params, &r.Options, &resp)
	return
}

// This method recurses through all servers on a Bandwidth Pool for a given snapshot range, gathers the necessary parameters, and then calls the bandwidth graphing server.  The return result is a container that includes the min and max dates for all servers to be used in the query, as well as an image in PNG format.  This method uses the new and improved drawing routines which should return in a reasonable time frame now that the new backend data warehouse is used.
func (r Network_Bandwidth_Version1_Allotment) GetBandwidthImage(networkType *string, snapshotRange *string, draw *bool, dateSpecified *datatypes.Time, dateSpecifiedEnd *datatypes.Time) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		networkType,
		snapshotRange,
		draw,
		dateSpecified,
		dateSpecifiedEnd,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBandwidthImage", params, &r.Options, &resp)
	return
}

// Retrieve The bare metal server instances contained within a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetBareMetalInstances() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBareMetalInstances", nil, &r.Options, &resp)
	return
}

// Retrieve A virtual rack's raw bandwidth usage data for an account's current billing cycle. One object is returned for each network this server is attached to.
func (r Network_Bandwidth_Version1_Allotment) GetBillingCycleBandwidthUsage() (resp []datatypes.Network_Bandwidth_Usage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBillingCycleBandwidthUsage", nil, &r.Options, &resp)
	return
}

// Retrieve A virtual rack's raw private network bandwidth usage data for an account's current billing cycle.
func (r Network_Bandwidth_Version1_Allotment) GetBillingCyclePrivateBandwidthUsage() (resp datatypes.Network_Bandwidth_Usage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBillingCyclePrivateBandwidthUsage", nil, &r.Options, &resp)
	return
}

// Retrieve A virtual rack's raw public network bandwidth usage data for an account's current billing cycle.
func (r Network_Bandwidth_Version1_Allotment) GetBillingCyclePublicBandwidthUsage() (resp datatypes.Network_Bandwidth_Usage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBillingCyclePublicBandwidthUsage", nil, &r.Options, &resp)
	return
}

// Retrieve The total public bandwidth used in this virtual rack for an account's current billing cycle.
func (r Network_Bandwidth_Version1_Allotment) GetBillingCyclePublicUsageTotal() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBillingCyclePublicUsageTotal", nil, &r.Options, &resp)
	return
}

// Retrieve A virtual rack's billing item.
func (r Network_Bandwidth_Version1_Allotment) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve An object that provides commonly used bandwidth summary components for the current billing cycle.
func (r Network_Bandwidth_Version1_Allotment) GetCurrentBandwidthSummary() (resp datatypes.Metric_Tracking_Object_Bandwidth_Summary, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getCurrentBandwidthSummary", nil, &r.Options, &resp)
	return
}

// Retrieve bandwidth graph by date.
func (r Network_Bandwidth_Version1_Allotment) GetCustomBandwidthDataByDate(graphData *datatypes.Container_Graph) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		graphData,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getCustomBandwidthDataByDate", params, &r.Options, &resp)
	return
}

// Retrieve The bandwidth allotment detail records associated with this virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetDetails() (resp []datatypes.Network_Bandwidth_Version1_Allotment_Detail, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getDetails", nil, &r.Options, &resp)
	return
}

// This method recurses through all servers on a Bandwidth Pool for 24 hour time span starting at a given date/time. To get the public data set for all servers on a Bandwidth Pool from midnight Feb 1st, 2008 to 23:59 on Feb 1st, you would pass a parameter of '02/01/2008 0:00'.  The ending date / time is calculated for you to prevent requesting data from the server for periods larger than 24 hours as this method requires processing a lot of data records and can get slow at times.
func (r Network_Bandwidth_Version1_Allotment) GetFrontendBandwidthByHour(date *datatypes.Time) (resp []datatypes.Container_Network_Bandwidth_Version1_Usage, err error) {
	params := []interface{}{
		date,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getFrontendBandwidthByHour", params, &r.Options, &resp)
	return
}

// This method recurses through all servers on a Bandwidth Pool between the given start and end dates to retrieve private bandwidth data.
func (r Network_Bandwidth_Version1_Allotment) GetFrontendBandwidthUse(startDate *datatypes.Time, endDate *datatypes.Time) (resp []datatypes.Network_Bandwidth_Version1_Usage_Detail, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getFrontendBandwidthUse", params, &r.Options, &resp)
	return
}

// Retrieve The hardware contained within a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The total public inbound bandwidth used in this virtual rack for an account's current billing cycle.
func (r Network_Bandwidth_Version1_Allotment) GetInboundPublicBandwidthUsage() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getInboundPublicBandwidthUsage", nil, &r.Options, &resp)
	return
}

// Retrieve The location group associated with this virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetLocationGroup() (resp datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getLocationGroup", nil, &r.Options, &resp)
	return
}

// Retrieve The managed bare metal server instances contained within a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetManagedBareMetalInstances() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getManagedBareMetalInstances", nil, &r.Options, &resp)
	return
}

// Retrieve The managed hardware contained within a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetManagedHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getManagedHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The managed Virtual Server contained within a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetManagedVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getManagedVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve A virtual rack's metric tracking object. This object records all periodic polled data available to this rack.
func (r Network_Bandwidth_Version1_Allotment) GetMetricTrackingObject() (resp datatypes.Metric_Tracking_Object_VirtualDedicatedRack, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getMetricTrackingObject", nil, &r.Options, &resp)
	return
}

// Retrieve The metric tracking object id for this allotment.
func (r Network_Bandwidth_Version1_Allotment) GetMetricTrackingObjectId() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getMetricTrackingObjectId", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Bandwidth_Version1_Allotment object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Hardware service. You can only retrieve an allotment associated with the account that your portal user is assigned to.
func (r Network_Bandwidth_Version1_Allotment) GetObject() (resp datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The total public outbound bandwidth used in this virtual rack for an account's current billing cycle.
func (r Network_Bandwidth_Version1_Allotment) GetOutboundPublicBandwidthUsage() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getOutboundPublicBandwidthUsage", nil, &r.Options, &resp)
	return
}

// Retrieve Whether the bandwidth usage for this bandwidth pool for the current billing cycle exceeds the allocation.
func (r Network_Bandwidth_Version1_Allotment) GetOverBandwidthAllocationFlag() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getOverBandwidthAllocationFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The private network only hardware contained within a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetPrivateNetworkOnlyHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getPrivateNetworkOnlyHardware", nil, &r.Options, &resp)
	return
}

// Retrieve Whether the bandwidth usage for this bandwidth pool for the current billing cycle is projected to exceed the allocation.
func (r Network_Bandwidth_Version1_Allotment) GetProjectedOverBandwidthAllocationFlag() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getProjectedOverBandwidthAllocationFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The projected public outbound bandwidth for this virtual server for the current billing cycle.
func (r Network_Bandwidth_Version1_Allotment) GetProjectedPublicBandwidthUsage() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getProjectedPublicBandwidthUsage", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Bandwidth_Version1_Allotment) GetServiceProvider() (resp datatypes.Service_Provider, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getServiceProvider", nil, &r.Options, &resp)
	return
}

// Retrieve The combined allocated bandwidth for all servers in a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetTotalBandwidthAllocated() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getTotalBandwidthAllocated", nil, &r.Options, &resp)
	return
}

// Gets the monthly recurring fee of a pooled server.
func (r Network_Bandwidth_Version1_Allotment) GetVdrMemberRecurringFee() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getVdrMemberRecurringFee", nil, &r.Options, &resp)
	return
}

// Retrieve The Virtual Server contained within a virtual rack.
func (r Network_Bandwidth_Version1_Allotment) GetVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "getVirtualGuests", nil, &r.Options, &resp)
	return
}

// This method will reassign a collection of SoftLayer hardware to a bandwidth allotment Bandwidth Pool.
func (r Network_Bandwidth_Version1_Allotment) ReassignServers(templateObjects []datatypes.Hardware, newAllotmentId *int) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
		newAllotmentId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "reassignServers", params, &r.Options, &resp)
	return
}

// This will remove a bandwidth pooling from a customer's allotments by cancelling the billing item.  All servers in that allotment will get moved to the account's vpr.
func (r Network_Bandwidth_Version1_Allotment) RequestVdrCancellation() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "requestVdrCancellation", nil, &r.Options, &resp)
	return
}

// This will move servers into a bandwidth pool, removing them from their previous bandwidth pool and optionally remove the bandwidth pool on completion.
func (r Network_Bandwidth_Version1_Allotment) RequestVdrContentUpdates(hardwareToAdd []datatypes.Hardware, hardwareToRemove []datatypes.Hardware, cloudsToAdd []datatypes.Virtual_Guest, cloudsToRemove []datatypes.Virtual_Guest, optionalAllotmentId *int, adcToAdd []datatypes.Network_Application_Delivery_Controller, adcToRemove []datatypes.Network_Application_Delivery_Controller) (resp bool, err error) {
	params := []interface{}{
		hardwareToAdd,
		hardwareToRemove,
		cloudsToAdd,
		cloudsToRemove,
		optionalAllotmentId,
		adcToAdd,
		adcToRemove,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "requestVdrContentUpdates", params, &r.Options, &resp)
	return
}

// This will update the bandwidth pool to the servers provided.  Servers currently in the bandwidth pool not provided on update will be removed. Servers provided on update not currently in the bandwidth pool will be added. If all servers are removed, this removes the bandwidth pool on completion.
func (r Network_Bandwidth_Version1_Allotment) SetVdrContent(hardware []datatypes.Hardware, bareMetalServers []datatypes.Hardware, virtualServerInstance []datatypes.Virtual_Guest, adc []datatypes.Network_Application_Delivery_Controller, optionalAllotmentId *int) (resp bool, err error) {
	params := []interface{}{
		hardware,
		bareMetalServers,
		virtualServerInstance,
		adc,
		optionalAllotmentId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "setVdrContent", params, &r.Options, &resp)
	return
}

// This method will reassign a collection of SoftLayer hardware to the virtual private rack
func (r Network_Bandwidth_Version1_Allotment) UnassignServers(templateObjects []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "unassignServers", params, &r.Options, &resp)
	return
}

// This method will void a pending server removal from this bandwidth pooling. Pass in the id of the hardware object or virtual guest you wish to update. Assuming that object is currently pending removal from the bandwidth pool at the start of the next billing cycle, the bandwidth pool member status will be restored and the pending cancellation removed.
func (r Network_Bandwidth_Version1_Allotment) VoidPendingServerMove(id *int, typ *string) (resp bool, err error) {
	params := []interface{}{
		id,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "voidPendingServerMove", params, &r.Options, &resp)
	return
}

// This method will void a pending cancellation on a bandwidth pool. Note however any servers that belonged to the rack will have to be restored individually using the method voidPendingServerMove($id, $type).
func (r Network_Bandwidth_Version1_Allotment) VoidPendingVdrCancellation() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Bandwidth_Version1_Allotment", "voidPendingVdrCancellation", nil, &r.Options, &resp)
	return
}

// Every piece of hardware running in SoftLayer's datacenters connected to the public, private, or management networks (where applicable) have a corresponding network component. These network components are modeled by the SoftLayer_Network_Component data type. These data types reflect the servers' local ethernet and remote management interfaces.
type Network_Component struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkComponentService returns an instance of the Network_Component SoftLayer service
func GetNetworkComponentService(sess *session.Session) Network_Component {
	return Network_Component{Session: sess}
}

func (r Network_Component) Id(id int) Network_Component {
	r.Options.Id = &id
	return r
}

func (r Network_Component) Mask(mask string) Network_Component {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Component) Filter(filter string) Network_Component {
	r.Options.Filter = filter
	return r
}

func (r Network_Component) Limit(limit int) Network_Component {
	r.Options.Limit = &limit
	return r
}

func (r Network_Component) Offset(offset int) Network_Component {
	r.Options.Offset = &offset
	return r
}

// Add VLANs as trunks to a network component. The VLANs given must be assigned to your account, and on the router to which this network component is connected. The current native VLAN (networkVlanId/networkVlan) cannot be added as a trunk. This method should be called on a network component attached directly to customer assigned hardware, though all trunking operations will occur on the uplinkComponent. A current list of VLAN trunks for a network component on a customer server can be found at 'uplinkComponent->networkVlanTrunks'.
//
// This method returns an array of SoftLayer_Network_Vlans which were added as trunks. Any requested trunks which are already trunked will be silently ignored, and will not be returned.
//
// Configuration of network hardware is done asynchronously, do not depend on the return of this call as an indication that the newly trunked VLANs will be accessible.
func (r Network_Component) AddNetworkVlanTrunks(networkVlans []datatypes.Network_Vlan) (resp []datatypes.Network_Vlan, err error) {
	params := []interface{}{
		networkVlans,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Component", "addNetworkVlanTrunks", params, &r.Options, &resp)
	return
}

// This method will remove all VLANs trunked to this network component. The native VLAN (networkVlanId/networkVlan) will remain active, and cannot be removed via the API. Returns a list of SoftLayer_Network_Vlan objects for which the trunks were removed.
func (r Network_Component) ClearNetworkVlanTrunks() (resp []datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "clearNetworkVlanTrunks", nil, &r.Options, &resp)
	return
}

// Retrieve Reboot/power (rebootDefault, rebootSoft, rebootHard, powerOn, powerOff and powerCycle) command currently executing by the server's remote management card.
func (r Network_Component) GetActiveCommand() (resp datatypes.Hardware_Component_RemoteManagement_Command_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getActiveCommand", nil, &r.Options, &resp)
	return
}

// Retrieve bandwidth graph by date.
func (r Network_Component) GetCustomBandwidthDataByDate(graphData *datatypes.Container_Graph) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		graphData,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getCustomBandwidthDataByDate", params, &r.Options, &resp)
	return
}

// Retrieve The network component linking this object to a child device
func (r Network_Component) GetDownlinkComponent() (resp datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getDownlinkComponent", nil, &r.Options, &resp)
	return
}

// Retrieve The duplex mode of a network component.
func (r Network_Component) GetDuplexMode() (resp datatypes.Network_Component_Duplex_Mode, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getDuplexMode", nil, &r.Options, &resp)
	return
}

// Retrieve The hardware that a network component resides in.
func (r Network_Component) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Component) GetHighAvailabilityFirewallFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getHighAvailabilityFirewallFlag", nil, &r.Options, &resp)
	return
}

// Retrieve A hardware switch's interface to the bandwidth pod.
func (r Network_Component) GetInterface() (resp datatypes.Network_Bandwidth_Version1_Interface, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getInterface", nil, &r.Options, &resp)
	return
}

// Retrieve The records of all IP addresses bound to a network component.
func (r Network_Component) GetIpAddressBindings() (resp []datatypes.Network_Component_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getIpAddressBindings", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Component) GetIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve Last reboot/power (rebootDefault, rebootSoft, rebootHard, powerOn, powerOff and powerCycle) command issued to the server's remote management card.
func (r Network_Component) GetLastCommand() (resp datatypes.Hardware_Component_RemoteManagement_Command_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getLastCommand", nil, &r.Options, &resp)
	return
}

// Retrieve The metric tracking object for this network component.
func (r Network_Component) GetMetricTrackingObject() (resp datatypes.Metric_Tracking_Object, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getMetricTrackingObject", nil, &r.Options, &resp)
	return
}

// Retrieve The upstream network component firewall.
func (r Network_Component) GetNetworkComponentFirewall() (resp datatypes.Network_Component_Firewall, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getNetworkComponentFirewall", nil, &r.Options, &resp)
	return
}

// Retrieve A network component's associated group.
func (r Network_Component) GetNetworkComponentGroup() (resp datatypes.Network_Component_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getNetworkComponentGroup", nil, &r.Options, &resp)
	return
}

// Retrieve All network devices in SoftLayer's network hierarchy that this device is connected to.
func (r Network_Component) GetNetworkHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getNetworkHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The VLAN that a network component's subnet is associated with.
func (r Network_Component) GetNetworkVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getNetworkVlan", nil, &r.Options, &resp)
	return
}

// Retrieve The VLANs that are trunked to this network component.
func (r Network_Component) GetNetworkVlanTrunks() (resp []datatypes.Network_Component_Network_Vlan_Trunk, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getNetworkVlanTrunks", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Component) GetObject() (resp datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Component) GetParentModule() (resp datatypes.Hardware_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getParentModule", nil, &r.Options, &resp)
	return
}

//
// **DEPRECATED - This operation will cease to function after April 4th, 2016 and will be removed from v3.2**
// Retrieve various network statistics.  The network statistics are retrieved from the network device using snmpget. Below is a list of statistics retrieved:
// * Administrative Status
// * Operational Status
// * Maximum Transmission Unit
// * In Octets
// * Out Octets
// * In Unicast Packets
// * Out Unicast Packets
// * In Multicast Packets
// * Out Multicast Packets
func (r Network_Component) GetPortStatistics() (resp datatypes.Container_Network_Port_Statistic, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getPortStatistics", nil, &r.Options, &resp)
	return
}

// Retrieve The primary IPv4 Address record for a network component.
func (r Network_Component) GetPrimaryIpAddressRecord() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getPrimaryIpAddressRecord", nil, &r.Options, &resp)
	return
}

// Retrieve A network component's subnet for its primary IP address
func (r Network_Component) GetPrimarySubnet() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getPrimarySubnet", nil, &r.Options, &resp)
	return
}

// Retrieve The primary IPv6 Address record for a network component.
func (r Network_Component) GetPrimaryVersion6IpAddressRecord() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getPrimaryVersion6IpAddressRecord", nil, &r.Options, &resp)
	return
}

// Retrieve The last five reboot/power (rebootDefault, rebootSoft, rebootHard, powerOn, powerOff and powerCycle) commands issued to the server's remote management card.
func (r Network_Component) GetRecentCommands() (resp []datatypes.Hardware_Component_RemoteManagement_Command_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getRecentCommands", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates whether the network component is participating in a group of two or more components capable of being operationally redundant, if enabled.
func (r Network_Component) GetRedundancyCapableFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getRedundancyCapableFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates whether the network component is participating in a group of two or more components which is actively providing link redundancy.
func (r Network_Component) GetRedundancyEnabledFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getRedundancyEnabledFlag", nil, &r.Options, &resp)
	return
}

// Retrieve User(s) credentials to issue commands and/or interact with the server's remote management card.
func (r Network_Component) GetRemoteManagementUsers() (resp []datatypes.Hardware_Component_RemoteManagement_User, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getRemoteManagementUsers", nil, &r.Options, &resp)
	return
}

// Retrieve A network component's routers.
func (r Network_Component) GetRouter() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getRouter", nil, &r.Options, &resp)
	return
}

// Retrieve Whether a network component's primary ip address is from a storage network subnet or not.
func (r Network_Component) GetStorageNetworkFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getStorageNetworkFlag", nil, &r.Options, &resp)
	return
}

// Retrieve A network component's subnets. A subnet is a group of IP addresses
func (r Network_Component) GetSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The network component linking this object to parent
func (r Network_Component) GetUplinkComponent() (resp datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getUplinkComponent", nil, &r.Options, &resp)
	return
}

// Retrieve The duplex mode of the uplink network component linking to this object
func (r Network_Component) GetUplinkDuplexMode() (resp datatypes.Network_Component_Duplex_Mode, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component", "getUplinkDuplexMode", nil, &r.Options, &resp)
	return
}

// Remove one or more VLANs currently attached to the uplinkComponent of this networkComponent. The VLANs given must be assigned to your account, and on the router the network component is connected to. If any VLANs not currently trunked are given, they will be silently ignored.
//
// This method should be called on a network component attached directly to customer assigned hardware, though all trunking operations will occur on the uplinkComponent. A current list of VLAN trunks for a network component on a customer server can be found at 'uplinkComponent->networkVlanTrunks'.
//
// Configuration of network hardware is done asynchronously, do not depend on the return of this call as an indication that the removed VLANs will be inaccessible.
func (r Network_Component) RemoveNetworkVlanTrunks(networkVlans []datatypes.Network_Vlan) (resp []datatypes.Network_Vlan, err error) {
	params := []interface{}{
		networkVlans,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Component", "removeNetworkVlanTrunks", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Component_Firewall data type contains general information relating to a single SoftLayer network component firewall. This is the object which ties the running rules to a specific downstream server. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Component_Firewall struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkComponentFirewallService returns an instance of the Network_Component_Firewall SoftLayer service
func GetNetworkComponentFirewallService(sess *session.Session) Network_Component_Firewall {
	return Network_Component_Firewall{Session: sess}
}

func (r Network_Component_Firewall) Id(id int) Network_Component_Firewall {
	r.Options.Id = &id
	return r
}

func (r Network_Component_Firewall) Mask(mask string) Network_Component_Firewall {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Component_Firewall) Filter(filter string) Network_Component_Firewall {
	r.Options.Filter = filter
	return r
}

func (r Network_Component_Firewall) Limit(limit int) Network_Component_Firewall {
	r.Options.Limit = &limit
	return r
}

func (r Network_Component_Firewall) Offset(offset int) Network_Component_Firewall {
	r.Options.Offset = &offset
	return r
}

// Retrieve The additional subnets linked to this network component firewall, that inherit rules from the host that the context slot is attached to.
func (r Network_Component_Firewall) GetApplyServerRuleSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component_Firewall", "getApplyServerRuleSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The billing item for a Hardware Firewall (Dedicated).
func (r Network_Component_Firewall) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component_Firewall", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The network component of the guest virtual server that this network component firewall belongs to.
func (r Network_Component_Firewall) GetGuestNetworkComponent() (resp datatypes.Virtual_Guest_Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component_Firewall", "getGuestNetworkComponent", nil, &r.Options, &resp)
	return
}

// Retrieve The network component of the switch interface that this network component firewall belongs to.
func (r Network_Component_Firewall) GetNetworkComponent() (resp datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component_Firewall", "getNetworkComponent", nil, &r.Options, &resp)
	return
}

// Retrieve The update requests made for this firewall.
func (r Network_Component_Firewall) GetNetworkFirewallUpdateRequest() (resp []datatypes.Network_Firewall_Update_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component_Firewall", "getNetworkFirewallUpdateRequest", nil, &r.Options, &resp)
	return
}

// getObject returns a SoftLayer_Network_Firewall_Module_Context_Interface_AccessControlList_Network_Component object. You can only get objects for servers attached to your account that have a network firewall enabled.
func (r Network_Component_Firewall) GetObject() (resp datatypes.Network_Component_Firewall, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component_Firewall", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The currently running rule set of this network component firewall.
func (r Network_Component_Firewall) GetRules() (resp []datatypes.Network_Component_Firewall_Rule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component_Firewall", "getRules", nil, &r.Options, &resp)
	return
}

// Retrieve The additional subnets linked to this network component firewall.
func (r Network_Component_Firewall) GetSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Component_Firewall", "getSubnets", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_ContentDelivery_Account data type models an individual CDN account. CDN accounts contain references to the SoftLayer customer account they belong to, login credentials for upload services, and a CDN account's status. Please contact SoftLayer sales to purchase or cancel a CDN account
type Network_ContentDelivery_Account struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkContentDeliveryAccountService returns an instance of the Network_ContentDelivery_Account SoftLayer service
func GetNetworkContentDeliveryAccountService(sess *session.Session) Network_ContentDelivery_Account {
	return Network_ContentDelivery_Account{Session: sess}
}

func (r Network_ContentDelivery_Account) Id(id int) Network_ContentDelivery_Account {
	r.Options.Id = &id
	return r
}

func (r Network_ContentDelivery_Account) Mask(mask string) Network_ContentDelivery_Account {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_ContentDelivery_Account) Filter(filter string) Network_ContentDelivery_Account {
	r.Options.Filter = filter
	return r
}

func (r Network_ContentDelivery_Account) Limit(limit int) Network_ContentDelivery_Account {
	r.Options.Limit = &limit
	return r
}

func (r Network_ContentDelivery_Account) Offset(offset int) Network_ContentDelivery_Account {
	r.Options.Offset = &offset
	return r
}

// Internap servers attempts to validate a token before serving a protected content. SoftLayer customer does not need to invoke this method.  Please refer to [[SoftLayer_Network_ContentDelivery_Authentication_Token|Authentication Token]] object for more details on Content Authentication Service.
func (r Network_ContentDelivery_Account) AuthenticateResourceRequest(parameter *datatypes.Container_Network_ContentDelivery_Authentication_Parameter) (resp bool, err error) {
	params := []interface{}{
		parameter,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "authenticateResourceRequest", params, &r.Options, &resp)
	return
}

// You can further organize your contents on the CDN FTP server by creating sub directories.  This method creates a directory on the CDN FTP server. A user must have CDN_FILE_MANAGE privilege to use this method. A directory name must be an absolute path and you can only create sub directories in /media folder.
func (r Network_ContentDelivery_Account) CreateDirectory(directoryName *string) (resp bool, err error) {
	params := []interface{}{
		directoryName,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "createDirectory", params, &r.Options, &resp)
	return
}

// This method allows you to create a default CDN FTP user record on the ftp.cdnlayer.service.softlayer.com server. As with a CDN FTP user account, you can upload contents to the CDN host server through the SoftLayer private network.  SoftLayer currently allows only one FTP user for each CDN account. Your default CDN FTP user record is created upon successful creation of a CDN account.  You may not need to use this method at all. This is provided in support of the previous CDN customers. SoftLayer may offer multiple CDN FTP users for a single CDN account in the future.
//
// Optionally, you can provide a new password when invoking this method and a new password must follow the rules below:
// * ...must be between 8 and 20 characters long
// * ...must be an alphanumeric value
// * ...can contain these characters: - _ ! % # $ ^ & *
func (r Network_ContentDelivery_Account) CreateFtpUser(newPassword *string) (resp bool, err error) {
	params := []interface{}{
		newPassword,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "createFtpUser", params, &r.Options, &resp)
	return
}

// With Origin Pull, content is pulled from your origin server as needed and then delivered to visitors. You do not need to upload your files to the CDN FTP: you can utilize the files that currently exist on your origin server. It will take 10 to 15 minutes for this to take effect after you create an Origin Pull rule. Origin Pull is only supported for HTTP protocol and you would continue to use the CDN FTP for Flash and Windows Media streaming services.
//
// A valid origin host can include a directory information.  You may include an authentication username and password along with an origin host. If you set an authentication username and password, CDN servers will include "Authorization:" header in every request. You may use the "Authorization:" header to grant access to CDN servers or you may use it to distinguish CDN servers from normal visitors. Here is a list of valid origin hosts:
// * www.website.com
// * www.website.com/cdn_content
// * cdn_user:password@www.website.com
// * cdn_user:password@www.website.com/images
//
//
// An authentication username should be an alphanumeric string and allowed special characters are . - _<br /> An authentication password should be an alphanumeric string and allowed special characters are . - _ ! # $ % ^ & *<br /> Both username and password must be between 3 to 10 characters long.
//
// CDN nodes will cache your contents and you can control cache lifetime by modifying your web server's configuration. This method also creates a FTP directory restriction upon successful Origin Pull set up. You will not be able to access /media/http directory since contents will be pulled from your origin server. An origin domain must be a valid domain name and it can contain path information. This can help you organize contents on your origin server. For example, you could set an origin domain as: mydomain.com/cdn_contents
//
// A CNAME record allows you to have a customized URL. You can get rid of your CDN account name from the URL. A valid CNAME for the Origin Pull method must point to <CDN_AcccountName>.http.cdn.softlayer.net.
//
// There are 2 types of origin pull mappings.  The one with a CNAME record or the one without a CNAME record and they work very differently.
//
// gzip is supported if your web server sends a proper gzip header. For more details, visit our [http://knowledgelayer.softlayer.com/topic/cdn KnowledgeLayer]
func (r Network_ContentDelivery_Account) CreateOriginPullMapping(mappingObject *datatypes.Container_Network_ContentDelivery_OriginPull_Mapping) (resp bool, err error) {
	params := []interface{}{
		mappingObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "createOriginPullMapping", params, &r.Options, &resp)
	return
}

// This method is deprecated, please use [[[[SoftLayer_Network_ContentDelivery_Account::createOriginPullMapping|createOriginPullMapping]] method instead.
func (r Network_ContentDelivery_Account) CreateOriginPullRule(originDomain *string, cnameRecord *string) (resp bool, err error) {
	params := []interface{}{
		originDomain,
		cnameRecord,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "createOriginPullRule", params, &r.Options, &resp)
	return
}

// You need to specify a directory on your CDN FTP or on your origin host in which your secure content resides to enable the token authentication . It will take about about 30 minutes for a newly configured token authentication directory to take effect.
func (r Network_ContentDelivery_Account) CreateTokenAuthenticationDirectory(directory *string, mediaType *string) (resp bool, err error) {
	params := []interface{}{
		directory,
		mediaType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "createTokenAuthenticationDirectory", params, &r.Options, &resp)
	return
}

// This method deletes your FTP user record on the ftp.cdnlayer.service.softlayer.com server. Refer to the service overview of [[SoftLayer_Network_ContentDelivery_Account::createFtpUser|createFtpUser]] method for more information on the CDN FTP server.
func (r Network_ContentDelivery_Account) DeleteFtpUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "deleteFtpUser", nil, &r.Options, &resp)
	return
}

// This method removes an Origin Pull domain rule.  Once an Origin Pull rule is removed, you will be able to access the /media/http directory. It will take 10 to 15 minutes for this to take effect after you remove your Origin Pull rule.  Cached contents on CDN POPs may live longer than 15 minutes.
func (r Network_ContentDelivery_Account) DeleteOriginPullRule(originMappingId *string) (resp bool, err error) {
	params := []interface{}{
		originMappingId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "deleteOriginPullRule", params, &r.Options, &resp)
	return
}

// This method disables CDN access log.
func (r Network_ContentDelivery_Account) DisableLogging() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "disableLogging", nil, &r.Options, &resp)
	return
}

// This method enables CDN access log.
func (r Network_ContentDelivery_Account) EnableLogging() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "enableLogging", nil, &r.Options, &resp)
	return
}

// Retrieve The customer account that a CDN account belongs to.
func (r Network_ContentDelivery_Account) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getAccount", nil, &r.Options, &resp)
	return
}

// This method returns bandwidth data for each POP. [[SoftLayer_Container_Network_ContentDelivery_Bandwidth_PointsOfPresence_Summary|POP Bandwidth]] object contains a starting time, ending time, total bytes, POP name and bandwidth unit.
//
// POP bandwidth data is updated everyday at 22:50 CST (or CDT). It queries and stores POP data from the day before. It is a more resource intensive process than a regular CDN bandwidth update thus we run this once a day. Since the POP bandwidth data is delayed for a day, there is no correction process for POP data. The POP bandwidth is not associated with any billing process and is mainly used to generate a POP bandwidth graph.
func (r Network_ContentDelivery_Account) GetAllPopsBandwidthData(beginDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp []datatypes.Container_Network_ContentDelivery_Bandwidth_PointsOfPresence_Summary, err error) {
	params := []interface{}{
		beginDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getAllPopsBandwidthData", params, &r.Options, &resp)
	return
}

// This method returns a bandwidth graph for every POP wrapped in [[SoftLayer_Container_Bandwidth_GraphOutputsExtended|Bandwidth Graph]] object. A POP bandwidth graph shows bandwidth consumption per each POP in a bar graph. [[SoftLayer_Container_Bandwidth_GraphOutputsExtended|Bandwidth Graph]] object contains a begin time, end time, title of the graph, binary date, in and outbound total bandwidth in bytes
func (r Network_ContentDelivery_Account) GetAllPopsBandwidthImage(title *string, beginDateTime *datatypes.Time, endDateTime *datatypes.Time, unit *string) (resp datatypes.Container_Bandwidth_GraphOutputsExtended, err error) {
	params := []interface{}{
		title,
		beginDateTime,
		endDateTime,
		unit,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getAllPopsBandwidthImage", params, &r.Options, &resp)
	return
}

// Retrieve The CDN account id that this CDN account is associated with.
func (r Network_ContentDelivery_Account) GetAssociatedCdnAccountId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getAssociatedCdnAccountId", nil, &r.Options, &resp)
	return
}

// Retrieve The IP addresses that are used for the content authentication service.
func (r Network_ContentDelivery_Account) GetAuthenticationIpAddresses() (resp []datatypes.Network_ContentDelivery_Authentication_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getAuthenticationIpAddresses", nil, &r.Options, &resp)
	return
}

// CDN servers will invoke a Web Service method to validate a content authentication token. This method returns all token validation web service endpoints set for a CDN account. You can override the default web service by calling [[SoftLayer_Network_ContentDelivery_Authentication_Token|setContentAuthenticationWsdl setContentAuthenticationWsdl]] method.
func (r Network_ContentDelivery_Account) GetAuthenticationServiceEndpoints() (resp []datatypes.Container_Network_ContentDelivery_Authentication_ServiceEndpoint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getAuthenticationServiceEndpoints", nil, &r.Options, &resp)
	return
}

// This method returns bandwidth data for a given time range.  It returns an array of [[SoftLayer_Container_Network_ContentDelivery_Bandwidth_Summary|bandwidth summary]] objects. [[SoftLayer_Container_Network_ContentDelivery_Bandwidth_Summary|Bandwidth summary]] object contains a beginning time, ending time, and bandwidth in bytes.
//
// A Beginning and ending date parameters have to be a timestamp in "yyyy-mm-dd HH24:mi:ss" format and it assumes the time is in Central Standard Time (CST) or Central Daylight Time (CDT) time zone. CDN bandwidth data is stored in Greenwich Mean Time (GMT) internally and converts a beginning and ending time to GMT before querying.
//
// Unlike server bandwidth, CDN bandwidth returns total bytes consumed within an hour. For example, if you pass "2008-10-10 00:00:00" for a beginning time and "2008-10-10 05:00:00" for an ending time, your return value will have 6 elements of bandwidth summary objects. The first bandwidth summary object will have the total bytes consumed between 2008-10-10 00:00:00 and 2008-10-10 05:00:00. And the last object will have the bandwidth consumed between 2008-10-10 05:00:00 and 2008-10-10 00:59:59. The bandwidth data is updated at 10 minutes after every hour.  The queried data is on a two hour time delay. The two hour delay is required to gather bandwidth data from each POP and that is the minimum delay required to create a feasible graph. It usually takes about 8 hours to reconcile all the data from every CDN POP. This hourly data is corrected after 24 hours if necessary.  If you consume a large amount of bandwidth, your bandwidth data will be updated the next day.
func (r Network_ContentDelivery_Account) GetBandwidthData(beginDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp []datatypes.Container_Network_ContentDelivery_Bandwidth_Summary, err error) {
	params := []interface{}{
		beginDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getBandwidthData", params, &r.Options, &resp)
	return
}

// This method returns bandwidth data for a given time range.  It returns an array of [[SoftLayer_Container_Network_ContentDelivery_Report_Usage|bandwidth usage report]] objects.
//
// These will be first sorted by timestamp, and there will be one entry with that timestamp for each enabled region. The region type 'NONE' is provided only when non-region-specific data is returned. [[SoftLayer_Container_Network_ContentDelivery_Report_Usage|bandwidth usage report]] objects with a region will never contain non-region-specific data. Non-region-specific values are standardTotal and sslTotal; standardTotal is computed by adding the HTTP Large, Windows Media, Flash and Application Delivery Network bandwidth. The sslTotal is computed by adding the HTTP Large SSL bandwidth and the Application Delivery Network SSL bandwidth.
//
// A Beginning and ending date parameters have to be a timestamp in "yyyy-mm-dd HH24:mi:ss" format and it assumes the time is in Central Standard Time (CST) or Central Daylight Time (CDT) time zone. CDN bandwidth data is stored in Greenwich Mean Time (GMT) internally and converts a beginning and ending time to GMT before querying.
//
// Unlike server bandwidth, CDN bandwidth returns total bytes consumed within an hour. For example, if you pass "2008-10-10 00:00:00" for a beginning time and "2008-10-10 05:00:00" for an ending time, your return value will have 6 elements of bandwidth summary objects. The first bandwidth summary object will have the total bytes consumed between 2008-10-10 00:00:00 and 2008-10-10 05:00:00. And the last object will have the bandwidth consumed between 2008-10-10 05:00:00 and 2008-10-10 00:59:59. The bandwidth data is updated at 10 minutes after every hour.  The queried data is on a two hour time delay. The two hour delay is required to gather bandwidth data from each POP and that is the minimum delay required to create a feasible graph. It usually takes about 8 hours to reconcile all the data from every CDN POP. This hourly data is corrected after 24 hours if necessary.  If you consume a large amount of bandwidth, your bandwidth data will be updated the next day.
func (r Network_ContentDelivery_Account) GetBandwidthDataWithTypes(beginDateTime *datatypes.Time, endDateTime *datatypes.Time, period *string) (resp []datatypes.Container_Network_ContentDelivery_Report_Usage, err error) {
	params := []interface{}{
		beginDateTime,
		endDateTime,
		period,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getBandwidthDataWithTypes", params, &r.Options, &resp)
	return
}

// This method returns a bandwidth graph wrapped in [[SoftLayer_Container_Bandwidth_GraphOutputsExtended|Bandwidth Graph]] object. [[SoftLayer_Container_Bandwidth_GraphOutputsExtended|Bandwidth Graph]] object contains a starting time, ending time, graph title, graph binary data, and in and outbound total bytes.
func (r Network_ContentDelivery_Account) GetBandwidthImage(title *string, beginDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp datatypes.Container_Bandwidth_GraphOutputsExtended, err error) {
	params := []interface{}{
		title,
		beginDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getBandwidthImage", params, &r.Options, &resp)
	return
}

// Retrieve The current billing item for a CDN account.
func (r Network_ContentDelivery_Account) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The name of a CDN account.
func (r Network_ContentDelivery_Account) GetCdnAccountName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getCdnAccountName", nil, &r.Options, &resp)
	return
}

// Retrieve A brief note on a CDN account.
func (r Network_ContentDelivery_Account) GetCdnAccountNote() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getCdnAccountNote", nil, &r.Options, &resp)
	return
}

// Retrieve The solution type of a CDN account.
func (r Network_ContentDelivery_Account) GetCdnSolutionName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getCdnSolutionName", nil, &r.Options, &resp)
	return
}

// An origin pull mapping is a combination of your customer origin record and a CNAME (optional) record. You can now keep track of your customer origin records separate from your CNAME records. This service returns your customer origin records.
func (r Network_ContentDelivery_Account) GetCustomerOrigins(mediaType *string) (resp []datatypes.Container_Network_ContentDelivery_OriginPull_Mapping, err error) {
	params := []interface{}{
		mediaType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getCustomerOrigins", params, &r.Options, &resp)
	return
}

// Retrieve Indicates if CDN account is dependent on other service. If set, this CDN account is limited to these services: createOriginPullMapping, deleteOriginPullRule, getOriginPullMappingInformation, getCdnUrls, purgeCache, loadContent, manageHttpCompression
func (r Network_ContentDelivery_Account) GetDependantServiceFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getDependantServiceFlag", nil, &r.Options, &resp)
	return
}

// This method returns an array of [[SoftLayer_Container_Network_Directory_Listing|Directory Listing]] objects. You must have CDN_FILE_MANAGE privilege and you can only retrieve directory information within <b>/media</b> directory. A [[SoftLayer_Container_Network_Directory_Listing|Directory Listing]] object contains type (indicating whether it is a file or a directory), name and file count if it is a directory.
func (r Network_ContentDelivery_Account) GetDirectoryInformation(directoryName *string) (resp []datatypes.Container_Network_Directory_Listing, err error) {
	params := []interface{}{
		directoryName,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getDirectoryInformation", params, &r.Options, &resp)
	return
}

// This method returns disk space usage data for your CDN FTP.
func (r Network_ContentDelivery_Account) GetDiskSpaceUsageDataByDate(beginDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		beginDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getDiskSpaceUsageDataByDate", params, &r.Options, &resp)
	return
}

// This method returns a disk usage graph wrapped in [[SoftLayer_Container_Bandwidth_GraphOutputsExtended|Bandwidth Graph]] object. [[SoftLayer_Container_Bandwidth_GraphOutputsExtended|Bandwidth Graph]] object contains a starting time, ending time, graph title, graph binary data, and in and outbound total bytes.
func (r Network_ContentDelivery_Account) GetDiskSpaceUsageImageByDate(beginDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		beginDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getDiskSpaceUsageImageByDate", params, &r.Options, &resp)
	return
}

// This method returns your login credentials to the CDN FTP server (ftp.cdnlayer.service.softlayer.com server). You must have CDN_FILE_MANAGE privilege. Refer to the service overview of [[SoftLayer_Network_ContentDelivery_Account::createFtpUser|createFtpUser]] method for more information on the CDN FTP server.
//
// If you want to download raw log files, prefix the username with "LOGS-" (without quotes) when logging in. SoftLayer designed CDN accounts so they can have multiple CDN FTP users. However, this method returns the default CDN FTP user information: multi user support may be implemented in the future.
func (r Network_ContentDelivery_Account) GetFtpAttributes() (resp datatypes.Container_Network_Authentication_Data, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getFtpAttributes", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates if it is a legacy CDN or not
func (r Network_ContentDelivery_Account) GetLegacyCdnFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getLegacyCdnFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates if CDN logging is enabled.
func (r Network_ContentDelivery_Account) GetLogEnabledFlag() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getLogEnabledFlag", nil, &r.Options, &resp)
	return
}

// This method returns CDN URLs for static file (http), Flash streaming (rtmp) and Window Media (mms) streaming services. You can generate your CDN URLs based on the information retrieved by this method.
func (r Network_ContentDelivery_Account) GetMediaUrls() (resp []datatypes.Container_Network_ContentDelivery_SupportedProtocol, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getMediaUrls", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_ContentDelivery_Account object whose ID number corresponds to the ID number of the initial parameter passed to the SoftLayer_Network_ContentDelivery_Account service. You can only retrieve CDN accounts assigned to your SoftLayer customer account.
func (r Network_ContentDelivery_Account) GetObject() (resp datatypes.Network_ContentDelivery_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getObject", nil, &r.Options, &resp)
	return
}

// This method returns a list of origin pull configuration data.
func (r Network_ContentDelivery_Account) GetOriginPullMappingInformation() (resp []datatypes.Container_Network_ContentDelivery_OriginPull_Mapping, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getOriginPullMappingInformation", nil, &r.Options, &resp)
	return
}

// This method returns CDN URLs that supports Origin Pull mappings.
func (r Network_ContentDelivery_Account) GetOriginPullSupportedMediaUrls() (resp []datatypes.Container_Network_ContentDelivery_SupportedProtocol, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getOriginPullSupportedMediaUrls", nil, &r.Options, &resp)
	return
}

// This method returns the domain name of your Origin Pull rule.  It assumes you have already setup an Origin Pull rule.  Otherwise, it will throw an exception. A returning value is the value of the first parameter (origin pull domain) you provided to [[SoftLayer_Network_ContentDelivery_Account::createOriginPullRule|createOriginPullRule]] method. See Error Handling section below for possible errors.
func (r Network_ContentDelivery_Account) GetOriginPullUrl() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getOriginPullUrl", nil, &r.Options, &resp)
	return
}

// This method returns an array of CDN POPs (Points of Presence) object. [[SoftLayer_Container_Network_ContentDelivery_PointsOfPresence|POP object]] object contains the POP id and name.
func (r Network_ContentDelivery_Account) GetPopNames() (resp []datatypes.Container_Network_ContentDelivery_PointsOfPresence, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getPopNames", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates if customer is allowed to access the CDN provider's management portal.
func (r Network_ContentDelivery_Account) GetProviderPortalAccessFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getProviderPortalAccessFlag", nil, &r.Options, &resp)
	return
}

// This method returns your login credentials to the CDN provider portal.
func (r Network_ContentDelivery_Account) GetProviderPortalCredentials() (resp datatypes.Container_Network_Authentication_Data, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getProviderPortalCredentials", nil, &r.Options, &resp)
	return
}

// Retrieve A CDN account's status presented in a more detailed data type.
func (r Network_ContentDelivery_Account) GetStatus() (resp datatypes.Network_ContentDelivery_Account_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getStatus", nil, &r.Options, &resp)
	return
}

// This method returns all token authentication directories.
func (r Network_ContentDelivery_Account) GetTokenAuthenticationDirectories() (resp []datatypes.Container_Network_Directory_Listing, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getTokenAuthenticationDirectories", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates if the token authentication service is enabled or not.
func (r Network_ContentDelivery_Account) GetTokenAuthenticationEnabledFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getTokenAuthenticationEnabledFlag", nil, &r.Options, &resp)
	return
}

// This method returns your login credentials to the public CDN FTP.
func (r Network_ContentDelivery_Account) GetVendorFtpAttributes() (resp datatypes.Container_Network_Authentication_Data, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "getVendorFtpAttributes", nil, &r.Options, &resp)
	return
}

// Whether you are using Origin Pull or POP Pull, your content will be transferred and cached on CDN POP (node) on the initial request. If you wish to load your content to all CDN POPs, you may use this service to accomplish that. Please keep in mind, it will take about 10 to 15 minutes to load content to all CDN POPs depending on the load.
//
// You can only specify 5 URLs at a time.
func (r Network_ContentDelivery_Account) LoadContent(objectUrls []string) (resp bool, err error) {
	params := []interface{}{
		objectUrls,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "loadContent", params, &r.Options, &resp)
	return
}

// HTTP Compression is used to reduce the bandwidth used to deliver an object. You can specify list of content types that needs to be compressed. If you omit the content type parameter, these values will be used by default:
// * text/plain
// * text/html
// * text/css
// * application/x-javascript
// * text/javascript
//
//
// Note that files larger than 1MB will never be served with compression regardless of whether their content-type is enabled for compression.
func (r Network_ContentDelivery_Account) ManageHttpCompression(enableFlag *bool, mimeTypes []string) (resp bool, err error) {
	params := []interface{}{
		enableFlag,
		mimeTypes,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "manageHttpCompression", params, &r.Options, &resp)
	return
}

// CDN's cache mechanism works similar to that of web browsers. When CDN pulls a file from your origin server or from your CDN FTP directory for the first time, it creates a cache file on itself. CDN re-uses cache files to save trips to the origin server and thus it speeds up delivering content to visitors. This method removes cached objects on every server in the CDN network. If you see a stale content or a file that send an incorrect header, purging cache will correct the issue. CDN will pull a fresh content from your origin server or your CDN FTP.
//
// This method takes an array of URLs. A URL must be exact as it is being requested by clients. An example URLs may look like this:
// * http://<your CDN username>.http.cdn.softlayer.net/mycdnname/some_file.txt
//
//
// If you created a CNAME that points to CDN host, use your CNAME URL instead.
// * http://image.mydomain.com/some_file.txt
//
//
// It takes approximately 3-5 minutes for the system to delete the requested object on every CDN server from submission .
func (r Network_ContentDelivery_Account) PurgeCache(objectUrls []string) (resp []datatypes.Container_Network_ContentDelivery_PurgeService_Response, err error) {
	params := []interface{}{
		objectUrls,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "purgeCache", params, &r.Options, &resp)
	return
}

// If you want to turn off the token authentication, use this method to remove a directory from the token authentication directory.
func (r Network_ContentDelivery_Account) RemoveAuthenticationDirectory(directory *string, mediaType *string) (resp bool, err error) {
	params := []interface{}{
		directory,
		mediaType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "removeAuthenticationDirectory", params, &r.Options, &resp)
	return
}

// With this method you can remove a file or a directory on the CDN FTP server. If a source name ends with a slash (/), this method assumes it is a directory.  A source name must be an absolute path. It does not check to see if a file or directory exists before deletion. You can only remove files and directories that are in /media folder. Be sure to catch an exception for the detail on an error.
func (r Network_ContentDelivery_Account) RemoveFile(source *string) (resp bool, err error) {
	params := []interface{}{
		source,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "removeFile", params, &r.Options, &resp)
	return
}

// CDN servers will invoke a Web Service method to validate a content authentication token. CDN uses the default Web Service provided by SoftLayer to validate a token. A customer can use their own implementation of the token authentication Web Service. A valid SOAP WSDL will look similar [https://manage.softlayer.com/CdnService/authenticationWsdlExample/wsdl this].
func (r Network_ContentDelivery_Account) SetAuthenticationServiceEndpoint(webserviceEndpoint *string, protocol *string) (resp bool, err error) {
	params := []interface{}{
		webserviceEndpoint,
		protocol,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "setAuthenticationServiceEndpoint", params, &r.Options, &resp)
	return
}

// With a CDN FTP, you can upload contents to CDN host server. Once you uploaded contents, your contents will be fetched by the CDN POP (Points of Presence) servers as needed.
//
// CDN supports three protocols: Flash streaming (rtmp), Window Media streaming (mms) and HTTP. Once you log in to the CDN FTP server, you will see three directories under /media directory.  You have to upload your contents to a proper directory to use the different services. Refer to [[SoftLayer_Network_ContentDelivery_Account|CDN Account]] service overview for details on the CDN FTP server. "gzip" is supported if you compress your content before uploading and you have to change its extension to ".gz".  [SoftLayer_Network_ContentDelivery_Account::createOriginPullRule|Origin Pull] also supports "gzip" contents and you don't have to modify file extension with Origin Pull. Once uploaded, your contents should be available almost immediately to visitors.  However, it may take about 30 minutes to propagate files to the entire CDN network after uploading. For more details, visit our [hhttp://knowledgelayer.softlayer.com/topic/cdn KnowledgeLayer]
//
// This method updates the password for your CDN FTP account on the ftp.cdnlayer.service.softlayer.com server. You must provide an alphanumeric value for a new password.  - _ ! % # $ ^ & * characters are allowed beside an alphanumeric string.
func (r Network_ContentDelivery_Account) SetFtpPassword(newPassword *string) (resp bool, err error) {
	params := []interface{}{
		newPassword,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "setFtpPassword", params, &r.Options, &resp)
	return
}

// This method allows you to edit CDN account note. The maximum length for CDN account note is 30 characters.
func (r Network_ContentDelivery_Account) UpdateNote(note *string) (resp bool, err error) {
	params := []interface{}{
		note,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "updateNote", params, &r.Options, &resp)
	return
}

// With this method, you can upload binary data to the CDN FTP server.  This method supports files up to 20 Mega Bytes. You need to use the CDN FTP (ftp.cdnlayer.service.softlayer.com) to upload files larger than 20 MB.  This method takes [[SoftLayer_Container_Utility_File_Attachment]] a first parameter. A target name must be an absolute path and you can only upload a file to a directory that is in /media folder.
func (r Network_ContentDelivery_Account) UploadStream(source *datatypes.Container_Utility_File_Attachment, target *string) (resp bool, err error) {
	params := []interface{}{
		source,
		target,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Account", "uploadStream", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_ContentDelivery_Authentication_Address data type models an individual IP address that CDN allow or deny access from.
type Network_ContentDelivery_Authentication_Address struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkContentDeliveryAuthenticationAddressService returns an instance of the Network_ContentDelivery_Authentication_Address SoftLayer service
func GetNetworkContentDeliveryAuthenticationAddressService(sess *session.Session) Network_ContentDelivery_Authentication_Address {
	return Network_ContentDelivery_Authentication_Address{Session: sess}
}

func (r Network_ContentDelivery_Authentication_Address) Id(id int) Network_ContentDelivery_Authentication_Address {
	r.Options.Id = &id
	return r
}

func (r Network_ContentDelivery_Authentication_Address) Mask(mask string) Network_ContentDelivery_Authentication_Address {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_ContentDelivery_Authentication_Address) Filter(filter string) Network_ContentDelivery_Authentication_Address {
	r.Options.Filter = filter
	return r
}

func (r Network_ContentDelivery_Authentication_Address) Limit(limit int) Network_ContentDelivery_Authentication_Address {
	r.Options.Limit = &limit
	return r
}

func (r Network_ContentDelivery_Authentication_Address) Offset(offset int) Network_ContentDelivery_Authentication_Address {
	r.Options.Offset = &offset
	return r
}

// This method creates an authentication IP record.  Required parameters are
//
//
// * cdnAccountId - A CDN account id that belongs to your SoftLayer Account
// * ipAddress - An IP address or a IP range
// * accessType- It can be "ALLOW" or "DENY"
func (r Network_ContentDelivery_Authentication_Address) CreateObject(templateObject *datatypes.Network_ContentDelivery_Authentication_Address) (resp datatypes.Network_ContentDelivery_Authentication_Address, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Address", "createObject", params, &r.Options, &resp)
	return
}

// This method deletes an authentication IP address.
func (r Network_ContentDelivery_Authentication_Address) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Address", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method let you edit an authentication IP object by passing a modified object.
func (r Network_ContentDelivery_Authentication_Address) EditObject(templateObject *datatypes.Network_ContentDelivery_Authentication_Address) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Address", "editObject", params, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_ContentDelivery_Authentication_Address object whose ID number corresponds to the ID number of the initial parameter passed to the SoftLayer_Network_ContentDelivery_Authentication_Address service. You can only retrieve authentication IP addresses assigned to one of your CDN account.
func (r Network_ContentDelivery_Authentication_Address) GetObject() (resp datatypes.Network_ContentDelivery_Authentication_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Address", "getObject", nil, &r.Options, &resp)
	return
}

// The authentication IP address match occurs from the higher priority IP to the lower. This method will be helpful if you want to modify the order (priority) of the authentication IP addresses. You can use this method instead of editing individual authentication IP addresses.
//
// You can retrieve authentication IP address using [[SoftLayer_Network_ContentDelivery_Account::getAuthenticationIpAddresses|getAuthenticationIpAddresses]] method. Then, rearrange the authentication IP addresses and pass them to this method. When creating template objects as parameter, make sure to include the id of each authentication IP addresses. You must provide every authentication IP address.  New priorities will be assigned to each authentication IP addresses in the order of they are passed.
func (r Network_ContentDelivery_Authentication_Address) RearrangeAuthenticationIp(cdnAccountId *int, templateObjects []datatypes.Network_ContentDelivery_Authentication_Address) (resp bool, err error) {
	params := []interface{}{
		cdnAccountId,
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Address", "rearrangeAuthenticationIp", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_ContentDelivery_Authentication_Address data type models an individual IP address that CDN allow or deny access from.
type Network_ContentDelivery_Authentication_Token struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkContentDeliveryAuthenticationTokenService returns an instance of the Network_ContentDelivery_Authentication_Token SoftLayer service
func GetNetworkContentDeliveryAuthenticationTokenService(sess *session.Session) Network_ContentDelivery_Authentication_Token {
	return Network_ContentDelivery_Authentication_Token{Session: sess}
}

func (r Network_ContentDelivery_Authentication_Token) Id(id int) Network_ContentDelivery_Authentication_Token {
	r.Options.Id = &id
	return r
}

func (r Network_ContentDelivery_Authentication_Token) Mask(mask string) Network_ContentDelivery_Authentication_Token {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_ContentDelivery_Authentication_Token) Filter(filter string) Network_ContentDelivery_Authentication_Token {
	r.Options.Filter = filter
	return r
}

func (r Network_ContentDelivery_Authentication_Token) Limit(limit int) Network_ContentDelivery_Authentication_Token {
	r.Options.Limit = &limit
	return r
}

func (r Network_ContentDelivery_Authentication_Token) Offset(offset int) Network_ContentDelivery_Authentication_Token {
	r.Options.Offset = &offset
	return r
}

// This method is deprecated! Use the [[SoftLayer_Network_ContentDelivery_Authentication_Token::getTimedToken|getTimedToken]] method.
//
// This method creates a managed authentication token. When passing a parameter, the only required value is your CDN account id which can be obtained from the [[SoftLayer_Account::getCdnAccounts|getCdnAccounts]] method. There are 3 optional parameters you can pass:
//
//
// * name - This helps you keep track of managed tokens.
// * referrer - If set, the token validation will check the client's referrer. Keep in mind, if a client doesn't have the referrer information, the token validation will fail.
// * clientIp - If set, the token validation will check the client's IP address.
//
//
func (r Network_ContentDelivery_Authentication_Token) CreateObject(templateObject *datatypes.Network_ContentDelivery_Authentication_Token) (resp datatypes.Network_ContentDelivery_Authentication_Token, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Token", "createObject", params, &r.Options, &resp)
	return
}

// This method is deprecated!
//
// This method returns all managed tokens for a CDN account.
func (r Network_ContentDelivery_Authentication_Token) GetAllManagedTokens(cdnAccountId *int) (resp []datatypes.Network_ContentDelivery_Authentication_Token, err error) {
	params := []interface{}{
		cdnAccountId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Token", "getAllManagedTokens", params, &r.Options, &resp)
	return
}

// This method is deprecated!
//
// getObject retrieves the SoftLayer_Network_ContentDelivery_Authentication_Token object whose ID number corresponds to the ID number of the initial parameter passed to the SoftLayer_Network_ContentDelivery_Authentication_Token service. You can only retrieve managed tokens assigned to one of your CDN account.
func (r Network_ContentDelivery_Authentication_Token) GetObject() (resp datatypes.Network_ContentDelivery_Authentication_Token, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Token", "getObject", nil, &r.Options, &resp)
	return
}

// This method returns an authentication token that expires after the seconds you specify. You can provide number of seconds to manage the token life.  This parameter sets the expiration time for a token. A valid life time must be an integer between 60 and 604800 (1 week). A customer can also provide client ip and (or) referrer information.  If used, a client from the same IP and referrer can view the protected contents.
//
// A valid IP address must be an IPv4 format or an IP block. if you want to block access from IP 211.37.0.0/16, you can enter "211.37." instead. IP blocks can be specified in the manner of "8bit times n".
//
// The referrer is the URL of the previous webpage from which a link was followed.  A referrer should not include "http://" prefix and it can be maximum of 30 characters.
func (r Network_ContentDelivery_Authentication_Token) GetTimedToken(cdnAccountId *int, tokenLife *int, clientIp *string, referrer *string, mediaType *string) (resp string, err error) {
	params := []interface{}{
		cdnAccountId,
		tokenLife,
		clientIp,
		referrer,
		mediaType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Token", "getTimedToken", params, &r.Options, &resp)
	return
}

// This method is deprecated!
//
// This method revokes all managed tokens belong to a CDN account.
func (r Network_ContentDelivery_Authentication_Token) RevokeAllManagedTokens(cdnAccountId *int) (resp bool, err error) {
	params := []interface{}{
		cdnAccountId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Token", "revokeAllManagedTokens", params, &r.Options, &resp)
	return
}

// This method revokes all tokens belong to a CDN account.  Valid media types are "HTTP", "FLASH" and "WM".
func (r Network_ContentDelivery_Authentication_Token) RevokeAllTokens(cdnAccountId *int, mediaType *string) (resp bool, err error) {
	params := []interface{}{
		cdnAccountId,
		mediaType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Token", "revokeAllTokens", params, &r.Options, &resp)
	return
}

// This method is deprecated!
//
// Revokes a managed token. If you revoke a token, the token will be removed from SoftLayer's system but it will not remove your content on CDN FTP. The content that requires token validation will not be available to the visitor who is using a revoked token.
func (r Network_ContentDelivery_Authentication_Token) RevokeManagedToken(cdnAccountId *int, token *string) (resp bool, err error) {
	params := []interface{}{
		cdnAccountId,
		token,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Token", "revokeManagedToken", params, &r.Options, &resp)
	return
}

// This method is deprecated!
//
// Deletes multiple managed tokens
func (r Network_ContentDelivery_Authentication_Token) RevokeManagedTokens(templateObjects []datatypes.Network_ContentDelivery_Authentication_Token) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_ContentDelivery_Authentication_Token", "revokeManagedTokens", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Customer_Subnet data type contains general information relating to a single customer subnet (remote).
type Network_Customer_Subnet struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkCustomerSubnetService returns an instance of the Network_Customer_Subnet SoftLayer service
func GetNetworkCustomerSubnetService(sess *session.Session) Network_Customer_Subnet {
	return Network_Customer_Subnet{Session: sess}
}

func (r Network_Customer_Subnet) Id(id int) Network_Customer_Subnet {
	r.Options.Id = &id
	return r
}

func (r Network_Customer_Subnet) Mask(mask string) Network_Customer_Subnet {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Customer_Subnet) Filter(filter string) Network_Customer_Subnet {
	r.Options.Filter = filter
	return r
}

func (r Network_Customer_Subnet) Limit(limit int) Network_Customer_Subnet {
	r.Options.Limit = &limit
	return r
}

func (r Network_Customer_Subnet) Offset(offset int) Network_Customer_Subnet {
	r.Options.Offset = &offset
	return r
}

// For IPSec network tunnels, customers can create their local subnets using this method.  After the customer is created successfully, the customer subnet can then be added to the IPSec network tunnel.
func (r Network_Customer_Subnet) CreateObject(templateObject *datatypes.Network_Customer_Subnet) (resp datatypes.Network_Customer_Subnet, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Customer_Subnet", "createObject", params, &r.Options, &resp)
	return
}

// Retrieve All ip addresses associated with a subnet.
func (r Network_Customer_Subnet) GetIpAddresses() (resp []datatypes.Network_Customer_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Customer_Subnet", "getIpAddresses", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Customer_Subnet object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Customer_Subnet service. You can only retrieve the subnet whose account matches the account that your portal user is assigned to.
func (r Network_Customer_Subnet) GetObject() (resp datatypes.Network_Customer_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Customer_Subnet", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Firewall_AccessControlList data type contains general information relating to a single SoftLayer firewall access to controll list. This is the object which ties the running rules to a specific context. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Firewall_AccessControlList struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkFirewallAccessControlListService returns an instance of the Network_Firewall_AccessControlList SoftLayer service
func GetNetworkFirewallAccessControlListService(sess *session.Session) Network_Firewall_AccessControlList {
	return Network_Firewall_AccessControlList{Session: sess}
}

func (r Network_Firewall_AccessControlList) Id(id int) Network_Firewall_AccessControlList {
	r.Options.Id = &id
	return r
}

func (r Network_Firewall_AccessControlList) Mask(mask string) Network_Firewall_AccessControlList {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Firewall_AccessControlList) Filter(filter string) Network_Firewall_AccessControlList {
	r.Options.Filter = filter
	return r
}

func (r Network_Firewall_AccessControlList) Limit(limit int) Network_Firewall_AccessControlList {
	r.Options.Limit = &limit
	return r
}

func (r Network_Firewall_AccessControlList) Offset(offset int) Network_Firewall_AccessControlList {
	r.Options.Offset = &offset
	return r
}

// Retrieve The update requests made for this firewall.
func (r Network_Firewall_AccessControlList) GetNetworkFirewallUpdateRequests() (resp []datatypes.Network_Firewall_Update_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_AccessControlList", "getNetworkFirewallUpdateRequests", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Firewall_AccessControlList) GetNetworkVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_AccessControlList", "getNetworkVlan", nil, &r.Options, &resp)
	return
}

// getObject returns a SoftLayer_Network_Firewall_AccessControlList object. You can only get objects for servers attached to your account that have a network firewall enabled.
func (r Network_Firewall_AccessControlList) GetObject() (resp datatypes.Network_Firewall_AccessControlList, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_AccessControlList", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The currently running rule set of this context access control list firewall.
func (r Network_Firewall_AccessControlList) GetRules() (resp []datatypes.Network_Vlan_Firewall_Rule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_AccessControlList", "getRules", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Firewall_Interface data type contains general information relating to a single SoftLayer firewall interface. This is the object which ties the firewall context access control list to a firewall. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Firewall_Interface struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkFirewallInterfaceService returns an instance of the Network_Firewall_Interface SoftLayer service
func GetNetworkFirewallInterfaceService(sess *session.Session) Network_Firewall_Interface {
	return Network_Firewall_Interface{Session: sess}
}

func (r Network_Firewall_Interface) Id(id int) Network_Firewall_Interface {
	r.Options.Id = &id
	return r
}

func (r Network_Firewall_Interface) Mask(mask string) Network_Firewall_Interface {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Firewall_Interface) Filter(filter string) Network_Firewall_Interface {
	r.Options.Filter = filter
	return r
}

func (r Network_Firewall_Interface) Limit(limit int) Network_Firewall_Interface {
	r.Options.Limit = &limit
	return r
}

func (r Network_Firewall_Interface) Offset(offset int) Network_Firewall_Interface {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Network_Firewall_Interface) GetFirewallContextAccessControlLists() (resp []datatypes.Network_Firewall_AccessControlList, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Interface", "getFirewallContextAccessControlLists", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Firewall_Interface) GetNetworkVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Interface", "getNetworkVlan", nil, &r.Options, &resp)
	return
}

// getObject returns a SoftLayer_Network_Firewall_Interface object. You can only get objects for servers attached to your account that have a network firewall enabled.
func (r Network_Firewall_Interface) GetObject() (resp datatypes.Network_Firewall_Interface, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Interface", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Firewall_Module_Context_Interface struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkFirewallModuleContextInterfaceService returns an instance of the Network_Firewall_Module_Context_Interface SoftLayer service
func GetNetworkFirewallModuleContextInterfaceService(sess *session.Session) Network_Firewall_Module_Context_Interface {
	return Network_Firewall_Module_Context_Interface{Session: sess}
}

func (r Network_Firewall_Module_Context_Interface) Id(id int) Network_Firewall_Module_Context_Interface {
	r.Options.Id = &id
	return r
}

func (r Network_Firewall_Module_Context_Interface) Mask(mask string) Network_Firewall_Module_Context_Interface {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Firewall_Module_Context_Interface) Filter(filter string) Network_Firewall_Module_Context_Interface {
	r.Options.Filter = filter
	return r
}

func (r Network_Firewall_Module_Context_Interface) Limit(limit int) Network_Firewall_Module_Context_Interface {
	r.Options.Limit = &limit
	return r
}

func (r Network_Firewall_Module_Context_Interface) Offset(offset int) Network_Firewall_Module_Context_Interface {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Network_Firewall_Module_Context_Interface) GetFirewallContextAccessControlLists() (resp []datatypes.Network_Firewall_AccessControlList, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Module_Context_Interface", "getFirewallContextAccessControlLists", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Firewall_Module_Context_Interface) GetNetworkVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Module_Context_Interface", "getNetworkVlan", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Firewall_Module_Context_Interface) GetObject() (resp datatypes.Network_Firewall_Module_Context_Interface, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Module_Context_Interface", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Firewall_Template type contains general information for a SoftLayer network firewall template.
//
// Firewall templates are recommend rule sets for use with SoftLayer Hardware Firewall (Dedicated).  These optimized templates are designed to balance security restriction with application availability.  The templates given may be altered to provide custom network security, or may be used as-is for basic security. At least one rule set MUST be applied for the firewall to block traffic. Use the [[SoftLayer Network Component Firewall]] service to view current rules. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Firewall_Template struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkFirewallTemplateService returns an instance of the Network_Firewall_Template SoftLayer service
func GetNetworkFirewallTemplateService(sess *session.Session) Network_Firewall_Template {
	return Network_Firewall_Template{Session: sess}
}

func (r Network_Firewall_Template) Id(id int) Network_Firewall_Template {
	r.Options.Id = &id
	return r
}

func (r Network_Firewall_Template) Mask(mask string) Network_Firewall_Template {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Firewall_Template) Filter(filter string) Network_Firewall_Template {
	r.Options.Filter = filter
	return r
}

func (r Network_Firewall_Template) Limit(limit int) Network_Firewall_Template {
	r.Options.Limit = &limit
	return r
}

func (r Network_Firewall_Template) Offset(offset int) Network_Firewall_Template {
	r.Options.Offset = &offset
	return r
}

// Get all available firewall template objects.
//
// ''getAllObjects'' returns an array of SoftLayer_Network_Firewall_Template objects upon success.
func (r Network_Firewall_Template) GetAllObjects() (resp []datatypes.Network_Firewall_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Template", "getAllObjects", nil, &r.Options, &resp)
	return
}

// getObject returns a SoftLayer_Network_Firewall_Template object. You can retrieve all available firewall templates. getAllObjects returns an array of all available SoftLayer_Network_Firewall_Template objects. You can use these templates to generate a [[SoftLayer Network Firewall Update Request]].
//
// @SLDNDocumentation Service See Also SoftLayer_Network_Firewall_Update_Request
func (r Network_Firewall_Template) GetObject() (resp datatypes.Network_Firewall_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Template", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The rule set that belongs to this firewall rules template.
func (r Network_Firewall_Template) GetRules() (resp []datatypes.Network_Firewall_Template_Rule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Template", "getRules", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Firewall_Update_Request data type contains information relating to a SoftLayer network firewall update request. Use the [[SoftLayer Network Component Firewall]] service to view current rules. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates.
type Network_Firewall_Update_Request struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkFirewallUpdateRequestService returns an instance of the Network_Firewall_Update_Request SoftLayer service
func GetNetworkFirewallUpdateRequestService(sess *session.Session) Network_Firewall_Update_Request {
	return Network_Firewall_Update_Request{Session: sess}
}

func (r Network_Firewall_Update_Request) Id(id int) Network_Firewall_Update_Request {
	r.Options.Id = &id
	return r
}

func (r Network_Firewall_Update_Request) Mask(mask string) Network_Firewall_Update_Request {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Firewall_Update_Request) Filter(filter string) Network_Firewall_Update_Request {
	r.Options.Filter = filter
	return r
}

func (r Network_Firewall_Update_Request) Limit(limit int) Network_Firewall_Update_Request {
	r.Options.Limit = &limit
	return r
}

func (r Network_Firewall_Update_Request) Offset(offset int) Network_Firewall_Update_Request {
	r.Options.Offset = &offset
	return r
}

// Create a new firewall update request. The SoftLayer_Network_Firewall_Update_Request object passed to this function must have at least one rule.
//
// ''createObject'' returns a Boolean ''true'' on successful object creation or ''false'' if your firewall update request was unable to be created.
func (r Network_Firewall_Update_Request) CreateObject(templateObject *datatypes.Network_Firewall_Update_Request) (resp datatypes.Network_Firewall_Update_Request, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "createObject", params, &r.Options, &resp)
	return
}

// Retrieve The user that authorized this firewall update request.
func (r Network_Firewall_Update_Request) GetAuthorizingUser() (resp datatypes.User_Interface, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "getAuthorizingUser", nil, &r.Options, &resp)
	return
}

// Get the possible attribute values for a firewall update request rule.  These are the valid values which may be submitted as rule parameters for a firewall update request.
//
// ''getFirewallUpdateRequestRuleAttributes'' returns a SoftLayer_Container_Utility_Network_Firewall_Rule_Attribute object upon success.
func (r Network_Firewall_Update_Request) GetFirewallUpdateRequestRuleAttributes() (resp datatypes.Container_Utility_Network_Firewall_Rule_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "getFirewallUpdateRequestRuleAttributes", nil, &r.Options, &resp)
	return
}

// Retrieve The downstream virtual server that the rule set will be applied to.
func (r Network_Firewall_Update_Request) GetGuest() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "getGuest", nil, &r.Options, &resp)
	return
}

// Retrieve The downstream server that the rule set will be applied to.
func (r Network_Firewall_Update_Request) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The network component firewall that the rule set will be applied to.
func (r Network_Firewall_Update_Request) GetNetworkComponentFirewall() (resp datatypes.Network_Component_Firewall, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "getNetworkComponentFirewall", nil, &r.Options, &resp)
	return
}

// ''getObject'' returns a SoftLayer_Network_Firewall_Update_Request object. You can only get historical objects for servers attached to your account that have a network firewall enabled. ''createObject'' inserts a new SoftLayer_Network_Firewall_Update_Request object. You can only insert requests for servers attached to your account that have a network firewall enabled. ''getFirewallUpdateRequestRuleAttributes'' Get the possible attribute values for a firewall update request rule.
func (r Network_Firewall_Update_Request) GetObject() (resp datatypes.Network_Firewall_Update_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The group of rules contained within the update request.
func (r Network_Firewall_Update_Request) GetRules() (resp []datatypes.Network_Firewall_Update_Request_Rule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "getRules", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Firewall_Update_Request) UpdateRuleNote(fwRule *datatypes.Network_Component_Firewall_Rule, note *string) (resp bool, err error) {
	params := []interface{}{
		fwRule,
		note,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request", "updateRuleNote", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Firewall_Update_Request_Rule type contains information relating to a SoftLayer network firewall update request rule. This rule is a member of a [[SoftLayer Network Firewall Update Request]]. Use the [[SoftLayer Network Component Firewall]] service to view current rules. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates.
type Network_Firewall_Update_Request_Rule struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkFirewallUpdateRequestRuleService returns an instance of the Network_Firewall_Update_Request_Rule SoftLayer service
func GetNetworkFirewallUpdateRequestRuleService(sess *session.Session) Network_Firewall_Update_Request_Rule {
	return Network_Firewall_Update_Request_Rule{Session: sess}
}

func (r Network_Firewall_Update_Request_Rule) Id(id int) Network_Firewall_Update_Request_Rule {
	r.Options.Id = &id
	return r
}

func (r Network_Firewall_Update_Request_Rule) Mask(mask string) Network_Firewall_Update_Request_Rule {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Firewall_Update_Request_Rule) Filter(filter string) Network_Firewall_Update_Request_Rule {
	r.Options.Filter = filter
	return r
}

func (r Network_Firewall_Update_Request_Rule) Limit(limit int) Network_Firewall_Update_Request_Rule {
	r.Options.Limit = &limit
	return r
}

func (r Network_Firewall_Update_Request_Rule) Offset(offset int) Network_Firewall_Update_Request_Rule {
	r.Options.Offset = &offset
	return r
}

// Create a new firewall update request. The SoftLayer_Network_Firewall_Update_Request object passed to this function must have at least one rule.
//
// ''createObject'' returns a Boolean ''true'' on successful object creation or ''false'' if your firewall update request was unable to be created..
func (r Network_Firewall_Update_Request_Rule) CreateObject(templateObject *datatypes.Network_Firewall_Update_Request_Rule) (resp datatypes.Network_Firewall_Update_Request_Rule, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request_Rule", "createObject", params, &r.Options, &resp)
	return
}

// Retrieve The update request that this rule belongs to.
func (r Network_Firewall_Update_Request_Rule) GetFirewallUpdateRequest() (resp datatypes.Network_Firewall_Update_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request_Rule", "getFirewallUpdateRequest", nil, &r.Options, &resp)
	return
}

// getObject returns a SoftLayer_Network_Firewall_Update_Request_Rule object. You can only get historical objects for servers attached to your account that have a network firewall enabled. createObject inserts a new SoftLayer_Network_Firewall_Update_Request_Rule object. Use the SoftLayer_Network_Firewall_Update_Request to create groups of rules for an update request.
func (r Network_Firewall_Update_Request_Rule) GetObject() (resp datatypes.Network_Firewall_Update_Request_Rule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request_Rule", "getObject", nil, &r.Options, &resp)
	return
}

// Validate the supplied firewall request rule against the object it will apply to. For IPv4 rules, pass in an instance of SoftLayer_Network_Firewall_Update_Request_Rule. for IPv6 rules, pass in an instance of SoftLayer_Network_Firewall_Update_Request_Rule_Version6. The ID of the applied to object can either be applyToComponentId (an ID of a SoftLayer_Network_Component_Firewall) or applyToAclId (an ID of a SoftLayer_Network_Firewall_Module_Context_Interface_AccessControlList). One, and only one, of applyToComponentId and applyToAclId can be specified.
//
// If validation is successful, nothing is returned. If validation is unsuccessful, an exception is thrown explaining the nature of the validation error.
func (r Network_Firewall_Update_Request_Rule) ValidateRule(rule *datatypes.Network_Firewall_Update_Request_Rule, applyToComponentId *int, applyToAclId *int) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		rule,
		applyToComponentId,
		applyToAclId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Firewall_Update_Request_Rule", "validateRule", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Gateway struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkGatewayService returns an instance of the Network_Gateway SoftLayer service
func GetNetworkGatewayService(sess *session.Session) Network_Gateway {
	return Network_Gateway{Session: sess}
}

func (r Network_Gateway) Id(id int) Network_Gateway {
	r.Options.Id = &id
	return r
}

func (r Network_Gateway) Mask(mask string) Network_Gateway {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Gateway) Filter(filter string) Network_Gateway {
	r.Options.Filter = filter
	return r
}

func (r Network_Gateway) Limit(limit int) Network_Gateway {
	r.Options.Limit = &limit
	return r
}

func (r Network_Gateway) Offset(offset int) Network_Gateway {
	r.Options.Offset = &offset
	return r
}

// Start the asynchronous process to bypass all VLANs. Any VLANs that are already bypassed will be ignored. The status field can be checked for progress.
func (r Network_Gateway) BypassAllVlans() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "bypassAllVlans", nil, &r.Options, &resp)
	return
}

// Start the asynchronous process to bypass the provided VLANs. The VLANs must already be attached. Any VLANs that are already bypassed will be ignored. The status field can be checked for progress.
func (r Network_Gateway) BypassVlans(vlans []datatypes.Network_Gateway_Vlan) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		vlans,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "bypassVlans", params, &r.Options, &resp)
	return
}

// Create and return a new gateway. This object can be created with any number of members or VLANs, but they all must be in the same pod. By creating a gateway with members and/or VLANs attached, it is the equivalent of individually calling their createObject methods except this will start a single asynchronous process to setup the gateway. The status of this process can be checked using the status field.
func (r Network_Gateway) CreateObject(templateObject *datatypes.Network_Gateway) (resp datatypes.Network_Gateway, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "createObject", params, &r.Options, &resp)
	return
}

// Edit this gateway. Currently, the only value that can be edited is the name.
func (r Network_Gateway) EditObject(templateObject *datatypes.Network_Gateway) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account for this gateway.
func (r Network_Gateway) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve All VLANs trunked to this gateway.
func (r Network_Gateway) GetInsideVlans() (resp []datatypes.Network_Gateway_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getInsideVlans", nil, &r.Options, &resp)
	return
}

// Retrieve The members for this gateway.
func (r Network_Gateway) GetMembers() (resp []datatypes.Network_Gateway_Member, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getMembers", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Gateway) GetObject() (resp datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getObject", nil, &r.Options, &resp)
	return
}

// Get all VLANs that can become inside VLANs on this gateway. This means the VLAN must not already be an inside VLAN, on the same router as this gateway, not a gateway transit VLAN, and not firewalled.
func (r Network_Gateway) GetPossibleInsideVlans() (resp []datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getPossibleInsideVlans", nil, &r.Options, &resp)
	return
}

// Retrieve The private gateway IP address.
func (r Network_Gateway) GetPrivateIpAddress() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getPrivateIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The private VLAN for accessing this gateway.
func (r Network_Gateway) GetPrivateVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getPrivateVlan", nil, &r.Options, &resp)
	return
}

// Retrieve The public gateway IP address.
func (r Network_Gateway) GetPublicIpAddress() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getPublicIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The public gateway IPv6 address.
func (r Network_Gateway) GetPublicIpv6Address() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getPublicIpv6Address", nil, &r.Options, &resp)
	return
}

// Retrieve The public VLAN for accessing this gateway.
func (r Network_Gateway) GetPublicVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getPublicVlan", nil, &r.Options, &resp)
	return
}

// Retrieve The current status of the gateway.
func (r Network_Gateway) GetStatus() (resp datatypes.Network_Gateway_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "getStatus", nil, &r.Options, &resp)
	return
}

// Start the asynchronous process to unbypass all VLANs. Any VLANs that are already unbypassed will be ignored. The status field can be checked for progress.
func (r Network_Gateway) UnbypassAllVlans() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "unbypassAllVlans", nil, &r.Options, &resp)
	return
}

// Start the asynchronous process to unbypass the provided VLANs. The VLANs must already be attached. Any VLANs that are already unbypassed will be ignored. The status field can be checked for progress.
func (r Network_Gateway) UnbypassVlans(vlans []datatypes.Network_Gateway_Vlan) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		vlans,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway", "unbypassVlans", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Gateway_Member struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkGatewayMemberService returns an instance of the Network_Gateway_Member SoftLayer service
func GetNetworkGatewayMemberService(sess *session.Session) Network_Gateway_Member {
	return Network_Gateway_Member{Session: sess}
}

func (r Network_Gateway_Member) Id(id int) Network_Gateway_Member {
	r.Options.Id = &id
	return r
}

func (r Network_Gateway_Member) Mask(mask string) Network_Gateway_Member {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Gateway_Member) Filter(filter string) Network_Gateway_Member {
	r.Options.Filter = filter
	return r
}

func (r Network_Gateway_Member) Limit(limit int) Network_Gateway_Member {
	r.Options.Limit = &limit
	return r
}

func (r Network_Gateway_Member) Offset(offset int) Network_Gateway_Member {
	r.Options.Offset = &offset
	return r
}

// Create a new hardware member on the gateway. This also asynchronously sets up the network for this member. Progress of this process can be monitored via the gateway status. All members created with this object must have no VLANs attached.
func (r Network_Gateway_Member) CreateObject(templateObject *datatypes.Network_Gateway_Member) (resp datatypes.Network_Gateway_Member, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Member", "createObject", params, &r.Options, &resp)
	return
}

// Create multiple new hardware members on the gateway. This also asynchronously sets up the network for the members. Progress of this process can be monitored via the gateway status. All members created with this object must have no VLANs attached.
func (r Network_Gateway_Member) CreateObjects(templateObjects []datatypes.Network_Gateway_Member) (resp []datatypes.Network_Gateway_Member, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Member", "createObjects", params, &r.Options, &resp)
	return
}

// Retrieve The device for this member.
func (r Network_Gateway_Member) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Member", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The gateway this member belongs to.
func (r Network_Gateway_Member) GetNetworkGateway() (resp datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Member", "getNetworkGateway", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Gateway_Member) GetObject() (resp datatypes.Network_Gateway_Member, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Member", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Gateway_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkGatewayStatusService returns an instance of the Network_Gateway_Status SoftLayer service
func GetNetworkGatewayStatusService(sess *session.Session) Network_Gateway_Status {
	return Network_Gateway_Status{Session: sess}
}

func (r Network_Gateway_Status) Id(id int) Network_Gateway_Status {
	r.Options.Id = &id
	return r
}

func (r Network_Gateway_Status) Mask(mask string) Network_Gateway_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Gateway_Status) Filter(filter string) Network_Gateway_Status {
	r.Options.Filter = filter
	return r
}

func (r Network_Gateway_Status) Limit(limit int) Network_Gateway_Status {
	r.Options.Limit = &limit
	return r
}

func (r Network_Gateway_Status) Offset(offset int) Network_Gateway_Status {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Gateway_Status) GetObject() (resp datatypes.Network_Gateway_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Status", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Gateway_Vlan struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkGatewayVlanService returns an instance of the Network_Gateway_Vlan SoftLayer service
func GetNetworkGatewayVlanService(sess *session.Session) Network_Gateway_Vlan {
	return Network_Gateway_Vlan{Session: sess}
}

func (r Network_Gateway_Vlan) Id(id int) Network_Gateway_Vlan {
	r.Options.Id = &id
	return r
}

func (r Network_Gateway_Vlan) Mask(mask string) Network_Gateway_Vlan {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Gateway_Vlan) Filter(filter string) Network_Gateway_Vlan {
	r.Options.Filter = filter
	return r
}

func (r Network_Gateway_Vlan) Limit(limit int) Network_Gateway_Vlan {
	r.Options.Limit = &limit
	return r
}

func (r Network_Gateway_Vlan) Offset(offset int) Network_Gateway_Vlan {
	r.Options.Offset = &offset
	return r
}

// Start the asynchronous process to bypass/unroute the VLAN from this gateway.
func (r Network_Gateway_Vlan) Bypass() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "bypass", nil, &r.Options, &resp)
	return
}

// Create a new VLAN attachment. If the bypassFlag is false, this will also create an asynchronous process to route the VLAN through the gateway.
func (r Network_Gateway_Vlan) CreateObject(templateObject *datatypes.Network_Gateway_Vlan) (resp datatypes.Network_Gateway_Vlan, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "createObject", params, &r.Options, &resp)
	return
}

// Create multiple new VLAN attachments. If the bypassFlag is false, this will also create an asynchronous process to route the VLANs through the gateway.
func (r Network_Gateway_Vlan) CreateObjects(templateObjects []datatypes.Network_Gateway_Vlan) (resp []datatypes.Network_Gateway_Vlan, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "createObjects", params, &r.Options, &resp)
	return
}

// Start the asynchronous process to detach this VLANs from the gateway.
func (r Network_Gateway_Vlan) DeleteObject() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "deleteObject", nil, &r.Options, &resp)
	return
}

// Detach several VLANs. This will not detach them right away, but rather start an asynchronous process to detach.
func (r Network_Gateway_Vlan) DeleteObjects(templateObjects []datatypes.Network_Gateway_Vlan) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "deleteObjects", params, &r.Options, &resp)
	return
}

// Retrieve The gateway this VLAN is attached to.
func (r Network_Gateway_Vlan) GetNetworkGateway() (resp datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "getNetworkGateway", nil, &r.Options, &resp)
	return
}

// Retrieve The network VLAN record.
func (r Network_Gateway_Vlan) GetNetworkVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "getNetworkVlan", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Gateway_Vlan) GetObject() (resp datatypes.Network_Gateway_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "getObject", nil, &r.Options, &resp)
	return
}

// Start the asynchronous process to route the VLAN to this gateway.
func (r Network_Gateway_Vlan) Unbypass() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Network_Gateway_Vlan", "unbypass", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_LoadBalancer_Global_Account data type contains the properties for a single global load balancer account.  The properties you are able to edit are fallbackIp, loadBalanceTypeId, and notes. The hosts relational property can be used for creating and editing hosts that belong to the global load balancer account.  The [[SoftLayer_Network_LoadBalancer_Global_Account::editObject|editObject]] method contains details on creating and edited hosts through the hosts relational property.
type Network_LoadBalancer_Global_Account struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkLoadBalancerGlobalAccountService returns an instance of the Network_LoadBalancer_Global_Account SoftLayer service
func GetNetworkLoadBalancerGlobalAccountService(sess *session.Session) Network_LoadBalancer_Global_Account {
	return Network_LoadBalancer_Global_Account{Session: sess}
}

func (r Network_LoadBalancer_Global_Account) Id(id int) Network_LoadBalancer_Global_Account {
	r.Options.Id = &id
	return r
}

func (r Network_LoadBalancer_Global_Account) Mask(mask string) Network_LoadBalancer_Global_Account {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_LoadBalancer_Global_Account) Filter(filter string) Network_LoadBalancer_Global_Account {
	r.Options.Filter = filter
	return r
}

func (r Network_LoadBalancer_Global_Account) Limit(limit int) Network_LoadBalancer_Global_Account {
	r.Options.Limit = &limit
	return r
}

func (r Network_LoadBalancer_Global_Account) Offset(offset int) Network_LoadBalancer_Global_Account {
	r.Options.Offset = &offset
	return r
}

// If your globally load balanced domain is hosted on the SoftLayer nameservers this method will add the required NS resource record to your DNS zone file and remove any A records that match the host portion of a global load balancer account hostname.  A NS resource record is required to be able to use your SoftLayer global load balancer account. Please make sure the zone file for the hostname listed on your SoftLayer global load balancer account is setup prior to using this method.  If your globally load balanced domain is hosted on any other nameservers this method will not be able to add the required NS record.
func (r Network_LoadBalancer_Global_Account) AddNsRecord() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "addNsRecord", nil, &r.Options, &resp)
	return
}

// Edit the properties of a global load balancer account by passing in a modified instance of the object. The global load balancer account properties you are able to edit are: fallback ip, load balance type id, and notes. Hosts that belong to your SoftLayer global load balancer account are created and modified through this method. An example templateObject that updates global load balancer account properties, updates the properties of a host, and adds a new host is shown below:
//
//
// * id: 2
// * loadBalanceTypeId: 2
// * notes: Notes updated
// * fallbackIp: 1.1.1.1
// * hosts:
// ** id: 19
// ** destinationIp: 2.2.2.2
// ** weight: 25
// ** healthCheck: http
// ** destinationPort: 80
// ** enabled: 1<br /><br />
// ** destinationIp: 3.3.3.3
// ** weight: 25
// ** healthCheck: http
// ** destinationPort: 80
// ** enabled: 1
//
//
//
//
// The first section contains the properties of the global load balancer account that will be updated, while the second section contains the elements of the 'hosts' property of the global load balancer account.  The first host listed will have its properties updated because the 'id' property of the host is set, meaning the global load balancer host with an id of 19 will be updated. The second host listed will be created because it lacks the 'id' property.
//
// There is a limit to the maximum number of hosts that you are allowed to add, and is defined by the allowedNumberOfHosts property on the global load balancer account.  The destination IP address of a host must be an IP address that belongs to your SoftLayer Account, or a local load balancer virtual IP address that belongs to your account.  The destination IP address and destination port are required and must be provided when creating a host.
func (r Network_LoadBalancer_Global_Account) EditObject(templateObject *datatypes.Network_LoadBalancer_Global_Account) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve Your SoftLayer customer account.
func (r Network_LoadBalancer_Global_Account) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for a Global Load Balancer account.
func (r Network_LoadBalancer_Global_Account) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The hosts in the load balancing pool for a global load balancer account.
func (r Network_LoadBalancer_Global_Account) GetHosts() (resp []datatypes.Network_LoadBalancer_Global_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "getHosts", nil, &r.Options, &resp)
	return
}

// Retrieve The load balance method of a global load balancer account
func (r Network_LoadBalancer_Global_Account) GetLoadBalanceType() (resp datatypes.Network_LoadBalancer_Global_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "getLoadBalanceType", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating that the global load balancer is a managed resource.
func (r Network_LoadBalancer_Global_Account) GetManagedResourceFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "getManagedResourceFlag", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_LoadBalancer_Global_Account object whose ID number corresponds to the ID number of the init paramater passed to the SoftLayer_Network_LoadBalancer_Global_Account service. You can only retrieve a global load balancer account that is assigned to your SoftLayer customer account.
func (r Network_LoadBalancer_Global_Account) GetObject() (resp datatypes.Network_LoadBalancer_Global_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "getObject", nil, &r.Options, &resp)
	return
}

// If your globally load balanced domain is hosted on the SoftLayer nameservers this method will remove the NS resource record from your DNS zone file. Removing the NS resource record will basically disable your global load balancer account since no DNS requests will be forwarded to the global load balancers. Any A records that were removed when the NS resource record was added will not be created for you.  If your globally load balanced domain is hosted on any other nameservers this method will not be able to remove the required NS record.
func (r Network_LoadBalancer_Global_Account) RemoveNsRecord() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Account", "removeNsRecord", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_LoadBalancer_Global_Host data type represents a single host that belongs to a global load balancer account's load balancing pool.
//
// The destination IP address of a host must be one that belongs to your SoftLayer customer account, or to a datacenter load balancer virtual ip that belongs to your SoftLayer customer account.  The destination IP address and port of a global load balancer host is a required field and must exist during creation and can not be removed.  The acceptable values for the health check type are 'none', 'http', and 'tcp'. The status property is updated in 5 minute intervals and the hits property is updated in 10 minute intervals.
//
// The order of the host is only important if you are using the 'failover' load balance method, and the weight is only important if you are using the 'weighted round robin' load balance method.
type Network_LoadBalancer_Global_Host struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkLoadBalancerGlobalHostService returns an instance of the Network_LoadBalancer_Global_Host SoftLayer service
func GetNetworkLoadBalancerGlobalHostService(sess *session.Session) Network_LoadBalancer_Global_Host {
	return Network_LoadBalancer_Global_Host{Session: sess}
}

func (r Network_LoadBalancer_Global_Host) Id(id int) Network_LoadBalancer_Global_Host {
	r.Options.Id = &id
	return r
}

func (r Network_LoadBalancer_Global_Host) Mask(mask string) Network_LoadBalancer_Global_Host {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_LoadBalancer_Global_Host) Filter(filter string) Network_LoadBalancer_Global_Host {
	r.Options.Filter = filter
	return r
}

func (r Network_LoadBalancer_Global_Host) Limit(limit int) Network_LoadBalancer_Global_Host {
	r.Options.Limit = &limit
	return r
}

func (r Network_LoadBalancer_Global_Host) Offset(offset int) Network_LoadBalancer_Global_Host {
	r.Options.Offset = &offset
	return r
}

// Remove a host from the load balancing pool of a global load balancer account.
func (r Network_LoadBalancer_Global_Host) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Host", "deleteObject", nil, &r.Options, &resp)
	return
}

// Retrieve The global load balancer account a host belongs to.
func (r Network_LoadBalancer_Global_Host) GetLoadBalancerAccount() (resp datatypes.Network_LoadBalancer_Global_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Host", "getLoadBalancerAccount", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_LoadBalancer_Global_Host object whose ID number corresponds to the ID number of the init paramater passed to the SoftLayer_Network_LoadBalancer_Global_Host service. You can only retrieve a global load balancer host that is assigned to your SoftLayer global load balancer account.
func (r Network_LoadBalancer_Global_Host) GetObject() (resp datatypes.Network_LoadBalancer_Global_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Global_Host", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_LoadBalancer_Service data type contains all the information relating to a specific service (destination) on a particular load balancer.
//
// Information retained on the object itself is the the source and destination of the service, routing type, weight, and whether or not the service is currently enabled.
type Network_LoadBalancer_Service struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkLoadBalancerServiceService returns an instance of the Network_LoadBalancer_Service SoftLayer service
func GetNetworkLoadBalancerServiceService(sess *session.Session) Network_LoadBalancer_Service {
	return Network_LoadBalancer_Service{Session: sess}
}

func (r Network_LoadBalancer_Service) Id(id int) Network_LoadBalancer_Service {
	r.Options.Id = &id
	return r
}

func (r Network_LoadBalancer_Service) Mask(mask string) Network_LoadBalancer_Service {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_LoadBalancer_Service) Filter(filter string) Network_LoadBalancer_Service {
	r.Options.Filter = filter
	return r
}

func (r Network_LoadBalancer_Service) Limit(limit int) Network_LoadBalancer_Service {
	r.Options.Limit = &limit
	return r
}

func (r Network_LoadBalancer_Service) Offset(offset int) Network_LoadBalancer_Service {
	r.Options.Offset = &offset
	return r
}

// Calling deleteObject on a particular server will remove it from the load balancer.  This is the only way to remove a service from your load balancer.  If you wish to remove a server, first call this function, then reload the virtualIpAddress object and edit the remaining services to reflect the other changes that you wish to make.
func (r Network_LoadBalancer_Service) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Service", "deleteObject", nil, &r.Options, &resp)
	return
}

// Get the graph image for a load balancer service based on the supplied graph type and metric.  The available graph types are: 'connections' and 'status', and the available metrics are: 'day', 'week' and 'month'.
//
// This method returns the raw binary image data.
func (r Network_LoadBalancer_Service) GetGraphImage(graphType *string, metric *string) (resp []byte, err error) {
	params := []interface{}{
		graphType,
		metric,
	}
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Service", "getGraphImage", params, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_LoadBalancer_Service object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_LoadBalancer_Service service. You can only retrieve services on load balancers assigned to your account, and it is recommended that you simply retrieve the entire load balancer, as an individual service has no explicit purpose without its "siblings".
func (r Network_LoadBalancer_Service) GetObject() (resp datatypes.Network_LoadBalancer_Service, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Service", "getObject", nil, &r.Options, &resp)
	return
}

// Returns an array of SoftLayer_Container_Network_LoadBalancer_StatusEntry objects.  A SoftLayer_Container_Network_LoadBalancer_StatusEntry object has two variables, "Label" and "Value"
//
// Calling this function executes a command on the physical load balancer itself, and therefore should be called infrequently.  For a general idea of the load balancer service, use the "peakConnections" variable on the Type
//
// Possible values for "Label" are:
//
//
// * IP Address
// * Port
// * Server Status
// * Load Status
// * Current Connections
// * Total Hits
//
//
// Not all labels are guaranteed to be returned.
func (r Network_LoadBalancer_Service) GetStatus() (resp []datatypes.Container_Network_LoadBalancer_StatusEntry, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Service", "getStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The load balancer that this service belongs to.
func (r Network_LoadBalancer_Service) GetVip() (resp datatypes.Network_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Service", "getVip", nil, &r.Options, &resp)
	return
}

// Calling resetPeakConnections will set the peakConnections variable to zero on this particular object. Peak connections will continue to increase normally after this method call, it will only temporarily reset the statistic to zero, until the next time it is polled.
func (r Network_LoadBalancer_Service) ResetPeakConnections() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_Service", "resetPeakConnections", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_LoadBalancer_VirtualIpAddress data type contains all the information relating to a specific load balancer assigned to a customer account.
//
// Information retained on the object itself is the virtual IP address, load balancing method, and any notes that are related to the load balancer.  There is also an array of SoftLayer_Network_LoadBalancer_Service objects, which represent the load balancer services, explained more fully in the SoftLayer_Network_LoadBalancer_Service documentation.
type Network_LoadBalancer_VirtualIpAddress struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkLoadBalancerVirtualIpAddressService returns an instance of the Network_LoadBalancer_VirtualIpAddress SoftLayer service
func GetNetworkLoadBalancerVirtualIpAddressService(sess *session.Session) Network_LoadBalancer_VirtualIpAddress {
	return Network_LoadBalancer_VirtualIpAddress{Session: sess}
}

func (r Network_LoadBalancer_VirtualIpAddress) Id(id int) Network_LoadBalancer_VirtualIpAddress {
	r.Options.Id = &id
	return r
}

func (r Network_LoadBalancer_VirtualIpAddress) Mask(mask string) Network_LoadBalancer_VirtualIpAddress {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_LoadBalancer_VirtualIpAddress) Filter(filter string) Network_LoadBalancer_VirtualIpAddress {
	r.Options.Filter = filter
	return r
}

func (r Network_LoadBalancer_VirtualIpAddress) Limit(limit int) Network_LoadBalancer_VirtualIpAddress {
	r.Options.Limit = &limit
	return r
}

func (r Network_LoadBalancer_VirtualIpAddress) Offset(offset int) Network_LoadBalancer_VirtualIpAddress {
	r.Options.Offset = &offset
	return r
}

// Disable a Virtual IP Address, removing it from load balancer rotation and denying all connections to that IP address.
func (r Network_LoadBalancer_VirtualIpAddress) Disable() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "disable", nil, &r.Options, &resp)
	return
}

// Like any other API object, the load balancers can have their exposed properties edited by passing in a modified version of the object.  The load balancer object also can modify its services in this way.  Simply request the load balancer object you wish to edit, then modify the objects in the services array and pass the modified object to this function.  WARNING:  Services cannot be deleted in this manner, you must call deleteObject() on the service to physically remove them from the load balancer.
func (r Network_LoadBalancer_VirtualIpAddress) EditObject(templateObject *datatypes.Network_LoadBalancer_VirtualIpAddress) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "editObject", params, &r.Options, &resp)
	return
}

// Enable a disabled Virtual IP Address, allowing connections back to the IP address.
func (r Network_LoadBalancer_VirtualIpAddress) Enable() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "enable", nil, &r.Options, &resp)
	return
}

// Retrieve The account that owns this load balancer.
func (r Network_LoadBalancer_VirtualIpAddress) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for the Load Balancer.
func (r Network_LoadBalancer_VirtualIpAddress) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve If false, this VIP and associated services may be edited via the portal or the API. If true, you must configure this VIP manually on the device.
func (r Network_LoadBalancer_VirtualIpAddress) GetCustomerManagedFlag() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "getCustomerManagedFlag", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating that the load balancer is a managed resource.
func (r Network_LoadBalancer_VirtualIpAddress) GetManagedResourceFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "getManagedResourceFlag", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_LoadBalancer_VirtualIpAddress object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_LoadBalancer_VirtualIpAddress service. You can only retrieve Load Balancers assigned to your account.
func (r Network_LoadBalancer_VirtualIpAddress) GetObject() (resp datatypes.Network_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve the services on this load balancer.
func (r Network_LoadBalancer_VirtualIpAddress) GetServices() (resp []datatypes.Network_LoadBalancer_Service, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "getServices", nil, &r.Options, &resp)
	return
}

// Quickly remove all active external connections to a Virtual IP Address.
func (r Network_LoadBalancer_VirtualIpAddress) KickAllConnections() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "kickAllConnections", nil, &r.Options, &resp)
	return
}

// Upgrades the connection limit on the VirtualIp and changes the billing item on your account to reflect the change. This function will only upgrade you to the next "level" of service.  The next level follows this pattern Current Level  =>  Next Level 50                 100 100                200 200                500 500                1000 1000               1200 1200               1500 1500               2000 2000               2500 2500               3000
func (r Network_LoadBalancer_VirtualIpAddress) UpgradeConnectionLimit() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_LoadBalancer_VirtualIpAddress", "upgradeConnectionLimit", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Media_Transcode_Account contains information regarding a transcode account.
type Network_Media_Transcode_Account struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMediaTranscodeAccountService returns an instance of the Network_Media_Transcode_Account SoftLayer service
func GetNetworkMediaTranscodeAccountService(sess *session.Session) Network_Media_Transcode_Account {
	return Network_Media_Transcode_Account{Session: sess}
}

func (r Network_Media_Transcode_Account) Id(id int) Network_Media_Transcode_Account {
	r.Options.Id = &id
	return r
}

func (r Network_Media_Transcode_Account) Mask(mask string) Network_Media_Transcode_Account {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Media_Transcode_Account) Filter(filter string) Network_Media_Transcode_Account {
	r.Options.Filter = filter
	return r
}

func (r Network_Media_Transcode_Account) Limit(limit int) Network_Media_Transcode_Account {
	r.Options.Limit = &limit
	return r
}

func (r Network_Media_Transcode_Account) Offset(offset int) Network_Media_Transcode_Account {
	r.Options.Offset = &offset
	return r
}

// With this method, you can create a transcode account.  Individual SoftLayer account can have a single Transcode account. You have to pass your SoftLayer account id as a parameter.
func (r Network_Media_Transcode_Account) CreateTranscodeAccount() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "createTranscodeAccount", nil, &r.Options, &resp)
	return
}

// '''Note'''. This method is obsolete. Please use the [[SoftLayer_Network_Media_Transcode_Job::createObject|createObject]] method on SoftLayer_Network_Media_Transcode_Job object instead. SoftLayer_Network_Media_Transcode_Job::createObject returns an object of a newly created Transcode Job.
//
// With this method, you can create a transcode job.
//
// The very first step of creating a transcode job is to upload your media files to the /in directory on your Transcode FTP space. Then, you have to pass a [[SoftLayer_Network_Media_Transcode_Job|Transcode job]] object as a parameter for this method.
//
// There are 4 required properties of SoftLayer_Network_Media_Transcode_Job object: transcodePresetName, transcodePresetGuid, inputFile, and outputFile. A transcode preset is a configuration that defines a certain media output.  You can retrieve all the supported presets with the [[SoftLayer_Network_Media_Transcode_Account::getPresets|getPresets]] method. You can also use [[SoftLayer_Network_Media_Transcode_Account::getPresetDetail|getPresetDetail]] method to get more information on a preset. Use these two methods to determine appropriate values for "transcodePresetName" and "transcodePresetGuid" properties. For an "inputFile", you must specify a file that exists in the /in directory of your Transcode FTP space. An "outputFile" name will be used by the Transcode server for naming a transcoded file.  An output file name must be in /out directory. If your outputFile name already exists in the /out directory, the Transcode server will append a file name with _n (an underscore and the total number of files with the identical name plus 1).
//
// The "name" property is optional and it can help you keep track of transcode jobs easily. "autoDeleteDuration" is another optional property that you can specify.  It determines how soon your input file will be deleted. If autoDeleteDuration is set to zero, your input file will be removed immediately after the last transcode job running on it is completed. A value for autoDeleteDuration property is in seconds and the maximum value is 259200 which is 3 days.
//
// An example SoftLayer_Network_Media_Transcode_Job parameter looks like this:
//
//
// * name: My transcoding
// * transcodePresetName: F4V 896kbps 640x352 16x9 29.97fps
// * transcodePresetGuid: {87E01268-C3E3-4A85-9701-052C9AC42BD4}
// * inputFile: /in/my_birthday.wmv
// * outputFile: /out/my_birthday_flash
//
//
// Notice that an output file does not have a file extension.  The Transcode server will append a file extension based on an output format. A newly created transcode job will be in "Pending" status and it will be added to the Transcoding queue. You will receive a notification email whenever there is a status change on your transcode job.  For example, the Transcode server starts to process your transcode job, you will be notified via an email.
//
// You can add up to 3 pending jobs at a time. Transcode jobs with any other status such as "Complete" or "Error" will not be counted toward your pending jobs.
//
// Once a job is complete, the Transcode server will place the output file into the /out directory along with a notification email. The files in the /out directory will be removed 3 days after they were created.  You will need to use an FTP client to download transcoded files.
//
//
func (r Network_Media_Transcode_Account) CreateTranscodeJob(newJob *datatypes.Network_Media_Transcode_Job) (resp bool, err error) {
	params := []interface{}{
		newJob,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "createTranscodeJob", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer account information
func (r Network_Media_Transcode_Account) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "getAccount", nil, &r.Options, &resp)
	return
}

// This method returns a collection of SoftLayer_Container_Network_Ftp_Directory objects. You can retrieve directory information for /in and /out directories. A [[SoftLayer_Container_Network_Directory_Listing|Directory Listing]] object contains a type (indicating whether it is a file or a directory), name and file count if it is a directory.
func (r Network_Media_Transcode_Account) GetDirectoryInformation(directoryName *string, extensionFilter *string) (resp []datatypes.Container_Network_Directory_Listing, err error) {
	params := []interface{}{
		directoryName,
		extensionFilter,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "getDirectoryInformation", params, &r.Options, &resp)
	return
}

// This method returns detailed information of a media file that resides in the Transcode FTP server. A [[SoftLayer_Container_Network_Media_Information|media information]] object contains media details such as file size, media format, frame rate, aspect ratio and so on.  This information is merely for reference purposes. You should not rely on this data. Our library grabs small pieces of data from a media file to gather media details.  This information may not be available for some files.
func (r Network_Media_Transcode_Account) GetFileDetail(source *string) (resp datatypes.Container_Network_Media_Information, err error) {
	params := []interface{}{
		source,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "getFileDetail", params, &r.Options, &resp)
	return
}

// This method returns your Transcode FTP login credentials to the transcode.service.softlayer.com server.
//
// The Transcode FTP server is available via the SoftLayer private network. There is no API method that you can upload a file to Transcode server so you need to use an FTP client. You will have /in and /out directories on the Transcode FTP server.  You will have read-write privileges for /in directory and read-only privilege for /out directory. All the files in both /in and /out directories will be deleted after 72 hours from the creation date.
func (r Network_Media_Transcode_Account) GetFtpAttributes() (resp datatypes.Container_Network_Authentication_Data, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "getFtpAttributes", nil, &r.Options, &resp)
	return
}

// getObject method retrieves the SoftLayer_Network_Media_Transcode_Account object whose ID number corresponds to the ID number of the initial parameter passed to the SoftLayer_Network_Media_Transcode_Account service. You can only retrieve a Transcode account assigned to your SoftLayer customer account.
func (r Network_Media_Transcode_Account) GetObject() (resp datatypes.Network_Media_Transcode_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "getObject", nil, &r.Options, &resp)
	return
}

// This method returns an array of [[SoftLayer_Container_Network_Media_Transcode_Preset_Element|preset element]] objects. Each preset has its own collection of preset elements such as encoder, frame rate, aspect ratio and so on. Each element object has a default value for itself and an array of [[SoftLayer_Container_Network_Media_Transcode_Preset_Element_Option|element option]] objects. For example, "Frame Rate" element for "Windows Media 9 - Download - 1 Mbps - NTSC - Constrained VBR" preset has 19 element options. 15.0 frame rate is selected by default.  Currently, you are not able to change the default value. Customizing these values may be possible in the future.
func (r Network_Media_Transcode_Account) GetPresetDetail(guid *string) (resp []datatypes.Container_Network_Media_Transcode_Preset_Element, err error) {
	params := []interface{}{
		guid,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "getPresetDetail", params, &r.Options, &resp)
	return
}

// A transcode preset is a configuration that defines a certain media output. This method returns an array of transcoding preset objects supported by SoftLayer's Transcode server. Each [[SoftLayer_Container_Network_Media_Transcode_Preset|preset object]] contains a GUID property. You will need a GUID string when you create a new transcode job.
func (r Network_Media_Transcode_Account) GetPresets() (resp []datatypes.Container_Network_Media_Transcode_Preset, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "getPresets", nil, &r.Options, &resp)
	return
}

// Retrieve Transcode jobs
func (r Network_Media_Transcode_Account) GetTranscodeJobs() (resp []datatypes.Network_Media_Transcode_Job, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Account", "getTranscodeJobs", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Media_Transcode_Job contains information regarding a transcode job such as input file, output format, user id and so on.
type Network_Media_Transcode_Job struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMediaTranscodeJobService returns an instance of the Network_Media_Transcode_Job SoftLayer service
func GetNetworkMediaTranscodeJobService(sess *session.Session) Network_Media_Transcode_Job {
	return Network_Media_Transcode_Job{Session: sess}
}

func (r Network_Media_Transcode_Job) Id(id int) Network_Media_Transcode_Job {
	r.Options.Id = &id
	return r
}

func (r Network_Media_Transcode_Job) Mask(mask string) Network_Media_Transcode_Job {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Media_Transcode_Job) Filter(filter string) Network_Media_Transcode_Job {
	r.Options.Filter = filter
	return r
}

func (r Network_Media_Transcode_Job) Limit(limit int) Network_Media_Transcode_Job {
	r.Options.Limit = &limit
	return r
}

func (r Network_Media_Transcode_Job) Offset(offset int) Network_Media_Transcode_Job {
	r.Options.Offset = &offset
	return r
}

// With this method, you can create a transcode job.
//
// The very first step of creating a transcode job is to upload your media files to the /in directory on your Transcode FTP space. Then, you have to pass a [[SoftLayer_Network_Media_Transcode_Job|Transcode job]] object as a parameter for this method.
//
// There are 4 required properties of SoftLayer_Network_Media_Transcode_Job object: transcodePresetName, transcodePresetGuid, inputFile, and outputFile. A transcode preset is a configuration that defines a certain media output.  You can retrieve all the supported presets with the [[SoftLayer_Network_Media_Transcode_Account::getPresets|getPresets]] method. You can also use [[SoftLayer_Network_Media_Transcode_Account::getPresetDetail|getPresetDetail]] method to get more information on a preset. Use these two methods to determine appropriate values for "transcodePresetName" and "transcodePresetGuid" properties. For an "inputFile", you must specify a file that exists in the /in directory of your Transcode FTP space. An "outputFile" name will be used by the Transcode server for naming a transcoded file.  An output file name must be in /out directory. If your outputFile name already exists in the /out directory, the Transcode server will append a file name with _n (an underscore and the total number of files with the identical name plus 1).
//
// The "name" property is optional and it can help you keep track of transcode jobs easily. "autoDeleteDuration" is another optional property that you can specify.  It determines how soon your input file will be deleted. If autoDeleteDuration is set to zero, your input file will be removed immediately after the last transcode job running on it is completed. A value for autoDeleteDuration property is in seconds and the maximum value is 259200 which is 3 days.
//
// An example SoftLayer_Network_Media_Transcode_Job parameter looks like this:
//
//
// * name: My transcoding
// * transcodePresetName: F4V 896kbps 640x352 16x9 29.97fps
// * transcodePresetGuid: {87E01268-C3E3-4A85-9701-052C9AC42BD4}
// * inputFile: /in/my_birthday.wmv
// * outputFile: /out/my_birthday_flash
//
//
// Notice that an output file does not have a file extension.  The Transcode server will append a file extension based on an output format. A newly created transcode job will be in "Pending" status and it will be added to the Transcoding queue. You will receive a notification email whenever there is a status change on your transcode job.  For example, the Transcode server starts to process your transcode job, you will be notified via an email.
//
// You can add up to 3 pending jobs at a time. Transcode jobs with any other status such as "Complete" or "Error" will not be counted toward your pending jobs.
//
// Once a job is complete, the Transcode server will place the output file into the /out directory along with a notification email. The files in the /out directory will be removed 3 days after they were created.  You will need to use an FTP client to download transcoded files.
//
//
func (r Network_Media_Transcode_Job) CreateObject(templateObject *datatypes.Network_Media_Transcode_Job) (resp datatypes.Network_Media_Transcode_Job, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job", "createObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Media_Transcode_Job) GetHistory() (resp []datatypes.Network_Media_Transcode_Job_History, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job", "getHistory", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Media_Transcode_Job) GetObject() (resp datatypes.Network_Media_Transcode_Job, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The transcode service account
func (r Network_Media_Transcode_Job) GetTranscodeAccount() (resp datatypes.Network_Media_Transcode_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job", "getTranscodeAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The status information of a transcode job
func (r Network_Media_Transcode_Job) GetTranscodeStatus() (resp datatypes.Network_Media_Transcode_Job_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job", "getTranscodeStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The status of a transcode job
func (r Network_Media_Transcode_Job) GetTranscodeStatusName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job", "getTranscodeStatusName", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer user that created the transcode job
func (r Network_Media_Transcode_Job) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job", "getUser", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Media_Transcode_Job_Status contains information on a transcode job status.
type Network_Media_Transcode_Job_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMediaTranscodeJobStatusService returns an instance of the Network_Media_Transcode_Job_Status SoftLayer service
func GetNetworkMediaTranscodeJobStatusService(sess *session.Session) Network_Media_Transcode_Job_Status {
	return Network_Media_Transcode_Job_Status{Session: sess}
}

func (r Network_Media_Transcode_Job_Status) Id(id int) Network_Media_Transcode_Job_Status {
	r.Options.Id = &id
	return r
}

func (r Network_Media_Transcode_Job_Status) Mask(mask string) Network_Media_Transcode_Job_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Media_Transcode_Job_Status) Filter(filter string) Network_Media_Transcode_Job_Status {
	r.Options.Filter = filter
	return r
}

func (r Network_Media_Transcode_Job_Status) Limit(limit int) Network_Media_Transcode_Job_Status {
	r.Options.Limit = &limit
	return r
}

func (r Network_Media_Transcode_Job_Status) Offset(offset int) Network_Media_Transcode_Job_Status {
	r.Options.Offset = &offset
	return r
}

// This method returns all transcode job statuses.
func (r Network_Media_Transcode_Job_Status) GetAllStatuses() (resp []datatypes.Network_Media_Transcode_Job_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job_Status", "getAllStatuses", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Media_Transcode_Job_Status) GetObject() (resp datatypes.Network_Media_Transcode_Job_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Media_Transcode_Job_Status", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Message_Delivery struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMessageDeliveryService returns an instance of the Network_Message_Delivery SoftLayer service
func GetNetworkMessageDeliveryService(sess *session.Session) Network_Message_Delivery {
	return Network_Message_Delivery{Session: sess}
}

func (r Network_Message_Delivery) Id(id int) Network_Message_Delivery {
	r.Options.Id = &id
	return r
}

func (r Network_Message_Delivery) Mask(mask string) Network_Message_Delivery {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Message_Delivery) Filter(filter string) Network_Message_Delivery {
	r.Options.Filter = filter
	return r
}

func (r Network_Message_Delivery) Limit(limit int) Network_Message_Delivery {
	r.Options.Limit = &limit
	return r
}

func (r Network_Message_Delivery) Offset(offset int) Network_Message_Delivery {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Message_Delivery) EditObject(templateObject *datatypes.Network_Message_Delivery) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer customer account that a network message delivery account belongs to.
func (r Network_Message_Delivery) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The billing item for a network message delivery account.
func (r Network_Message_Delivery) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery", "getBillingItem", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery) GetObject() (resp datatypes.Network_Message_Delivery, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The message delivery type of a network message delivery account.
func (r Network_Message_Delivery) GetType() (resp datatypes.Network_Message_Delivery_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve The vendor for a network message delivery account.
func (r Network_Message_Delivery) GetVendor() (resp datatypes.Network_Message_Delivery_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery", "getVendor", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Message_Delivery_Email_Sendgrid struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMessageDeliveryEmailSendgridService returns an instance of the Network_Message_Delivery_Email_Sendgrid SoftLayer service
func GetNetworkMessageDeliveryEmailSendgridService(sess *session.Session) Network_Message_Delivery_Email_Sendgrid {
	return Network_Message_Delivery_Email_Sendgrid{Session: sess}
}

func (r Network_Message_Delivery_Email_Sendgrid) Id(id int) Network_Message_Delivery_Email_Sendgrid {
	r.Options.Id = &id
	return r
}

func (r Network_Message_Delivery_Email_Sendgrid) Mask(mask string) Network_Message_Delivery_Email_Sendgrid {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Message_Delivery_Email_Sendgrid) Filter(filter string) Network_Message_Delivery_Email_Sendgrid {
	r.Options.Filter = filter
	return r
}

func (r Network_Message_Delivery_Email_Sendgrid) Limit(limit int) Network_Message_Delivery_Email_Sendgrid {
	r.Options.Limit = &limit
	return r
}

func (r Network_Message_Delivery_Email_Sendgrid) Offset(offset int) Network_Message_Delivery_Email_Sendgrid {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) AddUnsubscribeEmailAddress(emailAddress *string) (resp bool, err error) {
	params := []interface{}{
		emailAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "addUnsubscribeEmailAddress", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) DeleteEmailListEntries(list *string, entries []string) (resp bool, err error) {
	params := []interface{}{
		list,
		entries,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "deleteEmailListEntries", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) DisableSmtpAccess() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "disableSmtpAccess", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) EditObject(templateObject *datatypes.Network_Message_Delivery) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "editObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) EnableSmtpAccess() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "enableSmtpAccess", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer customer account that a network message delivery account belongs to.
func (r Network_Message_Delivery_Email_Sendgrid) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getAccount", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) GetAccountOverview() (resp datatypes.Container_Network_Message_Delivery_Email_Sendgrid_Account_Overview, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getAccountOverview", nil, &r.Options, &resp)
	return
}

// Retrieve The billing item for a network message delivery account.
func (r Network_Message_Delivery_Email_Sendgrid) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getBillingItem", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) GetCategoryList() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getCategoryList", nil, &r.Options, &resp)
	return
}

// Retrieve The contact e-mail address used by SendGrid.
func (r Network_Message_Delivery_Email_Sendgrid) GetEmailAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getEmailAddress", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) GetEmailList(list *string) (resp []datatypes.Container_Network_Message_Delivery_Email_Sendgrid_List_Entry, err error) {
	params := []interface{}{
		list,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getEmailList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) GetObject() (resp datatypes.Network_Message_Delivery_Email_Sendgrid, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve A flag that determines if a SendGrid e-mail delivery account has access to send mail through the SendGrid SMTP server.
func (r Network_Message_Delivery_Email_Sendgrid) GetSmtpAccess() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getSmtpAccess", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) GetStatistics(options *datatypes.Container_Network_Message_Delivery_Email_Sendgrid_Statistics_Options) (resp []datatypes.Container_Network_Message_Delivery_Email_Sendgrid_Statistics, err error) {
	params := []interface{}{
		options,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getStatistics", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) GetStatisticsGraph(options *datatypes.Container_Network_Message_Delivery_Email_Sendgrid_Statistics_Options) (resp datatypes.Container_Network_Message_Delivery_Email_Sendgrid_Statistics_Graph, err error) {
	params := []interface{}{
		options,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getStatisticsGraph", params, &r.Options, &resp)
	return
}

// Retrieve The message delivery type of a network message delivery account.
func (r Network_Message_Delivery_Email_Sendgrid) GetType() (resp datatypes.Network_Message_Delivery_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve The vendor for a network message delivery account.
func (r Network_Message_Delivery_Email_Sendgrid) GetVendor() (resp datatypes.Network_Message_Delivery_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getVendor", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) GetVendorPortalUrl() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "getVendorPortalUrl", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) SendEmail(emailContainer *datatypes.Container_Network_Message_Delivery_Email) (resp bool, err error) {
	params := []interface{}{
		emailContainer,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "sendEmail", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Delivery_Email_Sendgrid) UpdateEmailAddress(emailAddress *string) (resp bool, err error) {
	params := []interface{}{
		emailAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Delivery_Email_Sendgrid", "updateEmailAddress", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Message_Queue data type contains general information relating to Message Queue account
type Network_Message_Queue struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMessageQueueService returns an instance of the Network_Message_Queue SoftLayer service
func GetNetworkMessageQueueService(sess *session.Session) Network_Message_Queue {
	return Network_Message_Queue{Session: sess}
}

func (r Network_Message_Queue) Id(id int) Network_Message_Queue {
	r.Options.Id = &id
	return r
}

func (r Network_Message_Queue) Mask(mask string) Network_Message_Queue {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Message_Queue) Filter(filter string) Network_Message_Queue {
	r.Options.Filter = filter
	return r
}

func (r Network_Message_Queue) Limit(limit int) Network_Message_Queue {
	r.Options.Limit = &limit
	return r
}

func (r Network_Message_Queue) Offset(offset int) Network_Message_Queue {
	r.Options.Offset = &offset
	return r
}

// Retrieve The account that a message queue belongs to.
func (r Network_Message_Queue) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for this message queue account.
func (r Network_Message_Queue) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve All available message queue nodes
func (r Network_Message_Queue) GetNodes() (resp []datatypes.Network_Message_Queue_Node, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue", "getNodes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Queue) GetObject() (resp datatypes.Network_Message_Queue, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve A message queue account status.
func (r Network_Message_Queue) GetStatus() (resp datatypes.Network_Message_Queue_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue", "getStatus", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Message_Queue_Node data type contains general information relating to Message Queue node
type Network_Message_Queue_Node struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMessageQueueNodeService returns an instance of the Network_Message_Queue_Node SoftLayer service
func GetNetworkMessageQueueNodeService(sess *session.Session) Network_Message_Queue_Node {
	return Network_Message_Queue_Node{Session: sess}
}

func (r Network_Message_Queue_Node) Id(id int) Network_Message_Queue_Node {
	r.Options.Id = &id
	return r
}

func (r Network_Message_Queue_Node) Mask(mask string) Network_Message_Queue_Node {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Message_Queue_Node) Filter(filter string) Network_Message_Queue_Node {
	r.Options.Filter = filter
	return r
}

func (r Network_Message_Queue_Node) Limit(limit int) Network_Message_Queue_Node {
	r.Options.Limit = &limit
	return r
}

func (r Network_Message_Queue_Node) Offset(offset int) Network_Message_Queue_Node {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Message_Queue_Node) AddUser(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "addUser", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Queue_Node) DeleteUser(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "deleteUser", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Queue_Node) GetAllUsers() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "getAllUsers", nil, &r.Options, &resp)
	return
}

// Retrieve The message queue account this node belongs to.
func (r Network_Message_Queue_Node) GetMessageQueue() (resp datatypes.Network_Message_Queue, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "getMessageQueue", nil, &r.Options, &resp)
	return
}

// Retrieve A message queue node's metric tracking object. This object records all request and notification count data for this message queue node.
func (r Network_Message_Queue_Node) GetMetricTrackingObject() (resp datatypes.Metric_Tracking_Object, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "getMetricTrackingObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Message_Queue_Node) GetObject() (resp datatypes.Network_Message_Queue_Node, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Message_Queue_Node) GetServiceResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "getServiceResource", nil, &r.Options, &resp)
	return
}

// Retrieve usage graph by date.
func (r Network_Message_Queue_Node) GetUsage(startDate *datatypes.Time, endDate *datatypes.Time) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "getUsage", params, &r.Options, &resp)
	return
}

// Retrieve usage graph by date.
func (r Network_Message_Queue_Node) GetUsageGraph(graphData *datatypes.Container_Graph) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		graphData,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Node", "getUsageGraph", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Message_Queue_Status data type contains general information relating to Message Queue account status.
type Network_Message_Queue_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMessageQueueStatusService returns an instance of the Network_Message_Queue_Status SoftLayer service
func GetNetworkMessageQueueStatusService(sess *session.Session) Network_Message_Queue_Status {
	return Network_Message_Queue_Status{Session: sess}
}

func (r Network_Message_Queue_Status) Id(id int) Network_Message_Queue_Status {
	r.Options.Id = &id
	return r
}

func (r Network_Message_Queue_Status) Mask(mask string) Network_Message_Queue_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Message_Queue_Status) Filter(filter string) Network_Message_Queue_Status {
	r.Options.Filter = filter
	return r
}

func (r Network_Message_Queue_Status) Limit(limit int) Network_Message_Queue_Status {
	r.Options.Limit = &limit
	return r
}

func (r Network_Message_Queue_Status) Offset(offset int) Network_Message_Queue_Status {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Message_Queue_Status) GetObject() (resp datatypes.Network_Message_Queue_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Message_Queue_Status", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Monitor struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMonitorService returns an instance of the Network_Monitor SoftLayer service
func GetNetworkMonitorService(sess *session.Session) Network_Monitor {
	return Network_Monitor{Session: sess}
}

func (r Network_Monitor) Id(id int) Network_Monitor {
	r.Options.Id = &id
	return r
}

func (r Network_Monitor) Mask(mask string) Network_Monitor {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Monitor) Filter(filter string) Network_Monitor {
	r.Options.Filter = filter
	return r
}

func (r Network_Monitor) Limit(limit int) Network_Monitor {
	r.Options.Limit = &limit
	return r
}

func (r Network_Monitor) Offset(offset int) Network_Monitor {
	r.Options.Offset = &offset
	return r
}

// This will return an arrayObject of objects containing the ipaddresses.  Using an string parameter you can send a partial ipaddress to search within a given ipaddress.  You can also set the max limit as well using the setting the resultLimit.
func (r Network_Monitor) GetIpAddressesByHardware(hardware *datatypes.Hardware, partialIpAddress *string) (resp []datatypes.Network_Subnet_IpAddress, err error) {
	params := []interface{}{
		hardware,
		partialIpAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Monitor", "getIpAddressesByHardware", params, &r.Options, &resp)
	return
}

// This will return an arrayObject of objects containing the ipaddresses.  Using an string parameter you can send a partial ipaddress to search within a given ipaddress.  You can also set the max limit as well using the setting the resultLimit.
func (r Network_Monitor) GetIpAddressesByVirtualGuest(guest *datatypes.Virtual_Guest, partialIpAddress *string) (resp []datatypes.Network_Subnet_IpAddress, err error) {
	params := []interface{}{
		guest,
		partialIpAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Monitor", "getIpAddressesByVirtualGuest", params, &r.Options, &resp)
	return
}

// The Monitoring_Query_Host type represents a monitoring instance.  It consists of a hardware ID to monitor, an IP address attached to that hardware ID, a method of monitoring, and what to do in the instance that the monitor ever fails.
type Network_Monitor_Version1_Query_Host struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMonitorVersion1QueryHostService returns an instance of the Network_Monitor_Version1_Query_Host SoftLayer service
func GetNetworkMonitorVersion1QueryHostService(sess *session.Session) Network_Monitor_Version1_Query_Host {
	return Network_Monitor_Version1_Query_Host{Session: sess}
}

func (r Network_Monitor_Version1_Query_Host) Id(id int) Network_Monitor_Version1_Query_Host {
	r.Options.Id = &id
	return r
}

func (r Network_Monitor_Version1_Query_Host) Mask(mask string) Network_Monitor_Version1_Query_Host {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Monitor_Version1_Query_Host) Filter(filter string) Network_Monitor_Version1_Query_Host {
	r.Options.Filter = filter
	return r
}

func (r Network_Monitor_Version1_Query_Host) Limit(limit int) Network_Monitor_Version1_Query_Host {
	r.Options.Limit = &limit
	return r
}

func (r Network_Monitor_Version1_Query_Host) Offset(offset int) Network_Monitor_Version1_Query_Host {
	r.Options.Offset = &offset
	return r
}

// Passing in an unsaved instances of a Query_Host object into this function will create the object and return the results to the user.
func (r Network_Monitor_Version1_Query_Host) CreateObject(templateObject *datatypes.Network_Monitor_Version1_Query_Host) (resp datatypes.Network_Monitor_Version1_Query_Host, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "createObject", params, &r.Options, &resp)
	return
}

// Passing in a collection of unsaved instances of Query_Host objects into this function will create all objects and return the results to the user.
func (r Network_Monitor_Version1_Query_Host) CreateObjects(templateObjects []datatypes.Network_Monitor_Version1_Query_Host) (resp []datatypes.Network_Monitor_Version1_Query_Host, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "createObjects", params, &r.Options, &resp)
	return
}

// Like any other API object, the monitoring objects can be deleted by passing an instance of them into this function.  The ID on the object must be set.
func (r Network_Monitor_Version1_Query_Host) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "deleteObject", nil, &r.Options, &resp)
	return
}

// Like any other API object, the monitoring objects can be deleted by passing an instance of them into this function.  The ID on the object must be set.
func (r Network_Monitor_Version1_Query_Host) DeleteObjects(templateObjects []datatypes.Network_Monitor_Version1_Query_Host) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "deleteObjects", params, &r.Options, &resp)
	return
}

// Like any other API object, the monitoring objects can have their exposed properties edited by passing in a modified version of the object.
func (r Network_Monitor_Version1_Query_Host) EditObject(templateObject *datatypes.Network_Monitor_Version1_Query_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "editObject", params, &r.Options, &resp)
	return
}

// Like any other API object, the monitoring objects can have their exposed properties edited by passing in a modified version of the object.
func (r Network_Monitor_Version1_Query_Host) EditObjects(templateObjects []datatypes.Network_Monitor_Version1_Query_Host) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "editObjects", params, &r.Options, &resp)
	return
}

// This method returns all Query_Host objects associated with the passed in hardware ID as long as that hardware ID is owned by the current user's account.
//
// This behavior can also be accomplished by simply tapping networkMonitors on the Hardware_Server object.
func (r Network_Monitor_Version1_Query_Host) FindByHardwareId(hardwareId *int) (resp []datatypes.Network_Monitor_Version1_Query_Host, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "findByHardwareId", params, &r.Options, &resp)
	return
}

// Retrieve The hardware that is being monitored by this monitoring instance
func (r Network_Monitor_Version1_Query_Host) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The most recent result for this particular monitoring instance.
func (r Network_Monitor_Version1_Query_Host) GetLastResult() (resp datatypes.Network_Monitor_Version1_Query_Result, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "getLastResult", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Monitor_Version1_Query_Host object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Monitor_Version1_Query_Host service. You can only retrieve query hosts attached to hardware that belong to your account.
func (r Network_Monitor_Version1_Query_Host) GetObject() (resp datatypes.Network_Monitor_Version1_Query_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The type of monitoring query that is executed when this hardware is monitored.
func (r Network_Monitor_Version1_Query_Host) GetQueryType() (resp datatypes.Network_Monitor_Version1_Query_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "getQueryType", nil, &r.Options, &resp)
	return
}

// Retrieve The action taken when a monitor fails.
func (r Network_Monitor_Version1_Query_Host) GetResponseAction() (resp datatypes.Network_Monitor_Version1_Query_ResponseType, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host", "getResponseAction", nil, &r.Options, &resp)
	return
}

// The monitoring stratum type stores the maximum level of the various components of the monitoring system that a particular hardware object has access to.  This object cannot be accessed by ID, and cannot be modified. The user can access this object through Hardware_Server->availableMonitoring.
//
// There are two values on this object that are important:
// # monitorLevel determines the highest level of SoftLayer_Network_Monitor_Version1_Query_Type object that can be placed in a monitoring instance on this server
// # responseLevel determines the highest level of SoftLayer_Network_Monitor_Version1_Query_ResponseType object that can be placed in a monitoring instance on this server
//
//
// Also note that the query type and response types are available through getAllQueryTypes and getAllResponseTypes, respectively.
type Network_Monitor_Version1_Query_Host_Stratum struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkMonitorVersion1QueryHostStratumService returns an instance of the Network_Monitor_Version1_Query_Host_Stratum SoftLayer service
func GetNetworkMonitorVersion1QueryHostStratumService(sess *session.Session) Network_Monitor_Version1_Query_Host_Stratum {
	return Network_Monitor_Version1_Query_Host_Stratum{Session: sess}
}

func (r Network_Monitor_Version1_Query_Host_Stratum) Id(id int) Network_Monitor_Version1_Query_Host_Stratum {
	r.Options.Id = &id
	return r
}

func (r Network_Monitor_Version1_Query_Host_Stratum) Mask(mask string) Network_Monitor_Version1_Query_Host_Stratum {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Monitor_Version1_Query_Host_Stratum) Filter(filter string) Network_Monitor_Version1_Query_Host_Stratum {
	r.Options.Filter = filter
	return r
}

func (r Network_Monitor_Version1_Query_Host_Stratum) Limit(limit int) Network_Monitor_Version1_Query_Host_Stratum {
	r.Options.Limit = &limit
	return r
}

func (r Network_Monitor_Version1_Query_Host_Stratum) Offset(offset int) Network_Monitor_Version1_Query_Host_Stratum {
	r.Options.Offset = &offset
	return r
}

// Calling this function returns all possible query type objects. These objects are to be used to set the values on the SoftLayer_Network_Monitor_Version1_Query_Host when creating new monitoring instances.
func (r Network_Monitor_Version1_Query_Host_Stratum) GetAllQueryTypes() (resp []datatypes.Network_Monitor_Version1_Query_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host_Stratum", "getAllQueryTypes", nil, &r.Options, &resp)
	return
}

// Calling this function returns all possible response type objects. These objects are to be used to set the values on the SoftLayer_Network_Monitor_Version1_Query_Host when creating new monitoring instances.
func (r Network_Monitor_Version1_Query_Host_Stratum) GetAllResponseTypes() (resp []datatypes.Network_Monitor_Version1_Query_ResponseType, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host_Stratum", "getAllResponseTypes", nil, &r.Options, &resp)
	return
}

// Retrieve The hardware object that these monitoring permissions applies to.
func (r Network_Monitor_Version1_Query_Host_Stratum) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host_Stratum", "getHardware", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Monitor_Version1_Query_Host_Stratum object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Monitor_Version1_Query_Host_Stratum service. You can only retrieve strata attached to hardware that belong to your account.
func (r Network_Monitor_Version1_Query_Host_Stratum) GetObject() (resp datatypes.Network_Monitor_Version1_Query_Host_Stratum, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Monitor_Version1_Query_Host_Stratum", "getObject", nil, &r.Options, &resp)
	return
}

// SoftLayer_Network_Pod refers to a portion of a data center that share a Backend Customer Router (BCR) and usually a front-end counterpart known as a Frontend Customer Router (FCR). A Pod primarily denotes a logical location within the network and the physical aspects that support networks. This is in contrast to representing a specific physical location.
//
// A ``Pod`` is identified by a ``name``, which is unique. A Pod name follows the format 'dddnn.podii', where 'ddd' is a data center code, 'nn' is the data center number, 'pod' is a literal string and 'ii' is a two digit, left-zero- padded number which corresponds to a Backend Customer Router (BCR) of the desired data center. Examples:
// * dal09.pod01 = Dallas 9, Pod 1 (ie. bcr01)
// * sjc01.pod04 = San Jose 1, Pod 4 (ie. bcr04)
// * ams01.pod01 = Amsterdam 1, Pod 1 (ie. bcr01)
type Network_Pod struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkPodService returns an instance of the Network_Pod SoftLayer service
func GetNetworkPodService(sess *session.Session) Network_Pod {
	return Network_Pod{Session: sess}
}

func (r Network_Pod) Id(id int) Network_Pod {
	r.Options.Id = &id
	return r
}

func (r Network_Pod) Mask(mask string) Network_Pod {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Pod) Filter(filter string) Network_Pod {
	r.Options.Filter = filter
	return r
}

func (r Network_Pod) Limit(limit int) Network_Pod {
	r.Options.Limit = &limit
	return r
}

func (r Network_Pod) Offset(offset int) Network_Pod {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Pod) GetAllObjects() (resp []datatypes.Network_Pod, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Pod", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Provides the list of capabilities a Pod fulfills. See [[SoftLayer_Network_Pod/listCapabilities]] for more information on capabilities.
func (r Network_Pod) GetCapabilities() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Pod", "getCapabilities", nil, &r.Options, &resp)
	return
}

// Set the initialization parameter to the ``name`` of the Pod to retrieve.
func (r Network_Pod) GetObject() (resp datatypes.Network_Pod, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Pod", "getObject", nil, &r.Options, &resp)
	return
}

// A capability is simply a string literal that denotes the availability of a feature. Capabilities are generally self describing, but any additional details concerning the implications of a capability will be documented elsewhere; usually by the Service or Operation related to it.
func (r Network_Pod) ListCapabilities() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Pod", "listCapabilities", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Security_Scanner_Request data type represents a single vulnerability scan request. It provides information on when the scan was created, last updated, and the current status. The status messages are as follows:
// *Scan Pending
// *Scan Processing
// *Scan Complete
// *Scan Cancelled
// *Generating Report.
type Network_Security_Scanner_Request struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSecurityScannerRequestService returns an instance of the Network_Security_Scanner_Request SoftLayer service
func GetNetworkSecurityScannerRequestService(sess *session.Session) Network_Security_Scanner_Request {
	return Network_Security_Scanner_Request{Session: sess}
}

func (r Network_Security_Scanner_Request) Id(id int) Network_Security_Scanner_Request {
	r.Options.Id = &id
	return r
}

func (r Network_Security_Scanner_Request) Mask(mask string) Network_Security_Scanner_Request {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Security_Scanner_Request) Filter(filter string) Network_Security_Scanner_Request {
	r.Options.Filter = filter
	return r
}

func (r Network_Security_Scanner_Request) Limit(limit int) Network_Security_Scanner_Request {
	r.Options.Limit = &limit
	return r
}

func (r Network_Security_Scanner_Request) Offset(offset int) Network_Security_Scanner_Request {
	r.Options.Offset = &offset
	return r
}

// Create a new vulnerability scan request. New scan requests are picked up every five minutes, and the time to complete an actual scan may vary. Once the scan is finished, it can take up to another five minutes for the report to be generated and accessible.
func (r Network_Security_Scanner_Request) CreateObject(templateObject *datatypes.Network_Security_Scanner_Request) (resp datatypes.Network_Security_Scanner_Request, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Security_Scanner_Request", "createObject", params, &r.Options, &resp)
	return
}

// Retrieve The account associated with a security scan request.
func (r Network_Security_Scanner_Request) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Security_Scanner_Request", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The virtual guest a security scan is run against.
func (r Network_Security_Scanner_Request) GetGuest() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Security_Scanner_Request", "getGuest", nil, &r.Options, &resp)
	return
}

// Retrieve The hardware a security scan is run against.
func (r Network_Security_Scanner_Request) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Security_Scanner_Request", "getHardware", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Security_Scanner_Request object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Security_Scanner_Request service. You can only retrieve requests and reports that are assigned to your SoftLayer account.
func (r Network_Security_Scanner_Request) GetObject() (resp datatypes.Network_Security_Scanner_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Security_Scanner_Request", "getObject", nil, &r.Options, &resp)
	return
}

// Get the vulnerability report for a scan request, formatted as HTML string. Previous scan reports are held indefinitely.
func (r Network_Security_Scanner_Request) GetReport() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Security_Scanner_Request", "getReport", nil, &r.Options, &resp)
	return
}

// Retrieve Flag whether the requestor owns the hardware the scan was run on. This flag will  return for hardware servers only, virtual servers will result in a null return even if you have  a request out for them.
func (r Network_Security_Scanner_Request) GetRequestorOwnedFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Security_Scanner_Request", "getRequestorOwnedFlag", nil, &r.Options, &resp)
	return
}

// Retrieve A security scan request's status.
func (r Network_Security_Scanner_Request) GetStatus() (resp datatypes.Network_Security_Scanner_Request_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Security_Scanner_Request", "getStatus", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Service_Vpn_Overrides data type contains information relating user ids to subnet ids when VPN access is manually configured.  It is essentially an entry in a 'white list' of subnets a SoftLayer portal VPN user may access.
type Network_Service_Vpn_Overrides struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkServiceVpnOverridesService returns an instance of the Network_Service_Vpn_Overrides SoftLayer service
func GetNetworkServiceVpnOverridesService(sess *session.Session) Network_Service_Vpn_Overrides {
	return Network_Service_Vpn_Overrides{Session: sess}
}

func (r Network_Service_Vpn_Overrides) Id(id int) Network_Service_Vpn_Overrides {
	r.Options.Id = &id
	return r
}

func (r Network_Service_Vpn_Overrides) Mask(mask string) Network_Service_Vpn_Overrides {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Service_Vpn_Overrides) Filter(filter string) Network_Service_Vpn_Overrides {
	r.Options.Filter = filter
	return r
}

func (r Network_Service_Vpn_Overrides) Limit(limit int) Network_Service_Vpn_Overrides {
	r.Options.Limit = &limit
	return r
}

func (r Network_Service_Vpn_Overrides) Offset(offset int) Network_Service_Vpn_Overrides {
	r.Options.Offset = &offset
	return r
}

// Create Softlayer portal user VPN overrides.
func (r Network_Service_Vpn_Overrides) CreateObjects(templateObjects []datatypes.Network_Service_Vpn_Overrides) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Service_Vpn_Overrides", "createObjects", params, &r.Options, &resp)
	return
}

// Use this method to delete a single SoftLayer portal VPN user subnet override.
func (r Network_Service_Vpn_Overrides) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Service_Vpn_Overrides", "deleteObject", nil, &r.Options, &resp)
	return
}

// Use this method to delete a collection of SoftLayer portal VPN user subnet overrides.
func (r Network_Service_Vpn_Overrides) DeleteObjects(templateObjects []datatypes.Network_Service_Vpn_Overrides) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Service_Vpn_Overrides", "deleteObjects", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Service_Vpn_Overrides) GetObject() (resp datatypes.Network_Service_Vpn_Overrides, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Service_Vpn_Overrides", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve Subnet components accessible by a SoftLayer VPN portal user.
func (r Network_Service_Vpn_Overrides) GetSubnet() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Service_Vpn_Overrides", "getSubnet", nil, &r.Options, &resp)
	return
}

// Retrieve SoftLayer VPN portal user.
func (r Network_Service_Vpn_Overrides) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Service_Vpn_Overrides", "getUser", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Storage data type contains general information regarding a Storage product such as account id, access username and password, the Storage product type, and the server the Storage service is associated with. Currently, only EVault backup storage has an associated server.
type Network_Storage struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageService returns an instance of the Network_Storage SoftLayer service
func GetNetworkStorageService(sess *session.Session) Network_Storage {
	return Network_Storage{Session: sess}
}

func (r Network_Storage) Id(id int) Network_Storage {
	r.Options.Id = &id
	return r
}

func (r Network_Storage) Mask(mask string) Network_Storage {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage) Filter(filter string) Network_Storage {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage) Limit(limit int) Network_Storage {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage) Offset(offset int) Network_Storage {
	r.Options.Offset = &offset
	return r
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage) AllowAccessFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromHardware", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) AllowAccessFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage) AllowAccessFromHost(typeClassName *string, hostId *int) (resp datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		typeClassName,
		hostId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromHost", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage volume will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage) AllowAccessFromHostList(hostObjectTemplates []datatypes.Container_Network_Storage_Host) (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		hostObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromHostList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage will be listed in the allowedIpAddresses property of this storage volume.
func (r Network_Storage) AllowAccessFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromIpAddress", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) AllowAccessFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromIpAddressList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage) AllowAccessFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromSubnet", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) AllowAccessFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage) AllowAccessFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage) AllowAccessFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage replicant volume.
func (r Network_Storage) AllowAccessToReplicantFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessToReplicantFromHardware", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Hardware objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationHardware property of this storage volume.
func (r Network_Storage) AllowAccessToReplicantFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessToReplicantFromHardwareList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) AllowAccessToReplicantFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessToReplicantFromIpAddress", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationIpAddresses property of this storage volume.
func (r Network_Storage) AllowAccessToReplicantFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessToReplicantFromIpAddressList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage replicant volume.
func (r Network_Storage) AllowAccessToReplicantFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessToReplicantFromSubnet", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationSubnets property of this storage volume.
func (r Network_Storage) AllowAccessToReplicantFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessToReplicantFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage replicant volume.
func (r Network_Storage) AllowAccessToReplicantFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessToReplicantFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationVirtualGuests property of this storage volume.
func (r Network_Storage) AllowAccessToReplicantFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "allowAccessToReplicantFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method will assign an existing credential to the current volume. The credential must have been created using the 'addNewCredential' method. The volume type must support an additional credential.
func (r Network_Storage) AssignCredential(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "assignCredential", params, &r.Options, &resp)
	return
}

// This method will set up a new credential for the remote storage volume. The storage volume must support an additional credential. Once created, the credential will be automatically assigned to the current volume. If there are no volumes assigned to the credential it will be automatically deleted.
func (r Network_Storage) AssignNewCredential(typ *string) (resp datatypes.Network_Storage_Credential, err error) {
	params := []interface{}{
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "assignNewCredential", params, &r.Options, &resp)
	return
}

// The method will change the password for the given Storage/Virtual Server Storage account.
func (r Network_Storage) ChangePassword(username *string, currentPassword *string, newPassword *string) (resp bool, err error) {
	params := []interface{}{
		username,
		currentPassword,
		newPassword,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "changePassword", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// collectBandwidth() Retrieve the bandwidth usage for the current billing cycle.
func (r Network_Storage) CollectBandwidth(typ *string, startDate *datatypes.Time, endDate *datatypes.Time) (resp uint, err error) {
	params := []interface{}{
		typ,
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "collectBandwidth", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// collectBytesUsed() retrieves the number of bytes capacity currently in use on a Storage account.
func (r Network_Storage) CollectBytesUsed() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "collectBytesUsed", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) CreateFolder(folder *string) (resp bool, err error) {
	params := []interface{}{
		folder,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "createFolder", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) CreateSnapshot(notes *string) (resp datatypes.Network_Storage, err error) {
	params := []interface{}{
		notes,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "createSnapshot", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete all files within a Storage account. Depending on the type of Storage account, Deleting either deletes files permanently or sends files to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the files are in the account's recycle bin. If the files exist in the recycle bin, then they are permanently deleted.
//
// Please note, files can not be restored once they are permanently deleted.
func (r Network_Storage) DeleteAllFiles() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "deleteAllFiles", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete an individual file within a Storage account. Depending on the type of Storage account, Deleting a file either deletes the file permanently or sends the file to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the file is in the account's recycle bin. If the file exist in the recycle bin, then it is permanently deleted.
//
// Please note, a file can not be restored once it is permanently deleted.
func (r Network_Storage) DeleteFile(fileId *string) (resp bool, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "deleteFile", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete multiple files within a Storage account. Depending on the type of Storage account, Deleting either deletes files permanently or sends files to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the files are in the account's recycle bin. If the files exist in the recycle bin, then they are permanently deleted.
//
// Please note, files can not be restored once they are permanently deleted.
func (r Network_Storage) DeleteFiles(fileIds []string) (resp bool, err error) {
	params := []interface{}{
		fileIds,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "deleteFiles", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) DeleteFolder(folder *string) (resp bool, err error) {
	params := []interface{}{
		folder,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "deleteFolder", params, &r.Options, &resp)
	return
}

// Delete a network storage volume. '''This cannot be undone.''' At this time only network storage snapshots may be deleted with this method.
//
// ''deleteObject'' returns Boolean ''true'' on successful deletion or ''false'' if it was unable to remove a volume;
func (r Network_Storage) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method is not valid for Legacy iSCSI Storage Volumes.
//
// Disable scheduled snapshots of this storage volume. Scheduling options include HOURLY, DAILY and WEEKLY schedules.
func (r Network_Storage) DisableSnapshots(scheduleType *string) (resp bool, err error) {
	params := []interface{}{
		scheduleType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "disableSnapshots", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Download a file from a Storage account. This method returns a file's details including the file's raw content.
func (r Network_Storage) DownloadFile(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "downloadFile", params, &r.Options, &resp)
	return
}

// This method will change the password of a credential created using the 'addNewCredential' method. If the credential exists on multiple storage volumes it will change for those volumes as well.
func (r Network_Storage) EditCredential(username *string, newPassword *string) (resp bool, err error) {
	params := []interface{}{
		username,
		newPassword,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "editCredential", params, &r.Options, &resp)
	return
}

// The password and/or notes may be modified for the Storage service except evault passwords and notes.
func (r Network_Storage) EditObject(templateObject *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "editObject", params, &r.Options, &resp)
	return
}

// This method is not valid for Legacy iSCSI Storage Volumes.
//
// Enable scheduled snapshots of this storage volume. Scheduling options include HOURLY, DAILY and WEEKLY schedules. For HOURLY schedules, provide relevant data for $scheduleType, $retentionCount and $minute. For DAILY schedules, provide relevant data for $scheduleType, $retentionCount, $minute, and $hour. For WEEKLY schedules, provide relevant data for all parameters of this method.
func (r Network_Storage) EnableSnapshots(scheduleType *string, retentionCount *int, minute *int, hour *int, dayOfWeek *string) (resp bool, err error) {
	params := []interface{}{
		scheduleType,
		retentionCount,
		minute,
		hour,
		dayOfWeek,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "enableSnapshots", params, &r.Options, &resp)
	return
}

// Failback from a volume replicant. In order to failback the volume must have already been failed over to a replicant.
func (r Network_Storage) FailbackFromReplicant() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "failbackFromReplicant", nil, &r.Options, &resp)
	return
}

// Failover to a volume replicant.  During the time which the replicant is in use the local nas volume will not be available.
func (r Network_Storage) FailoverToReplicant(replicantId *int) (resp bool, err error) {
	params := []interface{}{
		replicantId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "failoverToReplicant", params, &r.Options, &resp)
	return
}

// Retrieve The account that a Storage services belongs to.
func (r Network_Storage) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve Other usernames and passwords associated with a Storage volume.
func (r Network_Storage) GetAccountPassword() (resp datatypes.Account_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAccountPassword", nil, &r.Options, &resp)
	return
}

// Retrieve The currently active transactions on a network storage volume.
func (r Network_Storage) GetActiveTransactions() (resp []datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getActiveTransactions", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date for all files in a Storage account's root directory. This does not download file content.
func (r Network_Storage) GetAllFiles() (resp []datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllFiles", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date for all files matching the filter's criteria in a Storage account's root directory. This does not download file content.
func (r Network_Storage) GetAllFilesByFilter(filter *datatypes.Container_Utility_File_Entity) (resp []datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		filter,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllFilesByFilter", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Hardware that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage) GetAllowableHardware(filterHostname *string) (resp []datatypes.Hardware, err error) {
	params := []interface{}{
		filterHostname,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowableHardware", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Network_Subnet_IpAddress that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage) GetAllowableIpAddresses(subnetId *int, filterIpAddress *string) (resp []datatypes.Network_Subnet_IpAddress, err error) {
	params := []interface{}{
		subnetId,
		filterIpAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowableIpAddresses", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Network_Subnet that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage) GetAllowableSubnets(filterNetworkIdentifier *string) (resp []datatypes.Network_Subnet, err error) {
	params := []interface{}{
		filterNetworkIdentifier,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowableSubnets", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Virtual_Guest that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage) GetAllowableVirtualGuests(filterHostname *string) (resp []datatypes.Virtual_Guest, err error) {
	params := []interface{}{
		filterHostname,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowableVirtualGuests", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume.
func (r Network_Storage) GetAllowedHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedHardware", nil, &r.Options, &resp)
	return
}

// Retrieves the total number of allowed hosts limit per volume.
func (r Network_Storage) GetAllowedHostsLimit() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedHostsLimit", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume.
func (r Network_Storage) GetAllowedIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage) GetAllowedReplicationHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedReplicationHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage) GetAllowedReplicationIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedReplicationIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage) GetAllowedReplicationSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedReplicationSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage) GetAllowedReplicationVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedReplicationVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet objects which are allowed access to this storage volume.
func (r Network_Storage) GetAllowedSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Virtual_Guest objects which are allowed access to this storage volume.
func (r Network_Storage) GetAllowedVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getAllowedVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for a Storage volume.
func (r Network_Storage) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage) GetBillingItemCategory() (resp datatypes.Product_Item_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getBillingItemCategory", nil, &r.Options, &resp)
	return
}

// Retrieve network storage accounts by username and storage account type. Use this method if you wish to retrieve a storage record by username rather than by id. The ''type'' parameter must correspond to one of the available ''nasType'' values in the SoftLayer_Network_Storage data type.
func (r Network_Storage) GetByUsername(username *string, typ *string) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		username,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getByUsername", params, &r.Options, &resp)
	return
}

// Retrieve The amount of space used by the volume, in bytes.
func (r Network_Storage) GetBytesUsed() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getBytesUsed", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetCdnUrls() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_ContentDeliveryUrl, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getCdnUrls", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetClusterResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getClusterResource", nil, &r.Options, &resp)
	return
}

// Retrieve The schedule id which was executed to create a snapshot.
func (r Network_Storage) GetCreationScheduleId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getCreationScheduleId", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage) GetCredentials() (resp []datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getCredentials", nil, &r.Options, &resp)
	return
}

// Retrieve The Daily Schedule which is associated with this network storage volume.
func (r Network_Storage) GetDailySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getDailySchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The events which have taken place on a network storage volume.
func (r Network_Storage) GetEvents() (resp []datatypes.Network_Storage_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getEvents", nil, &r.Options, &resp)
	return
}

//
//
//
func (r Network_Storage) GetFileBlockEncryptedLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getFileBlockEncryptedLocations", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date of a file within a Storage account. This does not download file content.
func (r Network_Storage) GetFileByIdentifier(identifier *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		identifier,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getFileByIdentifier", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the file number of files in a Virtual Server Storage account's root directory. This does not include the files stored in the recycle bin.
func (r Network_Storage) GetFileCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getFileCount", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetFileList(folder *string, path *string) (resp []datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		folder,
		path,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getFileList", params, &r.Options, &resp)
	return
}

// Retrieve Retrieves the NFS Network Mount Address Name for a given File Storage Volume.
func (r Network_Storage) GetFileNetworkMountAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getFileNetworkMountAddress", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the number of files pending deletion in a Storage account's recycle bin. Files in an account's recycle bin may either be restored to the account's root directory or permanently deleted.
func (r Network_Storage) GetFilePendingDeleteCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getFilePendingDeleteCount", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve a list of files that are pending deletion in a Storage account's recycle bin. Files in an account's recycle bin may either be restored to the account's root directory or permanently deleted. This method does not download file content.
func (r Network_Storage) GetFilesPendingDelete() (resp []datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getFilesPendingDelete", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetFolderList() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_Folder, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getFolderList", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// getGraph() retrieves a Storage account's usage and returns a PNG graph image, title, and the minimum and maximum dates included in the graphed date range. Virtual Server storage accounts can also graph upload and download bandwidth usage.
func (r Network_Storage) GetGraph(startDate *datatypes.Time, endDate *datatypes.Time, typ *string) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		startDate,
		endDate,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getGraph", params, &r.Options, &resp)
	return
}

// Retrieve When applicable, the hardware associated with a Storage service.
func (r Network_Storage) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage) GetHasEncryptionAtRest() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getHasEncryptionAtRest", nil, &r.Options, &resp)
	return
}

// Retrieve The Hourly Schedule which is associated with this network storage volume.
func (r Network_Storage) GetHourlySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getHourlySchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The maximum number of IOPs selected for this volume.
func (r Network_Storage) GetIops() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getIops", nil, &r.Options, &resp)
	return
}

// Retrieve Relationship between a container volume and iSCSI LUNs.
func (r Network_Storage) GetIscsiLuns() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getIscsiLuns", nil, &r.Options, &resp)
	return
}

// Retrieve The ID of the LUN volume.
func (r Network_Storage) GetLunId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getLunId", nil, &r.Options, &resp)
	return
}

// Retrieve The manually-created snapshots associated with this SoftLayer_Network_Storage volume. Does not support pagination by result limit and offset.
func (r Network_Storage) GetManualSnapshots() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getManualSnapshots", nil, &r.Options, &resp)
	return
}

// Retrieve A network storage volume's metric tracking object. This object records all periodic polled data available to this volume.
func (r Network_Storage) GetMetricTrackingObject() (resp datatypes.Metric_Tracking_Object, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getMetricTrackingObject", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a network storage volume may be mounted.
func (r Network_Storage) GetMountableFlag() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getMountableFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetNetworkConnectionDetails() (resp datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetNetworkMountAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getNetworkMountAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The subscribers that will be notified for usage amount warnings and overages.
func (r Network_Storage) GetNotificationSubscribers() (resp []datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getNotificationSubscribers", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Storage object whose ID corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Storage service.
//
// Please use the associated methods in the [[SoftLayer_Network_Storage]] service to retrieve a Storage account's id.
func (r Network_Storage) GetObject() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetObjectStorageConnectionInformation() (resp []datatypes.Container_Network_Service_Resource_ObjectStorage_ConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getObjectStorageConnectionInformation", nil, &r.Options, &resp)
	return
}

// Retrieve network storage accounts by SoftLayer_Network_Storage_Credential object. Use this method if you wish to retrieve a storage record by a credential rather than by id.
func (r Network_Storage) GetObjectsByCredential(credentialObject *datatypes.Network_Storage_Credential) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		credentialObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getObjectsByCredential", params, &r.Options, &resp)
	return
}

// Retrieve A volume's configured SoftLayer_Network_Storage_Iscsi_OS_Type.
func (r Network_Storage) GetOsType() (resp datatypes.Network_Storage_Iscsi_OS_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getOsType", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's configured SoftLayer_Network_Storage_Iscsi_OS_Type ID.
func (r Network_Storage) GetOsTypeId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getOsTypeId", nil, &r.Options, &resp)
	return
}

// Retrieve The volumes or snapshots partnered with a network storage volume in a parental role.
func (r Network_Storage) GetParentPartnerships() (resp []datatypes.Network_Storage_Partnership, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getParentPartnerships", nil, &r.Options, &resp)
	return
}

// Retrieve The parent volume of a volume in a complex storage relationship.
func (r Network_Storage) GetParentVolume() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getParentVolume", nil, &r.Options, &resp)
	return
}

// Retrieve The volumes or snapshots partnered with a network storage volume.
func (r Network_Storage) GetPartnerships() (resp []datatypes.Network_Storage_Partnership, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getPartnerships", nil, &r.Options, &resp)
	return
}

// Retrieve All permissions group(s) this volume is in.
func (r Network_Storage) GetPermissionsGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getPermissionsGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The properties used to provide additional details about a network storage volume.
func (r Network_Storage) GetProperties() (resp []datatypes.Network_Storage_Property, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getProperties", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the details of a file that is pending deletion in a Storage account's a recycle bin.
func (r Network_Storage) GetRecycleBinFileByIdentifier(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getRecycleBinFileByIdentifier", params, &r.Options, &resp)
	return
}

// Retrieves the remaining number of allowed hosts per volume.
func (r Network_Storage) GetRemainingAllowedHosts() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getRemainingAllowedHosts", nil, &r.Options, &resp)
	return
}

// Retrieve The iSCSI LUN volumes being replicated by this network storage volume.
func (r Network_Storage) GetReplicatingLuns() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getReplicatingLuns", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volume being replicated by a volume.
func (r Network_Storage) GetReplicatingVolume() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getReplicatingVolume", nil, &r.Options, &resp)
	return
}

// Retrieve The volume replication events.
func (r Network_Storage) GetReplicationEvents() (resp []datatypes.Network_Storage_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getReplicationEvents", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volumes configured to be replicants of a volume.
func (r Network_Storage) GetReplicationPartners() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getReplicationPartners", nil, &r.Options, &resp)
	return
}

// Retrieve The Replication Schedule associated with a network storage volume.
func (r Network_Storage) GetReplicationSchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getReplicationSchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The current replication status of a network storage volume. Indicates Failover or Failback status.
func (r Network_Storage) GetReplicationStatus() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getReplicationStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The schedules which are associated with a network storage volume.
func (r Network_Storage) GetSchedules() (resp []datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getSchedules", nil, &r.Options, &resp)
	return
}

// Retrieve The network resource a Storage service is connected to.
func (r Network_Storage) GetServiceResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getServiceResource", nil, &r.Options, &resp)
	return
}

// Retrieve The IP address of a Storage resource.
func (r Network_Storage) GetServiceResourceBackendIpAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getServiceResourceBackendIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The name of a Storage's network resource.
func (r Network_Storage) GetServiceResourceName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getServiceResourceName", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's configured snapshot space size.
func (r Network_Storage) GetSnapshotCapacityGb() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getSnapshotCapacityGb", nil, &r.Options, &resp)
	return
}

// Retrieve The creation timestamp of the snapshot on the storage platform.
func (r Network_Storage) GetSnapshotCreationTimestamp() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getSnapshotCreationTimestamp", nil, &r.Options, &resp)
	return
}

// Retrieve The percentage of used snapshot space after which to delete automated snapshots.
func (r Network_Storage) GetSnapshotDeletionThresholdPercentage() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getSnapshotDeletionThresholdPercentage", nil, &r.Options, &resp)
	return
}

// Retrieve The snapshot size in bytes.
func (r Network_Storage) GetSnapshotSizeBytes() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getSnapshotSizeBytes", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's available snapshot reservation space.
func (r Network_Storage) GetSnapshotSpaceAvailable() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getSnapshotSpaceAvailable", nil, &r.Options, &resp)
	return
}

// Retrieve The snapshots associated with this SoftLayer_Network_Storage volume.
func (r Network_Storage) GetSnapshots() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getSnapshots", nil, &r.Options, &resp)
	return
}

// Retrieves a list of snapshots for this SoftLayer_Network_Storage volume. This method works with the result limits and offset to support pagination.
func (r Network_Storage) GetSnapshotsForVolume() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getSnapshotsForVolume", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage) GetStaasVersion() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getStaasVersion", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage groups this volume is attached to.
func (r Network_Storage) GetStorageGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getStorageGroups", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetStorageGroupsNetworkConnectionDetails() (resp []datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getStorageGroupsNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage) GetStorageTierLevel() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getStorageTierLevel", nil, &r.Options, &resp)
	return
}

// Retrieve A description of the Storage object.
func (r Network_Storage) GetStorageType() (resp datatypes.Network_Storage_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getStorageType", nil, &r.Options, &resp)
	return
}

// Retrieve The amount of space used by the volume.
func (r Network_Storage) GetTotalBytesUsed() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getTotalBytesUsed", nil, &r.Options, &resp)
	return
}

// Retrieve The total snapshot retention count of all schedules on this network storage volume.
func (r Network_Storage) GetTotalScheduleSnapshotRetentionCount() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getTotalScheduleSnapshotRetentionCount", nil, &r.Options, &resp)
	return
}

// Retrieve The usage notification for SL Storage services.
func (r Network_Storage) GetUsageNotification() (resp datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getUsageNotification", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) GetValidReplicationTargetDatacenterLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getValidReplicationTargetDatacenterLocations", nil, &r.Options, &resp)
	return
}

// Retrieve The type of network storage service.
func (r Network_Storage) GetVendorName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getVendorName", nil, &r.Options, &resp)
	return
}

// Retrieve When applicable, the virtual guest associated with a Storage service.
func (r Network_Storage) GetVirtualGuest() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getVirtualGuest", nil, &r.Options, &resp)
	return
}

// Retrieve The username and password history for a Storage service.
func (r Network_Storage) GetVolumeHistory() (resp []datatypes.Network_Storage_History, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getVolumeHistory", nil, &r.Options, &resp)
	return
}

// Retrieve The current status of a network storage volume.
func (r Network_Storage) GetVolumeStatus() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getVolumeStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The account username and password for the EVault webCC interface.
func (r Network_Storage) GetWebccAccount() (resp datatypes.Account_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getWebccAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The Weekly Schedule which is associated with this network storage volume.
func (r Network_Storage) GetWeeklySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "getWeeklySchedule", nil, &r.Options, &resp)
	return
}

// Immediate Failover to a volume replicant.  During the time which the replicant is in use the local nas volume will not be available.
func (r Network_Storage) ImmediateFailoverToReplicant(replicantId *int) (resp bool, err error) {
	params := []interface{}{
		replicantId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "immediateFailoverToReplicant", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) IsBlockingOperationInProgress(exemptStatusKeyNames []string) (resp bool, err error) {
	params := []interface{}{
		exemptStatusKeyNames,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "isBlockingOperationInProgress", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage) RemoveAccessFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromHardware", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage) RemoveAccessFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage) RemoveAccessFromHost(typeClassName *string, hostId *int) (resp datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		typeClassName,
		hostId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromHost", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage) RemoveAccessFromHostList(hostObjectTemplates []datatypes.Container_Network_Storage_Host) (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		hostObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromHostList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage will be listed in the allowedIpAddresses property of this storage volume.
func (r Network_Storage) RemoveAccessFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromIpAddress", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) RemoveAccessFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromIpAddressList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) RemoveAccessFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromSubnet", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) RemoveAccessFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage) RemoveAccessFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage) RemoveAccessFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Hardware objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationHardware property of this storage volume.
func (r Network_Storage) RemoveAccessToReplicantFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessToReplicantFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationIpAddresses property of this storage volume.
func (r Network_Storage) RemoveAccessToReplicantFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessToReplicantFromIpAddressList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) RemoveAccessToReplicantFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessToReplicantFromSubnet", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationSubnets property of this storage volume.
func (r Network_Storage) RemoveAccessToReplicantFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessToReplicantFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationVirtualGuests property of this storage volume.
func (r Network_Storage) RemoveAccessToReplicantFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeAccessToReplicantFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method will remove a credential from the current volume. The credential must have been created using the 'addNewCredential' method.
func (r Network_Storage) RemoveCredential(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "removeCredential", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Restore an individual file so that it may be used as it was before it was deleted.
//
// If a file is deleted from a Virtual Server Storage account, the file is placed into the account's recycle bin and not permanently deleted. Therefore, restoreFile can be used to place the file back into your Virtual Server account's root directory.
func (r Network_Storage) RestoreFile(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "restoreFile", params, &r.Options, &resp)
	return
}

// Restore the volume from a snapshot that was previously taken.
func (r Network_Storage) RestoreFromSnapshot(snapshotId *int) (resp bool, err error) {
	params := []interface{}{
		snapshotId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "restoreFromSnapshot", params, &r.Options, &resp)
	return
}

// The method will retrieve the password for the StorageLayer or Virtual Server Storage Account and email the password.  The Storage Account passwords will be emailed to the master user.  For Virtual Server Storage, the password will be sent to the email address used as the username.
func (r Network_Storage) SendPasswordReminderEmail(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "sendPasswordReminderEmail", params, &r.Options, &resp)
	return
}

// Enable or disable the mounting of a Storage volume. When mounting is enabled the Storage volume will be mountable or available for use.
//
// For Virtual Server volumes, disabling mounting will deny access to the Virtual Server Account, remove published material and deny all file interaction including uploads and downloads.
//
// Enabling or disabling mounting for Storage volumes is not possible if mounting has been disabled by SoftLayer or a parent account.
func (r Network_Storage) SetMountable(mountable *bool) (resp bool, err error) {
	params := []interface{}{
		mountable,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "setMountable", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage) SetSnapshotAllocation(capacityGb *int) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		capacityGb,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "setSnapshotAllocation", params, &r.Options, &resp)
	return
}

// Upgrade the Storage volume to one of the upgradable packages (for example from 10 Gigs of EVault storage to 100 Gigs of EVault storage).
func (r Network_Storage) UpgradeVolumeCapacity(itemId *int) (resp bool, err error) {
	params := []interface{}{
		itemId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "upgradeVolumeCapacity", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Upload a file to a Storage account's root directory. Once uploaded, this method returns new file entity identifier for the upload file.
//
// The following properties are required in the ''file'' parameter.
// *'''name''': The name of the file you wish to upload
// *'''content''': The raw contents of the file you wish to upload.
// *'''contentType''': The MIME-type of content that you wish to upload.
func (r Network_Storage) UploadFile(file *datatypes.Container_Utility_File_Entity) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		file,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage", "uploadFile", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Allowed_Host struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageAllowedHostService returns an instance of the Network_Storage_Allowed_Host SoftLayer service
func GetNetworkStorageAllowedHostService(sess *session.Session) Network_Storage_Allowed_Host {
	return Network_Storage_Allowed_Host{Session: sess}
}

func (r Network_Storage_Allowed_Host) Id(id int) Network_Storage_Allowed_Host {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Allowed_Host) Mask(mask string) Network_Storage_Allowed_Host {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Allowed_Host) Filter(filter string) Network_Storage_Allowed_Host {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Allowed_Host) Limit(limit int) Network_Storage_Allowed_Host {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Allowed_Host) Offset(offset int) Network_Storage_Allowed_Host {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Storage_Allowed_Host) CreateObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host) EditObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Group objects this SoftLayer_Network_Storage_Allowed_Host is present in.
func (r Network_Storage_Allowed_Host) GetAssignedGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "getAssignedGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage primary volumes whose replicas are allowed access.
func (r Network_Storage_Allowed_Host) GetAssignedReplicationVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "getAssignedReplicationVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage volumes to which this SoftLayer_Network_Storage_Allowed_Host is allowed access.
func (r Network_Storage_Allowed_Host) GetAssignedVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "getAssignedVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Credential this allowed host uses.
func (r Network_Storage_Allowed_Host) GetCredential() (resp datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "getCredential", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host) GetObject() (resp datatypes.Network_Storage_Allowed_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "getObject", nil, &r.Options, &resp)
	return
}

// Use this method to modify the credential password for a SoftLayer_Network_Storage_Allowed_Host object.
func (r Network_Storage_Allowed_Host) SetCredentialPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host", "setCredentialPassword", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Allowed_Host_Hardware struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageAllowedHostHardwareService returns an instance of the Network_Storage_Allowed_Host_Hardware SoftLayer service
func GetNetworkStorageAllowedHostHardwareService(sess *session.Session) Network_Storage_Allowed_Host_Hardware {
	return Network_Storage_Allowed_Host_Hardware{Session: sess}
}

func (r Network_Storage_Allowed_Host_Hardware) Id(id int) Network_Storage_Allowed_Host_Hardware {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Allowed_Host_Hardware) Mask(mask string) Network_Storage_Allowed_Host_Hardware {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Allowed_Host_Hardware) Filter(filter string) Network_Storage_Allowed_Host_Hardware {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Allowed_Host_Hardware) Limit(limit int) Network_Storage_Allowed_Host_Hardware {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Allowed_Host_Hardware) Offset(offset int) Network_Storage_Allowed_Host_Hardware {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Storage_Allowed_Host_Hardware) CreateObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_Hardware) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_Hardware) EditObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Group objects this SoftLayer_Network_Storage_Allowed_Host is present in.
func (r Network_Storage_Allowed_Host_Hardware) GetAssignedGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "getAssignedGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage primary volumes whose replicas are allowed access.
func (r Network_Storage_Allowed_Host_Hardware) GetAssignedReplicationVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "getAssignedReplicationVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage volumes to which this SoftLayer_Network_Storage_Allowed_Host is allowed access.
func (r Network_Storage_Allowed_Host_Hardware) GetAssignedVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "getAssignedVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Credential this allowed host uses.
func (r Network_Storage_Allowed_Host_Hardware) GetCredential() (resp datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "getCredential", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_Hardware) GetObject() (resp datatypes.Network_Storage_Allowed_Host_Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware object which this SoftLayer_Network_Storage_Allowed_Host is referencing.
func (r Network_Storage_Allowed_Host_Hardware) GetResource() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "getResource", nil, &r.Options, &resp)
	return
}

// Use this method to modify the credential password for a SoftLayer_Network_Storage_Allowed_Host object.
func (r Network_Storage_Allowed_Host_Hardware) SetCredentialPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Hardware", "setCredentialPassword", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Allowed_Host_IpAddress struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageAllowedHostIpAddressService returns an instance of the Network_Storage_Allowed_Host_IpAddress SoftLayer service
func GetNetworkStorageAllowedHostIpAddressService(sess *session.Session) Network_Storage_Allowed_Host_IpAddress {
	return Network_Storage_Allowed_Host_IpAddress{Session: sess}
}

func (r Network_Storage_Allowed_Host_IpAddress) Id(id int) Network_Storage_Allowed_Host_IpAddress {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Allowed_Host_IpAddress) Mask(mask string) Network_Storage_Allowed_Host_IpAddress {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Allowed_Host_IpAddress) Filter(filter string) Network_Storage_Allowed_Host_IpAddress {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Allowed_Host_IpAddress) Limit(limit int) Network_Storage_Allowed_Host_IpAddress {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Allowed_Host_IpAddress) Offset(offset int) Network_Storage_Allowed_Host_IpAddress {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Storage_Allowed_Host_IpAddress) CreateObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_IpAddress) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_IpAddress) EditObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Group objects this SoftLayer_Network_Storage_Allowed_Host is present in.
func (r Network_Storage_Allowed_Host_IpAddress) GetAssignedGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "getAssignedGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage primary volumes whose replicas are allowed access.
func (r Network_Storage_Allowed_Host_IpAddress) GetAssignedReplicationVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "getAssignedReplicationVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage volumes to which this SoftLayer_Network_Storage_Allowed_Host is allowed access.
func (r Network_Storage_Allowed_Host_IpAddress) GetAssignedVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "getAssignedVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Credential this allowed host uses.
func (r Network_Storage_Allowed_Host_IpAddress) GetCredential() (resp datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "getCredential", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_IpAddress) GetObject() (resp datatypes.Network_Storage_Allowed_Host_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet_IpAddress object which this SoftLayer_Network_Storage_Allowed_Host is referencing.
func (r Network_Storage_Allowed_Host_IpAddress) GetResource() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "getResource", nil, &r.Options, &resp)
	return
}

// Use this method to modify the credential password for a SoftLayer_Network_Storage_Allowed_Host object.
func (r Network_Storage_Allowed_Host_IpAddress) SetCredentialPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_IpAddress", "setCredentialPassword", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Allowed_Host_Subnet struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageAllowedHostSubnetService returns an instance of the Network_Storage_Allowed_Host_Subnet SoftLayer service
func GetNetworkStorageAllowedHostSubnetService(sess *session.Session) Network_Storage_Allowed_Host_Subnet {
	return Network_Storage_Allowed_Host_Subnet{Session: sess}
}

func (r Network_Storage_Allowed_Host_Subnet) Id(id int) Network_Storage_Allowed_Host_Subnet {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Allowed_Host_Subnet) Mask(mask string) Network_Storage_Allowed_Host_Subnet {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Allowed_Host_Subnet) Filter(filter string) Network_Storage_Allowed_Host_Subnet {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Allowed_Host_Subnet) Limit(limit int) Network_Storage_Allowed_Host_Subnet {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Allowed_Host_Subnet) Offset(offset int) Network_Storage_Allowed_Host_Subnet {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Storage_Allowed_Host_Subnet) CreateObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_Subnet) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_Subnet) EditObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Group objects this SoftLayer_Network_Storage_Allowed_Host is present in.
func (r Network_Storage_Allowed_Host_Subnet) GetAssignedGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "getAssignedGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage primary volumes whose replicas are allowed access.
func (r Network_Storage_Allowed_Host_Subnet) GetAssignedReplicationVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "getAssignedReplicationVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage volumes to which this SoftLayer_Network_Storage_Allowed_Host is allowed access.
func (r Network_Storage_Allowed_Host_Subnet) GetAssignedVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "getAssignedVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Credential this allowed host uses.
func (r Network_Storage_Allowed_Host_Subnet) GetCredential() (resp datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "getCredential", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_Subnet) GetObject() (resp datatypes.Network_Storage_Allowed_Host_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet object which this SoftLayer_Network_Storage_Allowed_Host is referencing.
func (r Network_Storage_Allowed_Host_Subnet) GetResource() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "getResource", nil, &r.Options, &resp)
	return
}

// Use this method to modify the credential password for a SoftLayer_Network_Storage_Allowed_Host object.
func (r Network_Storage_Allowed_Host_Subnet) SetCredentialPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_Subnet", "setCredentialPassword", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Allowed_Host_VirtualGuest struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageAllowedHostVirtualGuestService returns an instance of the Network_Storage_Allowed_Host_VirtualGuest SoftLayer service
func GetNetworkStorageAllowedHostVirtualGuestService(sess *session.Session) Network_Storage_Allowed_Host_VirtualGuest {
	return Network_Storage_Allowed_Host_VirtualGuest{Session: sess}
}

func (r Network_Storage_Allowed_Host_VirtualGuest) Id(id int) Network_Storage_Allowed_Host_VirtualGuest {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Allowed_Host_VirtualGuest) Mask(mask string) Network_Storage_Allowed_Host_VirtualGuest {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Allowed_Host_VirtualGuest) Filter(filter string) Network_Storage_Allowed_Host_VirtualGuest {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Allowed_Host_VirtualGuest) Limit(limit int) Network_Storage_Allowed_Host_VirtualGuest {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Allowed_Host_VirtualGuest) Offset(offset int) Network_Storage_Allowed_Host_VirtualGuest {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Storage_Allowed_Host_VirtualGuest) CreateObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_VirtualGuest) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_VirtualGuest) EditObject(templateObject *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Group objects this SoftLayer_Network_Storage_Allowed_Host is present in.
func (r Network_Storage_Allowed_Host_VirtualGuest) GetAssignedGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "getAssignedGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage primary volumes whose replicas are allowed access.
func (r Network_Storage_Allowed_Host_VirtualGuest) GetAssignedReplicationVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "getAssignedReplicationVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage volumes to which this SoftLayer_Network_Storage_Allowed_Host is allowed access.
func (r Network_Storage_Allowed_Host_VirtualGuest) GetAssignedVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "getAssignedVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Credential this allowed host uses.
func (r Network_Storage_Allowed_Host_VirtualGuest) GetCredential() (resp datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "getCredential", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Allowed_Host_VirtualGuest) GetObject() (resp datatypes.Network_Storage_Allowed_Host_VirtualGuest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Virtual_Guest object which this SoftLayer_Network_Storage_Allowed_Host is referencing.
func (r Network_Storage_Allowed_Host_VirtualGuest) GetResource() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "getResource", nil, &r.Options, &resp)
	return
}

// Use this method to modify the credential password for a SoftLayer_Network_Storage_Allowed_Host object.
func (r Network_Storage_Allowed_Host_VirtualGuest) SetCredentialPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Allowed_Host_VirtualGuest", "setCredentialPassword", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Storage_Backup_Evault contains general information regarding an EVault Storage service such as account id, username, maximum capacity, password, Storage's product type and the server id.
type Network_Storage_Backup_Evault struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageBackupEvaultService returns an instance of the Network_Storage_Backup_Evault SoftLayer service
func GetNetworkStorageBackupEvaultService(sess *session.Session) Network_Storage_Backup_Evault {
	return Network_Storage_Backup_Evault{Session: sess}
}

func (r Network_Storage_Backup_Evault) Id(id int) Network_Storage_Backup_Evault {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Backup_Evault) Mask(mask string) Network_Storage_Backup_Evault {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Backup_Evault) Filter(filter string) Network_Storage_Backup_Evault {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Backup_Evault) Limit(limit int) Network_Storage_Backup_Evault {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Backup_Evault) Offset(offset int) Network_Storage_Backup_Evault {
	r.Options.Offset = &offset
	return r
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromHardware", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) AllowAccessFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessFromHost(typeClassName *string, hostId *int) (resp datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		typeClassName,
		hostId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromHost", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage volume will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessFromHostList(hostObjectTemplates []datatypes.Container_Network_Storage_Host) (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		hostObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromHostList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage will be listed in the allowedIpAddresses property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromIpAddress", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) AllowAccessFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromIpAddressList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromSubnet", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) AllowAccessFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage replicant volume.
func (r Network_Storage_Backup_Evault) AllowAccessToReplicantFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessToReplicantFromHardware", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Hardware objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationHardware property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessToReplicantFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessToReplicantFromHardwareList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) AllowAccessToReplicantFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessToReplicantFromIpAddress", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationIpAddresses property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessToReplicantFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessToReplicantFromIpAddressList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage replicant volume.
func (r Network_Storage_Backup_Evault) AllowAccessToReplicantFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessToReplicantFromSubnet", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationSubnets property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessToReplicantFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessToReplicantFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage replicant volume.
func (r Network_Storage_Backup_Evault) AllowAccessToReplicantFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessToReplicantFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationVirtualGuests property of this storage volume.
func (r Network_Storage_Backup_Evault) AllowAccessToReplicantFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "allowAccessToReplicantFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method will assign an existing credential to the current volume. The credential must have been created using the 'addNewCredential' method. The volume type must support an additional credential.
func (r Network_Storage_Backup_Evault) AssignCredential(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "assignCredential", params, &r.Options, &resp)
	return
}

// This method will set up a new credential for the remote storage volume. The storage volume must support an additional credential. Once created, the credential will be automatically assigned to the current volume. If there are no volumes assigned to the credential it will be automatically deleted.
func (r Network_Storage_Backup_Evault) AssignNewCredential(typ *string) (resp datatypes.Network_Storage_Credential, err error) {
	params := []interface{}{
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "assignNewCredential", params, &r.Options, &resp)
	return
}

// The method will change the password for the given Storage/Virtual Server Storage account.
func (r Network_Storage_Backup_Evault) ChangePassword(username *string, currentPassword *string, newPassword *string) (resp bool, err error) {
	params := []interface{}{
		username,
		currentPassword,
		newPassword,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "changePassword", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// collectBandwidth() Retrieve the bandwidth usage for the current billing cycle.
func (r Network_Storage_Backup_Evault) CollectBandwidth(typ *string, startDate *datatypes.Time, endDate *datatypes.Time) (resp uint, err error) {
	params := []interface{}{
		typ,
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "collectBandwidth", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// collectBytesUsed() retrieves the number of bytes capacity currently in use on a Storage account.
func (r Network_Storage_Backup_Evault) CollectBytesUsed() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "collectBytesUsed", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) CreateFolder(folder *string) (resp bool, err error) {
	params := []interface{}{
		folder,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "createFolder", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) CreateSnapshot(notes *string) (resp datatypes.Network_Storage, err error) {
	params := []interface{}{
		notes,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "createSnapshot", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete all files within a Storage account. Depending on the type of Storage account, Deleting either deletes files permanently or sends files to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the files are in the account's recycle bin. If the files exist in the recycle bin, then they are permanently deleted.
//
// Please note, files can not be restored once they are permanently deleted.
func (r Network_Storage_Backup_Evault) DeleteAllFiles() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "deleteAllFiles", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete an individual file within a Storage account. Depending on the type of Storage account, Deleting a file either deletes the file permanently or sends the file to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the file is in the account's recycle bin. If the file exist in the recycle bin, then it is permanently deleted.
//
// Please note, a file can not be restored once it is permanently deleted.
func (r Network_Storage_Backup_Evault) DeleteFile(fileId *string) (resp bool, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "deleteFile", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete multiple files within a Storage account. Depending on the type of Storage account, Deleting either deletes files permanently or sends files to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the files are in the account's recycle bin. If the files exist in the recycle bin, then they are permanently deleted.
//
// Please note, files can not be restored once they are permanently deleted.
func (r Network_Storage_Backup_Evault) DeleteFiles(fileIds []string) (resp bool, err error) {
	params := []interface{}{
		fileIds,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "deleteFiles", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) DeleteFolder(folder *string) (resp bool, err error) {
	params := []interface{}{
		folder,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "deleteFolder", params, &r.Options, &resp)
	return
}

// Delete a network storage volume. '''This cannot be undone.''' At this time only network storage snapshots may be deleted with this method.
//
// ''deleteObject'' returns Boolean ''true'' on successful deletion or ''false'' if it was unable to remove a volume;
func (r Network_Storage_Backup_Evault) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method can be used to help maintain the storage space on a vault.  When a job is removed from the Webcc, the task and stored usage still exists on the vault. This method can be used to delete the associated task and its usage.
//
// All that is required for the use of the method is to pass in an integer array of task(s).
//
//
func (r Network_Storage_Backup_Evault) DeleteTasks(tasks []int) (resp bool, err error) {
	params := []interface{}{
		tasks,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "deleteTasks", params, &r.Options, &resp)
	return
}

// This method is not valid for Legacy iSCSI Storage Volumes.
//
// Disable scheduled snapshots of this storage volume. Scheduling options include HOURLY, DAILY and WEEKLY schedules.
func (r Network_Storage_Backup_Evault) DisableSnapshots(scheduleType *string) (resp bool, err error) {
	params := []interface{}{
		scheduleType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "disableSnapshots", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Download a file from a Storage account. This method returns a file's details including the file's raw content.
func (r Network_Storage_Backup_Evault) DownloadFile(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "downloadFile", params, &r.Options, &resp)
	return
}

// This method will change the password of a credential created using the 'addNewCredential' method. If the credential exists on multiple storage volumes it will change for those volumes as well.
func (r Network_Storage_Backup_Evault) EditCredential(username *string, newPassword *string) (resp bool, err error) {
	params := []interface{}{
		username,
		newPassword,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "editCredential", params, &r.Options, &resp)
	return
}

// The password and/or notes may be modified for the Storage service except evault passwords and notes.
func (r Network_Storage_Backup_Evault) EditObject(templateObject *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "editObject", params, &r.Options, &resp)
	return
}

// This method is not valid for Legacy iSCSI Storage Volumes.
//
// Enable scheduled snapshots of this storage volume. Scheduling options include HOURLY, DAILY and WEEKLY schedules. For HOURLY schedules, provide relevant data for $scheduleType, $retentionCount and $minute. For DAILY schedules, provide relevant data for $scheduleType, $retentionCount, $minute, and $hour. For WEEKLY schedules, provide relevant data for all parameters of this method.
func (r Network_Storage_Backup_Evault) EnableSnapshots(scheduleType *string, retentionCount *int, minute *int, hour *int, dayOfWeek *string) (resp bool, err error) {
	params := []interface{}{
		scheduleType,
		retentionCount,
		minute,
		hour,
		dayOfWeek,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "enableSnapshots", params, &r.Options, &resp)
	return
}

// Failback from a volume replicant. In order to failback the volume must have already been failed over to a replicant.
func (r Network_Storage_Backup_Evault) FailbackFromReplicant() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "failbackFromReplicant", nil, &r.Options, &resp)
	return
}

// Failover to a volume replicant.  During the time which the replicant is in use the local nas volume will not be available.
func (r Network_Storage_Backup_Evault) FailoverToReplicant(replicantId *int) (resp bool, err error) {
	params := []interface{}{
		replicantId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "failoverToReplicant", params, &r.Options, &resp)
	return
}

// Retrieve The account that a Storage services belongs to.
func (r Network_Storage_Backup_Evault) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve Other usernames and passwords associated with a Storage volume.
func (r Network_Storage_Backup_Evault) GetAccountPassword() (resp datatypes.Account_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAccountPassword", nil, &r.Options, &resp)
	return
}

// Retrieve The currently active transactions on a network storage volume.
func (r Network_Storage_Backup_Evault) GetActiveTransactions() (resp []datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getActiveTransactions", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date for all files in a Storage account's root directory. This does not download file content.
func (r Network_Storage_Backup_Evault) GetAllFiles() (resp []datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllFiles", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date for all files matching the filter's criteria in a Storage account's root directory. This does not download file content.
func (r Network_Storage_Backup_Evault) GetAllFilesByFilter(filter *datatypes.Container_Utility_File_Entity) (resp []datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		filter,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllFilesByFilter", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Hardware that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage_Backup_Evault) GetAllowableHardware(filterHostname *string) (resp []datatypes.Hardware, err error) {
	params := []interface{}{
		filterHostname,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowableHardware", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Network_Subnet_IpAddress that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage_Backup_Evault) GetAllowableIpAddresses(subnetId *int, filterIpAddress *string) (resp []datatypes.Network_Subnet_IpAddress, err error) {
	params := []interface{}{
		subnetId,
		filterIpAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowableIpAddresses", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Network_Subnet that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage_Backup_Evault) GetAllowableSubnets(filterNetworkIdentifier *string) (resp []datatypes.Network_Subnet, err error) {
	params := []interface{}{
		filterNetworkIdentifier,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowableSubnets", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Virtual_Guest that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage_Backup_Evault) GetAllowableVirtualGuests(filterHostname *string) (resp []datatypes.Virtual_Guest, err error) {
	params := []interface{}{
		filterHostname,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowableVirtualGuests", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume.
func (r Network_Storage_Backup_Evault) GetAllowedHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedHardware", nil, &r.Options, &resp)
	return
}

// Retrieves the total number of allowed hosts limit per volume.
func (r Network_Storage_Backup_Evault) GetAllowedHostsLimit() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedHostsLimit", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume.
func (r Network_Storage_Backup_Evault) GetAllowedIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage_Backup_Evault) GetAllowedReplicationHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedReplicationHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage_Backup_Evault) GetAllowedReplicationIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedReplicationIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage_Backup_Evault) GetAllowedReplicationSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedReplicationSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage_Backup_Evault) GetAllowedReplicationVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedReplicationVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet objects which are allowed access to this storage volume.
func (r Network_Storage_Backup_Evault) GetAllowedSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Virtual_Guest objects which are allowed access to this storage volume.
func (r Network_Storage_Backup_Evault) GetAllowedVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getAllowedVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for a Storage volume.
func (r Network_Storage_Backup_Evault) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Backup_Evault) GetBillingItemCategory() (resp datatypes.Product_Item_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getBillingItemCategory", nil, &r.Options, &resp)
	return
}

// Retrieve network storage accounts by username and storage account type. Use this method if you wish to retrieve a storage record by username rather than by id. The ''type'' parameter must correspond to one of the available ''nasType'' values in the SoftLayer_Network_Storage data type.
func (r Network_Storage_Backup_Evault) GetByUsername(username *string, typ *string) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		username,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getByUsername", params, &r.Options, &resp)
	return
}

// Retrieve The amount of space used by the volume, in bytes.
func (r Network_Storage_Backup_Evault) GetBytesUsed() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getBytesUsed", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetCdnUrls() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_ContentDeliveryUrl, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getCdnUrls", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetClusterResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getClusterResource", nil, &r.Options, &resp)
	return
}

// Retrieve The schedule id which was executed to create a snapshot.
func (r Network_Storage_Backup_Evault) GetCreationScheduleId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getCreationScheduleId", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Backup_Evault) GetCredentials() (resp []datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getCredentials", nil, &r.Options, &resp)
	return
}

// Retrieve The Daily Schedule which is associated with this network storage volume.
func (r Network_Storage_Backup_Evault) GetDailySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getDailySchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The events which have taken place on a network storage volume.
func (r Network_Storage_Backup_Evault) GetEvents() (resp []datatypes.Network_Storage_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getEvents", nil, &r.Options, &resp)
	return
}

//
//
//
func (r Network_Storage_Backup_Evault) GetFileBlockEncryptedLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getFileBlockEncryptedLocations", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date of a file within a Storage account. This does not download file content.
func (r Network_Storage_Backup_Evault) GetFileByIdentifier(identifier *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		identifier,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getFileByIdentifier", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the file number of files in a Virtual Server Storage account's root directory. This does not include the files stored in the recycle bin.
func (r Network_Storage_Backup_Evault) GetFileCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getFileCount", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetFileList(folder *string, path *string) (resp []datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		folder,
		path,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getFileList", params, &r.Options, &resp)
	return
}

// Retrieve Retrieves the NFS Network Mount Address Name for a given File Storage Volume.
func (r Network_Storage_Backup_Evault) GetFileNetworkMountAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getFileNetworkMountAddress", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the number of files pending deletion in a Storage account's recycle bin. Files in an account's recycle bin may either be restored to the account's root directory or permanently deleted.
func (r Network_Storage_Backup_Evault) GetFilePendingDeleteCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getFilePendingDeleteCount", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve a list of files that are pending deletion in a Storage account's recycle bin. Files in an account's recycle bin may either be restored to the account's root directory or permanently deleted. This method does not download file content.
func (r Network_Storage_Backup_Evault) GetFilesPendingDelete() (resp []datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getFilesPendingDelete", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetFolderList() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_Folder, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getFolderList", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// getGraph() retrieves a Storage account's usage and returns a PNG graph image, title, and the minimum and maximum dates included in the graphed date range. Virtual Server storage accounts can also graph upload and download bandwidth usage.
func (r Network_Storage_Backup_Evault) GetGraph(startDate *datatypes.Time, endDate *datatypes.Time, typ *string) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		startDate,
		endDate,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getGraph", params, &r.Options, &resp)
	return
}

// Retrieve When applicable, the hardware associated with a Storage service.
func (r Network_Storage_Backup_Evault) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve a list of hardware associated with a SoftLayer customer account, placing all hardware with associated EVault storage accounts at the beginning of the list. The return type is SoftLayer_Hardware_Server[] contains the results; the number of items returned in the result will be returned in the soap header (totalItems). ''getHardwareWithEvaultFirst'' is useful in situations where you wish to search for hardware and provide paginated output.
//
//
//
//
//
// Results are only returned for hardware belonging to the account of the user making the API call.
//
// This method drives the backup page of the SoftLayer customer portal. It serves a very specific function, but we have exposed it as it may prove useful for API developers too.
func (r Network_Storage_Backup_Evault) GetHardwareWithEvaultFirst(option *string, exactMatch *bool, criteria *string, mode *string) (resp []datatypes.Hardware, err error) {
	params := []interface{}{
		option,
		exactMatch,
		criteria,
		mode,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getHardwareWithEvaultFirst", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Backup_Evault) GetHasEncryptionAtRest() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getHasEncryptionAtRest", nil, &r.Options, &resp)
	return
}

// Retrieve The Hourly Schedule which is associated with this network storage volume.
func (r Network_Storage_Backup_Evault) GetHourlySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getHourlySchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The maximum number of IOPs selected for this volume.
func (r Network_Storage_Backup_Evault) GetIops() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getIops", nil, &r.Options, &resp)
	return
}

// Retrieve Relationship between a container volume and iSCSI LUNs.
func (r Network_Storage_Backup_Evault) GetIscsiLuns() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getIscsiLuns", nil, &r.Options, &resp)
	return
}

// Retrieve The ID of the LUN volume.
func (r Network_Storage_Backup_Evault) GetLunId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getLunId", nil, &r.Options, &resp)
	return
}

// Retrieve The manually-created snapshots associated with this SoftLayer_Network_Storage volume. Does not support pagination by result limit and offset.
func (r Network_Storage_Backup_Evault) GetManualSnapshots() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getManualSnapshots", nil, &r.Options, &resp)
	return
}

// Retrieve A network storage volume's metric tracking object. This object records all periodic polled data available to this volume.
func (r Network_Storage_Backup_Evault) GetMetricTrackingObject() (resp datatypes.Metric_Tracking_Object, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getMetricTrackingObject", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a network storage volume may be mounted.
func (r Network_Storage_Backup_Evault) GetMountableFlag() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getMountableFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetNetworkConnectionDetails() (resp datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetNetworkMountAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getNetworkMountAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The subscribers that will be notified for usage amount warnings and overages.
func (r Network_Storage_Backup_Evault) GetNotificationSubscribers() (resp []datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getNotificationSubscribers", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Storage_Backup_Evault object whose ID corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Storage_Backup_Evault service.
func (r Network_Storage_Backup_Evault) GetObject() (resp datatypes.Network_Storage_Backup_Evault, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetObjectStorageConnectionInformation() (resp []datatypes.Container_Network_Service_Resource_ObjectStorage_ConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getObjectStorageConnectionInformation", nil, &r.Options, &resp)
	return
}

// Retrieve network storage accounts by SoftLayer_Network_Storage_Credential object. Use this method if you wish to retrieve a storage record by a credential rather than by id.
func (r Network_Storage_Backup_Evault) GetObjectsByCredential(credentialObject *datatypes.Network_Storage_Credential) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		credentialObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getObjectsByCredential", params, &r.Options, &resp)
	return
}

// Retrieve A volume's configured SoftLayer_Network_Storage_Iscsi_OS_Type.
func (r Network_Storage_Backup_Evault) GetOsType() (resp datatypes.Network_Storage_Iscsi_OS_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getOsType", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's configured SoftLayer_Network_Storage_Iscsi_OS_Type ID.
func (r Network_Storage_Backup_Evault) GetOsTypeId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getOsTypeId", nil, &r.Options, &resp)
	return
}

// Retrieve The volumes or snapshots partnered with a network storage volume in a parental role.
func (r Network_Storage_Backup_Evault) GetParentPartnerships() (resp []datatypes.Network_Storage_Partnership, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getParentPartnerships", nil, &r.Options, &resp)
	return
}

// Retrieve The parent volume of a volume in a complex storage relationship.
func (r Network_Storage_Backup_Evault) GetParentVolume() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getParentVolume", nil, &r.Options, &resp)
	return
}

// Retrieve The volumes or snapshots partnered with a network storage volume.
func (r Network_Storage_Backup_Evault) GetPartnerships() (resp []datatypes.Network_Storage_Partnership, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getPartnerships", nil, &r.Options, &resp)
	return
}

// Retrieve All permissions group(s) this volume is in.
func (r Network_Storage_Backup_Evault) GetPermissionsGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getPermissionsGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The properties used to provide additional details about a network storage volume.
func (r Network_Storage_Backup_Evault) GetProperties() (resp []datatypes.Network_Storage_Property, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getProperties", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the details of a file that is pending deletion in a Storage account's a recycle bin.
func (r Network_Storage_Backup_Evault) GetRecycleBinFileByIdentifier(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getRecycleBinFileByIdentifier", params, &r.Options, &resp)
	return
}

// Retrieves the remaining number of allowed hosts per volume.
func (r Network_Storage_Backup_Evault) GetRemainingAllowedHosts() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getRemainingAllowedHosts", nil, &r.Options, &resp)
	return
}

// Retrieve The iSCSI LUN volumes being replicated by this network storage volume.
func (r Network_Storage_Backup_Evault) GetReplicatingLuns() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getReplicatingLuns", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volume being replicated by a volume.
func (r Network_Storage_Backup_Evault) GetReplicatingVolume() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getReplicatingVolume", nil, &r.Options, &resp)
	return
}

// Retrieve The volume replication events.
func (r Network_Storage_Backup_Evault) GetReplicationEvents() (resp []datatypes.Network_Storage_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getReplicationEvents", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volumes configured to be replicants of a volume.
func (r Network_Storage_Backup_Evault) GetReplicationPartners() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getReplicationPartners", nil, &r.Options, &resp)
	return
}

// Retrieve The Replication Schedule associated with a network storage volume.
func (r Network_Storage_Backup_Evault) GetReplicationSchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getReplicationSchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The current replication status of a network storage volume. Indicates Failover or Failback status.
func (r Network_Storage_Backup_Evault) GetReplicationStatus() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getReplicationStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The schedules which are associated with a network storage volume.
func (r Network_Storage_Backup_Evault) GetSchedules() (resp []datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getSchedules", nil, &r.Options, &resp)
	return
}

// Retrieve The network resource a Storage service is connected to.
func (r Network_Storage_Backup_Evault) GetServiceResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getServiceResource", nil, &r.Options, &resp)
	return
}

// Retrieve The IP address of a Storage resource.
func (r Network_Storage_Backup_Evault) GetServiceResourceBackendIpAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getServiceResourceBackendIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The name of a Storage's network resource.
func (r Network_Storage_Backup_Evault) GetServiceResourceName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getServiceResourceName", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's configured snapshot space size.
func (r Network_Storage_Backup_Evault) GetSnapshotCapacityGb() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getSnapshotCapacityGb", nil, &r.Options, &resp)
	return
}

// Retrieve The creation timestamp of the snapshot on the storage platform.
func (r Network_Storage_Backup_Evault) GetSnapshotCreationTimestamp() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getSnapshotCreationTimestamp", nil, &r.Options, &resp)
	return
}

// Retrieve The percentage of used snapshot space after which to delete automated snapshots.
func (r Network_Storage_Backup_Evault) GetSnapshotDeletionThresholdPercentage() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getSnapshotDeletionThresholdPercentage", nil, &r.Options, &resp)
	return
}

// Retrieve The snapshot size in bytes.
func (r Network_Storage_Backup_Evault) GetSnapshotSizeBytes() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getSnapshotSizeBytes", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's available snapshot reservation space.
func (r Network_Storage_Backup_Evault) GetSnapshotSpaceAvailable() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getSnapshotSpaceAvailable", nil, &r.Options, &resp)
	return
}

// Retrieve The snapshots associated with this SoftLayer_Network_Storage volume.
func (r Network_Storage_Backup_Evault) GetSnapshots() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getSnapshots", nil, &r.Options, &resp)
	return
}

// Retrieves a list of snapshots for this SoftLayer_Network_Storage volume. This method works with the result limits and offset to support pagination.
func (r Network_Storage_Backup_Evault) GetSnapshotsForVolume() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getSnapshotsForVolume", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Backup_Evault) GetStaasVersion() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getStaasVersion", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage groups this volume is attached to.
func (r Network_Storage_Backup_Evault) GetStorageGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getStorageGroups", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetStorageGroupsNetworkConnectionDetails() (resp []datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getStorageGroupsNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Backup_Evault) GetStorageTierLevel() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getStorageTierLevel", nil, &r.Options, &resp)
	return
}

// Retrieve A description of the Storage object.
func (r Network_Storage_Backup_Evault) GetStorageType() (resp datatypes.Network_Storage_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getStorageType", nil, &r.Options, &resp)
	return
}

// Retrieve The amount of space used by the volume.
func (r Network_Storage_Backup_Evault) GetTotalBytesUsed() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getTotalBytesUsed", nil, &r.Options, &resp)
	return
}

// Retrieve The total snapshot retention count of all schedules on this network storage volume.
func (r Network_Storage_Backup_Evault) GetTotalScheduleSnapshotRetentionCount() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getTotalScheduleSnapshotRetentionCount", nil, &r.Options, &resp)
	return
}

// Retrieve The usage notification for SL Storage services.
func (r Network_Storage_Backup_Evault) GetUsageNotification() (resp datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getUsageNotification", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetValidReplicationTargetDatacenterLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getValidReplicationTargetDatacenterLocations", nil, &r.Options, &resp)
	return
}

// Retrieve The type of network storage service.
func (r Network_Storage_Backup_Evault) GetVendorName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getVendorName", nil, &r.Options, &resp)
	return
}

// Retrieve When applicable, the virtual guest associated with a Storage service.
func (r Network_Storage_Backup_Evault) GetVirtualGuest() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getVirtualGuest", nil, &r.Options, &resp)
	return
}

// Retrieve The username and password history for a Storage service.
func (r Network_Storage_Backup_Evault) GetVolumeHistory() (resp []datatypes.Network_Storage_History, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getVolumeHistory", nil, &r.Options, &resp)
	return
}

// Retrieve The current status of a network storage volume.
func (r Network_Storage_Backup_Evault) GetVolumeStatus() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getVolumeStatus", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) GetWebCCAuthenticationDetails() (resp datatypes.Container_Network_Storage_Backup_Evault_WebCc_Authentication_Details, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getWebCCAuthenticationDetails", nil, &r.Options, &resp)
	return
}

// Retrieve The account username and password for the EVault webCC interface.
func (r Network_Storage_Backup_Evault) GetWebccAccount() (resp datatypes.Account_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getWebccAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The Weekly Schedule which is associated with this network storage volume.
func (r Network_Storage_Backup_Evault) GetWeeklySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "getWeeklySchedule", nil, &r.Options, &resp)
	return
}

// Immediate Failover to a volume replicant.  During the time which the replicant is in use the local nas volume will not be available.
func (r Network_Storage_Backup_Evault) ImmediateFailoverToReplicant(replicantId *int) (resp bool, err error) {
	params := []interface{}{
		replicantId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "immediateFailoverToReplicant", params, &r.Options, &resp)
	return
}

// Evault Bare Metal Restore is a special version of Rescue Kernel designed specifically for making full system restores made with Evault's BMR backup. This process works very similar to Rescue Kernel, except only the Evault restore program is available. The process takes approximately 10 minutes. Once completed you will be able to access your server to do a restore through VNC or your servers KVM-over-IP. IP information and credentials can be found on the hardware page of the customer portal. The Evault Application will be running automatically upon startup, and will walk you through the restore process.
func (r Network_Storage_Backup_Evault) InitiateBareMetalRestore() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "initiateBareMetalRestore", nil, &r.Options, &resp)
	return
}

// This method operates the same as the initiateBareMetalRestore() method.  However, using this method, the Bare Metal Restore can be initiated on any Windows server under the account.
func (r Network_Storage_Backup_Evault) InitiateBareMetalRestoreForServer(hardwareId *int) (resp bool, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "initiateBareMetalRestoreForServer", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) IsBlockingOperationInProgress(exemptStatusKeyNames []string) (resp bool, err error) {
	params := []interface{}{
		exemptStatusKeyNames,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "isBlockingOperationInProgress", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromHardware", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessFromHost(typeClassName *string, hostId *int) (resp datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		typeClassName,
		hostId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromHost", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessFromHostList(hostObjectTemplates []datatypes.Container_Network_Storage_Host) (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		hostObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromHostList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage will be listed in the allowedIpAddresses property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromIpAddress", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) RemoveAccessFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromIpAddressList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) RemoveAccessFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromSubnet", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) RemoveAccessFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Hardware objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationHardware property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessToReplicantFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessToReplicantFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationIpAddresses property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessToReplicantFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessToReplicantFromIpAddressList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) RemoveAccessToReplicantFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessToReplicantFromSubnet", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationSubnets property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessToReplicantFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessToReplicantFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationVirtualGuests property of this storage volume.
func (r Network_Storage_Backup_Evault) RemoveAccessToReplicantFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeAccessToReplicantFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method will remove a credential from the current volume. The credential must have been created using the 'addNewCredential' method.
func (r Network_Storage_Backup_Evault) RemoveCredential(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "removeCredential", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Restore an individual file so that it may be used as it was before it was deleted.
//
// If a file is deleted from a Virtual Server Storage account, the file is placed into the account's recycle bin and not permanently deleted. Therefore, restoreFile can be used to place the file back into your Virtual Server account's root directory.
func (r Network_Storage_Backup_Evault) RestoreFile(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "restoreFile", params, &r.Options, &resp)
	return
}

// Restore the volume from a snapshot that was previously taken.
func (r Network_Storage_Backup_Evault) RestoreFromSnapshot(snapshotId *int) (resp bool, err error) {
	params := []interface{}{
		snapshotId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "restoreFromSnapshot", params, &r.Options, &resp)
	return
}

// The method will retrieve the password for the StorageLayer or Virtual Server Storage Account and email the password.  The Storage Account passwords will be emailed to the master user.  For Virtual Server Storage, the password will be sent to the email address used as the username.
func (r Network_Storage_Backup_Evault) SendPasswordReminderEmail(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "sendPasswordReminderEmail", params, &r.Options, &resp)
	return
}

// Enable or disable the mounting of a Storage volume. When mounting is enabled the Storage volume will be mountable or available for use.
//
// For Virtual Server volumes, disabling mounting will deny access to the Virtual Server Account, remove published material and deny all file interaction including uploads and downloads.
//
// Enabling or disabling mounting for Storage volumes is not possible if mounting has been disabled by SoftLayer or a parent account.
func (r Network_Storage_Backup_Evault) SetMountable(mountable *bool) (resp bool, err error) {
	params := []interface{}{
		mountable,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "setMountable", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Backup_Evault) SetSnapshotAllocation(capacityGb *int) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		capacityGb,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "setSnapshotAllocation", params, &r.Options, &resp)
	return
}

// Upgrade the Storage volume to one of the upgradable packages (for example from 10 Gigs of EVault storage to 100 Gigs of EVault storage).
func (r Network_Storage_Backup_Evault) UpgradeVolumeCapacity(itemId *int) (resp bool, err error) {
	params := []interface{}{
		itemId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "upgradeVolumeCapacity", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Upload a file to a Storage account's root directory. Once uploaded, this method returns new file entity identifier for the upload file.
//
// The following properties are required in the ''file'' parameter.
// *'''name''': The name of the file you wish to upload
// *'''content''': The raw contents of the file you wish to upload.
// *'''contentType''': The MIME-type of content that you wish to upload.
func (r Network_Storage_Backup_Evault) UploadFile(file *datatypes.Container_Utility_File_Entity) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		file,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Backup_Evault", "uploadFile", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Group struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageGroupService returns an instance of the Network_Storage_Group SoftLayer service
func GetNetworkStorageGroupService(sess *session.Session) Network_Storage_Group {
	return Network_Storage_Group{Session: sess}
}

func (r Network_Storage_Group) Id(id int) Network_Storage_Group {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Group) Mask(mask string) Network_Storage_Group {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Group) Filter(filter string) Network_Storage_Group {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Group) Limit(limit int) Network_Storage_Group {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Group) Offset(offset int) Network_Storage_Group {
	r.Options.Offset = &offset
	return r
}

// Use this method to attach a SoftLayer_Network_Storage_Allowed_Host object to this group.  This will automatically enable access from this host to any SoftLayer_Network_Storage volumes currently attached to this group.
func (r Network_Storage_Group) AddAllowedHost(allowedHost *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		allowedHost,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "addAllowedHost", params, &r.Options, &resp)
	return
}

// Use this method to attach a SoftLayer_Network_Storage volume to this group.  This will automatically enable access to this volume for any SoftLayer_Network_Storage_Allowed_Host objects currently attached to this group.
func (r Network_Storage_Group) AttachToVolume(volume *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		volume,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "attachToVolume", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group) CreateObject(templateObject *datatypes.Network_Storage_Group) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group) EditObject(templateObject *datatypes.Network_Storage_Group) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Account which owns this group.
func (r Network_Storage_Group) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getAccount", nil, &r.Options, &resp)
	return
}

// Use this method to retrieve all network storage groups.
func (r Network_Storage_Group) GetAllObjects() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve The allowed hosts list for this group.
func (r Network_Storage_Group) GetAllowedHosts() (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getAllowedHosts", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volumes this group is attached to.
func (r Network_Storage_Group) GetAttachedVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getAttachedVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The type which defines this group.
func (r Network_Storage_Group) GetGroupType() (resp datatypes.Network_Storage_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getGroupType", nil, &r.Options, &resp)
	return
}

// Use this method to retrieve network connection information for SoftLayer_Network_Storage_Allowed_Host objects within this group.
func (r Network_Storage_Group) GetNetworkConnectionDetails() (resp datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group) GetObject() (resp datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The OS Type this group is configured for.
func (r Network_Storage_Group) GetOsType() (resp datatypes.Network_Storage_Iscsi_OS_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getOsType", nil, &r.Options, &resp)
	return
}

// Retrieve The network resource this group is created on.
func (r Network_Storage_Group) GetServiceResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "getServiceResource", nil, &r.Options, &resp)
	return
}

// Use this method to remove a SoftLayer_Network_Storage_Allowed_Host object from this group.  This will automatically disable access from this host to any SoftLayer_Network_Storage volumes currently attached to this group.
func (r Network_Storage_Group) RemoveAllowedHost(allowedHost *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		allowedHost,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "removeAllowedHost", params, &r.Options, &resp)
	return
}

// Use this method to remove a SoftLayer_Network_Storage volume from this group.  This will automatically disable access to this volume for any SoftLayer_Network_Storage_Allowed_Host objects currently attached to this group.
func (r Network_Storage_Group) RemoveFromVolume(volume *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		volume,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group", "removeFromVolume", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Group_Iscsi struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageGroupIscsiService returns an instance of the Network_Storage_Group_Iscsi SoftLayer service
func GetNetworkStorageGroupIscsiService(sess *session.Session) Network_Storage_Group_Iscsi {
	return Network_Storage_Group_Iscsi{Session: sess}
}

func (r Network_Storage_Group_Iscsi) Id(id int) Network_Storage_Group_Iscsi {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Group_Iscsi) Mask(mask string) Network_Storage_Group_Iscsi {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Group_Iscsi) Filter(filter string) Network_Storage_Group_Iscsi {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Group_Iscsi) Limit(limit int) Network_Storage_Group_Iscsi {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Group_Iscsi) Offset(offset int) Network_Storage_Group_Iscsi {
	r.Options.Offset = &offset
	return r
}

// Use this method to attach a SoftLayer_Network_Storage_Allowed_Host object to this group.  This will automatically enable access from this host to any SoftLayer_Network_Storage volumes currently attached to this group.
func (r Network_Storage_Group_Iscsi) AddAllowedHost(allowedHost *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		allowedHost,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "addAllowedHost", params, &r.Options, &resp)
	return
}

// Use this method to attach a SoftLayer_Network_Storage volume to this group.  This will automatically enable access to this volume for any SoftLayer_Network_Storage_Allowed_Host objects currently attached to this group.
func (r Network_Storage_Group_Iscsi) AttachToVolume(volume *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		volume,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "attachToVolume", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Iscsi) CreateObject(templateObject *datatypes.Network_Storage_Group) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Iscsi) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Iscsi) EditObject(templateObject *datatypes.Network_Storage_Group) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Account which owns this group.
func (r Network_Storage_Group_Iscsi) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getAccount", nil, &r.Options, &resp)
	return
}

// Use this method to retrieve all network storage groups.
func (r Network_Storage_Group_Iscsi) GetAllObjects() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve The allowed hosts list for this group.
func (r Network_Storage_Group_Iscsi) GetAllowedHosts() (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getAllowedHosts", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volumes this group is attached to.
func (r Network_Storage_Group_Iscsi) GetAttachedVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getAttachedVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The type which defines this group.
func (r Network_Storage_Group_Iscsi) GetGroupType() (resp datatypes.Network_Storage_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getGroupType", nil, &r.Options, &resp)
	return
}

// Use this method to retrieve network connection information for SoftLayer_Network_Storage_Allowed_Host objects within this group.
func (r Network_Storage_Group_Iscsi) GetNetworkConnectionDetails() (resp datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Iscsi) GetObject() (resp datatypes.Network_Storage_Group_Iscsi, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The OS Type this group is configured for.
func (r Network_Storage_Group_Iscsi) GetOsType() (resp datatypes.Network_Storage_Iscsi_OS_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getOsType", nil, &r.Options, &resp)
	return
}

// Retrieve The network resource this group is created on.
func (r Network_Storage_Group_Iscsi) GetServiceResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "getServiceResource", nil, &r.Options, &resp)
	return
}

// Use this method to remove a SoftLayer_Network_Storage_Allowed_Host object from this group.  This will automatically disable access from this host to any SoftLayer_Network_Storage volumes currently attached to this group.
func (r Network_Storage_Group_Iscsi) RemoveAllowedHost(allowedHost *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		allowedHost,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "removeAllowedHost", params, &r.Options, &resp)
	return
}

// Use this method to remove a SoftLayer_Network_Storage volume from this group.  This will automatically disable access to this volume for any SoftLayer_Network_Storage_Allowed_Host objects currently attached to this group.
func (r Network_Storage_Group_Iscsi) RemoveFromVolume(volume *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		volume,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Iscsi", "removeFromVolume", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Group_Nfs struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageGroupNfsService returns an instance of the Network_Storage_Group_Nfs SoftLayer service
func GetNetworkStorageGroupNfsService(sess *session.Session) Network_Storage_Group_Nfs {
	return Network_Storage_Group_Nfs{Session: sess}
}

func (r Network_Storage_Group_Nfs) Id(id int) Network_Storage_Group_Nfs {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Group_Nfs) Mask(mask string) Network_Storage_Group_Nfs {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Group_Nfs) Filter(filter string) Network_Storage_Group_Nfs {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Group_Nfs) Limit(limit int) Network_Storage_Group_Nfs {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Group_Nfs) Offset(offset int) Network_Storage_Group_Nfs {
	r.Options.Offset = &offset
	return r
}

// Use this method to attach a SoftLayer_Network_Storage_Allowed_Host object to this group.  This will automatically enable access from this host to any SoftLayer_Network_Storage volumes currently attached to this group.
func (r Network_Storage_Group_Nfs) AddAllowedHost(allowedHost *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		allowedHost,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "addAllowedHost", params, &r.Options, &resp)
	return
}

// Use this method to attach a SoftLayer_Network_Storage volume to this group.  This will automatically enable access to this volume for any SoftLayer_Network_Storage_Allowed_Host objects currently attached to this group.
func (r Network_Storage_Group_Nfs) AttachToVolume(volume *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		volume,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "attachToVolume", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Nfs) CreateObject(templateObject *datatypes.Network_Storage_Group) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Nfs) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Nfs) EditObject(templateObject *datatypes.Network_Storage_Group) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Account which owns this group.
func (r Network_Storage_Group_Nfs) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getAccount", nil, &r.Options, &resp)
	return
}

// Use this method to retrieve all network storage groups.
func (r Network_Storage_Group_Nfs) GetAllObjects() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve The allowed hosts list for this group.
func (r Network_Storage_Group_Nfs) GetAllowedHosts() (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getAllowedHosts", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volumes this group is attached to.
func (r Network_Storage_Group_Nfs) GetAttachedVolumes() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getAttachedVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve The type which defines this group.
func (r Network_Storage_Group_Nfs) GetGroupType() (resp datatypes.Network_Storage_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getGroupType", nil, &r.Options, &resp)
	return
}

// Use this method to retrieve network connection information for SoftLayer_Network_Storage_Allowed_Host objects within this group.
func (r Network_Storage_Group_Nfs) GetNetworkConnectionDetails() (resp datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Nfs) GetObject() (resp datatypes.Network_Storage_Group_Nfs, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The OS Type this group is configured for.
func (r Network_Storage_Group_Nfs) GetOsType() (resp datatypes.Network_Storage_Iscsi_OS_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getOsType", nil, &r.Options, &resp)
	return
}

// Retrieve The network resource this group is created on.
func (r Network_Storage_Group_Nfs) GetServiceResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "getServiceResource", nil, &r.Options, &resp)
	return
}

// Use this method to remove a SoftLayer_Network_Storage_Allowed_Host object from this group.  This will automatically disable access from this host to any SoftLayer_Network_Storage volumes currently attached to this group.
func (r Network_Storage_Group_Nfs) RemoveAllowedHost(allowedHost *datatypes.Network_Storage_Allowed_Host) (resp bool, err error) {
	params := []interface{}{
		allowedHost,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "removeAllowedHost", params, &r.Options, &resp)
	return
}

// Use this method to remove a SoftLayer_Network_Storage volume from this group.  This will automatically disable access to this volume for any SoftLayer_Network_Storage_Allowed_Host objects currently attached to this group.
func (r Network_Storage_Group_Nfs) RemoveFromVolume(volume *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		volume,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Nfs", "removeFromVolume", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Group_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageGroupTypeService returns an instance of the Network_Storage_Group_Type SoftLayer service
func GetNetworkStorageGroupTypeService(sess *session.Session) Network_Storage_Group_Type {
	return Network_Storage_Group_Type{Session: sess}
}

func (r Network_Storage_Group_Type) Id(id int) Network_Storage_Group_Type {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Group_Type) Mask(mask string) Network_Storage_Group_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Group_Type) Filter(filter string) Network_Storage_Group_Type {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Group_Type) Limit(limit int) Network_Storage_Group_Type {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Group_Type) Offset(offset int) Network_Storage_Group_Type {
	r.Options.Offset = &offset
	return r
}

// Use this method to retrieve all storage group types available.
func (r Network_Storage_Group_Type) GetAllObjects() (resp []datatypes.Network_Storage_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Group_Type) GetObject() (resp datatypes.Network_Storage_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Group_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Hub_Cleversafe_Account struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageHubCleversafeAccountService returns an instance of the Network_Storage_Hub_Cleversafe_Account SoftLayer service
func GetNetworkStorageHubCleversafeAccountService(sess *session.Session) Network_Storage_Hub_Cleversafe_Account {
	return Network_Storage_Hub_Cleversafe_Account{Session: sess}
}

func (r Network_Storage_Hub_Cleversafe_Account) Id(id int) Network_Storage_Hub_Cleversafe_Account {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Hub_Cleversafe_Account) Mask(mask string) Network_Storage_Hub_Cleversafe_Account {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Hub_Cleversafe_Account) Filter(filter string) Network_Storage_Hub_Cleversafe_Account {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Hub_Cleversafe_Account) Limit(limit int) Network_Storage_Hub_Cleversafe_Account {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Hub_Cleversafe_Account) Offset(offset int) Network_Storage_Hub_Cleversafe_Account {
	r.Options.Offset = &offset
	return r
}

// Create credentials for an IBM Cloud Object Storage Account
func (r Network_Storage_Hub_Cleversafe_Account) CredentialCreate() (resp []datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "credentialCreate", nil, &r.Options, &resp)
	return
}

// Delete a credential
func (r Network_Storage_Hub_Cleversafe_Account) CredentialDelete(credential *datatypes.Network_Storage_Credential) (resp bool, err error) {
	params := []interface{}{
		credential,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "credentialDelete", params, &r.Options, &resp)
	return
}

// Retrieve SoftLayer account to which an IBM Cloud Object Storage account belongs to.
func (r Network_Storage_Hub_Cleversafe_Account) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve An associated parent billing item which is active. Includes billing items which are scheduled to be cancelled in the future.
func (r Network_Storage_Hub_Cleversafe_Account) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve An associated parent billing item which has been cancelled.
func (r Network_Storage_Hub_Cleversafe_Account) GetCancelledBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getCancelledBillingItem", nil, &r.Options, &resp)
	return
}

// Returns the capacity usage for an IBM Cloud Object Storage account.
func (r Network_Storage_Hub_Cleversafe_Account) GetCapacityUsage() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getCapacityUsage", nil, &r.Options, &resp)
	return
}

// Returns a collection of valid storage policies to be used during bucket creation.
func (r Network_Storage_Hub_Cleversafe_Account) GetCloudObjectStoragePolicy() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_Policy, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getCloudObjectStoragePolicy", nil, &r.Options, &resp)
	return
}

// Returns credential limits for this IBM Cloud Object Storage account.
func (r Network_Storage_Hub_Cleversafe_Account) GetCredentialLimit() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getCredentialLimit", nil, &r.Options, &resp)
	return
}

// Retrieve Credentials used for generating an AWS signature. Max of 2.
func (r Network_Storage_Hub_Cleversafe_Account) GetCredentials() (resp []datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getCredentials", nil, &r.Options, &resp)
	return
}

// Returns a collection of endpoint URLs available to this IBM Cloud Object Storage account.
func (r Network_Storage_Hub_Cleversafe_Account) GetEndpoints() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_Endpoint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getEndpoints", nil, &r.Options, &resp)
	return
}

// Retrieve Provides an interface to various metrics relating to the usage of an IBM Cloud Object Storage account.
func (r Network_Storage_Hub_Cleversafe_Account) GetMetricTrackingObject() (resp datatypes.Metric_Tracking_Object, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getMetricTrackingObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Hub_Cleversafe_Account) GetObject() (resp datatypes.Network_Storage_Hub_Cleversafe_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve Unique identifier for an IBM Cloud Object Storage account.
func (r Network_Storage_Hub_Cleversafe_Account) GetUuid() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Cleversafe_Account", "getUuid", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Hub_Swift_Share struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageHubSwiftShareService returns an instance of the Network_Storage_Hub_Swift_Share SoftLayer service
func GetNetworkStorageHubSwiftShareService(sess *session.Session) Network_Storage_Hub_Swift_Share {
	return Network_Storage_Hub_Swift_Share{Session: sess}
}

func (r Network_Storage_Hub_Swift_Share) Id(id int) Network_Storage_Hub_Swift_Share {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Hub_Swift_Share) Mask(mask string) Network_Storage_Hub_Swift_Share {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Hub_Swift_Share) Filter(filter string) Network_Storage_Hub_Swift_Share {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Hub_Swift_Share) Limit(limit int) Network_Storage_Hub_Swift_Share {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Hub_Swift_Share) Offset(offset int) Network_Storage_Hub_Swift_Share {
	r.Options.Offset = &offset
	return r
}

// This method returns a collection of container objects.
func (r Network_Storage_Hub_Swift_Share) GetContainerList() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_Folder, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Swift_Share", "getContainerList", nil, &r.Options, &resp)
	return
}

// This method returns a file object given the file's full name.
func (r Network_Storage_Hub_Swift_Share) GetFile(fileName *string, container *string) (resp datatypes.Container_Network_Storage_Hub_ObjectStorage_File, err error) {
	params := []interface{}{
		fileName,
		container,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Swift_Share", "getFile", params, &r.Options, &resp)
	return
}

// This method returns a collection of the file objects within a container and the given path.
func (r Network_Storage_Hub_Swift_Share) GetFileList(container *string, path *string) (resp []datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		container,
		path,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Hub_Swift_Share", "getFileList", params, &r.Options, &resp)
	return
}

// The iscsi data type provides access to additional information about an iscsi volume such as the snapshot capacity limit and replication partners.
type Network_Storage_Iscsi struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageIscsiService returns an instance of the Network_Storage_Iscsi SoftLayer service
func GetNetworkStorageIscsiService(sess *session.Session) Network_Storage_Iscsi {
	return Network_Storage_Iscsi{Session: sess}
}

func (r Network_Storage_Iscsi) Id(id int) Network_Storage_Iscsi {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Iscsi) Mask(mask string) Network_Storage_Iscsi {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Iscsi) Filter(filter string) Network_Storage_Iscsi {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Iscsi) Limit(limit int) Network_Storage_Iscsi {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Iscsi) Offset(offset int) Network_Storage_Iscsi {
	r.Options.Offset = &offset
	return r
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromHardware", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) AllowAccessFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessFromHost(typeClassName *string, hostId *int) (resp datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		typeClassName,
		hostId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromHost", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage volume will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessFromHostList(hostObjectTemplates []datatypes.Container_Network_Storage_Host) (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		hostObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromHostList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) AllowAccessFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromIpAddress", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) AllowAccessFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromIpAddressList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromSubnet", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) AllowAccessFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage replicant volume.
func (r Network_Storage_Iscsi) AllowAccessToReplicantFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessToReplicantFromHardware", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replica volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage replica volume.
func (r Network_Storage_Iscsi) AllowAccessToReplicantFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessToReplicantFromHardwareList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) AllowAccessToReplicantFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessToReplicantFromIpAddress", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessToReplicantFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessToReplicantFromIpAddressList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage replicant volume.
func (r Network_Storage_Iscsi) AllowAccessToReplicantFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessToReplicantFromSubnet", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationSubnets property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessToReplicantFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessToReplicantFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replicant volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage replicant volume.
func (r Network_Storage_Iscsi) AllowAccessToReplicantFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessToReplicantFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Iscsi) AllowAccessToReplicantFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "allowAccessToReplicantFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method will assign an existing credential to the current volume. The credential must have been created using the 'addNewCredential' method. The volume type must support an additional credential.
func (r Network_Storage_Iscsi) AssignCredential(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "assignCredential", params, &r.Options, &resp)
	return
}

// This method will set up a new credential for the remote storage volume. The storage volume must support an additional credential. Once created, the credential will be automatically assigned to the current volume. If there are no volumes assigned to the credential it will be automatically deleted.
func (r Network_Storage_Iscsi) AssignNewCredential(typ *string) (resp datatypes.Network_Storage_Credential, err error) {
	params := []interface{}{
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "assignNewCredential", params, &r.Options, &resp)
	return
}

// The method will change the password for the given Storage/Virtual Server Storage account.
func (r Network_Storage_Iscsi) ChangePassword(username *string, currentPassword *string, newPassword *string) (resp bool, err error) {
	params := []interface{}{
		username,
		currentPassword,
		newPassword,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "changePassword", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// collectBandwidth() Retrieve the bandwidth usage for the current billing cycle.
func (r Network_Storage_Iscsi) CollectBandwidth(typ *string, startDate *datatypes.Time, endDate *datatypes.Time) (resp uint, err error) {
	params := []interface{}{
		typ,
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "collectBandwidth", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// collectBytesUsed() retrieves the number of bytes capacity currently in use on a Storage account.
func (r Network_Storage_Iscsi) CollectBytesUsed() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "collectBytesUsed", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) CreateFolder(folder *string) (resp bool, err error) {
	params := []interface{}{
		folder,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "createFolder", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) CreateSnapshot(notes *string) (resp datatypes.Network_Storage, err error) {
	params := []interface{}{
		notes,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "createSnapshot", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete all files within a Storage account. Depending on the type of Storage account, Deleting either deletes files permanently or sends files to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the files are in the account's recycle bin. If the files exist in the recycle bin, then they are permanently deleted.
//
// Please note, files can not be restored once they are permanently deleted.
func (r Network_Storage_Iscsi) DeleteAllFiles() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "deleteAllFiles", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete an individual file within a Storage account. Depending on the type of Storage account, Deleting a file either deletes the file permanently or sends the file to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the file is in the account's recycle bin. If the file exist in the recycle bin, then it is permanently deleted.
//
// Please note, a file can not be restored once it is permanently deleted.
func (r Network_Storage_Iscsi) DeleteFile(fileId *string) (resp bool, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "deleteFile", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Delete multiple files within a Storage account. Depending on the type of Storage account, Deleting either deletes files permanently or sends files to your account's recycle bin.
//
// Currently, Virtual Server storage is the only type of Storage account that sends files to a recycle bin when deleted. When called against a Virtual Server storage account , this method also determines if the files are in the account's recycle bin. If the files exist in the recycle bin, then they are permanently deleted.
//
// Please note, files can not be restored once they are permanently deleted.
func (r Network_Storage_Iscsi) DeleteFiles(fileIds []string) (resp bool, err error) {
	params := []interface{}{
		fileIds,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "deleteFiles", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) DeleteFolder(folder *string) (resp bool, err error) {
	params := []interface{}{
		folder,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "deleteFolder", params, &r.Options, &resp)
	return
}

// Delete a network storage volume. '''This cannot be undone.''' At this time only network storage snapshots may be deleted with this method.
//
// ''deleteObject'' returns Boolean ''true'' on successful deletion or ''false'' if it was unable to remove a volume;
func (r Network_Storage_Iscsi) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method is not valid for Legacy iSCSI Storage Volumes.
//
// Disable scheduled snapshots of this storage volume. Scheduling options include HOURLY, DAILY and WEEKLY schedules.
func (r Network_Storage_Iscsi) DisableSnapshots(scheduleType *string) (resp bool, err error) {
	params := []interface{}{
		scheduleType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "disableSnapshots", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Download a file from a Storage account. This method returns a file's details including the file's raw content.
func (r Network_Storage_Iscsi) DownloadFile(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "downloadFile", params, &r.Options, &resp)
	return
}

// This method will change the password of a credential created using the 'addNewCredential' method. If the credential exists on multiple storage volumes it will change for those volumes as well.
func (r Network_Storage_Iscsi) EditCredential(username *string, newPassword *string) (resp bool, err error) {
	params := []interface{}{
		username,
		newPassword,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "editCredential", params, &r.Options, &resp)
	return
}

// The password and/or notes may be modified for the Storage service except evault passwords and notes.
func (r Network_Storage_Iscsi) EditObject(templateObject *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "editObject", params, &r.Options, &resp)
	return
}

// This method is not valid for Legacy iSCSI Storage Volumes.
//
// Enable scheduled snapshots of this storage volume. Scheduling options include HOURLY, DAILY and WEEKLY schedules. For HOURLY schedules, provide relevant data for $scheduleType, $retentionCount and $minute. For DAILY schedules, provide relevant data for $scheduleType, $retentionCount, $minute, and $hour. For WEEKLY schedules, provide relevant data for all parameters of this method.
func (r Network_Storage_Iscsi) EnableSnapshots(scheduleType *string, retentionCount *int, minute *int, hour *int, dayOfWeek *string) (resp bool, err error) {
	params := []interface{}{
		scheduleType,
		retentionCount,
		minute,
		hour,
		dayOfWeek,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "enableSnapshots", params, &r.Options, &resp)
	return
}

// Failback from a volume replicant. In order to failback the volume must have already been failed over to a replicant.
func (r Network_Storage_Iscsi) FailbackFromReplicant() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "failbackFromReplicant", nil, &r.Options, &resp)
	return
}

// Failover to a volume replicant.  During the time which the replicant is in use the local nas volume will not be available.
func (r Network_Storage_Iscsi) FailoverToReplicant(replicantId *int) (resp bool, err error) {
	params := []interface{}{
		replicantId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "failoverToReplicant", params, &r.Options, &resp)
	return
}

// Retrieve The account that a Storage services belongs to.
func (r Network_Storage_Iscsi) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve Other usernames and passwords associated with a Storage volume.
func (r Network_Storage_Iscsi) GetAccountPassword() (resp datatypes.Account_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAccountPassword", nil, &r.Options, &resp)
	return
}

// Retrieve The currently active transactions on a network storage volume.
func (r Network_Storage_Iscsi) GetActiveTransactions() (resp []datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getActiveTransactions", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date for all files in a Storage account's root directory. This does not download file content.
func (r Network_Storage_Iscsi) GetAllFiles() (resp []datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllFiles", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date for all files matching the filter's criteria in a Storage account's root directory. This does not download file content.
func (r Network_Storage_Iscsi) GetAllFilesByFilter(filter *datatypes.Container_Utility_File_Entity) (resp []datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		filter,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllFilesByFilter", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Hardware that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage_Iscsi) GetAllowableHardware(filterHostname *string) (resp []datatypes.Hardware, err error) {
	params := []interface{}{
		filterHostname,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowableHardware", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Network_Subnet_IpAddress that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage_Iscsi) GetAllowableIpAddresses(subnetId *int, filterIpAddress *string) (resp []datatypes.Network_Subnet_IpAddress, err error) {
	params := []interface{}{
		subnetId,
		filterIpAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowableIpAddresses", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Network_Subnet that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage_Iscsi) GetAllowableSubnets(filterNetworkIdentifier *string) (resp []datatypes.Network_Subnet, err error) {
	params := []interface{}{
		filterNetworkIdentifier,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowableSubnets", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Virtual_Guest that can be authorized to this SoftLayer_Network_Storage.
func (r Network_Storage_Iscsi) GetAllowableVirtualGuests(filterHostname *string) (resp []datatypes.Virtual_Guest, err error) {
	params := []interface{}{
		filterHostname,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowableVirtualGuests", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume.
func (r Network_Storage_Iscsi) GetAllowedHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedHardware", nil, &r.Options, &resp)
	return
}

// Retrieves the total number of allowed hosts limit per volume.
func (r Network_Storage_Iscsi) GetAllowedHostsLimit() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedHostsLimit", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume.
func (r Network_Storage_Iscsi) GetAllowedIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage_Iscsi) GetAllowedReplicationHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedReplicationHardware", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage_Iscsi) GetAllowedReplicationIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedReplicationIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage_Iscsi) GetAllowedReplicationSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedReplicationSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
func (r Network_Storage_Iscsi) GetAllowedReplicationVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedReplicationVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Subnet objects which are allowed access to this storage volume.
func (r Network_Storage_Iscsi) GetAllowedSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Virtual_Guest objects which are allowed access to this storage volume.
func (r Network_Storage_Iscsi) GetAllowedVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getAllowedVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for a Storage volume.
func (r Network_Storage_Iscsi) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Iscsi) GetBillingItemCategory() (resp datatypes.Product_Item_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getBillingItemCategory", nil, &r.Options, &resp)
	return
}

// Retrieve network storage accounts by username and storage account type. Use this method if you wish to retrieve a storage record by username rather than by id. The ''type'' parameter must correspond to one of the available ''nasType'' values in the SoftLayer_Network_Storage data type.
func (r Network_Storage_Iscsi) GetByUsername(username *string, typ *string) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		username,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getByUsername", params, &r.Options, &resp)
	return
}

// Retrieve The amount of space used by the volume, in bytes.
func (r Network_Storage_Iscsi) GetBytesUsed() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getBytesUsed", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetCdnUrls() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_ContentDeliveryUrl, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getCdnUrls", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetClusterResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getClusterResource", nil, &r.Options, &resp)
	return
}

// Retrieve The schedule id which was executed to create a snapshot.
func (r Network_Storage_Iscsi) GetCreationScheduleId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getCreationScheduleId", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Iscsi) GetCredentials() (resp []datatypes.Network_Storage_Credential, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getCredentials", nil, &r.Options, &resp)
	return
}

// Retrieve The Daily Schedule which is associated with this network storage volume.
func (r Network_Storage_Iscsi) GetDailySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getDailySchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The events which have taken place on a network storage volume.
func (r Network_Storage_Iscsi) GetEvents() (resp []datatypes.Network_Storage_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getEvents", nil, &r.Options, &resp)
	return
}

//
//
//
func (r Network_Storage_Iscsi) GetFileBlockEncryptedLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getFileBlockEncryptedLocations", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve details such as id, name, size, create date of a file within a Storage account. This does not download file content.
func (r Network_Storage_Iscsi) GetFileByIdentifier(identifier *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		identifier,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getFileByIdentifier", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the file number of files in a Virtual Server Storage account's root directory. This does not include the files stored in the recycle bin.
func (r Network_Storage_Iscsi) GetFileCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getFileCount", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetFileList(folder *string, path *string) (resp []datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		folder,
		path,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getFileList", params, &r.Options, &resp)
	return
}

// Retrieve Retrieves the NFS Network Mount Address Name for a given File Storage Volume.
func (r Network_Storage_Iscsi) GetFileNetworkMountAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getFileNetworkMountAddress", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the number of files pending deletion in a Storage account's recycle bin. Files in an account's recycle bin may either be restored to the account's root directory or permanently deleted.
func (r Network_Storage_Iscsi) GetFilePendingDeleteCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getFilePendingDeleteCount", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve a list of files that are pending deletion in a Storage account's recycle bin. Files in an account's recycle bin may either be restored to the account's root directory or permanently deleted. This method does not download file content.
func (r Network_Storage_Iscsi) GetFilesPendingDelete() (resp []datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getFilesPendingDelete", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetFolderList() (resp []datatypes.Container_Network_Storage_Hub_ObjectStorage_Folder, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getFolderList", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}}
//
// getGraph() retrieves a Storage account's usage and returns a PNG graph image, title, and the minimum and maximum dates included in the graphed date range. Virtual Server storage accounts can also graph upload and download bandwidth usage.
func (r Network_Storage_Iscsi) GetGraph(startDate *datatypes.Time, endDate *datatypes.Time, typ *string) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		startDate,
		endDate,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getGraph", params, &r.Options, &resp)
	return
}

// Retrieve When applicable, the hardware associated with a Storage service.
func (r Network_Storage_Iscsi) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Iscsi) GetHasEncryptionAtRest() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getHasEncryptionAtRest", nil, &r.Options, &resp)
	return
}

// Retrieve The Hourly Schedule which is associated with this network storage volume.
func (r Network_Storage_Iscsi) GetHourlySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getHourlySchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The maximum number of IOPs selected for this volume.
func (r Network_Storage_Iscsi) GetIops() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getIops", nil, &r.Options, &resp)
	return
}

// Retrieve Relationship between a container volume and iSCSI LUNs.
func (r Network_Storage_Iscsi) GetIscsiLuns() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getIscsiLuns", nil, &r.Options, &resp)
	return
}

// Retrieve The ID of the LUN volume.
func (r Network_Storage_Iscsi) GetLunId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getLunId", nil, &r.Options, &resp)
	return
}

// Retrieve The manually-created snapshots associated with this SoftLayer_Network_Storage volume. Does not support pagination by result limit and offset.
func (r Network_Storage_Iscsi) GetManualSnapshots() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getManualSnapshots", nil, &r.Options, &resp)
	return
}

// Retrieve A network storage volume's metric tracking object. This object records all periodic polled data available to this volume.
func (r Network_Storage_Iscsi) GetMetricTrackingObject() (resp datatypes.Metric_Tracking_Object, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getMetricTrackingObject", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a network storage volume may be mounted.
func (r Network_Storage_Iscsi) GetMountableFlag() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getMountableFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetNetworkConnectionDetails() (resp datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetNetworkMountAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getNetworkMountAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The subscribers that will be notified for usage amount warnings and overages.
func (r Network_Storage_Iscsi) GetNotificationSubscribers() (resp []datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getNotificationSubscribers", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetObject() (resp datatypes.Network_Storage_Iscsi, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetObjectStorageConnectionInformation() (resp []datatypes.Container_Network_Service_Resource_ObjectStorage_ConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getObjectStorageConnectionInformation", nil, &r.Options, &resp)
	return
}

// Retrieve network storage accounts by SoftLayer_Network_Storage_Credential object. Use this method if you wish to retrieve a storage record by a credential rather than by id.
func (r Network_Storage_Iscsi) GetObjectsByCredential(credentialObject *datatypes.Network_Storage_Credential) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		credentialObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getObjectsByCredential", params, &r.Options, &resp)
	return
}

// Retrieve A volume's configured SoftLayer_Network_Storage_Iscsi_OS_Type.
func (r Network_Storage_Iscsi) GetOsType() (resp datatypes.Network_Storage_Iscsi_OS_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getOsType", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's configured SoftLayer_Network_Storage_Iscsi_OS_Type ID.
func (r Network_Storage_Iscsi) GetOsTypeId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getOsTypeId", nil, &r.Options, &resp)
	return
}

// Retrieve The volumes or snapshots partnered with a network storage volume in a parental role.
func (r Network_Storage_Iscsi) GetParentPartnerships() (resp []datatypes.Network_Storage_Partnership, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getParentPartnerships", nil, &r.Options, &resp)
	return
}

// Retrieve The parent volume of a volume in a complex storage relationship.
func (r Network_Storage_Iscsi) GetParentVolume() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getParentVolume", nil, &r.Options, &resp)
	return
}

// Retrieve The volumes or snapshots partnered with a network storage volume.
func (r Network_Storage_Iscsi) GetPartnerships() (resp []datatypes.Network_Storage_Partnership, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getPartnerships", nil, &r.Options, &resp)
	return
}

// Retrieve All permissions group(s) this volume is in.
func (r Network_Storage_Iscsi) GetPermissionsGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getPermissionsGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The properties used to provide additional details about a network storage volume.
func (r Network_Storage_Iscsi) GetProperties() (resp []datatypes.Network_Storage_Property, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getProperties", nil, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Retrieve the details of a file that is pending deletion in a Storage account's a recycle bin.
func (r Network_Storage_Iscsi) GetRecycleBinFileByIdentifier(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getRecycleBinFileByIdentifier", params, &r.Options, &resp)
	return
}

// Retrieves the remaining number of allowed hosts per volume.
func (r Network_Storage_Iscsi) GetRemainingAllowedHosts() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getRemainingAllowedHosts", nil, &r.Options, &resp)
	return
}

// Retrieve The iSCSI LUN volumes being replicated by this network storage volume.
func (r Network_Storage_Iscsi) GetReplicatingLuns() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getReplicatingLuns", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volume being replicated by a volume.
func (r Network_Storage_Iscsi) GetReplicatingVolume() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getReplicatingVolume", nil, &r.Options, &resp)
	return
}

// Retrieve The volume replication events.
func (r Network_Storage_Iscsi) GetReplicationEvents() (resp []datatypes.Network_Storage_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getReplicationEvents", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage volumes configured to be replicants of a volume.
func (r Network_Storage_Iscsi) GetReplicationPartners() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getReplicationPartners", nil, &r.Options, &resp)
	return
}

// Retrieve The Replication Schedule associated with a network storage volume.
func (r Network_Storage_Iscsi) GetReplicationSchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getReplicationSchedule", nil, &r.Options, &resp)
	return
}

// Retrieve The current replication status of a network storage volume. Indicates Failover or Failback status.
func (r Network_Storage_Iscsi) GetReplicationStatus() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getReplicationStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The schedules which are associated with a network storage volume.
func (r Network_Storage_Iscsi) GetSchedules() (resp []datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getSchedules", nil, &r.Options, &resp)
	return
}

// Retrieve The network resource a Storage service is connected to.
func (r Network_Storage_Iscsi) GetServiceResource() (resp datatypes.Network_Service_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getServiceResource", nil, &r.Options, &resp)
	return
}

// Retrieve The IP address of a Storage resource.
func (r Network_Storage_Iscsi) GetServiceResourceBackendIpAddress() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getServiceResourceBackendIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The name of a Storage's network resource.
func (r Network_Storage_Iscsi) GetServiceResourceName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getServiceResourceName", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's configured snapshot space size.
func (r Network_Storage_Iscsi) GetSnapshotCapacityGb() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getSnapshotCapacityGb", nil, &r.Options, &resp)
	return
}

// Retrieve The creation timestamp of the snapshot on the storage platform.
func (r Network_Storage_Iscsi) GetSnapshotCreationTimestamp() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getSnapshotCreationTimestamp", nil, &r.Options, &resp)
	return
}

// Retrieve The percentage of used snapshot space after which to delete automated snapshots.
func (r Network_Storage_Iscsi) GetSnapshotDeletionThresholdPercentage() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getSnapshotDeletionThresholdPercentage", nil, &r.Options, &resp)
	return
}

// Retrieve The snapshot size in bytes.
func (r Network_Storage_Iscsi) GetSnapshotSizeBytes() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getSnapshotSizeBytes", nil, &r.Options, &resp)
	return
}

// Retrieve A volume's available snapshot reservation space.
func (r Network_Storage_Iscsi) GetSnapshotSpaceAvailable() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getSnapshotSpaceAvailable", nil, &r.Options, &resp)
	return
}

// Retrieve The snapshots associated with this SoftLayer_Network_Storage volume.
func (r Network_Storage_Iscsi) GetSnapshots() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getSnapshots", nil, &r.Options, &resp)
	return
}

// Retrieves a list of snapshots for this SoftLayer_Network_Storage volume. This method works with the result limits and offset to support pagination.
func (r Network_Storage_Iscsi) GetSnapshotsForVolume() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getSnapshotsForVolume", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Iscsi) GetStaasVersion() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getStaasVersion", nil, &r.Options, &resp)
	return
}

// Retrieve The network storage groups this volume is attached to.
func (r Network_Storage_Iscsi) GetStorageGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getStorageGroups", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetStorageGroupsNetworkConnectionDetails() (resp []datatypes.Container_Network_Storage_NetworkConnectionInformation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getStorageGroupsNetworkConnectionDetails", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Storage_Iscsi) GetStorageTierLevel() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getStorageTierLevel", nil, &r.Options, &resp)
	return
}

// Retrieve A description of the Storage object.
func (r Network_Storage_Iscsi) GetStorageType() (resp datatypes.Network_Storage_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getStorageType", nil, &r.Options, &resp)
	return
}

// Retrieve The amount of space used by the volume.
func (r Network_Storage_Iscsi) GetTotalBytesUsed() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getTotalBytesUsed", nil, &r.Options, &resp)
	return
}

// Retrieve The total snapshot retention count of all schedules on this network storage volume.
func (r Network_Storage_Iscsi) GetTotalScheduleSnapshotRetentionCount() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getTotalScheduleSnapshotRetentionCount", nil, &r.Options, &resp)
	return
}

// Retrieve The usage notification for SL Storage services.
func (r Network_Storage_Iscsi) GetUsageNotification() (resp datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getUsageNotification", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) GetValidReplicationTargetDatacenterLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getValidReplicationTargetDatacenterLocations", nil, &r.Options, &resp)
	return
}

// Retrieve The type of network storage service.
func (r Network_Storage_Iscsi) GetVendorName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getVendorName", nil, &r.Options, &resp)
	return
}

// Retrieve When applicable, the virtual guest associated with a Storage service.
func (r Network_Storage_Iscsi) GetVirtualGuest() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getVirtualGuest", nil, &r.Options, &resp)
	return
}

// Retrieve The username and password history for a Storage service.
func (r Network_Storage_Iscsi) GetVolumeHistory() (resp []datatypes.Network_Storage_History, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getVolumeHistory", nil, &r.Options, &resp)
	return
}

// Retrieve The current status of a network storage volume.
func (r Network_Storage_Iscsi) GetVolumeStatus() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getVolumeStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The account username and password for the EVault webCC interface.
func (r Network_Storage_Iscsi) GetWebccAccount() (resp datatypes.Account_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getWebccAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The Weekly Schedule which is associated with this network storage volume.
func (r Network_Storage_Iscsi) GetWeeklySchedule() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "getWeeklySchedule", nil, &r.Options, &resp)
	return
}

// Immediate Failover to a volume replicant.  During the time which the replicant is in use the local nas volume will not be available.
func (r Network_Storage_Iscsi) ImmediateFailoverToReplicant(replicantId *int) (resp bool, err error) {
	params := []interface{}{
		replicantId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "immediateFailoverToReplicant", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) IsBlockingOperationInProgress(exemptStatusKeyNames []string) (resp bool, err error) {
	params := []interface{}{
		exemptStatusKeyNames,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "isBlockingOperationInProgress", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage_Iscsi) RemoveAccessFromHardware(hardwareObjectTemplate *datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromHardware", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage volume.
func (r Network_Storage_Iscsi) RemoveAccessFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage_Iscsi) RemoveAccessFromHost(typeClassName *string, hostId *int) (resp datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		typeClassName,
		hostId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromHost", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The [[SoftLayer_Hardware|SoftLayer_Virtual_Guest|SoftLayer_Network_Subnet|SoftLayer_Network_Subnet_IpAddress]] objects which have been allowed access to this storage will be listed in the [[allowedHardware|allowedVirtualGuests|allowedSubnets|allowedIpAddresses]] property of this storage volume.
func (r Network_Storage_Iscsi) RemoveAccessFromHostList(hostObjectTemplates []datatypes.Container_Network_Storage_Host) (resp []datatypes.Network_Storage_Allowed_Host, err error) {
	params := []interface{}{
		hostObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromHostList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) RemoveAccessFromIpAddress(ipAddressObjectTemplate *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromIpAddress", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) RemoveAccessFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromIpAddressList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) RemoveAccessFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromSubnet", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) RemoveAccessFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Iscsi) RemoveAccessFromVirtualGuest(virtualGuestObjectTemplate *datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromVirtualGuest", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage volume.
func (r Network_Storage_Iscsi) RemoveAccessFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replica volume.  The SoftLayer_Hardware objects which have been allowed access to this storage will be listed in the allowedHardware property of this storage replica volume.
func (r Network_Storage_Iscsi) RemoveAccessToReplicantFromHardwareList(hardwareObjectTemplates []datatypes.Hardware) (resp bool, err error) {
	params := []interface{}{
		hardwareObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessToReplicantFromHardwareList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replica volume.  The SoftLayer_Network_Subnet_IpAddress objects which have been allowed access to this storage will be listed in the allowedIpAddresses property of this storage replica volume.
func (r Network_Storage_Iscsi) RemoveAccessToReplicantFromIpAddressList(ipAddressObjectTemplates []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		ipAddressObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessToReplicantFromIpAddressList", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) RemoveAccessToReplicantFromSubnet(subnetObjectTemplate *datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessToReplicantFromSubnet", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage volume's replica.  The SoftLayer_Network_Subnet objects which have been allowed access to this storage volume's replica will be listed in the allowedReplicationSubnets property of this storage volume.
func (r Network_Storage_Iscsi) RemoveAccessToReplicantFromSubnetList(subnetObjectTemplates []datatypes.Network_Subnet) (resp bool, err error) {
	params := []interface{}{
		subnetObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessToReplicantFromSubnetList", params, &r.Options, &resp)
	return
}

// This method is used to modify the access control list for this Storage replica volume.  The SoftLayer_Virtual_Guest objects which have been allowed access to this storage will be listed in the allowedVirtualGuests property of this storage replica volume.
func (r Network_Storage_Iscsi) RemoveAccessToReplicantFromVirtualGuestList(virtualGuestObjectTemplates []datatypes.Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		virtualGuestObjectTemplates,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeAccessToReplicantFromVirtualGuestList", params, &r.Options, &resp)
	return
}

// This method will remove a credential from the current volume. The credential must have been created using the 'addNewCredential' method.
func (r Network_Storage_Iscsi) RemoveCredential(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "removeCredential", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Restore an individual file so that it may be used as it was before it was deleted.
//
// If a file is deleted from a Virtual Server Storage account, the file is placed into the account's recycle bin and not permanently deleted. Therefore, restoreFile can be used to place the file back into your Virtual Server account's root directory.
func (r Network_Storage_Iscsi) RestoreFile(fileId *string) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		fileId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "restoreFile", params, &r.Options, &resp)
	return
}

// Restore the volume from a snapshot that was previously taken.
func (r Network_Storage_Iscsi) RestoreFromSnapshot(snapshotId *int) (resp bool, err error) {
	params := []interface{}{
		snapshotId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "restoreFromSnapshot", params, &r.Options, &resp)
	return
}

// The method will retrieve the password for the StorageLayer or Virtual Server Storage Account and email the password.  The Storage Account passwords will be emailed to the master user.  For Virtual Server Storage, the password will be sent to the email address used as the username.
func (r Network_Storage_Iscsi) SendPasswordReminderEmail(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "sendPasswordReminderEmail", params, &r.Options, &resp)
	return
}

// Enable or disable the mounting of a Storage volume. When mounting is enabled the Storage volume will be mountable or available for use.
//
// For Virtual Server volumes, disabling mounting will deny access to the Virtual Server Account, remove published material and deny all file interaction including uploads and downloads.
//
// Enabling or disabling mounting for Storage volumes is not possible if mounting has been disabled by SoftLayer or a parent account.
func (r Network_Storage_Iscsi) SetMountable(mountable *bool) (resp bool, err error) {
	params := []interface{}{
		mountable,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "setMountable", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi) SetSnapshotAllocation(capacityGb *int) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		capacityGb,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "setSnapshotAllocation", params, &r.Options, &resp)
	return
}

// Upgrade the Storage volume to one of the upgradable packages (for example from 10 Gigs of EVault storage to 100 Gigs of EVault storage).
func (r Network_Storage_Iscsi) UpgradeVolumeCapacity(itemId *int) (resp bool, err error) {
	params := []interface{}{
		itemId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "upgradeVolumeCapacity", params, &r.Options, &resp)
	return
}

// {{CloudLayerOnlyMethod}} Upload a file to a Storage account's root directory. Once uploaded, this method returns new file entity identifier for the upload file.
//
// The following properties are required in the ''file'' parameter.
// *'''name''': The name of the file you wish to upload
// *'''content''': The raw contents of the file you wish to upload.
// *'''contentType''': The MIME-type of content that you wish to upload.
func (r Network_Storage_Iscsi) UploadFile(file *datatypes.Container_Utility_File_Entity) (resp datatypes.Container_Utility_File_Entity, err error) {
	params := []interface{}{
		file,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi", "uploadFile", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Storage_Iscsi_OS_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageIscsiOSTypeService returns an instance of the Network_Storage_Iscsi_OS_Type SoftLayer service
func GetNetworkStorageIscsiOSTypeService(sess *session.Session) Network_Storage_Iscsi_OS_Type {
	return Network_Storage_Iscsi_OS_Type{Session: sess}
}

func (r Network_Storage_Iscsi_OS_Type) Id(id int) Network_Storage_Iscsi_OS_Type {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Iscsi_OS_Type) Mask(mask string) Network_Storage_Iscsi_OS_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Iscsi_OS_Type) Filter(filter string) Network_Storage_Iscsi_OS_Type {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Iscsi_OS_Type) Limit(limit int) Network_Storage_Iscsi_OS_Type {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Iscsi_OS_Type) Offset(offset int) Network_Storage_Iscsi_OS_Type {
	r.Options.Offset = &offset
	return r
}

// Use this method to retrieve all iSCSI OS Types.
func (r Network_Storage_Iscsi_OS_Type) GetAllObjects() (resp []datatypes.Network_Storage_Iscsi_OS_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi_OS_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Iscsi_OS_Type) GetObject() (resp datatypes.Network_Storage_Iscsi_OS_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Iscsi_OS_Type", "getObject", nil, &r.Options, &resp)
	return
}

// Schedules can be created for select Storage services, such as iscsi. These schedules are used to perform various tasks such as scheduling snapshots or synchronizing replicants.
type Network_Storage_Schedule struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageScheduleService returns an instance of the Network_Storage_Schedule SoftLayer service
func GetNetworkStorageScheduleService(sess *session.Session) Network_Storage_Schedule {
	return Network_Storage_Schedule{Session: sess}
}

func (r Network_Storage_Schedule) Id(id int) Network_Storage_Schedule {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Schedule) Mask(mask string) Network_Storage_Schedule {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Schedule) Filter(filter string) Network_Storage_Schedule {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Schedule) Limit(limit int) Network_Storage_Schedule {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Schedule) Offset(offset int) Network_Storage_Schedule {
	r.Options.Offset = &offset
	return r
}

// Create a nas volume schedule
func (r Network_Storage_Schedule) CreateObject(templateObject *datatypes.Network_Storage_Schedule) (resp datatypes.Network_Storage_Schedule, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "createObject", params, &r.Options, &resp)
	return
}

// Delete a network storage schedule. '''This cannot be undone.''' ''deleteObject'' returns Boolean ''true'' on successful deletion or ''false'' if it was unable to remove a schedule;
func (r Network_Storage_Schedule) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "deleteObject", nil, &r.Options, &resp)
	return
}

// Edit a nas volume schedule
func (r Network_Storage_Schedule) EditObject(templateObject *datatypes.Network_Storage_Schedule) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The day of the month parameter of this schedule.
func (r Network_Storage_Schedule) GetDayOfMonth() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getDayOfMonth", nil, &r.Options, &resp)
	return
}

// Retrieve The day of the week parameter of this schedule.
func (r Network_Storage_Schedule) GetDayOfWeek() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getDayOfWeek", nil, &r.Options, &resp)
	return
}

// Retrieve Events which have been created as the result of a schedule execution.
func (r Network_Storage_Schedule) GetEvents() (resp []datatypes.Network_Storage_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getEvents", nil, &r.Options, &resp)
	return
}

// Retrieve The hour parameter of this schedule.
func (r Network_Storage_Schedule) GetHour() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getHour", nil, &r.Options, &resp)
	return
}

// Retrieve The minute parameter of this schedule.
func (r Network_Storage_Schedule) GetMinute() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getMinute", nil, &r.Options, &resp)
	return
}

// Retrieve The month of the year parameter of this schedule.
func (r Network_Storage_Schedule) GetMonthOfYear() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getMonthOfYear", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Schedule) GetObject() (resp datatypes.Network_Storage_Schedule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The associated partnership for a schedule.
func (r Network_Storage_Schedule) GetPartnership() (resp datatypes.Network_Storage_Partnership, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getPartnership", nil, &r.Options, &resp)
	return
}

// Retrieve Properties used for configuration of a schedule.
func (r Network_Storage_Schedule) GetProperties() (resp []datatypes.Network_Storage_Schedule_Property, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getProperties", nil, &r.Options, &resp)
	return
}

// Retrieve Replica snapshots which have been created as the result of this schedule's execution.
func (r Network_Storage_Schedule) GetReplicaSnapshots() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getReplicaSnapshots", nil, &r.Options, &resp)
	return
}

// Retrieve The number of snapshots this schedule is configured to retain.
func (r Network_Storage_Schedule) GetRetentionCount() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getRetentionCount", nil, &r.Options, &resp)
	return
}

// Retrieve Snapshots which have been created as the result of this schedule's execution.
func (r Network_Storage_Schedule) GetSnapshots() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getSnapshots", nil, &r.Options, &resp)
	return
}

// Retrieve The type provides a standardized definition for a schedule.
func (r Network_Storage_Schedule) GetType() (resp datatypes.Network_Storage_Schedule_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve The associated volume for a schedule.
func (r Network_Storage_Schedule) GetVolume() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule", "getVolume", nil, &r.Options, &resp)
	return
}

// A schedule property type is used to allow for a standardized method of defining network storage schedules.
type Network_Storage_Schedule_Property_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkStorageSchedulePropertyTypeService returns an instance of the Network_Storage_Schedule_Property_Type SoftLayer service
func GetNetworkStorageSchedulePropertyTypeService(sess *session.Session) Network_Storage_Schedule_Property_Type {
	return Network_Storage_Schedule_Property_Type{Session: sess}
}

func (r Network_Storage_Schedule_Property_Type) Id(id int) Network_Storage_Schedule_Property_Type {
	r.Options.Id = &id
	return r
}

func (r Network_Storage_Schedule_Property_Type) Mask(mask string) Network_Storage_Schedule_Property_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Storage_Schedule_Property_Type) Filter(filter string) Network_Storage_Schedule_Property_Type {
	r.Options.Filter = filter
	return r
}

func (r Network_Storage_Schedule_Property_Type) Limit(limit int) Network_Storage_Schedule_Property_Type {
	r.Options.Limit = &limit
	return r
}

func (r Network_Storage_Schedule_Property_Type) Offset(offset int) Network_Storage_Schedule_Property_Type {
	r.Options.Offset = &offset
	return r
}

// Use this method to retrieve all network storage schedule property types.
func (r Network_Storage_Schedule_Property_Type) GetAllObjects() (resp []datatypes.Network_Storage_Schedule_Property_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule_Property_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Storage_Schedule_Property_Type) GetObject() (resp datatypes.Network_Storage_Schedule_Property_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Storage_Schedule_Property_Type", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Subnet data type contains general information relating to a single SoftLayer subnet. Personal information in this type such as names, addresses, and phone numbers are assigned to the account only and not to users belonging to the account.
type Network_Subnet struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSubnetService returns an instance of the Network_Subnet SoftLayer service
func GetNetworkSubnetService(sess *session.Session) Network_Subnet {
	return Network_Subnet{Session: sess}
}

func (r Network_Subnet) Id(id int) Network_Subnet {
	r.Options.Id = &id
	return r
}

func (r Network_Subnet) Mask(mask string) Network_Subnet {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Subnet) Filter(filter string) Network_Subnet {
	r.Options.Filter = filter
	return r
}

func (r Network_Subnet) Limit(limit int) Network_Subnet {
	r.Options.Limit = &limit
	return r
}

func (r Network_Subnet) Offset(offset int) Network_Subnet {
	r.Options.Offset = &offset
	return r
}

// This method is used to allow access to a SoftLayer_Network_Storage volume that supports host- or network-level access control.
func (r Network_Subnet) AllowAccessToNetworkStorage(networkStorageTemplateObject *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		networkStorageTemplateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "allowAccessToNetworkStorage", params, &r.Options, &resp)
	return
}

// This method is used to allow access to multiple SoftLayer_Network_Storage volumes that support host- or network-level access control.
func (r Network_Subnet) AllowAccessToNetworkStorageList(networkStorageTemplateObjects []datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		networkStorageTemplateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "allowAccessToNetworkStorageList", params, &r.Options, &resp)
	return
}

// Create the default PTR records for this subnet
func (r Network_Subnet) CreateReverseDomainRecords() (resp datatypes.Dns_Domain_Reverse, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "createReverseDomainRecords", nil, &r.Options, &resp)
	return
}

// This function is used to create a new transaction to modify a subnet route. Routes are updated in one to two minutes depending on the number of transactions that are pending for a router.
//
// Usage of this function is restricted and may only be called from authorized accounts. It is not available for general API users without justification and consent from a SoftLayer representative.
func (r Network_Subnet) CreateSubnetRouteUpdateTransaction(newEndPointIpAddress *string) (resp bool, err error) {
	params := []interface{}{
		newEndPointIpAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "createSubnetRouteUpdateTransaction", params, &r.Options, &resp)
	return
}

// This function is used to create a new SoftLayer SWIP transaction to register your RWHOIS data with ARIN. SWIP transactions can only be initiated on subnets that contain more than 8 IP addresses.
func (r Network_Subnet) CreateSwipTransaction() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "createSwipTransaction", nil, &r.Options, &resp)
	return
}

// Edit the note for this subnet.
func (r Network_Subnet) EditNote(note *string) (resp bool, err error) {
	params := []interface{}{
		note,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "editNote", params, &r.Options, &resp)
	return
}

// Retrieve a list of a SoftLayer customer's subnets along with their SWIP transaction statuses. This is a shortcut method that combines the SoftLayer_Network_Subnet retrieval methods along with [[object masks]] to retrieve their subnets' associated SWIP transactions as well.
//
// This is a special function built for SoftLayer's use on the SWIP section of the customer portal, but may also be useful for API users looking for the same data.
func (r Network_Subnet) FindAllSubnetsAndActiveSwipTransactionStatus() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "findAllSubnetsAndActiveSwipTransactionStatus", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve If present, the active registration for this subnet.
func (r Network_Subnet) GetActiveRegistration() (resp datatypes.Network_Subnet_Registration, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getActiveRegistration", nil, &r.Options, &resp)
	return
}

// Retrieve All the swip transactions associated with a subnet that are still active.
func (r Network_Subnet) GetActiveSwipTransaction() (resp datatypes.Network_Subnet_Swip_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getActiveSwipTransaction", nil, &r.Options, &resp)
	return
}

// Retrieve The billing item for a subnet.
func (r Network_Subnet) GetActiveTransaction() (resp datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getActiveTransaction", nil, &r.Options, &resp)
	return
}

// Retrieve Identifier which distinguishes whether the subnet is public or private address space.
func (r Network_Subnet) GetAddressSpace() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getAddressSpace", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Allowed_Host information to connect this Subnet to Network Storage supporting access control lists.
func (r Network_Subnet) GetAllowedHost() (resp datatypes.Network_Storage_Allowed_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getAllowedHost", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage objects that this SoftLayer_Hardware has access to.
func (r Network_Subnet) GetAllowedNetworkStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getAllowedNetworkStorage", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage objects whose Replica that this SoftLayer_Hardware has access to.
func (r Network_Subnet) GetAllowedNetworkStorageReplicas() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getAllowedNetworkStorageReplicas", nil, &r.Options, &resp)
	return
}

// This method is retrieve a list of SoftLayer_Network_Storage volumes that are authorized access to this SoftLayer_Network_Subnet.
func (r Network_Subnet) GetAttachedNetworkStorages(nasType *string) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		nasType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getAttachedNetworkStorages", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Network_Storage volumes that can be authorized to this SoftLayer_Network_Subnet.
func (r Network_Subnet) GetAvailableNetworkStorages(nasType *string) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		nasType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getAvailableNetworkStorages", params, &r.Options, &resp)
	return
}

// Retrieve The billing item for a subnet.
func (r Network_Subnet) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetBoundDescendants() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getBoundDescendants", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not this subnet is associated with a router. Subnets that are not associated with a router cannot be routed.
func (r Network_Subnet) GetBoundRouterFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getBoundRouterFlag", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetBoundRouters() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getBoundRouters", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetChildren() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getChildren", nil, &r.Options, &resp)
	return
}

// Retrieve The data center this subnet may be routed within.
func (r Network_Subnet) GetDatacenter() (resp datatypes.Location_Datacenter, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getDatacenter", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetDescendants() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getDescendants", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetDisplayLabel() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getDisplayLabel", nil, &r.Options, &resp)
	return
}

// Retrieve A static routed ip address
func (r Network_Subnet) GetEndPointIpAddress() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getEndPointIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetGlobalIpRecord() (resp datatypes.Network_Subnet_IpAddress_Global, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getGlobalIpRecord", nil, &r.Options, &resp)
	return
}

// Retrieve The hardware using IP addresses on this subnet.
func (r Network_Subnet) GetHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve All the ip addresses associated with a subnet.
func (r Network_Subnet) GetIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve A subnet's associated network component.
func (r Network_Subnet) GetNetworkComponent() (resp datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getNetworkComponent", nil, &r.Options, &resp)
	return
}

// Retrieve The upstream network component firewall.
func (r Network_Subnet) GetNetworkComponentFirewall() (resp datatypes.Network_Component_Firewall, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getNetworkComponentFirewall", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetNetworkProtectionAddresses() (resp []datatypes.Network_Protection_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getNetworkProtectionAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve IPSec network tunnels that have access to a private subnet.
func (r Network_Subnet) GetNetworkTunnelContexts() (resp []datatypes.Network_Tunnel_Module_Context, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getNetworkTunnelContexts", nil, &r.Options, &resp)
	return
}

// Retrieve The VLAN object that a subnet is associated with.
func (r Network_Subnet) GetNetworkVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getNetworkVlan", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Subnet object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Subnet service. You can only retrieve the subnet whose vlan is associated with the account that you portal user is assigned to.
func (r Network_Subnet) GetObject() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The pod in which this subnet resides.
func (r Network_Subnet) GetPodName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getPodName", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetProtectedIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getProtectedIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetRegionalInternetRegistry() (resp datatypes.Network_Regional_Internet_Registry, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getRegionalInternetRegistry", nil, &r.Options, &resp)
	return
}

// Retrieve All registrations that have been created for this subnet.
func (r Network_Subnet) GetRegistrations() (resp []datatypes.Network_Subnet_Registration, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getRegistrations", nil, &r.Options, &resp)
	return
}

// Retrieve The resource groups in which this subnet is a member.
func (r Network_Subnet) GetResourceGroups() (resp []datatypes.Resource_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getResourceGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The reverse DNS domain associated with this subnet.
func (r Network_Subnet) GetReverseDomain() (resp datatypes.Dns_Domain, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getReverseDomain", nil, &r.Options, &resp)
	return
}

// Retrieve all reverse DNS records associated with a subnet.
func (r Network_Subnet) GetReverseDomainRecords() (resp []datatypes.Dns_Domain, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getReverseDomainRecords", nil, &r.Options, &resp)
	return
}

// Retrieve An identifier of the role the subnet is within. Roles dictate how a subnet may be used.
func (r Network_Subnet) GetRoleKeyName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getRoleKeyName", nil, &r.Options, &resp)
	return
}

// Retrieve The name of the role the subnet is within. Roles dictate how a subnet may be used.
func (r Network_Subnet) GetRoleName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getRoleName", nil, &r.Options, &resp)
	return
}

// getRoutableEndpointAddresses retrieves valid routable endpoint addresses for a subnet. You may use any IP address in a portable subnet, but may not use the network identifier, gateway, or broadcast address for primary and secondary on VLAN subnets.
func (r Network_Subnet) GetRoutableEndpointIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getRoutableEndpointIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The identifier for the type of route then subnet is currently configured for.
func (r Network_Subnet) GetRoutingTypeKeyName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getRoutingTypeKeyName", nil, &r.Options, &resp)
	return
}

// Retrieve The name for the type of route then subnet is currently configured for.
func (r Network_Subnet) GetRoutingTypeName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getRoutingTypeName", nil, &r.Options, &resp)
	return
}

// Retrieve the subnet associated with an IP address. You may only retrieve subnets assigned to your SoftLayer customer account.
func (r Network_Subnet) GetSubnetForIpAddress(ipAddress *string) (resp datatypes.Network_Subnet, err error) {
	params := []interface{}{
		ipAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getSubnetForIpAddress", params, &r.Options, &resp)
	return
}

// Retrieve All the swip transactions associated with a subnet.
func (r Network_Subnet) GetSwipTransaction() (resp []datatypes.Network_Subnet_Swip_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getSwipTransaction", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet) GetUnboundDescendants() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getUnboundDescendants", nil, &r.Options, &resp)
	return
}

// Retrieve The Virtual Servers using IP addresses on this subnet.
func (r Network_Subnet) GetVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "getVirtualGuests", nil, &r.Options, &resp)
	return
}

// This method is used to remove access to multiple SoftLayer_Network_Storage volumes
func (r Network_Subnet) RemoveAccessToNetworkStorageList(networkStorageTemplateObjects []datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		networkStorageTemplateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet", "removeAccessToNetworkStorageList", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Subnet_IpAddress data type contains general information relating to a single SoftLayer IPv4 address.
type Network_Subnet_IpAddress struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSubnetIpAddressService returns an instance of the Network_Subnet_IpAddress SoftLayer service
func GetNetworkSubnetIpAddressService(sess *session.Session) Network_Subnet_IpAddress {
	return Network_Subnet_IpAddress{Session: sess}
}

func (r Network_Subnet_IpAddress) Id(id int) Network_Subnet_IpAddress {
	r.Options.Id = &id
	return r
}

func (r Network_Subnet_IpAddress) Mask(mask string) Network_Subnet_IpAddress {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Subnet_IpAddress) Filter(filter string) Network_Subnet_IpAddress {
	r.Options.Filter = filter
	return r
}

func (r Network_Subnet_IpAddress) Limit(limit int) Network_Subnet_IpAddress {
	r.Options.Limit = &limit
	return r
}

func (r Network_Subnet_IpAddress) Offset(offset int) Network_Subnet_IpAddress {
	r.Options.Offset = &offset
	return r
}

// This method is used to allow access to a SoftLayer_Network_Storage volume that supports host- or network-level access control.
func (r Network_Subnet_IpAddress) AllowAccessToNetworkStorage(networkStorageTemplateObject *datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		networkStorageTemplateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "allowAccessToNetworkStorage", params, &r.Options, &resp)
	return
}

// This method is used to allow access to multiple SoftLayer_Network_Storage volumes that support host- or network-level access control.
func (r Network_Subnet_IpAddress) AllowAccessToNetworkStorageList(networkStorageTemplateObjects []datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		networkStorageTemplateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "allowAccessToNetworkStorageList", params, &r.Options, &resp)
	return
}

// Edit a subnet IP address.
func (r Network_Subnet_IpAddress) EditObject(templateObject *datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "editObject", params, &r.Options, &resp)
	return
}

// This function is used to edit multiple objects at the same time.
func (r Network_Subnet_IpAddress) EditObjects(templateObjects []datatypes.Network_Subnet_IpAddress) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "editObjects", params, &r.Options, &resp)
	return
}

// Search for an IP address record by IPv4 address.
func (r Network_Subnet_IpAddress) FindByIpv4Address(ipAddress *string) (resp datatypes.Network_Subnet_IpAddress, err error) {
	params := []interface{}{
		ipAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "findByIpv4Address", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage_Allowed_Host information to connect this IP Address to Network Storage supporting access control lists.
func (r Network_Subnet_IpAddress) GetAllowedHost() (resp datatypes.Network_Storage_Allowed_Host, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getAllowedHost", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage objects that this SoftLayer_Hardware has access to.
func (r Network_Subnet_IpAddress) GetAllowedNetworkStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getAllowedNetworkStorage", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Network_Storage objects whose Replica that this SoftLayer_Hardware has access to.
func (r Network_Subnet_IpAddress) GetAllowedNetworkStorageReplicas() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getAllowedNetworkStorageReplicas", nil, &r.Options, &resp)
	return
}

// Retrieve The application delivery controller using this address.
func (r Network_Subnet_IpAddress) GetApplicationDeliveryController() (resp datatypes.Network_Application_Delivery_Controller, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getApplicationDeliveryController", nil, &r.Options, &resp)
	return
}

// This method is retrieve a list of SoftLayer_Network_Storage volumes that are authorized access to this SoftLayer_Network_Subnet_IpAddress.
func (r Network_Subnet_IpAddress) GetAttachedNetworkStorages(nasType *string) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		nasType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getAttachedNetworkStorages", params, &r.Options, &resp)
	return
}

// This method retrieves a list of SoftLayer_Network_Storage volumes that can be authorized to this SoftLayer_Network_Subnet_IpAddress.
func (r Network_Subnet_IpAddress) GetAvailableNetworkStorages(nasType *string) (resp []datatypes.Network_Storage, err error) {
	params := []interface{}{
		nasType,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getAvailableNetworkStorages", params, &r.Options, &resp)
	return
}

// Search for an IP address record by IP address.
func (r Network_Subnet_IpAddress) GetByIpAddress(ipAddress *string) (resp datatypes.Network_Subnet_IpAddress, err error) {
	params := []interface{}{
		ipAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getByIpAddress", params, &r.Options, &resp)
	return
}

// Retrieve An IPSec network tunnel's address translations. These translations use a SoftLayer ip address from an assigned static NAT subnet to deliver the packets to the remote (customer) destination.
func (r Network_Subnet_IpAddress) GetContextTunnelTranslations() (resp []datatypes.Network_Tunnel_Module_Context_Address_Translation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getContextTunnelTranslations", nil, &r.Options, &resp)
	return
}

// Retrieve All the subnets routed to an IP address.
func (r Network_Subnet_IpAddress) GetEndpointSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getEndpointSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve A network component that is statically routed to an IP address.
func (r Network_Subnet_IpAddress) GetGuestNetworkComponent() (resp datatypes.Virtual_Guest_Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getGuestNetworkComponent", nil, &r.Options, &resp)
	return
}

// Retrieve A network component that is statically routed to an IP address.
func (r Network_Subnet_IpAddress) GetGuestNetworkComponentBinding() (resp datatypes.Virtual_Guest_Network_Component_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getGuestNetworkComponentBinding", nil, &r.Options, &resp)
	return
}

// Retrieve A server that this IP address is routed to.
func (r Network_Subnet_IpAddress) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve A network component that is statically routed to an IP address.
func (r Network_Subnet_IpAddress) GetNetworkComponent() (resp datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getNetworkComponent", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Subnet_IpAddress object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Subnet_IpAddress service. You can only retrieve the IP address whose subnet is associated with a VLAN that is associated with the account that your portal user is assigned to.
func (r Network_Subnet_IpAddress) GetObject() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The network gateway appliance using this address as the private IP address.
func (r Network_Subnet_IpAddress) GetPrivateNetworkGateway() (resp datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getPrivateNetworkGateway", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet_IpAddress) GetProtectionAddress() (resp []datatypes.Network_Protection_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getProtectionAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The network gateway appliance using this address as the public IP address.
func (r Network_Subnet_IpAddress) GetPublicNetworkGateway() (resp datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getPublicNetworkGateway", nil, &r.Options, &resp)
	return
}

// Retrieve An IPMI-based management network component of the IP address.
func (r Network_Subnet_IpAddress) GetRemoteManagementNetworkComponent() (resp datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getRemoteManagementNetworkComponent", nil, &r.Options, &resp)
	return
}

// Retrieve An IP address' associated subnet.
func (r Network_Subnet_IpAddress) GetSubnet() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getSubnet", nil, &r.Options, &resp)
	return
}

// Retrieve All events for this IP address stored in the datacenter syslogs from the last 24 hours
func (r Network_Subnet_IpAddress) GetSyslogEventsOneDay() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getSyslogEventsOneDay", nil, &r.Options, &resp)
	return
}

// Retrieve All events for this IP address stored in the datacenter syslogs from the last 7 days
func (r Network_Subnet_IpAddress) GetSyslogEventsSevenDays() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getSyslogEventsSevenDays", nil, &r.Options, &resp)
	return
}

// Retrieve Top Ten network datacenter syslog events, grouped by destination port, for the last 24 hours
func (r Network_Subnet_IpAddress) GetTopTenSyslogEventsByDestinationPortOneDay() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getTopTenSyslogEventsByDestinationPortOneDay", nil, &r.Options, &resp)
	return
}

// Retrieve Top Ten network datacenter syslog events, grouped by destination port, for the last 7 days
func (r Network_Subnet_IpAddress) GetTopTenSyslogEventsByDestinationPortSevenDays() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getTopTenSyslogEventsByDestinationPortSevenDays", nil, &r.Options, &resp)
	return
}

// Retrieve Top Ten network datacenter syslog events, grouped by source port, for the last 24 hours
func (r Network_Subnet_IpAddress) GetTopTenSyslogEventsByProtocolsOneDay() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getTopTenSyslogEventsByProtocolsOneDay", nil, &r.Options, &resp)
	return
}

// Retrieve Top Ten network datacenter syslog events, grouped by source port, for the last 7 days
func (r Network_Subnet_IpAddress) GetTopTenSyslogEventsByProtocolsSevenDays() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getTopTenSyslogEventsByProtocolsSevenDays", nil, &r.Options, &resp)
	return
}

// Retrieve Top Ten network datacenter syslog events, grouped by source ip address, for the last 24 hours
func (r Network_Subnet_IpAddress) GetTopTenSyslogEventsBySourceIpOneDay() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getTopTenSyslogEventsBySourceIpOneDay", nil, &r.Options, &resp)
	return
}

// Retrieve Top Ten network datacenter syslog events, grouped by source ip address, for the last 7 days
func (r Network_Subnet_IpAddress) GetTopTenSyslogEventsBySourceIpSevenDays() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getTopTenSyslogEventsBySourceIpSevenDays", nil, &r.Options, &resp)
	return
}

// Retrieve Top Ten network datacenter syslog events, grouped by source port, for the last 24 hours
func (r Network_Subnet_IpAddress) GetTopTenSyslogEventsBySourcePortOneDay() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getTopTenSyslogEventsBySourcePortOneDay", nil, &r.Options, &resp)
	return
}

// Retrieve Top Ten network datacenter syslog events, grouped by source port, for the last 7 days
func (r Network_Subnet_IpAddress) GetTopTenSyslogEventsBySourcePortSevenDays() (resp []datatypes.Network_Logging_Syslog, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getTopTenSyslogEventsBySourcePortSevenDays", nil, &r.Options, &resp)
	return
}

// Retrieve A virtual guest that this IP address is routed to.
func (r Network_Subnet_IpAddress) GetVirtualGuest() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getVirtualGuest", nil, &r.Options, &resp)
	return
}

// Retrieve Virtual licenses allocated for an IP Address.
func (r Network_Subnet_IpAddress) GetVirtualLicenses() (resp []datatypes.Software_VirtualLicense, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "getVirtualLicenses", nil, &r.Options, &resp)
	return
}

// This method is used to remove access to multiple SoftLayer_Network_Storage volumes
func (r Network_Subnet_IpAddress) RemoveAccessToNetworkStorageList(networkStorageTemplateObjects []datatypes.Network_Storage) (resp bool, err error) {
	params := []interface{}{
		networkStorageTemplateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress", "removeAccessToNetworkStorageList", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Subnet_IpAddress_Global struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSubnetIpAddressGlobalService returns an instance of the Network_Subnet_IpAddress_Global SoftLayer service
func GetNetworkSubnetIpAddressGlobalService(sess *session.Session) Network_Subnet_IpAddress_Global {
	return Network_Subnet_IpAddress_Global{Session: sess}
}

func (r Network_Subnet_IpAddress_Global) Id(id int) Network_Subnet_IpAddress_Global {
	r.Options.Id = &id
	return r
}

func (r Network_Subnet_IpAddress_Global) Mask(mask string) Network_Subnet_IpAddress_Global {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Subnet_IpAddress_Global) Filter(filter string) Network_Subnet_IpAddress_Global {
	r.Options.Filter = filter
	return r
}

func (r Network_Subnet_IpAddress_Global) Limit(limit int) Network_Subnet_IpAddress_Global {
	r.Options.Limit = &limit
	return r
}

func (r Network_Subnet_IpAddress_Global) Offset(offset int) Network_Subnet_IpAddress_Global {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Network_Subnet_IpAddress_Global) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress_Global", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The active transaction associated with this Global IP.
func (r Network_Subnet_IpAddress_Global) GetActiveTransaction() (resp datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress_Global", "getActiveTransaction", nil, &r.Options, &resp)
	return
}

// Retrieve The billing item for this Global IP.
func (r Network_Subnet_IpAddress_Global) GetBillingItem() (resp datatypes.Billing_Item_Network_Subnet_IpAddress_Global, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress_Global", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet_IpAddress_Global) GetDestinationIpAddress() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress_Global", "getDestinationIpAddress", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Subnet_IpAddress_Global) GetIpAddress() (resp datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress_Global", "getIpAddress", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Subnet_IpAddress_Global) GetObject() (resp datatypes.Network_Subnet_IpAddress_Global, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress_Global", "getObject", nil, &r.Options, &resp)
	return
}

// This function is used to create a new transaction to modify a global IP route. Routes are updated in one to two minutes depending on the number of transactions that are pending for a router.
func (r Network_Subnet_IpAddress_Global) Route(newEndPointIpAddress *string) (resp datatypes.Provisioning_Version1_Transaction, err error) {
	params := []interface{}{
		newEndPointIpAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress_Global", "route", params, &r.Options, &resp)
	return
}

// This function is used to create a new transaction to unroute a global IP address. Routes are updated in one to two minutes depending on the number of transactions that are pending for a router.
func (r Network_Subnet_IpAddress_Global) Unroute() (resp datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_IpAddress_Global", "unroute", nil, &r.Options, &resp)
	return
}

// The subnet registration data type contains general information relating to a single subnet registration instance. These registration instances can be updated to reflect changes, and will record the changes in the [[SoftLayer_Network_Subnet_Registration_Event|events]].
type Network_Subnet_Registration struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSubnetRegistrationService returns an instance of the Network_Subnet_Registration SoftLayer service
func GetNetworkSubnetRegistrationService(sess *session.Session) Network_Subnet_Registration {
	return Network_Subnet_Registration{Session: sess}
}

func (r Network_Subnet_Registration) Id(id int) Network_Subnet_Registration {
	r.Options.Id = &id
	return r
}

func (r Network_Subnet_Registration) Mask(mask string) Network_Subnet_Registration {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Subnet_Registration) Filter(filter string) Network_Subnet_Registration {
	r.Options.Filter = filter
	return r
}

func (r Network_Subnet_Registration) Limit(limit int) Network_Subnet_Registration {
	r.Options.Limit = &limit
	return r
}

func (r Network_Subnet_Registration) Offset(offset int) Network_Subnet_Registration {
	r.Options.Offset = &offset
	return r
}

// This method will initiate the removal of a subnet registration.
func (r Network_Subnet_Registration) ClearRegistration() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "clearRegistration", nil, &r.Options, &resp)
	return
}

// <style type="text/css">.create_object > li > div { padding-top: .5em; padding-bottom: .5em}</style> This method will create a new SoftLayer_Network_Subnet_Registration object.
//
// <b>Input</b> - [[SoftLayer_Network_Subnet_Registration (type)|SoftLayer_Network_Subnet_Registration]] <ul class="create_object"> <li><code>networkIdentifier</code> <div> The base address of the [[SoftLayer_Network_Subnet|subnet]] being registered. This can be derived directly from the SoftLayer_Network_Subnet object's networkIdentifier property. </div> <ul> <li><b>Required</b></li> <li><b>Type</b> - string</li> </ul> </li> <li><code>cidr</code> <div> The CIDR prefix of the [[SoftLayer_Network_Subnet|subnet]] being registered. This can be derived directly from the SoftLayer_Network_Subnet object's cidr property. </div> <ul> <li><b>Required</b></li> <li><b>Type</b> - integer</li> </ul> </li> </ul>
func (r Network_Subnet_Registration) CreateObject(templateObject *datatypes.Network_Subnet_Registration) (resp datatypes.Network_Subnet_Registration, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "createObject", params, &r.Options, &resp)
	return
}

// This method will edit an existing SoftLayer_Network_Subnet_Registration object. For more detail, see [[SoftLayer_Network_Subnet_Registration::createObject|createObject]].
func (r Network_Subnet_Registration) EditObject(templateObject *datatypes.Network_Subnet_Registration) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "editObject", params, &r.Options, &resp)
	return
}

// This method modifies a single registration by modifying the current [[SoftLayer_Network_Subnet_Registration_Details]] objects that are linked to that registration.
func (r Network_Subnet_Registration) EditRegistrationAttachedDetails(personObjectSkeleton *datatypes.Network_Subnet_Registration_Details, networkObjectSkeleton *datatypes.Network_Subnet_Registration_Details) (resp bool, err error) {
	params := []interface{}{
		personObjectSkeleton,
		networkObjectSkeleton,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "editRegistrationAttachedDetails", params, &r.Options, &resp)
	return
}

// Retrieve The account that this registration belongs to.
func (r Network_Subnet_Registration) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The cross-reference records that tie the [[SoftLayer_Account_Regional_Registry_Detail]] objects to the registration object.
func (r Network_Subnet_Registration) GetDetailReferences() (resp []datatypes.Network_Subnet_Registration_Details, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getDetailReferences", nil, &r.Options, &resp)
	return
}

// Retrieve The related registration events.
func (r Network_Subnet_Registration) GetEvents() (resp []datatypes.Network_Subnet_Registration_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getEvents", nil, &r.Options, &resp)
	return
}

// Retrieve The "network" detail object.
func (r Network_Subnet_Registration) GetNetworkDetail() (resp datatypes.Account_Regional_Registry_Detail, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getNetworkDetail", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Subnet_Registration) GetObject() (resp datatypes.Network_Subnet_Registration, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The "person" detail object.
func (r Network_Subnet_Registration) GetPersonDetail() (resp datatypes.Account_Regional_Registry_Detail, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getPersonDetail", nil, &r.Options, &resp)
	return
}

// Retrieve The related Regional Internet Registry.
func (r Network_Subnet_Registration) GetRegionalInternetRegistry() (resp datatypes.Network_Regional_Internet_Registry, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getRegionalInternetRegistry", nil, &r.Options, &resp)
	return
}

// Retrieve The RIR handle that this registration object belongs to. This field may not be populated until the registration is complete.
func (r Network_Subnet_Registration) GetRegionalInternetRegistryHandle() (resp datatypes.Account_Rwhois_Handle, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getRegionalInternetRegistryHandle", nil, &r.Options, &resp)
	return
}

// Retrieve The status of this registration.
func (r Network_Subnet_Registration) GetStatus() (resp datatypes.Network_Subnet_Registration_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The subnet that this registration pertains to.
func (r Network_Subnet_Registration) GetSubnet() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration", "getSubnet", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Subnet_Registration_Details objects are used to relate [[SoftLayer_Account_Regional_Registry_Detail]] objects to a [[SoftLayer_Network_Subnet_Registration]] object. This allows for easy reuse of registration details. It is important to note that only one detail object per type may be associated to a registration object.
type Network_Subnet_Registration_Details struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSubnetRegistrationDetailsService returns an instance of the Network_Subnet_Registration_Details SoftLayer service
func GetNetworkSubnetRegistrationDetailsService(sess *session.Session) Network_Subnet_Registration_Details {
	return Network_Subnet_Registration_Details{Session: sess}
}

func (r Network_Subnet_Registration_Details) Id(id int) Network_Subnet_Registration_Details {
	r.Options.Id = &id
	return r
}

func (r Network_Subnet_Registration_Details) Mask(mask string) Network_Subnet_Registration_Details {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Subnet_Registration_Details) Filter(filter string) Network_Subnet_Registration_Details {
	r.Options.Filter = filter
	return r
}

func (r Network_Subnet_Registration_Details) Limit(limit int) Network_Subnet_Registration_Details {
	r.Options.Limit = &limit
	return r
}

func (r Network_Subnet_Registration_Details) Offset(offset int) Network_Subnet_Registration_Details {
	r.Options.Offset = &offset
	return r
}

// <style type="text/css">.create_object > li > div { padding-top: .5em; padding-bottom: .5em}</style> This method will create a new SoftLayer_Network_Subnet_Registration_Details object.
//
// <b>Input</b> - [[SoftLayer_Network_Subnet_Registration_Details (type)|SoftLayer_Network_Subnet_Registration_Details]] <ul class="create_object"> <li><code>detailId</code> <div> The numeric ID of the [[SoftLayer_Account_Regional_Registry_Detail|detail]] object to relate. </div> <ul> <li><b>Required</b></li> <li><b>Type</b> - integer</li> </ul> </li> <li><code>registrationId</code> <div> The numeric ID of the [[SoftLayer_Network_Subnet_Registration|registration]] object to relate. </div> <ul> <li><b>Required</b></li> <li><b>Type</b> - integer</li> </ul> </li> </ul>
func (r Network_Subnet_Registration_Details) CreateObject(templateObject *datatypes.Network_Subnet_Registration_Details) (resp datatypes.Network_Subnet_Registration_Details, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration_Details", "createObject", params, &r.Options, &resp)
	return
}

// This method will delete an existing SoftLayer_Account_Regional_Registry_Detail object.
func (r Network_Subnet_Registration_Details) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration_Details", "deleteObject", nil, &r.Options, &resp)
	return
}

// Retrieve The related [[SoftLayer_Account_Regional_Registry_Detail|detail object]].
func (r Network_Subnet_Registration_Details) GetDetail() (resp datatypes.Account_Regional_Registry_Detail, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration_Details", "getDetail", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Subnet_Registration_Details) GetObject() (resp datatypes.Network_Subnet_Registration_Details, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration_Details", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The related [[SoftLayer_Network_Subnet_Registration|registration object]].
func (r Network_Subnet_Registration_Details) GetRegistration() (resp datatypes.Network_Subnet_Registration, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration_Details", "getRegistration", nil, &r.Options, &resp)
	return
}

// Subnet Registration Status objects describe the current status of a subnet registration.
//
// The standard values for these objects are as follows: <ul> <li><strong>OPEN</strong> - Indicates that the registration object is new and has yet to be submitted to the RIR</li> <li><strong>PENDING</strong> - Indicates that the registration object has been submitted to the RIR and is awaiting response</li> <li><strong>COMPLETE</strong> - Indicates that the RIR action has completed</li> <li><strong>DELETED</strong> - Indicates that the registration object has been gracefully removed is no longer valid</li> <li><strong>CANCELLED</strong> - Indicates that the registration object has been abruptly removed is no longer valid</li> </ul>
type Network_Subnet_Registration_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSubnetRegistrationStatusService returns an instance of the Network_Subnet_Registration_Status SoftLayer service
func GetNetworkSubnetRegistrationStatusService(sess *session.Session) Network_Subnet_Registration_Status {
	return Network_Subnet_Registration_Status{Session: sess}
}

func (r Network_Subnet_Registration_Status) Id(id int) Network_Subnet_Registration_Status {
	r.Options.Id = &id
	return r
}

func (r Network_Subnet_Registration_Status) Mask(mask string) Network_Subnet_Registration_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Subnet_Registration_Status) Filter(filter string) Network_Subnet_Registration_Status {
	r.Options.Filter = filter
	return r
}

func (r Network_Subnet_Registration_Status) Limit(limit int) Network_Subnet_Registration_Status {
	r.Options.Limit = &limit
	return r
}

func (r Network_Subnet_Registration_Status) Offset(offset int) Network_Subnet_Registration_Status {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Subnet_Registration_Status) GetAllObjects() (resp []datatypes.Network_Subnet_Registration_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration_Status", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Network_Subnet_Registration_Status) GetObject() (resp datatypes.Network_Subnet_Registration_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Registration_Status", "getObject", nil, &r.Options, &resp)
	return
}

// Every SoftLayer customer account has contact information associated with it for reverse WHOIS purposes. An account's RWHOIS data, modeled by the SoftLayer_Network_Subnet_Rwhois_Data data type, is used by SoftLayer's reverse WHOIS server as well as for SWIP transactions. SoftLayer's reverse WHOIS servers respond to WHOIS queries for IP addresses belonging to a customer's servers, returning this RWHOIS data.
//
// A SoftLayer customer's RWHOIS data may not necessarily match their account or portal users' contact information.
type Network_Subnet_Rwhois_Data struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSubnetRwhoisDataService returns an instance of the Network_Subnet_Rwhois_Data SoftLayer service
func GetNetworkSubnetRwhoisDataService(sess *session.Session) Network_Subnet_Rwhois_Data {
	return Network_Subnet_Rwhois_Data{Session: sess}
}

func (r Network_Subnet_Rwhois_Data) Id(id int) Network_Subnet_Rwhois_Data {
	r.Options.Id = &id
	return r
}

func (r Network_Subnet_Rwhois_Data) Mask(mask string) Network_Subnet_Rwhois_Data {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Subnet_Rwhois_Data) Filter(filter string) Network_Subnet_Rwhois_Data {
	r.Options.Filter = filter
	return r
}

func (r Network_Subnet_Rwhois_Data) Limit(limit int) Network_Subnet_Rwhois_Data {
	r.Options.Limit = &limit
	return r
}

func (r Network_Subnet_Rwhois_Data) Offset(offset int) Network_Subnet_Rwhois_Data {
	r.Options.Offset = &offset
	return r
}

// Edit the RWHOIS record by passing in a modified version of the record object.  All fields are editable.
func (r Network_Subnet_Rwhois_Data) EditObject(templateObject *datatypes.Network_Subnet_Rwhois_Data) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Rwhois_Data", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer customer account associated with this reverse WHOIS data.
func (r Network_Subnet_Rwhois_Data) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Rwhois_Data", "getAccount", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Subnet_Rwhois_Data object whose ID corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Subnet_Rwhois_Data service.
//
// The best way to get Rwhois Data for an account is through getRhwoisData on the Account service.
func (r Network_Subnet_Rwhois_Data) GetObject() (resp datatypes.Network_Subnet_Rwhois_Data, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Rwhois_Data", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Subnet_Swip_Transaction data type contains basic information tracked at SoftLayer to allow automation of Swip creation, update, and removal requests.  A specific transaction is attached to an accountId and a subnetId. This also contains a "Status Name" which tells the customer what the transaction is doing:
//
//
// * REQUEST QUEUED:  Request is queued up to be sent to ARIN
// * REQUEST SENT:  The email request has been sent to ARIN
// * REQUEST CONFIRMED:  ARIN has confirmed that the request is good, and should be available in 24 hours
// * OK:  The subnet has been checked with WHOIS and it the SWIP transaction has completed correctly
// * REMOVE QUEUED:  A subnet is queued to be removed from ARIN's systems
// * REMOVE SENT:  The removal email request has been sent to ARIN
// * REMOVE CONFIRMED:  ARIN has confirmed that the removal request is good, and the subnet should be clear in WHOIS in 24 hours
// * DELETED:  This specific SWIP Transaction has been removed from ARIN and is no longer in effect
// * SOFTLAYER MANUALLY PROCESSING:  Sometimes a request doesn't go through correctly and has to be manually processed by SoftLayer.  This may take some time.
type Network_Subnet_Swip_Transaction struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkSubnetSwipTransactionService returns an instance of the Network_Subnet_Swip_Transaction SoftLayer service
func GetNetworkSubnetSwipTransactionService(sess *session.Session) Network_Subnet_Swip_Transaction {
	return Network_Subnet_Swip_Transaction{Session: sess}
}

func (r Network_Subnet_Swip_Transaction) Id(id int) Network_Subnet_Swip_Transaction {
	r.Options.Id = &id
	return r
}

func (r Network_Subnet_Swip_Transaction) Mask(mask string) Network_Subnet_Swip_Transaction {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Subnet_Swip_Transaction) Filter(filter string) Network_Subnet_Swip_Transaction {
	r.Options.Filter = filter
	return r
}

func (r Network_Subnet_Swip_Transaction) Limit(limit int) Network_Subnet_Swip_Transaction {
	r.Options.Limit = &limit
	return r
}

func (r Network_Subnet_Swip_Transaction) Offset(offset int) Network_Subnet_Swip_Transaction {
	r.Options.Offset = &offset
	return r
}

// This function will return an array of SoftLayer_Network_Subnet_Swip_Transaction objects, one for each SWIP that is currently in transaction with ARIN.  This includes all swip registrations, swip removal requests, and SWIP objects that are currently OK.
func (r Network_Subnet_Swip_Transaction) FindMyTransactions() (resp []datatypes.Network_Subnet_Swip_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "findMyTransactions", nil, &r.Options, &resp)
	return
}

// Retrieve The Account whose RWHOIS data was used to SWIP this subnet
func (r Network_Subnet_Swip_Transaction) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "getAccount", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Subnet_Swip_Transaction object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Subnet_Swip_transaction service. You can only retrieve Swip transactions tied to the account.
func (r Network_Subnet_Swip_Transaction) GetObject() (resp datatypes.Network_Subnet_Swip_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The subnet that this SWIP transaction was created for.
func (r Network_Subnet_Swip_Transaction) GetSubnet() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "getSubnet", nil, &r.Options, &resp)
	return
}

// This method finds all subnets attached to your account that are in OK status and starts "DELETE" transactions with ARIN, allowing you to remove your SWIP registration information.
func (r Network_Subnet_Swip_Transaction) RemoveAllSubnetSwips() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "removeAllSubnetSwips", nil, &r.Options, &resp)
	return
}

// This function, when called on an instantiated SWIP transaction, will allow you to start a "DELETE" transaction with ARIN, allowing you to remove your SWIP registration information.
func (r Network_Subnet_Swip_Transaction) RemoveSwipData() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "removeSwipData", nil, &r.Options, &resp)
	return
}

// This function will allow you to update ARIN's registration data for a subnet to your current RWHOIS data.
func (r Network_Subnet_Swip_Transaction) ResendSwipData() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "resendSwipData", nil, &r.Options, &resp)
	return
}

// swipAllSubnets finds all subnets attached to your account and attempts to create a SWIP transaction for all subnets that do not already have a SWIP transaction in progress.
func (r Network_Subnet_Swip_Transaction) SwipAllSubnets() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "swipAllSubnets", nil, &r.Options, &resp)
	return
}

// This method finds all subnets attached to your account that are in "OK" status and updates their data with ARIN.  Use this function after you have updated your RWHOIS data if you want to keep SWIP up to date.
func (r Network_Subnet_Swip_Transaction) UpdateAllSubnetSwips() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Subnet_Swip_Transaction", "updateAllSubnetSwips", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Network_TippingPointReporting struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkTippingPointReportingService returns an instance of the Network_TippingPointReporting SoftLayer service
func GetNetworkTippingPointReportingService(sess *session.Session) Network_TippingPointReporting {
	return Network_TippingPointReporting{Session: sess}
}

func (r Network_TippingPointReporting) Id(id int) Network_TippingPointReporting {
	r.Options.Id = &id
	return r
}

func (r Network_TippingPointReporting) Mask(mask string) Network_TippingPointReporting {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_TippingPointReporting) Filter(filter string) Network_TippingPointReporting {
	r.Options.Filter = filter
	return r
}

func (r Network_TippingPointReporting) Limit(limit int) Network_TippingPointReporting {
	r.Options.Limit = &limit
	return r
}

func (r Network_TippingPointReporting) Offset(offset int) Network_TippingPointReporting {
	r.Options.Offset = &offset
	return r
}

// This method, when given an attack signature ID (available in the return values of getReportForIpAddressOrSubnet and  getSubnetReportForEntireAccount) and an IP Address and subnet mask, returns all attacks for that subnet in the specified time frame and direction.  Once the results have been filtered, additional data is available, including starting and ending times for the attack, originating IP address and port, and destination IP address and port.
//
// CVE and Bugtraq information is not available at this level.
func (r Network_TippingPointReporting) DrillDownAttack(signatureId *string, IpAddress *string, subnetMask *int, timeFrame *int, direction *string) (resp datatypes.Container_Network_IntrusionProtection_SubnetReport, err error) {
	params := []interface{}{
		signatureId,
		IpAddress,
		subnetMask,
		timeFrame,
		direction,
	}
	err = r.Session.DoRequest("SoftLayer_Network_TippingPointReporting", "drillDownAttack", params, &r.Options, &resp)
	return
}

// This method returns the attack statistics for the current user's account and for the entire SoftLayer network.  These attacks are recorded and monitored at the entry point to the network, and represent attacks in both directions.
//
// The data returned is:
// * Top attacks (by attack name) on datacenter Dal01 in the last hour (and last 24 hours)
// * Top attacks (by attack name) on IPs you own in the last hour (and last 24 hours)
// * Top IPs attacking IPs you own in the last hour (and last 24 hours)
// Each one of these lists can contain any number of items, the default is 5.  The usable limit is less than 10, but setting the limit to an abnormally high value will effectively return all records.
//
// The data is returned as a collection of SoftLayer_Container_Network_IntrusionProtection_Statistics objects.
func (r Network_TippingPointReporting) GetMainStatistics(numberOfAttacks *int) (resp []datatypes.Container_Network_IntrusionProtection_Statistics, err error) {
	params := []interface{}{
		numberOfAttacks,
	}
	err = r.Session.DoRequest("SoftLayer_Network_TippingPointReporting", "getMainStatistics", params, &r.Options, &resp)
	return
}

// This method expands on the getSubnetReportForEntireAccount method by offering the ability to filter by subnet or IP address. This method is identical to getSubnetReportForEntireAccount, but allows filtering by subnet.  Like in the getSubnetReportForEntireAccount method, CVE and BugTraq IDs are provided, if available.
//
// This method should be called once an attack has been identified using getSubnetReportForEntireAccount (in which case "All Subnets" is the subnet) or getReportForIpAddressOrSubnet.
func (r Network_TippingPointReporting) GetReportForIpAddressOrSubnet(IpAddress *string, subnetMask *int, timeFrame *int, orderBy *string, orderDirection *string) (resp []datatypes.Container_Network_IntrusionProtection_SubnetReport, err error) {
	params := []interface{}{
		IpAddress,
		subnetMask,
		timeFrame,
		orderBy,
		orderDirection,
	}
	err = r.Session.DoRequest("SoftLayer_Network_TippingPointReporting", "getReportForIpAddressOrSubnet", params, &r.Options, &resp)
	return
}

// This method returns specific attacks by name for all subnets on the current user's account.
//
// The data returned is stored in SoftLayer_Container_Network_IntrusionProtection_SubnetReport objects, with the "subnet" value set to "All Subnets"
//
// The data is separated into "Inbound" and "Outbound" traffic.  A significant amount of outbound attack traffic could indicate that your servers have been compromised.
//
// The data returned includes Attack Count, attack name, extended attack description, and IDs that correspond with the BugTraq or CVE databases. BugTraq can be accessed at [http://www.securityfocus.com/vulnerabilities] The CVE database is located at [http://cve.mitre.org/find/index.html]
//
// For more detailed information, use the getReportForIpAddressOrSubnet method
func (r Network_TippingPointReporting) GetSubnetReportForEntireAccount(timeFrame *int, orderBy *string, orderDirection *string, returnSubnetGroups *bool) (resp []datatypes.Container_Network_IntrusionProtection_SubnetReport, err error) {
	params := []interface{}{
		timeFrame,
		orderBy,
		orderDirection,
		returnSubnetGroups,
	}
	err = r.Session.DoRequest("SoftLayer_Network_TippingPointReporting", "getSubnetReportForEntireAccount", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Tunnel_Module_Context data type contains general information relating to a single SoftLayer network tunnel.  The SoftLayer_Network_Tunnel_Module_Context is useful to gather information such as related customer subnets (remote) and internal subnets (local) associated with the network tunnel as well as other information needed to manage the network tunnel.  Account and billing information related to the network tunnel can also be retrieved.
type Network_Tunnel_Module_Context struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkTunnelModuleContextService returns an instance of the Network_Tunnel_Module_Context SoftLayer service
func GetNetworkTunnelModuleContextService(sess *session.Session) Network_Tunnel_Module_Context {
	return Network_Tunnel_Module_Context{Session: sess}
}

func (r Network_Tunnel_Module_Context) Id(id int) Network_Tunnel_Module_Context {
	r.Options.Id = &id
	return r
}

func (r Network_Tunnel_Module_Context) Mask(mask string) Network_Tunnel_Module_Context {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Tunnel_Module_Context) Filter(filter string) Network_Tunnel_Module_Context {
	r.Options.Filter = filter
	return r
}

func (r Network_Tunnel_Module_Context) Limit(limit int) Network_Tunnel_Module_Context {
	r.Options.Limit = &limit
	return r
}

func (r Network_Tunnel_Module_Context) Offset(offset int) Network_Tunnel_Module_Context {
	r.Options.Offset = &offset
	return r
}

// Associates a remote subnet to the network tunnel.  When a remote subnet is associated, a network tunnel will allow the customer (remote) network to communicate with the private and service subnets on the SoftLayer network which are on the other end of this network tunnel.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for the association described above to take effect.
func (r Network_Tunnel_Module_Context) AddCustomerSubnetToNetworkTunnel(subnetId *int) (resp bool, err error) {
	params := []interface{}{
		subnetId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "addCustomerSubnetToNetworkTunnel", params, &r.Options, &resp)
	return
}

// Associates a private subnet to the network tunnel.  When a private subnet is associated, the network tunnel will allow the customer (remote) network to access the private subnet.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for the association described above to take effect.
func (r Network_Tunnel_Module_Context) AddPrivateSubnetToNetworkTunnel(subnetId *int) (resp bool, err error) {
	params := []interface{}{
		subnetId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "addPrivateSubnetToNetworkTunnel", params, &r.Options, &resp)
	return
}

// Associates a service subnet to the network tunnel.  When a service subnet is associated, a network tunnel will allow the customer (remote) network to communicate with the private and service subnets on the SoftLayer network which are on the other end of this network tunnel.  Service subnets provide access to SoftLayer services such as the customer management portal and the SoftLayer API.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for the association described above to take effect.
func (r Network_Tunnel_Module_Context) AddServiceSubnetToNetworkTunnel(subnetId *int) (resp bool, err error) {
	params := []interface{}{
		subnetId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "addServiceSubnetToNetworkTunnel", params, &r.Options, &resp)
	return
}

// A transaction will be created to apply the IPSec network tunnel's configuration to SoftLayer network devices.  During this time, an IPSec network tunnel cannot be modified in anyway.  Only one network tunnel configuration transaction can be created.  If a transaction has been created or is running, a new transaction cannot be created until the previous transaction completes.
func (r Network_Tunnel_Module_Context) ApplyConfigurationsToDevice() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "applyConfigurationsToDevice", nil, &r.Options, &resp)
	return
}

// Create an address translation for a network tunnel.
//
// To create an address translation, ip addresses from an assigned /30 static route subnet are used.  Address translations deliver packets to a destination ip address that is on a customer (remote) subnet.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for an address translation to be created.
func (r Network_Tunnel_Module_Context) CreateAddressTranslation(translation *datatypes.Network_Tunnel_Module_Context_Address_Translation) (resp datatypes.Network_Tunnel_Module_Context_Address_Translation, err error) {
	params := []interface{}{
		translation,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "createAddressTranslation", params, &r.Options, &resp)
	return
}

// This has the same functionality as the SoftLayer_Network_Tunnel_Module_Context::createAddressTranslation.  However, it allows multiple translations to be passed in for creation.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for the address translations to be created.
func (r Network_Tunnel_Module_Context) CreateAddressTranslations(translations []datatypes.Network_Tunnel_Module_Context_Address_Translation) (resp []datatypes.Network_Tunnel_Module_Context_Address_Translation, err error) {
	params := []interface{}{
		translations,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "createAddressTranslations", params, &r.Options, &resp)
	return
}

// Remove an existing address translation from a network tunnel.
//
// Address translations deliver packets to a destination ip address that is on a customer subnet (remote).
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for an address translation to be deleted.
func (r Network_Tunnel_Module_Context) DeleteAddressTranslation(translationId *int) (resp bool, err error) {
	params := []interface{}{
		translationId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "deleteAddressTranslation", params, &r.Options, &resp)
	return
}

// Provides all of the address translation configurations for an IPSec VPN tunnel in a text file
func (r Network_Tunnel_Module_Context) DownloadAddressTranslationConfigurations() (resp datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "downloadAddressTranslationConfigurations", nil, &r.Options, &resp)
	return
}

// Provides all of the configurations for an IPSec VPN network tunnel in a text file
func (r Network_Tunnel_Module_Context) DownloadParameterConfigurations() (resp datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "downloadParameterConfigurations", nil, &r.Options, &resp)
	return
}

// Edit name, source (SoftLayer IP) ip address and/or destination (Customer IP) ip address for an existing address translation for a network tunnel.
//
// Address translations deliver packets to a destination ip address that is on a customer (remote) subnet.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for an address translation to be created.
func (r Network_Tunnel_Module_Context) EditAddressTranslation(translation *datatypes.Network_Tunnel_Module_Context_Address_Translation) (resp datatypes.Network_Tunnel_Module_Context_Address_Translation, err error) {
	params := []interface{}{
		translation,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "editAddressTranslation", params, &r.Options, &resp)
	return
}

// Edit name, source (SoftLayer IP) ip address and/or destination (Customer IP) ip address for existing address translations for a network tunnel.
//
// Address translations deliver packets to a destination ip address that is on a customer (remote) subnet.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for an address translation to be modified.
func (r Network_Tunnel_Module_Context) EditAddressTranslations(translations []datatypes.Network_Tunnel_Module_Context_Address_Translation) (resp []datatypes.Network_Tunnel_Module_Context_Address_Translation, err error) {
	params := []interface{}{
		translations,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "editAddressTranslations", params, &r.Options, &resp)
	return
}

// Negotiation parameters for both phases one and two are editable. Here are the phase one and two parameters that can modified:
//
//
// *Phase One
// **Authentication
// ***Default value is set to MD5.
// ***Valid Options are: MD5, SHA1, SHA256.
// **Encryption
// ***Default value is set to 3DES.
// ***Valid Options are: DES, 3DES, AES128, AES192, AES256.
// **Diffie-Hellman Group
// ***Default value is set to 2.
// ***Valid Options are: 0 (None), 1, 2, 5.
// **Keylife
// ***Default value is set to 3600.
// ***Limits are:  MIN = 120, MAX = 172800
// **Preshared Key
// *Phase Two
// **Authentication
// ***Default value is set to MD5.
// ***Valid Options are: MD5, SHA1, SHA256.
// **Encryption
// ***Default value is set to 3DES.
// ***Valid Options are: DES, 3DES, AES128, AES192, AES256.
// **Diffie-Hellman Group
// ***Default value is set to 2.
// ***Valid Options are: 0 (None), 1, 2, 5.
// **Keylife
// ***Default value is set to 28800.
// ***Limits are:  MIN = 120, MAX = 172800
// **Perfect Forward Secrecy
// ***Valid Options are: Off = 0, On = 1.
// ***NOTE:  If perfect forward secrecy is turned On (set to 1), then a phase 2 diffie-hellman group is required.
//
//
// The remote peer address for the network tunnel may also be modified if needed.  Invalid options will not be accepted and will cause an exception to be thrown.  There are properties that provide valid options and limits for each negotiation parameter.  Those properties are as follows:
// * encryptionDefault
// * encryptionOptions
// * authenticationDefault
// * authenticationOptions
// * diffieHellmanGroupDefault
// * diffieHellmanGroupOptions
// * phaseOneKeylifeDefault
// * phaseTwoKeylifeDefault
// * keylifeLimits
//
//
// Configurations cannot be modified if a network tunnel's requires complex manual setups/configuration modifications by the SoftLayer Network department.  If the former is required, the configurations for the network tunnel will be locked until the manual configurations are complete. A network tunnel's configurations are applied via a transaction.  If a network tunnel configuration change transaction is currently running, the network tunnel's setting cannot be modified until the running transaction completes.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for the modifications made to take effect.
func (r Network_Tunnel_Module_Context) EditObject(templateObject *datatypes.Network_Tunnel_Module_Context) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account that a network tunnel belongs to.
func (r Network_Tunnel_Module_Context) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The transaction that is currently applying configurations for the network tunnel.
func (r Network_Tunnel_Module_Context) GetActiveTransaction() (resp datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getActiveTransaction", nil, &r.Options, &resp)
	return
}

// The address translations will be returned.  All the translations will be formatted so that the configurations can be copied into a host file.
//
// Format:
//
// {address translation SoftLayer IP Address}        {address translation name}
func (r Network_Tunnel_Module_Context) GetAddressTranslationConfigurations() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getAddressTranslationConfigurations", nil, &r.Options, &resp)
	return
}

// Retrieve A network tunnel's address translations.
func (r Network_Tunnel_Module_Context) GetAddressTranslations() (resp []datatypes.Network_Tunnel_Module_Context_Address_Translation, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getAddressTranslations", nil, &r.Options, &resp)
	return
}

// Retrieve Subnets that provide access to SoftLayer services such as the management portal and the SoftLayer API.
func (r Network_Tunnel_Module_Context) GetAllAvailableServiceSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getAllAvailableServiceSubnets", nil, &r.Options, &resp)
	return
}

// The default authentication type used for both phases of the negotiation process.  The default value is set to MD5.
func (r Network_Tunnel_Module_Context) GetAuthenticationDefault() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getAuthenticationDefault", nil, &r.Options, &resp)
	return
}

// Authentication options available for both phases of the negotiation process.
//
// The authentication options are as follows:
// * MD5
// * SHA1
// * SHA256
func (r Network_Tunnel_Module_Context) GetAuthenticationOptions() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getAuthenticationOptions", nil, &r.Options, &resp)
	return
}

// Retrieve The current billing item for network tunnel.
func (r Network_Tunnel_Module_Context) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve Remote subnets that are allowed access through a network tunnel.
func (r Network_Tunnel_Module_Context) GetCustomerSubnets() (resp []datatypes.Network_Customer_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getCustomerSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The datacenter location for one end of the network tunnel that allows access to account's private subnets.
func (r Network_Tunnel_Module_Context) GetDatacenter() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getDatacenter", nil, &r.Options, &resp)
	return
}

// The default Diffie-Hellman group used for both phases of the negotiation process.  The default value is set to 2.
func (r Network_Tunnel_Module_Context) GetDiffieHellmanGroupDefault() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getDiffieHellmanGroupDefault", nil, &r.Options, &resp)
	return
}

// The Diffie-Hellman group options used for both phases of the negotiation process.
//
// The diffie-hellman group options are as follows:
// * 0 (None)
// * 1
// * 2
// * 5
func (r Network_Tunnel_Module_Context) GetDiffieHellmanGroupOptions() (resp []int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getDiffieHellmanGroupOptions", nil, &r.Options, &resp)
	return
}

// The default encryption type used for both phases of the negotiation process.  The default value is set to 3DES.
func (r Network_Tunnel_Module_Context) GetEncryptionDefault() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getEncryptionDefault", nil, &r.Options, &resp)
	return
}

// Encryption options available for both phases of the negotiation process.
//
// The valid encryption options are as follows:
// * DES
// * 3DES
// * AES128
// * AES192
// * AES256
func (r Network_Tunnel_Module_Context) GetEncryptionOptions() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getEncryptionOptions", nil, &r.Options, &resp)
	return
}

// Retrieve Private subnets that can be accessed through the network tunnel.
func (r Network_Tunnel_Module_Context) GetInternalSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getInternalSubnets", nil, &r.Options, &resp)
	return
}

// The keylife limits.  Keylife max limit is set to 120.  Keylife min limit is set to 172800.
func (r Network_Tunnel_Module_Context) GetKeylifeLimits() (resp []int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getKeylifeLimits", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Tunnel_Module_Context object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Tunnel_Module_Context service. The IPSec network tunnel will be returned if it is associated with the account and the user has proper permission to manage network tunnels.
func (r Network_Tunnel_Module_Context) GetObject() (resp datatypes.Network_Tunnel_Module_Context, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getObject", nil, &r.Options, &resp)
	return
}

// All of the IPSec VPN tunnel's configurations will be returned.  It will list all of phase one and two negotiation parameters.  Both remote and local subnets will be provided as well.  This is useful when the configurations need to be passed on to another team and/or company for internal network configuration.
func (r Network_Tunnel_Module_Context) GetParameterConfigurationsForCustomerView() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getParameterConfigurationsForCustomerView", nil, &r.Options, &resp)
	return
}

// The default phase 1 keylife used if a value is not provided.  The default value is set to 3600.
func (r Network_Tunnel_Module_Context) GetPhaseOneKeylifeDefault() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getPhaseOneKeylifeDefault", nil, &r.Options, &resp)
	return
}

// The default phase 2 keylife used if a value is not provided.  The default value is set to 28800.
func (r Network_Tunnel_Module_Context) GetPhaseTwoKeylifeDefault() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getPhaseTwoKeylifeDefault", nil, &r.Options, &resp)
	return
}

// Retrieve Service subnets that can be access through the network tunnel.
func (r Network_Tunnel_Module_Context) GetServiceSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getServiceSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve Subnets used for a network tunnel's address translations.
func (r Network_Tunnel_Module_Context) GetStaticRouteSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getStaticRouteSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The transaction history for this network tunnel.
func (r Network_Tunnel_Module_Context) GetTransactionHistory() (resp []datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "getTransactionHistory", nil, &r.Options, &resp)
	return
}

// Disassociate a customer subnet (remote) from a network tunnel.  When a remote subnet is disassociated, that subnet will not able to communicate with private and service subnets on the SoftLayer network.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for the disassociation described above to take effect.
func (r Network_Tunnel_Module_Context) RemoveCustomerSubnetFromNetworkTunnel(subnetId *int) (resp bool, err error) {
	params := []interface{}{
		subnetId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "removeCustomerSubnetFromNetworkTunnel", params, &r.Options, &resp)
	return
}

// Disassociate a private subnet from a network tunnel.  When a private subnet is disassociated, the customer (remote) subnet on the other end of the tunnel will not able to communicate with the private subnet that was just disassociated.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for the disassociation described above to take effect.
func (r Network_Tunnel_Module_Context) RemovePrivateSubnetFromNetworkTunnel(subnetId *int) (resp bool, err error) {
	params := []interface{}{
		subnetId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "removePrivateSubnetFromNetworkTunnel", params, &r.Options, &resp)
	return
}

// Disassociate a service subnet from a network tunnel.  When a service subnet is disassociated, that customer (remote) subnet on the other end of the network tunnel will not able to communicate with that service subnet on the SoftLayer network.
//
// NOTE:  A network tunnel's configurations must be applied to the network device in order for the disassociation described above to take effect.
func (r Network_Tunnel_Module_Context) RemoveServiceSubnetFromNetworkTunnel(subnetId *int) (resp bool, err error) {
	params := []interface{}{
		subnetId,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Tunnel_Module_Context", "removeServiceSubnetFromNetworkTunnel", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Vlan data type models a single VLAN within SoftLayer's public and private networks. a Virtual LAN is a structure that associates network interfaces on routers, switches, and servers in different locations to act as if they were on the same local network broadcast domain. VLANs are a central part of the SoftLayer network. They can determine how new IP subnets are routed and how individual servers communicate to each other.
type Network_Vlan struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkVlanService returns an instance of the Network_Vlan SoftLayer service
func GetNetworkVlanService(sess *session.Session) Network_Vlan {
	return Network_Vlan{Session: sess}
}

func (r Network_Vlan) Id(id int) Network_Vlan {
	r.Options.Id = &id
	return r
}

func (r Network_Vlan) Mask(mask string) Network_Vlan {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Vlan) Filter(filter string) Network_Vlan {
	r.Options.Filter = filter
	return r
}

func (r Network_Vlan) Limit(limit int) Network_Vlan {
	r.Options.Limit = &limit
	return r
}

func (r Network_Vlan) Offset(offset int) Network_Vlan {
	r.Options.Offset = &offset
	return r
}

// Edit a VLAN's properties
func (r Network_Vlan) EditObject(templateObject *datatypes.Network_Vlan) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer customer account associated with a VLAN.
func (r Network_Vlan) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve A VLAN's additional primary subnets. These are used to extend the number of servers attached to the VLAN by adding more ip addresses to the primary IP address pool.
func (r Network_Vlan) GetAdditionalPrimarySubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getAdditionalPrimarySubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The gateway this VLAN is inside of.
func (r Network_Vlan) GetAttachedNetworkGateway() (resp datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getAttachedNetworkGateway", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not this VLAN is inside a gateway.
func (r Network_Vlan) GetAttachedNetworkGatewayFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getAttachedNetworkGatewayFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The inside VLAN record if this VLAN is inside a network gateway.
func (r Network_Vlan) GetAttachedNetworkGatewayVlan() (resp datatypes.Network_Gateway_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getAttachedNetworkGatewayVlan", nil, &r.Options, &resp)
	return
}

// Retrieve The billing item for a network vlan.
func (r Network_Vlan) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Get a set of reasons why this VLAN may not be cancelled. If the result is empty, this VLAN may be cancelled.
func (r Network_Vlan) GetCancelFailureReasons() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getCancelFailureReasons", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating that a network vlan is on a Hardware Firewall (Dedicated).
func (r Network_Vlan) GetDedicatedFirewallFlag() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getDedicatedFirewallFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The extension router that a VLAN is associated with.
func (r Network_Vlan) GetExtensionRouter() (resp datatypes.Hardware_Router, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getExtensionRouter", nil, &r.Options, &resp)
	return
}

// Retrieve A firewalled Vlan's network components.
func (r Network_Vlan) GetFirewallGuestNetworkComponents() (resp []datatypes.Network_Component_Firewall, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getFirewallGuestNetworkComponents", nil, &r.Options, &resp)
	return
}

// Retrieve A firewalled vlan's inbound/outbound interfaces.
func (r Network_Vlan) GetFirewallInterfaces() (resp []datatypes.Network_Firewall_Module_Context_Interface, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getFirewallInterfaces", nil, &r.Options, &resp)
	return
}

// Retrieve A firewalled Vlan's network components.
func (r Network_Vlan) GetFirewallNetworkComponents() (resp []datatypes.Network_Component_Firewall, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getFirewallNetworkComponents", nil, &r.Options, &resp)
	return
}

// Get the IP addresses associated with this server that are protectable by a network component firewall. Note, this may not return all values for IPv6 subnets for this VLAN. Please use getFirewallProtectableSubnets to get all protectable subnets.
func (r Network_Vlan) GetFirewallProtectableIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getFirewallProtectableIpAddresses", nil, &r.Options, &resp)
	return
}

// Get the subnets associated with this server that are protectable by a network component firewall.
func (r Network_Vlan) GetFirewallProtectableSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getFirewallProtectableSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The currently running rule set of a firewalled VLAN.
func (r Network_Vlan) GetFirewallRules() (resp []datatypes.Network_Vlan_Firewall_Rule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getFirewallRules", nil, &r.Options, &resp)
	return
}

// Retrieve The networking components that are connected to a VLAN.
func (r Network_Vlan) GetGuestNetworkComponents() (resp []datatypes.Virtual_Guest_Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getGuestNetworkComponents", nil, &r.Options, &resp)
	return
}

// Retrieve All of the hardware that exists on a VLAN. Hardware is associated with a VLAN by its networking components.
func (r Network_Vlan) GetHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Vlan) GetHighAvailabilityFirewallFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getHighAvailabilityFirewallFlag", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating that a vlan can be assigned to a host that has local disk functionality.
func (r Network_Vlan) GetLocalDiskStorageCapabilityFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getLocalDiskStorageCapabilityFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The network in which this VLAN resides.
func (r Network_Vlan) GetNetwork() (resp datatypes.Network, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getNetwork", nil, &r.Options, &resp)
	return
}

// Retrieve The network components that are connected to this VLAN through a trunk.
func (r Network_Vlan) GetNetworkComponentTrunks() (resp []datatypes.Network_Component_Network_Vlan_Trunk, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getNetworkComponentTrunks", nil, &r.Options, &resp)
	return
}

// Retrieve The networking components that are connected to a VLAN.
func (r Network_Vlan) GetNetworkComponents() (resp []datatypes.Network_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getNetworkComponents", nil, &r.Options, &resp)
	return
}

// Retrieve Identifier to denote whether a VLAN is used for public or private connectivity.
func (r Network_Vlan) GetNetworkSpace() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getNetworkSpace", nil, &r.Options, &resp)
	return
}

// Retrieve The Hardware Firewall (Dedicated) for a network vlan.
func (r Network_Vlan) GetNetworkVlanFirewall() (resp datatypes.Network_Vlan_Firewall, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getNetworkVlanFirewall", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Network_Vlan object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Network_Vlan service. You can only retrieve VLANs that are associated with your SoftLayer customer account.
func (r Network_Vlan) GetObject() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The primary router that a VLAN is associated with. Every SoftLayer VLAN is connected to more than one router for greater network redundancy.
func (r Network_Vlan) GetPrimaryRouter() (resp datatypes.Hardware_Router, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPrimaryRouter", nil, &r.Options, &resp)
	return
}

// Retrieve A VLAN's primary subnet. Each VLAN has at least one subnet, usually the subnet that is assigned to a server or new IP address block when it's purchased.
func (r Network_Vlan) GetPrimarySubnet() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPrimarySubnet", nil, &r.Options, &resp)
	return
}

// Retrieve A VLAN's primary IPv6 subnet. Some VLAN's may not have a primary IPv6 subnet.
func (r Network_Vlan) GetPrimarySubnetVersion6() (resp datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPrimarySubnetVersion6", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Vlan) GetPrimarySubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPrimarySubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The gateways this VLAN is the private VLAN of.
func (r Network_Vlan) GetPrivateNetworkGateways() (resp []datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPrivateNetworkGateways", nil, &r.Options, &resp)
	return
}

// Retrieve a VLAN's associated private network VLAN. getPrivateVlan gathers it's information by retrieving the private VLAN of a VLAN's primary hardware object.
func (r Network_Vlan) GetPrivateVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPrivateVlan", nil, &r.Options, &resp)
	return
}

// Retrieve the private network VLAN associated with an IP address.
func (r Network_Vlan) GetPrivateVlanByIpAddress(ipAddress *string) (resp datatypes.Network_Vlan, err error) {
	params := []interface{}{
		ipAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPrivateVlanByIpAddress", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Vlan) GetProtectedIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getProtectedIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The gateways this VLAN is the public VLAN of.
func (r Network_Vlan) GetPublicNetworkGateways() (resp []datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPublicNetworkGateways", nil, &r.Options, &resp)
	return
}

// Retrieve the VLAN that belongs to a server's public network interface, as described by a server's fully-qualified domain name. A server's ''FQDN'' is it's hostname, followed by a period then it's domain name.
func (r Network_Vlan) GetPublicVlanByFqdn(fqdn *string) (resp datatypes.Network_Vlan, err error) {
	params := []interface{}{
		fqdn,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getPublicVlanByFqdn", params, &r.Options, &resp)
	return
}

// Retrieve The resource group member for a network vlan.
func (r Network_Vlan) GetResourceGroupMember() (resp []datatypes.Resource_Group_Member, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getResourceGroupMember", nil, &r.Options, &resp)
	return
}

// Retrieve The resource groups in which this VLAN is a member.
func (r Network_Vlan) GetResourceGroups() (resp []datatypes.Resource_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getResourceGroups", nil, &r.Options, &resp)
	return
}

// Retrieve all reverse DNS records associated with the subnets assigned to a VLAN.
func (r Network_Vlan) GetReverseDomainRecords() (resp []datatypes.Dns_Domain, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getReverseDomainRecords", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating that a vlan can be assigned to a host that has SAN disk functionality.
func (r Network_Vlan) GetSanStorageCapabilityFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getSanStorageCapabilityFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Collection of scale VLANs this VLAN applies to.
func (r Network_Vlan) GetScaleVlans() (resp []datatypes.Scale_Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getScaleVlans", nil, &r.Options, &resp)
	return
}

// Retrieve The secondary router that a VLAN is associated with. Every SoftLayer VLAN is connected to more than one router for greater network redundancy.
func (r Network_Vlan) GetSecondaryRouter() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getSecondaryRouter", nil, &r.Options, &resp)
	return
}

// Retrieve The subnets that exist as secondary interfaces on a VLAN
func (r Network_Vlan) GetSecondarySubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getSecondarySubnets", nil, &r.Options, &resp)
	return
}

// Retrieve All of the subnets that exist as VLAN interfaces.
func (r Network_Vlan) GetSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve References to all tags for this VLAN.
func (r Network_Vlan) GetTagReferences() (resp []datatypes.Tag_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getTagReferences", nil, &r.Options, &resp)
	return
}

// Retrieve The number of primary IP addresses in a VLAN.
func (r Network_Vlan) GetTotalPrimaryIpAddressCount() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getTotalPrimaryIpAddressCount", nil, &r.Options, &resp)
	return
}

// Retrieve The type of this VLAN.
func (r Network_Vlan) GetType() (resp datatypes.Network_Vlan_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve All of the Virtual Servers that are connected to a VLAN.
func (r Network_Vlan) GetVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve the VLAN associated with an IP address via the IP's associated subnet.
func (r Network_Vlan) GetVlanForIpAddress(ipAddress *string) (resp datatypes.Network_Vlan, err error) {
	params := []interface{}{
		ipAddress,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "getVlanForIpAddress", params, &r.Options, &resp)
	return
}

// Tag a VLAN by passing in one or more tags separated by a comma. Tag references are cleared out every time this method is called. If your VLAN is already tagged you will need to pass the current tags along with any new ones. To remove all tag references pass an empty string. To remove one or more tags omit them from the tag list.
func (r Network_Vlan) SetTags(tags *string) (resp bool, err error) {
	params := []interface{}{
		tags,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "setTags", params, &r.Options, &resp)
	return
}

// The '''getSensorData''' method updates a VLAN's firewall to allow or disallow intra-VLAN communication.
func (r Network_Vlan) UpdateFirewallIntraVlanCommunication(enabled *bool) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		enabled,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Vlan", "updateFirewallIntraVlanCommunication", params, &r.Options, &resp)
	return
}

// The SoftLayer_Network_Vlan_Firewall data type contains general information relating to a single SoftLayer VLAN firewall. This is the object which ties the running rules to a specific downstream server. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Vlan_Firewall struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkVlanFirewallService returns an instance of the Network_Vlan_Firewall SoftLayer service
func GetNetworkVlanFirewallService(sess *session.Session) Network_Vlan_Firewall {
	return Network_Vlan_Firewall{Session: sess}
}

func (r Network_Vlan_Firewall) Id(id int) Network_Vlan_Firewall {
	r.Options.Id = &id
	return r
}

func (r Network_Vlan_Firewall) Mask(mask string) Network_Vlan_Firewall {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Vlan_Firewall) Filter(filter string) Network_Vlan_Firewall {
	r.Options.Filter = filter
	return r
}

func (r Network_Vlan_Firewall) Limit(limit int) Network_Vlan_Firewall {
	r.Options.Limit = &limit
	return r
}

func (r Network_Vlan_Firewall) Offset(offset int) Network_Vlan_Firewall {
	r.Options.Offset = &offset
	return r
}

// Retrieve The billing item for a Hardware Firewall (Dedicated).
func (r Network_Vlan_Firewall) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The datacenter that the firewall resides in.
func (r Network_Vlan_Firewall) GetDatacenter() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getDatacenter", nil, &r.Options, &resp)
	return
}

// Retrieve The firewall device type.
func (r Network_Vlan_Firewall) GetFirewallType() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getFirewallType", nil, &r.Options, &resp)
	return
}

// Retrieve A name reflecting the hostname and domain of the firewall. This is created from the combined values of the firewall's logical name and vlan number automatically, and thus can not be edited directly.
func (r Network_Vlan_Firewall) GetFullyQualifiedDomainName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getFullyQualifiedDomainName", nil, &r.Options, &resp)
	return
}

// Retrieve The credentials to log in to a firewall device. This is only present for dedicated appliances.
func (r Network_Vlan_Firewall) GetManagementCredentials() (resp datatypes.Software_Component_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getManagementCredentials", nil, &r.Options, &resp)
	return
}

// Retrieve The update requests made for this firewall.
func (r Network_Vlan_Firewall) GetNetworkFirewallUpdateRequests() (resp []datatypes.Network_Firewall_Update_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getNetworkFirewallUpdateRequests", nil, &r.Options, &resp)
	return
}

// Retrieve The VLAN object that a firewall is associated with and protecting.
func (r Network_Vlan_Firewall) GetNetworkVlan() (resp datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getNetworkVlan", nil, &r.Options, &resp)
	return
}

// Retrieve The VLAN objects that a firewall is associated with and protecting.
func (r Network_Vlan_Firewall) GetNetworkVlans() (resp []datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getNetworkVlans", nil, &r.Options, &resp)
	return
}

// getObject returns a SoftLayer_Network_Vlan_Firewall object. You can only get objects for vlans attached to your account that have a network firewall enabled.
func (r Network_Vlan_Firewall) GetObject() (resp datatypes.Network_Vlan_Firewall, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The currently running rule set of this network component firewall.
func (r Network_Vlan_Firewall) GetRules() (resp []datatypes.Network_Vlan_Firewall_Rule, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getRules", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Network_Vlan_Firewall) GetTagReferences() (resp []datatypes.Tag_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "getTagReferences", nil, &r.Options, &resp)
	return
}

// This will completely reset the firewall to factory settings. If the firewall is not a dedicated appliance an error will occur. Note, this process is performed asynchronously. During the process all traffic will not be routed through the firewall.
func (r Network_Vlan_Firewall) RestoreDefaults() (resp datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "restoreDefaults", nil, &r.Options, &resp)
	return
}

// This method will associate a comma separated list of tags with this object.
func (r Network_Vlan_Firewall) SetTags(tags *string) (resp bool, err error) {
	params := []interface{}{
		tags,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "setTags", params, &r.Options, &resp)
	return
}

// Enable or disable route bypass for this context. If enabled, this will bypass the firewall entirely and all traffic will be routed directly to the host(s) behind it. If disabled, traffic will flow through the firewall normally. This feature is only available for Hardware Firewall (Dedicated) and dedicated appliances.
func (r Network_Vlan_Firewall) UpdateRouteBypass(bypass *bool) (resp datatypes.Provisioning_Version1_Transaction, err error) {
	params := []interface{}{
		bypass,
	}
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Firewall", "updateRouteBypass", params, &r.Options, &resp)
	return
}

// no documentation yet
type Network_Vlan_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetNetworkVlanTypeService returns an instance of the Network_Vlan_Type SoftLayer service
func GetNetworkVlanTypeService(sess *session.Session) Network_Vlan_Type {
	return Network_Vlan_Type{Session: sess}
}

func (r Network_Vlan_Type) Id(id int) Network_Vlan_Type {
	r.Options.Id = &id
	return r
}

func (r Network_Vlan_Type) Mask(mask string) Network_Vlan_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Network_Vlan_Type) Filter(filter string) Network_Vlan_Type {
	r.Options.Filter = filter
	return r
}

func (r Network_Vlan_Type) Limit(limit int) Network_Vlan_Type {
	r.Options.Limit = &limit
	return r
}

func (r Network_Vlan_Type) Offset(offset int) Network_Vlan_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Network_Vlan_Type) GetObject() (resp datatypes.Network_Vlan_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Network_Vlan_Type", "getObject", nil, &r.Options, &resp)
	return
}
