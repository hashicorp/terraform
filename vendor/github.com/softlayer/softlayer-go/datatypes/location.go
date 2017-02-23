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

package datatypes

// Every piece of hardware and network connection owned by SoftLayer is tracked physically by location and stored in the SoftLayer_Location data type. SoftLayer locations exist in parent/child relationships, a convenient way to track equipment from it's city, datacenter, server room, rack, then slot. Network backbones are tied to datacenters only, not to a room, rack, or slot.
type Location struct {
	Entity

	// A count of
	BackboneDependentCount *uint `json:"backboneDependentCount,omitempty" xmlrpc:"backboneDependentCount,omitempty"`

	// no documentation yet
	BackboneDependents []Network_Backbone_Location_Dependent `json:"backboneDependents,omitempty" xmlrpc:"backboneDependents,omitempty"`

	// A count of a location can be a member of 1 or more groups. This will show which groups to which a location belongs.
	GroupCount *uint `json:"groupCount,omitempty" xmlrpc:"groupCount,omitempty"`

	// A location can be a member of 1 or more groups. This will show which groups to which a location belongs.
	Groups []Location_Group `json:"groups,omitempty" xmlrpc:"groups,omitempty"`

	// A count of
	HardwareFirewallCount *uint `json:"hardwareFirewallCount,omitempty" xmlrpc:"hardwareFirewallCount,omitempty"`

	// no documentation yet
	HardwareFirewalls []Hardware `json:"hardwareFirewalls,omitempty" xmlrpc:"hardwareFirewalls,omitempty"`

	// The unique identifier of a specific location.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A location's physical address.
	LocationAddress *Account_Address `json:"locationAddress,omitempty" xmlrpc:"locationAddress,omitempty"`

	// A location's Dedicated Rack member
	LocationReservationMember *Location_Reservation_Rack_Member `json:"locationReservationMember,omitempty" xmlrpc:"locationReservationMember,omitempty"`

	// The current locations status.
	LocationStatus *Location_Status `json:"locationStatus,omitempty" xmlrpc:"locationStatus,omitempty"`

	// A longer location description.
	LongName *string `json:"longName,omitempty" xmlrpc:"longName,omitempty"`

	// A short location description.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	NetworkConfigurationAttribute *Hardware_Attribute `json:"networkConfigurationAttribute,omitempty" xmlrpc:"networkConfigurationAttribute,omitempty"`

	// The total number of users online using SoftLayer's PPTP VPN service for a location.
	OnlinePptpVpnUserCount *int `json:"onlinePptpVpnUserCount,omitempty" xmlrpc:"onlinePptpVpnUserCount,omitempty"`

	// The total number of users online using SoftLayer's SSL VPN service for a location.
	OnlineSslVpnUserCount *int `json:"onlineSslVpnUserCount,omitempty" xmlrpc:"onlineSslVpnUserCount,omitempty"`

	// no documentation yet
	PathString *string `json:"pathString,omitempty" xmlrpc:"pathString,omitempty"`

	// A count of a location can be a member of 1 or more Price Groups. This will show which groups to which a location belongs.
	PriceGroupCount *uint `json:"priceGroupCount,omitempty" xmlrpc:"priceGroupCount,omitempty"`

	// A location can be a member of 1 or more Price Groups. This will show which groups to which a location belongs.
	PriceGroups []Location_Group `json:"priceGroups,omitempty" xmlrpc:"priceGroups,omitempty"`

	// A count of a location can be a member of 1 or more regions. This will show which regions to which a location belongs.
	RegionCount *uint `json:"regionCount,omitempty" xmlrpc:"regionCount,omitempty"`

	// A location can be a member of 1 or more regions. This will show which regions to which a location belongs.
	Regions []Location_Region `json:"regions,omitempty" xmlrpc:"regions,omitempty"`

	// no documentation yet
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// no documentation yet
	Timezone *Locale_Timezone `json:"timezone,omitempty" xmlrpc:"timezone,omitempty"`

	// A location can be a member of 1 Bandwidth Pooling Group. This will show which group to which a location belongs.
	VdrGroup *Location_Group_Location_CrossReference `json:"vdrGroup,omitempty" xmlrpc:"vdrGroup,omitempty"`
}

