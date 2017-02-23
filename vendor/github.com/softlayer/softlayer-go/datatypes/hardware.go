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

// The SoftLayer_Hardware data type contains general information relating to a single SoftLayer hardware.
type Hardware struct {
	Entity

	// The account associated with a piece of hardware.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A hardware's associated [[SoftLayer_Account|account]] id.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of a piece of hardware's active physical components.
	ActiveComponentCount *uint `json:"activeComponentCount,omitempty" xmlrpc:"activeComponentCount,omitempty"`

	// A piece of hardware's active physical components.
	ActiveComponents []Hardware_Component `json:"activeComponents,omitempty" xmlrpc:"activeComponents,omitempty"`

	// A piece of hardware's active network monitoring incidents.
	ActiveNetworkMonitorIncident []Network_Monitor_Version1_Incident `json:"activeNetworkMonitorIncident,omitempty" xmlrpc:"activeNetworkMonitorIncident,omitempty"`

	// A count of a piece of hardware's active network monitoring incidents.
	ActiveNetworkMonitorIncidentCount *uint `json:"activeNetworkMonitorIncidentCount,omitempty" xmlrpc:"activeNetworkMonitorIncidentCount,omitempty"`

	// A count of
	AllPowerComponentCount *uint `json:"allPowerComponentCount,omitempty" xmlrpc:"allPowerComponentCount,omitempty"`

	// no documentation yet
	AllPowerComponents []Hardware_Power_Component `json:"allPowerComponents,omitempty" xmlrpc:"allPowerComponents,omitempty"`

	// The SoftLayer_Network_Storage_Allowed_Host information to connect this server to Network Storage volumes that require access control lists.
	AllowedHost *Network_Storage_Allowed_Host `json:"allowedHost,omitempty" xmlrpc:"allowedHost,omitempty"`

	// The SoftLayer_Network_Storage objects that this SoftLayer_Hardware has access to.
	AllowedNetworkStorage []Network_Storage `json:"allowedNetworkStorage,omitempty" xmlrpc:"allowedNetworkStorage,omitempty"`

	// A count of the SoftLayer_Network_Storage objects that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageCount *uint `json:"allowedNetworkStorageCount,omitempty" xmlrpc:"allowedNetworkStorageCount,omitempty"`

	// A count of the SoftLayer_Network_Storage objects whose Replica that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageReplicaCount *uint `json:"allowedNetworkStorageReplicaCount,omitempty" xmlrpc:"allowedNetworkStorageReplicaCount,omitempty"`

	// The SoftLayer_Network_Storage objects whose Replica that this SoftLayer_Hardware has access to.
	AllowedNetworkStorageReplicas []Network_Storage `json:"allowedNetworkStorageReplicas,omitempty" xmlrpc:"allowedNetworkStorageReplicas,omitempty"`

	// Information regarding an antivirus/spyware software component object.
	AntivirusSpywareSoftwareComponent *Software_Component `json:"antivirusSpywareSoftwareComponent,omitempty" xmlrpc:"antivirusSpywareSoftwareComponent,omitempty"`

	// A count of information regarding a piece of hardware's specific attributes.
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// Information regarding a piece of hardware's specific attributes.
	Attributes []Hardware_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// The average daily public bandwidth usage for the current billing cycle.
	AverageDailyPublicBandwidthUsage *Float64 `json:"averageDailyPublicBandwidthUsage,omitempty" xmlrpc:"averageDailyPublicBandwidthUsage,omitempty"`

	// A count of a piece of hardware's back-end or private network components.
	BackendNetworkComponentCount *uint `json:"backendNetworkComponentCount,omitempty" xmlrpc:"backendNetworkComponentCount,omitempty"`

	// A piece of hardware's back-end or private network components.
	BackendNetworkComponents []Network_Component `json:"backendNetworkComponents,omitempty" xmlrpc:"backendNetworkComponents,omitempty"`

	// A count of a hardware's backend or private router.
	BackendRouterCount *uint `json:"backendRouterCount,omitempty" xmlrpc:"backendRouterCount,omitempty"`

	// A hardware's backend or private router.
	BackendRouters []Hardware `json:"backendRouters,omitempty" xmlrpc:"backendRouters,omitempty"`

	// A hardware's allotted bandwidth (measured in GB).
	BandwidthAllocation *Float64 `json:"bandwidthAllocation,omitempty" xmlrpc:"bandwidthAllocation,omitempty"`

	// A hardware's allotted detail record. Allotment details link bandwidth allocation with allotments.
	BandwidthAllotmentDetail *Network_Bandwidth_Version1_Allotment_Detail `json:"bandwidthAllotmentDetail,omitempty" xmlrpc:"bandwidthAllotmentDetail,omitempty"`

	// When true, this flag specifies that a hardware is Bare Metal Server. Bare Metal Servers are physical bare metal servers that are billed with the same options as Virtual Servers, with monthly and hourly rates.  Bare Metal instances are ordered based on processor core count and ram amount.
	BareMetalInstanceFlag *int `json:"bareMetalInstanceFlag,omitempty" xmlrpc:"bareMetalInstanceFlag,omitempty"`

	// A count of information regarding a piece of hardware's benchmark certifications.
	BenchmarkCertificationCount *uint `json:"benchmarkCertificationCount,omitempty" xmlrpc:"benchmarkCertificationCount,omitempty"`

	// Information regarding a piece of hardware's benchmark certifications.
	BenchmarkCertifications []Hardware_Benchmark_Certification `json:"benchmarkCertifications,omitempty" xmlrpc:"benchmarkCertifications,omitempty"`

	// Information regarding the billing item for a server.
	BillingItem *Billing_Item_Hardware `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// A flag indicating that a billing item exists.
	BillingItemFlag *bool `json:"billingItemFlag,omitempty" xmlrpc:"billingItemFlag,omitempty"`

	// Determines whether the hardware is ineligible for cancellation because it is disconnected.
	BlockCancelBecauseDisconnectedFlag *bool `json:"blockCancelBecauseDisconnectedFlag,omitempty" xmlrpc:"blockCancelBecauseDisconnectedFlag,omitempty"`

	// Status indicating whether or not a piece of hardware has business continuance insurance.
	BusinessContinuanceInsuranceFlag *bool `json:"businessContinuanceInsuranceFlag,omitempty" xmlrpc:"businessContinuanceInsuranceFlag,omitempty"`

	// A count of a piece of hardware's components.
	ComponentCount *uint `json:"componentCount,omitempty" xmlrpc:"componentCount,omitempty"`

	// A piece of hardware's components.
	Components []Hardware_Component `json:"components,omitempty" xmlrpc:"components,omitempty"`

	// A continuous data protection/server backup software component object.
	ContinuousDataProtectionSoftwareComponent *Software_Component `json:"continuousDataProtectionSoftwareComponent,omitempty" xmlrpc:"continuousDataProtectionSoftwareComponent,omitempty"`

	// The current billable public outbound bandwidth for this hardware for the current billing cycle.
	CurrentBillableBandwidthUsage *Float64 `json:"currentBillableBandwidthUsage,omitempty" xmlrpc:"currentBillableBandwidthUsage,omitempty"`

	// Information regarding the datacenter in which a piece of hardware resides.
	Datacenter *Location `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// The name of the datacenter in which a piece of hardware resides.
	DatacenterName *string `json:"datacenterName,omitempty" xmlrpc:"datacenterName,omitempty"`

	// A piece of hardware's local network domain name.
	Domain *string `json:"domain,omitempty" xmlrpc:"domain,omitempty"`

	// All hardware that has uplink network connections to a piece of hardware.
	DownlinkHardware []Hardware `json:"downlinkHardware,omitempty" xmlrpc:"downlinkHardware,omitempty"`

	// A count of all hardware that has uplink network connections to a piece of hardware.
	DownlinkHardwareCount *uint `json:"downlinkHardwareCount,omitempty" xmlrpc:"downlinkHardwareCount,omitempty"`

	// All hardware that has uplink network connections to a piece of hardware.
	DownlinkNetworkHardware []Hardware `json:"downlinkNetworkHardware,omitempty" xmlrpc:"downlinkNetworkHardware,omitempty"`

	// A count of all hardware that has uplink network connections to a piece of hardware.
	DownlinkNetworkHardwareCount *uint `json:"downlinkNetworkHardwareCount,omitempty" xmlrpc:"downlinkNetworkHardwareCount,omitempty"`

	// A count of information regarding all servers attached to a piece of network hardware.
	DownlinkServerCount *uint `json:"downlinkServerCount,omitempty" xmlrpc:"downlinkServerCount,omitempty"`

	// Information regarding all servers attached to a piece of network hardware.
	DownlinkServers []Hardware `json:"downlinkServers,omitempty" xmlrpc:"downlinkServers,omitempty"`

	// A count of information regarding all virtual guests attached to a piece of network hardware.
	DownlinkVirtualGuestCount *uint `json:"downlinkVirtualGuestCount,omitempty" xmlrpc:"downlinkVirtualGuestCount,omitempty"`

	// Information regarding all virtual guests attached to a piece of network hardware.
	DownlinkVirtualGuests []Virtual_Guest `json:"downlinkVirtualGuests,omitempty" xmlrpc:"downlinkVirtualGuests,omitempty"`

	// A count of all hardware downstream from a network device.
	DownstreamHardwareBindingCount *uint `json:"downstreamHardwareBindingCount,omitempty" xmlrpc:"downstreamHardwareBindingCount,omitempty"`

	// All hardware downstream from a network device.
	DownstreamHardwareBindings []Network_Component_Uplink_Hardware `json:"downstreamHardwareBindings,omitempty" xmlrpc:"downstreamHardwareBindings,omitempty"`

	// All network hardware downstream from the selected piece of hardware.
	DownstreamNetworkHardware []Hardware `json:"downstreamNetworkHardware,omitempty" xmlrpc:"downstreamNetworkHardware,omitempty"`

	// A count of all network hardware downstream from the selected piece of hardware.
	DownstreamNetworkHardwareCount *uint `json:"downstreamNetworkHardwareCount,omitempty" xmlrpc:"downstreamNetworkHardwareCount,omitempty"`

	// A count of all network hardware with monitoring warnings or errors that are downstream from the selected piece of hardware.
	DownstreamNetworkHardwareWithIncidentCount *uint `json:"downstreamNetworkHardwareWithIncidentCount,omitempty" xmlrpc:"downstreamNetworkHardwareWithIncidentCount,omitempty"`

	// All network hardware with monitoring warnings or errors that are downstream from the selected piece of hardware.
	DownstreamNetworkHardwareWithIncidents []Hardware `json:"downstreamNetworkHardwareWithIncidents,omitempty" xmlrpc:"downstreamNetworkHardwareWithIncidents,omitempty"`

	// A count of information regarding all servers attached downstream to a piece of network hardware.
	DownstreamServerCount *uint `json:"downstreamServerCount,omitempty" xmlrpc:"downstreamServerCount,omitempty"`

	// Information regarding all servers attached downstream to a piece of network hardware.
	DownstreamServers []Hardware `json:"downstreamServers,omitempty" xmlrpc:"downstreamServers,omitempty"`

	// A count of information regarding all virtual guests attached to a piece of network hardware.
	DownstreamVirtualGuestCount *uint `json:"downstreamVirtualGuestCount,omitempty" xmlrpc:"downstreamVirtualGuestCount,omitempty"`

	// Information regarding all virtual guests attached to a piece of network hardware.
	DownstreamVirtualGuests []Virtual_Guest `json:"downstreamVirtualGuests,omitempty" xmlrpc:"downstreamVirtualGuests,omitempty"`

	// A count of the drive controllers contained within a piece of hardware.
	DriveControllerCount *uint `json:"driveControllerCount,omitempty" xmlrpc:"driveControllerCount,omitempty"`

	// The drive controllers contained within a piece of hardware.
	DriveControllers []Hardware_Component `json:"driveControllers,omitempty" xmlrpc:"driveControllers,omitempty"`

	// Information regarding a piece of hardware's associated EVault network storage service account.
	EvaultNetworkStorage []Network_Storage `json:"evaultNetworkStorage,omitempty" xmlrpc:"evaultNetworkStorage,omitempty"`

	// A count of information regarding a piece of hardware's associated EVault network storage service account.
	EvaultNetworkStorageCount *uint `json:"evaultNetworkStorageCount,omitempty" xmlrpc:"evaultNetworkStorageCount,omitempty"`

	// Information regarding a piece of hardware's firewall services.
	FirewallServiceComponent *Network_Component_Firewall `json:"firewallServiceComponent,omitempty" xmlrpc:"firewallServiceComponent,omitempty"`

	// Defines the fixed components in a fixed configuration bare metal server.
	FixedConfigurationPreset *Product_Package_Preset `json:"fixedConfigurationPreset,omitempty" xmlrpc:"fixedConfigurationPreset,omitempty"`

	// A count of a piece of hardware's front-end or public network components.
	FrontendNetworkComponentCount *uint `json:"frontendNetworkComponentCount,omitempty" xmlrpc:"frontendNetworkComponentCount,omitempty"`

	// A piece of hardware's front-end or public network components.
	FrontendNetworkComponents []Network_Component `json:"frontendNetworkComponents,omitempty" xmlrpc:"frontendNetworkComponents,omitempty"`

	// A count of a hardware's frontend or public router.
	FrontendRouterCount *uint `json:"frontendRouterCount,omitempty" xmlrpc:"frontendRouterCount,omitempty"`

	// A hardware's frontend or public router.
	FrontendRouters []Hardware `json:"frontendRouters,omitempty" xmlrpc:"frontendRouters,omitempty"`

	// A name reflecting the hostname and domain of the hardware. This is created from the combined values of the hardware's hostname and domain name automatically, and thus should not be edited directly.
	FullyQualifiedDomainName *string `json:"fullyQualifiedDomainName,omitempty" xmlrpc:"fullyQualifiedDomainName,omitempty"`

	// A hardware's universally unique identifier.
	GlobalIdentifier *string `json:"globalIdentifier,omitempty" xmlrpc:"globalIdentifier,omitempty"`

	// A count of the hard drives contained within a piece of hardware.
	HardDriveCount *uint `json:"hardDriveCount,omitempty" xmlrpc:"hardDriveCount,omitempty"`

	// The hard drives contained within a piece of hardware.
	HardDrives []Hardware_Component `json:"hardDrives,omitempty" xmlrpc:"hardDrives,omitempty"`

	// The chassis that a piece of hardware is housed in.
	HardwareChassis *Hardware_Chassis `json:"hardwareChassis,omitempty" xmlrpc:"hardwareChassis,omitempty"`

	// A hardware's function.
	HardwareFunction *Hardware_Function `json:"hardwareFunction,omitempty" xmlrpc:"hardwareFunction,omitempty"`

	// A hardware's function.
	HardwareFunctionDescription *string `json:"hardwareFunctionDescription,omitempty" xmlrpc:"hardwareFunctionDescription,omitempty"`

	// A hardware's status.
	HardwareStatus *Hardware_Status `json:"hardwareStatus,omitempty" xmlrpc:"hardwareStatus,omitempty"`

	// A number reflecting the state of a hardware
	HardwareStatusId *int `json:"hardwareStatusId,omitempty" xmlrpc:"hardwareStatusId,omitempty"`

	// Determine in hardware object has TPM enabled.
	HasTrustedPlatformModuleBillingItemFlag *bool `json:"hasTrustedPlatformModuleBillingItemFlag,omitempty" xmlrpc:"hasTrustedPlatformModuleBillingItemFlag,omitempty"`

	// Information regarding a host IPS software component object.
	HostIpsSoftwareComponent *Software_Component `json:"hostIpsSoftwareComponent,omitempty" xmlrpc:"hostIpsSoftwareComponent,omitempty"`

	// A hardware's hostname
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// A server's hourly billing status.
	HourlyBillingFlag *bool `json:"hourlyBillingFlag,omitempty" xmlrpc:"hourlyBillingFlag,omitempty"`

	// A hardware's internal identification number
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The sum of all the inbound network traffic data for the last 30 days.
	InboundBandwidthUsage *Float64 `json:"inboundBandwidthUsage,omitempty" xmlrpc:"inboundBandwidthUsage,omitempty"`

	// The total public inbound bandwidth for this hardware for the current billing cycle.
	InboundPublicBandwidthUsage *Float64 `json:"inboundPublicBandwidthUsage,omitempty" xmlrpc:"inboundPublicBandwidthUsage,omitempty"`

	// Information regarding the last transaction a server performed.
	LastTransaction *Provisioning_Version1_Transaction `json:"lastTransaction,omitempty" xmlrpc:"lastTransaction,omitempty"`

	// A piece of hardware's latest network monitoring incident.
	LatestNetworkMonitorIncident *Network_Monitor_Version1_Incident `json:"latestNetworkMonitorIncident,omitempty" xmlrpc:"latestNetworkMonitorIncident,omitempty"`

	// Where a piece of hardware is located within SoftLayer's location hierarchy.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	LocationPathString *string `json:"locationPathString,omitempty" xmlrpc:"locationPathString,omitempty"`

	// Information regarding a lockbox account associated with a server.
	LockboxNetworkStorage *Network_Storage `json:"lockboxNetworkStorage,omitempty" xmlrpc:"lockboxNetworkStorage,omitempty"`

	// A flag indicating that the hardware is a managed resource.
	ManagedResourceFlag *bool `json:"managedResourceFlag,omitempty" xmlrpc:"managedResourceFlag,omitempty"`

	// A hardware's serial number that is supplied by the manufacturer.
	ManufacturerSerialNumber *string `json:"manufacturerSerialNumber,omitempty" xmlrpc:"manufacturerSerialNumber,omitempty"`

	// Information regarding a piece of hardware's memory.
	Memory []Hardware_Component `json:"memory,omitempty" xmlrpc:"memory,omitempty"`

	// The amount of memory a piece of hardware has, measured in gigabytes.
	MemoryCapacity *uint `json:"memoryCapacity,omitempty" xmlrpc:"memoryCapacity,omitempty"`

	// A count of information regarding a piece of hardware's memory.
	MemoryCount *uint `json:"memoryCount,omitempty" xmlrpc:"memoryCount,omitempty"`

	// A piece of hardware's metric tracking object.
	MetricTrackingObject *Metric_Tracking_Object_HardwareServer `json:"metricTrackingObject,omitempty" xmlrpc:"metricTrackingObject,omitempty"`

	// A count of information regarding the monitoring agents associated with a piece of hardware.
	MonitoringAgentCount *uint `json:"monitoringAgentCount,omitempty" xmlrpc:"monitoringAgentCount,omitempty"`

	// Information regarding the monitoring agents associated with a piece of hardware.
	MonitoringAgents []Monitoring_Agent `json:"monitoringAgents,omitempty" xmlrpc:"monitoringAgents,omitempty"`

	// Information regarding the hardware's monitoring robot.
	MonitoringRobot *Monitoring_Robot `json:"monitoringRobot,omitempty" xmlrpc:"monitoringRobot,omitempty"`

	// Information regarding a piece of hardware's network monitoring services.
	MonitoringServiceComponent *Network_Monitor_Version1_Query_Host_Stratum `json:"monitoringServiceComponent,omitempty" xmlrpc:"monitoringServiceComponent,omitempty"`

	// The monitoring service flag eligibility status for a piece of hardware.
	MonitoringServiceEligibilityFlag *bool `json:"monitoringServiceEligibilityFlag,omitempty" xmlrpc:"monitoringServiceEligibilityFlag,omitempty"`

	// The service flag status for a piece of hardware.
	MonitoringServiceFlag *bool `json:"monitoringServiceFlag,omitempty" xmlrpc:"monitoringServiceFlag,omitempty"`

	// Information regarding a piece of hardware's motherboard.
	Motherboard *Hardware_Component `json:"motherboard,omitempty" xmlrpc:"motherboard,omitempty"`

	// A count of information regarding a piece of hardware's network cards.
	NetworkCardCount *uint `json:"networkCardCount,omitempty" xmlrpc:"networkCardCount,omitempty"`

	// Information regarding a piece of hardware's network cards.
	NetworkCards []Hardware_Component `json:"networkCards,omitempty" xmlrpc:"networkCards,omitempty"`

	// A count of returns a hardware's network components.
	NetworkComponentCount *uint `json:"networkComponentCount,omitempty" xmlrpc:"networkComponentCount,omitempty"`

	// Returns a hardware's network components.
	NetworkComponents []Network_Component `json:"networkComponents,omitempty" xmlrpc:"networkComponents,omitempty"`

	// The gateway member if this device is part of a network gateway.
	NetworkGatewayMember *Network_Gateway_Member `json:"networkGatewayMember,omitempty" xmlrpc:"networkGatewayMember,omitempty"`

	// Whether or not this device is part of a network gateway.
	NetworkGatewayMemberFlag *bool `json:"networkGatewayMemberFlag,omitempty" xmlrpc:"networkGatewayMemberFlag,omitempty"`

	// A piece of hardware's network management IP address.
	NetworkManagementIpAddress *string `json:"networkManagementIpAddress,omitempty" xmlrpc:"networkManagementIpAddress,omitempty"`

	// All servers with failed monitoring that are attached downstream to a piece of hardware.
	NetworkMonitorAttachedDownHardware []Hardware `json:"networkMonitorAttachedDownHardware,omitempty" xmlrpc:"networkMonitorAttachedDownHardware,omitempty"`

	// A count of all servers with failed monitoring that are attached downstream to a piece of hardware.
	NetworkMonitorAttachedDownHardwareCount *uint `json:"networkMonitorAttachedDownHardwareCount,omitempty" xmlrpc:"networkMonitorAttachedDownHardwareCount,omitempty"`

	// A count of virtual guests that are attached downstream to a hardware that have failed monitoring
	NetworkMonitorAttachedDownVirtualGuestCount *uint `json:"networkMonitorAttachedDownVirtualGuestCount,omitempty" xmlrpc:"networkMonitorAttachedDownVirtualGuestCount,omitempty"`

	// Virtual guests that are attached downstream to a hardware that have failed monitoring
	NetworkMonitorAttachedDownVirtualGuests []Virtual_Guest `json:"networkMonitorAttachedDownVirtualGuests,omitempty" xmlrpc:"networkMonitorAttachedDownVirtualGuests,omitempty"`

	// A count of information regarding a piece of hardware's network monitors.
	NetworkMonitorCount *uint `json:"networkMonitorCount,omitempty" xmlrpc:"networkMonitorCount,omitempty"`

	// A count of the status of all of a piece of hardware's network monitoring incidents.
	NetworkMonitorIncidentCount *uint `json:"networkMonitorIncidentCount,omitempty" xmlrpc:"networkMonitorIncidentCount,omitempty"`

	// The status of all of a piece of hardware's network monitoring incidents.
	NetworkMonitorIncidents []Network_Monitor_Version1_Incident `json:"networkMonitorIncidents,omitempty" xmlrpc:"networkMonitorIncidents,omitempty"`

	// Information regarding a piece of hardware's network monitors.
	NetworkMonitors []Network_Monitor_Version1_Query_Host `json:"networkMonitors,omitempty" xmlrpc:"networkMonitors,omitempty"`

	// The value of a hardware's network status attribute.
	NetworkStatus *string `json:"networkStatus,omitempty" xmlrpc:"networkStatus,omitempty"`

	// The hardware's related network status attribute.
	NetworkStatusAttribute *Hardware_Attribute `json:"networkStatusAttribute,omitempty" xmlrpc:"networkStatusAttribute,omitempty"`

	// Information regarding a piece of hardware's associated network storage service account.
	NetworkStorage []Network_Storage `json:"networkStorage,omitempty" xmlrpc:"networkStorage,omitempty"`

	// A count of information regarding a piece of hardware's associated network storage service account.
	NetworkStorageCount *uint `json:"networkStorageCount,omitempty" xmlrpc:"networkStorageCount,omitempty"`

	// A count of the network virtual LANs (VLANs) associated with a piece of hardware's network components.
	NetworkVlanCount *uint `json:"networkVlanCount,omitempty" xmlrpc:"networkVlanCount,omitempty"`

	// The network virtual LANs (VLANs) associated with a piece of hardware's network components.
	NetworkVlans []Network_Vlan `json:"networkVlans,omitempty" xmlrpc:"networkVlans,omitempty"`

	// A hardware's allotted bandwidth for the next billing cycle (measured in GB).
	NextBillingCycleBandwidthAllocation *Float64 `json:"nextBillingCycleBandwidthAllocation,omitempty" xmlrpc:"nextBillingCycleBandwidthAllocation,omitempty"`

	// A small note about a piece of hardware to use at your discretion.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// no documentation yet
	NotesHistory []Hardware_Note `json:"notesHistory,omitempty" xmlrpc:"notesHistory,omitempty"`

	// A count of
	NotesHistoryCount *uint `json:"notesHistoryCount,omitempty" xmlrpc:"notesHistoryCount,omitempty"`

	// Information regarding a piece of hardware's operating system.
	OperatingSystem *Software_Component_OperatingSystem `json:"operatingSystem,omitempty" xmlrpc:"operatingSystem,omitempty"`

	// A hardware's operating system software description.
	OperatingSystemReferenceCode *string `json:"operatingSystemReferenceCode,omitempty" xmlrpc:"operatingSystemReferenceCode,omitempty"`

	// The sum of all the outbound network traffic data for the last 30 days.
	OutboundBandwidthUsage *Float64 `json:"outboundBandwidthUsage,omitempty" xmlrpc:"outboundBandwidthUsage,omitempty"`

	// The total public outbound bandwidth for this hardware for the current billing cycle.
	OutboundPublicBandwidthUsage *Float64 `json:"outboundPublicBandwidthUsage,omitempty" xmlrpc:"outboundPublicBandwidthUsage,omitempty"`

	// Information regarding the Point of Presence (PoP) location in which a piece of hardware resides.
	PointOfPresenceLocation *Location `json:"pointOfPresenceLocation,omitempty" xmlrpc:"pointOfPresenceLocation,omitempty"`

	// URI of the script to be downloaded and executed after installation is complete.
	PostInstallScriptUri *string `json:"postInstallScriptUri,omitempty" xmlrpc:"postInstallScriptUri,omitempty"`

	// A count of the power components for a hardware object.
	PowerComponentCount *uint `json:"powerComponentCount,omitempty" xmlrpc:"powerComponentCount,omitempty"`

	// The power components for a hardware object.
	PowerComponents []Hardware_Power_Component `json:"powerComponents,omitempty" xmlrpc:"powerComponents,omitempty"`

	// Information regarding a piece of hardware's power supply.
	PowerSupply []Hardware_Component `json:"powerSupply,omitempty" xmlrpc:"powerSupply,omitempty"`

	// A count of information regarding a piece of hardware's power supply.
	PowerSupplyCount *uint `json:"powerSupplyCount,omitempty" xmlrpc:"powerSupplyCount,omitempty"`

	// The hardware's primary private IP address.
	PrimaryBackendIpAddress *string `json:"primaryBackendIpAddress,omitempty" xmlrpc:"primaryBackendIpAddress,omitempty"`

	// Information regarding the hardware's primary back-end network component.
	PrimaryBackendNetworkComponent *Network_Component `json:"primaryBackendNetworkComponent,omitempty" xmlrpc:"primaryBackendNetworkComponent,omitempty"`

	// The hardware's primary public IP address.
	PrimaryIpAddress *string `json:"primaryIpAddress,omitempty" xmlrpc:"primaryIpAddress,omitempty"`

	// Information regarding the hardware's primary public network component.
	PrimaryNetworkComponent *Network_Component `json:"primaryNetworkComponent,omitempty" xmlrpc:"primaryNetworkComponent,omitempty"`

	// Whether the hardware only has access to the private network.
	PrivateNetworkOnlyFlag *bool `json:"privateNetworkOnlyFlag,omitempty" xmlrpc:"privateNetworkOnlyFlag,omitempty"`

	// The total number of processor cores, summed from all processors that are attached to a piece of hardware
	ProcessorCoreAmount *uint `json:"processorCoreAmount,omitempty" xmlrpc:"processorCoreAmount,omitempty"`

	// A count of information regarding a piece of hardware's processors.
	ProcessorCount *uint `json:"processorCount,omitempty" xmlrpc:"processorCount,omitempty"`

	// The total number of physical processor cores, summed from all processors that are attached to a piece of hardware
	ProcessorPhysicalCoreAmount *uint `json:"processorPhysicalCoreAmount,omitempty" xmlrpc:"processorPhysicalCoreAmount,omitempty"`

	// Information regarding a piece of hardware's processors.
	Processors []Hardware_Component `json:"processors,omitempty" xmlrpc:"processors,omitempty"`

	// no documentation yet
	ProvisionDate *Time `json:"provisionDate,omitempty" xmlrpc:"provisionDate,omitempty"`

	// no documentation yet
	Rack *Location `json:"rack,omitempty" xmlrpc:"rack,omitempty"`

	// A count of the RAID controllers contained within a piece of hardware.
	RaidControllerCount *uint `json:"raidControllerCount,omitempty" xmlrpc:"raidControllerCount,omitempty"`

	// The RAID controllers contained within a piece of hardware.
	RaidControllers []Hardware_Component `json:"raidControllers,omitempty" xmlrpc:"raidControllers,omitempty"`

	// A count of recent events that impact this hardware.
	RecentEventCount *uint `json:"recentEventCount,omitempty" xmlrpc:"recentEventCount,omitempty"`

	// Recent events that impact this hardware.
	RecentEvents []Notification_Occurrence_Event `json:"recentEvents,omitempty" xmlrpc:"recentEvents,omitempty"`

	// A count of user credentials to issue commands and/or interact with the server's remote management card.
	RemoteManagementAccountCount *uint `json:"remoteManagementAccountCount,omitempty" xmlrpc:"remoteManagementAccountCount,omitempty"`

	// User credentials to issue commands and/or interact with the server's remote management card.
	RemoteManagementAccounts []Hardware_Component_RemoteManagement_User `json:"remoteManagementAccounts,omitempty" xmlrpc:"remoteManagementAccounts,omitempty"`

	// A hardware's associated remote management component. This is normally IPMI.
	RemoteManagementComponent *Network_Component `json:"remoteManagementComponent,omitempty" xmlrpc:"remoteManagementComponent,omitempty"`

	// A count of the resource groups in which this hardware is a member.
	ResourceGroupCount *uint `json:"resourceGroupCount,omitempty" xmlrpc:"resourceGroupCount,omitempty"`

	// A count of
	ResourceGroupMemberReferenceCount *uint `json:"resourceGroupMemberReferenceCount,omitempty" xmlrpc:"resourceGroupMemberReferenceCount,omitempty"`

	// no documentation yet
	ResourceGroupMemberReferences []Resource_Group_Member `json:"resourceGroupMemberReferences,omitempty" xmlrpc:"resourceGroupMemberReferences,omitempty"`

	// A count of
	ResourceGroupRoleCount *uint `json:"resourceGroupRoleCount,omitempty" xmlrpc:"resourceGroupRoleCount,omitempty"`

	// no documentation yet
	ResourceGroupRoles []Resource_Group_Role `json:"resourceGroupRoles,omitempty" xmlrpc:"resourceGroupRoles,omitempty"`

	// The resource groups in which this hardware is a member.
	ResourceGroups []Resource_Group `json:"resourceGroups,omitempty" xmlrpc:"resourceGroups,omitempty"`

	// A count of a hardware's routers.
	RouterCount *uint `json:"routerCount,omitempty" xmlrpc:"routerCount,omitempty"`

	// A hardware's routers.
	Routers []Hardware `json:"routers,omitempty" xmlrpc:"routers,omitempty"`

	// A count of collection of scale assets this hardware corresponds to.
	ScaleAssetCount *uint `json:"scaleAssetCount,omitempty" xmlrpc:"scaleAssetCount,omitempty"`

	// Collection of scale assets this hardware corresponds to.
	ScaleAssets []Scale_Asset `json:"scaleAssets,omitempty" xmlrpc:"scaleAssets,omitempty"`

	// A count of information regarding a piece of hardware's vulnerability scan requests.
	SecurityScanRequestCount *uint `json:"securityScanRequestCount,omitempty" xmlrpc:"securityScanRequestCount,omitempty"`

	// Information regarding a piece of hardware's vulnerability scan requests.
	SecurityScanRequests []Network_Security_Scanner_Request `json:"securityScanRequests,omitempty" xmlrpc:"securityScanRequests,omitempty"`

	// A hardware's serial number that is supplied by SoftLayer.
	SerialNumber *string `json:"serialNumber,omitempty" xmlrpc:"serialNumber,omitempty"`

	// Information regarding the server room in which the hardware is located.
	ServerRoom *Location `json:"serverRoom,omitempty" xmlrpc:"serverRoom,omitempty"`

	// Information regarding the piece of hardware's service provider.
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`

	// no documentation yet
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`

	// A hardware's internal identification number at its service provider
	ServiceProviderResourceId *int `json:"serviceProviderResourceId,omitempty" xmlrpc:"serviceProviderResourceId,omitempty"`

	// A count of information regarding a piece of hardware's installed software.
	SoftwareComponentCount *uint `json:"softwareComponentCount,omitempty" xmlrpc:"softwareComponentCount,omitempty"`

	// Information regarding a piece of hardware's installed software.
	SoftwareComponents []Software_Component `json:"softwareComponents,omitempty" xmlrpc:"softwareComponents,omitempty"`

	// Information regarding the billing item for a spare pool server.
	SparePoolBillingItem *Billing_Item_Hardware `json:"sparePoolBillingItem,omitempty" xmlrpc:"sparePoolBillingItem,omitempty"`

	// A count of sSH keys to be installed on the server during provisioning or an OS reload.
	SshKeyCount *uint `json:"sshKeyCount,omitempty" xmlrpc:"sshKeyCount,omitempty"`

	// SSH keys to be installed on the server during provisioning or an OS reload.
	SshKeys []Security_Ssh_Key `json:"sshKeys,omitempty" xmlrpc:"sshKeys,omitempty"`

	// A count of
	StorageNetworkComponentCount *uint `json:"storageNetworkComponentCount,omitempty" xmlrpc:"storageNetworkComponentCount,omitempty"`

	// no documentation yet
	StorageNetworkComponents []Network_Component `json:"storageNetworkComponents,omitempty" xmlrpc:"storageNetworkComponents,omitempty"`

	// A count of
	TagReferenceCount *uint `json:"tagReferenceCount,omitempty" xmlrpc:"tagReferenceCount,omitempty"`

	// no documentation yet
	TagReferences []Tag_Reference `json:"tagReferences,omitempty" xmlrpc:"tagReferences,omitempty"`

	// no documentation yet
	TopLevelLocation *Location `json:"topLevelLocation,omitempty" xmlrpc:"topLevelLocation,omitempty"`

	// An account's associated upgrade request object, if any.
	UpgradeRequest *Product_Upgrade_Request `json:"upgradeRequest,omitempty" xmlrpc:"upgradeRequest,omitempty"`

	// The network device connected to a piece of hardware.
	UplinkHardware *Hardware `json:"uplinkHardware,omitempty" xmlrpc:"uplinkHardware,omitempty"`

	// A count of information regarding the network component that is one level higher than a piece of hardware on the network infrastructure.
	UplinkNetworkComponentCount *uint `json:"uplinkNetworkComponentCount,omitempty" xmlrpc:"uplinkNetworkComponentCount,omitempty"`

	// Information regarding the network component that is one level higher than a piece of hardware on the network infrastructure.
	UplinkNetworkComponents []Network_Component `json:"uplinkNetworkComponents,omitempty" xmlrpc:"uplinkNetworkComponents,omitempty"`

	// A string containing custom user data for a hardware order.
	UserData []Hardware_Attribute `json:"userData,omitempty" xmlrpc:"userData,omitempty"`

	// A count of a string containing custom user data for a hardware order.
	UserDataCount *uint `json:"userDataCount,omitempty" xmlrpc:"userDataCount,omitempty"`

	// Information regarding the virtual chassis for a piece of hardware.
	VirtualChassis *Hardware_Group `json:"virtualChassis,omitempty" xmlrpc:"virtualChassis,omitempty"`

	// A count of information regarding the virtual chassis siblings for a piece of hardware.
	VirtualChassisSiblingCount *uint `json:"virtualChassisSiblingCount,omitempty" xmlrpc:"virtualChassisSiblingCount,omitempty"`

	// Information regarding the virtual chassis siblings for a piece of hardware.
	VirtualChassisSiblings []Hardware `json:"virtualChassisSiblings,omitempty" xmlrpc:"virtualChassisSiblings,omitempty"`

	// A piece of hardware's virtual host record.
	VirtualHost *Virtual_Host `json:"virtualHost,omitempty" xmlrpc:"virtualHost,omitempty"`

	// A count of information regarding a piece of hardware's virtual software licenses.
	VirtualLicenseCount *uint `json:"virtualLicenseCount,omitempty" xmlrpc:"virtualLicenseCount,omitempty"`

	// Information regarding a piece of hardware's virtual software licenses.
	VirtualLicenses []Software_VirtualLicense `json:"virtualLicenses,omitempty" xmlrpc:"virtualLicenses,omitempty"`

	// Information regarding the bandwidth allotment to which a piece of hardware belongs.
	VirtualRack *Network_Bandwidth_Version1_Allotment `json:"virtualRack,omitempty" xmlrpc:"virtualRack,omitempty"`

	// The name of the bandwidth allotment belonging to a piece of hardware.
	VirtualRackId *int `json:"virtualRackId,omitempty" xmlrpc:"virtualRackId,omitempty"`

	// The name of the bandwidth allotment belonging to a piece of hardware.
	VirtualRackName *string `json:"virtualRackName,omitempty" xmlrpc:"virtualRackName,omitempty"`

	// A piece of hardware's virtualization platform software.
	VirtualizationPlatform *Software_Component `json:"virtualizationPlatform,omitempty" xmlrpc:"virtualizationPlatform,omitempty"`
}

