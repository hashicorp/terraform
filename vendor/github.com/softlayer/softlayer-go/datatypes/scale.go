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

// no documentation yet
type Scale_Asset struct {
	Entity

	// When this asset was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// When set and true any edit that happens on this object, be it calling edit on this directly or setting as a child while editing a parent object, will end up being a deletion.
	DeleteFlag *bool `json:"deleteFlag,omitempty" xmlrpc:"deleteFlag,omitempty"`

	// An asset's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The group this asset belongs to.
	ScaleGroup *Scale_Group `json:"scaleGroup,omitempty" xmlrpc:"scaleGroup,omitempty"`

	// The identifier of the group this asset belongs to.
	ScaleGroupId *int `json:"scaleGroupId,omitempty" xmlrpc:"scaleGroupId,omitempty"`
}

// no documentation yet
type Scale_Asset_Hardware struct {
	Scale_Asset

	// The hardware for this asset.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The identifier of the hardware for this asset.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`
}

// no documentation yet
type Scale_Asset_Virtual_Guest struct {
	Scale_Asset

	// The guest for this asset.
	VirtualGuest *Virtual_Guest `json:"virtualGuest,omitempty" xmlrpc:"virtualGuest,omitempty"`

	// The identifier of the guest for this asset.
	VirtualGuestId *int `json:"virtualGuestId,omitempty" xmlrpc:"virtualGuestId,omitempty"`
}

