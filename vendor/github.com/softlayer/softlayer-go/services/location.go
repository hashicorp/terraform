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

// Every piece of hardware and network connection owned by SoftLayer is tracked physically by location and stored in the SoftLayer_Location data type. SoftLayer locations exist in parent/child relationships, a convenient way to track equipment from it's city, datacenter, server room, rack, then slot. Network backbones are tied to datacenters only, not to a room, rack, or slot.
type Location struct {
	Session *session.Session
	Options sl.Options
}

// GetLocationService returns an instance of the Location SoftLayer service
func GetLocationService(sess *session.Session) Location {
	return Location{Session: sess}
}

func (r Location) Id(id int) Location {
	r.Options.Id = &id
	return r
}

func (r Location) Mask(mask string) Location {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Location) Filter(filter string) Location {
	r.Options.Filter = filter
	return r
}

func (r Location) Limit(limit int) Location {
	r.Options.Limit = &limit
	return r
}

func (r Location) Offset(offset int) Location {
	r.Options.Offset = &offset
	return r
}

// Object Storage is only available in select datacenters. This method will return all the datacenters where object storage is available.
func (r Location) GetAvailableObjectStorageDatacenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getAvailableObjectStorageDatacenters", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location) GetBackboneDependents() (resp []datatypes.Network_Backbone_Location_Dependent, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getBackboneDependents", nil, &r.Options, &resp)
	return
}

// Retrieve all datacenter locations. SoftLayer's datacenters exist in various cities and each contain one or more server rooms which house network and server infrastructure.
func (r Location) GetDatacenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getDatacenters", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location) GetDatacentersWithVirtualImageStoreServiceResourceRecord() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getDatacentersWithVirtualImageStoreServiceResourceRecord", nil, &r.Options, &resp)
	return
}

// Retrieve A location can be a member of 1 or more groups. This will show which groups to which a location belongs.
func (r Location) GetGroups() (resp []datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getGroups", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location) GetHardwareFirewalls() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getHardwareFirewalls", nil, &r.Options, &resp)
	return
}

// Retrieve A location's physical address.
func (r Location) GetLocationAddress() (resp datatypes.Account_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getLocationAddress", nil, &r.Options, &resp)
	return
}

// Retrieve A location's Dedicated Rack member
func (r Location) GetLocationReservationMember() (resp datatypes.Location_Reservation_Rack_Member, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getLocationReservationMember", nil, &r.Options, &resp)
	return
}

// Retrieve The current locations status.
func (r Location) GetLocationStatus() (resp datatypes.Location_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getLocationStatus", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location) GetNetworkConfigurationAttribute() (resp datatypes.Hardware_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getNetworkConfigurationAttribute", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location) GetObject() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The total number of users online using SoftLayer's PPTP VPN service for a location.
func (r Location) GetOnlinePptpVpnUserCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getOnlinePptpVpnUserCount", nil, &r.Options, &resp)
	return
}

// Retrieve The total number of users online using SoftLayer's SSL VPN service for a location.
func (r Location) GetOnlineSslVpnUserCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getOnlineSslVpnUserCount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location) GetPathString() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getPathString", nil, &r.Options, &resp)
	return
}

// Retrieve A location can be a member of 1 or more Price Groups. This will show which groups to which a location belongs.
func (r Location) GetPriceGroups() (resp []datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getPriceGroups", nil, &r.Options, &resp)
	return
}

// Retrieve A location can be a member of 1 or more regions. This will show which regions to which a location belongs.
func (r Location) GetRegions() (resp []datatypes.Location_Region, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getRegions", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location) GetTimezone() (resp datatypes.Locale_Timezone, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getTimezone", nil, &r.Options, &resp)
	return
}

// Retrieve A location can be a member of 1 Bandwidth Pooling Group. This will show which group to which a location belongs.
func (r Location) GetVdrGroup() (resp datatypes.Location_Group_Location_CrossReference, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getVdrGroup", nil, &r.Options, &resp)
	return
}

// Retrieve all datacenter locations. SoftLayer's datacenters exist in various cities and each contain one or more server rooms which house network and server infrastructure.
func (r Location) GetViewableDatacenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getViewableDatacenters", nil, &r.Options, &resp)
	return
}

