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

// Metric tracking objects provides a common interface to all metrics provided by SoftLayer. These metrics range from network component traffic for a server to aggregated Bandwidth Pooling traffic and more. Every object within SoftLayer's range of objects that has data that can be tracked over time has an associated tracking object. Use the [[SoftLayer_Metric_Tracking_Object]] service to retrieve raw and graph data from a tracking object.
type Metric_Tracking_Object struct {
	Entity

	// The data recorded by a tracking object.
	Data []Metric_Tracking_Object_Data `json:"data,omitempty" xmlrpc:"data,omitempty"`

	// A tracking object's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Tracking object label
	Label *string `json:"label,omitempty" xmlrpc:"label,omitempty"`

	// The identifier of the existing resource this object is attempting to track.
	ResourceTableId *int `json:"resourceTableId,omitempty" xmlrpc:"resourceTableId,omitempty"`

	// The date this tracker began tracking this particular resource.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// The type of data that a tracking object polls.
	Type *Metric_Tracking_Object_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// SoftLayer_Metric_Tracking_Object_Abstract models a generic tracking object type. Typically a tracking object with a specific purpose has it's own data type defined within the SoftLayer API.
type Metric_Tracking_Object_Abstract struct {
	Metric_Tracking_Object
}

// This data type provides commonly used bandwidth summary components for the current billing cycle.
type Metric_Tracking_Object_Bandwidth_Summary struct {
	Entity

	// This is the amount of bandwidth (measured in gigabytes) allocated for this tracking object.
	AllocationAmount *Float64 `json:"allocationAmount,omitempty" xmlrpc:"allocationAmount,omitempty"`

	// no documentation yet
	AllocationId *int `json:"allocationId,omitempty" xmlrpc:"allocationId,omitempty"`

	// The amount of outbound bandwidth (measured in gigabytes) currently used this billing period. Same as $outboundBandwidthAmount. Aliased for backward compatability.
	AmountOut *Float64 `json:"amountOut,omitempty" xmlrpc:"amountOut,omitempty"`

	// The daily average amount of outbound bandwidth usage.
	AverageDailyUsage *Float64 `json:"averageDailyUsage,omitempty" xmlrpc:"averageDailyUsage,omitempty"`

	// A flag that tells whether or not this tracking object's bandwidth usage is already over the allocation. 1 means yes, 0 means no.
	CurrentlyOverAllocationFlag *int `json:"currentlyOverAllocationFlag,omitempty" xmlrpc:"currentlyOverAllocationFlag,omitempty"`

	// The metric tracking id for this resource.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The amount of outbound bandwidth (measured in gigabytes) currently used this billing period
	OutboundBandwidthAmount *Float64 `json:"outboundBandwidthAmount,omitempty" xmlrpc:"outboundBandwidthAmount,omitempty"`

	// The amount of bandwidth (measured in gigabytes) of projected usage, using a basic average calculation of daily usage.
	ProjectedBandwidthUsage *Float64 `json:"projectedBandwidthUsage,omitempty" xmlrpc:"projectedBandwidthUsage,omitempty"`

	// A flag that tells whether or not this tracking object's bandwidth usage is projected to go over the allocation, based on daily average usage. 1 means yes, 0 means no.
	ProjectedOverAllocationFlag *int `json:"projectedOverAllocationFlag,omitempty" xmlrpc:"projectedOverAllocationFlag,omitempty"`
}

// SoftLayer_Metric_Tracking_Object_Data models an individual unit of data tracked by a SoftLayer tracking object, including the type of data polled, the date it was polled at, and the counter value that was measured at polling time.
type Metric_Tracking_Object_Data struct {
	Entity

	// The value stored for a data record.
	Counter *Float64 `json:"counter,omitempty" xmlrpc:"counter,omitempty"`

	// The time a data record was stored.
	DateTime *Time `json:"dateTime,omitempty" xmlrpc:"dateTime,omitempty"`

	// The type of data held in a record.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// SoftLayer_Metric_Tracking_Object_Data_Network_ContentDelivery_Account models usage data polled from the CDN system.
type Metric_Tracking_Object_Data_Network_ContentDelivery_Account struct {
	Metric_Tracking_Object_Data

	// The name of a file. This value is only populated in file-based bandwidth reports.
	FileName *string `json:"fileName,omitempty" xmlrpc:"fileName,omitempty"`

	// The internal identifier of a CDN POP (Points of Presence).
	PopId *int `json:"popId,omitempty" xmlrpc:"popId,omitempty"`
}

// SoftLayer_Metric_Tracking_Object_HardwareServer models tracking objects specific to physical hardware and the data that are recorded by those servers.
type Metric_Tracking_Object_HardwareServer struct {
	Metric_Tracking_Object_Abstract

	// The raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCycleBandwidthUsage []Network_Bandwidth_Usage `json:"billingCycleBandwidthUsage,omitempty" xmlrpc:"billingCycleBandwidthUsage,omitempty"`

	// A count of the raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCycleBandwidthUsageCount *uint `json:"billingCycleBandwidthUsageCount,omitempty" xmlrpc:"billingCycleBandwidthUsageCount,omitempty"`

	// The raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCyclePrivateBandwidthUsage []Network_Bandwidth_Usage `json:"billingCyclePrivateBandwidthUsage,omitempty" xmlrpc:"billingCyclePrivateBandwidthUsage,omitempty"`

	// A count of the raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCyclePrivateBandwidthUsageCount *uint `json:"billingCyclePrivateBandwidthUsageCount,omitempty" xmlrpc:"billingCyclePrivateBandwidthUsageCount,omitempty"`

	// The total private inbound bandwidth for this item's resource for the current billing cycle.
	BillingCyclePrivateUsageIn *Float64 `json:"billingCyclePrivateUsageIn,omitempty" xmlrpc:"billingCyclePrivateUsageIn,omitempty"`

	// The total private outbound bandwidth for this item's resource for the current billing cycle.
	BillingCyclePrivateUsageOut *Float64 `json:"billingCyclePrivateUsageOut,omitempty" xmlrpc:"billingCyclePrivateUsageOut,omitempty"`

	// The total private bandwidth for this item's resource for the current billing cycle.
	BillingCyclePrivateUsageTotal *uint `json:"billingCyclePrivateUsageTotal,omitempty" xmlrpc:"billingCyclePrivateUsageTotal,omitempty"`

	// The raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCyclePublicBandwidthUsage *Network_Bandwidth_Usage `json:"billingCyclePublicBandwidthUsage,omitempty" xmlrpc:"billingCyclePublicBandwidthUsage,omitempty"`

	// The total public inbound bandwidth for this item's resource for the current billing cycle.
	BillingCyclePublicUsageIn *Float64 `json:"billingCyclePublicUsageIn,omitempty" xmlrpc:"billingCyclePublicUsageIn,omitempty"`

	// The total public outbound bandwidth for this item's resource for the current billing cycle.
	BillingCyclePublicUsageOut *Float64 `json:"billingCyclePublicUsageOut,omitempty" xmlrpc:"billingCyclePublicUsageOut,omitempty"`

	// The total public bandwidth for this item's resource for the current billing cycle.
	BillingCyclePublicUsageTotal *uint `json:"billingCyclePublicUsageTotal,omitempty" xmlrpc:"billingCyclePublicUsageTotal,omitempty"`

	// The server that this tracking object tracks.
	Resource *Hardware_Server `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// SoftLayer [[SoftLayer_Metric_Tracking_Object|tracking objects]] can model various kinds of measured data, from server and virtual server bandwidth usage to CPU use to remote storage usage. SoftLayer_Metric_Tracking_Object_Type models one of these types and is referred to in tracking objects to reflect what type of data they track.
type Metric_Tracking_Object_Type struct {
	Entity

	// Description A tracking object type's key name. This is a shorter description of what kind of data a tracking object group is polling.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// Description A tracking object type's name. This describes what kind of data a tracking object group is polling.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// SoftLayer_Metric_Tracking_Object_VirtualDedicatedRack models tracking objects specific to virtual dedicated racks. Bandwidth Pooling aggregate the bandwidth used by multiple servers within the rack.
type Metric_Tracking_Object_VirtualDedicatedRack struct {
	Metric_Tracking_Object_Abstract

	// The raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCycleBandwidthUsage []Network_Bandwidth_Usage `json:"billingCycleBandwidthUsage,omitempty" xmlrpc:"billingCycleBandwidthUsage,omitempty"`

	// A count of the raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCycleBandwidthUsageCount *uint `json:"billingCycleBandwidthUsageCount,omitempty" xmlrpc:"billingCycleBandwidthUsageCount,omitempty"`

	// The raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCyclePrivateBandwidthUsage []Network_Bandwidth_Usage `json:"billingCyclePrivateBandwidthUsage,omitempty" xmlrpc:"billingCyclePrivateBandwidthUsage,omitempty"`

	// A count of the raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCyclePrivateBandwidthUsageCount *uint `json:"billingCyclePrivateBandwidthUsageCount,omitempty" xmlrpc:"billingCyclePrivateBandwidthUsageCount,omitempty"`

	// The total private inbound bandwidth for this item's resource for the current billing cycle.
	BillingCyclePrivateUsageIn *Float64 `json:"billingCyclePrivateUsageIn,omitempty" xmlrpc:"billingCyclePrivateUsageIn,omitempty"`

	// The total private outbound bandwidth for this item's resource for the current billing cycle.
	BillingCyclePrivateUsageOut *Float64 `json:"billingCyclePrivateUsageOut,omitempty" xmlrpc:"billingCyclePrivateUsageOut,omitempty"`

	// The total private bandwidth for this item's resource for the current billing cycle.
	BillingCyclePrivateUsageTotal *uint `json:"billingCyclePrivateUsageTotal,omitempty" xmlrpc:"billingCyclePrivateUsageTotal,omitempty"`

	// The raw bandwidth usage data for the current billing cycle. One object is returned for each network this server is attached to.
	BillingCyclePublicBandwidthUsage *Network_Bandwidth_Usage `json:"billingCyclePublicBandwidthUsage,omitempty" xmlrpc:"billingCyclePublicBandwidthUsage,omitempty"`

	// The total public inbound bandwidth for this item's resource for the current billing cycle.
	BillingCyclePublicUsageIn *Float64 `json:"billingCyclePublicUsageIn,omitempty" xmlrpc:"billingCyclePublicUsageIn,omitempty"`

	// The total public outbound bandwidth for this item's resource for the current billing cycle.
	BillingCyclePublicUsageOut *Float64 `json:"billingCyclePublicUsageOut,omitempty" xmlrpc:"billingCyclePublicUsageOut,omitempty"`

	// The total public bandwidth for this item's resource for the current billing cycle.
	BillingCyclePublicUsageTotal *uint `json:"billingCyclePublicUsageTotal,omitempty" xmlrpc:"billingCyclePublicUsageTotal,omitempty"`

	// The virtual rack that this tracking object tracks.
	Resource *Network_Bandwidth_Version1_Allotment `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Metric_Tracking_Object_Virtual_Storage_Repository struct {
	Metric_Tracking_Object_Abstract

	// The virtual storage repository that this tracking object tracks.
	Resource *Virtual_Storage_Repository `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}