// no documentation yet
type Scale_Group struct {
	Entity

	// The account for this scaling group.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The identifier of the account assigned to this group.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// If this is true, this group will scale down members in a way to preserve the balance across VLANs. If there is ambiguity about which member to use to maintain balance, the terminationPolicy is used to resolve it. This is false by default and can only be set to true if there are multiple VLANs that are being balanced across.
	BalancedTerminationFlag *bool `json:"balancedTerminationFlag,omitempty" xmlrpc:"balancedTerminationFlag,omitempty"`

	// The number of seconds this group will wait after lastActionDate before performing another action. Be advised, this can be overridden per policy. While strongly discouraged, a value of 0 effectively disables cooldown.
	Cooldown *int `json:"cooldown,omitempty" xmlrpc:"cooldown,omitempty"`

	// When this group was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// This value is only available on the template for creating and editing a group. It will be null when retrieved. When this value is provided on create or edit, guests will be scaled up or down to meet this number. This number must be in the range provided by minimumMemberCount and maximumMemberCount. This value can only be present during create or edit when this group is active. Note, guests that are created as a result of this value can possibly be removed after cooldown by a policy.
	DesiredMemberCount *int `json:"desiredMemberCount,omitempty" xmlrpc:"desiredMemberCount,omitempty"`

	// A group's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date of the last action on this group or its create date
	LastActionDate *Time `json:"lastActionDate,omitempty" xmlrpc:"lastActionDate,omitempty"`

	// A count of collection of load balancers for this auto scale group.
	LoadBalancerCount *uint `json:"loadBalancerCount,omitempty" xmlrpc:"loadBalancerCount,omitempty"`

	// Collection of load balancers for this auto scale group.
	LoadBalancers []Scale_LoadBalancer `json:"loadBalancers,omitempty" xmlrpc:"loadBalancers,omitempty"`

	// A count of collection of log entries for this group.
	LogCount *uint `json:"logCount,omitempty" xmlrpc:"logCount,omitempty"`

	// Collection of log entries for this group.
	Logs []Scale_Group_Log `json:"logs,omitempty" xmlrpc:"logs,omitempty"`

	// The greatest number of virtual guest members that are allowed on this group. Any attempts to add a guest member will fail if it will result in the total guest member count of this group to be above this number. If this number is edited and is less than the current guest member count, guests will be removed to at least be no greater than this number.
	MaximumMemberCount *int `json:"maximumMemberCount,omitempty" xmlrpc:"maximumMemberCount,omitempty"`

	// The fewest number of virtual guest members that are allowed on this group. Any attempts to remove a guest member will fail if it will result in the total guest member count of this group to be below this number. If this number is edited and is larger than the current guest member count, guests will be added to at least reach this number.
	MinimumMemberCount *int `json:"minimumMemberCount,omitempty" xmlrpc:"minimumMemberCount,omitempty"`

	// When this group was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of this scale group. It must be unique on the account.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of collection of VLANs for this auto scale group. VLANs are optional. This can contain a public or private VLAN or both. When a single VLAN for a public/private type is given it can be a non-purchased VLAN only if the minimumMemberCount on the group is >= 1. This can also contain any number of public/private purchased VLANs and members are staggered across them when scaled up.
	NetworkVlanCount *uint `json:"networkVlanCount,omitempty" xmlrpc:"networkVlanCount,omitempty"`

	// Collection of VLANs for this auto scale group. VLANs are optional. This can contain a public or private VLAN or both. When a single VLAN for a public/private type is given it can be a non-purchased VLAN only if the minimumMemberCount on the group is >= 1. This can also contain any number of public/private purchased VLANs and members are staggered across them when scaled up.
	NetworkVlans []Scale_Network_Vlan `json:"networkVlans,omitempty" xmlrpc:"networkVlans,omitempty"`

	// Collection of policies for this group. This can be empty.
	Policies []Scale_Policy `json:"policies,omitempty" xmlrpc:"policies,omitempty"`

	// A count of collection of policies for this group. This can be empty.
	PolicyCount *uint `json:"policyCount,omitempty" xmlrpc:"policyCount,omitempty"`

	// The regional group for this scale group.
	RegionalGroup *Location_Group_Regional `json:"regionalGroup,omitempty" xmlrpc:"regionalGroup,omitempty"`

	// The identifier of the regional group this scaling group is assigned to.
	RegionalGroupId *int `json:"regionalGroupId,omitempty" xmlrpc:"regionalGroupId,omitempty"`

	// The status for this scale group.
	Status *Scale_Group_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// If true, this group is suspended.
	SuspendedFlag *bool `json:"suspendedFlag,omitempty" xmlrpc:"suspendedFlag,omitempty"`

	// The termination policy for this scaling group.
	TerminationPolicy *Scale_Termination_Policy `json:"terminationPolicy,omitempty" xmlrpc:"terminationPolicy,omitempty"`

	// The termination policy for the group. This determines which member to choose to delete when scaling downwards.
	TerminationPolicyId *int `json:"terminationPolicyId,omitempty" xmlrpc:"terminationPolicyId,omitempty"`

	// A count of collection of guests that have been pinned to this group. Guest assets are only used for certain trigger checks such as resource watches. They do not count towards the auto scaling guest counts of this group in anyway and are never automatically added or removed.
	VirtualGuestAssetCount *uint `json:"virtualGuestAssetCount,omitempty" xmlrpc:"virtualGuestAssetCount,omitempty"`

	// Collection of guests that have been pinned to this group. Guest assets are only used for certain trigger checks such as resource watches. They do not count towards the auto scaling guest counts of this group in anyway and are never automatically added or removed.
	VirtualGuestAssets []Scale_Asset_Virtual_Guest `json:"virtualGuestAssets,omitempty" xmlrpc:"virtualGuestAssets,omitempty"`

	// A count of collection of guests that have been scaled with the group. When this group is active, the count of guests here is guaranteed to be between minimumMemberCount and maximumMemberCount inclusively.
	VirtualGuestMemberCount *uint `json:"virtualGuestMemberCount,omitempty" xmlrpc:"virtualGuestMemberCount,omitempty"`

	// This is the template to create guest members with. This is the same template accepted by the createObject call on SoftLayer_Virtual_Guest with some caveats. The hostname provided will have an arbitrary value appended to it for each guest created. Also, hourlyBillingFlag cannot be false, and if the datacenter is provided it must be in the region of this group. Finally, VLANs cannot be provided for the template, it will use VLANs provided to this group instead.
	//
	// Note, if this template is edited on an existing group the previous template values are not kept and are not considered during termination. This means a group's guest members could effectively be a hybrid of multiple templates because this value was changed after some guest members were created but before others were created.
	VirtualGuestMemberTemplate *Virtual_Guest `json:"virtualGuestMemberTemplate,omitempty" xmlrpc:"virtualGuestMemberTemplate,omitempty"`

	// Collection of guests that have been scaled with the group. When this group is active, the count of guests here is guaranteed to be between minimumMemberCount and maximumMemberCount inclusively.
	VirtualGuestMembers []Scale_Member_Virtual_Guest `json:"virtualGuestMembers,omitempty" xmlrpc:"virtualGuestMembers,omitempty"`
}

// no documentation yet
type Scale_Group_Log struct {
	Entity

	// When this event occurred.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A textual description of what happened during this action.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// This log's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The group this log refers to.
	ScaleGroup *Scale_Group `json:"scaleGroup,omitempty" xmlrpc:"scaleGroup,omitempty"`

	// The identifier of the group this log refers to.
	ScaleGroupId *int `json:"scaleGroupId,omitempty" xmlrpc:"scaleGroupId,omitempty"`
}