// The SoftLayer_Hardware_Attribute type contains general information for a hardware attribute. Hardware attributes can be assigned to specific hardware objects to describe relatively arbitrary information.
type Hardware_Attribute struct {
	Entity

	// The type of hardware attribute that this represents.
	HardwareAttributeType *Hardware_Attribute_Type `json:"hardwareAttributeType,omitempty" xmlrpc:"hardwareAttributeType,omitempty"`

	// The unique identifier of a hardware attribute's type.
	HardwareAttributeTypeId *int `json:"hardwareAttributeTypeId,omitempty" xmlrpc:"hardwareAttributeTypeId,omitempty"`

	// A hardware attribute's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A hardware attribute's value.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// Retrieve attributes associated with a hardware object.
type Hardware_Attribute_Type struct {
	Entity

	// The attribute type key name or code.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// The attribute type name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Hardware_Benchmark_Certification data type contains general information relating to a single SoftLayer hardware benchmark certification document.
type Hardware_Benchmark_Certification struct {
	Entity

	// Information regarding a benchmark certification result's associated SoftLayer customer account.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The internal identifier of the SoftLayer customer account associated with a benchmark certification result.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The date that a benchmark certification result was generated.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Information regarding the piece of hardware on which a benchmark certification test was performed.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// A benchmark certification results's associated hardware's internal identification number.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`
}

// Every piece of hardware in SoftLayer's datacenters, including customer servers, are housed in one of many hardware chassis. The SoftLayer_Hardware_Chassis data type defines these chassis.
type Hardware_Chassis struct {
	Entity

	// no documentation yet
	BackplaneCapacity *string `json:"backplaneCapacity,omitempty" xmlrpc:"backplaneCapacity,omitempty"`

	// no documentation yet
	BayCapacity *string `json:"bayCapacity,omitempty" xmlrpc:"bayCapacity,omitempty"`

	// no documentation yet
	BookCapacity *string `json:"bookCapacity,omitempty" xmlrpc:"bookCapacity,omitempty"`

	// no documentation yet
	DriveCapacity *string `json:"driveCapacity,omitempty" xmlrpc:"driveCapacity,omitempty"`

	// no documentation yet
	DriveControllerCapacity *string `json:"driveControllerCapacity,omitempty" xmlrpc:"driveControllerCapacity,omitempty"`

	// A hardware form factor internal identifier.
	FormFactorId *int `json:"formFactorId,omitempty" xmlrpc:"formFactorId,omitempty"`

	// no documentation yet
	GpuCapacity *string `json:"gpuCapacity,omitempty" xmlrpc:"gpuCapacity,omitempty"`

	// A hardware's function.
	HardwareFunction *Hardware_Function `json:"hardwareFunction,omitempty" xmlrpc:"hardwareFunction,omitempty"`

	// A hardware chassis' internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A hardware chassis' manufacturer.
	Manufacturer *string `json:"manufacturer,omitempty" xmlrpc:"manufacturer,omitempty"`

	// A hardware chassis' name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	PowerCapacity *string `json:"powerCapacity,omitempty" xmlrpc:"powerCapacity,omitempty"`

	// The physical size of a hardware chassis.  Currently this relates to the 'U' size of a chassis buy default.
	UnitSize *int `json:"unitSize,omitempty" xmlrpc:"unitSize,omitempty"`

	// A hardware chassis' revision number.
	Version *string `json:"version,omitempty" xmlrpc:"version,omitempty"`
}

// The SoftLayer_Hardware_Component data type abstracts information related to a hardware component.
type Hardware_Component struct {
	Entity

	// A component's capacity.
	Capacity *Float64 `json:"capacity,omitempty" xmlrpc:"capacity,omitempty"`

	// A components sub components. Devices that are usually integrated or in some way attached to a component.
	Children []Hardware_Component `json:"children,omitempty" xmlrpc:"children,omitempty"`

	// A count of a components sub components. Devices that are usually integrated or in some way attached to a component.
	ChildrenCount *uint `json:"childrenCount,omitempty" xmlrpc:"childrenCount,omitempty"`

	// A count of
	DownlinkHardwareComponentCount *uint `json:"downlinkHardwareComponentCount,omitempty" xmlrpc:"downlinkHardwareComponentCount,omitempty"`

	// no documentation yet
	DownlinkHardwareComponents []Hardware_Component `json:"downlinkHardwareComponents,omitempty" xmlrpc:"downlinkHardwareComponents,omitempty"`

	// The hardware object that this component belongs to.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The general group of component models.
	HardwareComponentModel *Hardware_Component_Model `json:"hardwareComponentModel,omitempty" xmlrpc:"hardwareComponentModel,omitempty"`

	// The internal identifier of a hardware component's component model.
	HardwareComponentModelId *int `json:"hardwareComponentModelId,omitempty" xmlrpc:"hardwareComponentModelId,omitempty"`

	// A components type.
	HardwareComponentType *Hardware_Component_Type `json:"hardwareComponentType,omitempty" xmlrpc:"hardwareComponentType,omitempty"`

	// The internal identifier of the hardware that a hardware component resides inside.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// A hardware component's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date that a hardware component was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A count of the module's hardware components
	ModuleComponentCount *uint `json:"moduleComponentCount,omitempty" xmlrpc:"moduleComponentCount,omitempty"`

	// The module's hardware components
	ModuleComponents []Hardware_Component `json:"moduleComponents,omitempty" xmlrpc:"moduleComponents,omitempty"`

	// A count of the module's hardware components
	ModuleHardwareComponentCount *uint `json:"moduleHardwareComponentCount,omitempty" xmlrpc:"moduleHardwareComponentCount,omitempty"`

	// The module's hardware components
	ModuleHardwareComponents []Hardware_Component `json:"moduleHardwareComponents,omitempty" xmlrpc:"moduleHardwareComponents,omitempty"`

	// A count of the module's network components
	ModuleNetworkComponentCount *uint `json:"moduleNetworkComponentCount,omitempty" xmlrpc:"moduleNetworkComponentCount,omitempty"`

	// The module's network components
	ModuleNetworkComponents []Hardware_Component `json:"moduleNetworkComponents,omitempty" xmlrpc:"moduleNetworkComponents,omitempty"`

	// The name of this component as referenced by the operating system.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of the components local ethernet and remote management interfaces
	NetworkComponentCount *uint `json:"networkComponentCount,omitempty" xmlrpc:"networkComponentCount,omitempty"`

	// The components local ethernet and remote management interfaces
	NetworkComponents []Network_Component `json:"networkComponents,omitempty" xmlrpc:"networkComponents,omitempty"`

	// The account this component belongs to.
	Owner *Account `json:"owner,omitempty" xmlrpc:"owner,omitempty"`

	// A components parent. Devices that are usually integrated or in some way attached to a component.
	Parent *Hardware_Component `json:"parent,omitempty" xmlrpc:"parent,omitempty"`

	// no documentation yet
	ParentModule *Hardware_Component `json:"parentModule,omitempty" xmlrpc:"parentModule,omitempty"`

	// no documentation yet
	PrefixAttribute *Hardware_Component_Model_Attribute `json:"prefixAttribute,omitempty" xmlrpc:"prefixAttribute,omitempty"`

	// A RAID controllers RAID mode.
	RaidMode *string `json:"raidMode,omitempty" xmlrpc:"raidMode,omitempty"`

	// The component serial number.
	SerialNumber *string `json:"serialNumber,omitempty" xmlrpc:"serialNumber,omitempty"`

	// no documentation yet
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`

	// A hardware's internal identification number at its service provider
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`

	// A count of
	UplinkHardwareComponentCount *uint `json:"uplinkHardwareComponentCount,omitempty" xmlrpc:"uplinkHardwareComponentCount,omitempty"`

	// no documentation yet
	UplinkHardwareComponents []Hardware_Component `json:"uplinkHardwareComponents,omitempty" xmlrpc:"uplinkHardwareComponents,omitempty"`
}

// The SoftLayer_Hardware_Component_Attribute data type contains general information relating to a single hardware setting or attribute for a component model. For Example: A RAID controller may be setup for many different RAID configurations.  A RAID controller with a configuration of RAID-1 will have a single attribute for this RAID setting.
type Hardware_Component_Attribute struct {
	Entity

	// A hardware component attribute's associated [[SoftLayer_Hardware_Component|Hardware Component]].
	HardwareComponent *Hardware_Component `json:"hardwareComponent,omitempty" xmlrpc:"hardwareComponent,omitempty"`

	// A hardware component attribute's associated [[SoftLayer_Hardware_Component_Attribute_Type|type]].
	HardwareComponentAttributeType *Hardware_Component_Attribute_Type `json:"hardwareComponentAttributeType,omitempty" xmlrpc:"hardwareComponentAttributeType,omitempty"`

	// A hardware component attribute's associated [[SoftLayer_Hardware_Component_Attribute_Type|type]] Id.
	HardwareComponentAttributeTypeId *int `json:"hardwareComponentAttributeTypeId,omitempty" xmlrpc:"hardwareComponentAttributeTypeId,omitempty"`

	// A hardware component attribute's associated [[SoftLayer_Hardware_Component|hardware component]] Id.
	HardwareComponentId *int `json:"hardwareComponentId,omitempty" xmlrpc:"hardwareComponentId,omitempty"`

	// A hardware component attribute's value.  A value can have many different values depending on the attributes [[SoftLayer_Hardware_Component_Attribute_Type|type]].
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The SoftLayer_Hardware_Component_Attribute_Type data type contains general information for the type of an attribute for a hardware component.
type Hardware_Component_Attribute_Type struct {
	Entity

	// The description for the date that a hardware component attribute type's [[SoftLayer_Hardware_Component_Attribute|Attribute]] contains.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A hardware component attribute type's Id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A hardware component attribute type's unique name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A hardware component attribute type's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Hardware_Component_DriveController data type abstracts information related to a drive controller.
type Hardware_Component_DriveController struct {
	Hardware_Component
}

// The SoftLayer_Hardware_Component_HardDrive data type abstracts information related to a hard drive.
type Hardware_Component_HardDrive struct {
	Hardware_Component

	// A count of the attached component partitions.
	PartitionCount *uint `json:"partitionCount,omitempty" xmlrpc:"partitionCount,omitempty"`

	// The attached component partitions.
	Partitions []Hardware_Component_Partition `json:"partitions,omitempty" xmlrpc:"partitions,omitempty"`
}

// The SoftLayer_Hardware_Component_Model data type contains general information relating to a single SoftLayer component model.  A component model represents a vendor specific representation of a hardware component.  Every piece of hardware on a server will have a specific hardware component model.
type Hardware_Component_Model struct {
	Entity

	// no documentation yet
	ArchitectureType *Hardware_Component_Model_Architecture_Type `json:"architectureType,omitempty" xmlrpc:"architectureType,omitempty"`

	// no documentation yet
	ArchitectureTypeId *string `json:"architectureTypeId,omitempty" xmlrpc:"architectureTypeId,omitempty"`

	// A count of
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// no documentation yet
	Attributes []Hardware_Component_Model_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// A component model's capacity. The capacity of a component model depends on the model itself.  For Example: Hard drives have a capacity that reflects the amount of data that hard drive can store.
	Capacity *Float64 `json:"capacity,omitempty" xmlrpc:"capacity,omitempty"`

	// A count of
	CompatibleArrayTypeCount *uint `json:"compatibleArrayTypeCount,omitempty" xmlrpc:"compatibleArrayTypeCount,omitempty"`

	// no documentation yet
	CompatibleArrayTypes []Configuration_Storage_Group_Array_Type `json:"compatibleArrayTypes,omitempty" xmlrpc:"compatibleArrayTypes,omitempty"`

	// A count of all the component models that are compatible with a hardware component model.
	CompatibleChildComponentModelCount *uint `json:"compatibleChildComponentModelCount,omitempty" xmlrpc:"compatibleChildComponentModelCount,omitempty"`

	// All the component models that are compatible with a hardware component model.
	CompatibleChildComponentModels []Hardware_Component_Model `json:"compatibleChildComponentModels,omitempty" xmlrpc:"compatibleChildComponentModels,omitempty"`

	// A count of all the component models that a hardware component model is compatible with.
	CompatibleParentComponentModelCount *uint `json:"compatibleParentComponentModelCount,omitempty" xmlrpc:"compatibleParentComponentModelCount,omitempty"`

	// All the component models that a hardware component model is compatible with.
	CompatibleParentComponentModels []Hardware_Component_Model `json:"compatibleParentComponentModels,omitempty" xmlrpc:"compatibleParentComponentModels,omitempty"`

	// A colon delimited list of hardware component model attributes.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A hardware component model's physical components in inventory.
	HardwareComponents []Hardware_Component `json:"hardwareComponents,omitempty" xmlrpc:"hardwareComponents,omitempty"`

	// The non-vendor specific generic component model for a hardware component model.
	HardwareGenericComponentModel *Hardware_Component_Model_Generic `json:"hardwareGenericComponentModel,omitempty" xmlrpc:"hardwareGenericComponentModel,omitempty"`

	// The internal identifier of the generic component model for a component model.
	HardwareGenericComponentModelId *int `json:"hardwareGenericComponentModelId,omitempty" xmlrpc:"hardwareGenericComponentModelId,omitempty"`

	// A hardware component model's internal identifier number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	InfinibandCompatibleAttribute *Hardware_Component_Model_Attribute `json:"infinibandCompatibleAttribute,omitempty" xmlrpc:"infinibandCompatibleAttribute,omitempty"`

	// no documentation yet
	IsInfinibandCompatible *bool `json:"isInfinibandCompatible,omitempty" xmlrpc:"isInfinibandCompatible,omitempty"`

	// no documentation yet
	LongDescription *string `json:"longDescription,omitempty" xmlrpc:"longDescription,omitempty"`

	// A hardware component model's manufacturer.
	Manufacturer *string `json:"manufacturer,omitempty" xmlrpc:"manufacturer,omitempty"`

	// The model name of a hardware component model.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A motherboard's average reboot time.
	RebootTime *Hardware_Component_Motherboard_Reboot_Time `json:"rebootTime,omitempty" xmlrpc:"rebootTime,omitempty"`

	// A hardware component model's type.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// A count of the types of attributes that are allowed for a given hardware component model.
	ValidAttributeTypeCount *uint `json:"validAttributeTypeCount,omitempty" xmlrpc:"validAttributeTypeCount,omitempty"`

	// The types of attributes that are allowed for a given hardware component model.
	ValidAttributeTypes []Hardware_Component_Model_Attribute_Type `json:"validAttributeTypes,omitempty" xmlrpc:"validAttributeTypes,omitempty"`

	// The model number or model description of a hardware component model.
	Version *string `json:"version,omitempty" xmlrpc:"version,omitempty"`
}

// no documentation yet
type Hardware_Component_Model_Architecture_Type struct {
	Entity

	// no documentation yet
	Children []Hardware_Component_Model_Architecture_Type `json:"children,omitempty" xmlrpc:"children,omitempty"`

	// A count of
	ChildrenCount *uint `json:"childrenCount,omitempty" xmlrpc:"childrenCount,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Parent *Hardware_Component_Model_Architecture_Type `json:"parent,omitempty" xmlrpc:"parent,omitempty"`

	// no documentation yet
	ParentId *string `json:"parentId,omitempty" xmlrpc:"parentId,omitempty"`
}

// The SoftLayer_Hardware_Component__Model_Attribute data type contains general information relating to a single hardware setting or attribute for a component model.
type Hardware_Component_Model_Attribute struct {
	Entity

	// A hardware component model attribute's associated [[SoftLayer_Hardware_Component_Model_Attribute_Type|type]] Id.
	AttributeTypeId *int `json:"attributeTypeId,omitempty" xmlrpc:"attributeTypeId,omitempty"`

	// no documentation yet
	HardwareComponent *Hardware_Component_Model `json:"hardwareComponent,omitempty" xmlrpc:"hardwareComponent,omitempty"`

	// no documentation yet
	HardwareComponentAttributeType *Hardware_Component_Model_Attribute_Type `json:"hardwareComponentAttributeType,omitempty" xmlrpc:"hardwareComponentAttributeType,omitempty"`

	// A hardware component model attribute's associated [[SoftLayer_Hardware_Component_Model|hardware component model]] Id.
	HardwareComponentModelId *int `json:"hardwareComponentModelId,omitempty" xmlrpc:"hardwareComponentModelId,omitempty"`

	// A hardware component model attribute's value.  A value can have many different values depending on the attributes [[SoftLayer_Hardware_Component_Model_Attribute_Type|type]].
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The SoftLayer_Hardware_Component_Model_Attribute_Type data type contains general information for the type of an attribute for a hardware component model.
type Hardware_Component_Model_Attribute_Type struct {
	Entity

	// The description for the data that a hardware component model type's [[SoftLayer_Hardware_Component_Model_Attribute|Attribute]] contains.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A hardware component model attribute type's Id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A hardware component model attribute type's unique name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A hardware component model attribute type's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of
	ValidComponentTypeCount *uint `json:"validComponentTypeCount,omitempty" xmlrpc:"validComponentTypeCount,omitempty"`

	// no documentation yet
	ValidComponentTypes []Hardware_Component_Type `json:"validComponentTypes,omitempty" xmlrpc:"validComponentTypes,omitempty"`
}

// The SoftLayer_Hardware_Component_Model_Generic data type contains general information relating to a single SoftLayer generic component model.  A generic component model represents a non-vendor specific representation of a hardware component.  Frequently SoftLayer utilizes components from various vendors in the servers they provision. For Example: Several different vendors produce 6GB DDR2 memory.  The generic component model for the 6GB stick of RAM encompasses every instance of this component regardless of make and model.
type Hardware_Component_Model_Generic struct {
	Entity

	// A generic component model's capacity. The capacity of a generic component model depends on the model itself.  For Example: Hard drives have a capacity that reflects the amount of data that hard drive can store.
	Capacity *Float64 `json:"capacity,omitempty" xmlrpc:"capacity,omitempty"`

	// A brief description for a generic component model that typically defines it's function.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A count of a generic component model's hardware component model.
	HardwareComponentModelCount *uint `json:"hardwareComponentModelCount,omitempty" xmlrpc:"hardwareComponentModelCount,omitempty"`

	// A generic component model's hardware component model.
	HardwareComponentModels []Hardware_Component_Model `json:"hardwareComponentModels,omitempty" xmlrpc:"hardwareComponentModels,omitempty"`

	// A generic component model's type.
	HardwareComponentType *Hardware_Component_Type `json:"hardwareComponentType,omitempty" xmlrpc:"hardwareComponentType,omitempty"`

	// The internal identifier of the component type for a generic component model.
	HardwareComponentTypeId *int `json:"hardwareComponentTypeId,omitempty" xmlrpc:"hardwareComponentTypeId,omitempty"`

	// A generic component model's internal identification number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A list of features that a generic component model can provide.
	MarketingFeatures *Hardware_Component_Model_Generic_MarketingFeature `json:"marketingFeatures,omitempty" xmlrpc:"marketingFeatures,omitempty"`

	// The unit of measurement for the capacity of a generic component model.
	Units *string `json:"units,omitempty" xmlrpc:"units,omitempty"`

	// A generic component model's upgrade priority. The upgrade priority indicates the order a generic component model should be considered over other generic component models. A higher number indicates that a generic component model receives a higher upgrade preference in comparison to a generic component model with a lower priority number.
	UpgradePriority *int `json:"upgradePriority,omitempty" xmlrpc:"upgradePriority,omitempty"`
}

// The SoftLayer_Hardware_Component_Model_Generic_Attribute data type contains information relating to a single SoftLayer generic component model.  Generic component model attributes can hold any information to describe functionality of the model. For Example: The number of cores that a processor has.
type Hardware_Component_Model_Generic_Attribute struct {
	Entity

	// An attributes generic component model.
	HardwareGenericComponentModel *Hardware_Component_Model_Generic `json:"hardwareGenericComponentModel,omitempty" xmlrpc:"hardwareGenericComponentModel,omitempty"`

	// A generic component model attribute's value.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The SoftLayer_Hardware_Component_Model_Generic_MarketingFeature data type contains general information relating to all the advertising features of a single SoftLayer hardware generic component model.
type Hardware_Component_Model_Generic_MarketingFeature struct {
	Entity

	// An html formatted list of all features.
	Features *string `json:"features,omitempty" xmlrpc:"features,omitempty"`

	// The generic component model for a list of advertising or marketing features
	HardwareGenericComponentModel *Hardware_Component_Model_Generic `json:"hardwareGenericComponentModel,omitempty" xmlrpc:"hardwareGenericComponentModel,omitempty"`

	// A hardware component's upgrade price.
	Price *string `json:"price,omitempty" xmlrpc:"price,omitempty"`
}

// The SoftLayer_Hardware_Component_DriveController data type abstracts information related to a motherboard.
type Hardware_Component_Motherboard struct {
	Hardware_Component
}

// The SoftLayer_Hardware_Component_Motherboard_Reboot_Time contains the average reboot times for motherboards. There are two types of average times.  One is for motherboards without raid, and the other is for motherboards with raid.  These times are based on averages and have been gathered through numerous test cases.
type Hardware_Component_Motherboard_Reboot_Time struct {
	Entity

	// Motherboard's specifications (manufacturer, version, etc....)
	HardwareComponentModel *Hardware_Component_Model `json:"hardwareComponentModel,omitempty" xmlrpc:"hardwareComponentModel,omitempty"`

	// Average reboot time in seconds for the motherboard when raid is installed.
	WithRaid *int `json:"withRaid,omitempty" xmlrpc:"withRaid,omitempty"`

	// Average reboot time in seconds for the motherboard when NO raid is installed.
	WithoutRaid *int `json:"withoutRaid,omitempty" xmlrpc:"withoutRaid,omitempty"`
}

// The SoftLayer_Hardware_Component_NetworkCard data type abstracts information related to a network card.
type Hardware_Component_NetworkCard struct {
	Hardware_Component
}

// The SoftLayer_Hardware_Component_Partition data type contains general information relating to a single hard drive partition.
type Hardware_Component_Partition struct {
	Entity

	// A hardware component partition's order in the [[SoftLayer_Hardware_Hardware|hardware]].
	DiskNumber *int `json:"diskNumber,omitempty" xmlrpc:"diskNumber,omitempty"`

	// A flag indicating if a partition is the grow partition. The grow partition will grow to fill all remaining space on a disk.  There can only be one.
	Grow *int `json:"grow,omitempty" xmlrpc:"grow,omitempty"`

	// A hardware component partitions's associated [[SoftLayer_Hardware_Component|Hardware Component]]. Likely to be a [[SoftLayer_Hardware_Component_HardDrive|Hard Drive]]
	HardwareComponent *Hardware_Component `json:"hardwareComponent,omitempty" xmlrpc:"hardwareComponent,omitempty"`

	// A hardware component partition's associated [[SoftLayer_Hardware_Component|hardware component]] Id.
	HardwareComponentId *int `json:"hardwareComponentId,omitempty" xmlrpc:"hardwareComponentId,omitempty"`

	// A hardware component partition's minimum size(GB).
	MinimumSize *Float64 `json:"minimumSize,omitempty" xmlrpc:"minimumSize,omitempty"`

	// A hardware component partition's name. On a server with windows this may be 'C' and on Linux this may be '/var'
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Hardware_Component_Partition_OperatingSystem data type contains general information relating to a single SoftLayer Operating System Partition Template.
type Hardware_Component_Partition_OperatingSystem struct {
	Entity

	// A partition template operating system's description.  Typically the title of the Operating System.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A partition template operating system's id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Information about the kinds of partition templates assigned to this operating system.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// A count of information regarding an operating system's [[SoftLayer_Hardware_Component_Partition_Template|Partition Templates]].
	PartitionTemplateCount *uint `json:"partitionTemplateCount,omitempty" xmlrpc:"partitionTemplateCount,omitempty"`

	// Information regarding an operating system's [[SoftLayer_Hardware_Component_Partition_Template|Partition Templates]].
	PartitionTemplates []Hardware_Component_Partition_Template `json:"partitionTemplates,omitempty" xmlrpc:"partitionTemplates,omitempty"`
}

// The SoftLayer_Hardware_Component_Partition_Template data type contains general information relating to a single SoftLayer partition template.  Partition templates group 1 or more partition configurations that can be used to predefine how a hard drive's partitions will be configured.
type Hardware_Component_Partition_Template struct {
	Entity

	// A partition template's associated [[SoftLayer_Account|Account]].
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A partition template's owner. The [[SoftLayer_Account|Account]] that a template was created by.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// An individual partition for a partition template. This is identical to 'partitionTemplatePartition' except this will sort unix partitions.
	Data []Hardware_Component_Partition_Template_Partition `json:"data,omitempty" xmlrpc:"data,omitempty"`

	// A count of an individual partition for a partition template. This is identical to 'partitionTemplatePartition' except this will sort unix partitions.
	DataCount *uint `json:"dataCount,omitempty" xmlrpc:"dataCount,omitempty"`

	// A partition template's description.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	ExpireDate *string `json:"expireDate,omitempty" xmlrpc:"expireDate,omitempty"`

	// A partition template's id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A partition template's associated [[SoftLayer_Hardware_Component_Partition_OperatingSystem|Operating System]].
	PartitionOperatingSystem *Hardware_Component_Partition_OperatingSystem `json:"partitionOperatingSystem,omitempty" xmlrpc:"partitionOperatingSystem,omitempty"`

	// A partition template's associated [[SoftLayer_Hardware_Component_Partition_OperatingSystem|Operating System]] Id.
	PartitionOperatingSystemId *int `json:"partitionOperatingSystemId,omitempty" xmlrpc:"partitionOperatingSystemId,omitempty"`

	// An individual partition for a partition template.
	PartitionTemplatePartition []Hardware_Component_Partition_Template_Partition `json:"partitionTemplatePartition,omitempty" xmlrpc:"partitionTemplatePartition,omitempty"`

	// A count of an individual partition for a partition template.
	PartitionTemplatePartitionCount *uint `json:"partitionTemplatePartitionCount,omitempty" xmlrpc:"partitionTemplatePartitionCount,omitempty"`

	// A partition template's status code. ACTIVE ,INACTIVE.
	StatusCode *string `json:"statusCode,omitempty" xmlrpc:"statusCode,omitempty"`

	// A partition template's Type. SYSTEM - template generated by softlayer.  CUSTOM - templates generated by SoftLayer customers.
	TemplateType *string `json:"templateType,omitempty" xmlrpc:"templateType,omitempty"`
}

// The SoftLayer_Hardware_Component_Partition_Template_Partition data type contains general information relating to a single SoftLayer Template Partition.
type Hardware_Component_Partition_Template_Partition struct {
	Entity

	// The filesystem type of a partition
	FilesystemType *Configuration_Storage_Filesystem_Type `json:"filesystemType,omitempty" xmlrpc:"filesystemType,omitempty"`

	// A partition's id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A flag indication if a partition will be the grow partition.  The grow partition will have its size adjusted to fill all available space on a hard drive.
	IsGrow *bool `json:"isGrow,omitempty" xmlrpc:"isGrow,omitempty"`

	// A partition's default name.
	PartitionName *string `json:"partitionName,omitempty" xmlrpc:"partitionName,omitempty"`

	// A partition's default size.
	PartitionSize *Float64 `json:"partitionSize,omitempty" xmlrpc:"partitionSize,omitempty"`

	// A partition's [[SoftLayer_Hardware_Component_Partition_Template|Partition Template]].
	PartitionTemplate *Hardware_Component_Partition_Template `json:"partitionTemplate,omitempty" xmlrpc:"partitionTemplate,omitempty"`

	// A partition's associated [[SoftLayer_Hardware_Component_Partition_Template|Partition Template]] Id.
	PartitionTemplateId *int `json:"partitionTemplateId,omitempty" xmlrpc:"partitionTemplateId,omitempty"`
}

// The SoftLayer_Hardware_Component_Processor data type abstracts information related to a processor.
type Hardware_Component_Processor struct {
	Hardware_Component
}

// The SoftLayer_Hardware_Component_Ram data type abstracts information related to RAM.
type Hardware_Component_Ram struct {
	Hardware_Component
}

// This class adds functionality to the base SoftLayer_Hardware class for web servers (all server hardware)
type Hardware_Component_RemoteManagement struct {
	Hardware_Component

	// A network component data type.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`
}

// The SoftLayer_Network_Storage_Evault_Version6 contains the names of the remote management commands.  Currently, only the reboot and power commands for the remote management card exist.
type Hardware_Component_RemoteManagement_Command struct {
	Entity

	// The name of the remote management command.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A count of all requests issued for the remote management command.
	RequestCount *uint `json:"requestCount,omitempty" xmlrpc:"requestCount,omitempty"`

	// All requests issued for the remote management command.
	Requests []Hardware_Component_RemoteManagement_Command_Request `json:"requests,omitempty" xmlrpc:"requests,omitempty"`
}

// The SoftLayer_Hardware_Component_RemoteManagement_Command_Request contains details for remote management commands issued to a server's remote management card.  Details for remote management commands such as powerOn, powerOff, powerCycle, rebootDefault, rebootSoft, rebootHard can be retrieved.  Details such as the user who issued the command, the id of the remote management card the command was issued, when the command was issued may be retrieved.
type Hardware_Component_RemoteManagement_Command_Request struct {
	Entity

	// The timestamp the remote management command was issued.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The id of the hardware to perform the remote management or powerstrip command on.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The hardware id the command was issued for.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// The timestamp recorded when the remote management command returned a status of the command issued.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A hardware's network components. Network components are hardware components such as IPMI cards or Ethernet cards.
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`

	// Execution status of the remote management command.  True is successful.  False is failure.
	Processed *bool `json:"processed,omitempty" xmlrpc:"processed,omitempty"`

	// The remote management command issued.
	RemoteManagementCommand *Hardware_Component_RemoteManagement_Command `json:"remoteManagementCommand,omitempty" xmlrpc:"remoteManagementCommand,omitempty"`

	// Information regarding the user who issued the remote management command.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`
}

// The credentials used for remote management such as username, password, etc...
type Hardware_Component_RemoteManagement_User struct {
	Entity

	// no documentation yet
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// no documentation yet
	NetworkComponent *Network_Component `json:"networkComponent,omitempty" xmlrpc:"networkComponent,omitempty"`

	// The password used for this remote management command.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// The username used for this remote management command.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// The SoftLayer_Hardware_Component_SecurityDevice is used to determine the security devices attached to the hardware component.
type Hardware_Component_SecurityDevice struct {
	Hardware_Component
}

// The SoftLayer_Hardware_Component_SecurityDevice_Infineon is used to determine the Infineon security device attached to the hardware component.
type Hardware_Component_SecurityDevice_Infineon struct {
	Hardware_Component_SecurityDevice
}

// The SoftLayer_Hardware_Component_Type data type provides details on the type of component requested
type Hardware_Component_Type struct {
	Entity

	// A count of the generic component model description for this component type object.
	HardwareGenericComponentModelCount *uint `json:"hardwareGenericComponentModelCount,omitempty" xmlrpc:"hardwareGenericComponentModelCount,omitempty"`

	// The generic component model description for this component type object.
	HardwareGenericComponentModels []Hardware_Component_Model_Generic `json:"hardwareGenericComponentModels,omitempty" xmlrpc:"hardwareGenericComponentModels,omitempty"`

	// The ID associated with this component type.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The hardware component type key name or code.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The type associated with this component type.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The parent generic component model object for this generic component model object.
	TypeParent *Hardware_Component_Type `json:"typeParent,omitempty" xmlrpc:"typeParent,omitempty"`

	// The parent id associated with this component type.
	TypeParentId *int `json:"typeParentId,omitempty" xmlrpc:"typeParentId,omitempty"`
}

// The SoftLayer_Hardware_Firewall data type contains general information relating to a single SoftLayer firewall.
type Hardware_Firewall struct {
	Hardware_Switch

	// A count of a list of users that have access to this hardware firewall.
	UserCount *uint `json:"userCount,omitempty" xmlrpc:"userCount,omitempty"`

	// A list of users that have access to this hardware firewall.
	Users []User_Customer `json:"users,omitempty" xmlrpc:"users,omitempty"`
}

// The SoftLayer_Hardware_Function data type contains a generic object type for a piece of hardware, like switch, firewall, server, etc..
type Hardware_Function struct {
	Entity

	// The code associated with this hardware function.
	Code *string `json:"code,omitempty" xmlrpc:"code,omitempty"`

	// The description for a hardware function.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The id associated with a hardware function.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`
}

// no documentation yet
type Hardware_Group struct {
	Entity

	// no documentation yet
	Domain *string `json:"domain,omitempty" xmlrpc:"domain,omitempty"`

	// A count of all servers attached to a network hardware.
	DownlinkServerCount *uint `json:"downlinkServerCount,omitempty" xmlrpc:"downlinkServerCount,omitempty"`

	// All servers attached to a network hardware.
	DownlinkServers []Hardware `json:"downlinkServers,omitempty" xmlrpc:"downlinkServers,omitempty"`

	// A count of all virtual guests attached to a network hardware.
	DownlinkVirtualGuestCount *uint `json:"downlinkVirtualGuestCount,omitempty" xmlrpc:"downlinkVirtualGuestCount,omitempty"`

	// All virtual guests attached to a network hardware.
	DownlinkVirtualGuests []Virtual_Guest `json:"downlinkVirtualGuests,omitempty" xmlrpc:"downlinkVirtualGuests,omitempty"`

	// All network hardware downstream from this hardware.
	DownstreamNetworkHardware []Hardware `json:"downstreamNetworkHardware,omitempty" xmlrpc:"downstreamNetworkHardware,omitempty"`

	// A count of all network hardware downstream from this hardware.
	DownstreamNetworkHardwareCount *uint `json:"downstreamNetworkHardwareCount,omitempty" xmlrpc:"downstreamNetworkHardwareCount,omitempty"`

	// A count of all network hardware with monitoring warnings or errors downstream from this hardware.
	DownstreamNetworkHardwareWithIncidentCount *uint `json:"downstreamNetworkHardwareWithIncidentCount,omitempty" xmlrpc:"downstreamNetworkHardwareWithIncidentCount,omitempty"`

	// All network hardware with monitoring warnings or errors downstream from this hardware.
	DownstreamNetworkHardwareWithIncidents []Hardware `json:"downstreamNetworkHardwareWithIncidents,omitempty" xmlrpc:"downstreamNetworkHardwareWithIncidents,omitempty"`

	// The chassis that a piece of hardware is housed in.
	HardwareChassis *Hardware_Chassis `json:"hardwareChassis,omitempty" xmlrpc:"hardwareChassis,omitempty"`

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// All servers attached downstream to a hardware that have failed monitoring
	NetworkMonitorAttachedDownHardware []Hardware `json:"networkMonitorAttachedDownHardware,omitempty" xmlrpc:"networkMonitorAttachedDownHardware,omitempty"`

	// A count of all servers attached downstream to a hardware that have failed monitoring
	NetworkMonitorAttachedDownHardwareCount *uint `json:"networkMonitorAttachedDownHardwareCount,omitempty" xmlrpc:"networkMonitorAttachedDownHardwareCount,omitempty"`

	// A count of virtual guests that are attached downstream to a hardware that have failed monitoring
	NetworkMonitorAttachedDownVirtualGuestCount *uint `json:"networkMonitorAttachedDownVirtualGuestCount,omitempty" xmlrpc:"networkMonitorAttachedDownVirtualGuestCount,omitempty"`

	// Virtual guests that are attached downstream to a hardware that have failed monitoring
	NetworkMonitorAttachedDownVirtualGuests []Virtual_Guest `json:"networkMonitorAttachedDownVirtualGuests,omitempty" xmlrpc:"networkMonitorAttachedDownVirtualGuests,omitempty"`

	// The value of a hardware's network status attribute.
	NetworkStatus *string `json:"networkStatus,omitempty" xmlrpc:"networkStatus,omitempty"`
}

// no documentation yet
type Hardware_LoadBalancer struct {
	Hardware

	// no documentation yet
	ModelFamily *string `json:"modelFamily,omitempty" xmlrpc:"modelFamily,omitempty"`

	// A count of a list of users that have access to this hardware load balancer.
	UserCount *uint `json:"userCount,omitempty" xmlrpc:"userCount,omitempty"`

	// A list of users that have access to this hardware load balancer.
	Users []User_Customer `json:"users,omitempty" xmlrpc:"users,omitempty"`
}

// no documentation yet
type Hardware_Note struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Employee *User_Employee `json:"employee,omitempty" xmlrpc:"employee,omitempty"`

	// no documentation yet
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// no documentation yet
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	// no documentation yet
	Type *Hardware_Note_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// no documentation yet
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// no documentation yet
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// no documentation yet
	UserRecordId *int `json:"userRecordId,omitempty" xmlrpc:"userRecordId,omitempty"`
}

// no documentation yet
type Hardware_Note_Type struct {
	Entity

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`
}

// no documentation yet
type Hardware_Power_Component struct {
	Entity

	// no documentation yet
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// no documentation yet
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`
}

// The SoftLayer_Hardware_Router data type contains general information relating to a single SoftLayer router.
type Hardware_Router struct {
	Hardware_Switch

	// A count of associated subnets for a router object.
	BoundSubnetCount *uint `json:"boundSubnetCount,omitempty" xmlrpc:"boundSubnetCount,omitempty"`

	// Associated subnets for a router object.
	BoundSubnets []Network_Subnet `json:"boundSubnets,omitempty" xmlrpc:"boundSubnets,omitempty"`

	// A flag indicating that a VLAN on the router can be assigned to a host that has local disk functionality.
	LocalDiskStorageCapabilityFlag *bool `json:"localDiskStorageCapabilityFlag,omitempty" xmlrpc:"localDiskStorageCapabilityFlag,omitempty"`

	// A flag indicating that a VLAN on the router can be assigned to a host that has SAN disk functionality.
	SanStorageCapabilityFlag *bool `json:"sanStorageCapabilityFlag,omitempty" xmlrpc:"sanStorageCapabilityFlag,omitempty"`
}

// The SoftLayer_Hardware_Router_Backend data type contains general information relating to a single SoftLayer router item for hardware.
type Hardware_Router_Backend struct {
	Hardware_Router
}

// The SoftLayer_Hardware_Router_Frontend data type contains general information relating to a single SoftLayer router item for hardware.
type Hardware_Router_Frontend struct {
	Hardware_Router
}

// no documentation yet
type Hardware_SecurityModule struct {
	Hardware_Server
}

// The SoftLayer_Hardware_Server data type contains general information relating to a single SoftLayer server.
type Hardware_Server struct {
	Hardware

	// The billing item for a server's attached network firewall.
	ActiveNetworkFirewallBillingItem *Billing_Item `json:"activeNetworkFirewallBillingItem,omitempty" xmlrpc:"activeNetworkFirewallBillingItem,omitempty"`

	// A count of
	ActiveTicketCount *uint `json:"activeTicketCount,omitempty" xmlrpc:"activeTicketCount,omitempty"`

	// no documentation yet
	ActiveTickets []Ticket `json:"activeTickets,omitempty" xmlrpc:"activeTickets,omitempty"`

	// Transaction currently running for server.
	ActiveTransaction *Provisioning_Version1_Transaction `json:"activeTransaction,omitempty" xmlrpc:"activeTransaction,omitempty"`

	// A count of any active transaction(s) that are currently running for the server (example: os reload).
	ActiveTransactionCount *uint `json:"activeTransactionCount,omitempty" xmlrpc:"activeTransactionCount,omitempty"`

	// Any active transaction(s) that are currently running for the server (example: os reload).
	ActiveTransactions []Provisioning_Version1_Transaction `json:"activeTransactions,omitempty" xmlrpc:"activeTransactions,omitempty"`

	// An object that stores the maximum level for the monitoring query types and response types.
	AvailableMonitoring []Network_Monitor_Version1_Query_Host_Stratum `json:"availableMonitoring,omitempty" xmlrpc:"availableMonitoring,omitempty"`

	// A count of an object that stores the maximum level for the monitoring query types and response types.
	AvailableMonitoringCount *uint `json:"availableMonitoringCount,omitempty" xmlrpc:"availableMonitoringCount,omitempty"`

	// The average daily total bandwidth usage for the current billing cycle.
	AverageDailyBandwidthUsage *Float64 `json:"averageDailyBandwidthUsage,omitempty" xmlrpc:"averageDailyBandwidthUsage,omitempty"`

	// The average daily private bandwidth usage for the current billing cycle.
	AverageDailyPrivateBandwidthUsage *Float64 `json:"averageDailyPrivateBandwidthUsage,omitempty" xmlrpc:"averageDailyPrivateBandwidthUsage,omitempty"`

	// The raw bandwidth usage data for the current billing cycle. One object will be returned for each network this server is attached to.
	BillingCycleBandwidthUsage []Network_Bandwidth_Usage `json:"billingCycleBandwidthUsage,omitempty" xmlrpc:"billingCycleBandwidthUsage,omitempty"`

	// A count of the raw bandwidth usage data for the current billing cycle. One object will be returned for each network this server is attached to.
	BillingCycleBandwidthUsageCount *uint `json:"billingCycleBandwidthUsageCount,omitempty" xmlrpc:"billingCycleBandwidthUsageCount,omitempty"`

	// The raw private bandwidth usage data for the current billing cycle.
	BillingCyclePrivateBandwidthUsage *Network_Bandwidth_Usage `json:"billingCyclePrivateBandwidthUsage,omitempty" xmlrpc:"billingCyclePrivateBandwidthUsage,omitempty"`

	// The raw public bandwidth usage data for the current billing cycle.
	BillingCyclePublicBandwidthUsage *Network_Bandwidth_Usage `json:"billingCyclePublicBandwidthUsage,omitempty" xmlrpc:"billingCyclePublicBandwidthUsage,omitempty"`

	// no documentation yet
	ContainsSolidStateDrivesFlag *bool `json:"containsSolidStateDrivesFlag,omitempty" xmlrpc:"containsSolidStateDrivesFlag,omitempty"`

	// A server's control panel.
	ControlPanel *Software_Component_ControlPanel `json:"controlPanel,omitempty" xmlrpc:"controlPanel,omitempty"`

	// The total cost of a server, measured in US Dollars ($USD).
	Cost *Float64 `json:"cost,omitempty" xmlrpc:"cost,omitempty"`

	// An object that provides commonly used bandwidth summary components for the current billing cycle.
	CurrentBandwidthSummary *Metric_Tracking_Object_Bandwidth_Summary `json:"currentBandwidthSummary,omitempty" xmlrpc:"currentBandwidthSummary,omitempty"`

	// Indicates if a server has a Customer Installed OS
	CustomerInstalledOperatingSystemFlag *bool `json:"customerInstalledOperatingSystemFlag,omitempty" xmlrpc:"customerInstalledOperatingSystemFlag,omitempty"`

	// Indicates if a server is a customer owned device.
	CustomerOwnedFlag *bool `json:"customerOwnedFlag,omitempty" xmlrpc:"customerOwnedFlag,omitempty"`

	// The total private inbound bandwidth for this hardware for the current billing cycle.
	InboundPrivateBandwidthUsage *Float64 `json:"inboundPrivateBandwidthUsage,omitempty" xmlrpc:"inboundPrivateBandwidthUsage,omitempty"`

	// The last transaction that a server's operating system was loaded.
	LastOperatingSystemReload *Provisioning_Version1_Transaction `json:"lastOperatingSystemReload,omitempty" xmlrpc:"lastOperatingSystemReload,omitempty"`

	// The metric tracking object id for this server.
	MetricTrackingObjectId *int `json:"metricTrackingObjectId,omitempty" xmlrpc:"metricTrackingObjectId,omitempty"`

	// The monitoring notification objects for this hardware. Each object links this hardware instance to a user account that will be notified if monitoring on this hardware object fails
	MonitoringUserNotification []User_Customer_Notification_Hardware `json:"monitoringUserNotification,omitempty" xmlrpc:"monitoringUserNotification,omitempty"`

	// A count of the monitoring notification objects for this hardware. Each object links this hardware instance to a user account that will be notified if monitoring on this hardware object fails
	MonitoringUserNotificationCount *uint `json:"monitoringUserNotificationCount,omitempty" xmlrpc:"monitoringUserNotificationCount,omitempty"`

	// An open ticket requesting cancellation of this server, if one exists.
	OpenCancellationTicket *Ticket `json:"openCancellationTicket,omitempty" xmlrpc:"openCancellationTicket,omitempty"`

	// The total private outbound bandwidth for this hardware for the current billing cycle.
	OutboundPrivateBandwidthUsage *Float64 `json:"outboundPrivateBandwidthUsage,omitempty" xmlrpc:"outboundPrivateBandwidthUsage,omitempty"`

	// Whether the bandwidth usage for this hardware for the current billing cycle exceeds the allocation.
	OverBandwidthAllocationFlag *int `json:"overBandwidthAllocationFlag,omitempty" xmlrpc:"overBandwidthAllocationFlag,omitempty"`

	// A server's primary private IP address.
	PrivateIpAddress *string `json:"privateIpAddress,omitempty" xmlrpc:"privateIpAddress,omitempty"`

	// Whether the bandwidth usage for this hardware for the current billing cycle is projected to exceed the allocation.
	ProjectedOverBandwidthAllocationFlag *int `json:"projectedOverBandwidthAllocationFlag,omitempty" xmlrpc:"projectedOverBandwidthAllocationFlag,omitempty"`

	// The projected public outbound bandwidth for this hardware for the current billing cycle.
	ProjectedPublicBandwidthUsage *Float64 `json:"projectedPublicBandwidthUsage,omitempty" xmlrpc:"projectedPublicBandwidthUsage,omitempty"`

	// A count of the last five commands issued to the server's remote management card.
	RecentRemoteManagementCommandCount *uint `json:"recentRemoteManagementCommandCount,omitempty" xmlrpc:"recentRemoteManagementCommandCount,omitempty"`

	// The last five commands issued to the server's remote management card.
	RecentRemoteManagementCommands []Hardware_Component_RemoteManagement_Command_Request `json:"recentRemoteManagementCommands,omitempty" xmlrpc:"recentRemoteManagementCommands,omitempty"`

	// no documentation yet
	RegionalInternetRegistry *Network_Regional_Internet_Registry `json:"regionalInternetRegistry,omitempty" xmlrpc:"regionalInternetRegistry,omitempty"`

	// A server's remote management card.
	RemoteManagement *Hardware_Component_RemoteManagement `json:"remoteManagement,omitempty" xmlrpc:"remoteManagement,omitempty"`

	// A count of user(s) who have access to issue commands and/or interact with the server's remote management card.
	RemoteManagementUserCount *uint `json:"remoteManagementUserCount,omitempty" xmlrpc:"remoteManagementUserCount,omitempty"`

	// User(s) who have access to issue commands and/or interact with the server's remote management card.
	RemoteManagementUsers []Hardware_Component_RemoteManagement_User `json:"remoteManagementUsers,omitempty" xmlrpc:"remoteManagementUsers,omitempty"`

	// A server's remote management card used for statistics.
	StatisticsRemoteManagement *Hardware_Component_RemoteManagement `json:"statisticsRemoteManagement,omitempty" xmlrpc:"statisticsRemoteManagement,omitempty"`

	// A count of a list of users that have access to this computing instance.
	UserCount *uint `json:"userCount,omitempty" xmlrpc:"userCount,omitempty"`

	// A list of users that have access to this computing instance.
	Users []User_Customer `json:"users,omitempty" xmlrpc:"users,omitempty"`

	// A count of a hardware server's virtual servers.
	VirtualGuestCount *uint `json:"virtualGuestCount,omitempty" xmlrpc:"virtualGuestCount,omitempty"`

	// A hardware server's virtual servers.
	VirtualGuests []Virtual_Guest `json:"virtualGuests,omitempty" xmlrpc:"virtualGuests,omitempty"`
}

// SoftLayer_Hardware_Status models the inventory state of any piece of hardware in SoftLayer's inventory. Most of these statuses are used by SoftLayer while a server is not provisioned or undergoing provisioning. SoftLayer uses the following status codes:
//
//
// *'''ACTIVE''': This server is active and in use.
// *'''DEPLOY''': Used during server provisioning.
// *'''DEPLOY2''': Used during server provisioning.
// *'''MACWAIT''': Used during server provisioning.
// *'''RECLAIM''': This server has been reclaimed by SoftLayer and is awaiting de-provisioning.
//
//
// Servers in production and in use should stay in the ACTIVE state. If a server's status ever reads anything else then please contact SoftLayer support.
type Hardware_Status struct {
	Entity

	// A hardware status' internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A hardware's status code. See the SoftLayer_Hardware_Status Overview for ''status''' possible values.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// The SoftLayer_Hardware_Switch object extends the base functionality of the SoftLayer_Hardware service.
type Hardware_Switch struct {
	Hardware
}
