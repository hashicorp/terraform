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

// SoftLayer_Software_AccountLicense is a class that represents software licenses that are tied only to a customer's account and not to any particular hardware, IP address, etc.
type Software_AccountLicense struct {
	Entity

	// The customer account this Account License belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The ID of the SoftLayer Account to which this Account License belongs to.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The billing item for a software account license.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// Some Account Licenses have capacity information such as CPU specified in the units key. This provides the numerical representation of the capacity of the units.
	Capacity *string `json:"capacity,omitempty" xmlrpc:"capacity,omitempty"`

	// The License Key for this specific Account License.
	Key *string `json:"key,omitempty" xmlrpc:"key,omitempty"`

	// The SoftLayer_Software_Description that this account license is for.
	SoftwareDescription *Software_Description `json:"softwareDescription,omitempty" xmlrpc:"softwareDescription,omitempty"`

	// The unit of measurement that an account license has the capacity of.
	Units *string `json:"units,omitempty" xmlrpc:"units,omitempty"`
}

// A SoftLayer_Software_Component ties the installation of a specific piece of software onto a specific piece of hardware.
//
// SoftLayer_Software_Component works with SoftLayer_Software_License and SoftLayer_Software_Description to tie this all together.
//
// <ul> <li>SoftLayer_Software_Component is the installation of a specific piece of software onto a specific piece of hardware in accordance to a software license. <ul> <li>SoftLayer_Software_License dictates when and how a specific piece of software may be installed onto a piece of hardware. <ul> <li>SoftLayer_Software_Description describes a specific piece of software which can be installed onto hardware in accordance with it's license agreement. </li></ul></li></ul></li></ul>
type Software_Component struct {
	Entity

	// The average amount of time that a software component takes to install.
	AverageInstallationDuration *uint `json:"averageInstallationDuration,omitempty" xmlrpc:"averageInstallationDuration,omitempty"`

	// The billing item for a software component.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The hardware this Software Component is installed upon.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// Hardware Identification Number for the server this Software Component is installed upon.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// An ID number identifying this Software Component (Software Installation)
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The manufacturer code that is needed to activate a license.
	ManufacturerActivationCode *string `json:"manufacturerActivationCode,omitempty" xmlrpc:"manufacturerActivationCode,omitempty"`

	// A license key for this specific installation of software, if it is needed.
	ManufacturerLicenseInstance *string `json:"manufacturerLicenseInstance,omitempty" xmlrpc:"manufacturerLicenseInstance,omitempty"`

	// A count of username/Password pairs used for access to this Software Installation.
	PasswordCount *uint `json:"passwordCount,omitempty" xmlrpc:"passwordCount,omitempty"`

	// History Records for Software Passwords.
	PasswordHistory []Software_Component_Password_History `json:"passwordHistory,omitempty" xmlrpc:"passwordHistory,omitempty"`

	// A count of history Records for Software Passwords.
	PasswordHistoryCount *uint `json:"passwordHistoryCount,omitempty" xmlrpc:"passwordHistoryCount,omitempty"`

	// Username/Password pairs used for access to this Software Installation.
	Passwords []Software_Component_Password `json:"passwords,omitempty" xmlrpc:"passwords,omitempty"`

	// The Software Description of this Software Component.
	SoftwareDescription *Software_Description `json:"softwareDescription,omitempty" xmlrpc:"softwareDescription,omitempty"`

	// The License this Software Component uses.
	SoftwareLicense *Software_License `json:"softwareLicense,omitempty" xmlrpc:"softwareLicense,omitempty"`

	// The virtual guest this software component is installed upon.
	VirtualGuest *Virtual_Guest `json:"virtualGuest,omitempty" xmlrpc:"virtualGuest,omitempty"`
}

// This object specifies a specific type of Software Component:  An analytics instance. Analytics installations have a specific default ports and patterns for usernames and passwords.  Defaults are initiated by this object.
type Software_Component_Analytics struct {
	Software_Component
}

// This object specifies a specific Software Component:  An Urchin instance. Urchin installations have a specific default port (9999) and a pattern for usernames and passwords.  Defaults are initiated by this object.
type Software_Component_Analytics_Urchin struct {
	Software_Component_Analytics
}

// This object specifies a specific type of Software Component:  An Anti-virus/spyware instance. Anti-virus/spyware installations have specific properties and methods such as SoftLayer_Software_Component_AntivirusSpyware::updateAntivirusSpywarePolicy. Defaults are initiated by this object.
type Software_Component_AntivirusSpyware struct {
	Software_Component
}

// The SoftLayer_Software_Component_AntivirusSpyware_Mcafee represents a single anti-virus/spyware software component.
type Software_Component_AntivirusSpyware_Mcafee struct {
	Software_Component_AntivirusSpyware
}