// no documentation yet
type Scale_Group_Status struct {
	Entity

	// A status's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A status's programmatic name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A status's human-friendly name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Scale_LoadBalancer struct {
	Entity

	// The percentage of connections allocated to this virtual server.
	AllocationPercent *int `json:"allocationPercent,omitempty" xmlrpc:"allocationPercent,omitempty"`

	// When this load balancer configuration was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// When set and true any edit that happens on this object, be it calling edit on this directly or setting as a child while editing a parent object, will end up being a deletion.
	DeleteFlag *bool `json:"deleteFlag,omitempty" xmlrpc:"deleteFlag,omitempty"`

	// The health check for this configuration.
	HealthCheck *Network_Application_Delivery_Controller_LoadBalancer_Health_Check `json:"healthCheck,omitempty" xmlrpc:"healthCheck,omitempty"`

	// The identifier for the health check of this load balancer configuration
	HealthCheckId *int `json:"healthCheckId,omitempty" xmlrpc:"healthCheckId,omitempty"`

	// The load balancer configuration's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// When this load balancer configuration was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The port for this load balancer configuration.
	Port *int `json:"port,omitempty" xmlrpc:"port,omitempty"`

	// The routing method.
	RoutingMethod *Network_Application_Delivery_Controller_LoadBalancer_Routing_Method `json:"routingMethod,omitempty" xmlrpc:"routingMethod,omitempty"`

	// The routing type.
	RoutingType *Network_Application_Delivery_Controller_LoadBalancer_Routing_Type `json:"routingType,omitempty" xmlrpc:"routingType,omitempty"`

	// The group this load balancer configuration is for.
	ScaleGroup *Scale_Group `json:"scaleGroup,omitempty" xmlrpc:"scaleGroup,omitempty"`

	// The identifier of the group this load balancer configuration applies to.
	ScaleGroupId *int `json:"scaleGroupId,omitempty" xmlrpc:"scaleGroupId,omitempty"`

	// The ID of the virtual IP address.
	VirtualIpAddressId *int `json:"virtualIpAddressId,omitempty" xmlrpc:"virtualIpAddressId,omitempty"`

	// The virtual server for this configuration.
	VirtualServer *Network_Application_Delivery_Controller_LoadBalancer_VirtualServer `json:"virtualServer,omitempty" xmlrpc:"virtualServer,omitempty"`

	// The identifier of the virtual server this load balancer configuration uses.
	VirtualServerId *int `json:"virtualServerId,omitempty" xmlrpc:"virtualServerId,omitempty"`

	// The port on the virtual server.
	VirtualServerPort *int `json:"virtualServerPort,omitempty" xmlrpc:"virtualServerPort,omitempty"`
}

// no documentation yet
type Scale_Member struct {
	Entity

	// When this member was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A member's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The group this member belongs to.
	ScaleGroup *Scale_Group `json:"scaleGroup,omitempty" xmlrpc:"scaleGroup,omitempty"`

	// The identifier of the group this member belongs to.
	ScaleGroupId *int `json:"scaleGroupId,omitempty" xmlrpc:"scaleGroupId,omitempty"`
}

// no documentation yet
type Scale_Member_Virtual_Guest struct {
	Scale_Member

	// The guest for this member.
	VirtualGuest *Virtual_Guest `json:"virtualGuest,omitempty" xmlrpc:"virtualGuest,omitempty"`

	// The identifier of the guest for this member.
	VirtualGuestId *int `json:"virtualGuestId,omitempty" xmlrpc:"virtualGuestId,omitempty"`
}