// SoftLayer_Location_Datacenter extends the [[SoftLayer_Location]] data type to include datacenter-specific properties.
type Location_Datacenter struct {
	Location

	// A count of
	ActiveItemPresaleEventCount *uint `json:"activeItemPresaleEventCount,omitempty" xmlrpc:"activeItemPresaleEventCount,omitempty"`

	// no documentation yet
	ActiveItemPresaleEvents []Sales_Presale_Event `json:"activeItemPresaleEvents,omitempty" xmlrpc:"activeItemPresaleEvents,omitempty"`

	// A count of
	ActivePresaleEventCount *uint `json:"activePresaleEventCount,omitempty" xmlrpc:"activePresaleEventCount,omitempty"`

	// no documentation yet
	ActivePresaleEvents []Sales_Presale_Event `json:"activePresaleEvents,omitempty" xmlrpc:"activePresaleEvents,omitempty"`

	// A count of
	BackendHardwareRouterCount *uint `json:"backendHardwareRouterCount,omitempty" xmlrpc:"backendHardwareRouterCount,omitempty"`

	// no documentation yet
	BackendHardwareRouters []Hardware `json:"backendHardwareRouters,omitempty" xmlrpc:"backendHardwareRouters,omitempty"`

	// A count of subnets which are directly bound to one or more routers in a given datacenter, and currently allow routing.
	BoundSubnetCount *uint `json:"boundSubnetCount,omitempty" xmlrpc:"boundSubnetCount,omitempty"`

	// Subnets which are directly bound to one or more routers in a given datacenter, and currently allow routing.
	BoundSubnets []Network_Subnet `json:"boundSubnets,omitempty" xmlrpc:"boundSubnets,omitempty"`

	// A count of this references relationship between brands, locations and countries associated with a user's account that are ineligible when ordering products. For example, the India datacenter may not be available on this brand for customers that live in Great Britain.
	BrandCountryRestrictionCount *uint `json:"brandCountryRestrictionCount,omitempty" xmlrpc:"brandCountryRestrictionCount,omitempty"`

	// This references relationship between brands, locations and countries associated with a user's account that are ineligible when ordering products. For example, the India datacenter may not be available on this brand for customers that live in Great Britain.
	BrandCountryRestrictions []Brand_Restriction_Location_CustomerCountry `json:"brandCountryRestrictions,omitempty" xmlrpc:"brandCountryRestrictions,omitempty"`

	// A count of
	FrontendHardwareRouterCount *uint `json:"frontendHardwareRouterCount,omitempty" xmlrpc:"frontendHardwareRouterCount,omitempty"`

	// no documentation yet
	FrontendHardwareRouters []Hardware `json:"frontendHardwareRouters,omitempty" xmlrpc:"frontendHardwareRouters,omitempty"`

	// A count of
	HardwareRouterCount *uint `json:"hardwareRouterCount,omitempty" xmlrpc:"hardwareRouterCount,omitempty"`

	// no documentation yet
	HardwareRouters []Hardware `json:"hardwareRouters,omitempty" xmlrpc:"hardwareRouters,omitempty"`

	// A count of
	PresaleEventCount *uint `json:"presaleEventCount,omitempty" xmlrpc:"presaleEventCount,omitempty"`

	// no documentation yet
	PresaleEvents []Sales_Presale_Event `json:"presaleEvents,omitempty" xmlrpc:"presaleEvents,omitempty"`

	// The regional group this datacenter belongs to.
	RegionalGroup *Location_Group_Regional `json:"regionalGroup,omitempty" xmlrpc:"regionalGroup,omitempty"`

	// no documentation yet
	RegionalInternetRegistry *Network_Regional_Internet_Registry `json:"regionalInternetRegistry,omitempty" xmlrpc:"regionalInternetRegistry,omitempty"`

	// A count of retrieve all subnets that are eligible to be routed; those which the account has permission to associate with a vlan.
	RoutableBoundSubnetCount *uint `json:"routableBoundSubnetCount,omitempty" xmlrpc:"routableBoundSubnetCount,omitempty"`

	// Retrieve all subnets that are eligible to be routed; those which the account has permission to associate with a vlan.
	RoutableBoundSubnets []Network_Subnet `json:"routableBoundSubnets,omitempty" xmlrpc:"routableBoundSubnets,omitempty"`
}