// The SoftLayer_Software_Component_AntivirusSpyware_Mcafee_Epo_Version36 data type represents a single McAfee Secure anti-virus/spyware software component that uses the ePolicy Orchestrator version 3.6 backend.
type Software_Component_AntivirusSpyware_Mcafee_Epo_Version36 struct {
	Software_Component_AntivirusSpyware_Mcafee

	// The virus scan agent details.
	AgentDetails *McAfee_Epolicy_Orchestrator_Version36_Agent_Details `json:"agentDetails,omitempty" xmlrpc:"agentDetails,omitempty"`

	// The current anti-virus policy.
	CurrentAntivirusPolicy *int `json:"currentAntivirusPolicy,omitempty" xmlrpc:"currentAntivirusPolicy,omitempty"`

	// The virus definition file version.
	DataFileVersion *McAfee_Epolicy_Orchestrator_Version36_Product_Properties `json:"dataFileVersion,omitempty" xmlrpc:"dataFileVersion,omitempty"`

	// The version of ePolicy Orchestrator that the anti-virus/spyware client communicates with.
	EpoVersion *string `json:"epoVersion,omitempty" xmlrpc:"epoVersion,omitempty"`

	// A count of the latest access protection events.
	LatestAccessProtectionEventCount *uint `json:"latestAccessProtectionEventCount,omitempty" xmlrpc:"latestAccessProtectionEventCount,omitempty"`

	// The latest access protection events.
	LatestAccessProtectionEvents []McAfee_Epolicy_Orchestrator_Version36_Antivirus_Event_AccessProtection `json:"latestAccessProtectionEvents,omitempty" xmlrpc:"latestAccessProtectionEvents,omitempty"`

	// A count of the latest anti-virus events.
	LatestAntivirusEventCount *uint `json:"latestAntivirusEventCount,omitempty" xmlrpc:"latestAntivirusEventCount,omitempty"`

	// The latest anti-virus events.
	LatestAntivirusEvents []McAfee_Epolicy_Orchestrator_Version36_Antivirus_Event `json:"latestAntivirusEvents,omitempty" xmlrpc:"latestAntivirusEvents,omitempty"`

	// A count of the latest spyware events.
	LatestSpywareEventCount *uint `json:"latestSpywareEventCount,omitempty" xmlrpc:"latestSpywareEventCount,omitempty"`

	// The latest spyware events.
	LatestSpywareEvents []McAfee_Epolicy_Orchestrator_Version36_Antivirus_Event `json:"latestSpywareEvents,omitempty" xmlrpc:"latestSpywareEvents,omitempty"`

	// The current transaction status of a server.
	TransactionStatus *string `json:"transactionStatus,omitempty" xmlrpc:"transactionStatus,omitempty"`
}

// The SoftLayer_Software_Component_AntivirusSpyware_Mcafee_Epo_Version45 data type represents a single McAfee Secure anti-virus/spyware software component that uses the ePolicy Orchestrator version 4.5 backend.
type Software_Component_AntivirusSpyware_Mcafee_Epo_Version45 struct {
	Software_Component_AntivirusSpyware_Mcafee

	// The virus scan agent details.
	AgentDetails *McAfee_Epolicy_Orchestrator_Version45_Agent_Details `json:"agentDetails,omitempty" xmlrpc:"agentDetails,omitempty"`

	// The current anti-virus policy.
	CurrentAntivirusPolicy *int `json:"currentAntivirusPolicy,omitempty" xmlrpc:"currentAntivirusPolicy,omitempty"`

	// The virus definition file version.
	DataFileVersion *McAfee_Epolicy_Orchestrator_Version45_Product_Properties `json:"dataFileVersion,omitempty" xmlrpc:"dataFileVersion,omitempty"`

	// The version of ePolicy Orchestrator that the anti-virus/spyware client communicates with.
	EpoVersion *string `json:"epoVersion,omitempty" xmlrpc:"epoVersion,omitempty"`

	// A count of the latest access protection events.
	LatestAccessProtectionEventCount *uint `json:"latestAccessProtectionEventCount,omitempty" xmlrpc:"latestAccessProtectionEventCount,omitempty"`

	// The latest access protection events.
	LatestAccessProtectionEvents []McAfee_Epolicy_Orchestrator_Version45_Event `json:"latestAccessProtectionEvents,omitempty" xmlrpc:"latestAccessProtectionEvents,omitempty"`

	// A count of the latest anti-virus events.
	LatestAntivirusEventCount *uint `json:"latestAntivirusEventCount,omitempty" xmlrpc:"latestAntivirusEventCount,omitempty"`

	// The latest anti-virus events.
	LatestAntivirusEvents []McAfee_Epolicy_Orchestrator_Version45_Event `json:"latestAntivirusEvents,omitempty" xmlrpc:"latestAntivirusEvents,omitempty"`

	// A count of the latest spyware events
	LatestSpywareEventCount *uint `json:"latestSpywareEventCount,omitempty" xmlrpc:"latestSpywareEventCount,omitempty"`

	// The latest spyware events
	LatestSpywareEvents []McAfee_Epolicy_Orchestrator_Version45_Event `json:"latestSpywareEvents,omitempty" xmlrpc:"latestSpywareEvents,omitempty"`

	// The current transaction status of a server.
	TransactionStatus *string `json:"transactionStatus,omitempty" xmlrpc:"transactionStatus,omitempty"`
}

// This object specifies a specific type of Software Component:  A control panel instance. Control panel installations have a specific default ports and patterns for usernames and passwords.  Defaults are initiated by this object.
type Software_Component_ControlPanel struct {
	Software_Component
}