// no documentation yet
type Scale_Network_Vlan struct {
	Entity

	// When this network VLAN reference was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// When set and true any edit that happens on this object, be it calling edit on this directly or setting as a child while editing a parent object, will end up being a deletion.
	DeleteFlag *bool `json:"deleteFlag,omitempty" xmlrpc:"deleteFlag,omitempty"`

	// The network VLAN reference's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The network VLAN to scale with.
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`

	// The identifier for the VLAN to scale with.
	NetworkVlanId *int `json:"networkVlanId,omitempty" xmlrpc:"networkVlanId,omitempty"`

	// The group this network VLAN is for.
	ScaleGroup *Scale_Group `json:"scaleGroup,omitempty" xmlrpc:"scaleGroup,omitempty"`

	// The identifier of the group this network VLAN reference applies to.
	ScaleGroupId *int `json:"scaleGroupId,omitempty" xmlrpc:"scaleGroupId,omitempty"`
}

// no documentation yet
type Scale_Policy struct {
	Entity

	// A count of the actions to perform upon any trigger hit. Currently this must be a single value.
	ActionCount *uint `json:"actionCount,omitempty" xmlrpc:"actionCount,omitempty"`

	// The actions to perform upon any trigger hit. Currently this must be a single value.
	Actions []Scale_Policy_Action `json:"actions,omitempty" xmlrpc:"actions,omitempty"`

	// The number of seconds this policy will wait after lastActionDate on group before performing another action. If not present, the group's cooldown value is used.
	Cooldown *int `json:"cooldown,omitempty" xmlrpc:"cooldown,omitempty"`

	// When this policy was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// When set and true any edit that happens on this object, be it calling edit on this directly or setting as a child while editing a parent object, will end up being a deletion.
	DeleteFlag *bool `json:"deleteFlag,omitempty" xmlrpc:"deleteFlag,omitempty"`

	// A policy's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// When this policy was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of this policy. It must be unique within the group.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of the one-time triggers to check for this group.
	OneTimeTriggerCount *uint `json:"oneTimeTriggerCount,omitempty" xmlrpc:"oneTimeTriggerCount,omitempty"`

	// The one-time triggers to check for this group.
	OneTimeTriggers []Scale_Policy_Trigger_OneTime `json:"oneTimeTriggers,omitempty" xmlrpc:"oneTimeTriggers,omitempty"`

	// A count of the repeating triggers to check for this group.
	RepeatingTriggerCount *uint `json:"repeatingTriggerCount,omitempty" xmlrpc:"repeatingTriggerCount,omitempty"`

	// The repeating triggers to check for this group.
	RepeatingTriggers []Scale_Policy_Trigger_Repeating `json:"repeatingTriggers,omitempty" xmlrpc:"repeatingTriggers,omitempty"`

	// A count of the resource-use triggers to check for this group.
	ResourceUseTriggerCount *uint `json:"resourceUseTriggerCount,omitempty" xmlrpc:"resourceUseTriggerCount,omitempty"`

	// The resource-use triggers to check for this group.
	ResourceUseTriggers []Scale_Policy_Trigger_ResourceUse `json:"resourceUseTriggers,omitempty" xmlrpc:"resourceUseTriggers,omitempty"`

	// A count of the scale actions to perform upon any trigger hit. Currently this must be a single value.
	ScaleActionCount *uint `json:"scaleActionCount,omitempty" xmlrpc:"scaleActionCount,omitempty"`

	// The scale actions to perform upon any trigger hit. Currently this must be a single value.
	ScaleActions []Scale_Policy_Action_Scale `json:"scaleActions,omitempty" xmlrpc:"scaleActions,omitempty"`

	// The group this policy is on.
	ScaleGroup *Scale_Group `json:"scaleGroup,omitempty" xmlrpc:"scaleGroup,omitempty"`

	// The identifier of the group this member belongs to.
	ScaleGroupId *int `json:"scaleGroupId,omitempty" xmlrpc:"scaleGroupId,omitempty"`

	// A count of the triggers to check for this group.
	TriggerCount *uint `json:"triggerCount,omitempty" xmlrpc:"triggerCount,omitempty"`

	// The triggers to check for this group.
	Triggers []Scale_Policy_Trigger `json:"triggers,omitempty" xmlrpc:"triggers,omitempty"`
}

// no documentation yet
type Scale_Policy_Action struct {
	Entity

	// When this action was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// When set and true any edit that happens on this object, be it calling edit on this directly or setting as a child while editing a parent object, will end up being a deletion.
	DeleteFlag *bool `json:"deleteFlag,omitempty" xmlrpc:"deleteFlag,omitempty"`

	// An action's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Then this action was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The policy this action is on.
	ScalePolicy *Scale_Policy `json:"scalePolicy,omitempty" xmlrpc:"scalePolicy,omitempty"`

	// The policy this action is on.
	ScalePolicyId *int `json:"scalePolicyId,omitempty" xmlrpc:"scalePolicyId,omitempty"`

	// The type of action.
	Type *Scale_Policy_Action_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The identifier of this action's type.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`
}

// no documentation yet
type Scale_Policy_Action_Scale struct {
	Scale_Policy_Action

	// The number to scale by. This number has different meanings based on type.
	Amount *int `json:"amount,omitempty" xmlrpc:"amount,omitempty"`

	// The type of scale to perform. Possible values:
	//
	//
	// * ABSOLUTE - Force the group to be set at a specific number of group members. This may include scaling up or
	// down or not at all. If the amount is outside of the min/max range of the group, an error occurs.
	// * PERCENT - Scale the group up or down based on the positive or negative percentage given in amount. The
	// number is a percent of the current group member count. Any extra percent after the decimal point is always ignored. If the resulting amount is zero, -1 or 1 is used depending upon whether the percentage was negative or positive respectively.
	// * RELATIVE - Scale the group up or down by the positive or negative value given in amount.
	ScaleType *string `json:"scaleType,omitempty" xmlrpc:"scaleType,omitempty"`
}