// Retrieve all viewable pop and datacenter locations.
func (r Location) GetViewablePopsAndDataCenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getViewablePopsAndDataCenters", nil, &r.Options, &resp)
	return
}

// Retrieve all viewable network locations.
func (r Location) GetViewablepointOfPresence() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getViewablepointOfPresence", nil, &r.Options, &resp)
	return
}

// Retrieve all point of presence locations.
func (r Location) GetpointOfPresence() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location", "getpointOfPresence", nil, &r.Options, &resp)
	return
}

// SoftLayer_Location_Datacenter extends the [[SoftLayer_Location]] data type to include datacenter-specific properties.
type Location_Datacenter struct {
	Session *session.Session
	Options sl.Options
}

// GetLocationDatacenterService returns an instance of the Location_Datacenter SoftLayer service
func GetLocationDatacenterService(sess *session.Session) Location_Datacenter {
	return Location_Datacenter{Session: sess}
}

func (r Location_Datacenter) Id(id int) Location_Datacenter {
	r.Options.Id = &id
	return r
}

func (r Location_Datacenter) Mask(mask string) Location_Datacenter {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Location_Datacenter) Filter(filter string) Location_Datacenter {
	r.Options.Filter = filter
	return r
}

func (r Location_Datacenter) Limit(limit int) Location_Datacenter {
	r.Options.Limit = &limit
	return r
}

func (r Location_Datacenter) Offset(offset int) Location_Datacenter {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Location_Datacenter) GetActiveItemPresaleEvents() (resp []datatypes.Sales_Presale_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getActiveItemPresaleEvents", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetActivePresaleEvents() (resp []datatypes.Sales_Presale_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getActivePresaleEvents", nil, &r.Options, &resp)
	return
}

// Object Storage is only available in select datacenters. This method will return all the datacenters where object storage is available.
func (r Location_Datacenter) GetAvailableObjectStorageDatacenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getAvailableObjectStorageDatacenters", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetBackboneDependents() (resp []datatypes.Network_Backbone_Location_Dependent, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getBackboneDependents", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetBackendHardwareRouters() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getBackendHardwareRouters", nil, &r.Options, &resp)
	return
}

// Retrieve Subnets which are directly bound to one or more routers in a given datacenter, and currently allow routing.
func (r Location_Datacenter) GetBoundSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getBoundSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve This references relationship between brands, locations and countries associated with a user's account that are ineligible when ordering products. For example, the India datacenter may not be available on this brand for customers that live in Great Britain.
func (r Location_Datacenter) GetBrandCountryRestrictions() (resp []datatypes.Brand_Restriction_Location_CustomerCountry, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getBrandCountryRestrictions", nil, &r.Options, &resp)
	return
}

// Retrieve all datacenter locations. SoftLayer's datacenters exist in various cities and each contain one or more server rooms which house network and server infrastructure.
func (r Location_Datacenter) GetDatacenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getDatacenters", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Datacenter) GetDatacentersWithVirtualImageStoreServiceResourceRecord() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getDatacentersWithVirtualImageStoreServiceResourceRecord", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetFrontendHardwareRouters() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getFrontendHardwareRouters", nil, &r.Options, &resp)
	return
}

// Retrieve A location can be a member of 1 or more groups. This will show which groups to which a location belongs.
func (r Location_Datacenter) GetGroups() (resp []datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getGroups", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetHardwareFirewalls() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getHardwareFirewalls", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetHardwareRouters() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getHardwareRouters", nil, &r.Options, &resp)
	return
}

// Retrieve A location's physical address.
func (r Location_Datacenter) GetLocationAddress() (resp datatypes.Account_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getLocationAddress", nil, &r.Options, &resp)
	return
}

// Retrieve A location's Dedicated Rack member
func (r Location_Datacenter) GetLocationReservationMember() (resp datatypes.Location_Reservation_Rack_Member, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getLocationReservationMember", nil, &r.Options, &resp)
	return
}

// Retrieve The current locations status.
func (r Location_Datacenter) GetLocationStatus() (resp datatypes.Location_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getLocationStatus", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetNetworkConfigurationAttribute() (resp datatypes.Hardware_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getNetworkConfigurationAttribute", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Datacenter) GetObject() (resp datatypes.Location_Datacenter, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The total number of users online using SoftLayer's PPTP VPN service for a location.
func (r Location_Datacenter) GetOnlinePptpVpnUserCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getOnlinePptpVpnUserCount", nil, &r.Options, &resp)
	return
}