// This object specifies a specific Software Component:  A cPanel instance. cPanel installations have a specific default port (2086) and a pattern for usernames and passwords.  Defaults are initiated by this object.
type Software_Component_ControlPanel_Cpanel struct {
	Software_Component
}

// This object specifies a specific type of control panel Software Component:  An Idera instance.
type Software_Component_ControlPanel_Idera struct {
	Software_Component
}

// This object specifies a specific type of Software Component:  A Idera Server Backup instance.
type Software_Component_ControlPanel_Idera_ServerBackup struct {
	Software_Component_ControlPanel_Idera
}

// This object is a parent class for Microsoft Products, like Web Matrix
type Software_Component_ControlPanel_Microsoft struct {
	Software_Component
}

// This object specifies a specific Software Component:  A WebPlatform instance. WebPlatform installations have a specific xml config with usernames and passwords.  Defaults are initiated by this object.
type Software_Component_ControlPanel_Microsoft_WebPlatform struct {
	Software_Component_ControlPanel_Microsoft
}

// This object is a parent class for SWSoft Products, like Plesk
type Software_Component_ControlPanel_Parallels struct {
	Software_Component
}

// This object specifies a specific Software Component:  A Plesk instance produced by SWSoft. SWSoft Plesk installations have a specific default port (8443) and a pattern for usernames and passwords.  Defaults are initiated by this object.
type Software_Component_ControlPanel_Parallels_Plesk struct {
	Software_Component_ControlPanel_Parallels
}

// This object specifies a specific type of control panel Software Component:  A R1soft instance.
type Software_Component_ControlPanel_R1soft struct {
	Software_Component
}

// This object specifies a specific type of Software Component:  A R1soft continuous data protection instance.
type Software_Component_ControlPanel_R1soft_Cdp struct {
	Software_Component_ControlPanel_R1soft
}

// This object specifies a specific type of Software Component:  A R1Soft Server Backup instance.
type Software_Component_ControlPanel_R1soft_ServerBackup struct {
	Software_Component_ControlPanel_R1soft
}

// This object is a parent class for SWSoft Products, like Plesk
type Software_Component_ControlPanel_Swsoft struct {
	Software_Component
}

// This object specifies a specific Software Component:  A Helm instance produced by Webhost Automation. WEbhost Automation's Helm installations have a specific default port (8086) and a pattern for usernames and passwords.  Defaults are initiated by this object.
type Software_Component_ControlPanel_WebhostAutomation struct {
	Software_Component
}

// This object specifies a specific type of Software Component:  A Host Intrusion Protection System instance.
type Software_Component_HostIps struct {
	Software_Component
}

// The SoftLayer_Software_Component_HostIps_Mcafee represents a single host IPS software component.
type Software_Component_HostIps_Mcafee struct {
	Software_Component_HostIps
}

// The SoftLayer_Software_Component_HostIps_Mcafee_Epo_Version36_Hips data type represents a single McAfee Secure Host IPS software component that uses the ePolicy Orchestrator version 3.6 backend.
type Software_Component_HostIps_Mcafee_Epo_Version36_Hips struct {
	Software_Component_HostIps_Mcafee

	// The host IPS agent details.
	AgentDetails *McAfee_Epolicy_Orchestrator_Version36_Agent_Details `json:"agentDetails,omitempty" xmlrpc:"agentDetails,omitempty"`

	// A count of the names of the possible policy options for the application mode setting.
	ApplicationModePolicyNameCount *uint `json:"applicationModePolicyNameCount,omitempty" xmlrpc:"applicationModePolicyNameCount,omitempty"`

	// The names of the possible policy options for the application mode setting.
	ApplicationModePolicyNames []McAfee_Epolicy_Orchestrator_Version36_Policy_Object `json:"applicationModePolicyNames,omitempty" xmlrpc:"applicationModePolicyNames,omitempty"`

	// A count of the names of the possible policy options for the application rule set setting.
	ApplicationRuleSetPolicyNameCount *uint `json:"applicationRuleSetPolicyNameCount,omitempty" xmlrpc:"applicationRuleSetPolicyNameCount,omitempty"`

	// The names of the possible policy options for the application rule set setting.
	ApplicationRuleSetPolicyNames []McAfee_Epolicy_Orchestrator_Version36_Policy_Object `json:"applicationRuleSetPolicyNames,omitempty" xmlrpc:"applicationRuleSetPolicyNames,omitempty"`

	// A count of the names of the possible options for the enforcement policy setting.
	EnforcementPolicyNameCount *uint `json:"enforcementPolicyNameCount,omitempty" xmlrpc:"enforcementPolicyNameCount,omitempty"`

	// The names of the possible options for the enforcement policy setting.
	EnforcementPolicyNames []McAfee_Epolicy_Orchestrator_Version36_Policy_Object `json:"enforcementPolicyNames,omitempty" xmlrpc:"enforcementPolicyNames,omitempty"`

	// The version of ePolicy Orchestrator that the host IPS client communicates with.
	EpoVersion *string `json:"epoVersion,omitempty" xmlrpc:"epoVersion,omitempty"`

	// A count of the names of the possible policy options for the firewall mode setting.
	FirewallModePolicyNameCount *uint `json:"firewallModePolicyNameCount,omitempty" xmlrpc:"firewallModePolicyNameCount,omitempty"`

	// The names of the possible policy options for the firewall mode setting.
	FirewallModePolicyNames []McAfee_Epolicy_Orchestrator_Version36_Policy_Object `json:"firewallModePolicyNames,omitempty" xmlrpc:"firewallModePolicyNames,omitempty"`

	// A count of the names of the possible policy options for the firewall rule set setting.
	FirewallRuleSetPolicyNameCount *uint `json:"firewallRuleSetPolicyNameCount,omitempty" xmlrpc:"firewallRuleSetPolicyNameCount,omitempty"`

	// The names of the possible policy options for the firewall rule set setting.
	FirewallRuleSetPolicyNames []McAfee_Epolicy_Orchestrator_Version36_Policy_Object `json:"firewallRuleSetPolicyNames,omitempty" xmlrpc:"firewallRuleSetPolicyNames,omitempty"`

	// A count of the names of the possible policy options for the host IPS mode setting.
	IpsModePolicyNameCount *uint `json:"ipsModePolicyNameCount,omitempty" xmlrpc:"ipsModePolicyNameCount,omitempty"`

	// The names of the possible policy options for the host IPS mode setting.
	IpsModePolicyNames []McAfee_Epolicy_Orchestrator_Version36_Policy_Object `json:"ipsModePolicyNames,omitempty" xmlrpc:"ipsModePolicyNames,omitempty"`

	// A count of the names of the possible policy options for the host IPS protection setting.
	IpsProtectionPolicyNameCount *uint `json:"ipsProtectionPolicyNameCount,omitempty" xmlrpc:"ipsProtectionPolicyNameCount,omitempty"`

	// The names of the possible policy options for the host IPS protection setting.
	IpsProtectionPolicyNames []McAfee_Epolicy_Orchestrator_Version36_Policy_Object `json:"ipsProtectionPolicyNames,omitempty" xmlrpc:"ipsProtectionPolicyNames,omitempty"`

	// The current transaction status of a server.
	TransactionStatus *string `json:"transactionStatus,omitempty" xmlrpc:"transactionStatus,omitempty"`
}