// no documentation yet
type Location_Group struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of the locations in a group.
	LocationCount *uint `json:"locationCount,omitempty" xmlrpc:"locationCount,omitempty"`

	// The type for this location group.
	LocationGroupType *Location_Group_Type `json:"locationGroupType,omitempty" xmlrpc:"locationGroupType,omitempty"`

	// no documentation yet
	LocationGroupTypeId *int `json:"locationGroupTypeId,omitempty" xmlrpc:"locationGroupTypeId,omitempty"`

	// The locations in a group.
	Locations []Location `json:"locations,omitempty" xmlrpc:"locations,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	SecurityLevelId *int `json:"securityLevelId,omitempty" xmlrpc:"securityLevelId,omitempty"`
}

// no documentation yet
type Location_Group_Location_CrossReference struct {
	Entity

	// no documentation yet
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	LocationGroup *Location_Group `json:"locationGroup,omitempty" xmlrpc:"locationGroup,omitempty"`

	// no documentation yet
	LocationGroupId *int `json:"locationGroupId,omitempty" xmlrpc:"locationGroupId,omitempty"`

	// no documentation yet
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// If set, this is the priority of this cross reference record in the group.
	Priority *int `json:"priority,omitempty" xmlrpc:"priority,omitempty"`
}

// no documentation yet
type Location_Group_Pricing struct {
	Location_Group

	// A count of the prices that this pricing location group limits. All of these prices will only be available in the locations defined by this pricing location group.
	PriceCount *uint `json:"priceCount,omitempty" xmlrpc:"priceCount,omitempty"`

	// The prices that this pricing location group limits. All of these prices will only be available in the locations defined by this pricing location group.
	Prices []Product_Item_Price `json:"prices,omitempty" xmlrpc:"prices,omitempty"`
}

// no documentation yet
type Location_Group_Regional struct {
	Location_Group

	// A count of the datacenters in a group.
	DatacenterCount *uint `json:"datacenterCount,omitempty" xmlrpc:"datacenterCount,omitempty"`

	// The datacenters in a group.
	Datacenters []Location `json:"datacenters,omitempty" xmlrpc:"datacenters,omitempty"`

	// The preferred datacenters of a group.
	PreferredDatacenter *Location_Datacenter `json:"preferredDatacenter,omitempty" xmlrpc:"preferredDatacenter,omitempty"`
}

// no documentation yet
type Location_Group_Type struct {
	Entity

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// SoftLayer_Location_Inventory_Room extends the [[SoftLayer_Location]] data type to include inventory room-specific properties.
type Location_Inventory_Room struct {
	Location
}

// SoftLayer_Location_Network_Operations_Center extends the [[SoftLayer_Location]] data type to include network operation center-specific properties.
type Location_Network_Operations_Center struct {
	Location
}

// SoftLayer_Location_Office extends the [[SoftLayer_Location]] data type to include office-specific properties.
type Location_Office struct {
	Location
}

// SoftLayer_Location_Rack extends the [[SoftLayer_Location]] data type to include rack-specific properties.
type Location_Rack struct {
	Location
}

// A region is made up of a keyname and a description of that region. A region keyname can be used as part of an order. Check the SoftLayer_Product_Order service for more details.
type Location_Region struct {
	Entity

	// A short description of a region's name. This description is seen on the order forms.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A unique key name for a region. Provided for easy debugging. This is to be sent in with an order.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// Each region can have many datacenter locations tied to it. However, this is the location we currently provision to for a region. This location is the current valid location for a region.
	Location *Location_Region_Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// An integer representing the order in which this element is displayed.
	SortOrder *int `json:"sortOrder,omitempty" xmlrpc:"sortOrder,omitempty"`
}

// The SoftLayer_Location_Region_Location is very specific to the location where services will actually be provisioned. When accessed through a package, this location is the top priority location for a region. All new servers and services are provisioned at this location. When a server is ordered and a region is selected, this is the location within that region where the server will actually exist and have software/services installed.
type Location_Region_Location struct {
	Entity

	// The SoftLayer_Location tied to a region's location. This provides more information about the location, including specific datacenter information.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// A count of a region's location also has delivery information as well as other information to be determined. For now, availability is provided and could weigh into the decision as to where to decide to have a server provisioned.'
	LocationPackageDetailCount *uint `json:"locationPackageDetailCount,omitempty" xmlrpc:"locationPackageDetailCount,omitempty"`

	// A region's location also has delivery information as well as other information to be determined. For now, availability is provided and could weigh into the decision as to where to decide to have a server provisioned.'
	LocationPackageDetails []Product_Package_Locations `json:"locationPackageDetails,omitempty" xmlrpc:"locationPackageDetails,omitempty"`

	// The region to which this location belongs.
	Region *Location_Region `json:"region,omitempty" xmlrpc:"region,omitempty"`
}

// no documentation yet
type Location_Reservation struct {
	Entity

	// The account that a billing item belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The bandwidth allotment that the reservation belongs to.
	Allotment *Network_Bandwidth_Version1_Allotment `json:"allotment,omitempty" xmlrpc:"allotment,omitempty"`

	// no documentation yet
	AllotmentId *int `json:"allotmentId,omitempty" xmlrpc:"allotmentId,omitempty"`

	// The bandwidth allotment that the reservation belongs to.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The datacenter location that the reservation belongs to.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// Rack information for the reservation
	LocationReservationRack *Location_Reservation_Rack `json:"locationReservationRack,omitempty" xmlrpc:"locationReservationRack,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`
}