// Retrieve The total number of users online using SoftLayer's SSL VPN service for a location.
func (r Location_Datacenter) GetOnlineSslVpnUserCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getOnlineSslVpnUserCount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetPathString() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getPathString", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetPresaleEvents() (resp []datatypes.Sales_Presale_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getPresaleEvents", nil, &r.Options, &resp)
	return
}

// Retrieve A location can be a member of 1 or more Price Groups. This will show which groups to which a location belongs.
func (r Location_Datacenter) GetPriceGroups() (resp []datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getPriceGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The regional group this datacenter belongs to.
func (r Location_Datacenter) GetRegionalGroup() (resp datatypes.Location_Group_Regional, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getRegionalGroup", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetRegionalInternetRegistry() (resp datatypes.Network_Regional_Internet_Registry, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getRegionalInternetRegistry", nil, &r.Options, &resp)
	return
}

// Retrieve A location can be a member of 1 or more regions. This will show which regions to which a location belongs.
func (r Location_Datacenter) GetRegions() (resp []datatypes.Location_Region, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getRegions", nil, &r.Options, &resp)
	return
}

// Retrieve Retrieve all subnets that are eligible to be routed; those which the account has permission to associate with a vlan.
func (r Location_Datacenter) GetRoutableBoundSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getRoutableBoundSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve a graph of a SoftLayer datacenter's last 48 hours of network activity. Statistics graphs show traffic outbound from a datacenter on top and inbound traffic on the bottom followed by a legend of the network services tracked in the graph. getStatisticsGraphImage returns a PNG image of variable width and height depending on the number of services reported in the image.
func (r Location_Datacenter) GetStatisticsGraphImage() (resp []byte, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getStatisticsGraphImage", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Datacenter) GetTimezone() (resp datatypes.Locale_Timezone, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getTimezone", nil, &r.Options, &resp)
	return
}

// Retrieve A location can be a member of 1 Bandwidth Pooling Group. This will show which group to which a location belongs.
func (r Location_Datacenter) GetVdrGroup() (resp datatypes.Location_Group_Location_CrossReference, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getVdrGroup", nil, &r.Options, &resp)
	return
}

// Retrieve all datacenter locations. SoftLayer's datacenters exist in various cities and each contain one or more server rooms which house network and server infrastructure.
func (r Location_Datacenter) GetViewableDatacenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getViewableDatacenters", nil, &r.Options, &resp)
	return
}

// Retrieve all viewable pop and datacenter locations.
func (r Location_Datacenter) GetViewablePopsAndDataCenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getViewablePopsAndDataCenters", nil, &r.Options, &resp)
	return
}

// Retrieve all viewable network locations.
func (r Location_Datacenter) GetViewablepointOfPresence() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getViewablepointOfPresence", nil, &r.Options, &resp)
	return
}

// Retrieve all point of presence locations.
func (r Location_Datacenter) GetpointOfPresence() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Datacenter", "getpointOfPresence", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Location_Group struct {
	Session *session.Session
	Options sl.Options
}

// GetLocationGroupService returns an instance of the Location_Group SoftLayer service
func GetLocationGroupService(sess *session.Session) Location_Group {
	return Location_Group{Session: sess}
}

func (r Location_Group) Id(id int) Location_Group {
	r.Options.Id = &id
	return r
}

func (r Location_Group) Mask(mask string) Location_Group {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Location_Group) Filter(filter string) Location_Group {
	r.Options.Filter = filter
	return r
}

func (r Location_Group) Limit(limit int) Location_Group {
	r.Options.Limit = &limit
	return r
}

func (r Location_Group) Offset(offset int) Location_Group {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Location_Group) GetAllObjects() (resp []datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve The type for this location group.
func (r Location_Group) GetLocationGroupType() (resp datatypes.Location_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group", "getLocationGroupType", nil, &r.Options, &resp)
	return
}

// Retrieve The locations in a group.
func (r Location_Group) GetLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group", "getLocations", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Group) GetObject() (resp datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Location_Group_Pricing struct {
	Session *session.Session
	Options sl.Options
}