// The SoftLayer_Software_Component_HostIps_Mcafee_Epo_Version36_Hips_Version6 data type represents a single McAfee Secure Host IPS software component for version 6 of the Host IPS client and uses the ePolicy Orchestrator version 3.6 backend.
type Software_Component_HostIps_Mcafee_Epo_Version36_Hips_Version6 struct {
	Software_Component_HostIps_Mcafee_Epo_Version36_Hips

	// A count of the blocked application events for this software component.
	BlockedApplicationEventCount *uint `json:"blockedApplicationEventCount,omitempty" xmlrpc:"blockedApplicationEventCount,omitempty"`

	// The blocked application events for this software component.
	BlockedApplicationEvents []McAfee_Epolicy_Orchestrator_Version36_Hips_Version6_BlockedApplicationEvent `json:"blockedApplicationEvents,omitempty" xmlrpc:"blockedApplicationEvents,omitempty"`

	// A count of the host IPS events for this software component.
	IpsEventCount *uint `json:"ipsEventCount,omitempty" xmlrpc:"ipsEventCount,omitempty"`

	// The host IPS events for this software component.
	IpsEvents []McAfee_Epolicy_Orchestrator_Version36_Hips_Version6_IPSEvent `json:"ipsEvents,omitempty" xmlrpc:"ipsEvents,omitempty"`
}

// The SoftLayer_Software_Component_HostIps_Mcafee_Epo_Version36_Hips_Version7 data type represents a single McAfee Secure Host IPS software component for version 7 of the Host IPS client and uses the ePolicy Orchestrator version 3.6 backend.
type Software_Component_HostIps_Mcafee_Epo_Version36_Hips_Version7 struct {
	Software_Component_HostIps_Mcafee_Epo_Version36_Hips

	// A count of the blocked application events for this software component.
	BlockedApplicationEventCount *uint `json:"blockedApplicationEventCount,omitempty" xmlrpc:"blockedApplicationEventCount,omitempty"`

	// The blocked application events for this software component.
	BlockedApplicationEvents []McAfee_Epolicy_Orchestrator_Version36_Hips_Version7_BlockedApplicationEvent `json:"blockedApplicationEvents,omitempty" xmlrpc:"blockedApplicationEvents,omitempty"`

	// A count of the host IPS events for this software component.
	IpsEventCount *uint `json:"ipsEventCount,omitempty" xmlrpc:"ipsEventCount,omitempty"`

	// The host IPS events for this software component.
	IpsEvents []McAfee_Epolicy_Orchestrator_Version36_Hips_Version7_IPSEvent `json:"ipsEvents,omitempty" xmlrpc:"ipsEvents,omitempty"`
}

