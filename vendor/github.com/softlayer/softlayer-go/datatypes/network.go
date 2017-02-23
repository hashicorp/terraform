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
type Network struct {
	Entity

	// The owning account identifier.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The size of the Network specified in CIDR notation. Specified in conjunction with the ``networkIdentifier`` to describe the bounding subnet size for the Network. Required for creation. See [[SoftLayer_Network/createObject]] documentation for creation details.
	Cidr *int `json:"cidr,omitempty" xmlrpc:"cidr,omitempty"`

	// Unique identifier for the network.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A name for the Network. This is required during creation of a Network and is entirely user defined.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The starting IP address of the Network. Specified in conjunction with the ``cidr`` property to specify the bounding IP address space for the Network. Required for creation. See [[SoftLayer_Network/createObject]] documentation for creation details.
	NetworkIdentifier *string `json:"networkIdentifier,omitempty" xmlrpc:"networkIdentifier,omitempty"`

	// Notes, or a description of the Network. This is entirely user defined.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// A count of the Subnets within the Network. These represent the realized segments of the Network and reside within a [[SoftLayer_Network_Pod|Pod]]. A Subnet must be specified when provisioning a compute resource within a Network.
	SubnetCount *uint `json:"subnetCount,omitempty" xmlrpc:"subnetCount,omitempty"`

	// The Subnets within the Network. These represent the realized segments of the Network and reside within a [[SoftLayer_Network_Pod|Pod]]. A Subnet must be specified when provisioning a compute resource within a Network.
	Subnets []Network_Subnet `json:"subnets,omitempty" xmlrpc:"subnets,omitempty"`
}

// The SoftLayer_Network_Application_Delivery_Controller data type models a single instance of an application delivery controller. Local properties are read only, except for a ''notes'' property, which can be used to describe your application delivery controller service. The type's relational properties provide more information to the service's function and login information to the controller's backend management if advanced view is enabled.
type Network_Application_Delivery_Controller struct {
	Entity

	// The SoftLayer customer account that owns an application delivery controller record.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The unique identifier of the SoftLayer customer account that owns an application delivery controller record
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The average daily public bandwidth usage for the current billing cycle.
	AverageDailyPublicBandwidthUsage *Float64 `json:"averageDailyPublicBandwidthUsage,omitempty" xmlrpc:"averageDailyPublicBandwidthUsage,omitempty"`

	// The billing item for a Application Delivery Controller.
	BillingItem *Billing_Item_Network_Application_Delivery_Controller `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// Previous configurations for an Application Delivery Controller.
	ConfigurationHistory []Network_Application_Delivery_Controller_Configuration_History `json:"configurationHistory,omitempty" xmlrpc:"configurationHistory,omitempty"`

	// A count of previous configurations for an Application Delivery Controller.
	ConfigurationHistoryCount *uint `json:"configurationHistoryCount,omitempty" xmlrpc:"configurationHistoryCount,omitempty"`

	// The date that an application delivery controller record was created
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The datacenter that the application delivery controller resides in.
	Datacenter *Location `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// A brief description of an application delivery controller record.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// An application delivery controller's unique identifier
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date in which the license for this application delivery controller will expire.
	LicenseExpirationDate *Time `json:"licenseExpirationDate,omitempty" xmlrpc:"licenseExpirationDate,omitempty"`

	// A count of the virtual IP address records that belong to an application delivery controller based load balancer.
	LoadBalancerCount *uint `json:"loadBalancerCount,omitempty" xmlrpc:"loadBalancerCount,omitempty"`

	// The virtual IP address records that belong to an application delivery controller based load balancer.
	LoadBalancers []Network_LoadBalancer_VirtualIpAddress `json:"loadBalancers,omitempty" xmlrpc:"loadBalancers,omitempty"`

	// A flag indicating that this Application Delivery Controller is a managed resource.
	ManagedResourceFlag *bool `json:"managedResourceFlag,omitempty" xmlrpc:"managedResourceFlag,omitempty"`

	// An application delivery controller's management ip address.
	ManagementIpAddress *string `json:"managementIpAddress,omitempty" xmlrpc:"managementIpAddress,omitempty"`

	// The date that an application delivery controller record was last modified
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// An application delivery controller's name
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The network VLAN that an application delivery controller resides on.
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`

	// A count of the network VLANs that an application delivery controller resides on.
	NetworkVlanCount *uint `json:"networkVlanCount,omitempty" xmlrpc:"networkVlanCount,omitempty"`

	// The network VLANs that an application delivery controller resides on.
	NetworkVlans []Network_Vlan `json:"networkVlans,omitempty" xmlrpc:"networkVlans,omitempty"`

	// Editable notes used to describe an application delivery controller's function
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The total public outbound bandwidth for the current billing cycle.
	OutboundPublicBandwidthUsage *Float64 `json:"outboundPublicBandwidthUsage,omitempty" xmlrpc:"outboundPublicBandwidthUsage,omitempty"`

	// The password used to connect to an application delivery controller's management interface when it is operating in advanced view mode.
	Password *Software_Component_Password `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// An application delivery controller's primary public IP address.
	PrimaryIpAddress *string `json:"primaryIpAddress,omitempty" xmlrpc:"primaryIpAddress,omitempty"`

	// The projected public outbound bandwidth for the current billing cycle.
	ProjectedPublicBandwidthUsage *Float64 `json:"projectedPublicBandwidthUsage,omitempty" xmlrpc:"projectedPublicBandwidthUsage,omitempty"`

	// A count of a network application controller's subnets. A subnet is a group of IP addresses
	SubnetCount *uint `json:"subnetCount,omitempty" xmlrpc:"subnetCount,omitempty"`

	// A network application controller's subnets. A subnet is a group of IP addresses
	Subnets []Network_Subnet `json:"subnets,omitempty" xmlrpc:"subnets,omitempty"`

	// A count of
	TagReferenceCount *uint `json:"tagReferenceCount,omitempty" xmlrpc:"tagReferenceCount,omitempty"`

	// no documentation yet
	TagReferences []Tag_Reference `json:"tagReferences,omitempty" xmlrpc:"tagReferences,omitempty"`

	// no documentation yet
	Type *Network_Application_Delivery_Controller_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// no documentation yet
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// A count of
	VirtualIpAddressCount *uint `json:"virtualIpAddressCount,omitempty" xmlrpc:"virtualIpAddressCount,omitempty"`

	// no documentation yet
	VirtualIpAddresses []Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress `json:"virtualIpAddresses,omitempty" xmlrpc:"virtualIpAddresses,omitempty"`
}