// GetLocationGroupPricingService returns an instance of the Location_Group_Pricing SoftLayer service
func GetLocationGroupPricingService(sess *session.Session) Location_Group_Pricing {
	return Location_Group_Pricing{Session: sess}
}

func (r Location_Group_Pricing) Id(id int) Location_Group_Pricing {
	r.Options.Id = &id
	return r
}

func (r Location_Group_Pricing) Mask(mask string) Location_Group_Pricing {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Location_Group_Pricing) Filter(filter string) Location_Group_Pricing {
	r.Options.Filter = filter
	return r
}

func (r Location_Group_Pricing) Limit(limit int) Location_Group_Pricing {
	r.Options.Limit = &limit
	return r
}

func (r Location_Group_Pricing) Offset(offset int) Location_Group_Pricing {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Location_Group_Pricing) GetAllObjects() (resp []datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Pricing", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve The type for this location group.
func (r Location_Group_Pricing) GetLocationGroupType() (resp datatypes.Location_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Pricing", "getLocationGroupType", nil, &r.Options, &resp)
	return
}

// Retrieve The locations in a group.
func (r Location_Group_Pricing) GetLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Pricing", "getLocations", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Group_Pricing) GetObject() (resp datatypes.Location_Group_Pricing, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Pricing", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The prices that this pricing location group limits. All of these prices will only be available in the locations defined by this pricing location group.
func (r Location_Group_Pricing) GetPrices() (resp []datatypes.Product_Item_Price, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Pricing", "getPrices", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Location_Group_Regional struct {
	Session *session.Session
	Options sl.Options
}

// GetLocationGroupRegionalService returns an instance of the Location_Group_Regional SoftLayer service
func GetLocationGroupRegionalService(sess *session.Session) Location_Group_Regional {
	return Location_Group_Regional{Session: sess}
}

func (r Location_Group_Regional) Id(id int) Location_Group_Regional {
	r.Options.Id = &id
	return r
}

func (r Location_Group_Regional) Mask(mask string) Location_Group_Regional {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Location_Group_Regional) Filter(filter string) Location_Group_Regional {
	r.Options.Filter = filter
	return r
}

func (r Location_Group_Regional) Limit(limit int) Location_Group_Regional {
	r.Options.Limit = &limit
	return r
}

func (r Location_Group_Regional) Offset(offset int) Location_Group_Regional {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Location_Group_Regional) GetAllObjects() (resp []datatypes.Location_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Regional", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve The datacenters in a group.
func (r Location_Group_Regional) GetDatacenters() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Regional", "getDatacenters", nil, &r.Options, &resp)
	return
}

// Retrieve The type for this location group.
func (r Location_Group_Regional) GetLocationGroupType() (resp datatypes.Location_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Regional", "getLocationGroupType", nil, &r.Options, &resp)
	return
}

// Retrieve The locations in a group.
func (r Location_Group_Regional) GetLocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Regional", "getLocations", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Group_Regional) GetObject() (resp datatypes.Location_Group_Regional, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Regional", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The preferred datacenters of a group.
func (r Location_Group_Regional) GetPreferredDatacenter() (resp datatypes.Location_Datacenter, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Group_Regional", "getPreferredDatacenter", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Location_Reservation struct {
	Session *session.Session
	Options sl.Options
}

// GetLocationReservationService returns an instance of the Location_Reservation SoftLayer service
func GetLocationReservationService(sess *session.Session) Location_Reservation {
	return Location_Reservation{Session: sess}
}

func (r Location_Reservation) Id(id int) Location_Reservation {
	r.Options.Id = &id
	return r
}

func (r Location_Reservation) Mask(mask string) Location_Reservation {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Location_Reservation) Filter(filter string) Location_Reservation {
	r.Options.Filter = filter
	return r
}

func (r Location_Reservation) Limit(limit int) Location_Reservation {
	r.Options.Limit = &limit
	return r
}

func (r Location_Reservation) Offset(offset int) Location_Reservation {
	r.Options.Offset = &offset
	return r
}

// Retrieve The account that a billing item belongs to.
func (r Location_Reservation) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation", "getAccount", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Reservation) GetAccountReservations() (resp []datatypes.Location_Reservation, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation", "getAccountReservations", nil, &r.Options, &resp)
	return
}