// The SoftLayer_Software_Component_HostIps_Mcafee_Epo_Version45_Hips data type represents a single McAfee Secure Host IPS software component that uses the ePolicy Orchestrator version 4.5 backend.
type Software_Component_HostIps_Mcafee_Epo_Version45_Hips struct {
	Software_Component_HostIps_Mcafee

	// The host IPS agent details.
	AgentDetails *McAfee_Epolicy_Orchestrator_Version45_Agent_Details `json:"agentDetails,omitempty" xmlrpc:"agentDetails,omitempty"`

	// A count of the names of the possible policy options for the application mode setting.
	ApplicationModePolicyNameCount *uint `json:"applicationModePolicyNameCount,omitempty" xmlrpc:"applicationModePolicyNameCount,omitempty"`

	// The names of the possible policy options for the application mode setting.
	ApplicationModePolicyNames []McAfee_Epolicy_Orchestrator_Version45_Policy_Object `json:"applicationModePolicyNames,omitempty" xmlrpc:"applicationModePolicyNames,omitempty"`

	// A count of the names of the possible policy options for the application rule set setting.
	ApplicationRuleSetPolicyNameCount *uint `json:"applicationRuleSetPolicyNameCount,omitempty" xmlrpc:"applicationRuleSetPolicyNameCount,omitempty"`

	// The names of the possible policy options for the application rule set setting.
	ApplicationRuleSetPolicyNames []McAfee_Epolicy_Orchestrator_Version45_Policy_Object `json:"applicationRuleSetPolicyNames,omitempty" xmlrpc:"applicationRuleSetPolicyNames,omitempty"`

	// A count of the blocked application events for this software component.
	BlockedApplicationEventCount *uint `json:"blockedApplicationEventCount,omitempty" xmlrpc:"blockedApplicationEventCount,omitempty"`

	// The blocked application events for this software component.
	BlockedApplicationEvents []McAfee_Epolicy_Orchestrator_Version45_Event `json:"blockedApplicationEvents,omitempty" xmlrpc:"blockedApplicationEvents,omitempty"`

	// A count of the names of the possible options for the enforcement policy setting.
	EnforcementPolicyNameCount *uint `json:"enforcementPolicyNameCount,omitempty" xmlrpc:"enforcementPolicyNameCount,omitempty"`

	// The names of the possible options for the enforcement policy setting.
	EnforcementPolicyNames []McAfee_Epolicy_Orchestrator_Version45_Policy_Object `json:"enforcementPolicyNames,omitempty" xmlrpc:"enforcementPolicyNames,omitempty"`

	// The version of ePolicy Orchestrator that the host IPS client communicates with.
	EpoVersion *string `json:"epoVersion,omitempty" xmlrpc:"epoVersion,omitempty"`

	// A count of the names of the possible policy options for the firewall mode setting.
	FirewallModePolicyNameCount *uint `json:"firewallModePolicyNameCount,omitempty" xmlrpc:"firewallModePolicyNameCount,omitempty"`

	// The names of the possible policy options for the firewall mode setting.
	FirewallModePolicyNames []McAfee_Epolicy_Orchestrator_Version45_Policy_Object `json:"firewallModePolicyNames,omitempty" xmlrpc:"firewallModePolicyNames,omitempty"`

	// A count of the names of the possible policy options for the firewall rule set setting.
	FirewallRuleSetPolicyNameCount *uint `json:"firewallRuleSetPolicyNameCount,omitempty" xmlrpc:"firewallRuleSetPolicyNameCount,omitempty"`

	// The names of the possible policy options for the firewall rule set setting.
	FirewallRuleSetPolicyNames []McAfee_Epolicy_Orchestrator_Version45_Policy_Object `json:"firewallRuleSetPolicyNames,omitempty" xmlrpc:"firewallRuleSetPolicyNames,omitempty"`

	// A count of the host IPS events for this software component.
	IpsEventCount *uint `json:"ipsEventCount,omitempty" xmlrpc:"ipsEventCount,omitempty"`

	// The host IPS events for this software component.
	IpsEvents []McAfee_Epolicy_Orchestrator_Version45_Event `json:"ipsEvents,omitempty" xmlrpc:"ipsEvents,omitempty"`

	// A count of the names of the possible policy options for the host IPS mode setting.
	IpsModePolicyNameCount *uint `json:"ipsModePolicyNameCount,omitempty" xmlrpc:"ipsModePolicyNameCount,omitempty"`

	// The names of the possible policy options for the host IPS mode setting.
	IpsModePolicyNames []McAfee_Epolicy_Orchestrator_Version45_Policy_Object `json:"ipsModePolicyNames,omitempty" xmlrpc:"ipsModePolicyNames,omitempty"`

	// A count of the names of the possible policy options for the host IPS protection setting.
	IpsProtectionPolicyNameCount *uint `json:"ipsProtectionPolicyNameCount,omitempty" xmlrpc:"ipsProtectionPolicyNameCount,omitempty"`

	// The names of the possible policy options for the host IPS protection setting.
	IpsProtectionPolicyNames []McAfee_Epolicy_Orchestrator_Version45_Policy_Object `json:"ipsProtectionPolicyNames,omitempty" xmlrpc:"ipsProtectionPolicyNames,omitempty"`

	// The current transaction status of a server.
	TransactionStatus *string `json:"transactionStatus,omitempty" xmlrpc:"transactionStatus,omitempty"`
}

// The SoftLayer_Software_Component_HostIps_Mcafee_Epo_Version45_Hips_Version7 data type represents a single McAfee Secure Host IPS software component for version 7 of the Host IPS client and uses the ePolicy Orchestrator version 4.5 backend.
type Software_Component_HostIps_Mcafee_Epo_Version45_Hips_Version7 struct {
	Software_Component_HostIps_Mcafee_Epo_Version45_Hips
}