// no documentation yet
type Scale_Policy_Action_Type struct {
	Entity

	// This type's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// An action type's programmatic name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// An action type's human-friendly name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Scale_Policy_Trigger struct {
	Entity

	// When this trigger was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// When set and true any edit that happens on this object, be it calling edit on this directly or setting as a child while editing a parent object, will end up being a deletion.
	DeleteFlag *bool `json:"deleteFlag,omitempty" xmlrpc:"deleteFlag,omitempty"`

	// A trigger's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// When this trigger was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The policy this trigger is on.
	ScalePolicy *Scale_Policy `json:"scalePolicy,omitempty" xmlrpc:"scalePolicy,omitempty"`

	// The policy this trigger is on.
	ScalePolicyId *int `json:"scalePolicyId,omitempty" xmlrpc:"scalePolicyId,omitempty"`

	// The type of trigger.
	Type *Scale_Policy_Trigger_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The type of trigger this is.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`
}

// no documentation yet
type Scale_Policy_Trigger_OneTime struct {
	Scale_Policy_Trigger

	// The date to execute the policy.
	Date *Time `json:"date,omitempty" xmlrpc:"date,omitempty"`
}

// no documentation yet
type Scale_Policy_Trigger_Repeating struct {
	Scale_Policy_Trigger

	// The cron-formatted schedule. This is run in the UTC timezone.
	Schedule *string `json:"schedule,omitempty" xmlrpc:"schedule,omitempty"`
}

// no documentation yet
type Scale_Policy_Trigger_ResourceUse struct {
	Scale_Policy_Trigger

	// A count of the resource watches for this trigger.
	WatchCount *uint `json:"watchCount,omitempty" xmlrpc:"watchCount,omitempty"`

	// The resource watches for this trigger.
	Watches []Scale_Policy_Trigger_ResourceUse_Watch `json:"watches,omitempty" xmlrpc:"watches,omitempty"`
}

// no documentation yet
type Scale_Policy_Trigger_ResourceUse_Watch struct {
	Entity

	// The algorithm to use when aggregating and comparing. Currently, the only value that is accepted is EWMA (Exponential Weighted Moving Average). EWMA is the default value if no value is given.
	Algorithm *string `json:"algorithm,omitempty" xmlrpc:"algorithm,omitempty"`

	// When this watch was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// When set and true any edit that happens on this object, be it calling edit on this directly or setting as a child while editing a parent object, will end up being a deletion.
	DeleteFlag *bool `json:"deleteFlag,omitempty" xmlrpc:"deleteFlag,omitempty"`

	// A watch's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The metric to watch. Possible values:
	//
	//
	// * host.cpu.percent - On a scale of 0 to 100, the percent CPU a guest is using.
	// * host.network.backend.in and host.network.frontend.in - The network bytes-per-second incoming on the interface
	// of either the frontend or backend network.
	// * host.network.backend.out and host.network.frontend.out - The network bytes-per-second incoming on the interface
	// of either the frontend or backend network.
	Metric *string `json:"metric,omitempty" xmlrpc:"metric,omitempty"`

	// When this watch was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The operator to use for comparison. The only two valid values are ">" and "<".
	Operator *string `json:"operator,omitempty" xmlrpc:"operator,omitempty"`

	// The number of seconds the values are aggregated for when compared to value. If values are not retrieved steadily and consecutively for the length of this period, nothing is compared.
	Period *int `json:"period,omitempty" xmlrpc:"period,omitempty"`

	// The trigger this watch is on.
	ScalePolicyTrigger *Scale_Policy_Trigger_ResourceUse `json:"scalePolicyTrigger,omitempty" xmlrpc:"scalePolicyTrigger,omitempty"`

	// The trigger this watch is on.
	ScalePolicyTriggerId *int `json:"scalePolicyTriggerId,omitempty" xmlrpc:"scalePolicyTriggerId,omitempty"`

	// The value to compare against. Although the value is a string, validation will be done on the value for restrictions (such as numeric-only) based on the metric.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Scale_Policy_Trigger_Type struct {
	Entity

	// A trigger type's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A trigger type's programmatic name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A trigger type's human-friendly name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Scale_Termination_Policy struct {
	Entity

	// A termination policy's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A termination policy's programmatic name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A termination policy's human-friendly name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}