// no documentation yet
type Location_Reservation_Rack struct {
	Entity

	// The bandwidth allotment that the reservation belongs to.
	Allotment *Network_Bandwidth_Version1_Allotment `json:"allotment,omitempty" xmlrpc:"allotment,omitempty"`

	// Members of the rack.
	Children []Location_Reservation_Rack_Member `json:"children,omitempty" xmlrpc:"children,omitempty"`

	// A count of members of the rack.
	ChildrenCount *uint `json:"childrenCount,omitempty" xmlrpc:"childrenCount,omitempty"`

	// no documentation yet
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// no documentation yet
	LocationReservation *Location_Reservation `json:"locationReservation,omitempty" xmlrpc:"locationReservation,omitempty"`

	// no documentation yet
	LocationReservationId *int `json:"locationReservationId,omitempty" xmlrpc:"locationReservationId,omitempty"`

	// no documentation yet
	NetworkConnectionCapacity *int `json:"networkConnectionCapacity,omitempty" xmlrpc:"networkConnectionCapacity,omitempty"`

	// no documentation yet
	NetworkConnectionReservation *int `json:"networkConnectionReservation,omitempty" xmlrpc:"networkConnectionReservation,omitempty"`

	// no documentation yet
	PowerConnectionCapacity *int `json:"powerConnectionCapacity,omitempty" xmlrpc:"powerConnectionCapacity,omitempty"`

	// no documentation yet
	PowerConnectionReservation *int `json:"powerConnectionReservation,omitempty" xmlrpc:"powerConnectionReservation,omitempty"`

	// no documentation yet
	SlotCapacity *int `json:"slotCapacity,omitempty" xmlrpc:"slotCapacity,omitempty"`

	// no documentation yet
	SlotReservation *int `json:"slotReservation,omitempty" xmlrpc:"slotReservation,omitempty"`
}

// no documentation yet
type Location_Reservation_Rack_Member struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Location relation for the rack member
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// no documentation yet
	LocationReservationRack *Location_Reservation `json:"locationReservationRack,omitempty" xmlrpc:"locationReservationRack,omitempty"`
}

// SoftLayer_Location_Root extends the [[SoftLayer_Location]] data type to include root-specific properties.
type Location_Root struct {
	Location
}

// SoftLayer_Location_Server_Room extends the [[SoftLayer_Location]] data type to include server room-specific properties.
type Location_Server_Room struct {
	Location
}

// SoftLayer_Location_Slot extends the [[SoftLayer_Location]] data type to include slot-specific properties.
type Location_Slot struct {
	Location
}

// SoftLayer_Location_Status models the state of any location. SoftLayer uses the following status codes:
//
//
// *'''ACTIVE''': The location is currently active and available for public usage.
// *'''PLANNED''': Used when a location is planned but not yet active.
// *'''RETIRED''': Used when a location has been retired and no longer active.
//
//
// Locations in use should stay in the ACTIVE state. If a locations status ever reads anything else and contains active hardware then please contact SoftLayer support.
type Location_Status struct {
	Entity

	// A locations status's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A Location's status code. See the SoftLayer_Locaiton_Status Overview for ''status''' possible values.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// SoftLayer_Location_Storage_Room extends the [[SoftLayer_Location]] data type to include storage room-specific properties.
type Location_Storage_Room struct {
	Location
}