// The SoftLayer_Network_Application_Delivery_Controller_Configuration_History data type models a single instance of a configuration history entry for an application delivery controller. The configuration history entries are used to support creating backups of an application delivery controller's configuration state in order to restore them later if needed.
type Network_Application_Delivery_Controller_Configuration_History struct {
	Entity

	// The application delivery controller that a configuration history record belongs to.
	Controller *Network_Application_Delivery_Controller `json:"controller,omitempty" xmlrpc:"controller,omitempty"`

	// The date a configuration history record was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// An configuration history record's unique identifier
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Editable notes used to describe a configuration history record
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute struct {
	Entity

	// no documentation yet
	HealthAttributeTypeId *int `json:"healthAttributeTypeId,omitempty" xmlrpc:"healthAttributeTypeId,omitempty"`

	// no documentation yet
	HealthCheck *Network_Application_Delivery_Controller_LoadBalancer_Health_Check `json:"healthCheck,omitempty" xmlrpc:"healthCheck,omitempty"`

	// no documentation yet
	HealthCheckId *int `json:"healthCheckId,omitempty" xmlrpc:"healthCheckId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Type *Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// no documentation yet
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute_Type struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	ValueExpression *string `json:"valueExpression,omitempty" xmlrpc:"valueExpression,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Health_Check struct {
	Entity

	// A count of
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// no documentation yet
	Attributes []Network_Application_Delivery_Controller_LoadBalancer_Health_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// no documentation yet
	HealthCheckTypeId *int `json:"healthCheckTypeId,omitempty" xmlrpc:"healthCheckTypeId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// A count of collection of scale load balancers that use this health check.
	ScaleLoadBalancerCount *uint `json:"scaleLoadBalancerCount,omitempty" xmlrpc:"scaleLoadBalancerCount,omitempty"`

	// Collection of scale load balancers that use this health check.
	ScaleLoadBalancers []Scale_LoadBalancer `json:"scaleLoadBalancers,omitempty" xmlrpc:"scaleLoadBalancers,omitempty"`

	// A count of
	ServiceCount *uint `json:"serviceCount,omitempty" xmlrpc:"serviceCount,omitempty"`

	// no documentation yet
	Services []Network_Application_Delivery_Controller_LoadBalancer_Service `json:"services,omitempty" xmlrpc:"services,omitempty"`

	// no documentation yet
	Type *Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Health_Check_Type struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Routing_Method struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Routing_Type struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Service struct {
	Entity

	// no documentation yet
	Enabled *int `json:"enabled,omitempty" xmlrpc:"enabled,omitempty"`

	// A count of
	GroupCount *uint `json:"groupCount,omitempty" xmlrpc:"groupCount,omitempty"`

	// A count of
	GroupReferenceCount *uint `json:"groupReferenceCount,omitempty" xmlrpc:"groupReferenceCount,omitempty"`

	// no documentation yet
	GroupReferences []Network_Application_Delivery_Controller_LoadBalancer_Service_Group_CrossReference `json:"groupReferences,omitempty" xmlrpc:"groupReferences,omitempty"`

	// no documentation yet
	Groups []Network_Application_Delivery_Controller_LoadBalancer_Service_Group `json:"groups,omitempty" xmlrpc:"groups,omitempty"`

	// no documentation yet
	HealthCheck *Network_Application_Delivery_Controller_LoadBalancer_Health_Check `json:"healthCheck,omitempty" xmlrpc:"healthCheck,omitempty"`

	// A count of
	HealthCheckCount *uint `json:"healthCheckCount,omitempty" xmlrpc:"healthCheckCount,omitempty"`

	// no documentation yet
	HealthChecks []Network_Application_Delivery_Controller_LoadBalancer_Health_Check `json:"healthChecks,omitempty" xmlrpc:"healthChecks,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	IpAddress *Network_Subnet_IpAddress `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// no documentation yet
	IpAddressId *int `json:"ipAddressId,omitempty" xmlrpc:"ipAddressId,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// no documentation yet
	Port *int `json:"port,omitempty" xmlrpc:"port,omitempty"`

	// no documentation yet
	ServiceGroup *Network_Application_Delivery_Controller_LoadBalancer_Service_Group `json:"serviceGroup,omitempty" xmlrpc:"serviceGroup,omitempty"`

	// no documentation yet
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Service_Group struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// no documentation yet
	RoutingMethod *Network_Application_Delivery_Controller_LoadBalancer_Routing_Method `json:"routingMethod,omitempty" xmlrpc:"routingMethod,omitempty"`

	// no documentation yet
	RoutingMethodId *int `json:"routingMethodId,omitempty" xmlrpc:"routingMethodId,omitempty"`

	// no documentation yet
	RoutingType *Network_Application_Delivery_Controller_LoadBalancer_Routing_Type `json:"routingType,omitempty" xmlrpc:"routingType,omitempty"`

	// no documentation yet
	RoutingTypeId *int `json:"routingTypeId,omitempty" xmlrpc:"routingTypeId,omitempty"`

	// A count of
	ServiceCount *uint `json:"serviceCount,omitempty" xmlrpc:"serviceCount,omitempty"`

	// A count of
	ServiceReferenceCount *uint `json:"serviceReferenceCount,omitempty" xmlrpc:"serviceReferenceCount,omitempty"`

	// no documentation yet
	ServiceReferences []Network_Application_Delivery_Controller_LoadBalancer_Service_Group_CrossReference `json:"serviceReferences,omitempty" xmlrpc:"serviceReferences,omitempty"`

	// no documentation yet
	Services []Network_Application_Delivery_Controller_LoadBalancer_Service `json:"services,omitempty" xmlrpc:"services,omitempty"`

	// The timeout value for connections from remote clients to the load balancer. Timeout values are only valid for HTTP service groups.
	Timeout *int `json:"timeout,omitempty" xmlrpc:"timeout,omitempty"`

	// no documentation yet
	VirtualServer *Network_Application_Delivery_Controller_LoadBalancer_VirtualServer `json:"virtualServer,omitempty" xmlrpc:"virtualServer,omitempty"`

	// A count of
	VirtualServerCount *uint `json:"virtualServerCount,omitempty" xmlrpc:"virtualServerCount,omitempty"`

	// no documentation yet
	VirtualServers []Network_Application_Delivery_Controller_LoadBalancer_VirtualServer `json:"virtualServers,omitempty" xmlrpc:"virtualServers,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_Service_Group_CrossReference struct {
	Entity

	// no documentation yet
	Service *Network_Application_Delivery_Controller_LoadBalancer_Service `json:"service,omitempty" xmlrpc:"service,omitempty"`

	// no documentation yet
	ServiceGroup *Network_Application_Delivery_Controller_LoadBalancer_Service_Group `json:"serviceGroup,omitempty" xmlrpc:"serviceGroup,omitempty"`

	// no documentation yet
	ServiceGroupId *int `json:"serviceGroupId,omitempty" xmlrpc:"serviceGroupId,omitempty"`

	// no documentation yet
	ServiceId *int `json:"serviceId,omitempty" xmlrpc:"serviceId,omitempty"`

	// no documentation yet
	Weight *int `json:"weight,omitempty" xmlrpc:"weight,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The unique identifier of the SoftLayer customer account that owns the virtual IP address
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A virtual IP address's associated application delivery controller.
	ApplicationDeliveryController *Network_Application_Delivery_Controller `json:"applicationDeliveryController,omitempty" xmlrpc:"applicationDeliveryController,omitempty"`

	// A count of a virtual IP address's associated application delivery controllers.
	ApplicationDeliveryControllerCount *uint `json:"applicationDeliveryControllerCount,omitempty" xmlrpc:"applicationDeliveryControllerCount,omitempty"`

	// A virtual IP address's associated application delivery controllers.
	ApplicationDeliveryControllers []Network_Application_Delivery_Controller `json:"applicationDeliveryControllers,omitempty" xmlrpc:"applicationDeliveryControllers,omitempty"`

	// The current billing item for the load balancer virtual IP. This is only valid when dedicatedFlag is false. This is an independent virtual IP, and if canceled, will only affect the associated virtual IP.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The connection limit for this virtual IP address
	ConnectionLimit *int `json:"connectionLimit,omitempty" xmlrpc:"connectionLimit,omitempty"`

	// The units for the connection limit
	ConnectionLimitUnits *string `json:"connectionLimitUnits,omitempty" xmlrpc:"connectionLimitUnits,omitempty"`

	// The current billing item for the load balancing device housing the virtual IP. This billing item represents a device which could contain other virtual IPs. Caution should be taken when canceling. This is only valid when dedicatedFlag is true.
	DedicatedBillingItem *Billing_Item_Network_LoadBalancer `json:"dedicatedBillingItem,omitempty" xmlrpc:"dedicatedBillingItem,omitempty"`

	// A flag that determines if a VIP is dedicated or not. This is used to override the connection limit and use an unlimited value.
	DedicatedFlag *bool `json:"dedicatedFlag,omitempty" xmlrpc:"dedicatedFlag,omitempty"`

	// Denotes whether the virtual IP is configured within a high availability cluster.
	HighAvailabilityFlag *bool `json:"highAvailabilityFlag,omitempty" xmlrpc:"highAvailabilityFlag,omitempty"`

	// The unique identifier of the virtual IP address record
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	IpAddress *Network_Subnet_IpAddress `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// ID of the IP address this virtual IP utilizes
	IpAddressId *int `json:"ipAddressId,omitempty" xmlrpc:"ipAddressId,omitempty"`

	// no documentation yet
	LoadBalancerHardware []Hardware `json:"loadBalancerHardware,omitempty" xmlrpc:"loadBalancerHardware,omitempty"`

	// A count of
	LoadBalancerHardwareCount *uint `json:"loadBalancerHardwareCount,omitempty" xmlrpc:"loadBalancerHardwareCount,omitempty"`

	// A flag indicating that the load balancer is a managed resource.
	ManagedResourceFlag *bool `json:"managedResourceFlag,omitempty" xmlrpc:"managedResourceFlag,omitempty"`

	// User-created notes for this load balancer virtual IP address
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// A count of the list of security ciphers enabled for this virtual IP address
	SecureTransportCipherCount *uint `json:"secureTransportCipherCount,omitempty" xmlrpc:"secureTransportCipherCount,omitempty"`

	// The list of security ciphers enabled for this virtual IP address
	SecureTransportCiphers []Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress_SecureTransportCipher `json:"secureTransportCiphers,omitempty" xmlrpc:"secureTransportCiphers,omitempty"`

	// A count of the list of secure transport protocols enabled for this virtual IP address
	SecureTransportProtocolCount *uint `json:"secureTransportProtocolCount,omitempty" xmlrpc:"secureTransportProtocolCount,omitempty"`

	// The list of secure transport protocols enabled for this virtual IP address
	SecureTransportProtocols []Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress_SecureTransportProtocol `json:"secureTransportProtocols,omitempty" xmlrpc:"secureTransportProtocols,omitempty"`

	// The SSL certificate currently associated with the VIP.
	SecurityCertificate *Security_Certificate `json:"securityCertificate,omitempty" xmlrpc:"securityCertificate,omitempty"`

	// The SSL certificate currently associated with the VIP. Provides chosen certificate visibility to unprivileged users.
	SecurityCertificateEntry *Security_Certificate_Entry `json:"securityCertificateEntry,omitempty" xmlrpc:"securityCertificateEntry,omitempty"`

	// The unique identifier of the Security Certificate to be utilized when SSL support is enabled.
	SecurityCertificateId *int `json:"securityCertificateId,omitempty" xmlrpc:"securityCertificateId,omitempty"`

	// Determines if the VIP currently has SSL acceleration enabled
	SslActiveFlag *bool `json:"sslActiveFlag,omitempty" xmlrpc:"sslActiveFlag,omitempty"`

	// Determines if the VIP is _allowed_ to utilize SSL acceleration
	SslEnabledFlag *bool `json:"sslEnabledFlag,omitempty" xmlrpc:"sslEnabledFlag,omitempty"`

	// A count of
	VirtualServerCount *uint `json:"virtualServerCount,omitempty" xmlrpc:"virtualServerCount,omitempty"`

	// no documentation yet
	VirtualServers []Network_Application_Delivery_Controller_LoadBalancer_VirtualServer `json:"virtualServers,omitempty" xmlrpc:"virtualServers,omitempty"`
}

// A single cipher configured for a load balancer virtual IP address instance. Instances of this class are immutable and should reflect a cipher that is configurable on a load balancer device.
type Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress_SecureTransportCipher struct {
	Entity

	// Unique identifier for the cipher instance
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Identifier for the associated encryption algorithm
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	VirtualIpAddress *Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress `json:"virtualIpAddress,omitempty" xmlrpc:"virtualIpAddress,omitempty"`

	// Identifier for the associated [[SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress (type)|virtual IP address]] instance
	VirtualIpAddressId *int `json:"virtualIpAddressId,omitempty" xmlrpc:"virtualIpAddressId,omitempty"`
}

// Links a SSL transport protocol with a virtual IP address instance. Instances of this class are immutable and should reflect a protocol that is configurable on a load balancer device.
type Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress_SecureTransportProtocol struct {
	Entity

	// Unique identifier for the protocol instance
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Identifier for the associated communication protocol
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	VirtualIpAddress *Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress `json:"virtualIpAddress,omitempty" xmlrpc:"virtualIpAddress,omitempty"`

	// Identifier for the associated [[SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress (type)|virtual IP address]] instance
	VirtualIpAddressId *int `json:"virtualIpAddressId,omitempty" xmlrpc:"virtualIpAddressId,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_LoadBalancer_VirtualServer struct {
	Entity

	// no documentation yet
	Allocation *int `json:"allocation,omitempty" xmlrpc:"allocation,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// no documentation yet
	Port *int `json:"port,omitempty" xmlrpc:"port,omitempty"`

	// no documentation yet
	RoutingMethod *Network_Application_Delivery_Controller_LoadBalancer_Routing_Method `json:"routingMethod,omitempty" xmlrpc:"routingMethod,omitempty"`

	// no documentation yet
	RoutingMethodId *int `json:"routingMethodId,omitempty" xmlrpc:"routingMethodId,omitempty"`

	// A count of collection of scale load balancers this virtual server applies to.
	ScaleLoadBalancerCount *uint `json:"scaleLoadBalancerCount,omitempty" xmlrpc:"scaleLoadBalancerCount,omitempty"`

	// Collection of scale load balancers this virtual server applies to.
	ScaleLoadBalancers []Scale_LoadBalancer `json:"scaleLoadBalancers,omitempty" xmlrpc:"scaleLoadBalancers,omitempty"`

	// A count of
	ServiceGroupCount *uint `json:"serviceGroupCount,omitempty" xmlrpc:"serviceGroupCount,omitempty"`

	// no documentation yet
	ServiceGroups []Network_Application_Delivery_Controller_LoadBalancer_Service_Group `json:"serviceGroups,omitempty" xmlrpc:"serviceGroups,omitempty"`

	// no documentation yet
	VirtualIpAddress *Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress `json:"virtualIpAddress,omitempty" xmlrpc:"virtualIpAddress,omitempty"`

	// no documentation yet
	VirtualIpAddressId *int `json:"virtualIpAddressId,omitempty" xmlrpc:"virtualIpAddressId,omitempty"`
}

// no documentation yet
type Network_Application_Delivery_Controller_Type struct {
	Entity

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// A SoftLayer_Network_Backbone represents a single backbone connection from SoftLayer to the public Internet, from the Internet to the SoftLayer private network, or a link that connects the private networks between SoftLayer's datacenters. The SoftLayer_Network_Backbone data type is a collection of data associated with one of those connections.
type Network_Backbone struct {
	Entity

	// The numeric portion of the bandwidth capacity of a SoftLayer backbone. For instance, if a backbone is rated at "1 GigE" capacity then the capacity property of the backbone is 1.
	Capacity *int `json:"capacity,omitempty" xmlrpc:"capacity,omitempty"`

	// The unit portion of the bandwidth capacity of a SoftLayer backbone. For instance, if a backbone is rated at "10 G" capacity then the capacityUnits property of the backbone is "G".
	CapacityUnits *string `json:"capacityUnits,omitempty" xmlrpc:"capacityUnits,omitempty"`

	// A backbone's status.
	Health *string `json:"health,omitempty" xmlrpc:"health,omitempty"`

	// A backbone's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Which of the SoftLayer datacenters a backbone is connected to.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// A backbone's name. This is usually the name of the backbone's network provider followed by a number in case SoftLayer uses more than one backbone from a provider. Backbone provider numbers start with the number one and increment from there.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A backbone's primary network component.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`

	// The internal identifier of the network component that backbone is connected to.
	NetworkComponentId *int `json:"networkComponentId,omitempty" xmlrpc:"networkComponentId,omitempty"`

	// Whether a SoftLayer backbone connects to the public Internet, to the private network, or connecting the private networks of SoftLayer's datacenters. Type is either the string "public", "private", or "private-interconnect".
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// no documentation yet
type Network_Backbone_Location_Dependent struct {
	Entity

	// no documentation yet
	DependentLocation *Location `json:"dependentLocation,omitempty" xmlrpc:"dependentLocation,omitempty"`

	// no documentation yet
	DependentLocationId *int `json:"dependentLocationId,omitempty" xmlrpc:"dependentLocationId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	SourceLocation *Location `json:"sourceLocation,omitempty" xmlrpc:"sourceLocation,omitempty"`

	// no documentation yet
	SourceLocationId *int `json:"sourceLocationId,omitempty" xmlrpc:"sourceLocationId,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Usage data type contains specific information relating to bandwidth utilization at a specific point in time on a given network interface.
type Network_Bandwidth_Usage struct {
	Entity

	// Incoming bandwidth utilization.
	AmountIn *Float64 `json:"amountIn,omitempty" xmlrpc:"amountIn,omitempty"`

	// Outgoing bandwidth utilization.
	AmountOut *Float64 `json:"amountOut,omitempty" xmlrpc:"amountOut,omitempty"`

	// ID of the bandwidth usage detail type for this record.
	BandwidthUsageDetailTypeId *Float64 `json:"bandwidthUsageDetailTypeId,omitempty" xmlrpc:"bandwidthUsageDetailTypeId,omitempty"`

	// The tracking object this bandwidth usage record describes.
	TrackingObject *Metric_Tracking_Object `json:"trackingObject,omitempty" xmlrpc:"trackingObject,omitempty"`

	// In and out bandwidth utilization for a specified time stamp.
	Type *Network_Bandwidth_Version1_Usage_Detail_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Usage_Detail data type contains specific information relating to bandwidth utilization at a specific point in time on a given network interface.
type Network_Bandwidth_Usage_Detail struct {
	Entity

	// The account tied to this tracking object
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// Incoming bandwidth utilization.
	AmountIn *Float64 `json:"amountIn,omitempty" xmlrpc:"amountIn,omitempty"`

	// Outgoing bandwidth utilization.
	AmountOut *Float64 `json:"amountOut,omitempty" xmlrpc:"amountOut,omitempty"`

	// ID of the bandwidth usage detail type for this record.
	BandwidthUsageDetailTypeId *Float64 `json:"bandwidthUsageDetailTypeId,omitempty" xmlrpc:"bandwidthUsageDetailTypeId,omitempty"`

	// The tracking object this bandwidth usage record describes.
	TrackingObject *Metric_Tracking_Object `json:"trackingObject,omitempty" xmlrpc:"trackingObject,omitempty"`

	// In and out bandwidth utilization for a specified time stamp.
	Type *Network_Bandwidth_Version1_Usage_Detail_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Version1_Allocation data type contains general information relating to a single bandwidth allocation record.
type Network_Bandwidth_Version1_Allocation struct {
	Entity

	// A bandwidth allotment detail.
	AllotmentDetail *Network_Bandwidth_Version1_Allotment_Detail `json:"allotmentDetail,omitempty" xmlrpc:"allotmentDetail,omitempty"`

	// The amount of bandwidth allocated.
	Amount *Float64 `json:"amount,omitempty" xmlrpc:"amount,omitempty"`

	// Billing item associated with this hardware allocation.
	BillingItem *Billing_Item_Hardware `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// Internal ID associated with this allocation.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Version1_Allotment class provides methods and data structures necessary to work with an array of hardware objects associated with a single Bandwidth Pooling.
type Network_Bandwidth_Version1_Allotment struct {
	Entity

	// The account associated with this virtual rack.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The user account identifier associated with this allotment.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of the bandwidth allotment detail records associated with this virtual rack.
	ActiveDetailCount *uint `json:"activeDetailCount,omitempty" xmlrpc:"activeDetailCount,omitempty"`

	// The bandwidth allotment detail records associated with this virtual rack.
	ActiveDetails []Network_Bandwidth_Version1_Allotment_Detail `json:"activeDetails,omitempty" xmlrpc:"activeDetails,omitempty"`

	// A count of the Application Delivery Controller contained within a virtual rack.
	ApplicationDeliveryControllerCount *uint `json:"applicationDeliveryControllerCount,omitempty" xmlrpc:"applicationDeliveryControllerCount,omitempty"`

	// The Application Delivery Controller contained within a virtual rack.
	ApplicationDeliveryControllers []Network_Application_Delivery_Controller `json:"applicationDeliveryControllers,omitempty" xmlrpc:"applicationDeliveryControllers,omitempty"`

	// The average daily public bandwidth usage for the current billing cycle.
	AverageDailyPublicBandwidthUsage *Float64 `json:"averageDailyPublicBandwidthUsage,omitempty" xmlrpc:"averageDailyPublicBandwidthUsage,omitempty"`

	// An identifier marking this allotment as a virtual private rack (1) or a bandwidth pooling(2).
	BandwidthAllotmentTypeId *int `json:"bandwidthAllotmentTypeId,omitempty" xmlrpc:"bandwidthAllotmentTypeId,omitempty"`

	// A count of the bare metal server instances contained within a virtual rack.
	BareMetalInstanceCount *uint `json:"bareMetalInstanceCount,omitempty" xmlrpc:"bareMetalInstanceCount,omitempty"`

	// The bare metal server instances contained within a virtual rack.
	BareMetalInstances []Hardware `json:"bareMetalInstances,omitempty" xmlrpc:"bareMetalInstances,omitempty"`

	// A virtual rack's raw bandwidth usage data for an account's current billing cycle. One object is returned for each network this server is attached to.
	BillingCycleBandwidthUsage []Network_Bandwidth_Usage `json:"billingCycleBandwidthUsage,omitempty" xmlrpc:"billingCycleBandwidthUsage,omitempty"`

	// A count of a virtual rack's raw bandwidth usage data for an account's current billing cycle. One object is returned for each network this server is attached to.
	BillingCycleBandwidthUsageCount *uint `json:"billingCycleBandwidthUsageCount,omitempty" xmlrpc:"billingCycleBandwidthUsageCount,omitempty"`

	// A virtual rack's raw private network bandwidth usage data for an account's current billing cycle.
	BillingCyclePrivateBandwidthUsage *Network_Bandwidth_Usage `json:"billingCyclePrivateBandwidthUsage,omitempty" xmlrpc:"billingCyclePrivateBandwidthUsage,omitempty"`

	// A virtual rack's raw public network bandwidth usage data for an account's current billing cycle.
	BillingCyclePublicBandwidthUsage *Network_Bandwidth_Usage `json:"billingCyclePublicBandwidthUsage,omitempty" xmlrpc:"billingCyclePublicBandwidthUsage,omitempty"`

	// The total public bandwidth used in this virtual rack for an account's current billing cycle.
	BillingCyclePublicUsageTotal *uint `json:"billingCyclePublicUsageTotal,omitempty" xmlrpc:"billingCyclePublicUsageTotal,omitempty"`

	// A virtual rack's billing item.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// Creation date for an allotment.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// An object that provides commonly used bandwidth summary components for the current billing cycle.
	CurrentBandwidthSummary *Metric_Tracking_Object_Bandwidth_Summary `json:"currentBandwidthSummary,omitempty" xmlrpc:"currentBandwidthSummary,omitempty"`

	// A count of the bandwidth allotment detail records associated with this virtual rack.
	DetailCount *uint `json:"detailCount,omitempty" xmlrpc:"detailCount,omitempty"`

	// The bandwidth allotment detail records associated with this virtual rack.
	Details []Network_Bandwidth_Version1_Allotment_Detail `json:"details,omitempty" xmlrpc:"details,omitempty"`

	// End date for an allotment.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// The hardware contained within a virtual rack.
	Hardware []Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// A count of the hardware contained within a virtual rack.
	HardwareCount *uint `json:"hardwareCount,omitempty" xmlrpc:"hardwareCount,omitempty"`

	// A virtual rack's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The total public inbound bandwidth used in this virtual rack for an account's current billing cycle.
	InboundPublicBandwidthUsage *Float64 `json:"inboundPublicBandwidthUsage,omitempty" xmlrpc:"inboundPublicBandwidthUsage,omitempty"`

	// The location group associated with this virtual rack.
	LocationGroup *Location_Group `json:"locationGroup,omitempty" xmlrpc:"locationGroup,omitempty"`

	// Location Group Id for an allotment
	LocationGroupId *int `json:"locationGroupId,omitempty" xmlrpc:"locationGroupId,omitempty"`

	// A count of the managed bare metal server instances contained within a virtual rack.
	ManagedBareMetalInstanceCount *uint `json:"managedBareMetalInstanceCount,omitempty" xmlrpc:"managedBareMetalInstanceCount,omitempty"`

	// The managed bare metal server instances contained within a virtual rack.
	ManagedBareMetalInstances []Hardware `json:"managedBareMetalInstances,omitempty" xmlrpc:"managedBareMetalInstances,omitempty"`

	// The managed hardware contained within a virtual rack.
	ManagedHardware []Hardware `json:"managedHardware,omitempty" xmlrpc:"managedHardware,omitempty"`

	// A count of the managed hardware contained within a virtual rack.
	ManagedHardwareCount *uint `json:"managedHardwareCount,omitempty" xmlrpc:"managedHardwareCount,omitempty"`

	// A count of the managed Virtual Server contained within a virtual rack.
	ManagedVirtualGuestCount *uint `json:"managedVirtualGuestCount,omitempty" xmlrpc:"managedVirtualGuestCount,omitempty"`

	// The managed Virtual Server contained within a virtual rack.
	ManagedVirtualGuests []Virtual_Guest `json:"managedVirtualGuests,omitempty" xmlrpc:"managedVirtualGuests,omitempty"`

	// A virtual rack's metric tracking object. This object records all periodic polled data available to this rack.
	MetricTrackingObject *Metric_Tracking_Object_VirtualDedicatedRack `json:"metricTrackingObject,omitempty" xmlrpc:"metricTrackingObject,omitempty"`

	// The metric tracking object id for this allotment.
	MetricTrackingObjectId *int `json:"metricTrackingObjectId,omitempty" xmlrpc:"metricTrackingObjectId,omitempty"`

	// Text A virtual rack's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The total public outbound bandwidth used in this virtual rack for an account's current billing cycle.
	OutboundPublicBandwidthUsage *Float64 `json:"outboundPublicBandwidthUsage,omitempty" xmlrpc:"outboundPublicBandwidthUsage,omitempty"`

	// Whether the bandwidth usage for this bandwidth pool for the current billing cycle exceeds the allocation.
	OverBandwidthAllocationFlag *int `json:"overBandwidthAllocationFlag,omitempty" xmlrpc:"overBandwidthAllocationFlag,omitempty"`

	// The private network only hardware contained within a virtual rack.
	PrivateNetworkOnlyHardware []Hardware `json:"privateNetworkOnlyHardware,omitempty" xmlrpc:"privateNetworkOnlyHardware,omitempty"`

	// A count of the private network only hardware contained within a virtual rack.
	PrivateNetworkOnlyHardwareCount *uint `json:"privateNetworkOnlyHardwareCount,omitempty" xmlrpc:"privateNetworkOnlyHardwareCount,omitempty"`

	// Whether the bandwidth usage for this bandwidth pool for the current billing cycle is projected to exceed the allocation.
	ProjectedOverBandwidthAllocationFlag *int `json:"projectedOverBandwidthAllocationFlag,omitempty" xmlrpc:"projectedOverBandwidthAllocationFlag,omitempty"`

	// The projected public outbound bandwidth for this virtual server for the current billing cycle.
	ProjectedPublicBandwidthUsage *Float64 `json:"projectedPublicBandwidthUsage,omitempty" xmlrpc:"projectedPublicBandwidthUsage,omitempty"`

	// no documentation yet
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`

	// Service Provider Id for an allotment
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`

	// The combined allocated bandwidth for all servers in a virtual rack.
	TotalBandwidthAllocated *uint `json:"totalBandwidthAllocated,omitempty" xmlrpc:"totalBandwidthAllocated,omitempty"`

	// A count of the Virtual Server contained within a virtual rack.
	VirtualGuestCount *uint `json:"virtualGuestCount,omitempty" xmlrpc:"virtualGuestCount,omitempty"`

	// The Virtual Server contained within a virtual rack.
	VirtualGuests []Virtual_Guest `json:"virtualGuests,omitempty" xmlrpc:"virtualGuests,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Version1_Allotment_Detail data type contains specific information relating to a single bandwidth allotment record.
type Network_Bandwidth_Version1_Allotment_Detail struct {
	Entity

	// Allocated bandwidth.
	Allocation *Network_Bandwidth_Version1_Allocation `json:"allocation,omitempty" xmlrpc:"allocation,omitempty"`

	// Allocated bandwidth.
	AllocationId *int `json:"allocationId,omitempty" xmlrpc:"allocationId,omitempty"`

	// The parent Bandwidth Pool.
	BandwidthAllotment *Network_Bandwidth_Version1_Allotment `json:"bandwidthAllotment,omitempty" xmlrpc:"bandwidthAllotment,omitempty"`

	// Bandwidth Pool associated with this detail.
	BandwidthAllotmentId *int `json:"bandwidthAllotmentId,omitempty" xmlrpc:"bandwidthAllotmentId,omitempty"`

	// Bandwidth used.
	BandwidthUsage []Network_Bandwidth_Version1_Usage `json:"bandwidthUsage,omitempty" xmlrpc:"bandwidthUsage,omitempty"`

	// A count of bandwidth used.
	BandwidthUsageCount *uint `json:"bandwidthUsageCount,omitempty" xmlrpc:"bandwidthUsageCount,omitempty"`

	// Beginning this date the bandwidth allotment is active.
	EffectiveDate *Time `json:"effectiveDate,omitempty" xmlrpc:"effectiveDate,omitempty"`

	// From this date the bandwidth allotment is no longer active.
	EndEffectiveDate *Time `json:"endEffectiveDate,omitempty" xmlrpc:"endEffectiveDate,omitempty"`

	// Internal ID associated with this allotment detail.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Service Provider Id for an allotment
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Version1_Host type contains general information used to the route a server to its pod.
type Network_Bandwidth_Version1_Host struct {
	Entity

	// Pod ID for this host device.
	PodId *int `json:"podId,omitempty" xmlrpc:"podId,omitempty"`
}

// All bandwidth tracking is maintained through the switch that the bandwidth is used through.  All bandwidth is stored in a "pod" repository.  An interface links the hardware switch with the pod repository identification number. This is only relevant to bandwidth data.  It is not common to use this.
type Network_Bandwidth_Version1_Interface struct {
	Entity

	// The host for an interface. This is not to be confused with a SoftLayer hardware
	Host *Network_Bandwidth_Version1_Host `json:"host,omitempty" xmlrpc:"host,omitempty"`

	// A interface's host.  The host stores the pod number for the bandwidth data.
	HostId *int `json:"hostId,omitempty" xmlrpc:"hostId,omitempty"`

	// The switch for an interface.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`

	// The network component for this interface.
	NetworkComponentId *int `json:"networkComponentId,omitempty" xmlrpc:"networkComponentId,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Version1_Usage data type contains general information relating to a single bandwidth usage record.
type Network_Bandwidth_Version1_Usage struct {
	Entity

	// Bandwidth allotment detail for this hardware.
	BandwidthAllotmentDetail *Network_Bandwidth_Version1_Allotment_Detail `json:"bandwidthAllotmentDetail,omitempty" xmlrpc:"bandwidthAllotmentDetail,omitempty"`

	// Bandwidth usage details for this hardware.
	BandwidthUsageDetail []Network_Bandwidth_Version1_Usage_Detail `json:"bandwidthUsageDetail,omitempty" xmlrpc:"bandwidthUsageDetail,omitempty"`

	// A count of bandwidth usage details for this hardware.
	BandwidthUsageDetailCount *uint `json:"bandwidthUsageDetailCount,omitempty" xmlrpc:"bandwidthUsageDetailCount,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Version1_Usage_Detail data type contains specific information relating to bandwidth utilization at a specific point in time on a given network interface.
type Network_Bandwidth_Version1_Usage_Detail struct {
	Entity

	// Incoming bandwidth utilization .
	AmountIn *Float64 `json:"amountIn,omitempty" xmlrpc:"amountIn,omitempty"`

	// Outgoing bandwidth utilization .
	AmountOut *Float64 `json:"amountOut,omitempty" xmlrpc:"amountOut,omitempty"`

	// In and out bandwidth utilization for a specified time stamp.
	BandwidthUsage *Network_Bandwidth_Version1_Usage `json:"bandwidthUsage,omitempty" xmlrpc:"bandwidthUsage,omitempty"`

	// Describes this bandwidth utilization record as on the public or private network interface.
	BandwidthUsageDetailType *Network_Bandwidth_Version1_Usage_Detail_Type `json:"bandwidthUsageDetailType,omitempty" xmlrpc:"bandwidthUsageDetailType,omitempty"`

	// Day and time this bandwidth utilization event was recorded.
	Day *Time `json:"day,omitempty" xmlrpc:"day,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Usage_Detail data type contains specific information relating to bandwidth utilization at a specific point in time on a given network interface.
type Network_Bandwidth_Version1_Usage_Detail_Total struct {
	Entity

	// The account tied to this tracking object
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// Incoming bandwidth utilization.
	AmountIn *Float64 `json:"amountIn,omitempty" xmlrpc:"amountIn,omitempty"`

	// Outgoing bandwidth utilization.
	AmountOut *Float64 `json:"amountOut,omitempty" xmlrpc:"amountOut,omitempty"`

	// ID of the bandwidth usage detail type for this record.
	BandwidthUsageDetailTypeId *Float64 `json:"bandwidthUsageDetailTypeId,omitempty" xmlrpc:"bandwidthUsageDetailTypeId,omitempty"`

	// The tracking object this bandwidth usage record describes.
	TrackingObject *Metric_Tracking_Object `json:"trackingObject,omitempty" xmlrpc:"trackingObject,omitempty"`

	// In and out bandwidth utilization for a specified time stamp.
	Type *Network_Bandwidth_Version1_Usage_Detail_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// The SoftLayer_Network_Bandwidth_Version1_Usage_Detail_Type data type contains generic information relating to the types of bandwidth records available, currently just public and private.
type Network_Bandwidth_Version1_Usage_Detail_Type struct {
	Entity

	// Database key associated with this bandwidth detail type.
	Alias *string `json:"alias,omitempty" xmlrpc:"alias,omitempty"`
}

// Every piece of hardware running in SoftLayer's datacenters connected to the public, private, or management networks (where applicable) have a corresponding network component. These network components are modeled by the SoftLayer_Network_Component data type. These data types reflect the servers' local ethernet and remote management interfaces.
type Network_Component struct {
	Entity

	// Reboot/power (rebootDefault, rebootSoft, rebootHard, powerOn, powerOff and powerCycle) command currently executing by the server's remote management card.
	ActiveCommand *Hardware_Component_RemoteManagement_Command_Request `json:"activeCommand,omitempty" xmlrpc:"activeCommand,omitempty"`

	// The network component linking this object to a child device
	DownlinkComponent *Network_Component `json:"downlinkComponent,omitempty" xmlrpc:"downlinkComponent,omitempty"`

	// The duplex mode of a network component.
	DuplexMode *Network_Component_Duplex_Mode `json:"duplexMode,omitempty" xmlrpc:"duplexMode,omitempty"`

	// A network component's Duplex mode.
	DuplexModeId *string `json:"duplexModeId,omitempty" xmlrpc:"duplexModeId,omitempty"`

	// The hardware that a network component resides in.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The internal identifier of the hardware that a network component belongs to.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// no documentation yet
	HighAvailabilityFirewallFlag *bool `json:"highAvailabilityFirewallFlag,omitempty" xmlrpc:"highAvailabilityFirewallFlag,omitempty"`

	// A network component's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A hardware switch's interface to the bandwidth pod.
	Interface *Network_Bandwidth_Version1_Interface `json:"interface,omitempty" xmlrpc:"interface,omitempty"`

	// A count of the records of all IP addresses bound to a network component.
	IpAddressBindingCount *uint `json:"ipAddressBindingCount,omitempty" xmlrpc:"ipAddressBindingCount,omitempty"`

	// The records of all IP addresses bound to a network component.
	IpAddressBindings []Network_Component_IpAddress `json:"ipAddressBindings,omitempty" xmlrpc:"ipAddressBindings,omitempty"`

	// A count of
	IpAddressCount *uint `json:"ipAddressCount,omitempty" xmlrpc:"ipAddressCount,omitempty"`

	// no documentation yet
	IpAddresses []Network_Subnet_IpAddress `json:"ipAddresses,omitempty" xmlrpc:"ipAddresses,omitempty"`

	// The IP address of an IPMI-based management network component.
	IpmiIpAddress *string `json:"ipmiIpAddress,omitempty" xmlrpc:"ipmiIpAddress,omitempty"`

	// The MAC address of an IPMI-based management network component.
	IpmiMacAddress *string `json:"ipmiMacAddress,omitempty" xmlrpc:"ipmiMacAddress,omitempty"`

	// Last reboot/power (rebootDefault, rebootSoft, rebootHard, powerOn, powerOff and powerCycle) command issued to the server's remote management card.
	LastCommand *Hardware_Component_RemoteManagement_Command_Request `json:"lastCommand,omitempty" xmlrpc:"lastCommand,omitempty"`

	// A network component's unique MAC address. IPMI-based management network interfaces may not have a MAC address.
	MacAddress *string `json:"macAddress,omitempty" xmlrpc:"macAddress,omitempty"`

	// A network component's maximum allowed speed, measured in Mbit per second. ''maxSpeed'' is determined by the capabilities of the network interface and the port speed purchased on your SoftLayer server.
	MaxSpeed *int `json:"maxSpeed,omitempty" xmlrpc:"maxSpeed,omitempty"`

	// The metric tracking object for this network component.
	MetricTrackingObject *Metric_Tracking_Object `json:"metricTrackingObject,omitempty" xmlrpc:"metricTrackingObject,omitempty"`

	// The date a network component was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A network component's short name. For most servers this is the string "eth" for ethernet ports or "mgmt" for remote management ports. Use this in conjunction with the ''port'' property to identify a network component. For instance, the "eth0" interface on a server has the network component name "eth" and port 0.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The upstream network component firewall.
	NetworkComponentFirewall *Network_Component_Firewall `json:"networkComponentFirewall,omitempty" xmlrpc:"networkComponentFirewall,omitempty"`

	// A network component's associated group.
	NetworkComponentGroup *Network_Component_Group `json:"networkComponentGroup,omitempty" xmlrpc:"networkComponentGroup,omitempty"`

	// All network devices in SoftLayer's network hierarchy that this device is connected to.
	NetworkHardware []Hardware `json:"networkHardware,omitempty" xmlrpc:"networkHardware,omitempty"`

	// A count of all network devices in SoftLayer's network hierarchy that this device is connected to.
	NetworkHardwareCount *uint `json:"networkHardwareCount,omitempty" xmlrpc:"networkHardwareCount,omitempty"`

	// The VLAN that a network component's subnet is associated with.
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`

	// The unique internal id of the network VLAN that the port belongs to.
	NetworkVlanId *int `json:"networkVlanId,omitempty" xmlrpc:"networkVlanId,omitempty"`

	// A count of the VLANs that are trunked to this network component.
	NetworkVlanTrunkCount *uint `json:"networkVlanTrunkCount,omitempty" xmlrpc:"networkVlanTrunkCount,omitempty"`

	// The VLANs that are trunked to this network component.
	NetworkVlanTrunks []Network_Component_Network_Vlan_Trunk `json:"networkVlanTrunks,omitempty" xmlrpc:"networkVlanTrunks,omitempty"`

	// no documentation yet
	ParentModule *Hardware_Component `json:"parentModule,omitempty" xmlrpc:"parentModule,omitempty"`

	// A network component's port number. Most hardware has more than one network interface. The port property separates these interfaces. Use this in conjunction with the ''name'' property to identify a network component. For instance, the "eth0" interface on a server has the network component name "eth" and port 0.
	Port *int `json:"port,omitempty" xmlrpc:"port,omitempty"`

	// A network component's primary IP address. IPMI-based management network interfaces may not have an IP address.
	PrimaryIpAddress *string `json:"primaryIpAddress,omitempty" xmlrpc:"primaryIpAddress,omitempty"`

	// The primary IPv4 Address record for a network component.
	PrimaryIpAddressRecord *Network_Subnet_IpAddress `json:"primaryIpAddressRecord,omitempty" xmlrpc:"primaryIpAddressRecord,omitempty"`

	// A network component's subnet for its primary IP address
	PrimarySubnet *Network_Subnet `json:"primarySubnet,omitempty" xmlrpc:"primarySubnet,omitempty"`

	// The primary IPv6 Address record for a network component.
	PrimaryVersion6IpAddressRecord *Network_Subnet_IpAddress `json:"primaryVersion6IpAddressRecord,omitempty" xmlrpc:"primaryVersion6IpAddressRecord,omitempty"`

	// A count of the last five reboot/power (rebootDefault, rebootSoft, rebootHard, powerOn, powerOff and powerCycle) commands issued to the server's remote management card.
	RecentCommandCount *uint `json:"recentCommandCount,omitempty" xmlrpc:"recentCommandCount,omitempty"`

	// The last five reboot/power (rebootDefault, rebootSoft, rebootHard, powerOn, powerOff and powerCycle) commands issued to the server's remote management card.
	RecentCommands []Hardware_Component_RemoteManagement_Command_Request `json:"recentCommands,omitempty" xmlrpc:"recentCommands,omitempty"`

	// Indicates whether the network component is participating in a group of two or more components capable of being operationally redundant, if enabled.
	RedundancyCapableFlag *bool `json:"redundancyCapableFlag,omitempty" xmlrpc:"redundancyCapableFlag,omitempty"`

	// Indicates whether the network component is participating in a group of two or more components which is actively providing link redundancy.
	RedundancyEnabledFlag *bool `json:"redundancyEnabledFlag,omitempty" xmlrpc:"redundancyEnabledFlag,omitempty"`

	// A count of user(s) credentials to issue commands and/or interact with the server's remote management card.
	RemoteManagementUserCount *uint `json:"remoteManagementUserCount,omitempty" xmlrpc:"remoteManagementUserCount,omitempty"`

	// User(s) credentials to issue commands and/or interact with the server's remote management card.
	RemoteManagementUsers []Hardware_Component_RemoteManagement_User `json:"remoteManagementUsers,omitempty" xmlrpc:"remoteManagementUsers,omitempty"`

	// A network component's routers.
	Router *Hardware `json:"router,omitempty" xmlrpc:"router,omitempty"`

	// A network component's speed, measured in Mbit per second.
	Speed *int `json:"speed,omitempty" xmlrpc:"speed,omitempty"`

	// A network component's status. This can take one of four possible values: "ACTIVE", "DISABLE", "USER_OFF", or "MACWAIT". "ACTIVE" network components are enabled and in use on a servers. "DISABLE" status components have been administratively disabled by SoftLayer accounting or abuse. "USER_OFF" components have been administratively disabled by you, the user. "MACWAIT" components only exist on network components that have not been provisioned. You should never see a network interface in MACWAIT state. If you happen to see one please contact SoftLayer support.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// Whether a network component's primary ip address is from a storage network subnet or not.
	StorageNetworkFlag *bool `json:"storageNetworkFlag,omitempty" xmlrpc:"storageNetworkFlag,omitempty"`

	// A count of a network component's subnets. A subnet is a group of IP addresses
	SubnetCount *uint `json:"subnetCount,omitempty" xmlrpc:"subnetCount,omitempty"`

	// A network component's subnets. A subnet is a group of IP addresses
	Subnets []Network_Subnet `json:"subnets,omitempty" xmlrpc:"subnets,omitempty"`

	// The network component linking this object to parent
	UplinkComponent *Network_Component `json:"uplinkComponent,omitempty" xmlrpc:"uplinkComponent,omitempty"`

	// The duplex mode of the uplink network component linking to this object
	UplinkDuplexMode *Network_Component_Duplex_Mode `json:"uplinkDuplexMode,omitempty" xmlrpc:"uplinkDuplexMode,omitempty"`
}

// Duplex Mode allows finer grained control over networking options and settings.
type Network_Component_Duplex_Mode struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Network_Component_Firewall data type contains general information relating to a single SoftLayer network component firewall. This is the object which ties the running rules to a specific downstream server. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Component_Firewall struct {
	Entity

	// A count of the additional subnets linked to this network component firewall, that inherit rules from the host that the context slot is attached to.
	ApplyServerRuleSubnetCount *uint `json:"applyServerRuleSubnetCount,omitempty" xmlrpc:"applyServerRuleSubnetCount,omitempty"`

	// The additional subnets linked to this network component firewall, that inherit rules from the host that the context slot is attached to.
	ApplyServerRuleSubnets []Network_Subnet `json:"applyServerRuleSubnets,omitempty" xmlrpc:"applyServerRuleSubnets,omitempty"`

	// The billing item for a Hardware Firewall (Dedicated).
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The network component of the guest virtual server that this network component firewall belongs to.
	GuestNetworkComponent *Virtual_Guest_Network_Component `json:"guestNetworkComponent,omitempty" xmlrpc:"guestNetworkComponent,omitempty"`

	// Unique ID for the network component of the switch interface that this network component firewall is attached to.
	GuestNetworkComponentId *int `json:"guestNetworkComponentId,omitempty" xmlrpc:"guestNetworkComponentId,omitempty"`

	// Unique ID for the network component firewall.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The network component of the switch interface that this network component firewall belongs to.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`

	// Unique ID for the network component of the switch interface that this network component firewall is attached to.
	NetworkComponentId *int `json:"networkComponentId,omitempty" xmlrpc:"networkComponentId,omitempty"`

	// The update requests made for this firewall.
	NetworkFirewallUpdateRequest []Network_Firewall_Update_Request `json:"networkFirewallUpdateRequest,omitempty" xmlrpc:"networkFirewallUpdateRequest,omitempty"`

	// A count of the update requests made for this firewall.
	NetworkFirewallUpdateRequestCount *uint `json:"networkFirewallUpdateRequestCount,omitempty" xmlrpc:"networkFirewallUpdateRequestCount,omitempty"`

	// A count of the currently running rule set of this network component firewall.
	RuleCount *uint `json:"ruleCount,omitempty" xmlrpc:"ruleCount,omitempty"`

	// The currently running rule set of this network component firewall.
	Rules []Network_Component_Firewall_Rule `json:"rules,omitempty" xmlrpc:"rules,omitempty"`

	// Current status of the network component firewall.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// A count of the additional subnets linked to this network component firewall.
	SubnetCount *uint `json:"subnetCount,omitempty" xmlrpc:"subnetCount,omitempty"`

	// The additional subnets linked to this network component firewall.
	Subnets []Network_Subnet `json:"subnets,omitempty" xmlrpc:"subnets,omitempty"`
}

// A SoftLayer_Network_Component_Firewall_Rule object type represents a currently running firewall rule and contains relative information. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates.
type Network_Component_Firewall_Rule struct {
	Entity

	// The action that the rule is to take [permit or deny].
	Action *string `json:"action,omitempty" xmlrpc:"action,omitempty"`

	// The destination IP address considered for determining rule application.
	DestinationIpAddress *string `json:"destinationIpAddress,omitempty" xmlrpc:"destinationIpAddress,omitempty"`

	// The CIDR is used for determining rule application. This value will
	DestinationIpCidr *int `json:"destinationIpCidr,omitempty" xmlrpc:"destinationIpCidr,omitempty"`

	// The destination IP subnet mask considered for determining rule application.
	DestinationIpSubnetMask *string `json:"destinationIpSubnetMask,omitempty" xmlrpc:"destinationIpSubnetMask,omitempty"`

	// The ending (upper end of range) destination port considered for determining rule application.
	DestinationPortRangeEnd *int `json:"destinationPortRangeEnd,omitempty" xmlrpc:"destinationPortRangeEnd,omitempty"`

	// The starting (lower end of range) destination port considered for determining rule application.
	DestinationPortRangeStart *int `json:"destinationPortRangeStart,omitempty" xmlrpc:"destinationPortRangeStart,omitempty"`

	// The rule's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The network component firewall that this rule belongs to.
	NetworkComponentFirewall *Network_Component_Firewall `json:"networkComponentFirewall,omitempty" xmlrpc:"networkComponentFirewall,omitempty"`

	// The notes field for the rule.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The numeric value describing the order in which the rule should be applied.
	OrderValue *int `json:"orderValue,omitempty" xmlrpc:"orderValue,omitempty"`

	// The protocol considered for determining rule application.
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`

	// The source IP address considered for determining rule application.
	SourceIpAddress *string `json:"sourceIpAddress,omitempty" xmlrpc:"sourceIpAddress,omitempty"`

	// The CIDR is used for determining rule application. This value will
	SourceIpCidr *int `json:"sourceIpCidr,omitempty" xmlrpc:"sourceIpCidr,omitempty"`

	// The source IP subnet mask considered for determining rule application.
	SourceIpSubnetMask *string `json:"sourceIpSubnetMask,omitempty" xmlrpc:"sourceIpSubnetMask,omitempty"`

	// Current status of the network component firewall.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// Whether this rule is an IPv4 rule or an IPv6 rule. If
	Version *int `json:"version,omitempty" xmlrpc:"version,omitempty"`
}

// A SoftLayer_Network_Component_Firewall_Subnets object type represents the current linked subnets and contains relative information. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates.
type Network_Component_Firewall_Subnets struct {
	Entity

	// A boolean flag that indicates whether the subnet should receive all the rules intended for the host on this context slot.
	ApplyServerRulesFlag *bool `json:"applyServerRulesFlag,omitempty" xmlrpc:"applyServerRulesFlag,omitempty"`

	// The network component firewall that write rules for this subnet.
	NetworkComponentFirewall *Network_Component_Firewall `json:"networkComponentFirewall,omitempty" xmlrpc:"networkComponentFirewall,omitempty"`

	// The subnet that this link binds to the network component firewall.
	Subnet *Network_Subnet `json:"subnet,omitempty" xmlrpc:"subnet,omitempty"`

	// The unique identifier of the subnet being linked to the network component firewall.
	SubnetId *int `json:"subnetId,omitempty" xmlrpc:"subnetId,omitempty"`
}

// no documentation yet
type Network_Component_Group struct {
	Entity

	// no documentation yet
	GroupTypeId *int `json:"groupTypeId,omitempty" xmlrpc:"groupTypeId,omitempty"`

	// A count of a network component group's associated network components.
	NetworkComponentCount *uint `json:"networkComponentCount,omitempty" xmlrpc:"networkComponentCount,omitempty"`

	// A network component group's associated network components.
	NetworkComponents []Network_Component `json:"networkComponents,omitempty" xmlrpc:"networkComponents,omitempty"`
}

// The SoftLayer_Network_Component_IpAddress data type contains general information relating to the binding of a single network component to a single SoftLayer IP address.
type Network_Component_IpAddress struct {
	Entity

	// The IP address associated with this object's network component.
	IpAddress *Network_Subnet_IpAddress `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// The network component associated with this object's IP address.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`
}

// Represents the association between a Network_Component and Network_Vlan in the manner of a 'trunk'. Trunking a VLAN to a port allows that ports to receive and send packets tagged with the corresponding VLAN number.
type Network_Component_Network_Vlan_Trunk struct {
	Entity

	// The network component that the VLAN is being trunked to.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`

	// The network component's identifier.
	NetworkComponentId *int `json:"networkComponentId,omitempty" xmlrpc:"networkComponentId,omitempty"`

	// The VLAN that is being trunked to the network component.
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`

	// The identifier of the network VLAN that is a trunk on the network component.
	NetworkVlanId *int `json:"networkVlanId,omitempty" xmlrpc:"networkVlanId,omitempty"`
}

// The SoftLayer_Network_Component_RemoteManagement data type contains general information relating to a single SoftLayer remote management network component.
type Network_Component_RemoteManagement struct {
	Network_Component
}

// The SoftLayer_Network_Component_Uplink_Hardware data type abstracts information related to network connections between SoftLayer hardware and SoftLayer network components.
//
// It is populated via triggers on the network_connection table (SoftLayer_Network_Connection), so you shouldn't have to delete or insert records into this table, ever.
//
//
type Network_Component_Uplink_Hardware struct {
	Entity

	// A network component uplink's connected [[SoftLayer_Hardware|Hardware]].
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The [[SoftLayer_Network_Component|Network Component]] that a uplink connection belongs to..
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`
}

// The SoftLayer_Network_ContentDelivery_Account data type models an individual CDN account. CDN accounts contain references to the SoftLayer customer account they belong to, login credentials for upload services, and a CDN account's status. Please contact SoftLayer sales to purchase or cancel a CDN account
type Network_ContentDelivery_Account struct {
	Entity

	// The customer account that a CDN account belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The internal identifier of the customer account that a CDN account belongs to.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The CDN account id that this CDN account is associated with.
	AssociatedCdnAccountId *string `json:"associatedCdnAccountId,omitempty" xmlrpc:"associatedCdnAccountId,omitempty"`

	// A count of the IP addresses that are used for the content authentication service.
	AuthenticationIpAddressCount *uint `json:"authenticationIpAddressCount,omitempty" xmlrpc:"authenticationIpAddressCount,omitempty"`

	// The IP addresses that are used for the content authentication service.
	AuthenticationIpAddresses []Network_ContentDelivery_Authentication_Address `json:"authenticationIpAddresses,omitempty" xmlrpc:"authenticationIpAddresses,omitempty"`

	// The current billing item for a CDN account.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The name of a CDN account.
	CdnAccountName *string `json:"cdnAccountName,omitempty" xmlrpc:"cdnAccountName,omitempty"`

	// A brief note on a CDN account.
	CdnAccountNote *string `json:"cdnAccountNote,omitempty" xmlrpc:"cdnAccountNote,omitempty"`

	// The solution type of a CDN account.
	CdnSolutionName *string `json:"cdnSolutionName,omitempty" xmlrpc:"cdnSolutionName,omitempty"`

	// The date that a CDN account was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Indicates if CDN account is dependent on other service. If set, this CDN account is limited to these services: createOriginPullMapping, deleteOriginPullRule, getOriginPullMappingInformation, getCdnUrls, purgeCache, loadContent, manageHttpCompression
	DependantServiceFlag *bool `json:"dependantServiceFlag,omitempty" xmlrpc:"dependantServiceFlag,omitempty"`

	// A CDN account's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Indicates if it is a legacy CDN or not
	LegacyCdnFlag *bool `json:"legacyCdnFlag,omitempty" xmlrpc:"legacyCdnFlag,omitempty"`

	// Indicates if CDN logging is enabled.
	LogEnabledFlag *string `json:"logEnabledFlag,omitempty" xmlrpc:"logEnabledFlag,omitempty"`

	// Indicates if customer is allowed to access the CDN provider's management portal.
	ProviderPortalAccessFlag *bool `json:"providerPortalAccessFlag,omitempty" xmlrpc:"providerPortalAccessFlag,omitempty"`

	// A CDN account's status presented in a more detailed data type.
	Status *Network_ContentDelivery_Account_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The internal identifier of a CDN status
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// Indicates if the token authentication service is enabled or not.
	TokenAuthenticationEnabledFlag *bool `json:"tokenAuthenticationEnabledFlag,omitempty" xmlrpc:"tokenAuthenticationEnabledFlag,omitempty"`
}

// The SoftLayer_Network_ContentDelivery_Account_Status contains information on a CDN account.
type Network_ContentDelivery_Account_Status struct {
	Entity

	// A longer description of a CDN account's status.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A CDN account status' internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A CDN account status' name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Network_ContentDelivery_Authentication_Address data type models an individual IP address that CDN allow or deny access from.
type Network_ContentDelivery_Authentication_Address struct {
	Entity

	// The type of access on an IP address.  It can be "ALLOW" or "DENY"
	AccessType *string `json:"accessType,omitempty" xmlrpc:"accessType,omitempty"`

	// The internal identifier of the CDN account
	CdnAccountId *int `json:"cdnAccountId,omitempty" xmlrpc:"cdnAccountId,omitempty"`

	// The created date
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The internal identifier of an authentication IP address
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The IP address that you want to block or allow access to
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// The last modified date
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of an authentication IP.  This helps you to keep track of IP addresses.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The priority of an authentication IP address.  The smaller number, the higher in priority.  Higher priority IP will be matched first.
	Priority *int `json:"priority,omitempty" xmlrpc:"priority,omitempty"`

	// The internal identifier of the user who created an authentication IP record
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// The SoftLayer_Network_ContentDelivery_Authentication_Address data type models an individual IP address that CDN allow or deny access from.
type Network_ContentDelivery_Authentication_Token struct {
	Entity

	// The internal identifier of a CDN account
	CdnAccountId *int `json:"cdnAccountId,omitempty" xmlrpc:"cdnAccountId,omitempty"`

	// The client IP address. This is optional.
	ClientIp *string `json:"clientIp,omitempty" xmlrpc:"clientIp,omitempty"`

	// The created date
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The customer id.  You can use this optional value to tie a user id to an authentication token.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The referrer information.  This is optional.
	Referrer *string `json:"referrer,omitempty" xmlrpc:"referrer,omitempty"`

	// The managed token string
	Token *string `json:"token,omitempty" xmlrpc:"token,omitempty"`
}

// The SoftLayer_Network_Customer_Subnet data type contains general information relating to a single customer subnet (remote).
type Network_Customer_Subnet struct {
	Entity

	// The account id a customer subnet belongs to.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A subnet's Classless Inter-Domain Routing prefix. This is a number between 0 and 32 signifying the number of bits in a subnet's netmask. These bits separate a subnet's network address from it's host addresses. It performs the same function as the ''netmask'' property, but is represented as an integer.
	Cidr *int `json:"cidr,omitempty" xmlrpc:"cidr,omitempty"`

	// A customer subnet's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of all ip addresses associated with a subnet.
	IpAddressCount *uint `json:"ipAddressCount,omitempty" xmlrpc:"ipAddressCount,omitempty"`

	// All ip addresses associated with a subnet.
	IpAddresses []Network_Customer_Subnet_IpAddress `json:"ipAddresses,omitempty" xmlrpc:"ipAddresses,omitempty"`

	// A bitmask in dotted-quad format that is used to separate a subnet's network address from it's host addresses. This performs the same function as the ''cidr'' property, but is expressed in a string format.
	Netmask *string `json:"netmask,omitempty" xmlrpc:"netmask,omitempty"`

	// A subnet's network identifier. This is the first IP address of a subnet.
	NetworkIdentifier *string `json:"networkIdentifier,omitempty" xmlrpc:"networkIdentifier,omitempty"`

	// The total number of ip addresses in a subnet.
	TotalIpAddresses *int `json:"totalIpAddresses,omitempty" xmlrpc:"totalIpAddresses,omitempty"`
}

// The SoftLayer_Network_Customer_Subnet_IpAddress data type contains general information relating to a single Customer Subnet (Remote) IPv4 address.
type Network_Customer_Subnet_IpAddress struct {
	Entity

	// Unique identifier for an ip address.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// An IP address expressed in dotted quad format.
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// An IP address' user defined note.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The customer subnet (remote) that the ip address belongs to.
	Subnet *Network_Customer_Subnet `json:"subnet,omitempty" xmlrpc:"subnet,omitempty"`

	// The unique identifier for the customer subnet (remote) the ip address belongs to.
	SubnetId *int `json:"subnetId,omitempty" xmlrpc:"subnetId,omitempty"`

	// A count of all the address translations that are tied to an IP address.
	TranslationCount *uint `json:"translationCount,omitempty" xmlrpc:"translationCount,omitempty"`

	// All the address translations that are tied to an IP address.
	Translations []Network_Tunnel_Module_Context_Address_Translation `json:"translations,omitempty" xmlrpc:"translations,omitempty"`
}

// The SoftLayer_Network_Firewall_AccessControlList data type contains general information relating to a single SoftLayer firewall access to controll list. This is the object which ties the running rules to a specific context. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Firewall_AccessControlList struct {
	Entity

	// no documentation yet
	Direction *string `json:"direction,omitempty" xmlrpc:"direction,omitempty"`

	// no documentation yet
	FirewallContextInterfaceId *int `json:"firewallContextInterfaceId,omitempty" xmlrpc:"firewallContextInterfaceId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of the update requests made for this firewall.
	NetworkFirewallUpdateRequestCount *uint `json:"networkFirewallUpdateRequestCount,omitempty" xmlrpc:"networkFirewallUpdateRequestCount,omitempty"`

	// The update requests made for this firewall.
	NetworkFirewallUpdateRequests []Network_Firewall_Update_Request `json:"networkFirewallUpdateRequests,omitempty" xmlrpc:"networkFirewallUpdateRequests,omitempty"`

	// no documentation yet
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`

	// A count of the currently running rule set of this context access control list firewall.
	RuleCount *uint `json:"ruleCount,omitempty" xmlrpc:"ruleCount,omitempty"`

	// The currently running rule set of this context access control list firewall.
	Rules []Network_Vlan_Firewall_Rule `json:"rules,omitempty" xmlrpc:"rules,omitempty"`
}

// The SoftLayer_Network_Firewall_Interface data type contains general information relating to a single SoftLayer firewall interface. This is the object which ties the firewall context access control list to a firewall. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Firewall_Interface struct {
	Network_Firewall_Module_Context_Interface
}

// no documentation yet
type Network_Firewall_Module_Context_Interface struct {
	Entity

	// A count of
	FirewallContextAccessControlListCount *uint `json:"firewallContextAccessControlListCount,omitempty" xmlrpc:"firewallContextAccessControlListCount,omitempty"`

	// no documentation yet
	FirewallContextAccessControlLists []Network_Firewall_AccessControlList `json:"firewallContextAccessControlLists,omitempty" xmlrpc:"firewallContextAccessControlLists,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`
}

// The SoftLayer_Network_Firewall_Template type contains general information for a SoftLayer network firewall template.
//
// Firewall templates are recommend rule sets for use with SoftLayer Hardware Firewall (Dedicated).  These optimized templates are designed to balance security restriction with application availability.  The templates given may be altered to provide custom network security, or may be used as-is for basic security. At least one rule set MUST be applied for the firewall to block traffic. Use the [[SoftLayer Network Component Firewall]] service to view current rules. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Firewall_Template struct {
	Entity

	// A Firewall template's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The name of the firewall rules template.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of the rule set that belongs to this firewall rules template.
	RuleCount *uint `json:"ruleCount,omitempty" xmlrpc:"ruleCount,omitempty"`

	// The rule set that belongs to this firewall rules template.
	Rules []Network_Firewall_Template_Rule `json:"rules,omitempty" xmlrpc:"rules,omitempty"`
}

// The SoftLayer_Network_Component_Firewall_Rule type contains general information relating to a single SoftLayer firewall template rule. Use the [[SoftLayer Network Component Firewall]] service to view current rules. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Firewall_Template_Rule struct {
	Entity

	// The action that this template rule is to take [permit or deny].
	Action *string `json:"action,omitempty" xmlrpc:"action,omitempty"`

	// The destination IP address considered for determining rule application.
	DestinationIpAddress *string `json:"destinationIpAddress,omitempty" xmlrpc:"destinationIpAddress,omitempty"`

	// The destination IP subnet mask considered for determining rule application.
	DestinationIpSubnetMask *string `json:"destinationIpSubnetMask,omitempty" xmlrpc:"destinationIpSubnetMask,omitempty"`

	// The ending (upper end of range) destination port considered for determining rule application.
	DestinationPortRangeEnd *int `json:"destinationPortRangeEnd,omitempty" xmlrpc:"destinationPortRangeEnd,omitempty"`

	// The starting (lower end of range) destination port considered for determining rule application.
	DestinationPortRangeStart *int `json:"destinationPortRangeStart,omitempty" xmlrpc:"destinationPortRangeStart,omitempty"`

	// The firewall template that this rule is attached to.
	FirewallTemplate *Network_Firewall_Template `json:"firewallTemplate,omitempty" xmlrpc:"firewallTemplate,omitempty"`

	// The unique identifier of the firewall template that a firewall template rule is associated with.
	FirewallTemplateId *int `json:"firewallTemplateId,omitempty" xmlrpc:"firewallTemplateId,omitempty"`

	// A Firewall template rule's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The notes field for the firewall template rule.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The numeric value describing the order in which the rule set should be applied.
	OrderValue *int `json:"orderValue,omitempty" xmlrpc:"orderValue,omitempty"`

	// The protocol considered for determining rule application.
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`

	// The source IP address considered for determining rule application.
	SourceIpAddress *string `json:"sourceIpAddress,omitempty" xmlrpc:"sourceIpAddress,omitempty"`

	// The source IP subnet mask considered for determining rule application.
	SourceIpSubnetMask *string `json:"sourceIpSubnetMask,omitempty" xmlrpc:"sourceIpSubnetMask,omitempty"`
}

// The SoftLayer_Network_Firewall_Update_Request data type contains information relating to a SoftLayer network firewall update request. Use the [[SoftLayer Network Component Firewall]] service to view current rules. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates.
type Network_Firewall_Update_Request struct {
	Entity

	// Timestamp of when the rules from the update request were applied to the firewall.
	ApplyDate *Time `json:"applyDate,omitempty" xmlrpc:"applyDate,omitempty"`

	// The user that authorized this firewall update request.
	AuthorizingUser *User_Interface `json:"authorizingUser,omitempty" xmlrpc:"authorizingUser,omitempty"`

	// The unique identifier of the user that authorized the update request.
	AuthorizingUserId *int `json:"authorizingUserId,omitempty" xmlrpc:"authorizingUserId,omitempty"`

	// The type of user that authorized the update request [EMP or USR].
	AuthorizingUserType *string `json:"authorizingUserType,omitempty" xmlrpc:"authorizingUserType,omitempty"`

	// Flag indicating whether the request is for a rule bypass configuration [0 or 1].
	BypassFlag *bool `json:"bypassFlag,omitempty" xmlrpc:"bypassFlag,omitempty"`

	// Timestamp of the creation of the record.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The unique identifier of the firewall access control list that the rule set is destined for.
	FirewallContextAccessControlListId *int `json:"firewallContextAccessControlListId,omitempty" xmlrpc:"firewallContextAccessControlListId,omitempty"`

	// The downstream virtual server that the rule set will be applied to.
	Guest *Virtual_Guest `json:"guest,omitempty" xmlrpc:"guest,omitempty"`

	// The downstream server that the rule set will be applied to.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The unique identifier of the server that the rule set is destined to protect.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// The unique identifier of the firewall update request.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The network component firewall that the rule set will be applied to.
	NetworkComponentFirewall *Network_Component_Firewall `json:"networkComponentFirewall,omitempty" xmlrpc:"networkComponentFirewall,omitempty"`

	// The unique identifier of the network component firewall that the rule set is destined for.
	NetworkComponentFirewallId *int `json:"networkComponentFirewallId,omitempty" xmlrpc:"networkComponentFirewallId,omitempty"`

	// A count of the group of rules contained within the update request.
	RuleCount *uint `json:"ruleCount,omitempty" xmlrpc:"ruleCount,omitempty"`

	// The group of rules contained within the update request.
	Rules []Network_Firewall_Update_Request_Rule `json:"rules,omitempty" xmlrpc:"rules,omitempty"`
}

// A SoftLayer_Ticket_Update_Customer is a single update made by a customer to a ticket.
type Network_Firewall_Update_Request_Customer struct {
	Network_Firewall_Update_Request
}

// The SoftLayer_Network_Firewall_Update_Request_Employee data type returns a user object for the SoftLayer employee that created the request.
type Network_Firewall_Update_Request_Employee struct {
	Network_Firewall_Update_Request
}

// The SoftLayer_Network_Firewall_Update_Request_Rule type contains information relating to a SoftLayer network firewall update request rule. This rule is a member of a [[SoftLayer Network Firewall Update Request]]. Use the [[SoftLayer Network Component Firewall]] service to view current rules. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates.
type Network_Firewall_Update_Request_Rule struct {
	Entity

	// The action that this update request rule is to take [permit or deny].
	Action *string `json:"action,omitempty" xmlrpc:"action,omitempty"`

	// The destination IP address considered for determining rule application.
	DestinationIpAddress *string `json:"destinationIpAddress,omitempty" xmlrpc:"destinationIpAddress,omitempty"`

	// The CIDR is used for determining rule application. This value will
	DestinationIpCidr *int `json:"destinationIpCidr,omitempty" xmlrpc:"destinationIpCidr,omitempty"`

	// The destination IP subnet mask considered for determining rule application.
	DestinationIpSubnetMask *string `json:"destinationIpSubnetMask,omitempty" xmlrpc:"destinationIpSubnetMask,omitempty"`

	// The ending (upper end of range) destination port considered for determining rule application.
	DestinationPortRangeEnd *int `json:"destinationPortRangeEnd,omitempty" xmlrpc:"destinationPortRangeEnd,omitempty"`

	// The starting (lower end of range) destination port considered for determining rule application.
	DestinationPortRangeStart *int `json:"destinationPortRangeStart,omitempty" xmlrpc:"destinationPortRangeStart,omitempty"`

	// The update request that this rule belongs to.
	FirewallUpdateRequest *Network_Firewall_Update_Request `json:"firewallUpdateRequest,omitempty" xmlrpc:"firewallUpdateRequest,omitempty"`

	// The unique identifier of the firewall update request that a firewall update request rule is associated with.
	FirewallUpdateRequestId *int `json:"firewallUpdateRequestId,omitempty" xmlrpc:"firewallUpdateRequestId,omitempty"`

	// A Firewall update request rule's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The notes field for the firewall update request rule.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The numeric value describing the order in which the rule should be applied.
	OrderValue *int `json:"orderValue,omitempty" xmlrpc:"orderValue,omitempty"`

	// The protocol considered for determining rule application.
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`

	// The source IP address considered for determining rule application.
	SourceIpAddress *string `json:"sourceIpAddress,omitempty" xmlrpc:"sourceIpAddress,omitempty"`

	// The CIDR is used for determining rule application. This value will
	SourceIpCidr *int `json:"sourceIpCidr,omitempty" xmlrpc:"sourceIpCidr,omitempty"`

	// The source IP subnet mask considered for determining rule application.
	SourceIpSubnetMask *string `json:"sourceIpSubnetMask,omitempty" xmlrpc:"sourceIpSubnetMask,omitempty"`

	// Whether this rule is an IPv4 rule or an IPv6 rule. If
	Version *int `json:"version,omitempty" xmlrpc:"version,omitempty"`
}

// The SoftLayer_Network_Firewall_Update_Request_Rule_Version6 type contains information relating to a SoftLayer network firewall update request rule for IPv6. This rule is a member of a [[SoftLayer Network Firewall Update Request]]. Use the [[SoftLayer Network Component Firewall]] service to view current rules. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates.
type Network_Firewall_Update_Request_Rule_Version6 struct {
	Network_Firewall_Update_Request_Rule
}

// no documentation yet
type Network_Gateway struct {
	Entity

	// The account for this gateway.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The internal identifier of the account assigned to this gateway.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The VRRP group number for this gateway. This is set internally and cannot be provided on create.
	GroupNumber *int `json:"groupNumber,omitempty" xmlrpc:"groupNumber,omitempty"`

	// A gateway's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of all VLANs trunked to this gateway.
	InsideVlanCount *uint `json:"insideVlanCount,omitempty" xmlrpc:"insideVlanCount,omitempty"`

	// All VLANs trunked to this gateway.
	InsideVlans []Network_Gateway_Vlan `json:"insideVlans,omitempty" xmlrpc:"insideVlans,omitempty"`

	// A count of the members for this gateway.
	MemberCount *uint `json:"memberCount,omitempty" xmlrpc:"memberCount,omitempty"`

	// The members for this gateway.
	Members []Network_Gateway_Member `json:"members,omitempty" xmlrpc:"members,omitempty"`

	// A gateway's name. This is required on create and can be no more than 255 characters.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A gateway's network space. Currently, only 'private'  or 'both' is allowed. When this value is 'private', it is a backend gateway only. Otherwise, it is a gateway for both frontend and backend traffic.
	NetworkSpace *string `json:"networkSpace,omitempty" xmlrpc:"networkSpace,omitempty"`

	// The private gateway IP address.
	PrivateIpAddress *Network_Subnet_IpAddress `json:"privateIpAddress,omitempty" xmlrpc:"privateIpAddress,omitempty"`

	// The internal identifier of the private IP address for this gateway.
	PrivateIpAddressId *int `json:"privateIpAddressId,omitempty" xmlrpc:"privateIpAddressId,omitempty"`

	// The private VLAN for accessing this gateway.
	PrivateVlan *Network_Vlan `json:"privateVlan,omitempty" xmlrpc:"privateVlan,omitempty"`

	// The internal identifier of the private VLAN for this gateway.
	PrivateVlanId *int `json:"privateVlanId,omitempty" xmlrpc:"privateVlanId,omitempty"`

	// The public gateway IP address.
	PublicIpAddress *Network_Subnet_IpAddress `json:"publicIpAddress,omitempty" xmlrpc:"publicIpAddress,omitempty"`

	// The internal identifier of the public IP address for this gateway.
	PublicIpAddressId *int `json:"publicIpAddressId,omitempty" xmlrpc:"publicIpAddressId,omitempty"`

	// The public gateway IPv6 address.
	PublicIpv6Address *Network_Subnet_IpAddress `json:"publicIpv6Address,omitempty" xmlrpc:"publicIpv6Address,omitempty"`

	// The internal identifier of the public IPv6 address for this gateway.
	PublicIpv6AddressId *int `json:"publicIpv6AddressId,omitempty" xmlrpc:"publicIpv6AddressId,omitempty"`

	// The public VLAN for accessing this gateway.
	PublicVlan *Network_Vlan `json:"publicVlan,omitempty" xmlrpc:"publicVlan,omitempty"`

	// The internal identifier of the public VLAN for this gateway. This is set internally and cannot be provided on create.
	PublicVlanId *int `json:"publicVlanId,omitempty" xmlrpc:"publicVlanId,omitempty"`

	// The current status of the gateway.
	Status *Network_Gateway_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The current status of this gateway. This is always active unless there is a process running to change the gateway. This can not be set on creation.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`
}

// no documentation yet
type Network_Gateway_Member struct {
	Entity

	// The device for this member.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The internal identifier of the hardware for this member.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// A gateway member's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The gateway this member belongs to.
	NetworkGateway *Network_Gateway `json:"networkGateway,omitempty" xmlrpc:"networkGateway,omitempty"`

	// The internal identifier of the gateway this member belongs to.
	NetworkGatewayId *int `json:"networkGatewayId,omitempty" xmlrpc:"networkGatewayId,omitempty"`

	// The priority for this gateway member. This is set internally and cannot be provided on create.
	Priority *int `json:"priority,omitempty" xmlrpc:"priority,omitempty"`
}

// no documentation yet
type Network_Gateway_Status struct {
	Entity

	// A gateway status's description.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A gateway status's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A gateway status's programmatic name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A gateway status's human-friendly name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Gateway_Vlan struct {
	Entity

	// If true, this VLAN is bypassed. If false, it is routed through the gateway.
	BypassFlag *bool `json:"bypassFlag,omitempty" xmlrpc:"bypassFlag,omitempty"`

	// A gateway VLAN's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The gateway this VLAN is attached to.
	NetworkGateway *Network_Gateway `json:"networkGateway,omitempty" xmlrpc:"networkGateway,omitempty"`

	// The internal identifier of the gateway this VLAN is attached to.
	NetworkGatewayId *int `json:"networkGatewayId,omitempty" xmlrpc:"networkGatewayId,omitempty"`

	// The network VLAN record.
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`

	// The internal identifier of the network VLAN.
	NetworkVlanId *int `json:"networkVlanId,omitempty" xmlrpc:"networkVlanId,omitempty"`
}

// The SoftLayer_Network_LoadBalancer_Global_Account data type contains the properties for a single global load balancer account.  The properties you are able to edit are fallbackIp, loadBalanceTypeId, and notes. The hosts relational property can be used for creating and editing hosts that belong to the global load balancer account.  The [[SoftLayer_Network_LoadBalancer_Global_Account::editObject|editObject]] method contains details on creating and edited hosts through the hosts relational property.
type Network_LoadBalancer_Global_Account struct {
	Entity

	// Your SoftLayer customer account.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The maximum number of hosts that a global load balancer account is allowed to have.
	AllowedNumberOfHosts *int `json:"allowedNumberOfHosts,omitempty" xmlrpc:"allowedNumberOfHosts,omitempty"`

	// The average amount of connections per second used within the current billing cycle.  This number is updated daily.
	AverageConnectionsPerSecond *Float64 `json:"averageConnectionsPerSecond,omitempty" xmlrpc:"averageConnectionsPerSecond,omitempty"`

	// The current billing item for a Global Load Balancer account.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The amount of connections per second a global load balancer account may use within a billing cycle without being billed for an overage.
	ConnectionsPerSecond *int `json:"connectionsPerSecond,omitempty" xmlrpc:"connectionsPerSecond,omitempty"`

	// The IP address that will be return to a DNS request when none of the hosts for a global load balancer account could be returned.
	FallbackIp *string `json:"fallbackIp,omitempty" xmlrpc:"fallbackIp,omitempty"`

	// A count of the hosts in the load balancing pool for a global load balancer account.
	HostCount *uint `json:"hostCount,omitempty" xmlrpc:"hostCount,omitempty"`

	// The hostname of a global load balancer account that is being load balanced.
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// The hosts in the load balancing pool for a global load balancer account.
	Hosts []Network_LoadBalancer_Global_Host `json:"hosts,omitempty" xmlrpc:"hosts,omitempty"`

	// The unique identifier of a global load balancer account.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The load balance method of a global load balancer account
	LoadBalanceType *Network_LoadBalancer_Global_Type `json:"loadBalanceType,omitempty" xmlrpc:"loadBalanceType,omitempty"`

	// The identifier of the load balance method for a global load balancer account.
	LoadBalanceTypeId *int `json:"loadBalanceTypeId,omitempty" xmlrpc:"loadBalanceTypeId,omitempty"`

	// A flag indicating that the global load balancer is a managed resource.
	ManagedResourceFlag *bool `json:"managedResourceFlag,omitempty" xmlrpc:"managedResourceFlag,omitempty"`

	// Additional customer defined information for a global load balancer account.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`
}

// The SoftLayer_Network_LoadBalancer_Global_Host data type represents a single host that belongs to a global load balancer account's load balancing pool.
//
// The destination IP address of a host must be one that belongs to your SoftLayer customer account, or to a datacenter load balancer virtual ip that belongs to your SoftLayer customer account.  The destination IP address and port of a global load balancer host is a required field and must exist during creation and can not be removed.  The acceptable values for the health check type are 'none', 'http', and 'tcp'. The status property is updated in 5 minute intervals and the hits property is updated in 10 minute intervals.
//
// The order of the host is only important if you are using the 'failover' load balance method, and the weight is only important if you are using the 'weighted round robin' load balance method.
type Network_LoadBalancer_Global_Host struct {
	Entity

	// The IP address of the host that will be returned by the global load balancers in response to a dns request.
	DestinationIp *string `json:"destinationIp,omitempty" xmlrpc:"destinationIp,omitempty"`

	// The port of the host that will be used for health checks.
	DestinationPort *int `json:"destinationPort,omitempty" xmlrpc:"destinationPort,omitempty"`

	// Whether the host is enabled or not.  The value can be '0' for disabled, or '1' for enabled.
	Enabled *int `json:"enabled,omitempty" xmlrpc:"enabled,omitempty"`

	// The health check type of a host.  Valid values include 'none', 'http', and 'tcp'.
	HealthCheck *string `json:"healthCheck,omitempty" xmlrpc:"healthCheck,omitempty"`

	// The number of times the host was selected by the load balance method.
	Hits *Float64 `json:"hits,omitempty" xmlrpc:"hits,omitempty"`

	// The unique identifier of a global load balancer host.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The order of this host within the load balance pool.  This is only significant if the load balance method is set to failover.
	LoadBalanceOrder *int `json:"loadBalanceOrder,omitempty" xmlrpc:"loadBalanceOrder,omitempty"`

	// The global load balancer account a host belongs to.
	LoadBalancerAccount *Network_LoadBalancer_Global_Account `json:"loadBalancerAccount,omitempty" xmlrpc:"loadBalancerAccount,omitempty"`

	// The location of a host in a datacenter.serverRoom format.
	Location *string `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// The health status of a host.  The status can be either 'UP', 'DOWN', or null which could mean that the health check type is set to 'none' or an update to the ip, port, or health check type was recently done and the host is waiting for the new status.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The load balance weight of a host.  The total weight of all hosts in the load balancing pool must not exceed 100.
	Weight *int `json:"weight,omitempty" xmlrpc:"weight,omitempty"`
}

// The SoftLayer_Network_LoadBalancer_Global_Type data type represents a single load balance method that can be assigned to a global load balancer account. The load balance method determines how hosts in a load balancing pool are chosen by the global load balancers.
type Network_LoadBalancer_Global_Type struct {
	Entity

	// The unique identifier of a load balance method.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The name of a load balance method.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Network_LoadBalancer_Service data type contains all the information relating to a specific service (destination) on a particular load balancer.
//
// Information retained on the object itself is the the source and destination of the service, routing type, weight, and whether or not the service is currently enabled.
type Network_LoadBalancer_Service struct {
	Entity

	// Connection limit on this service.
	ConnectionLimit *int `json:"connectionLimit,omitempty" xmlrpc:"connectionLimit,omitempty"`

	// Creation Date of this service
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The IP Address of the real server you wish to direct traffic to.  Your account must own this IP
	DestinationIpAddress *string `json:"destinationIpAddress,omitempty" xmlrpc:"destinationIpAddress,omitempty"`

	// The port on the real server to direct the traffic.  This can be different than the source port.  If you wish to obfuscate your HTTP traffic, you can accept requests on port 80 on the load balancer, then redirect them to port 932 on your real server.
	DestinationPort *int `json:"destinationPort,omitempty" xmlrpc:"destinationPort,omitempty"`

	// A flag (either true or false) that determines if this particular service should be enabled on the load balancer.  Set to false to bring the server out of rotation without losing your configuration
	Enabled *bool `json:"enabled,omitempty" xmlrpc:"enabled,omitempty"`

	// The health check type for this service.  If one is supplied, the load balancer will occasionally ping your server to determine if it is still up.  Servers that are down are removed from the queue and will not be used to handle requests until their status returns to "up".  The value of the health check is determined directly by what option you have selected for the routing type.
	//
	// {|
	// |-
	// ! Type
	// ! Valid Health Checks
	// |-
	// | HTTP
	// | HTTP, TCP, ICMP
	// |-
	// | TCP
	// | HTTP, TCP, ICMP
	// |-
	// | FTP
	// | TCP, ICMP
	// |-
	// | DNS
	// | DNS, ICMP
	// |-
	// | UDP
	// | None
	// |}
	//
	//
	HealthCheck *string `json:"healthCheck,omitempty" xmlrpc:"healthCheck,omitempty"`

	// The URL provided here (starting with /) is what the load balancer will request in order to perform a custom HTTP health check.  You must specify either "GET /location/of/file.html" or "HEAD /location/of/file.php"
	HealthCheckURL *string `json:"healthCheckURL,omitempty" xmlrpc:"healthCheckURL,omitempty"`

	// The expected response from the custom HTTP health check.  If the requested page contains this response, the check succeeds.
	HealthResponse *string `json:"healthResponse,omitempty" xmlrpc:"healthResponse,omitempty"`

	// Unique ID for this object, used for the getObject method, and must be set if you are editing this object.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Last modification date of this service
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Name of the load balancer service
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// Holds whether this server is up or down.  Does not affect load balancer configuration at all, just for the customer's informational purposes
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// Peak historical connections since the creation of this service.  Is reset any time you make a configuration change
	PeakConnections *int `json:"peakConnections,omitempty" xmlrpc:"peakConnections,omitempty"`

	// The port on the load balancer that this service maps to.  This is the port for incoming traffic, it needs to be shared with other services to form a group.
	SourcePort *int `json:"sourcePort,omitempty" xmlrpc:"sourcePort,omitempty"`

	// The connection type of this service.  Valid values are HTTP, FTP, TCP, UDP, and DNS.  The value of this variable affects available values of healthCheck, listed in that variable's description
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The load balancer that this service belongs to.
	Vip *Network_LoadBalancer_VirtualIpAddress `json:"vip,omitempty" xmlrpc:"vip,omitempty"`

	// Unique ID for this object's parent.  Probably not useful in the API, as this object will always be a child of a VirtualIpAddress anyway.
	VipId *int `json:"vipId,omitempty" xmlrpc:"vipId,omitempty"`

	// Weight affects the choices the load balancer makes between your services.  The weight of each service is expressed as a percentage of the TOTAL CONNECTION LIMIT on the virtual IP Address.  All services draw from the same pool of connections, so if you expect to have 4 times as much HTTP traffic as HTTPS, your weights for the above example routes would be 40%, 40%, 10%, 10% respectively.  The weights should add up to 100%  If you go over 100%, an exception will be thrown.  Weights must be whole numbers, no fractions or decimals are accepted.
	Weight *int `json:"weight,omitempty" xmlrpc:"weight,omitempty"`
}

// The SoftLayer_Network_LoadBalancer_VirtualIpAddress data type contains all the information relating to a specific load balancer assigned to a customer account.
//
// Information retained on the object itself is the virtual IP address, load balancing method, and any notes that are related to the load balancer.  There is also an array of SoftLayer_Network_LoadBalancer_Service objects, which represent the load balancer services, explained more fully in the SoftLayer_Network_LoadBalancer_Service documentation.
type Network_LoadBalancer_VirtualIpAddress struct {
	Entity

	// The account that owns this load balancer.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The current billing item for the Load Balancer.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// Connection limit on this VIP.  Can be upgraded through the upgradeConnectionLimit() function
	ConnectionLimit *int `json:"connectionLimit,omitempty" xmlrpc:"connectionLimit,omitempty"`

	// If false, this VIP and associated services may be edited via the portal or the API. If true, you must configure this VIP manually on the device.
	CustomerManagedFlag *int `json:"customerManagedFlag,omitempty" xmlrpc:"customerManagedFlag,omitempty"`

	// Unique ID for this object, used for the getObject method, and must be set if you are editing this object.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The load balancing method that determines which server is used "next" by the load balancer.  The method is stored in an abbreviated form, represented in parentheses after the full name. Methods include: Round Robin (Value "rr"):  Each server is used sequentially in a circular queue Shortest Response (Value "sr"):  The server with the lowest ping at the last health check gets the next request Least Connections (Value "lc"):  The server with the least current connections is given the next request Persistent IP - Round Robin (Value "pi"): The same server will be returned to a request during a users session.  Servers are chosen through round robin. Persistent IP - Shortest Response (Value "pi-sr"): The same server will be returned to a request during a users session.  Servers are chosen through shortest response. Persistent IP - Least Connections (Value "pi-lc"): The same server will be returned to a request during a users session.  Servers are chosen through least connections. Insert Cookie - Round Robin (Value "ic"):  Inserts a cookie into the HTTP stream that will tie that client to a particular balanced server. Servers are chosen through round robin. Insert Cookie - Shortest Response (Value "ic-sr"): Inserts a cookie into the HTTP stream that will tie that client to a particular balanced server. Servers are chosen through shortest response. Insert Cookie - Least Connections (Value "ic-lc"): Inserts a cookie into the HTTP stream that will tie that client to a particular balanced server. Servers are chosen through least connections.
	LoadBalancingMethod *string `json:"loadBalancingMethod,omitempty" xmlrpc:"loadBalancingMethod,omitempty"`

	// A human readable version of loadBalancingMethod, intended mainly for API users.
	LoadBalancingMethodFullName *string `json:"loadBalancingMethodFullName,omitempty" xmlrpc:"loadBalancingMethodFullName,omitempty"`

	// A flag indicating that the load balancer is a managed resource.
	ManagedResourceFlag *bool `json:"managedResourceFlag,omitempty" xmlrpc:"managedResourceFlag,omitempty"`

	// Date this load balancer was last modified
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of the load balancer instance
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// User-created notes on this load balancer.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The unique identifier of the Security Certificate to be utilized when SSL support is enabled.
	SecurityCertificateId *int `json:"securityCertificateId,omitempty" xmlrpc:"securityCertificateId,omitempty"`

	// A count of the services on this load balancer.
	ServiceCount *uint `json:"serviceCount,omitempty" xmlrpc:"serviceCount,omitempty"`

	// the services on this load balancer.
	Services []Network_LoadBalancer_Service `json:"services,omitempty" xmlrpc:"services,omitempty"`

	// This is the port for incoming traffic.
	SourcePort *int `json:"sourcePort,omitempty" xmlrpc:"sourcePort,omitempty"`

	// The connection type of this VIP.  Valid values are HTTP, FTP, TCP, UDP, and DNS.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The virtual, public-facing IP address for your load balancer.  This is the address of all incoming traffic
	VirtualIpAddress *string `json:"virtualIpAddress,omitempty" xmlrpc:"virtualIpAddress,omitempty"`
}

// The Syslog class holds a single line from the Networking Firewall "Syslog" record, for firewall detected and blocked attempts on a server.
type Network_Logging_Syslog struct {
	Entity

	// Timestamp for when the connection was blocked by the firewall
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The Destination IP Address of the blocked connection (your end)
	DestinationIpAddress *string `json:"destinationIpAddress,omitempty" xmlrpc:"destinationIpAddress,omitempty"`

	// The Destination Port of the blocked connection (your end)
	DestinationPort *int `json:"destinationPort,omitempty" xmlrpc:"destinationPort,omitempty"`

	// This tells you what kind of firewall event this log line is for:  accept or deny.
	EventType *string `json:"eventType,omitempty" xmlrpc:"eventType,omitempty"`

	// Raw syslog message for the event
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// Connection protocol used to make the call that was blocked (tcp, udp, etc)
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`

	// The Source IP Address of the call that was blocked (attacker's end)
	SourceIpAddress *string `json:"sourceIpAddress,omitempty" xmlrpc:"sourceIpAddress,omitempty"`

	// The Source Port where the blocked connection was established (attacker's end)
	SourcePort *int `json:"sourcePort,omitempty" xmlrpc:"sourcePort,omitempty"`

	// If this is an aggregation of syslog events, this property shows the total events.
	TotalEvents *int `json:"totalEvents,omitempty" xmlrpc:"totalEvents,omitempty"`
}

// The SoftLayer_Network_Media_Transcode_Account contains information regarding a transcode account.
type Network_Media_Transcode_Account struct {
	Entity

	// The SoftLayer account information
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The internal identifier of a SoftLayer account
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The created date
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The internal identifier of a transcode account
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The last modified date
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A count of transcode jobs
	TranscodeJobCount *uint `json:"transcodeJobCount,omitempty" xmlrpc:"transcodeJobCount,omitempty"`

	// Transcode jobs
	TranscodeJobs []Network_Media_Transcode_Job `json:"transcodeJobs,omitempty" xmlrpc:"transcodeJobs,omitempty"`
}

// The SoftLayer_Network_Media_Transcode_Job contains information regarding a transcode job such as input file, output format, user id and so on.
type Network_Media_Transcode_Job struct {
	Entity

	// The auto-deletion duration in seconds.  This value determines how long the input file will be kept on the storage.
	AutoDeleteDuration *int `json:"autoDeleteDuration,omitempty" xmlrpc:"autoDeleteDuration,omitempty"`

	// The size of an input file in byte
	ByteIn *int `json:"byteIn,omitempty" xmlrpc:"byteIn,omitempty"`

	// The created date
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	History []Network_Media_Transcode_Job_History `json:"history,omitempty" xmlrpc:"history,omitempty"`

	// A count of
	HistoryCount *uint `json:"historyCount,omitempty" xmlrpc:"historyCount,omitempty"`

	// The internal identifier of a transcode job
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The input file name
	InputFile *string `json:"inputFile,omitempty" xmlrpc:"inputFile,omitempty"`

	// The last modified date
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of a transcode job
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The output file name
	OutputFile *string `json:"outputFile,omitempty" xmlrpc:"outputFile,omitempty"`

	// The transcode service account
	TranscodeAccount *Network_Media_Transcode_Account `json:"transcodeAccount,omitempty" xmlrpc:"transcodeAccount,omitempty"`

	// The internal identifier of SoftLayer account
	TranscodeAccountId *int `json:"transcodeAccountId,omitempty" xmlrpc:"transcodeAccountId,omitempty"`

	// The unique id of a transcode job
	TranscodeJobGuid *string `json:"transcodeJobGuid,omitempty" xmlrpc:"transcodeJobGuid,omitempty"`

	// The unique id of a pre-defined output format
	TranscodePresetGuid *string `json:"transcodePresetGuid,omitempty" xmlrpc:"transcodePresetGuid,omitempty"`

	// The name of a transcode output preset
	TranscodePresetName *string `json:"transcodePresetName,omitempty" xmlrpc:"transcodePresetName,omitempty"`

	// The status information of a transcode job
	TranscodeStatus *Network_Media_Transcode_Job_Status `json:"transcodeStatus,omitempty" xmlrpc:"transcodeStatus,omitempty"`

	// The internal identifier of a transcode status
	TranscodeStatusId *int `json:"transcodeStatusId,omitempty" xmlrpc:"transcodeStatusId,omitempty"`

	// The status of a transcode job
	TranscodeStatusName *string `json:"transcodeStatusName,omitempty" xmlrpc:"transcodeStatusName,omitempty"`

	// The SoftLayer user that created the transcode job
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// The internal identifier of the user who created a transcode job
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// Watermark to apply to job
	Watermark *Container_Network_Media_Transcode_Job_Watermark `json:"watermark,omitempty" xmlrpc:"watermark,omitempty"`
}

// no documentation yet
type Network_Media_Transcode_Job_History struct {
	Entity

	// The creation date
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The note created by system
	PublicNotes *string `json:"publicNotes,omitempty" xmlrpc:"publicNotes,omitempty"`

	// The internal identifier of a transcode job
	TranscodeJobId *int `json:"transcodeJobId,omitempty" xmlrpc:"transcodeJobId,omitempty"`

	// The status of a transcode job
	TranscodeStatusName *string `json:"transcodeStatusName,omitempty" xmlrpc:"transcodeStatusName,omitempty"`
}

// The SoftLayer_Network_Media_Transcode_Job_Status contains information on a transcode job status.
type Network_Media_Transcode_Job_Status struct {
	Entity

	// The description of a transcode job status
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The internal identifier of a transcode job status
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The status name
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Message_Delivery struct {
	Entity

	// The SoftLayer customer account that a network message delivery account belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The billing item for a network message delivery account.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// The message delivery type of a network message delivery account.
	Type *Network_Message_Delivery_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// no documentation yet
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// no documentation yet
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`

	// The vendor for a network message delivery account.
	Vendor *Network_Message_Delivery_Vendor `json:"vendor,omitempty" xmlrpc:"vendor,omitempty"`

	// no documentation yet
	VendorId *int `json:"vendorId,omitempty" xmlrpc:"vendorId,omitempty"`
}

// no documentation yet
type Network_Message_Delivery_Attribute struct {
	Entity

	// no documentation yet
	NetworkMessageDelivery *Network_Message_Delivery `json:"networkMessageDelivery,omitempty" xmlrpc:"networkMessageDelivery,omitempty"`

	// no documentation yet
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Network_Message_Delivery_Email_Sendgrid struct {
	Network_Message_Delivery

	// The contact e-mail address used by SendGrid.
	EmailAddress *string `json:"emailAddress,omitempty" xmlrpc:"emailAddress,omitempty"`

	// A flag that determines if a SendGrid e-mail delivery account has access to send mail through the SendGrid SMTP server.
	SmtpAccess *string `json:"smtpAccess,omitempty" xmlrpc:"smtpAccess,omitempty"`
}

// no documentation yet
type Network_Message_Delivery_Type struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Message_Delivery_Vendor struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Network_Message_Queue data type contains general information relating to Message Queue account
type Network_Message_Queue struct {
	Entity

	// The account that a message queue belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A message queue's associated [[SoftLayer_Account|account]] id.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The current billing item for this message queue account.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The date that a message queue account was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A message queue's internal identification number
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A message queue status' internal identifier.
	MessageQueueStatusId *int `json:"messageQueueStatusId,omitempty" xmlrpc:"messageQueueStatusId,omitempty"`

	// A unique message queue account name
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of all available message queue nodes
	NodeCount *uint `json:"nodeCount,omitempty" xmlrpc:"nodeCount,omitempty"`

	// All available message queue nodes
	Nodes []Network_Message_Queue_Node `json:"nodes,omitempty" xmlrpc:"nodes,omitempty"`

	// Brief notes on this message queue account
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// A message queue account status.
	Status *Network_Message_Queue_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// The SoftLayer_Network_Message_Queue_Node data type contains general information relating to Message Queue node
type Network_Message_Queue_Node struct {
	Entity

	// A unique account name in this message queue node
	AccountName *string `json:"accountName,omitempty" xmlrpc:"accountName,omitempty"`

	// A message queue node's internal identification number
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The message queue account this node belongs to.
	MessageQueue *Network_Message_Queue `json:"messageQueue,omitempty" xmlrpc:"messageQueue,omitempty"`

	// A message queue node's associated message queue id.
	MessageQueueId *int `json:"messageQueueId,omitempty" xmlrpc:"messageQueueId,omitempty"`

	// A message queue node's metric tracking object. This object records all request and notification count data for this message queue node.
	MetricTrackingObject *Metric_Tracking_Object `json:"metricTrackingObject,omitempty" xmlrpc:"metricTrackingObject,omitempty"`

	// A user-friendly name of this message queue node
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// Brief notes on this message queue node
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// no documentation yet
	ServiceResource *Network_Service_Resource `json:"serviceResource,omitempty" xmlrpc:"serviceResource,omitempty"`
}

// The SoftLayer_Network_Message_Queue_Status data type contains general information relating to Message Queue account status.
type Network_Message_Queue_Status struct {
	Entity

	// A brief description on a message queue account status
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A message queue status's internal identification number
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A user-friendly name of a message queue account status
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Monitor struct {
	Entity
}

// The SoftLayer_Network_Monitor_Version1_Incident data type models a single virtual server or physical hardware network monitoring event. SoftLayer_Network_Monitor_Version1_Incidents are created when the SoftLayer monitoring system detects a service down on your hardware of virtual server. As the incident is resolved it's status changes from "SERVICE FAILURE" to "COMPLETED".
type Network_Monitor_Version1_Incident struct {
	Entity

	// A network monitoring incident's status, either the string "SERVICE FAILURE" denoting an ongoing incident or "COMPLETE" meaning the incident has been resolved.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// The Monitoring_Query_Host type represents a monitoring instance.  It consists of a hardware ID to monitor, an IP address attached to that hardware ID, a method of monitoring, and what to do in the instance that the monitor ever fails.
type Network_Monitor_Version1_Query_Host struct {
	Entity

	// The argument to be used for this monitor, if necessary.  The lowest monitoring levels (like ping) ignore this setting, but higher levels like HTTP custom use it.
	Arg1Value *string `json:"arg1Value,omitempty" xmlrpc:"arg1Value,omitempty"`

	// Virtual Guest Identification Number for the guest being monitored.
	GuestId *int `json:"guestId,omitempty" xmlrpc:"guestId,omitempty"`

	// The hardware that is being monitored by this monitoring instance
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The ID of the hardware being monitored
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// Identification Number for the host being monitored.
	HostId *int `json:"hostId,omitempty" xmlrpc:"hostId,omitempty"`

	// The unique identifier for this object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The IP address to be monitored.  Must be attached to the hardware on this object
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// The most recent result for this particular monitoring instance.
	LastResult *Network_Monitor_Version1_Query_Result `json:"lastResult,omitempty" xmlrpc:"lastResult,omitempty"`

	// The type of monitoring query that is executed when this hardware is monitored.
	QueryType *Network_Monitor_Version1_Query_Type `json:"queryType,omitempty" xmlrpc:"queryType,omitempty"`

	// The ID of the query type to use.
	QueryTypeId *int `json:"queryTypeId,omitempty" xmlrpc:"queryTypeId,omitempty"`

	// The action taken when a monitor fails.
	ResponseAction *Network_Monitor_Version1_Query_ResponseType `json:"responseAction,omitempty" xmlrpc:"responseAction,omitempty"`

	// The ID of the response action to take when the monitor fails
	ResponseActionId *int `json:"responseActionId,omitempty" xmlrpc:"responseActionId,omitempty"`

	// The status of this monitoring instance.  Anything other than "ON" means that the monitor has been disabled
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The number of 5-minute cycles to wait before the "responseAction" is taken.  If set to 0, the response action will be taken immediately
	WaitCycles *int `json:"waitCycles,omitempty" xmlrpc:"waitCycles,omitempty"`
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
	Entity

	// The hardware object that these monitoring permissions applies to.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The highest level of a monitoring query type allowed on this server
	MonitorLevel *int `json:"monitorLevel,omitempty" xmlrpc:"monitorLevel,omitempty"`

	// The highest level of a monitoring response type allowed on this server
	ResponseLevel *int `json:"responseLevel,omitempty" xmlrpc:"responseLevel,omitempty"`
}

// The ResponseType type stores only an ID and a description of the response type.  The only use for this object is in reference.  The user chooses a response action that would be appropriate for a monitoring instance, and sets the ResponseTypeId to the SoftLayer_Network_Monitor_Version1_Query_Host->responseActionId value.
//
// The user can retrieve all available ResponseTypes with the getAllObjects method on this service.
type Network_Monitor_Version1_Query_ResponseType struct {
	Entity

	// The description of the action the monitoring system will take on failure
	ActionDescription *string `json:"actionDescription,omitempty" xmlrpc:"actionDescription,omitempty"`

	// The unique identifier for this object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The level of this response.  The level the customer has access to is determined by values in SoftLayer_Network_Monitor_Version1_Query_Host_Stratum
	Level *int `json:"level,omitempty" xmlrpc:"level,omitempty"`
}

// The monitoring result object is used to show the status of the actions taken by the monitoring system.
//
// In general, only the responseStatus variable is needed, as it holds the information on the status of the service.
type Network_Monitor_Version1_Query_Result struct {
	Entity

	// The timestamp of when this monitor was co
	FinishTime *Time `json:"finishTime,omitempty" xmlrpc:"finishTime,omitempty"`

	// References the queryHost that this response relates to.
	QueryHost *Network_Monitor_Version1_Query_Host `json:"queryHost,omitempty" xmlrpc:"queryHost,omitempty"`

	// The response status for this server.  The response status meanings are: 0:  Down/Critical: Server is down and/or has passed the critical response threshold (extremely long ping response, abnormal behavior, etc.) 1:  Warning - Server may be recovering from a previous down state, or may have taken too long to respond 2:  Up 3:  Not used 4:  Unknown - An unknown error has occurred.  If the problem persists, contact support. 5:  Unknown - An unknown error has occurred.  If the problem persists, contact support.
	ResponseStatus *int `json:"responseStatus,omitempty" xmlrpc:"responseStatus,omitempty"`

	// The length of time it took the server to respond
	ResponseTime *Float64 `json:"responseTime,omitempty" xmlrpc:"responseTime,omitempty"`
}

// The MonitorType type stores a name, long description, and default arguments for the monitor types.  The only use for this object is in reference.  The user chooses a monitoring type that would be appropriate for their server, and sets the id of the Query_Type to SoftLayer_Network_Monitor_Version1_Query_Host->queryTypeId
//
// The user can retrieve all available Query Types with the getAllObjects method on this service.
type Network_Monitor_Version1_Query_Type struct {
	Entity

	// The type of parameter sent to the monitoring command.
	ArgumentDescription *string `json:"argumentDescription,omitempty" xmlrpc:"argumentDescription,omitempty"`

	// Long description of the monitoring type.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique identifier for this object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The level of this monitoring type.  The level the customer has access to is determined by values in SoftLayer_Network_Monitor_Version1_Query_Host_Stratum
	MonitorLevel *int `json:"monitorLevel,omitempty" xmlrpc:"monitorLevel,omitempty"`

	// Short name of the monitoring type
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// SoftLayer_Network_Pod refers to a portion of a data center that share a Backend Customer Router (BCR) and usually a front-end counterpart known as a Frontend Customer Router (FCR). A Pod primarily denotes a logical location within the network and the physical aspects that support networks. This is in contrast to representing a specific physical location.
//
// A ``Pod`` is identified by a ``name``, which is unique. A Pod name follows the format 'dddnn.podii', where 'ddd' is a data center code, 'nn' is the data center number, 'pod' is a literal string and 'ii' is a two digit, left-zero- padded number which corresponds to a Backend Customer Router (BCR) of the desired data center. Examples:
// * dal09.pod01 = Dallas 9, Pod 1 (ie. bcr01)
// * sjc01.pod04 = San Jose 1, Pod 4 (ie. bcr04)
// * ams01.pod01 = Amsterdam 1, Pod 1 (ie. bcr01)
type Network_Pod struct {
	Entity

	// Identifier for this Pod's Backend Customer Router (BCR)
	BackendRouterId *int `json:"backendRouterId,omitempty" xmlrpc:"backendRouterId,omitempty"`

	// Host name of Pod's Backend Customer Router (BCR), e.g. bcr01a.dal09
	BackendRouterName *string `json:"backendRouterName,omitempty" xmlrpc:"backendRouterName,omitempty"`

	// The list of capabilities this Pod has.
	Capabilities []string `json:"capabilities,omitempty" xmlrpc:"capabilities,omitempty"`

	// Long form name of the data center in which this Pod resides, e.g. Dallas 9
	DatacenterLongName *string `json:"datacenterLongName,omitempty" xmlrpc:"datacenterLongName,omitempty"`

	// Name of data center in which this Pod resides, e.g. dal09
	DatacenterName *string `json:"datacenterName,omitempty" xmlrpc:"datacenterName,omitempty"`

	// (optional) Identifier for this Pod's Frontend Customer Router (FCR)
	FrontendRouterId *int `json:"frontendRouterId,omitempty" xmlrpc:"frontendRouterId,omitempty"`

	// Host name of Pod's Frontend Customer Router (FCR), e.g. fcr01a.dal09
	FrontendRouterName *string `json:"frontendRouterName,omitempty" xmlrpc:"frontendRouterName,omitempty"`

	// The unique name of the Pod. See [[SoftLayer_Network_Pod (type)]] for details of the name's construction.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Protection_Address struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	DepartmentId *int `json:"departmentId,omitempty" xmlrpc:"departmentId,omitempty"`

	// no documentation yet
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// no documentation yet
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	ManagementMethodType *string `json:"managementMethodType,omitempty" xmlrpc:"managementMethodType,omitempty"`

	// no documentation yet
	ModifiedUser *User_Employee `json:"modifiedUser,omitempty" xmlrpc:"modifiedUser,omitempty"`

	// no documentation yet
	PrimaryRouter *Hardware_Router `json:"primaryRouter,omitempty" xmlrpc:"primaryRouter,omitempty"`

	// no documentation yet
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`

	// no documentation yet
	Subnet *Network_Subnet `json:"subnet,omitempty" xmlrpc:"subnet,omitempty"`

	// no documentation yet
	SubnetIpAddress *Network_Subnet_IpAddress `json:"subnetIpAddress,omitempty" xmlrpc:"subnetIpAddress,omitempty"`

	// no documentation yet
	TerminatedUser *User_Employee `json:"terminatedUser,omitempty" xmlrpc:"terminatedUser,omitempty"`

	// no documentation yet
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// A count of
	TransactionCount *uint `json:"transactionCount,omitempty" xmlrpc:"transactionCount,omitempty"`

	// no documentation yet
	Transactions []Provisioning_Version1_Transaction `json:"transactions,omitempty" xmlrpc:"transactions,omitempty"`

	// no documentation yet
	UserDepartment *User_Employee_Department `json:"userDepartment,omitempty" xmlrpc:"userDepartment,omitempty"`

	// no documentation yet
	UserRecord *User_Employee `json:"userRecord,omitempty" xmlrpc:"userRecord,omitempty"`
}

// Regional Internet Registries are the organizations who delegate IP address blocks to other groups or organizations around the Internet. The information contained in this data type is used throughout the networking-related services in our systems.
type Network_Regional_Internet_Registry struct {
	Entity

	// Unique ID of the object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The system-level name of the registry
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The friendly name of the registry
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Network_Security_Scanner_Request data type represents a single vulnerability scan request. It provides information on when the scan was created, last updated, and the current status. The status messages are as follows:
// *Scan Pending
// *Scan Processing
// *Scan Complete
// *Scan Cancelled
// *Generating Report.
type Network_Security_Scanner_Request struct {
	Entity

	// The account associated with a security scan request.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A request's associated customer account identifier.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The date and time that the request is created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The virtual guest a security scan is run against.
	Guest *Virtual_Guest `json:"guest,omitempty" xmlrpc:"guest,omitempty"`

	// Virtual Guest Identification Number for the guest this security scanner request belongs to.
	GuestId *int `json:"guestId,omitempty" xmlrpc:"guestId,omitempty"`

	// The hardware a security scan is run against.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The identifier of the hardware item a scan is run on.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// Identification Number for the host this security scanner request belongs to.
	HostId *int `json:"hostId,omitempty" xmlrpc:"hostId,omitempty"`

	// A security scan request's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The IP address that a scan will be performed on.
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// The date and time that the request was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Flag whether the requestor owns the hardware the scan was run on. This flag will  return for hardware servers only, virtual servers will result in a null return even if you have  a request out for them.
	RequestorOwnedFlag *bool `json:"requestorOwnedFlag,omitempty" xmlrpc:"requestorOwnedFlag,omitempty"`

	// A security scan request's status.
	Status *Network_Security_Scanner_Request_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// A request status identifier.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`
}

// The SoftLayer_Network_Security_Scanner_Request_Status data type represents the current status of a vulnerability scan. The status messages are as follows:
// *Scan Pending
// *Scan Processing
// *Scan Complete
// *Scan Cancelled
// *Generating Report.
//
//
// The status of a vulnerability scan will change over the course of a scan's execution.
type Network_Security_Scanner_Request_Status struct {
	Entity

	// The identifier of a vulnerability scan's status.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The status message of a vulnerability scan.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Many general services that SoftLayer provides are tracked on the customer portal with a quick status message. These status message provide users with a quick reference to the health of a service, whether it's up or down. These services include SoftLayer's Internet backbone connections, VPN entry points, and router networks. The SoftLayer_Network_Service_Health data type provides the relationship between these services and their health status.
type Network_Service_Health struct {
	Entity

	// The date that a service's status was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A service's location.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// A service's location identifier.
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// The date that a service's status was last changed.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The status portion of a service/status relationship.
	Status *Network_Service_Health_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// A service's status identifier.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`
}

// Many general services that SoftLayer provides are marked by a status message. These health messages give portal users a quick way of determining the state of a SoftLayer service. Services range from backbones to VPN endpoints and routers. Generally a health status is either "Up" or "Down".
type Network_Service_Health_Status struct {
	Entity

	// The status of a SoftLayer service. This is typically "Up" or "Down".
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Network_Service_Resource is used to store information related to a service.  It is used for determining the correct resource to connect to for a given service, like NAS, Evault, etc.
type Network_Service_Resource struct {
	Entity

	// no documentation yet
	ApiHost *string `json:"apiHost,omitempty" xmlrpc:"apiHost,omitempty"`

	// no documentation yet
	ApiPassword *string `json:"apiPassword,omitempty" xmlrpc:"apiPassword,omitempty"`

	// no documentation yet
	ApiPath *string `json:"apiPath,omitempty" xmlrpc:"apiPath,omitempty"`

	// no documentation yet
	ApiPort *string `json:"apiPort,omitempty" xmlrpc:"apiPort,omitempty"`

	// no documentation yet
	ApiProtocol *string `json:"apiProtocol,omitempty" xmlrpc:"apiProtocol,omitempty"`

	// no documentation yet
	ApiUsername *string `json:"apiUsername,omitempty" xmlrpc:"apiUsername,omitempty"`

	// no documentation yet
	ApiVersion *string `json:"apiVersion,omitempty" xmlrpc:"apiVersion,omitempty"`

	// A count of
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// no documentation yet
	Attributes []Network_Service_Resource_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// The backend IP address for this resource
	BackendIpAddress *string `json:"backendIpAddress,omitempty" xmlrpc:"backendIpAddress,omitempty"`

	// no documentation yet
	Datacenter *Location `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// The frontend IP address for this resource
	FrontendIpAddress *string `json:"frontendIpAddress,omitempty" xmlrpc:"frontendIpAddress,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The name associated with this resource
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The hardware information associated with this resource.
	NetworkDevice *Hardware `json:"networkDevice,omitempty" xmlrpc:"networkDevice,omitempty"`

	// no documentation yet
	SshUsername *string `json:"sshUsername,omitempty" xmlrpc:"sshUsername,omitempty"`

	// The network information associated with this resource.
	Type *Network_Service_Resource_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// no documentation yet
type Network_Service_Resource_Attribute struct {
	Entity

	// no documentation yet
	AttributeType *Network_Service_Resource_Attribute_Type `json:"attributeType,omitempty" xmlrpc:"attributeType,omitempty"`

	// no documentation yet
	ServiceResource *Network_Service_Resource `json:"serviceResource,omitempty" xmlrpc:"serviceResource,omitempty"`

	// no documentation yet
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Network_Service_Resource_Attribute_Type struct {
	Entity

	// no documentation yet
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`
}

// no documentation yet
type Network_Service_Resource_Hub struct {
	Network_Service_Resource
}

// no documentation yet
type Network_Service_Resource_Hub_Swift struct {
	Network_Service_Resource_Hub
}

// no documentation yet
type Network_Service_Resource_MonitoringHub struct {
	Network_Service_Resource

	// no documentation yet
	AdnServicesIp *string `json:"adnServicesIp,omitempty" xmlrpc:"adnServicesIp,omitempty"`

	// no documentation yet
	HubAddress *string `json:"hubAddress,omitempty" xmlrpc:"hubAddress,omitempty"`

	// no documentation yet
	HubConnectionTimeout *string `json:"hubConnectionTimeout,omitempty" xmlrpc:"hubConnectionTimeout,omitempty"`

	// no documentation yet
	RobotsCount *string `json:"robotsCount,omitempty" xmlrpc:"robotsCount,omitempty"`

	// no documentation yet
	RobotsMax *string `json:"robotsMax,omitempty" xmlrpc:"robotsMax,omitempty"`
}

// no documentation yet
type Network_Service_Resource_NimsoftLandingHub struct {
	Network_Service_Resource_MonitoringHub
}

// no documentation yet
type Network_Service_Resource_Type struct {
	Entity

	// A count of
	ServiceResourceCount *uint `json:"serviceResourceCount,omitempty" xmlrpc:"serviceResourceCount,omitempty"`

	// no documentation yet
	ServiceResources []Network_Service_Resource `json:"serviceResources,omitempty" xmlrpc:"serviceResources,omitempty"`

	// no documentation yet
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// The SoftLayer_Network_Service_Vpn_Overrides data type contains information relating user ids to subnet ids when VPN access is manually configured.  It is essentially an entry in a 'white list' of subnets a SoftLayer portal VPN user may access.
type Network_Service_Vpn_Overrides struct {
	Entity

	// The internal identifier of the record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Subnet components accessible by a SoftLayer VPN portal user.
	Subnet *Network_Subnet `json:"subnet,omitempty" xmlrpc:"subnet,omitempty"`

	// The identifier of a subnet accessible by the SoftLayer portal VPN user.
	SubnetId *int `json:"subnetId,omitempty" xmlrpc:"subnetId,omitempty"`

	// SoftLayer VPN portal user.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// The identifier of the SoftLayer portal VPN user.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// The SoftLayer_Network_Storage data type contains general information regarding a Storage product such as account id, access username and password, the Storage product type, and the server the Storage service is associated with. Currently, only EVault backup storage has an associated server.
type Network_Storage struct {
	Entity

	// The account that a Storage services belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The internal identifier of the SoftLayer customer account that a Storage account belongs to.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// Other usernames and passwords associated with a Storage volume.
	AccountPassword *Account_Password `json:"accountPassword,omitempty" xmlrpc:"accountPassword,omitempty"`

	// A count of the currently active transactions on a network storage volume.
	ActiveTransactionCount *uint `json:"activeTransactionCount,omitempty" xmlrpc:"activeTransactionCount,omitempty"`

	// The currently active transactions on a network storage volume.
	ActiveTransactions []Provisioning_Version1_Transaction `json:"activeTransactions,omitempty" xmlrpc:"activeTransactions,omitempty"`

	// The SoftLayer_Hardware objects which are allowed access to this storage volume.
	AllowedHardware []Hardware `json:"allowedHardware,omitempty" xmlrpc:"allowedHardware,omitempty"`

	// A count of the SoftLayer_Hardware objects which are allowed access to this storage volume.
	AllowedHardwareCount *uint `json:"allowedHardwareCount,omitempty" xmlrpc:"allowedHardwareCount,omitempty"`

	// A count of the SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume.
	AllowedIpAddressCount *uint `json:"allowedIpAddressCount,omitempty" xmlrpc:"allowedIpAddressCount,omitempty"`

	// The SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume.
	AllowedIpAddresses []Network_Subnet_IpAddress `json:"allowedIpAddresses,omitempty" xmlrpc:"allowedIpAddresses,omitempty"`

	// The SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
	AllowedReplicationHardware []Hardware `json:"allowedReplicationHardware,omitempty" xmlrpc:"allowedReplicationHardware,omitempty"`

	// A count of the SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
	AllowedReplicationHardwareCount *uint `json:"allowedReplicationHardwareCount,omitempty" xmlrpc:"allowedReplicationHardwareCount,omitempty"`

	// A count of the SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume's Replicant.
	AllowedReplicationIpAddressCount *uint `json:"allowedReplicationIpAddressCount,omitempty" xmlrpc:"allowedReplicationIpAddressCount,omitempty"`

	// The SoftLayer_Network_Subnet_IpAddress objects which are allowed access to this storage volume's Replicant.
	AllowedReplicationIpAddresses []Network_Subnet_IpAddress `json:"allowedReplicationIpAddresses,omitempty" xmlrpc:"allowedReplicationIpAddresses,omitempty"`

	// A count of the SoftLayer_Network_Subnet objects which are allowed access to this storage volume's Replicant.
	AllowedReplicationSubnetCount *uint `json:"allowedReplicationSubnetCount,omitempty" xmlrpc:"allowedReplicationSubnetCount,omitempty"`

	// The SoftLayer_Network_Subnet objects which are allowed access to this storage volume's Replicant.
	AllowedReplicationSubnets []Network_Subnet `json:"allowedReplicationSubnets,omitempty" xmlrpc:"allowedReplicationSubnets,omitempty"`

	// A count of the SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
	AllowedReplicationVirtualGuestCount *uint `json:"allowedReplicationVirtualGuestCount,omitempty" xmlrpc:"allowedReplicationVirtualGuestCount,omitempty"`

	// The SoftLayer_Hardware objects which are allowed access to this storage volume's Replicant.
	AllowedReplicationVirtualGuests []Virtual_Guest `json:"allowedReplicationVirtualGuests,omitempty" xmlrpc:"allowedReplicationVirtualGuests,omitempty"`

	// A count of the SoftLayer_Network_Subnet objects which are allowed access to this storage volume.
	AllowedSubnetCount *uint `json:"allowedSubnetCount,omitempty" xmlrpc:"allowedSubnetCount,omitempty"`

	// The SoftLayer_Network_Subnet objects which are allowed access to this storage volume.
	AllowedSubnets []Network_Subnet `json:"allowedSubnets,omitempty" xmlrpc:"allowedSubnets,omitempty"`

	// A count of the SoftLayer_Virtual_Guest objects which are allowed access to this storage volume.
	AllowedVirtualGuestCount *uint `json:"allowedVirtualGuestCount,omitempty" xmlrpc:"allowedVirtualGuestCount,omitempty"`

	// The SoftLayer_Virtual_Guest objects which are allowed access to this storage volume.
	AllowedVirtualGuests []Virtual_Guest `json:"allowedVirtualGuests,omitempty" xmlrpc:"allowedVirtualGuests,omitempty"`

	// The current billing item for a Storage volume.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// no documentation yet
	BillingItemCategory *Product_Item_Category `json:"billingItemCategory,omitempty" xmlrpc:"billingItemCategory,omitempty"`

	// The amount of space used by the volume, in bytes.
	BytesUsed *string `json:"bytesUsed,omitempty" xmlrpc:"bytesUsed,omitempty"`

	// A Storage account's capacity, measured in gigabytes.
	CapacityGb *int `json:"capacityGb,omitempty" xmlrpc:"capacityGb,omitempty"`

	// The date a network storage volume was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The schedule id which was executed to create a snapshot.
	CreationScheduleId *string `json:"creationScheduleId,omitempty" xmlrpc:"creationScheduleId,omitempty"`

	// A count of
	CredentialCount *uint `json:"credentialCount,omitempty" xmlrpc:"credentialCount,omitempty"`

	// no documentation yet
	Credentials []Network_Storage_Credential `json:"credentials,omitempty" xmlrpc:"credentials,omitempty"`

	// The Daily Schedule which is associated with this network storage volume.
	DailySchedule *Network_Storage_Schedule `json:"dailySchedule,omitempty" xmlrpc:"dailySchedule,omitempty"`

	// A count of the events which have taken place on a network storage volume.
	EventCount *uint `json:"eventCount,omitempty" xmlrpc:"eventCount,omitempty"`

	// The events which have taken place on a network storage volume.
	Events []Network_Storage_Event `json:"events,omitempty" xmlrpc:"events,omitempty"`

	// Retrieves the NFS Network Mount Address Name for a given File Storage Volume.
	FileNetworkMountAddress *string `json:"fileNetworkMountAddress,omitempty" xmlrpc:"fileNetworkMountAddress,omitempty"`

	// The unique identification number of the guest associated with a Storage volume.
	GuestId *int `json:"guestId,omitempty" xmlrpc:"guestId,omitempty"`

	// When applicable, the hardware associated with a Storage service.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The server that is associated with a Storage service.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// no documentation yet
	HasEncryptionAtRest *bool `json:"hasEncryptionAtRest,omitempty" xmlrpc:"hasEncryptionAtRest,omitempty"`

	// The unique identification number of the host associated with a Storage volume.
	HostId *int `json:"hostId,omitempty" xmlrpc:"hostId,omitempty"`

	// The Hourly Schedule which is associated with this network storage volume.
	HourlySchedule *Network_Storage_Schedule `json:"hourlySchedule,omitempty" xmlrpc:"hourlySchedule,omitempty"`

	// A Storage account's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The maximum number of IOPs selected for this volume.
	Iops *string `json:"iops,omitempty" xmlrpc:"iops,omitempty"`

	// A count of relationship between a container volume and iSCSI LUNs.
	IscsiLunCount *uint `json:"iscsiLunCount,omitempty" xmlrpc:"iscsiLunCount,omitempty"`

	// Relationship between a container volume and iSCSI LUNs.
	IscsiLuns []Network_Storage `json:"iscsiLuns,omitempty" xmlrpc:"iscsiLuns,omitempty"`

	// The ID of the LUN volume.
	LunId *string `json:"lunId,omitempty" xmlrpc:"lunId,omitempty"`

	// A count of the manually-created snapshots associated with this SoftLayer_Network_Storage volume. Does not support pagination by result limit and offset.
	ManualSnapshotCount *uint `json:"manualSnapshotCount,omitempty" xmlrpc:"manualSnapshotCount,omitempty"`

	// The manually-created snapshots associated with this SoftLayer_Network_Storage volume. Does not support pagination by result limit and offset.
	ManualSnapshots []Network_Storage `json:"manualSnapshots,omitempty" xmlrpc:"manualSnapshots,omitempty"`

	// A network storage volume's metric tracking object. This object records all periodic polled data available to this volume.
	MetricTrackingObject *Metric_Tracking_Object `json:"metricTrackingObject,omitempty" xmlrpc:"metricTrackingObject,omitempty"`

	// Whether or not a network storage volume may be mounted.
	MountableFlag *string `json:"mountableFlag,omitempty" xmlrpc:"mountableFlag,omitempty"`

	// A Storage account's type. Valid examples are "NAS", "LOCKBOX", "ISCSI", "EVAULT", and "HUB".
	NasType *string `json:"nasType,omitempty" xmlrpc:"nasType,omitempty"`

	// Public notes related to a Storage volume.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// A count of the subscribers that will be notified for usage amount warnings and overages.
	NotificationSubscriberCount *uint `json:"notificationSubscriberCount,omitempty" xmlrpc:"notificationSubscriberCount,omitempty"`

	// The subscribers that will be notified for usage amount warnings and overages.
	NotificationSubscribers []Notification_User_Subscriber `json:"notificationSubscribers,omitempty" xmlrpc:"notificationSubscribers,omitempty"`

	// A volume's configured SoftLayer_Network_Storage_Iscsi_OS_Type.
	OsType *Network_Storage_Iscsi_OS_Type `json:"osType,omitempty" xmlrpc:"osType,omitempty"`

	// A volume's configured SoftLayer_Network_Storage_Iscsi_OS_Type ID.
	OsTypeId *string `json:"osTypeId,omitempty" xmlrpc:"osTypeId,omitempty"`

	// A count of the volumes or snapshots partnered with a network storage volume in a parental role.
	ParentPartnershipCount *uint `json:"parentPartnershipCount,omitempty" xmlrpc:"parentPartnershipCount,omitempty"`

	// The volumes or snapshots partnered with a network storage volume in a parental role.
	ParentPartnerships []Network_Storage_Partnership `json:"parentPartnerships,omitempty" xmlrpc:"parentPartnerships,omitempty"`

	// The parent volume of a volume in a complex storage relationship.
	ParentVolume *Network_Storage `json:"parentVolume,omitempty" xmlrpc:"parentVolume,omitempty"`

	// A count of the volumes or snapshots partnered with a network storage volume.
	PartnershipCount *uint `json:"partnershipCount,omitempty" xmlrpc:"partnershipCount,omitempty"`

	// The volumes or snapshots partnered with a network storage volume.
	Partnerships []Network_Storage_Partnership `json:"partnerships,omitempty" xmlrpc:"partnerships,omitempty"`

	// The password used to access a non-EVault Storage volume. This password is used to register the EVault server agent with the vault backup system.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// A count of all permissions group(s) this volume is in.
	PermissionsGroupCount *uint `json:"permissionsGroupCount,omitempty" xmlrpc:"permissionsGroupCount,omitempty"`

	// All permissions group(s) this volume is in.
	PermissionsGroups []Network_Storage_Group `json:"permissionsGroups,omitempty" xmlrpc:"permissionsGroups,omitempty"`

	// The properties used to provide additional details about a network storage volume.
	Properties []Network_Storage_Property `json:"properties,omitempty" xmlrpc:"properties,omitempty"`

	// A count of the properties used to provide additional details about a network storage volume.
	PropertyCount *uint `json:"propertyCount,omitempty" xmlrpc:"propertyCount,omitempty"`

	// A count of the iSCSI LUN volumes being replicated by this network storage volume.
	ReplicatingLunCount *uint `json:"replicatingLunCount,omitempty" xmlrpc:"replicatingLunCount,omitempty"`

	// The iSCSI LUN volumes being replicated by this network storage volume.
	ReplicatingLuns []Network_Storage `json:"replicatingLuns,omitempty" xmlrpc:"replicatingLuns,omitempty"`

	// The network storage volume being replicated by a volume.
	ReplicatingVolume *Network_Storage `json:"replicatingVolume,omitempty" xmlrpc:"replicatingVolume,omitempty"`

	// A count of the volume replication events.
	ReplicationEventCount *uint `json:"replicationEventCount,omitempty" xmlrpc:"replicationEventCount,omitempty"`

	// The volume replication events.
	ReplicationEvents []Network_Storage_Event `json:"replicationEvents,omitempty" xmlrpc:"replicationEvents,omitempty"`

	// A count of the network storage volumes configured to be replicants of a volume.
	ReplicationPartnerCount *uint `json:"replicationPartnerCount,omitempty" xmlrpc:"replicationPartnerCount,omitempty"`

	// The network storage volumes configured to be replicants of a volume.
	ReplicationPartners []Network_Storage `json:"replicationPartners,omitempty" xmlrpc:"replicationPartners,omitempty"`

	// The Replication Schedule associated with a network storage volume.
	ReplicationSchedule *Network_Storage_Schedule `json:"replicationSchedule,omitempty" xmlrpc:"replicationSchedule,omitempty"`

	// The current replication status of a network storage volume. Indicates Failover or Failback status.
	ReplicationStatus *string `json:"replicationStatus,omitempty" xmlrpc:"replicationStatus,omitempty"`

	// A count of the schedules which are associated with a network storage volume.
	ScheduleCount *uint `json:"scheduleCount,omitempty" xmlrpc:"scheduleCount,omitempty"`

	// The schedules which are associated with a network storage volume.
	Schedules []Network_Storage_Schedule `json:"schedules,omitempty" xmlrpc:"schedules,omitempty"`

	// Service Provider ID
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`

	// The network resource a Storage service is connected to.
	ServiceResource *Network_Service_Resource `json:"serviceResource,omitempty" xmlrpc:"serviceResource,omitempty"`

	// The IP address of a Storage resource.
	ServiceResourceBackendIpAddress *string `json:"serviceResourceBackendIpAddress,omitempty" xmlrpc:"serviceResourceBackendIpAddress,omitempty"`

	// The name of a Storage's network resource.
	ServiceResourceName *string `json:"serviceResourceName,omitempty" xmlrpc:"serviceResourceName,omitempty"`

	// A volume's configured snapshot space size.
	SnapshotCapacityGb *string `json:"snapshotCapacityGb,omitempty" xmlrpc:"snapshotCapacityGb,omitempty"`

	// A count of the snapshots associated with this SoftLayer_Network_Storage volume.
	SnapshotCount *uint `json:"snapshotCount,omitempty" xmlrpc:"snapshotCount,omitempty"`

	// The creation timestamp of the snapshot on the storage platform.
	SnapshotCreationTimestamp *string `json:"snapshotCreationTimestamp,omitempty" xmlrpc:"snapshotCreationTimestamp,omitempty"`

	// The percentage of used snapshot space after which to delete automated snapshots.
	SnapshotDeletionThresholdPercentage *string `json:"snapshotDeletionThresholdPercentage,omitempty" xmlrpc:"snapshotDeletionThresholdPercentage,omitempty"`

	// The snapshot size in bytes.
	SnapshotSizeBytes *string `json:"snapshotSizeBytes,omitempty" xmlrpc:"snapshotSizeBytes,omitempty"`

	// A volume's available snapshot reservation space.
	SnapshotSpaceAvailable *string `json:"snapshotSpaceAvailable,omitempty" xmlrpc:"snapshotSpaceAvailable,omitempty"`

	// The snapshots associated with this SoftLayer_Network_Storage volume.
	Snapshots []Network_Storage `json:"snapshots,omitempty" xmlrpc:"snapshots,omitempty"`

	// no documentation yet
	StaasVersion *string `json:"staasVersion,omitempty" xmlrpc:"staasVersion,omitempty"`

	// A count of the network storage groups this volume is attached to.
	StorageGroupCount *uint `json:"storageGroupCount,omitempty" xmlrpc:"storageGroupCount,omitempty"`

	// The network storage groups this volume is attached to.
	StorageGroups []Network_Storage_Group `json:"storageGroups,omitempty" xmlrpc:"storageGroups,omitempty"`

	// no documentation yet
	StorageTierLevel *string `json:"storageTierLevel,omitempty" xmlrpc:"storageTierLevel,omitempty"`

	// A description of the Storage object.
	StorageType *Network_Storage_Type `json:"storageType,omitempty" xmlrpc:"storageType,omitempty"`

	// A storage object's type.
	StorageTypeId *string `json:"storageTypeId,omitempty" xmlrpc:"storageTypeId,omitempty"`

	// The amount of space used by the volume.
	TotalBytesUsed *string `json:"totalBytesUsed,omitempty" xmlrpc:"totalBytesUsed,omitempty"`

	// The total snapshot retention count of all schedules on this network storage volume.
	TotalScheduleSnapshotRetentionCount *uint `json:"totalScheduleSnapshotRetentionCount,omitempty" xmlrpc:"totalScheduleSnapshotRetentionCount,omitempty"`

	// This flag indicates whether this storage type is upgradable or not.
	UpgradableFlag *bool `json:"upgradableFlag,omitempty" xmlrpc:"upgradableFlag,omitempty"`

	// The usage notification for SL Storage services.
	UsageNotification *Notification `json:"usageNotification,omitempty" xmlrpc:"usageNotification,omitempty"`

	// The username used to access a non-EVault Storage volume. This username is used to register the EVault server agent with the vault backup system.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`

	// The type of network storage service.
	VendorName *string `json:"vendorName,omitempty" xmlrpc:"vendorName,omitempty"`

	// When applicable, the virtual guest associated with a Storage service.
	VirtualGuest *Virtual_Guest `json:"virtualGuest,omitempty" xmlrpc:"virtualGuest,omitempty"`

	// The username and password history for a Storage service.
	VolumeHistory []Network_Storage_History `json:"volumeHistory,omitempty" xmlrpc:"volumeHistory,omitempty"`

	// A count of the username and password history for a Storage service.
	VolumeHistoryCount *uint `json:"volumeHistoryCount,omitempty" xmlrpc:"volumeHistoryCount,omitempty"`

	// The current status of a network storage volume.
	VolumeStatus *string `json:"volumeStatus,omitempty" xmlrpc:"volumeStatus,omitempty"`

	// The account username and password for the EVault webCC interface.
	WebccAccount *Account_Password `json:"webccAccount,omitempty" xmlrpc:"webccAccount,omitempty"`

	// The Weekly Schedule which is associated with this network storage volume.
	WeeklySchedule *Network_Storage_Schedule `json:"weeklySchedule,omitempty" xmlrpc:"weeklySchedule,omitempty"`
}

// no documentation yet
type Network_Storage_Allowed_Host struct {
	Entity

	// A count of the SoftLayer_Network_Storage_Group objects this SoftLayer_Network_Storage_Allowed_Host is present in.
	AssignedGroupCount *uint `json:"assignedGroupCount,omitempty" xmlrpc:"assignedGroupCount,omitempty"`

	// The SoftLayer_Network_Storage_Group objects this SoftLayer_Network_Storage_Allowed_Host is present in.
	AssignedGroups []Network_Storage_Group `json:"assignedGroups,omitempty" xmlrpc:"assignedGroups,omitempty"`

	// A count of the SoftLayer_Network_Storage primary volumes whose replicas are allowed access.
	AssignedReplicationVolumeCount *uint `json:"assignedReplicationVolumeCount,omitempty" xmlrpc:"assignedReplicationVolumeCount,omitempty"`

	// The SoftLayer_Network_Storage primary volumes whose replicas are allowed access.
	AssignedReplicationVolumes []Network_Storage `json:"assignedReplicationVolumes,omitempty" xmlrpc:"assignedReplicationVolumes,omitempty"`

	// A count of the SoftLayer_Network_Storage volumes to which this SoftLayer_Network_Storage_Allowed_Host is allowed access.
	AssignedVolumeCount *uint `json:"assignedVolumeCount,omitempty" xmlrpc:"assignedVolumeCount,omitempty"`

	// The SoftLayer_Network_Storage volumes to which this SoftLayer_Network_Storage_Allowed_Host is allowed access.
	AssignedVolumes []Network_Storage `json:"assignedVolumes,omitempty" xmlrpc:"assignedVolumes,omitempty"`

	// The SoftLayer_Network_Storage_Credential this allowed host uses.
	Credential *Network_Storage_Credential `json:"credential,omitempty" xmlrpc:"credential,omitempty"`

	// The credential this allowed host will use
	CredentialId *int `json:"credentialId,omitempty" xmlrpc:"credentialId,omitempty"`

	// The internal identifier of the igroup
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The name of allowed host, usually an IQN or other identifier
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	ResourceTableId *int `json:"resourceTableId,omitempty" xmlrpc:"resourceTableId,omitempty"`

	// no documentation yet
	ResourceTableName *string `json:"resourceTableName,omitempty" xmlrpc:"resourceTableName,omitempty"`
}

// no documentation yet
type Network_Storage_Allowed_Host_Hardware struct {
	Network_Storage_Allowed_Host

	// The SoftLayer_Hardware object which this SoftLayer_Network_Storage_Allowed_Host is referencing.
	Resource *Hardware `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Network_Storage_Allowed_Host_IpAddress struct {
	Network_Storage_Allowed_Host

	// The SoftLayer_Network_Subnet_IpAddress object which this SoftLayer_Network_Storage_Allowed_Host is referencing.
	Resource *Network_Subnet_IpAddress `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Network_Storage_Allowed_Host_Subnet struct {
	Network_Storage_Allowed_Host

	// The SoftLayer_Network_Subnet object which this SoftLayer_Network_Storage_Allowed_Host is referencing.
	Resource *Network_Subnet `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Network_Storage_Allowed_Host_VirtualGuest struct {
	Network_Storage_Allowed_Host

	// The SoftLayer_Virtual_Guest object which this SoftLayer_Network_Storage_Allowed_Host is referencing.
	Resource *Virtual_Guest `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// The SoftLayer_Network_Storage_Backup contains general information regarding a Storage backup service such as account id, username, maximum capacity, password, Storage's product type and the server id.
type Network_Storage_Backup struct {
	Network_Storage

	// Peak number of bytes used in the vault for the current billing cycle.
	CurrentCyclePeakUsage *uint `json:"currentCyclePeakUsage,omitempty" xmlrpc:"currentCyclePeakUsage,omitempty"`

	// Peak number of bytes used in the vault for the previous billing cycle.
	PreviousCyclePeakUsage *uint `json:"previousCyclePeakUsage,omitempty" xmlrpc:"previousCyclePeakUsage,omitempty"`
}

// The SoftLayer_Network_Storage_Backup_Evault contains general information regarding an EVault Storage service such as account id, username, maximum capacity, password, Storage's product type and the server id.
type Network_Storage_Backup_Evault struct {
	Network_Storage_Backup
}

// The SoftLayer_Network_Storage_Backup_Evault_Version6 contains the same properties as the SoftLayer_Network_Storage_Backup_Evault. Additional properties available for the EVault Storage type:  softwareComponent, totalBytesUsed, backupJobDetails, restoreJobDetails and agentStatuses
type Network_Storage_Backup_Evault_Version6 struct {
	Network_Storage_Backup_Evault

	// A count of statuses (most of the time will be one status) for the agent tied to the EVault Storage services.
	AgentStatusCount *uint `json:"agentStatusCount,omitempty" xmlrpc:"agentStatusCount,omitempty"`

	// Statuses (most of the time will be one status) for the agent tied to the EVault Storage services.
	AgentStatuses []Container_Network_Storage_Evault_WebCc_AgentStatus `json:"agentStatuses,omitempty" xmlrpc:"agentStatuses,omitempty"`

	// A count of all the of the backup jobs for the EVault Storage account.
	BackupJobDetailCount *uint `json:"backupJobDetailCount,omitempty" xmlrpc:"backupJobDetailCount,omitempty"`

	// All the of the backup jobs for the EVault Storage account.
	BackupJobDetails []Container_Network_Storage_Evault_WebCc_JobDetails `json:"backupJobDetails,omitempty" xmlrpc:"backupJobDetails,omitempty"`

	// A count of the billing items for plugins tied to the EVault Storage service.
	PluginBillingItemCount *uint `json:"pluginBillingItemCount,omitempty" xmlrpc:"pluginBillingItemCount,omitempty"`

	// The billing items for plugins tied to the EVault Storage service.
	PluginBillingItems []Billing_Item `json:"pluginBillingItems,omitempty" xmlrpc:"pluginBillingItems,omitempty"`

	// A count of all the of the restore jobs for the EVault Storage account.
	RestoreJobDetailCount *uint `json:"restoreJobDetailCount,omitempty" xmlrpc:"restoreJobDetailCount,omitempty"`

	// All the of the restore jobs for the EVault Storage account.
	RestoreJobDetails []Container_Network_Storage_Evault_WebCc_JobDetails `json:"restoreJobDetails,omitempty" xmlrpc:"restoreJobDetails,omitempty"`

	// The software component for the EVault base client.
	SoftwareComponent *Software_Component `json:"softwareComponent,omitempty" xmlrpc:"softwareComponent,omitempty"`

	// A count of retrieve the task information for the EVault Storage service.
	TaskCount *uint `json:"taskCount,omitempty" xmlrpc:"taskCount,omitempty"`

	// Retrieve the task information for the EVault Storage service.
	Tasks []Container_Network_Storage_Evault_Vault_Task `json:"tasks,omitempty" xmlrpc:"tasks,omitempty"`
}

// The SoftLayer_Network_Storage_Credential data type will give you an overview of the usernames that are currently attached to your storage device.
type Network_Storage_Credential struct {
	Entity

	// This is the account that the storage credential is tied to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// This is the account id associated with the volume.
	AccountId *string `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// This is the data that the record was created in the table.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// This is the date that the record was last updated in the table.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// This is the id of the type of credential that this object represents.
	NasCredentialTypeId *int `json:"nasCredentialTypeId,omitempty" xmlrpc:"nasCredentialTypeId,omitempty"`

	// These are the SoftLayer_Network_Storage_Allowed_Host entries that this credential is assigned to.
	NetworkStorageAllowedHosts *Network_Storage_Allowed_Host `json:"networkStorageAllowedHosts,omitempty" xmlrpc:"networkStorageAllowedHosts,omitempty"`

	// This is the password associated with the volume.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// These are the types of storage that the credential can be assigned to.
	Type *Network_Storage_Credential_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// This is the username associated with the volume.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`

	// A count of these are the SoftLayer_Network_Storage volumes that this credential is assigned to.
	VolumeCount *uint `json:"volumeCount,omitempty" xmlrpc:"volumeCount,omitempty"`

	// These are the SoftLayer_Network_Storage volumes that this credential is assigned to.
	Volumes []Network_Storage `json:"volumes,omitempty" xmlrpc:"volumes,omitempty"`
}

// <<<
type Network_Storage_Credential_Type struct {
	Entity

	// The date a credential type was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A short description of the credential type
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The key name of the credential type.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The date a credential was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The human readable name of the credential type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Storage_Daily_Usage struct {
	Entity

	// no documentation yet
	BytesUsed *uint `json:"bytesUsed,omitempty" xmlrpc:"bytesUsed,omitempty"`

	// no documentation yet
	CdnHttpBandwidth *uint `json:"cdnHttpBandwidth,omitempty" xmlrpc:"cdnHttpBandwidth,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	NasVolume *Network_Storage `json:"nasVolume,omitempty" xmlrpc:"nasVolume,omitempty"`

	// no documentation yet
	NasVolumeId *int `json:"nasVolumeId,omitempty" xmlrpc:"nasVolumeId,omitempty"`

	// no documentation yet
	PublicBandwidthOut *uint `json:"publicBandwidthOut,omitempty" xmlrpc:"publicBandwidthOut,omitempty"`
}

// Storage volumes can create various events to keep track of what has occurred to the volume. Events provide an audit trail that can be used to verify that various tasks have occurred, such as snapshots to be created by a schedule or remote replication synchronization.
type Network_Storage_Event struct {
	Entity

	// The date an event was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The message text for an event.
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// A schedule that is associated with an event. Not all events will have a schedule.
	Schedule *Network_Storage_Schedule `json:"schedule,omitempty" xmlrpc:"schedule,omitempty"`

	// An identifier for the schedule which is associated with an event.
	ScheduleId *int `json:"scheduleId,omitempty" xmlrpc:"scheduleId,omitempty"`

	// An identifier for the type of an event.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// The associated volume for an event.
	Volume *Network_Storage `json:"volume,omitempty" xmlrpc:"volume,omitempty"`

	// The volume id which an event is associated with.
	VolumeId *int `json:"volumeId,omitempty" xmlrpc:"volumeId,omitempty"`
}

// no documentation yet
type Network_Storage_Group struct {
	Entity

	// The SoftLayer_Account which owns this group.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The account ID which owns this group
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The friendly name of this group
	Alias *string `json:"alias,omitempty" xmlrpc:"alias,omitempty"`

	// A count of the allowed hosts list for this group.
	AllowedHostCount *uint `json:"allowedHostCount,omitempty" xmlrpc:"allowedHostCount,omitempty"`

	// The allowed hosts list for this group.
	AllowedHosts []Network_Storage_Allowed_Host `json:"allowedHosts,omitempty" xmlrpc:"allowedHosts,omitempty"`

	// A count of the network storage volumes this group is attached to.
	AttachedVolumeCount *uint `json:"attachedVolumeCount,omitempty" xmlrpc:"attachedVolumeCount,omitempty"`

	// The network storage volumes this group is attached to.
	AttachedVolumes []Network_Storage `json:"attachedVolumes,omitempty" xmlrpc:"attachedVolumes,omitempty"`

	// The date this group was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The type which defines this group.
	GroupType *Network_Storage_Group_Type `json:"groupType,omitempty" xmlrpc:"groupType,omitempty"`

	// The SoftLayer_Network_Storage_Group_Type which describes this group.
	GroupTypeId *int `json:"groupTypeId,omitempty" xmlrpc:"groupTypeId,omitempty"`

	// The internal identifier of the group
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The OS Type this group is configured for.
	OsType *Network_Storage_Iscsi_OS_Type `json:"osType,omitempty" xmlrpc:"osType,omitempty"`

	// A SoftLayer_Network_Storage_OS_Type Operating System designation that this group was created for.
	OsTypeId *int `json:"osTypeId,omitempty" xmlrpc:"osTypeId,omitempty"`

	// The network resource this group is created on.
	ServiceResource *Network_Service_Resource `json:"serviceResource,omitempty" xmlrpc:"serviceResource,omitempty"`

	// A SoftLayer_Network_Service_Resource that this group was created on.
	ServiceResourceId *int `json:"serviceResourceId,omitempty" xmlrpc:"serviceResourceId,omitempty"`
}

// no documentation yet
type Network_Storage_Group_Iscsi struct {
	Network_Storage_Group
}

// no documentation yet
type Network_Storage_Group_Nfs struct {
	Network_Storage_Group
}

// no documentation yet
type Network_Storage_Group_Type struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Network_Storage_History contains the username/password past history for Storage services except Evault. Information such as the username, passwords, notes and the date of the password change may be retrieved.
type Network_Storage_History struct {
	Entity

	// The account that the Storage services belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// Date the password was changed.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The Storage service that the password history belongs to.
	NasVolume *Network_Storage `json:"nasVolume,omitempty" xmlrpc:"nasVolume,omitempty"`

	// Past notes for the Storage service.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// Password for the Storage service that was used in the past.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// Username for the Storage service.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// The SoftLayer_Network_Storage_Hub data type models Virtual Server type Storage storage offerings.
type Network_Storage_Hub struct {
	Network_Storage

	// A count of the billing items tied to a Storage service's bandwidth usage.
	BandwidthBillingItemCount *uint `json:"bandwidthBillingItemCount,omitempty" xmlrpc:"bandwidthBillingItemCount,omitempty"`

	// The billing items tied to a Storage service's bandwidth usage.
	BandwidthBillingItems []Billing_Item `json:"bandwidthBillingItems,omitempty" xmlrpc:"bandwidthBillingItems,omitempty"`
}

// no documentation yet
type Network_Storage_Hub_Cleversafe_Account struct {
	Entity

	// SoftLayer account to which an IBM Cloud Object Storage account belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The ID of the SoftLayer_Account which this IBM Cloud Object Storage account is associated with.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// An associated parent billing item which is active. Includes billing items which are scheduled to be cancelled in the future.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// An associated parent billing item which has been cancelled.
	CancelledBillingItem *Billing_Item `json:"cancelledBillingItem,omitempty" xmlrpc:"cancelledBillingItem,omitempty"`

	// A count of credentials used for generating an AWS signature. Max of 2.
	CredentialCount *uint `json:"credentialCount,omitempty" xmlrpc:"credentialCount,omitempty"`

	// Credentials used for generating an AWS signature. Max of 2.
	Credentials []Network_Storage_Credential `json:"credentials,omitempty" xmlrpc:"credentials,omitempty"`

	// The IMS ID of an IBM Cloud Object Storage account.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Provides an interface to various metrics relating to the usage of an IBM Cloud Object Storage account.
	MetricTrackingObject *Metric_Tracking_Object `json:"metricTrackingObject,omitempty" xmlrpc:"metricTrackingObject,omitempty"`

	// A user-defined field of notes.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// Human readable identifier of IBM Cloud Object Storage accounts.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`

	// Unique identifier for an IBM Cloud Object Storage account.
	Uuid *string `json:"uuid,omitempty" xmlrpc:"uuid,omitempty"`
}

// no documentation yet
type Network_Storage_Hub_Swift struct {
	Network_Storage_Hub

	// A count of
	StorageNodeCount *uint `json:"storageNodeCount,omitempty" xmlrpc:"storageNodeCount,omitempty"`

	// no documentation yet
	StorageNodes []Network_Service_Resource `json:"storageNodes,omitempty" xmlrpc:"storageNodes,omitempty"`
}

// no documentation yet
type Network_Storage_Hub_Swift_Container struct {
	Network_Storage_Hub_Swift
}

// no documentation yet
type Network_Storage_Hub_Swift_Share struct {
	Entity
}

// no documentation yet
type Network_Storage_Hub_Swift_Version1 struct {
	Network_Storage_Hub_Swift
}

// The iscsi data type provides access to additional information about an iscsi volume such as the snapshot capacity limit and replication partners.
type Network_Storage_Iscsi struct {
	Network_Storage
}

// The iscsi EqualLogic Version 3 data type provides access to additional information about an iscsi volume such as the available snapshot reserve space.
type Network_Storage_Iscsi_EqualLogic_Version3 struct {
	Network_Storage_Iscsi
}

// An iscsi replicant receives incoming data from an associated iscsi volume.  While the replicant is not in failover mode it will not be mountable.  Upon failover the replicant can be mounted and used as a normal volume.  It is suggested to only do this as part of a disaster recovery plan.
type Network_Storage_Iscsi_EqualLogic_Version3_Replicant struct {
	Network_Storage_Iscsi_EqualLogic_Version3

	// When a replicant is in the process of synchronizing with the parent volume this flag will be true.
	FailbackInProgressFlag *bool `json:"failbackInProgressFlag,omitempty" xmlrpc:"failbackInProgressFlag,omitempty"`

	// The volume name for an iscsi replicant.
	VolumeName *string `json:"volumeName,omitempty" xmlrpc:"volumeName,omitempty"`
}

// An iscsi snapshot is a point-in-time view of the data on an associated iscsi volume. Iscsi snapshots use a copy-on-write technology to minimize the amount of snapshot space used. When a snapshot is initially created it will use no snapshot space. At the time data changes on a volume which existed when a snapshot was created the original data will be saved in the associated volume's snapshot reserve space.
//
// As a snapshot is created offline it must be set mountable in order to mount it via an iscsi initiator service.
type Network_Storage_Iscsi_EqualLogic_Version3_Snapshot struct {
	Network_Storage_Iscsi_EqualLogic_Version3

	// If applicable, the schedule which was executed to create a snapshot.
	CreationSchedule *Network_Storage_Schedule `json:"creationSchedule,omitempty" xmlrpc:"creationSchedule,omitempty"`

	// The volume name for an iscsi snapshot.
	VolumeName *string `json:"volumeName,omitempty" xmlrpc:"volumeName,omitempty"`
}

// no documentation yet
type Network_Storage_Iscsi_OS_Type struct {
	Entity

	// The date this OS type record was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The description of this OS type
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The internal identifier of the OS type selection
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The key name of this OS type
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of this OS type
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Network_Storage_Nas contains general information regarding a NAS Storage service such as account id, username, password, maximum capacity, Storage's product type and capacity.
type Network_Storage_Nas struct {
	Network_Storage

	// no documentation yet
	RecentBytesUsed *Network_Storage_Daily_Usage `json:"recentBytesUsed,omitempty" xmlrpc:"recentBytesUsed,omitempty"`
}

// The SoftLayer_Network_Storage_OpenStack_Object data type models OpenStack specific object storage objects. These storages authenticate through Keystone to access Swift.
type Network_Storage_OpenStack_Object struct {
	Network_Storage

	// A count of the billing item tied to an OpenStack Object Storage's bandwidth service.
	BandwidthBillingItemCount *uint `json:"bandwidthBillingItemCount,omitempty" xmlrpc:"bandwidthBillingItemCount,omitempty"`

	// The billing item tied to an OpenStack Object Storage's bandwidth service.
	BandwidthBillingItems []Billing_Item `json:"bandwidthBillingItems,omitempty" xmlrpc:"bandwidthBillingItems,omitempty"`
}

// A network storage partnership is used to link multiple volumes to each other. These partnerships describe replication hierarchies or link volume snapshots to their associated storage volume.
type Network_Storage_Partnership struct {
	Entity

	// The date a partnership was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The date a partnership was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The associated child volume for a partnership.
	PartnerVolume *Network_Storage `json:"partnerVolume,omitempty" xmlrpc:"partnerVolume,omitempty"`

	// The child volume id which a partnership is associated with.
	PartnerVolumeId *int `json:"partnerVolumeId,omitempty" xmlrpc:"partnerVolumeId,omitempty"`

	// The type provides a standardized definition for a partnership.
	Type *Network_Storage_Partnership_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The associated parent volume for a partnership.
	Volume *Network_Storage `json:"volume,omitempty" xmlrpc:"volume,omitempty"`

	// The volume id which a partnership is associated with.
	VolumeId *int `json:"volumeId,omitempty" xmlrpc:"volumeId,omitempty"`
}

// A network storage partnership type is used to define the link between two volumes.
type Network_Storage_Partnership_Type struct {
	Entity

	// A type's description, for example 'ISCSI snapshot partnership'.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A type's key name, for example 'ISCSI_SNAPSHOT'.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// A type's name, for example 'ISCSI Snapshot'.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// A property provides additional information about a volume which it is assigned to. This information can range from "Mountable" flags to utilized snapshot space.
type Network_Storage_Property struct {
	Entity

	// The date a property was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The date a property was last modified;
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The type provides a standardized definition for a property.
	Type *Network_Storage_Property_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The value of a property.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`

	// The associated volume for a property.
	Volume *Network_Storage `json:"volume,omitempty" xmlrpc:"volume,omitempty"`

	// The volume id which a property is associated with.
	VolumeId *int `json:"volumeId,omitempty" xmlrpc:"volumeId,omitempty"`
}

// The storage property types provide standard definitions for properties which can be used with any type for Storage offering.  The properties provide additional information about a volume which they are assigned to.
type Network_Storage_Property_Type struct {
	Entity

	// A type's description, for example 'Determines whether the volume is currently mountable'.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A type's keyname, for example 'MOUNTABLE'.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// A type's name, for example 'Mountable'.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Storage_Replicant struct {
	Network_Storage

	// When a replicant is in the process of synchronizing with the parent volume this flag will be true.
	FailbackInProgressFlag *string `json:"failbackInProgressFlag,omitempty" xmlrpc:"failbackInProgressFlag,omitempty"`

	// The volume name for a replicant.
	VolumeName *string `json:"volumeName,omitempty" xmlrpc:"volumeName,omitempty"`
}

// Schedules can be created for select Storage services, such as iscsi. These schedules are used to perform various tasks such as scheduling snapshots or synchronizing replicants.
type Network_Storage_Schedule struct {
	Entity

	// A flag which determines if a schedule is active.
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// The date a schedule was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The day of the month parameter of this schedule.
	DayOfMonth *string `json:"dayOfMonth,omitempty" xmlrpc:"dayOfMonth,omitempty"`

	// The day of the week parameter of this schedule.
	DayOfWeek *string `json:"dayOfWeek,omitempty" xmlrpc:"dayOfWeek,omitempty"`

	// A count of events which have been created as the result of a schedule execution.
	EventCount *uint `json:"eventCount,omitempty" xmlrpc:"eventCount,omitempty"`

	// Events which have been created as the result of a schedule execution.
	Events []Network_Storage_Event `json:"events,omitempty" xmlrpc:"events,omitempty"`

	// The hour parameter of this schedule.
	Hour *string `json:"hour,omitempty" xmlrpc:"hour,omitempty"`

	// A schedule's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The minute parameter of this schedule.
	Minute *string `json:"minute,omitempty" xmlrpc:"minute,omitempty"`

	// The date a schedule was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The month of the year parameter of this schedule.
	MonthOfYear *string `json:"monthOfYear,omitempty" xmlrpc:"monthOfYear,omitempty"`

	// A schedule's name, for example 'Daily'.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The associated partnership for a schedule.
	Partnership *Network_Storage_Partnership `json:"partnership,omitempty" xmlrpc:"partnership,omitempty"`

	// The partnership id which a schedule is associated with.
	PartnershipId *int `json:"partnershipId,omitempty" xmlrpc:"partnershipId,omitempty"`

	// Properties used for configuration of a schedule.
	Properties []Network_Storage_Schedule_Property `json:"properties,omitempty" xmlrpc:"properties,omitempty"`

	// A count of properties used for configuration of a schedule.
	PropertyCount *uint `json:"propertyCount,omitempty" xmlrpc:"propertyCount,omitempty"`

	// A count of replica snapshots which have been created as the result of this schedule's execution.
	ReplicaSnapshotCount *uint `json:"replicaSnapshotCount,omitempty" xmlrpc:"replicaSnapshotCount,omitempty"`

	// Replica snapshots which have been created as the result of this schedule's execution.
	ReplicaSnapshots []Network_Storage `json:"replicaSnapshots,omitempty" xmlrpc:"replicaSnapshots,omitempty"`

	// The number of snapshots this schedule is configured to retain.
	RetentionCount *string `json:"retentionCount,omitempty" xmlrpc:"retentionCount,omitempty"`

	// A count of snapshots which have been created as the result of this schedule's execution.
	SnapshotCount *uint `json:"snapshotCount,omitempty" xmlrpc:"snapshotCount,omitempty"`

	// Snapshots which have been created as the result of this schedule's execution.
	Snapshots []Network_Storage `json:"snapshots,omitempty" xmlrpc:"snapshots,omitempty"`

	// The type provides a standardized definition for a schedule.
	Type *Network_Storage_Schedule_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The type id which a schedule is associated with.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// The associated volume for a schedule.
	Volume *Network_Storage `json:"volume,omitempty" xmlrpc:"volume,omitempty"`

	// The volume id which a schedule is associated with.
	VolumeId *int `json:"volumeId,omitempty" xmlrpc:"volumeId,omitempty"`
}

// Schedule properties provide attributes such as start date, end date, interval, and other properties to a storage schedule.
type Network_Storage_Schedule_Property struct {
	Entity

	// The date a schedule property was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A schedule property's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date a schedule property was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The associated schedule for a property.
	Schedule *Network_Storage_Schedule `json:"schedule,omitempty" xmlrpc:"schedule,omitempty"`

	// The type provides a standardized definition for a property.
	Type *Network_Storage_Schedule_Property_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// An identifier for the type of a property.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// The value of a property.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// A schedule property type is used to allow for a standardized method of defining network storage schedules.
type Network_Storage_Schedule_Property_Type struct {
	Entity

	// A type's description, for example 'Date for the schedule to start.'.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A schedule property type's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A schedule property type's key name, for example 'START_DATE'.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// A schedule property type's name, for example 'Start Date'.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The type of Storage volume type which a property type may be associated with.
	NasType *string `json:"nasType,omitempty" xmlrpc:"nasType,omitempty"`
}

// A schedule type is used to define what a schedule was created to do. When creating a schedule to take snapshots of a volume, the 'Snapshot' schedule type would be used.
type Network_Storage_Schedule_Type struct {
	Entity

	// A schedule type's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A schedule type's key name, for example 'SNAPSHOT'.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// A schedule type's name, for example 'Snapshot'.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Network_Storage_Snapshot struct {
	Network_Storage

	// If applicable, the schedule which was executed to create a snapshot.
	CreationSchedule *Network_Storage_Schedule `json:"creationSchedule,omitempty" xmlrpc:"creationSchedule,omitempty"`

	// The volume name for the snapshot.
	VolumeName *string `json:"volumeName,omitempty" xmlrpc:"volumeName,omitempty"`
}

// The SoftLayer_Network_Storage_Type contains a description of the associated SoftLayer_Network_Storage object.
type Network_Storage_Type struct {
	Entity

	// Human readable description for the associated SoftLayer_Network_Storage object.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// ID which corresponds with storageTypeId on storage objects.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Machine readable description code for the associated SoftLayer_Network_Storage object.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A count of the SoftLayer_Network_Storage object that uses this type.
	VolumeCount *uint `json:"volumeCount,omitempty" xmlrpc:"volumeCount,omitempty"`

	// The SoftLayer_Network_Storage object that uses this type.
	Volumes []Network_Storage `json:"volumes,omitempty" xmlrpc:"volumes,omitempty"`
}

// The SoftLayer_Network_Subnet data type contains general information relating to a single SoftLayer subnet. Personal information in this type such as names, addresses, and phone numbers are assigned to the account only and not to users belonging to the account.
type Network_Subnet struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// If present, the active registration for this subnet.
	ActiveRegistration *Network_Subnet_Registration `json:"activeRegistration,omitempty" xmlrpc:"activeRegistration,omitempty"`

	// All the swip transactions associated with a subnet that are still active.
	ActiveSwipTransaction *Network_Subnet_Swip_Transaction `json:"activeSwipTransaction,omitempty" xmlrpc:"activeSwipTransaction,omitempty"`

	// The billing item for a subnet.
	ActiveTransaction *Provisioning_Version1_Transaction `json:"activeTransaction,omitempty" xmlrpc:"activeTransaction,omitempty"`

	// Identifier which distinguishes whether the subnet is public or private address space.
	AddressSpace *string `json:"addressSpace,omitempty" xmlrpc:"addressSpace,omitempty"`

	// The SoftLayer_Network_Storage_Allowed_Host information to connect this Subnet to Network Storage supporting access control lists.
	AllowedHost *Network_Storage_Allowed_Host `json:"allowedHost,omitempty" xmlrpc:"allowedHost,omitempty"`

	// The SoftLayer_Network_Storage objects that this SoftLayer_Hardware has access to.
	AllowedNetworkStorage []Network_Storage `json:"allowedNetworkStorage,omitempty" xmlrpc:"allowedNetworkStorage,omitempty"`

	// A count of the SoftLayer_Network_Storage objects that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageCount *uint `json:"allowedNetworkStorageCount,omitempty" xmlrpc:"allowedNetworkStorageCount,omitempty"`

	// A count of the SoftLayer_Network_Storage objects whose Replica that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageReplicaCount *uint `json:"allowedNetworkStorageReplicaCount,omitempty" xmlrpc:"allowedNetworkStorageReplicaCount,omitempty"`

	// The SoftLayer_Network_Storage objects whose Replica that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageReplicas []Network_Storage `json:"allowedNetworkStorageReplicas,omitempty" xmlrpc:"allowedNetworkStorageReplicas,omitempty"`

	// The billing item for a subnet.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// A count of
	BoundDescendantCount *uint `json:"boundDescendantCount,omitempty" xmlrpc:"boundDescendantCount,omitempty"`

	// no documentation yet
	BoundDescendants []Network_Subnet `json:"boundDescendants,omitempty" xmlrpc:"boundDescendants,omitempty"`

	// A count of
	BoundRouterCount *uint `json:"boundRouterCount,omitempty" xmlrpc:"boundRouterCount,omitempty"`

	// Whether or not this subnet is associated with a router. Subnets that are not associated with a router cannot be routed.
	BoundRouterFlag *bool `json:"boundRouterFlag,omitempty" xmlrpc:"boundRouterFlag,omitempty"`

	// no documentation yet
	BoundRouters []Hardware `json:"boundRouters,omitempty" xmlrpc:"boundRouters,omitempty"`

	// The last IP address in a subnet is the subnet's broadcast address. This is an IP address that will broadcast network requests to the entire subnet and may not be assigned to a network interface.
	BroadcastAddress *string `json:"broadcastAddress,omitempty" xmlrpc:"broadcastAddress,omitempty"`

	// no documentation yet
	Children []Network_Subnet `json:"children,omitempty" xmlrpc:"children,omitempty"`

	// A count of
	ChildrenCount *uint `json:"childrenCount,omitempty" xmlrpc:"childrenCount,omitempty"`

	// A subnet's Classless Inter-Domain Routing prefix. This is a number between 0 and 32 signifying the number of bits in a subnet's netmask. These bits separate a subnet's network address from it's host addresses. It performs the same function as the ''netmask'' property, but is represented as an integer.
	Cidr *int `json:"cidr,omitempty" xmlrpc:"cidr,omitempty"`

	// The data center this subnet may be routed within.
	Datacenter *Location_Datacenter `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// A count of
	DescendantCount *uint `json:"descendantCount,omitempty" xmlrpc:"descendantCount,omitempty"`

	// no documentation yet
	Descendants []Network_Subnet `json:"descendants,omitempty" xmlrpc:"descendants,omitempty"`

	// no documentation yet
	DisplayLabel *string `json:"displayLabel,omitempty" xmlrpc:"displayLabel,omitempty"`

	// A static routed ip address
	EndPointIpAddress *Network_Subnet_IpAddress `json:"endPointIpAddress,omitempty" xmlrpc:"endPointIpAddress,omitempty"`

	// A subnet's gateway address. This is an IP address that belongs to the router on the subnet and may not be assigned to a network interface.
	Gateway *string `json:"gateway,omitempty" xmlrpc:"gateway,omitempty"`

	// no documentation yet
	GlobalIpRecord *Network_Subnet_IpAddress_Global `json:"globalIpRecord,omitempty" xmlrpc:"globalIpRecord,omitempty"`

	// The hardware using IP addresses on this subnet.
	Hardware []Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// A count of the hardware using IP addresses on this subnet.
	HardwareCount *uint `json:"hardwareCount,omitempty" xmlrpc:"hardwareCount,omitempty"`

	// A subnet's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of all the ip addresses associated with a subnet.
	IpAddressCount *uint `json:"ipAddressCount,omitempty" xmlrpc:"ipAddressCount,omitempty"`

	// All the ip addresses associated with a subnet.
	IpAddresses []Network_Subnet_IpAddress `json:"ipAddresses,omitempty" xmlrpc:"ipAddresses,omitempty"`

	// no documentation yet
	IsCustomerOwned *bool `json:"isCustomerOwned,omitempty" xmlrpc:"isCustomerOwned,omitempty"`

	// no documentation yet
	IsCustomerRoutable *bool `json:"isCustomerRoutable,omitempty" xmlrpc:"isCustomerRoutable,omitempty"`

	// The last time this subnet was last modified
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A bitmask in dotted-quad format that is used to separate a subnet's network address from it's host addresses. This performs the same function as the ''cidr'' property, but is expressed in a string format.
	Netmask *string `json:"netmask,omitempty" xmlrpc:"netmask,omitempty"`

	// A subnet's associated network component.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`

	// The upstream network component firewall.
	NetworkComponentFirewall *Network_Component_Firewall `json:"networkComponentFirewall,omitempty" xmlrpc:"networkComponentFirewall,omitempty"`

	// A subnet's network identifier. This is the first IP address of a subnet and may not be assigned to a network interface.
	NetworkIdentifier *string `json:"networkIdentifier,omitempty" xmlrpc:"networkIdentifier,omitempty"`

	// A count of
	NetworkProtectionAddressCount *uint `json:"networkProtectionAddressCount,omitempty" xmlrpc:"networkProtectionAddressCount,omitempty"`

	// no documentation yet
	NetworkProtectionAddresses []Network_Protection_Address `json:"networkProtectionAddresses,omitempty" xmlrpc:"networkProtectionAddresses,omitempty"`

	// A count of iPSec network tunnels that have access to a private subnet.
	NetworkTunnelContextCount *uint `json:"networkTunnelContextCount,omitempty" xmlrpc:"networkTunnelContextCount,omitempty"`

	// IPSec network tunnels that have access to a private subnet.
	NetworkTunnelContexts []Network_Tunnel_Module_Context `json:"networkTunnelContexts,omitempty" xmlrpc:"networkTunnelContexts,omitempty"`

	// The VLAN object that a subnet is associated with.
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`

	// A subnet's associated VLAN's internal identifier.
	NetworkVlanId *int `json:"networkVlanId,omitempty" xmlrpc:"networkVlanId,omitempty"`

	// This is the note field.
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	// The pod in which this subnet resides.
	PodName *string `json:"podName,omitempty" xmlrpc:"podName,omitempty"`

	// A count of
	ProtectedIpAddressCount *uint `json:"protectedIpAddressCount,omitempty" xmlrpc:"protectedIpAddressCount,omitempty"`

	// no documentation yet
	ProtectedIpAddresses []Network_Subnet_IpAddress `json:"protectedIpAddresses,omitempty" xmlrpc:"protectedIpAddresses,omitempty"`

	// no documentation yet
	RegionalInternetRegistry *Network_Regional_Internet_Registry `json:"regionalInternetRegistry,omitempty" xmlrpc:"regionalInternetRegistry,omitempty"`

	// A count of all registrations that have been created for this subnet.
	RegistrationCount *uint `json:"registrationCount,omitempty" xmlrpc:"registrationCount,omitempty"`

	// All registrations that have been created for this subnet.
	Registrations []Network_Subnet_Registration `json:"registrations,omitempty" xmlrpc:"registrations,omitempty"`

	// A count of the resource groups in which this subnet is a member.
	ResourceGroupCount *uint `json:"resourceGroupCount,omitempty" xmlrpc:"resourceGroupCount,omitempty"`

	// The resource groups in which this subnet is a member.
	ResourceGroups []Resource_Group `json:"resourceGroups,omitempty" xmlrpc:"resourceGroups,omitempty"`

	// The reverse DNS domain associated with this subnet.
	ReverseDomain *Dns_Domain `json:"reverseDomain,omitempty" xmlrpc:"reverseDomain,omitempty"`

	// An identifier of the role the subnet is within. Roles dictate how a subnet may be used.
	RoleKeyName *string `json:"roleKeyName,omitempty" xmlrpc:"roleKeyName,omitempty"`

	// The name of the role the subnet is within. Roles dictate how a subnet may be used.
	RoleName *string `json:"roleName,omitempty" xmlrpc:"roleName,omitempty"`

	// The identifier for the type of route then subnet is currently configured for.
	RoutingTypeKeyName *string `json:"routingTypeKeyName,omitempty" xmlrpc:"routingTypeKeyName,omitempty"`

	// The name for the type of route then subnet is currently configured for.
	RoutingTypeName *string `json:"routingTypeName,omitempty" xmlrpc:"routingTypeName,omitempty"`

	// A subnet can be one of several types. PRIMARY, ADDITIONAL_PRIMARY, SECONDARY, ROUTED_TO_VLAN, SECONDARY_ON_VLAN, and STATIC_IP_ROUTED. The type determines the order in which many subnets are sorted in the SoftLayer customer portal. This groups subnets of similar type together.
	SortOrder *string `json:"sortOrder,omitempty" xmlrpc:"sortOrder,omitempty"`

	// A subnet can be one of several types. PRIMARY, ADDITIONAL_PRIMARY, SECONDARY, ROUTED_TO_VLAN, SECONDARY_ON_VLAN, STORAGE_NETWORK, and STATIC_IP_ROUTED. A "PRIMARY" subnet is the primary network bound to a VLAN within the softlayer network. An "ADDITIONAL_PRIMARY" subnet is bound to a network VLAN to augment the pool of available primary IP addresses that may be assigned to a server. A "SECONDARY" subnet is any of the secondary subnet's bound to a VLAN interface. A "ROUTED_TO_VLAN" subnet is a portable subnet that can be routed to any server on a vlan. A "SECONDARY_ON_VLAN" subnet also doesn't exist as a VLAN interface, but is routed directly to a VLAN instead of a single IP address by SoftLayer's routers.
	SubnetType *string `json:"subnetType,omitempty" xmlrpc:"subnetType,omitempty"`

	// All the swip transactions associated with a subnet.
	SwipTransaction []Network_Subnet_Swip_Transaction `json:"swipTransaction,omitempty" xmlrpc:"swipTransaction,omitempty"`

	// A count of all the swip transactions associated with a subnet.
	SwipTransactionCount *uint `json:"swipTransactionCount,omitempty" xmlrpc:"swipTransactionCount,omitempty"`

	// The number of IP addresses contained within this subnet.
	TotalIpAddresses *Float64 `json:"totalIpAddresses,omitempty" xmlrpc:"totalIpAddresses,omitempty"`

	// A count of
	UnboundDescendantCount *uint `json:"unboundDescendantCount,omitempty" xmlrpc:"unboundDescendantCount,omitempty"`

	// no documentation yet
	UnboundDescendants []Network_Subnet `json:"unboundDescendants,omitempty" xmlrpc:"unboundDescendants,omitempty"`

	// The number of IP addresses that can be addressed within this subnet. For IPv4 subnets with a CIDR value of at most 30, a discount of 3 is taken from the total number of IP addresses for the subnet's unusable network, gateway and broadcast IP addresses. For IPv6 subnets with a CIDR value of at most 126, a discount of 2 is taken for the subnet's network and gateway IP addresses.
	UsableIpAddressCount *Float64 `json:"usableIpAddressCount,omitempty" xmlrpc:"usableIpAddressCount,omitempty"`

	// This is the Internet Protocol version. Current values may be either 4 or 6.
	Version *int `json:"version,omitempty" xmlrpc:"version,omitempty"`

	// A count of the Virtual Servers using IP addresses on this subnet.
	VirtualGuestCount *uint `json:"virtualGuestCount,omitempty" xmlrpc:"virtualGuestCount,omitempty"`

	// The Virtual Servers using IP addresses on this subnet.
	VirtualGuests []Virtual_Guest `json:"virtualGuests,omitempty" xmlrpc:"virtualGuests,omitempty"`
}

// The SoftLayer_Network_Subnet_IpAddress data type contains general information relating to a single SoftLayer IPv4 address.
type Network_Subnet_IpAddress struct {
	Entity

	// The SoftLayer_Network_Storage_Allowed_Host information to connect this IP Address to Network Storage supporting access control lists.
	AllowedHost *Network_Storage_Allowed_Host `json:"allowedHost,omitempty" xmlrpc:"allowedHost,omitempty"`

	// The SoftLayer_Network_Storage objects that this SoftLayer_Hardware has access to.
	AllowedNetworkStorage []Network_Storage `json:"allowedNetworkStorage,omitempty" xmlrpc:"allowedNetworkStorage,omitempty"`

	// A count of the SoftLayer_Network_Storage objects that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageCount *uint `json:"allowedNetworkStorageCount,omitempty" xmlrpc:"allowedNetworkStorageCount,omitempty"`

	// A count of the SoftLayer_Network_Storage objects whose Replica that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageReplicaCount *uint `json:"allowedNetworkStorageReplicaCount,omitempty" xmlrpc:"allowedNetworkStorageReplicaCount,omitempty"`

	// The SoftLayer_Network_Storage objects whose Replica that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageReplicas []Network_Storage `json:"allowedNetworkStorageReplicas,omitempty" xmlrpc:"allowedNetworkStorageReplicas,omitempty"`

	// The application delivery controller using this address.
	ApplicationDeliveryController *Network_Application_Delivery_Controller `json:"applicationDeliveryController,omitempty" xmlrpc:"applicationDeliveryController,omitempty"`

	// A count of an IPSec network tunnel's address translations. These translations use a SoftLayer ip address from an assigned static NAT subnet to deliver the packets to the remote (customer) destination.
	ContextTunnelTranslationCount *uint `json:"contextTunnelTranslationCount,omitempty" xmlrpc:"contextTunnelTranslationCount,omitempty"`

	// An IPSec network tunnel's address translations. These translations use a SoftLayer ip address from an assigned static NAT subnet to deliver the packets to the remote (customer) destination.
	ContextTunnelTranslations []Network_Tunnel_Module_Context_Address_Translation `json:"contextTunnelTranslations,omitempty" xmlrpc:"contextTunnelTranslations,omitempty"`

	// A count of all the subnets routed to an IP address.
	EndpointSubnetCount *uint `json:"endpointSubnetCount,omitempty" xmlrpc:"endpointSubnetCount,omitempty"`

	// All the subnets routed to an IP address.
	EndpointSubnets []Network_Subnet `json:"endpointSubnets,omitempty" xmlrpc:"endpointSubnets,omitempty"`

	// A network component that is statically routed to an IP address.
	GuestNetworkComponent *Virtual_Guest_Network_Component `json:"guestNetworkComponent,omitempty" xmlrpc:"guestNetworkComponent,omitempty"`

	// A network component that is statically routed to an IP address.
	GuestNetworkComponentBinding *Virtual_Guest_Network_Component_IpAddress `json:"guestNetworkComponentBinding,omitempty" xmlrpc:"guestNetworkComponentBinding,omitempty"`

	// A server that this IP address is routed to.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// An IP's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// An IP address expressed in dotted quad format.
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// Indicates if an IP address is reserved to be used as the network broadcast address and cannot be assigned to a network interface
	IsBroadcast *bool `json:"isBroadcast,omitempty" xmlrpc:"isBroadcast,omitempty"`

	// Indicates if an IP address is reserved to a gateway and cannot be assigned to a network interface
	IsGateway *bool `json:"isGateway,omitempty" xmlrpc:"isGateway,omitempty"`

	// Indicates if an IP address is reserved to a network address and cannot be assigned to a network interface
	IsNetwork *bool `json:"isNetwork,omitempty" xmlrpc:"isNetwork,omitempty"`

	// Indicates if an IP address is reserved and cannot be assigned to a network interface
	IsReserved *bool `json:"isReserved,omitempty" xmlrpc:"isReserved,omitempty"`

	// A network component that is statically routed to an IP address.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`

	// An IP address' user defined note.
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	// The network gateway appliance using this address as the private IP address.
	PrivateNetworkGateway *Network_Gateway `json:"privateNetworkGateway,omitempty" xmlrpc:"privateNetworkGateway,omitempty"`

	// no documentation yet
	ProtectionAddress []Network_Protection_Address `json:"protectionAddress,omitempty" xmlrpc:"protectionAddress,omitempty"`

	// A count of
	ProtectionAddressCount *uint `json:"protectionAddressCount,omitempty" xmlrpc:"protectionAddressCount,omitempty"`

	// The network gateway appliance using this address as the public IP address.
	PublicNetworkGateway *Network_Gateway `json:"publicNetworkGateway,omitempty" xmlrpc:"publicNetworkGateway,omitempty"`

	// An IPMI-based management network component of the IP address.
	RemoteManagementNetworkComponent *Network_Component `json:"remoteManagementNetworkComponent,omitempty" xmlrpc:"remoteManagementNetworkComponent,omitempty"`

	// An IP address' associated subnet.
	Subnet *Network_Subnet `json:"subnet,omitempty" xmlrpc:"subnet,omitempty"`

	// An IP address' subnet id.
	SubnetId *int `json:"subnetId,omitempty" xmlrpc:"subnetId,omitempty"`

	// All events for this IP address stored in the datacenter syslogs from the last 24 hours
	SyslogEventsOneDay []Network_Logging_Syslog `json:"syslogEventsOneDay,omitempty" xmlrpc:"syslogEventsOneDay,omitempty"`

	// A count of all events for this IP address stored in the datacenter syslogs from the last 24 hours
	SyslogEventsOneDayCount *uint `json:"syslogEventsOneDayCount,omitempty" xmlrpc:"syslogEventsOneDayCount,omitempty"`

	// A count of all events for this IP address stored in the datacenter syslogs from the last 7 days
	SyslogEventsSevenDayCount *uint `json:"syslogEventsSevenDayCount,omitempty" xmlrpc:"syslogEventsSevenDayCount,omitempty"`

	// All events for this IP address stored in the datacenter syslogs from the last 7 days
	SyslogEventsSevenDays []Network_Logging_Syslog `json:"syslogEventsSevenDays,omitempty" xmlrpc:"syslogEventsSevenDays,omitempty"`

	// Top Ten network datacenter syslog events, grouped by destination port, for the last 24 hours
	TopTenSyslogEventsByDestinationPortOneDay []Network_Logging_Syslog `json:"topTenSyslogEventsByDestinationPortOneDay,omitempty" xmlrpc:"topTenSyslogEventsByDestinationPortOneDay,omitempty"`

	// A count of top Ten network datacenter syslog events, grouped by destination port, for the last 24 hours
	TopTenSyslogEventsByDestinationPortOneDayCount *uint `json:"topTenSyslogEventsByDestinationPortOneDayCount,omitempty" xmlrpc:"topTenSyslogEventsByDestinationPortOneDayCount,omitempty"`

	// A count of top Ten network datacenter syslog events, grouped by destination port, for the last 7 days
	TopTenSyslogEventsByDestinationPortSevenDayCount *uint `json:"topTenSyslogEventsByDestinationPortSevenDayCount,omitempty" xmlrpc:"topTenSyslogEventsByDestinationPortSevenDayCount,omitempty"`

	// Top Ten network datacenter syslog events, grouped by destination port, for the last 7 days
	TopTenSyslogEventsByDestinationPortSevenDays []Network_Logging_Syslog `json:"topTenSyslogEventsByDestinationPortSevenDays,omitempty" xmlrpc:"topTenSyslogEventsByDestinationPortSevenDays,omitempty"`

	// Top Ten network datacenter syslog events, grouped by source port, for the last 24 hours
	TopTenSyslogEventsByProtocolsOneDay []Network_Logging_Syslog `json:"topTenSyslogEventsByProtocolsOneDay,omitempty" xmlrpc:"topTenSyslogEventsByProtocolsOneDay,omitempty"`

	// A count of top Ten network datacenter syslog events, grouped by source port, for the last 24 hours
	TopTenSyslogEventsByProtocolsOneDayCount *uint `json:"topTenSyslogEventsByProtocolsOneDayCount,omitempty" xmlrpc:"topTenSyslogEventsByProtocolsOneDayCount,omitempty"`

	// A count of top Ten network datacenter syslog events, grouped by source port, for the last 7 days
	TopTenSyslogEventsByProtocolsSevenDayCount *uint `json:"topTenSyslogEventsByProtocolsSevenDayCount,omitempty" xmlrpc:"topTenSyslogEventsByProtocolsSevenDayCount,omitempty"`

	// Top Ten network datacenter syslog events, grouped by source port, for the last 7 days
	TopTenSyslogEventsByProtocolsSevenDays []Network_Logging_Syslog `json:"topTenSyslogEventsByProtocolsSevenDays,omitempty" xmlrpc:"topTenSyslogEventsByProtocolsSevenDays,omitempty"`

	// Top Ten network datacenter syslog events, grouped by source ip address, for the last 24 hours
	TopTenSyslogEventsBySourceIpOneDay []Network_Logging_Syslog `json:"topTenSyslogEventsBySourceIpOneDay,omitempty" xmlrpc:"topTenSyslogEventsBySourceIpOneDay,omitempty"`

	// A count of top Ten network datacenter syslog events, grouped by source ip address, for the last 24 hours
	TopTenSyslogEventsBySourceIpOneDayCount *uint `json:"topTenSyslogEventsBySourceIpOneDayCount,omitempty" xmlrpc:"topTenSyslogEventsBySourceIpOneDayCount,omitempty"`

	// A count of top Ten network datacenter syslog events, grouped by source ip address, for the last 7 days
	TopTenSyslogEventsBySourceIpSevenDayCount *uint `json:"topTenSyslogEventsBySourceIpSevenDayCount,omitempty" xmlrpc:"topTenSyslogEventsBySourceIpSevenDayCount,omitempty"`

	// Top Ten network datacenter syslog events, grouped by source ip address, for the last 7 days
	TopTenSyslogEventsBySourceIpSevenDays []Network_Logging_Syslog `json:"topTenSyslogEventsBySourceIpSevenDays,omitempty" xmlrpc:"topTenSyslogEventsBySourceIpSevenDays,omitempty"`

	// Top Ten network datacenter syslog events, grouped by source port, for the last 24 hours
	TopTenSyslogEventsBySourcePortOneDay []Network_Logging_Syslog `json:"topTenSyslogEventsBySourcePortOneDay,omitempty" xmlrpc:"topTenSyslogEventsBySourcePortOneDay,omitempty"`

	// A count of top Ten network datacenter syslog events, grouped by source port, for the last 24 hours
	TopTenSyslogEventsBySourcePortOneDayCount *uint `json:"topTenSyslogEventsBySourcePortOneDayCount,omitempty" xmlrpc:"topTenSyslogEventsBySourcePortOneDayCount,omitempty"`

	// A count of top Ten network datacenter syslog events, grouped by source port, for the last 7 days
	TopTenSyslogEventsBySourcePortSevenDayCount *uint `json:"topTenSyslogEventsBySourcePortSevenDayCount,omitempty" xmlrpc:"topTenSyslogEventsBySourcePortSevenDayCount,omitempty"`

	// Top Ten network datacenter syslog events, grouped by source port, for the last 7 days
	TopTenSyslogEventsBySourcePortSevenDays []Network_Logging_Syslog `json:"topTenSyslogEventsBySourcePortSevenDays,omitempty" xmlrpc:"topTenSyslogEventsBySourcePortSevenDays,omitempty"`

	// A virtual guest that this IP address is routed to.
	VirtualGuest *Virtual_Guest `json:"virtualGuest,omitempty" xmlrpc:"virtualGuest,omitempty"`

	// A count of virtual licenses allocated for an IP Address.
	VirtualLicenseCount *uint `json:"virtualLicenseCount,omitempty" xmlrpc:"virtualLicenseCount,omitempty"`

	// Virtual licenses allocated for an IP Address.
	VirtualLicenses []Software_VirtualLicense `json:"virtualLicenses,omitempty" xmlrpc:"virtualLicenses,omitempty"`
}

// no documentation yet
type Network_Subnet_IpAddress_Global struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The active transaction associated with this Global IP.
	ActiveTransaction *Provisioning_Version1_Transaction `json:"activeTransaction,omitempty" xmlrpc:"activeTransaction,omitempty"`

	// The billing item for this Global IP.
	BillingItem *Billing_Item_Network_Subnet_IpAddress_Global `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// A Global IP Address' associated description
	Description *int `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	DestinationIpAddress *Network_Subnet_IpAddress `json:"destinationIpAddress,omitempty" xmlrpc:"destinationIpAddress,omitempty"`

	// A Global IP Address' associated [[SoftLayer_Network_Subnet_IpAddress|ipAddress]] ID
	DestinationIpAddressId *int `json:"destinationIpAddressId,omitempty" xmlrpc:"destinationIpAddressId,omitempty"`

	// A Global IP Address' unique identifier
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	IpAddress *Network_Subnet_IpAddress `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// A Global IP Address' associated [[SoftLayer_Account|account]] ID
	IpAddressId *int `json:"ipAddressId,omitempty" xmlrpc:"ipAddressId,omitempty"`

	// A Global IP Address' associated type [[SoftLayer_Network_Subnet_IpAddress_Global_Type|id]] ID
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`
}

// The SoftLayer_Network_Subnet_IpAddress data type contains general information relating to a single SoftLayer IPv6 address.
type Network_Subnet_IpAddress_Version6 struct {
	Network_Subnet_IpAddress

	// The network gateway appliance using this address as the public IPv6 address.
	PublicVersion6NetworkGateway *Network_Gateway `json:"publicVersion6NetworkGateway,omitempty" xmlrpc:"publicVersion6NetworkGateway,omitempty"`
}

// The subnet registration data type contains general information relating to a single subnet registration instance. These registration instances can be updated to reflect changes, and will record the changes in the [[SoftLayer_Network_Subnet_Registration_Event|events]].
type Network_Subnet_Registration struct {
	Entity

	// The account that this registration belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The registration object's associated [[SoftLayer_Account|account]] id
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The CIDR prefix for the registered subnet
	Cidr *int `json:"cidr,omitempty" xmlrpc:"cidr,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A count of the cross-reference records that tie the [[SoftLayer_Account_Regional_Registry_Detail]] objects to the registration object.
	DetailReferenceCount *uint `json:"detailReferenceCount,omitempty" xmlrpc:"detailReferenceCount,omitempty"`

	// The cross-reference records that tie the [[SoftLayer_Account_Regional_Registry_Detail]] objects to the registration object.
	DetailReferences []Network_Subnet_Registration_Details `json:"detailReferences,omitempty" xmlrpc:"detailReferences,omitempty"`

	// A count of the related registration events.
	EventCount *uint `json:"eventCount,omitempty" xmlrpc:"eventCount,omitempty"`

	// The related registration events.
	Events []Network_Subnet_Registration_Event `json:"events,omitempty" xmlrpc:"events,omitempty"`

	// Unique ID of the registration object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The "network" detail object.
	NetworkDetail *Account_Regional_Registry_Detail `json:"networkDetail,omitempty" xmlrpc:"networkDetail,omitempty"`

	// The RIR-specific handle or name of the registered subnet. This field is read-only.
	NetworkHandle *string `json:"networkHandle,omitempty" xmlrpc:"networkHandle,omitempty"`

	// The base IP address of the registered subnet
	NetworkIdentifier *string `json:"networkIdentifier,omitempty" xmlrpc:"networkIdentifier,omitempty"`

	// The "person" detail object.
	PersonDetail *Account_Regional_Registry_Detail `json:"personDetail,omitempty" xmlrpc:"personDetail,omitempty"`

	// The related Regional Internet Registry.
	RegionalInternetRegistry *Network_Regional_Internet_Registry `json:"regionalInternetRegistry,omitempty" xmlrpc:"regionalInternetRegistry,omitempty"`

	// The RIR handle that this registration object belongs to. This field may not be populated until the registration is complete.
	RegionalInternetRegistryHandle *Account_Rwhois_Handle `json:"regionalInternetRegistryHandle,omitempty" xmlrpc:"regionalInternetRegistryHandle,omitempty"`

	// The registration object's associated [[SoftLayer_Account_Rwhois_Handle|RIR handle]] id
	RegionalInternetRegistryHandleId *int `json:"regionalInternetRegistryHandleId,omitempty" xmlrpc:"regionalInternetRegistryHandleId,omitempty"`

	// The registration object's associated [[SoftLayer_Network_Regional_Internet_Registry|RIR]] id
	RegionalInternetRegistryId *int `json:"regionalInternetRegistryId,omitempty" xmlrpc:"regionalInternetRegistryId,omitempty"`

	// The status of this registration.
	Status *Network_Subnet_Registration_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The registration object's associated [[SoftLayer_Network_Subnet_Registration_Status|status]] id
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// The subnet that this registration pertains to.
	Subnet *Network_Subnet `json:"subnet,omitempty" xmlrpc:"subnet,omitempty"`
}

// APNIC-specific registration object. For more detail see [[SoftLayer_Network_Subnet_Registration (type)|SoftLayer_Network_Subnet_Registration]].
type Network_Subnet_Registration_Apnic struct {
	Network_Subnet_Registration
}

// ARIN-specific registration object. For more detail see [[SoftLayer_Network_Subnet_Registration (type)|SoftLayer_Network_Subnet_Registration]].
type Network_Subnet_Registration_Arin struct {
	Network_Subnet_Registration
}

// The SoftLayer_Network_Subnet_Registration_Details objects are used to relate [[SoftLayer_Account_Regional_Registry_Detail]] objects to a [[SoftLayer_Network_Subnet_Registration]] object. This allows for easy reuse of registration details. It is important to note that only one detail object per type may be associated to a registration object.
type Network_Subnet_Registration_Details struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The related [[SoftLayer_Account_Regional_Registry_Detail|detail object]].
	Detail *Account_Regional_Registry_Detail `json:"detail,omitempty" xmlrpc:"detail,omitempty"`

	// Numeric ID of the related [[SoftLayer_Account_Regional_Registry_Detail]] object
	DetailId *int `json:"detailId,omitempty" xmlrpc:"detailId,omitempty"`

	// Unique numeric ID of the object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The related [[SoftLayer_Network_Subnet_Registration|registration object]].
	Registration *Network_Subnet_Registration `json:"registration,omitempty" xmlrpc:"registration,omitempty"`

	// Numeric ID of the related [[SoftLayer_Network_Subnet_Registration]] object
	RegistrationId *int `json:"registrationId,omitempty" xmlrpc:"registrationId,omitempty"`
}

// Each time a [[SoftLayer_Network_Subnet_Registration|subnet registration]] object is created or modified, the system will generate an event for it. Additional actions that would create an event include RIR responses and error cases. *
type Network_Subnet_Registration_Event struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Unique numeric ID of the event object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A string message indicating what took place during this event
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The registration this event pertains to.
	Registration *Network_Subnet_Registration `json:"registration,omitempty" xmlrpc:"registration,omitempty"`

	// The numeric ID of the related [[SoftLayer_Network_Subnet_Registration]] object
	RegistrationId *int `json:"registrationId,omitempty" xmlrpc:"registrationId,omitempty"`

	// The type of this event.
	Type *Network_Subnet_Registration_Event_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The numeric ID of the associated [[SoftLayer_Network_Subnet_Registration_Event_Type|event type]] object
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`
}

// Subnet Registration Event Type objects describe the nature of a [[SoftLayer_Network_Subnet_Registration_Event]]
//
// The standard values for these objects are as follows: <ul> <li><strong>REGISTRATION_CREATED</strong> - Indicates that the registration has been created</li> <li><strong>REGISTRATION_UPDATED</strong> - Indicates that the registration has been updated</li> <li><strong>REGISTRATION_CANCELLED</strong> - Indicates that the registration has been cancelled</li> <li><strong>RIR_RESPONSE</strong> - Indicates that an action taken against the RIR has produced a response. More details will be provided in the event message.</li> <li><strong>ERROR</strong> - Indicates that an error has been encountered. More details will be provided in the event message.</li> <li><strong>NOTE</strong> - An employee or other system has entered a note regarding the registration. The note content will be provided in the event message.</li> </ul>
type Network_Subnet_Registration_Event_Type struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Unique numeric ID of the event type object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Code-friendly string name of the event type
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Human-readable name of the event type
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// RIPE-specific registration object. For more detail see [[SoftLayer_Network_Subnet_Registration (type)|SoftLayer_Network_Subnet_Registration]].
type Network_Subnet_Registration_Ripe struct {
	Network_Subnet_Registration
}

// Subnet Registration Status objects describe the current status of a subnet registration.
//
// The standard values for these objects are as follows: <ul> <li><strong>OPEN</strong> - Indicates that the registration object is new and has yet to be submitted to the RIR</li> <li><strong>PENDING</strong> - Indicates that the registration object has been submitted to the RIR and is awaiting response</li> <li><strong>COMPLETE</strong> - Indicates that the RIR action has completed</li> <li><strong>DELETED</strong> - Indicates that the registration object has been gracefully removed is no longer valid</li> <li><strong>CANCELLED</strong> - Indicates that the registration object has been abruptly removed is no longer valid</li> </ul>
type Network_Subnet_Registration_Status struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Unique numeric ID of the status object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Code-friendly string name of the status
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Human-readable name of the status
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Every SoftLayer customer account has contact information associated with it for reverse WHOIS purposes. An account's RWHOIS data, modeled by the SoftLayer_Network_Subnet_Rwhois_Data data type, is used by SoftLayer's reverse WHOIS server as well as for SWIP transactions. SoftLayer's reverse WHOIS servers respond to WHOIS queries for IP addresses belonging to a customer's servers, returning this RWHOIS data.
//
// A SoftLayer customer's RWHOIS data may not necessarily match their account or portal users' contact information.
type Network_Subnet_Rwhois_Data struct {
	Entity

	// An email address associated with an account's RWHOIS data that is responsible for responding to network abuse queries about malicious traffic coming from your servers' IP addresses.
	AbuseEmail *string `json:"abuseEmail,omitempty" xmlrpc:"abuseEmail,omitempty"`

	// The SoftLayer customer account associated with this reverse WHOIS data.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// An account's RWHOIS data's associated account identifier.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The first line of the mailing address associated with an account's RWHOIS data.
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// The second line of the mailing address associated with an account's RWHOIS data.
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// The city of the mailing address associated with an account's RWHOIS data.
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// The company name associated with an account's RWHOIS data.
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// A two-letter abbreviation of the country of the mailing address associated with an account's RWHOIS data.
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// The date an account's RWHOIS data was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The first name associated with an account's RWHOIS data.
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// An account's RWHOIS data's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The last name associated with an account's RWHOIS data.
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// The date an account's RWHOIS data was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The postal code of the mailing address associated with an account's RWHOIS data.
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// Whether an account's RWHOIS data refers to a private residence or not.
	PrivateResidenceFlag *bool `json:"privateResidenceFlag,omitempty" xmlrpc:"privateResidenceFlag,omitempty"`

	// A two-letter abbreviation of the state of the mailing address associated with an account's RWHOIS data. If an account does not reside in a province then this is typically blank.
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`
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
	Entity

	// The Account whose RWHOIS data was used to SWIP this subnet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A SWIP transaction's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A Name describing which state a SWIP  transaction is in.
	StatusName *string `json:"statusName,omitempty" xmlrpc:"statusName,omitempty"`

	// The subnet that this SWIP transaction was created for.
	Subnet *Network_Subnet `json:"subnet,omitempty" xmlrpc:"subnet,omitempty"`

	// ID Number of the Subnet for this SWIP transaction.
	SubnetId *int `json:"subnetId,omitempty" xmlrpc:"subnetId,omitempty"`
}

// no documentation yet
type Network_TippingPointReporting struct {
	Entity
}

// The SoftLayer_Network_Tunnel_Module_Context data type contains general information relating to a single SoftLayer network tunnel.  The SoftLayer_Network_Tunnel_Module_Context is useful to gather information such as related customer subnets (remote) and internal subnets (local) associated with the network tunnel as well as other information needed to manage the network tunnel.  Account and billing information related to the network tunnel can also be retrieved.
type Network_Tunnel_Module_Context struct {
	Entity

	// The account that a network tunnel belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A network tunnel's account identifier.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The transaction that is currently applying configurations for the network tunnel.
	ActiveTransaction *Provisioning_Version1_Transaction `json:"activeTransaction,omitempty" xmlrpc:"activeTransaction,omitempty"`

	// A count of a network tunnel's address translations.
	AddressTranslationCount *uint `json:"addressTranslationCount,omitempty" xmlrpc:"addressTranslationCount,omitempty"`

	// A network tunnel's address translations.
	AddressTranslations []Network_Tunnel_Module_Context_Address_Translation `json:"addressTranslations,omitempty" xmlrpc:"addressTranslations,omitempty"`

	// A flag used to specify when advanced configurations, complex configurations that require manual setup, are being applied to network devices for a network tunnel. When the flag is set to true (1), a network tunnel cannot be configured through the management portal nor the API.
	AdvancedConfigurationFlag *int `json:"advancedConfigurationFlag,omitempty" xmlrpc:"advancedConfigurationFlag,omitempty"`

	// A count of subnets that provide access to SoftLayer services such as the management portal and the SoftLayer API.
	AllAvailableServiceSubnetCount *uint `json:"allAvailableServiceSubnetCount,omitempty" xmlrpc:"allAvailableServiceSubnetCount,omitempty"`

	// Subnets that provide access to SoftLayer services such as the management portal and the SoftLayer API.
	AllAvailableServiceSubnets []Network_Subnet `json:"allAvailableServiceSubnets,omitempty" xmlrpc:"allAvailableServiceSubnets,omitempty"`

	// The current billing item for network tunnel.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The date a network tunnel was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The remote end of a network tunnel. This end of the network tunnel resides on an outside network and will be sending and receiving the IPSec packets.
	CustomerPeerIpAddress *string `json:"customerPeerIpAddress,omitempty" xmlrpc:"customerPeerIpAddress,omitempty"`

	// A count of remote subnets that are allowed access through a network tunnel.
	CustomerSubnetCount *uint `json:"customerSubnetCount,omitempty" xmlrpc:"customerSubnetCount,omitempty"`

	// Remote subnets that are allowed access through a network tunnel.
	CustomerSubnets []Network_Customer_Subnet `json:"customerSubnets,omitempty" xmlrpc:"customerSubnets,omitempty"`

	// The datacenter location for one end of the network tunnel that allows access to account's private subnets.
	Datacenter *Location `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// The name giving to a network tunnel by a user.
	FriendlyName *string `json:"friendlyName,omitempty" xmlrpc:"friendlyName,omitempty"`

	// A network tunnel's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The local  end of a network tunnel. This end of the network tunnel resides on the SoftLayer networks and allows access to remote end of the tunnel to subnets on SoftLayer networks.
	InternalPeerIpAddress *string `json:"internalPeerIpAddress,omitempty" xmlrpc:"internalPeerIpAddress,omitempty"`

	// A count of private subnets that can be accessed through the network tunnel.
	InternalSubnetCount *uint `json:"internalSubnetCount,omitempty" xmlrpc:"internalSubnetCount,omitempty"`

	// Private subnets that can be accessed through the network tunnel.
	InternalSubnets []Network_Subnet `json:"internalSubnets,omitempty" xmlrpc:"internalSubnets,omitempty"`

	// The date a network tunnel was last modified.
	//
	// NOTE:  This date should NOT be used to determine when the network tunnel configurations were last applied to the network device.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A network tunnel's unique name used on the network device.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// Authentication used to generate keys for protecting the negotiations for a network tunnel.
	PhaseOneAuthentication *string `json:"phaseOneAuthentication,omitempty" xmlrpc:"phaseOneAuthentication,omitempty"`

	// Determines the strength of the key used in the key exchange process.  The higher the group number the stronger the key is and the more secure it is.  However, processing time will increase as the strength of the key increases.  Both peers in the must use the Diffie-Hellman Group.
	PhaseOneDiffieHellmanGroup *int `json:"phaseOneDiffieHellmanGroup,omitempty" xmlrpc:"phaseOneDiffieHellmanGroup,omitempty"`

	// Encryption used to generate keys for protecting the negotiations for a network tunnel.
	PhaseOneEncryption *string `json:"phaseOneEncryption,omitempty" xmlrpc:"phaseOneEncryption,omitempty"`

	// Amount of time (in seconds) allowed to pass before the encryption key expires.  A new key is generated without interrupting service. Valid times are from 120 to 172800 seconds.
	PhaseOneKeylife *int `json:"phaseOneKeylife,omitempty" xmlrpc:"phaseOneKeylife,omitempty"`

	// The authentication used in phase 2 proposal negotiation process.
	PhaseTwoAuthentication *string `json:"phaseTwoAuthentication,omitempty" xmlrpc:"phaseTwoAuthentication,omitempty"`

	// Determines the strength of the key used in the key exchange process.  The higher the group number the stronger the key is and the more secure it is.  However, processing time will increase as the strength of the key increases.  Both peers must use the Diffie-Hellman Group.
	PhaseTwoDiffieHellmanGroup *int `json:"phaseTwoDiffieHellmanGroup,omitempty" xmlrpc:"phaseTwoDiffieHellmanGroup,omitempty"`

	// The encryption used in phase 2 proposal negotiation process.
	PhaseTwoEncryption *string `json:"phaseTwoEncryption,omitempty" xmlrpc:"phaseTwoEncryption,omitempty"`

	// Amount of time (in seconds) allowed to pass before the encryption key expires.  A new key is generated without interrupting service. Valid times are from 120 to 172800 seconds.
	PhaseTwoKeylife *int `json:"phaseTwoKeylife,omitempty" xmlrpc:"phaseTwoKeylife,omitempty"`

	// Determines if the generated keys are made from previous keys.  When PFS is specified, a Diffie-Hellman exchange occurs each time a new security association is negotiated.
	PhaseTwoPerfectForwardSecrecy *int `json:"phaseTwoPerfectForwardSecrecy,omitempty" xmlrpc:"phaseTwoPerfectForwardSecrecy,omitempty"`

	// A key used so that peers authenticate each other.  This key is hashed by using the phase one encryption and phase one authentication.
	PresharedKey *string `json:"presharedKey,omitempty" xmlrpc:"presharedKey,omitempty"`

	// A count of service subnets that can be access through the network tunnel.
	ServiceSubnetCount *uint `json:"serviceSubnetCount,omitempty" xmlrpc:"serviceSubnetCount,omitempty"`

	// Service subnets that can be access through the network tunnel.
	ServiceSubnets []Network_Subnet `json:"serviceSubnets,omitempty" xmlrpc:"serviceSubnets,omitempty"`

	// A count of subnets used for a network tunnel's address translations.
	StaticRouteSubnetCount *uint `json:"staticRouteSubnetCount,omitempty" xmlrpc:"staticRouteSubnetCount,omitempty"`

	// Subnets used for a network tunnel's address translations.
	StaticRouteSubnets []Network_Subnet `json:"staticRouteSubnets,omitempty" xmlrpc:"staticRouteSubnets,omitempty"`

	// The transaction history for this network tunnel.
	TransactionHistory []Provisioning_Version1_Transaction `json:"transactionHistory,omitempty" xmlrpc:"transactionHistory,omitempty"`

	// A count of the transaction history for this network tunnel.
	TransactionHistoryCount *uint `json:"transactionHistoryCount,omitempty" xmlrpc:"transactionHistoryCount,omitempty"`
}

// The SoftLayer_Network_Tunnel_Module_Context_Address_Translation data type contains general information relating to a single address translation. Information such as notes, ip addresses, along with record information, and network tunnel data may be retrieved.
type Network_Tunnel_Module_Context_Address_Translation struct {
	Entity

	// The ip address record that will receive the encrypted traffic.
	CustomerIpAddress *string `json:"customerIpAddress,omitempty" xmlrpc:"customerIpAddress,omitempty"`

	// The unique identifier for the ip address record that will receive the encrypted traffic.
	CustomerIpAddressId *int `json:"customerIpAddressId,omitempty" xmlrpc:"customerIpAddressId,omitempty"`

	// The ip address record for the ip that will receive the encrypted traffic from the IPSec network tunnel.
	CustomerIpAddressRecord *Network_Customer_Subnet_IpAddress `json:"customerIpAddressRecord,omitempty" xmlrpc:"customerIpAddressRecord,omitempty"`

	// An address translation's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The ip address record that will deliver the encrypted traffic.
	InternalIpAddress *string `json:"internalIpAddress,omitempty" xmlrpc:"internalIpAddress,omitempty"`

	// The unique identifier for the ip address record that will deliver the encrypted traffic.
	InternalIpAddressId *int `json:"internalIpAddressId,omitempty" xmlrpc:"internalIpAddressId,omitempty"`

	// The ip address record for the ip that will deliver the encrypted traffic from the IPSec network tunnel.
	InternalIpAddressRecord *Network_Subnet_IpAddress `json:"internalIpAddressRecord,omitempty" xmlrpc:"internalIpAddressRecord,omitempty"`

	// The IPSec network tunnel an address translation belongs to.
	NetworkTunnelContext *Network_Tunnel_Module_Context `json:"networkTunnelContext,omitempty" xmlrpc:"networkTunnelContext,omitempty"`

	// An address translation's network tunnel identifier.
	NetworkTunnelContextId *int `json:"networkTunnelContextId,omitempty" xmlrpc:"networkTunnelContextId,omitempty"`

	// A name or description given to an address translation to help identify the address translation.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`
}

// The SoftLayer_Network_Vlan data type models a single VLAN within SoftLayer's public and private networks. a Virtual LAN is a structure that associates network interfaces on routers, switches, and servers in different locations to act as if they were on the same local network broadcast domain. VLANs are a central part of the SoftLayer network. They can determine how new IP subnets are routed and how individual servers communicate to each other.
type Network_Vlan struct {
	Entity

	// The SoftLayer customer account associated with a VLAN.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The internal identifier of the SoftLayer customer account that a VLAN is associated with.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of a VLAN's additional primary subnets. These are used to extend the number of servers attached to the VLAN by adding more ip addresses to the primary IP address pool.
	AdditionalPrimarySubnetCount *uint `json:"additionalPrimarySubnetCount,omitempty" xmlrpc:"additionalPrimarySubnetCount,omitempty"`

	// A VLAN's additional primary subnets. These are used to extend the number of servers attached to the VLAN by adding more ip addresses to the primary IP address pool.
	AdditionalPrimarySubnets []Network_Subnet `json:"additionalPrimarySubnets,omitempty" xmlrpc:"additionalPrimarySubnets,omitempty"`

	// The gateway this VLAN is inside of.
	AttachedNetworkGateway *Network_Gateway `json:"attachedNetworkGateway,omitempty" xmlrpc:"attachedNetworkGateway,omitempty"`

	// Whether or not this VLAN is inside a gateway.
	AttachedNetworkGatewayFlag *bool `json:"attachedNetworkGatewayFlag,omitempty" xmlrpc:"attachedNetworkGatewayFlag,omitempty"`

	// The inside VLAN record if this VLAN is inside a network gateway.
	AttachedNetworkGatewayVlan *Network_Gateway_Vlan `json:"attachedNetworkGatewayVlan,omitempty" xmlrpc:"attachedNetworkGatewayVlan,omitempty"`

	// The billing item for a network vlan.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// A flag indicating that a network vlan is on a Hardware Firewall (Dedicated).
	DedicatedFirewallFlag *int `json:"dedicatedFirewallFlag,omitempty" xmlrpc:"dedicatedFirewallFlag,omitempty"`

	// The extension router that a VLAN is associated with.
	ExtensionRouter *Hardware_Router `json:"extensionRouter,omitempty" xmlrpc:"extensionRouter,omitempty"`

	// A count of a firewalled Vlan's network components.
	FirewallGuestNetworkComponentCount *uint `json:"firewallGuestNetworkComponentCount,omitempty" xmlrpc:"firewallGuestNetworkComponentCount,omitempty"`

	// A firewalled Vlan's network components.
	FirewallGuestNetworkComponents []Network_Component_Firewall `json:"firewallGuestNetworkComponents,omitempty" xmlrpc:"firewallGuestNetworkComponents,omitempty"`

	// A count of a firewalled vlan's inbound/outbound interfaces.
	FirewallInterfaceCount *uint `json:"firewallInterfaceCount,omitempty" xmlrpc:"firewallInterfaceCount,omitempty"`

	// A firewalled vlan's inbound/outbound interfaces.
	FirewallInterfaces []Network_Firewall_Module_Context_Interface `json:"firewallInterfaces,omitempty" xmlrpc:"firewallInterfaces,omitempty"`

	// A count of a firewalled Vlan's network components.
	FirewallNetworkComponentCount *uint `json:"firewallNetworkComponentCount,omitempty" xmlrpc:"firewallNetworkComponentCount,omitempty"`

	// A firewalled Vlan's network components.
	FirewallNetworkComponents []Network_Component_Firewall `json:"firewallNetworkComponents,omitempty" xmlrpc:"firewallNetworkComponents,omitempty"`

	// A count of the currently running rule set of a firewalled VLAN.
	FirewallRuleCount *uint `json:"firewallRuleCount,omitempty" xmlrpc:"firewallRuleCount,omitempty"`

	// The currently running rule set of a firewalled VLAN.
	FirewallRules []Network_Vlan_Firewall_Rule `json:"firewallRules,omitempty" xmlrpc:"firewallRules,omitempty"`

	// A count of the networking components that are connected to a VLAN.
	GuestNetworkComponentCount *uint `json:"guestNetworkComponentCount,omitempty" xmlrpc:"guestNetworkComponentCount,omitempty"`

	// The networking components that are connected to a VLAN.
	GuestNetworkComponents []Virtual_Guest_Network_Component `json:"guestNetworkComponents,omitempty" xmlrpc:"guestNetworkComponents,omitempty"`

	// All of the hardware that exists on a VLAN. Hardware is associated with a VLAN by its networking components.
	Hardware []Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// A count of all of the hardware that exists on a VLAN. Hardware is associated with a VLAN by its networking components.
	HardwareCount *uint `json:"hardwareCount,omitempty" xmlrpc:"hardwareCount,omitempty"`

	// no documentation yet
	HighAvailabilityFirewallFlag *bool `json:"highAvailabilityFirewallFlag,omitempty" xmlrpc:"highAvailabilityFirewallFlag,omitempty"`

	// A VLAN's internal identifier. This should not be confused with the ''vlanNumber'' property, which is used in network configuration.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A flag indicating that a vlan can be assigned to a host that has local disk functionality.
	LocalDiskStorageCapabilityFlag *bool `json:"localDiskStorageCapabilityFlag,omitempty" xmlrpc:"localDiskStorageCapabilityFlag,omitempty"`

	// The date a VLAN was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The optional name for this VLAN
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The network in which this VLAN resides.
	Network *Network `json:"network,omitempty" xmlrpc:"network,omitempty"`

	// A count of the networking components that are connected to a VLAN.
	NetworkComponentCount *uint `json:"networkComponentCount,omitempty" xmlrpc:"networkComponentCount,omitempty"`

	// A count of the network components that are connected to this VLAN through a trunk.
	NetworkComponentTrunkCount *uint `json:"networkComponentTrunkCount,omitempty" xmlrpc:"networkComponentTrunkCount,omitempty"`

	// The network components that are connected to this VLAN through a trunk.
	NetworkComponentTrunks []Network_Component_Network_Vlan_Trunk `json:"networkComponentTrunks,omitempty" xmlrpc:"networkComponentTrunks,omitempty"`

	// The networking components that are connected to a VLAN.
	NetworkComponents []Network_Component `json:"networkComponents,omitempty" xmlrpc:"networkComponents,omitempty"`

	// Identifier to denote whether a VLAN is used for public or private connectivity.
	NetworkSpace *string `json:"networkSpace,omitempty" xmlrpc:"networkSpace,omitempty"`

	// The Hardware Firewall (Dedicated) for a network vlan.
	NetworkVlanFirewall *Network_Vlan_Firewall `json:"networkVlanFirewall,omitempty" xmlrpc:"networkVlanFirewall,omitempty"`

	// The note for this vlan.
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	// The primary router that a VLAN is associated with. Every SoftLayer VLAN is connected to more than one router for greater network redundancy.
	PrimaryRouter *Hardware_Router `json:"primaryRouter,omitempty" xmlrpc:"primaryRouter,omitempty"`

	// A VLAN's primary subnet. Each VLAN has at least one subnet, usually the subnet that is assigned to a server or new IP address block when it's purchased.
	PrimarySubnet *Network_Subnet `json:"primarySubnet,omitempty" xmlrpc:"primarySubnet,omitempty"`

	// A count of
	PrimarySubnetCount *uint `json:"primarySubnetCount,omitempty" xmlrpc:"primarySubnetCount,omitempty"`

	// The internal identifier of the primary subnet addressed on a VLAN.
	PrimarySubnetId *int `json:"primarySubnetId,omitempty" xmlrpc:"primarySubnetId,omitempty"`

	// A VLAN's primary IPv6 subnet. Some VLAN's may not have a primary IPv6 subnet.
	PrimarySubnetVersion6 *Network_Subnet `json:"primarySubnetVersion6,omitempty" xmlrpc:"primarySubnetVersion6,omitempty"`

	// no documentation yet
	PrimarySubnets []Network_Subnet `json:"primarySubnets,omitempty" xmlrpc:"primarySubnets,omitempty"`

	// A count of the gateways this VLAN is the private VLAN of.
	PrivateNetworkGatewayCount *uint `json:"privateNetworkGatewayCount,omitempty" xmlrpc:"privateNetworkGatewayCount,omitempty"`

	// The gateways this VLAN is the private VLAN of.
	PrivateNetworkGateways []Network_Gateway `json:"privateNetworkGateways,omitempty" xmlrpc:"privateNetworkGateways,omitempty"`

	// A count of
	ProtectedIpAddressCount *uint `json:"protectedIpAddressCount,omitempty" xmlrpc:"protectedIpAddressCount,omitempty"`

	// no documentation yet
	ProtectedIpAddresses []Network_Subnet_IpAddress `json:"protectedIpAddresses,omitempty" xmlrpc:"protectedIpAddresses,omitempty"`

	// A count of the gateways this VLAN is the public VLAN of.
	PublicNetworkGatewayCount *uint `json:"publicNetworkGatewayCount,omitempty" xmlrpc:"publicNetworkGatewayCount,omitempty"`

	// The gateways this VLAN is the public VLAN of.
	PublicNetworkGateways []Network_Gateway `json:"publicNetworkGateways,omitempty" xmlrpc:"publicNetworkGateways,omitempty"`

	// A count of the resource groups in which this VLAN is a member.
	ResourceGroupCount *uint `json:"resourceGroupCount,omitempty" xmlrpc:"resourceGroupCount,omitempty"`

	// The resource group member for a network vlan.
	ResourceGroupMember []Resource_Group_Member `json:"resourceGroupMember,omitempty" xmlrpc:"resourceGroupMember,omitempty"`

	// A count of the resource group member for a network vlan.
	ResourceGroupMemberCount *uint `json:"resourceGroupMemberCount,omitempty" xmlrpc:"resourceGroupMemberCount,omitempty"`

	// The resource groups in which this VLAN is a member.
	ResourceGroups []Resource_Group `json:"resourceGroups,omitempty" xmlrpc:"resourceGroups,omitempty"`

	// A flag indicating that a vlan can be assigned to a host that has SAN disk functionality.
	SanStorageCapabilityFlag *bool `json:"sanStorageCapabilityFlag,omitempty" xmlrpc:"sanStorageCapabilityFlag,omitempty"`

	// A count of collection of scale VLANs this VLAN applies to.
	ScaleVlanCount *uint `json:"scaleVlanCount,omitempty" xmlrpc:"scaleVlanCount,omitempty"`

	// Collection of scale VLANs this VLAN applies to.
	ScaleVlans []Scale_Network_Vlan `json:"scaleVlans,omitempty" xmlrpc:"scaleVlans,omitempty"`

	// The secondary router that a VLAN is associated with. Every SoftLayer VLAN is connected to more than one router for greater network redundancy.
	SecondaryRouter *Hardware `json:"secondaryRouter,omitempty" xmlrpc:"secondaryRouter,omitempty"`

	// A count of the subnets that exist as secondary interfaces on a VLAN
	SecondarySubnetCount *uint `json:"secondarySubnetCount,omitempty" xmlrpc:"secondarySubnetCount,omitempty"`

	// The subnets that exist as secondary interfaces on a VLAN
	SecondarySubnets []Network_Subnet `json:"secondarySubnets,omitempty" xmlrpc:"secondarySubnets,omitempty"`

	// A count of all of the subnets that exist as VLAN interfaces.
	SubnetCount *uint `json:"subnetCount,omitempty" xmlrpc:"subnetCount,omitempty"`

	// All of the subnets that exist as VLAN interfaces.
	Subnets []Network_Subnet `json:"subnets,omitempty" xmlrpc:"subnets,omitempty"`

	// A count of references to all tags for this VLAN.
	TagReferenceCount *uint `json:"tagReferenceCount,omitempty" xmlrpc:"tagReferenceCount,omitempty"`

	// References to all tags for this VLAN.
	TagReferences []Tag_Reference `json:"tagReferences,omitempty" xmlrpc:"tagReferences,omitempty"`

	// The number of primary IP addresses in a VLAN.
	TotalPrimaryIpAddressCount *uint `json:"totalPrimaryIpAddressCount,omitempty" xmlrpc:"totalPrimaryIpAddressCount,omitempty"`

	// The type of this VLAN.
	Type *Network_Vlan_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// A count of all of the Virtual Servers that are connected to a VLAN.
	VirtualGuestCount *uint `json:"virtualGuestCount,omitempty" xmlrpc:"virtualGuestCount,omitempty"`

	// All of the Virtual Servers that are connected to a VLAN.
	VirtualGuests []Virtual_Guest `json:"virtualGuests,omitempty" xmlrpc:"virtualGuests,omitempty"`

	// A VLAN's number as recorded within the SoftLayer network. This is configured directly on SoftLayer's networking equipment and should not be confused with a VLAN's ''id'' property.
	VlanNumber *int `json:"vlanNumber,omitempty" xmlrpc:"vlanNumber,omitempty"`
}

// The SoftLayer_Network_Vlan_Firewall data type contains general information relating to a single SoftLayer VLAN firewall. This is the object which ties the running rules to a specific downstream server. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request.
type Network_Vlan_Firewall struct {
	Entity

	// A flag to indicate if the firewall is in administrative bypass mode. In other words, no rules are being applied to the traffic coming through.
	AdministrativeBypassFlag *string `json:"administrativeBypassFlag,omitempty" xmlrpc:"administrativeBypassFlag,omitempty"`

	// The billing item for a Hardware Firewall (Dedicated).
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// Whether or not this firewall can be directly logged in to.
	CustomerManagedFlag *bool `json:"customerManagedFlag,omitempty" xmlrpc:"customerManagedFlag,omitempty"`

	// The datacenter that the firewall resides in.
	Datacenter *Location `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// The firewall device type.
	FirewallType *string `json:"firewallType,omitempty" xmlrpc:"firewallType,omitempty"`

	// A name reflecting the hostname and domain of the firewall. This is created from the combined values of the firewall's logical name and vlan number automatically, and thus can not be edited directly.
	FullyQualifiedDomainName *string `json:"fullyQualifiedDomainName,omitempty" xmlrpc:"fullyQualifiedDomainName,omitempty"`

	// A firewall's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The credentials to log in to a firewall device. This is only present for dedicated appliances.
	ManagementCredentials *Software_Component_Password `json:"managementCredentials,omitempty" xmlrpc:"managementCredentials,omitempty"`

	// A count of the update requests made for this firewall.
	NetworkFirewallUpdateRequestCount *uint `json:"networkFirewallUpdateRequestCount,omitempty" xmlrpc:"networkFirewallUpdateRequestCount,omitempty"`

	// The update requests made for this firewall.
	NetworkFirewallUpdateRequests []Network_Firewall_Update_Request `json:"networkFirewallUpdateRequests,omitempty" xmlrpc:"networkFirewallUpdateRequests,omitempty"`

	// The VLAN object that a firewall is associated with and protecting.
	NetworkVlan *Network_Vlan `json:"networkVlan,omitempty" xmlrpc:"networkVlan,omitempty"`

	// A count of the VLAN objects that a firewall is associated with and protecting.
	NetworkVlanCount *uint `json:"networkVlanCount,omitempty" xmlrpc:"networkVlanCount,omitempty"`

	// The VLAN objects that a firewall is associated with and protecting.
	NetworkVlans []Network_Vlan `json:"networkVlans,omitempty" xmlrpc:"networkVlans,omitempty"`

	// A firewall's primary IP address. This field will be the IP shown when doing network traces and reverse DNS and is a read-only property.
	PrimaryIpAddress *string `json:"primaryIpAddress,omitempty" xmlrpc:"primaryIpAddress,omitempty"`

	// A count of the currently running rule set of this network component firewall.
	RuleCount *uint `json:"ruleCount,omitempty" xmlrpc:"ruleCount,omitempty"`

	// The currently running rule set of this network component firewall.
	Rules []Network_Vlan_Firewall_Rule `json:"rules,omitempty" xmlrpc:"rules,omitempty"`

	// A count of
	TagReferenceCount *uint `json:"tagReferenceCount,omitempty" xmlrpc:"tagReferenceCount,omitempty"`

	// no documentation yet
	TagReferences []Tag_Reference `json:"tagReferences,omitempty" xmlrpc:"tagReferences,omitempty"`
}

// A SoftLayer_Network_Component_Firewall_Rule object type represents a currently running firewall rule and contains relative information. Use the [[SoftLayer Network Firewall Update Request]] service to submit a firewall update request. Use the [[SoftLayer Network Firewall Template]] service to pull SoftLayer recommended rule set templates.
type Network_Vlan_Firewall_Rule struct {
	Entity

	// The action that the rule is to take [permit or deny].
	Action *string `json:"action,omitempty" xmlrpc:"action,omitempty"`

	// The destination IP address considered for determining rule application.
	DestinationIpAddress *string `json:"destinationIpAddress,omitempty" xmlrpc:"destinationIpAddress,omitempty"`

	// The CIDR is used for determining rule application. This value will
	DestinationIpCidr *int `json:"destinationIpCidr,omitempty" xmlrpc:"destinationIpCidr,omitempty"`

	// The destination IP subnet mask considered for determining rule application.
	DestinationIpSubnetMask *string `json:"destinationIpSubnetMask,omitempty" xmlrpc:"destinationIpSubnetMask,omitempty"`

	// The ending (upper end of range) destination port considered for determining rule application.
	DestinationPortRangeEnd *int `json:"destinationPortRangeEnd,omitempty" xmlrpc:"destinationPortRangeEnd,omitempty"`

	// The starting (lower end of range) destination port considered for determining rule application.
	DestinationPortRangeStart *int `json:"destinationPortRangeStart,omitempty" xmlrpc:"destinationPortRangeStart,omitempty"`

	// The rule's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The network component firewall that this rule belongs to.
	NetworkComponentFirewall *Network_Component_Firewall `json:"networkComponentFirewall,omitempty" xmlrpc:"networkComponentFirewall,omitempty"`

	// The notes field for the rule.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The numeric value describing the order in which the rule should be applied.
	OrderValue *int `json:"orderValue,omitempty" xmlrpc:"orderValue,omitempty"`

	// The protocol considered for determining rule application.
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`

	// The source IP address considered for determining rule application.
	SourceIpAddress *string `json:"sourceIpAddress,omitempty" xmlrpc:"sourceIpAddress,omitempty"`

	// The CIDR is used for determining rule application. This value will
	SourceIpCidr *int `json:"sourceIpCidr,omitempty" xmlrpc:"sourceIpCidr,omitempty"`

	// The source IP subnet mask considered for determining rule application.
	SourceIpSubnetMask *string `json:"sourceIpSubnetMask,omitempty" xmlrpc:"sourceIpSubnetMask,omitempty"`

	// Current status of the network component firewall.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// Whether this rule is an IPv4 rule or an IPv6 rule. If
	Version *int `json:"version,omitempty" xmlrpc:"version,omitempty"`
}

// no documentation yet
type Network_Vlan_Type struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}