// The SoftLayer_Software_Component_HostIps_Mcafee_Epo_Version45_Hips_Version8 data type represents a single McAfee Secure Host IPS software component for version 8 of the Host IPS client and uses the ePolicy Orchestrator version 4.5 backend.
type Software_Component_HostIps_Mcafee_Epo_Version45_Hips_Version8 struct {
	Software_Component_HostIps_Mcafee_Epo_Version45_Hips
}

// SoftLayer_Software_Component_OperatingSystem extends the [[SoftLayer_Software_Component]] data type to include operating system specific properties.
type Software_Component_OperatingSystem struct {
	Software_Component

	// The date in which the license for this software expires.
	LicenseExpirationDate *Time `json:"licenseExpirationDate,omitempty" xmlrpc:"licenseExpirationDate,omitempty"`

	// A count of an operating system's associated [[SoftLayer_Hardware_Component_Partition_Template|Partition Templates]] that can be used to configure a hardware drive.
	PartitionTemplateCount *uint `json:"partitionTemplateCount,omitempty" xmlrpc:"partitionTemplateCount,omitempty"`

	// An operating system's associated [[SoftLayer_Hardware_Component_Partition_Template|Partition Templates]] that can be used to configure a hardware drive.
	PartitionTemplates []Hardware_Component_Partition_Template `json:"partitionTemplates,omitempty" xmlrpc:"partitionTemplates,omitempty"`

	// An operating systems associated [[SoftLayer_Provisioning_Version1_Transaction_Group|Transaction Group]]. A transaction group is a list of operations that will occur during the installment of an operating system.
	ReloadTransactionGroup *Provisioning_Version1_Transaction_Group `json:"reloadTransactionGroup,omitempty" xmlrpc:"reloadTransactionGroup,omitempty"`
}

// This object specifies a specific type of Software Component:  A package instance.
type Software_Component_Package struct {
	Software_Component
}

// This object specifies a specific type of Software Component:  A package management instance.
type Software_Component_Package_Management struct {
	Software_Component_Package
}

// This object specifies a specific type of Software Component:  A Ksplice instance.
type Software_Component_Package_Management_Ksplice struct {
	Software_Component_Package_Management
}