// Retrieve The bandwidth allotment that the reservation belongs to.
func (r Location_Reservation) GetAllotment() (resp datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation", "getAllotment", nil, &r.Options, &resp)
	return
}

// Retrieve The bandwidth allotment that the reservation belongs to.
func (r Location_Reservation) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The datacenter location that the reservation belongs to.
func (r Location_Reservation) GetLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation", "getLocation", nil, &r.Options, &resp)
	return
}

// Retrieve Rack information for the reservation
func (r Location_Reservation) GetLocationReservationRack() (resp datatypes.Location_Reservation_Rack, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation", "getLocationReservationRack", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Reservation) GetObject() (resp datatypes.Location_Reservation, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Location_Reservation_Rack struct {
	Session *session.Session
	Options sl.Options
}

// GetLocationReservationRackService returns an instance of the Location_Reservation_Rack SoftLayer service
func GetLocationReservationRackService(sess *session.Session) Location_Reservation_Rack {
	return Location_Reservation_Rack{Session: sess}
}

func (r Location_Reservation_Rack) Id(id int) Location_Reservation_Rack {
	r.Options.Id = &id
	return r
}

func (r Location_Reservation_Rack) Mask(mask string) Location_Reservation_Rack {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Location_Reservation_Rack) Filter(filter string) Location_Reservation_Rack {
	r.Options.Filter = filter
	return r
}

func (r Location_Reservation_Rack) Limit(limit int) Location_Reservation_Rack {
	r.Options.Limit = &limit
	return r
}

func (r Location_Reservation_Rack) Offset(offset int) Location_Reservation_Rack {
	r.Options.Offset = &offset
	return r
}

// Retrieve The bandwidth allotment that the reservation belongs to.
func (r Location_Reservation_Rack) GetAllotment() (resp datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation_Rack", "getAllotment", nil, &r.Options, &resp)
	return
}

// Retrieve Members of the rack.
func (r Location_Reservation_Rack) GetChildren() (resp []datatypes.Location_Reservation_Rack_Member, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation_Rack", "getChildren", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Reservation_Rack) GetLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation_Rack", "getLocation", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Reservation_Rack) GetLocationReservation() (resp datatypes.Location_Reservation, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation_Rack", "getLocationReservation", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Reservation_Rack) GetObject() (resp datatypes.Location_Reservation_Rack, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation_Rack", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Location_Reservation_Rack_Member struct {
	Session *session.Session
	Options sl.Options
}

// GetLocationReservationRackMemberService returns an instance of the Location_Reservation_Rack_Member SoftLayer service
func GetLocationReservationRackMemberService(sess *session.Session) Location_Reservation_Rack_Member {
	return Location_Reservation_Rack_Member{Session: sess}
}

func (r Location_Reservation_Rack_Member) Id(id int) Location_Reservation_Rack_Member {
	r.Options.Id = &id
	return r
}

func (r Location_Reservation_Rack_Member) Mask(mask string) Location_Reservation_Rack_Member {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Location_Reservation_Rack_Member) Filter(filter string) Location_Reservation_Rack_Member {
	r.Options.Filter = filter
	return r
}

func (r Location_Reservation_Rack_Member) Limit(limit int) Location_Reservation_Rack_Member {
	r.Options.Limit = &limit
	return r
}

func (r Location_Reservation_Rack_Member) Offset(offset int) Location_Reservation_Rack_Member {
	r.Options.Offset = &offset
	return r
}

// Retrieve Location relation for the rack member
func (r Location_Reservation_Rack_Member) GetLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation_Rack_Member", "getLocation", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Location_Reservation_Rack_Member) GetLocationReservationRack() (resp datatypes.Location_Reservation, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation_Rack_Member", "getLocationReservationRack", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Location_Reservation_Rack_Member) GetObject() (resp datatypes.Location_Reservation_Rack_Member, err error) {
	err = r.Session.DoRequest("SoftLayer_Location_Reservation_Rack_Member", "getObject", nil, &r.Options, &resp)
	return
}