// This SoftLayer_Software_Component_Password data type contains a password for a specific software component instance.
type Software_Component_Password struct {
	Entity

	// The date this username/password pair was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// An id number for this specific username/password pair.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date of the last modification to this username/password pair.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A note string stored for this username/password pair.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The password part of the username/password pair.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// The application access port for the Software Component.
	Port *int `json:"port,omitempty" xmlrpc:"port,omitempty"`

	// The SoftLayer_Software_Component instance that this username/password pair is valid for.
	Software *Software_Component `json:"software,omitempty" xmlrpc:"software,omitempty"`

	// An id number for the software component this username/password pair is valid for.
	SoftwareId *int `json:"softwareId,omitempty" xmlrpc:"softwareId,omitempty"`

	// A count of sSH keys to be installed on the server during provisioning or an OS reload.
	SshKeyCount *uint `json:"sshKeyCount,omitempty" xmlrpc:"sshKeyCount,omitempty"`

	// SSH keys to be installed on the server during provisioning or an OS reload.
	SshKeys []Security_Ssh_Key `json:"sshKeys,omitempty" xmlrpc:"sshKeys,omitempty"`

	// The username part of the username/password pair.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// This object allows you to find the history of password changes for a specific SoftLayer_Software Component
type Software_Component_Password_History struct {
	Entity

	// The date this username/password pair was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A note string stored for this username/password pair.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The password part of this specific password history instance.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// An installed and licensed instance of a piece of software
	SoftwareComponent *Software_Component `json:"softwareComponent,omitempty" xmlrpc:"softwareComponent,omitempty"`

	// The id number for the Software Component this username/password pair is for.
	SoftwareComponentId *int `json:"softwareComponentId,omitempty" xmlrpc:"softwareComponentId,omitempty"`

	// The username part of this specific password history instance.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// This object specifies a specific type of Software Component:  A security instance. Security installations have custom configurations for password requirements.
type Software_Component_Security struct {
	Software_Component
}

// This object specifies a specific Software Component:  A SafeNet instance. SafeNet installations have custom configurations for password requirements.
type Software_Component_Security_SafeNet struct {
	Software_Component_Security
}

// This class holds a description for a specific installation of a Software Component.
//
// SoftLayer_Software_Licenses tie a Software Component (A specific installation on a piece of hardware) to it's description.
//
// The "Manufacturer" and "Name" properties of a SoftLayer_Software_Description are used by the framework to factory specific objects, objects that may have special methods for that specific piece of software, or objects that contain application specific data, such as default ports.  For example, if you create a SoftLayer_Software_Component who's SoftLayer_Software_License points to the SoftLayer_Software_Description for "Swsoft" "Plesk", you'll actually get a SoftLayer_Software_Component_Swsoft_Plesk object.
type Software_Description struct {
	Entity

	// A count of
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// no documentation yet
	Attributes []Software_Description_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// The average amount of time that a software description takes to install.
	AverageInstallationDuration *int `json:"averageInstallationDuration,omitempty" xmlrpc:"averageInstallationDuration,omitempty"`

	// A count of a list of the software descriptions that are compatible with this software description.
	CompatibleSoftwareDescriptionCount *uint `json:"compatibleSoftwareDescriptionCount,omitempty" xmlrpc:"compatibleSoftwareDescriptionCount,omitempty"`

	// A list of the software descriptions that are compatible with this software description.
	CompatibleSoftwareDescriptions []Software_Description `json:"compatibleSoftwareDescriptions,omitempty" xmlrpc:"compatibleSoftwareDescriptions,omitempty"`

	// This is set to '1' if this Software Description describes a Control Panel.
	ControlPanel *int `json:"controlPanel,omitempty" xmlrpc:"controlPanel,omitempty"`

	// A count of the feature attributes of a software description.
	FeatureCount *uint `json:"featureCount,omitempty" xmlrpc:"featureCount,omitempty"`

	// The feature attributes of a software description.
	Features []Software_Description_Feature `json:"features,omitempty" xmlrpc:"features,omitempty"`

	// An ID number to identify this Software Description.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The latest version of a software description.
	LatestVersion []Software_Description `json:"latestVersion,omitempty" xmlrpc:"latestVersion,omitempty"`

	// A count of the latest version of a software description.
	LatestVersionCount *uint `json:"latestVersionCount,omitempty" xmlrpc:"latestVersionCount,omitempty"`

	// The unit of measurement (day, month, or year) for license registration. Used in conjunction with licenseTermValue to determine overall license registration length of a new license.
	LicenseTermUnit *string `json:"licenseTermUnit,omitempty" xmlrpc:"licenseTermUnit,omitempty"`

	// The number of units (licenseTermUnit) a new license is valid for at the time of registration.
	LicenseTermValue *int `json:"licenseTermValue,omitempty" xmlrpc:"licenseTermValue,omitempty"`

	// The manufacturer, name and version of a piece of software.
	LongDescription *string `json:"longDescription,omitempty" xmlrpc:"longDescription,omitempty"`

	// The name of the manufacturer for this specific piece of software.  This name is used by SoftLayer_Software_Component to tailor make (factory) specific types of Software Components that know details like default ports.
	Manufacturer *string `json:"manufacturer,omitempty" xmlrpc:"manufacturer,omitempty"`

	// The name of this specific piece of software.  This name is used by SoftLayer_Software_Component to tailor make (factory) specific types of Software Components that know details like default ports.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// This is set to '1' if this Software Description describes an Operating System.
	OperatingSystem *int `json:"operatingSystem,omitempty" xmlrpc:"operatingSystem,omitempty"`

	// A count of the various product items to which this software description is linked.
	ProductItemCount *uint `json:"productItemCount,omitempty" xmlrpc:"productItemCount,omitempty"`

	// The various product items to which this software description is linked.
	ProductItems []Product_Item `json:"productItems,omitempty" xmlrpc:"productItems,omitempty"`

	// This details the provisioning transaction group for this software. This is only valid for Operating System software.
	ProvisionTransactionGroup *Provisioning_Version1_Transaction_Group `json:"provisionTransactionGroup,omitempty" xmlrpc:"provisionTransactionGroup,omitempty"`

	// A reference code is structured as three tokens separated by underscores. The first token represents the product, the second is the version of the product, and the third is whether the software is 32 or 64bit.
	ReferenceCode *string `json:"referenceCode,omitempty" xmlrpc:"referenceCode,omitempty"`

	// The transaction group that a software description belongs to. A transaction group is a sequence of transactions that must be performed in a specific order for the installation of software.
	ReloadTransactionGroup *Provisioning_Version1_Transaction_Group `json:"reloadTransactionGroup,omitempty" xmlrpc:"reloadTransactionGroup,omitempty"`

	// The default user created for a given a software description.
	RequiredUser *string `json:"requiredUser,omitempty" xmlrpc:"requiredUser,omitempty"`

	// A count of software Licenses that govern this Software Description.
	SoftwareLicenseCount *uint `json:"softwareLicenseCount,omitempty" xmlrpc:"softwareLicenseCount,omitempty"`

	// Software Licenses that govern this Software Description.
	SoftwareLicenses []Software_License `json:"softwareLicenses,omitempty" xmlrpc:"softwareLicenses,omitempty"`

	// A suggestion for an upgrade path from this Software Description
	UpgradeSoftwareDescription *Software_Description `json:"upgradeSoftwareDescription,omitempty" xmlrpc:"upgradeSoftwareDescription,omitempty"`

	// Contains the ID of the suggested upgrade from this Software_Description to a more powerful software installation.
	UpgradeSoftwareDescriptionId *int `json:"upgradeSoftwareDescriptionId,omitempty" xmlrpc:"upgradeSoftwareDescriptionId,omitempty"`

	// A suggestion for an upgrade path from this Software Description (Deprecated - Use upgradeSoftwareDescription)
	UpgradeSwDesc *Software_Description `json:"upgradeSwDesc,omitempty" xmlrpc:"upgradeSwDesc,omitempty"`

	// Contains the ID of the suggested upgrade from this Software_Description to a more powerful software installation. (Deprecated - Use upgradeSoftwareDescriptionId)
	UpgradeSwDescId *int `json:"upgradeSwDescId,omitempty" xmlrpc:"upgradeSwDescId,omitempty"`

	// A count of
	ValidFilesystemTypeCount *uint `json:"validFilesystemTypeCount,omitempty" xmlrpc:"validFilesystemTypeCount,omitempty"`

	// no documentation yet
	ValidFilesystemTypes []Configuration_Storage_Filesystem_Type `json:"validFilesystemTypes,omitempty" xmlrpc:"validFilesystemTypes,omitempty"`

	// The version of this specific piece of software.
	Version *string `json:"version,omitempty" xmlrpc:"version,omitempty"`

	// This is set to '1' if this Software Description can be licensed to a Virtual Machine (an IP address).
	VirtualLicense *int `json:"virtualLicense,omitempty" xmlrpc:"virtualLicense,omitempty"`

	// This is set to '1' if this Software Description a platform for hosting virtual servers.
	VirtualizationPlatform *int `json:"virtualizationPlatform,omitempty" xmlrpc:"virtualizationPlatform,omitempty"`
}

// The SoftLayer_Software_Description_Attribute data type represents an attributes associated with this software description.
type Software_Description_Attribute struct {
	Entity

	// no documentation yet
	SoftwareDescription *Software_Description `json:"softwareDescription,omitempty" xmlrpc:"softwareDescription,omitempty"`

	// no documentation yet
	Type *Software_Description_Attribute_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The value that was assigned to this attribute.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The SoftLayer_Software_Description_Attribute_Type data type represents the type of an attribute.
type Software_Description_Attribute_Type struct {
	Entity

	// The keyname for this attribute type.
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`
}

// The SoftLayer_Software_Description_Feature data type represents a single software description feature. A feature may show up on more than one software description and can not be created, modified, or removed.
type Software_Description_Feature struct {
	Entity

	// The unique identifier for a software description feature.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A unique name used to reference this software description feature.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of a software description feature.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The vendor that a software description feature belongs to.
	Vendor *string `json:"vendor,omitempty" xmlrpc:"vendor,omitempty"`
}

// This class represents a software description's required user
type Software_Description_RequiredUser struct {
	Entity

	// If the default password is set the user will be created with that password, otherwise a random password is generated.
	DefaultPassword *string `json:"defaultPassword,omitempty" xmlrpc:"defaultPassword,omitempty"`

	// If this software has a required user (such as "root") this string contains it's name.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// This class describes a specific type of license, like a Microsoft Windows Site License, a GPL license, or a license of another type.
type Software_License struct {
	Entity

	// The account that owns this specific License instance.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// An ID number for this specific License type.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The account that owns this specific License instance.
	Owner *Account `json:"owner,omitempty" xmlrpc:"owner,omitempty"`

	// A Description of the software that this license instance is valid for.
	SoftwareDescription *Software_Description `json:"softwareDescription,omitempty" xmlrpc:"softwareDescription,omitempty"`

	// The ID number of a Software Description that this specific license is valid for.
	SoftwareDescriptionId *int `json:"softwareDescriptionId,omitempty" xmlrpc:"softwareDescriptionId,omitempty"`
}

// SoftLayer_Software_VirtualLicense is the application class that handles a special type of Software License.  Most software licenses are licensed to a specific hardware ID;  virtual licenses are designed for virtual machines and therefore are assigned to an IP Address.  Not all software packages can be "virtual licensed".
type Software_VirtualLicense struct {
	Entity

	// The customer account this Virtual License belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The ID of the SoftLayer Account to which this Virtual License belongs to.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The billing item for a software virtual license.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The hardware record to which the software virtual license is assigned.
	HostHardware *Hardware_Server `json:"hostHardware,omitempty" xmlrpc:"hostHardware,omitempty"`

	// The ID of the SoftLayer Hardware Server record to which this Virtual License belongs.
	HostHardwareId *int `json:"hostHardwareId,omitempty" xmlrpc:"hostHardwareId,omitempty"`

	// An ID number for this Virtual License instance.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The specific IP address this Virtual License belongs to.
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// The IP Address record associated with a virtual license.
	IpAddressRecord *Network_Subnet_IpAddress `json:"ipAddressRecord,omitempty" xmlrpc:"ipAddressRecord,omitempty"`

	// The License Key for this specific Virtual License.
	Key *string `json:"key,omitempty" xmlrpc:"key,omitempty"`

	// A "notes" string attached to this specific Virtual License.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The SoftLayer_Software_Description that this virtual license is for.
	SoftwareDescription *Software_Description `json:"softwareDescription,omitempty" xmlrpc:"softwareDescription,omitempty"`

	// The Software Description ID this Virtual License is for.
	SoftwareDescriptionId *int `json:"softwareDescriptionId,omitempty" xmlrpc:"softwareDescriptionId,omitempty"`

	// The subnet this Virtual License's IP address belongs to.
	Subnet *Network_Subnet `json:"subnet,omitempty" xmlrpc:"subnet,omitempty"`

	// The ID of the SoftLayer Network Subnet this Virtual License belongs to.
	SubnetId *int `json:"subnetId,omitempty" xmlrpc:"subnetId,omitempty"`
}
