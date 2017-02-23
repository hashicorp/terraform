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

// The SoftLayer_Account data type contains general information relating to a single SoftLayer customer account. Personal information in this type such as names, addresses, and phone numbers are assigned to the account only and not to users belonging to the account. The SoftLayer_Account data type contains a number of relational properties that are used by the SoftLayer customer portal to quickly present a variety of account related services to it's users.
//
// SoftLayer customers are unable to change their company account information in the portal or the API. If you need to change this information please open a sales ticket in our customer portal and our account management staff will assist you.
type Account struct {
	Entity

	// An email address that is responsible for abuse and legal inquiries on behalf of an account. For instance, new legal and abuse tickets are sent to this address.
	AbuseEmail *string `json:"abuseEmail,omitempty" xmlrpc:"abuseEmail,omitempty"`

	// A count of email addresses that are responsible for abuse and legal inquiries on behalf of an account. For instance, new legal and abuse tickets are sent to these addresses.
	AbuseEmailCount *uint `json:"abuseEmailCount,omitempty" xmlrpc:"abuseEmailCount,omitempty"`

	// Email addresses that are responsible for abuse and legal inquiries on behalf of an account. For instance, new legal and abuse tickets are sent to these addresses.
	AbuseEmails []Account_AbuseEmail `json:"abuseEmails,omitempty" xmlrpc:"abuseEmails,omitempty"`

	// A count of the account contacts on an account.
	AccountContactCount *uint `json:"accountContactCount,omitempty" xmlrpc:"accountContactCount,omitempty"`

	// The account contacts on an account.
	AccountContacts []Account_Contact `json:"accountContacts,omitempty" xmlrpc:"accountContacts,omitempty"`

	// A count of the account software licenses owned by an account
	AccountLicenseCount *uint `json:"accountLicenseCount,omitempty" xmlrpc:"accountLicenseCount,omitempty"`

	// The account software licenses owned by an account
	AccountLicenses []Software_AccountLicense `json:"accountLicenses,omitempty" xmlrpc:"accountLicenses,omitempty"`

	// A count of
	AccountLinkCount *uint `json:"accountLinkCount,omitempty" xmlrpc:"accountLinkCount,omitempty"`

	// no documentation yet
	AccountLinks []Account_Link `json:"accountLinks,omitempty" xmlrpc:"accountLinks,omitempty"`

	// A flag indicating that the account has a managed resource.
	AccountManagedResourcesFlag *bool `json:"accountManagedResourcesFlag,omitempty" xmlrpc:"accountManagedResourcesFlag,omitempty"`

	// An account's status presented in a more detailed data type.
	AccountStatus *Account_Status `json:"accountStatus,omitempty" xmlrpc:"accountStatus,omitempty"`

	// A number reflecting the state of an account.
	AccountStatusId *int `json:"accountStatusId,omitempty" xmlrpc:"accountStatusId,omitempty"`

	// The billing item associated with an account's monthly discount.
	ActiveAccountDiscountBillingItem *Billing_Item `json:"activeAccountDiscountBillingItem,omitempty" xmlrpc:"activeAccountDiscountBillingItem,omitempty"`

	// A count of the active account software licenses owned by an account
	ActiveAccountLicenseCount *uint `json:"activeAccountLicenseCount,omitempty" xmlrpc:"activeAccountLicenseCount,omitempty"`

	// The active account software licenses owned by an account
	ActiveAccountLicenses []Software_AccountLicense `json:"activeAccountLicenses,omitempty" xmlrpc:"activeAccountLicenses,omitempty"`

	// A count of the active address(es) that belong to an account.
	ActiveAddressCount *uint `json:"activeAddressCount,omitempty" xmlrpc:"activeAddressCount,omitempty"`

	// The active address(es) that belong to an account.
	ActiveAddresses []Account_Address `json:"activeAddresses,omitempty" xmlrpc:"activeAddresses,omitempty"`

	// A count of all billing agreements for an account
	ActiveBillingAgreementCount *uint `json:"activeBillingAgreementCount,omitempty" xmlrpc:"activeBillingAgreementCount,omitempty"`

	// All billing agreements for an account
	ActiveBillingAgreements []Account_Agreement `json:"activeBillingAgreements,omitempty" xmlrpc:"activeBillingAgreements,omitempty"`

	// no documentation yet
	ActiveCatalystEnrollment *Catalyst_Enrollment `json:"activeCatalystEnrollment,omitempty" xmlrpc:"activeCatalystEnrollment,omitempty"`

	// A count of the account's active top level colocation containers.
	ActiveColocationContainerCount *uint `json:"activeColocationContainerCount,omitempty" xmlrpc:"activeColocationContainerCount,omitempty"`

	// The account's active top level colocation containers.
	ActiveColocationContainers []Billing_Item `json:"activeColocationContainers,omitempty" xmlrpc:"activeColocationContainers,omitempty"`

	// Account's currently active Flexible Credit enrollment.
	ActiveFlexibleCreditEnrollment *FlexibleCredit_Enrollment `json:"activeFlexibleCreditEnrollment,omitempty" xmlrpc:"activeFlexibleCreditEnrollment,omitempty"`

	// A count of
	ActiveNotificationSubscriberCount *uint `json:"activeNotificationSubscriberCount,omitempty" xmlrpc:"activeNotificationSubscriberCount,omitempty"`

	// no documentation yet
	ActiveNotificationSubscribers []Notification_Subscriber `json:"activeNotificationSubscribers,omitempty" xmlrpc:"activeNotificationSubscribers,omitempty"`

	// A count of an account's non-expired quotes.
	ActiveQuoteCount *uint `json:"activeQuoteCount,omitempty" xmlrpc:"activeQuoteCount,omitempty"`

	// An account's non-expired quotes.
	ActiveQuotes []Billing_Order_Quote `json:"activeQuotes,omitempty" xmlrpc:"activeQuotes,omitempty"`

	// A count of the virtual software licenses controlled by an account
	ActiveVirtualLicenseCount *uint `json:"activeVirtualLicenseCount,omitempty" xmlrpc:"activeVirtualLicenseCount,omitempty"`

	// The virtual software licenses controlled by an account
	ActiveVirtualLicenses []Software_VirtualLicense `json:"activeVirtualLicenses,omitempty" xmlrpc:"activeVirtualLicenses,omitempty"`

	// A count of an account's associated load balancers.
	AdcLoadBalancerCount *uint `json:"adcLoadBalancerCount,omitempty" xmlrpc:"adcLoadBalancerCount,omitempty"`

	// An account's associated load balancers.
	AdcLoadBalancers []Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress `json:"adcLoadBalancers,omitempty" xmlrpc:"adcLoadBalancers,omitempty"`

	// The first line of the mailing address belonging to an account.
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// The second line of the mailing address belonging to an account.
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// A count of all the address(es) that belong to an account.
	AddressCount *uint `json:"addressCount,omitempty" xmlrpc:"addressCount,omitempty"`

	// All the address(es) that belong to an account.
	Addresses []Account_Address `json:"addresses,omitempty" xmlrpc:"addresses,omitempty"`

	// An affiliate identifier associated with the customer account.
	AffiliateId *string `json:"affiliateId,omitempty" xmlrpc:"affiliateId,omitempty"`

	// The billing items that will be on an account's next invoice.
	AllBillingItems []Billing_Item `json:"allBillingItems,omitempty" xmlrpc:"allBillingItems,omitempty"`

	// A count of the billing items that will be on an account's next invoice.
	AllCommissionBillingItemCount *uint `json:"allCommissionBillingItemCount,omitempty" xmlrpc:"allCommissionBillingItemCount,omitempty"`

	// The billing items that will be on an account's next invoice.
	AllCommissionBillingItems []Billing_Item `json:"allCommissionBillingItems,omitempty" xmlrpc:"allCommissionBillingItems,omitempty"`

	// A count of the billing items that will be on an account's next invoice.
	AllRecurringTopLevelBillingItemCount *uint `json:"allRecurringTopLevelBillingItemCount,omitempty" xmlrpc:"allRecurringTopLevelBillingItemCount,omitempty"`

	// The billing items that will be on an account's next invoice.
	AllRecurringTopLevelBillingItems []Billing_Item `json:"allRecurringTopLevelBillingItems,omitempty" xmlrpc:"allRecurringTopLevelBillingItems,omitempty"`

	// The billing items that will be on an account's next invoice. Does not consider associated items.
	AllRecurringTopLevelBillingItemsUnfiltered []Billing_Item `json:"allRecurringTopLevelBillingItemsUnfiltered,omitempty" xmlrpc:"allRecurringTopLevelBillingItemsUnfiltered,omitempty"`

	// A count of the billing items that will be on an account's next invoice. Does not consider associated items.
	AllRecurringTopLevelBillingItemsUnfilteredCount *uint `json:"allRecurringTopLevelBillingItemsUnfilteredCount,omitempty" xmlrpc:"allRecurringTopLevelBillingItemsUnfilteredCount,omitempty"`

	// A count of the billing items that will be on an account's next invoice.
	AllSubnetBillingItemCount *uint `json:"allSubnetBillingItemCount,omitempty" xmlrpc:"allSubnetBillingItemCount,omitempty"`

	// The billing items that will be on an account's next invoice.
	AllSubnetBillingItems []Billing_Item `json:"allSubnetBillingItems,omitempty" xmlrpc:"allSubnetBillingItems,omitempty"`

	// A count of all billing items of an account.
	AllTopLevelBillingItemCount *uint `json:"allTopLevelBillingItemCount,omitempty" xmlrpc:"allTopLevelBillingItemCount,omitempty"`

	// All billing items of an account.
	AllTopLevelBillingItems []Billing_Item `json:"allTopLevelBillingItems,omitempty" xmlrpc:"allTopLevelBillingItems,omitempty"`

	// The billing items that will be on an account's next invoice. Does not consider associated items.
	AllTopLevelBillingItemsUnfiltered []Billing_Item `json:"allTopLevelBillingItemsUnfiltered,omitempty" xmlrpc:"allTopLevelBillingItemsUnfiltered,omitempty"`

	// A count of the billing items that will be on an account's next invoice. Does not consider associated items.
	AllTopLevelBillingItemsUnfilteredCount *uint `json:"allTopLevelBillingItemsUnfilteredCount,omitempty" xmlrpc:"allTopLevelBillingItemsUnfilteredCount,omitempty"`

	// Indicates whether this account is allowed to silently migrate to use IBMid Authentication.
	AllowIbmIdSilentMigrationFlag *bool `json:"allowIbmIdSilentMigrationFlag,omitempty" xmlrpc:"allowIbmIdSilentMigrationFlag,omitempty"`

	// The number of PPTP VPN users allowed on an account.
	AllowedPptpVpnQuantity *int `json:"allowedPptpVpnQuantity,omitempty" xmlrpc:"allowedPptpVpnQuantity,omitempty"`

	// Flag indicating if this account can be linked with Bluemix.
	AllowsBluemixAccountLinkingFlag *bool `json:"allowsBluemixAccountLinkingFlag,omitempty" xmlrpc:"allowsBluemixAccountLinkingFlag,omitempty"`

	// A secondary phone number assigned to an account.
	AlternatePhone *string `json:"alternatePhone,omitempty" xmlrpc:"alternatePhone,omitempty"`

	// A count of an account's associated application delivery controller records.
	ApplicationDeliveryControllerCount *uint `json:"applicationDeliveryControllerCount,omitempty" xmlrpc:"applicationDeliveryControllerCount,omitempty"`

	// An account's associated application delivery controller records.
	ApplicationDeliveryControllers []Network_Application_Delivery_Controller `json:"applicationDeliveryControllers,omitempty" xmlrpc:"applicationDeliveryControllers,omitempty"`

	// A count of the account attribute values for a SoftLayer customer account.
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// The account attribute values for a SoftLayer customer account.
	Attributes []Account_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// A count of the public network VLANs assigned to an account.
	AvailablePublicNetworkVlanCount *uint `json:"availablePublicNetworkVlanCount,omitempty" xmlrpc:"availablePublicNetworkVlanCount,omitempty"`

	// The public network VLANs assigned to an account.
	AvailablePublicNetworkVlans []Network_Vlan `json:"availablePublicNetworkVlans,omitempty" xmlrpc:"availablePublicNetworkVlans,omitempty"`

	// The account balance of a SoftLayer customer account. An account's balance is the amount of money owed to SoftLayer by the account holder, returned as a floating point number with two decimal places, measured in US Dollars ($USD). A negative account balance means the account holder has overpaid and is owed money by SoftLayer.
	Balance *Float64 `json:"balance,omitempty" xmlrpc:"balance,omitempty"`

	// A count of the bandwidth allotments for an account.
	BandwidthAllotmentCount *uint `json:"bandwidthAllotmentCount,omitempty" xmlrpc:"bandwidthAllotmentCount,omitempty"`

	// The bandwidth allotments for an account.
	BandwidthAllotments []Network_Bandwidth_Version1_Allotment `json:"bandwidthAllotments,omitempty" xmlrpc:"bandwidthAllotments,omitempty"`

	// The bandwidth allotments for an account currently over allocation.
	BandwidthAllotmentsOverAllocation []Network_Bandwidth_Version1_Allotment `json:"bandwidthAllotmentsOverAllocation,omitempty" xmlrpc:"bandwidthAllotmentsOverAllocation,omitempty"`

	// A count of the bandwidth allotments for an account currently over allocation.
	BandwidthAllotmentsOverAllocationCount *uint `json:"bandwidthAllotmentsOverAllocationCount,omitempty" xmlrpc:"bandwidthAllotmentsOverAllocationCount,omitempty"`

	// The bandwidth allotments for an account projected to go over allocation.
	BandwidthAllotmentsProjectedOverAllocation []Network_Bandwidth_Version1_Allotment `json:"bandwidthAllotmentsProjectedOverAllocation,omitempty" xmlrpc:"bandwidthAllotmentsProjectedOverAllocation,omitempty"`

	// A count of the bandwidth allotments for an account projected to go over allocation.
	BandwidthAllotmentsProjectedOverAllocationCount *uint `json:"bandwidthAllotmentsProjectedOverAllocationCount,omitempty" xmlrpc:"bandwidthAllotmentsProjectedOverAllocationCount,omitempty"`

	// A count of an account's associated bare metal server objects.
	BareMetalInstanceCount *uint `json:"bareMetalInstanceCount,omitempty" xmlrpc:"bareMetalInstanceCount,omitempty"`

	// An account's associated bare metal server objects.
	BareMetalInstances []Hardware `json:"bareMetalInstances,omitempty" xmlrpc:"bareMetalInstances,omitempty"`

	// A count of all billing agreements for an account
	BillingAgreementCount *uint `json:"billingAgreementCount,omitempty" xmlrpc:"billingAgreementCount,omitempty"`

	// All billing agreements for an account
	BillingAgreements []Account_Agreement `json:"billingAgreements,omitempty" xmlrpc:"billingAgreements,omitempty"`

	// An account's billing information.
	BillingInfo *Billing_Info `json:"billingInfo,omitempty" xmlrpc:"billingInfo,omitempty"`

	// A count of private template group objects (parent and children) and the shared template group objects (parent only) for an account.
	BlockDeviceTemplateGroupCount *uint `json:"blockDeviceTemplateGroupCount,omitempty" xmlrpc:"blockDeviceTemplateGroupCount,omitempty"`

	// Private template group objects (parent and children) and the shared template group objects (parent only) for an account.
	BlockDeviceTemplateGroups []Virtual_Guest_Block_Device_Template_Group `json:"blockDeviceTemplateGroups,omitempty" xmlrpc:"blockDeviceTemplateGroups,omitempty"`

	// Indicates whether this account requires blue id authentication.
	BlueIdAuthenticationRequiredFlag *bool `json:"blueIdAuthenticationRequiredFlag,omitempty" xmlrpc:"blueIdAuthenticationRequiredFlag,omitempty"`

	// Returns true if this account is linked to IBM Bluemix, false if not.
	BluemixLinkedFlag *bool `json:"bluemixLinkedFlag,omitempty" xmlrpc:"bluemixLinkedFlag,omitempty"`

	// no documentation yet
	Brand *Brand `json:"brand,omitempty" xmlrpc:"brand,omitempty"`

	// no documentation yet
	BrandAccountFlag *bool `json:"brandAccountFlag,omitempty" xmlrpc:"brandAccountFlag,omitempty"`

	// The Brand tied to an account.
	BrandId *int `json:"brandId,omitempty" xmlrpc:"brandId,omitempty"`

	// The brand keyName.
	BrandKeyName *string `json:"brandKeyName,omitempty" xmlrpc:"brandKeyName,omitempty"`

	// Indicating whether this account can order additional Vlans.
	CanOrderAdditionalVlansFlag *bool `json:"canOrderAdditionalVlansFlag,omitempty" xmlrpc:"canOrderAdditionalVlansFlag,omitempty"`

	// A count of an account's active carts.
	CartCount *uint `json:"cartCount,omitempty" xmlrpc:"cartCount,omitempty"`

	// An account's active carts.
	Carts []Billing_Order_Quote `json:"carts,omitempty" xmlrpc:"carts,omitempty"`

	// A count of
	CatalystEnrollmentCount *uint `json:"catalystEnrollmentCount,omitempty" xmlrpc:"catalystEnrollmentCount,omitempty"`

	// no documentation yet
	CatalystEnrollments []Catalyst_Enrollment `json:"catalystEnrollments,omitempty" xmlrpc:"catalystEnrollments,omitempty"`

	// A count of an account's associated CDN accounts.
	CdnAccountCount *uint `json:"cdnAccountCount,omitempty" xmlrpc:"cdnAccountCount,omitempty"`

	// An account's associated CDN accounts.
	CdnAccounts []Network_ContentDelivery_Account `json:"cdnAccounts,omitempty" xmlrpc:"cdnAccounts,omitempty"`

	// The city of the mailing address belonging to an account.
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// Whether an account is exempt from taxes on their invoices.
	ClaimedTaxExemptTxFlag *bool `json:"claimedTaxExemptTxFlag,omitempty" xmlrpc:"claimedTaxExemptTxFlag,omitempty"`

	// A count of all closed tickets associated with an account.
	ClosedTicketCount *uint `json:"closedTicketCount,omitempty" xmlrpc:"closedTicketCount,omitempty"`

	// All closed tickets associated with an account.
	ClosedTickets []Ticket `json:"closedTickets,omitempty" xmlrpc:"closedTickets,omitempty"`

	// The company name associated with an account.
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// A two-letter abbreviation of the country in the mailing address belonging to an account.
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// The date an account was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A count of datacenters which contain subnets that the account has access to route.
	DatacentersWithSubnetAllocationCount *uint `json:"datacentersWithSubnetAllocationCount,omitempty" xmlrpc:"datacentersWithSubnetAllocationCount,omitempty"`

	// Datacenters which contain subnets that the account has access to route.
	DatacentersWithSubnetAllocations []Location `json:"datacentersWithSubnetAllocations,omitempty" xmlrpc:"datacentersWithSubnetAllocations,omitempty"`

	// Device Fingerprint Identifier - Used internally and can safely be ignored.
	DeviceFingerprintId *string `json:"deviceFingerprintId,omitempty" xmlrpc:"deviceFingerprintId,omitempty"`

	// A flag indicating whether payments are processed for this account.
	DisablePaymentProcessingFlag *bool `json:"disablePaymentProcessingFlag,omitempty" xmlrpc:"disablePaymentProcessingFlag,omitempty"`

	// A count of the SoftLayer employees that an account is assigned to.
	DisplaySupportRepresentativeAssignmentCount *uint `json:"displaySupportRepresentativeAssignmentCount,omitempty" xmlrpc:"displaySupportRepresentativeAssignmentCount,omitempty"`

	// The SoftLayer employees that an account is assigned to.
	DisplaySupportRepresentativeAssignments []Account_Attachment_Employee `json:"displaySupportRepresentativeAssignments,omitempty" xmlrpc:"displaySupportRepresentativeAssignments,omitempty"`

	// A count of the DNS domains associated with an account.
	DomainCount *uint `json:"domainCount,omitempty" xmlrpc:"domainCount,omitempty"`

	// A count of
	DomainRegistrationCount *uint `json:"domainRegistrationCount,omitempty" xmlrpc:"domainRegistrationCount,omitempty"`

	// no documentation yet
	DomainRegistrations []Dns_Domain_Registration `json:"domainRegistrations,omitempty" xmlrpc:"domainRegistrations,omitempty"`

	// The DNS domains associated with an account.
	Domains []Dns_Domain `json:"domains,omitempty" xmlrpc:"domains,omitempty"`

	// A count of the DNS domains associated with an account that were not created as a result of a secondary DNS zone transfer.
	DomainsWithoutSecondaryDnsRecordCount *uint `json:"domainsWithoutSecondaryDnsRecordCount,omitempty" xmlrpc:"domainsWithoutSecondaryDnsRecordCount,omitempty"`

	// The DNS domains associated with an account that were not created as a result of a secondary DNS zone transfer.
	DomainsWithoutSecondaryDnsRecords []Dns_Domain `json:"domainsWithoutSecondaryDnsRecords,omitempty" xmlrpc:"domainsWithoutSecondaryDnsRecords,omitempty"`

	// A general email address assigned to an account.
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// The total capacity of Legacy EVault Volumes on an account, in GB.
	EvaultCapacityGB *uint `json:"evaultCapacityGB,omitempty" xmlrpc:"evaultCapacityGB,omitempty"`

	// A count of an account's master EVault user. This is only used when an account has EVault service.
	EvaultMasterUserCount *uint `json:"evaultMasterUserCount,omitempty" xmlrpc:"evaultMasterUserCount,omitempty"`

	// An account's master EVault user. This is only used when an account has EVault service.
	EvaultMasterUsers []Account_Password `json:"evaultMasterUsers,omitempty" xmlrpc:"evaultMasterUsers,omitempty"`

	// An account's associated EVault storage volumes.
	EvaultNetworkStorage []Network_Storage `json:"evaultNetworkStorage,omitempty" xmlrpc:"evaultNetworkStorage,omitempty"`

	// A count of an account's associated EVault storage volumes.
	EvaultNetworkStorageCount *uint `json:"evaultNetworkStorageCount,omitempty" xmlrpc:"evaultNetworkStorageCount,omitempty"`

	// A count of stored security certificates that are expired (ie. SSL)
	ExpiredSecurityCertificateCount *uint `json:"expiredSecurityCertificateCount,omitempty" xmlrpc:"expiredSecurityCertificateCount,omitempty"`

	// Stored security certificates that are expired (ie. SSL)
	ExpiredSecurityCertificates []Security_Certificate `json:"expiredSecurityCertificates,omitempty" xmlrpc:"expiredSecurityCertificates,omitempty"`

	// A count of logs of who entered a colocation area which is assigned to this account, or when a user under this account enters a datacenter.
	FacilityLogCount *uint `json:"facilityLogCount,omitempty" xmlrpc:"facilityLogCount,omitempty"`

	// Logs of who entered a colocation area which is assigned to this account, or when a user under this account enters a datacenter.
	FacilityLogs []User_Access_Facility_Log `json:"facilityLogs,omitempty" xmlrpc:"facilityLogs,omitempty"`

	// A fax phone number assigned to an account.
	FaxPhone *string `json:"faxPhone,omitempty" xmlrpc:"faxPhone,omitempty"`

	// Each customer account is listed under a single individual. This is that individual's first name.
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// A count of all of the account's current and former Flexible Credit enrollments.
	FlexibleCreditEnrollmentCount *uint `json:"flexibleCreditEnrollmentCount,omitempty" xmlrpc:"flexibleCreditEnrollmentCount,omitempty"`

	// All of the account's current and former Flexible Credit enrollments.
	FlexibleCreditEnrollments []FlexibleCredit_Enrollment `json:"flexibleCreditEnrollments,omitempty" xmlrpc:"flexibleCreditEnrollments,omitempty"`

	// A count of
	GlobalIpRecordCount *uint `json:"globalIpRecordCount,omitempty" xmlrpc:"globalIpRecordCount,omitempty"`

	// no documentation yet
	GlobalIpRecords []Network_Subnet_IpAddress_Global `json:"globalIpRecords,omitempty" xmlrpc:"globalIpRecords,omitempty"`

	// A count of
	GlobalIpv4RecordCount *uint `json:"globalIpv4RecordCount,omitempty" xmlrpc:"globalIpv4RecordCount,omitempty"`

	// no documentation yet
	GlobalIpv4Records []Network_Subnet_IpAddress_Global `json:"globalIpv4Records,omitempty" xmlrpc:"globalIpv4Records,omitempty"`

	// A count of
	GlobalIpv6RecordCount *uint `json:"globalIpv6RecordCount,omitempty" xmlrpc:"globalIpv6RecordCount,omitempty"`

	// no documentation yet
	GlobalIpv6Records []Network_Subnet_IpAddress_Global `json:"globalIpv6Records,omitempty" xmlrpc:"globalIpv6Records,omitempty"`

	// A count of the global load balancer accounts for a softlayer customer account.
	GlobalLoadBalancerAccountCount *uint `json:"globalLoadBalancerAccountCount,omitempty" xmlrpc:"globalLoadBalancerAccountCount,omitempty"`

	// The global load balancer accounts for a softlayer customer account.
	GlobalLoadBalancerAccounts []Network_LoadBalancer_Global_Account `json:"globalLoadBalancerAccounts,omitempty" xmlrpc:"globalLoadBalancerAccounts,omitempty"`

	// An account's associated hardware objects.
	Hardware []Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// A count of an account's associated hardware objects.
	HardwareCount *uint `json:"hardwareCount,omitempty" xmlrpc:"hardwareCount,omitempty"`

	// An account's associated hardware objects currently over bandwidth allocation.
	HardwareOverBandwidthAllocation []Hardware `json:"hardwareOverBandwidthAllocation,omitempty" xmlrpc:"hardwareOverBandwidthAllocation,omitempty"`

	// A count of an account's associated hardware objects currently over bandwidth allocation.
	HardwareOverBandwidthAllocationCount *uint `json:"hardwareOverBandwidthAllocationCount,omitempty" xmlrpc:"hardwareOverBandwidthAllocationCount,omitempty"`

	// An account's associated hardware objects projected to go over bandwidth allocation.
	HardwareProjectedOverBandwidthAllocation []Hardware `json:"hardwareProjectedOverBandwidthAllocation,omitempty" xmlrpc:"hardwareProjectedOverBandwidthAllocation,omitempty"`

	// A count of an account's associated hardware objects projected to go over bandwidth allocation.
	HardwareProjectedOverBandwidthAllocationCount *uint `json:"hardwareProjectedOverBandwidthAllocationCount,omitempty" xmlrpc:"hardwareProjectedOverBandwidthAllocationCount,omitempty"`

	// All hardware associated with an account that has the cPanel web hosting control panel installed.
	HardwareWithCpanel []Hardware `json:"hardwareWithCpanel,omitempty" xmlrpc:"hardwareWithCpanel,omitempty"`

	// A count of all hardware associated with an account that has the cPanel web hosting control panel installed.
	HardwareWithCpanelCount *uint `json:"hardwareWithCpanelCount,omitempty" xmlrpc:"hardwareWithCpanelCount,omitempty"`

	// All hardware associated with an account that has the Helm web hosting control panel installed.
	HardwareWithHelm []Hardware `json:"hardwareWithHelm,omitempty" xmlrpc:"hardwareWithHelm,omitempty"`

	// A count of all hardware associated with an account that has the Helm web hosting control panel installed.
	HardwareWithHelmCount *uint `json:"hardwareWithHelmCount,omitempty" xmlrpc:"hardwareWithHelmCount,omitempty"`

	// All hardware associated with an account that has McAfee Secure software components.
	HardwareWithMcafee []Hardware `json:"hardwareWithMcafee,omitempty" xmlrpc:"hardwareWithMcafee,omitempty"`

	// All hardware associated with an account that has McAfee Secure AntiVirus for Redhat software components.
	HardwareWithMcafeeAntivirusRedhat []Hardware `json:"hardwareWithMcafeeAntivirusRedhat,omitempty" xmlrpc:"hardwareWithMcafeeAntivirusRedhat,omitempty"`

	// A count of all hardware associated with an account that has McAfee Secure AntiVirus for Redhat software components.
	HardwareWithMcafeeAntivirusRedhatCount *uint `json:"hardwareWithMcafeeAntivirusRedhatCount,omitempty" xmlrpc:"hardwareWithMcafeeAntivirusRedhatCount,omitempty"`

	// A count of all hardware associated with an account that has McAfee Secure AntiVirus for Windows software components.
	HardwareWithMcafeeAntivirusWindowCount *uint `json:"hardwareWithMcafeeAntivirusWindowCount,omitempty" xmlrpc:"hardwareWithMcafeeAntivirusWindowCount,omitempty"`

	// All hardware associated with an account that has McAfee Secure AntiVirus for Windows software components.
	HardwareWithMcafeeAntivirusWindows []Hardware `json:"hardwareWithMcafeeAntivirusWindows,omitempty" xmlrpc:"hardwareWithMcafeeAntivirusWindows,omitempty"`

	// A count of all hardware associated with an account that has McAfee Secure software components.
	HardwareWithMcafeeCount *uint `json:"hardwareWithMcafeeCount,omitempty" xmlrpc:"hardwareWithMcafeeCount,omitempty"`

	// All hardware associated with an account that has McAfee Secure Intrusion Detection System software components.
	HardwareWithMcafeeIntrusionDetectionSystem []Hardware `json:"hardwareWithMcafeeIntrusionDetectionSystem,omitempty" xmlrpc:"hardwareWithMcafeeIntrusionDetectionSystem,omitempty"`

	// A count of all hardware associated with an account that has McAfee Secure Intrusion Detection System software components.
	HardwareWithMcafeeIntrusionDetectionSystemCount *uint `json:"hardwareWithMcafeeIntrusionDetectionSystemCount,omitempty" xmlrpc:"hardwareWithMcafeeIntrusionDetectionSystemCount,omitempty"`

	// All hardware associated with an account that has the Plesk web hosting control panel installed.
	HardwareWithPlesk []Hardware `json:"hardwareWithPlesk,omitempty" xmlrpc:"hardwareWithPlesk,omitempty"`

	// A count of all hardware associated with an account that has the Plesk web hosting control panel installed.
	HardwareWithPleskCount *uint `json:"hardwareWithPleskCount,omitempty" xmlrpc:"hardwareWithPleskCount,omitempty"`

	// All hardware associated with an account that has the QuantaStor storage system installed.
	HardwareWithQuantastor []Hardware `json:"hardwareWithQuantastor,omitempty" xmlrpc:"hardwareWithQuantastor,omitempty"`

	// A count of all hardware associated with an account that has the QuantaStor storage system installed.
	HardwareWithQuantastorCount *uint `json:"hardwareWithQuantastorCount,omitempty" xmlrpc:"hardwareWithQuantastorCount,omitempty"`

	// All hardware associated with an account that has the Urchin web traffic analytics package installed.
	HardwareWithUrchin []Hardware `json:"hardwareWithUrchin,omitempty" xmlrpc:"hardwareWithUrchin,omitempty"`

	// A count of all hardware associated with an account that has the Urchin web traffic analytics package installed.
	HardwareWithUrchinCount *uint `json:"hardwareWithUrchinCount,omitempty" xmlrpc:"hardwareWithUrchinCount,omitempty"`

	// A count of all hardware associated with an account that is running a version of the Microsoft Windows operating system.
	HardwareWithWindowCount *uint `json:"hardwareWithWindowCount,omitempty" xmlrpc:"hardwareWithWindowCount,omitempty"`

	// All hardware associated with an account that is running a version of the Microsoft Windows operating system.
	HardwareWithWindows []Hardware `json:"hardwareWithWindows,omitempty" xmlrpc:"hardwareWithWindows,omitempty"`

	// Return 1 if one of the account's hardware has the EVault Bare Metal Server Restore Plugin otherwise 0.
	HasEvaultBareMetalRestorePluginFlag *bool `json:"hasEvaultBareMetalRestorePluginFlag,omitempty" xmlrpc:"hasEvaultBareMetalRestorePluginFlag,omitempty"`

	// Return 1 if one of the account's hardware has an installation of Idera Server Backup otherwise 0.
	HasIderaBareMetalRestorePluginFlag *bool `json:"hasIderaBareMetalRestorePluginFlag,omitempty" xmlrpc:"hasIderaBareMetalRestorePluginFlag,omitempty"`

	// The number of orders in a PENDING status for a SoftLayer customer account.
	HasPendingOrder *uint `json:"hasPendingOrder,omitempty" xmlrpc:"hasPendingOrder,omitempty"`

	// Return 1 if one of the account's hardware has an installation of R1Soft CDP otherwise 0.
	HasR1softBareMetalRestorePluginFlag *bool `json:"hasR1softBareMetalRestorePluginFlag,omitempty" xmlrpc:"hasR1softBareMetalRestorePluginFlag,omitempty"`

	// A count of an account's associated hourly bare metal server objects.
	HourlyBareMetalInstanceCount *uint `json:"hourlyBareMetalInstanceCount,omitempty" xmlrpc:"hourlyBareMetalInstanceCount,omitempty"`

	// An account's associated hourly bare metal server objects.
	HourlyBareMetalInstances []Hardware `json:"hourlyBareMetalInstances,omitempty" xmlrpc:"hourlyBareMetalInstances,omitempty"`

	// A count of hourly service billing items that will be on an account's next invoice.
	HourlyServiceBillingItemCount *uint `json:"hourlyServiceBillingItemCount,omitempty" xmlrpc:"hourlyServiceBillingItemCount,omitempty"`

	// Hourly service billing items that will be on an account's next invoice.
	HourlyServiceBillingItems []Billing_Item `json:"hourlyServiceBillingItems,omitempty" xmlrpc:"hourlyServiceBillingItems,omitempty"`

	// A count of an account's associated hourly virtual guest objects.
	HourlyVirtualGuestCount *uint `json:"hourlyVirtualGuestCount,omitempty" xmlrpc:"hourlyVirtualGuestCount,omitempty"`

	// An account's associated hourly virtual guest objects.
	HourlyVirtualGuests []Virtual_Guest `json:"hourlyVirtualGuests,omitempty" xmlrpc:"hourlyVirtualGuests,omitempty"`

	// An account's associated Virtual Storage volumes.
	HubNetworkStorage []Network_Storage `json:"hubNetworkStorage,omitempty" xmlrpc:"hubNetworkStorage,omitempty"`

	// A count of an account's associated Virtual Storage volumes.
	HubNetworkStorageCount *uint `json:"hubNetworkStorageCount,omitempty" xmlrpc:"hubNetworkStorageCount,omitempty"`

	// Timestamp representing the point in time when an account is required to use IBMid authentication.
	IbmIdMigrationExpirationTimestamp *string `json:"ibmIdMigrationExpirationTimestamp,omitempty" xmlrpc:"ibmIdMigrationExpirationTimestamp,omitempty"`

	// A customer account's internal identifier. Account numbers are typically preceded by the string "SL" in the customer portal. Every SoftLayer account has at least one portal user whose username follows the "SL" + account number naming scheme.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of
	InternalNoteCount *uint `json:"internalNoteCount,omitempty" xmlrpc:"internalNoteCount,omitempty"`

	// no documentation yet
	InternalNotes []Account_Note `json:"internalNotes,omitempty" xmlrpc:"internalNotes,omitempty"`

	// A count of an account's associated billing invoices.
	InvoiceCount *uint `json:"invoiceCount,omitempty" xmlrpc:"invoiceCount,omitempty"`

	// An account's associated billing invoices.
	Invoices []Billing_Invoice `json:"invoices,omitempty" xmlrpc:"invoices,omitempty"`

	// A count of
	IpAddressCount *uint `json:"ipAddressCount,omitempty" xmlrpc:"ipAddressCount,omitempty"`

	// no documentation yet
	IpAddresses []Network_Subnet_IpAddress `json:"ipAddresses,omitempty" xmlrpc:"ipAddresses,omitempty"`

	// A flag indicating if an account belongs to a reseller or not.
	IsReseller *int `json:"isReseller,omitempty" xmlrpc:"isReseller,omitempty"`

	// An account's associated iSCSI storage volumes.
	IscsiNetworkStorage []Network_Storage `json:"iscsiNetworkStorage,omitempty" xmlrpc:"iscsiNetworkStorage,omitempty"`

	// A count of an account's associated iSCSI storage volumes.
	IscsiNetworkStorageCount *uint `json:"iscsiNetworkStorageCount,omitempty" xmlrpc:"iscsiNetworkStorageCount,omitempty"`

	// The most recently canceled billing item.
	LastCanceledBillingItem *Billing_Item `json:"lastCanceledBillingItem,omitempty" xmlrpc:"lastCanceledBillingItem,omitempty"`

	// The most recent cancelled server billing item.
	LastCancelledServerBillingItem *Billing_Item `json:"lastCancelledServerBillingItem,omitempty" xmlrpc:"lastCancelledServerBillingItem,omitempty"`

	// A count of the five most recently closed abuse tickets associated with an account.
	LastFiveClosedAbuseTicketCount *uint `json:"lastFiveClosedAbuseTicketCount,omitempty" xmlrpc:"lastFiveClosedAbuseTicketCount,omitempty"`

	// The five most recently closed abuse tickets associated with an account.
	LastFiveClosedAbuseTickets []Ticket `json:"lastFiveClosedAbuseTickets,omitempty" xmlrpc:"lastFiveClosedAbuseTickets,omitempty"`

	// A count of the five most recently closed accounting tickets associated with an account.
	LastFiveClosedAccountingTicketCount *uint `json:"lastFiveClosedAccountingTicketCount,omitempty" xmlrpc:"lastFiveClosedAccountingTicketCount,omitempty"`

	// The five most recently closed accounting tickets associated with an account.
	LastFiveClosedAccountingTickets []Ticket `json:"lastFiveClosedAccountingTickets,omitempty" xmlrpc:"lastFiveClosedAccountingTickets,omitempty"`

	// A count of the five most recently closed tickets that do not belong to the abuse, accounting, sales, or support groups associated with an account.
	LastFiveClosedOtherTicketCount *uint `json:"lastFiveClosedOtherTicketCount,omitempty" xmlrpc:"lastFiveClosedOtherTicketCount,omitempty"`

	// The five most recently closed tickets that do not belong to the abuse, accounting, sales, or support groups associated with an account.
	LastFiveClosedOtherTickets []Ticket `json:"lastFiveClosedOtherTickets,omitempty" xmlrpc:"lastFiveClosedOtherTickets,omitempty"`

	// A count of the five most recently closed sales tickets associated with an account.
	LastFiveClosedSalesTicketCount *uint `json:"lastFiveClosedSalesTicketCount,omitempty" xmlrpc:"lastFiveClosedSalesTicketCount,omitempty"`

	// The five most recently closed sales tickets associated with an account.
	LastFiveClosedSalesTickets []Ticket `json:"lastFiveClosedSalesTickets,omitempty" xmlrpc:"lastFiveClosedSalesTickets,omitempty"`

	// A count of the five most recently closed support tickets associated with an account.
	LastFiveClosedSupportTicketCount *uint `json:"lastFiveClosedSupportTicketCount,omitempty" xmlrpc:"lastFiveClosedSupportTicketCount,omitempty"`

	// The five most recently closed support tickets associated with an account.
	LastFiveClosedSupportTickets []Ticket `json:"lastFiveClosedSupportTickets,omitempty" xmlrpc:"lastFiveClosedSupportTickets,omitempty"`

	// A count of the five most recently closed tickets associated with an account.
	LastFiveClosedTicketCount *uint `json:"lastFiveClosedTicketCount,omitempty" xmlrpc:"lastFiveClosedTicketCount,omitempty"`

	// The five most recently closed tickets associated with an account.
	LastFiveClosedTickets []Ticket `json:"lastFiveClosedTickets,omitempty" xmlrpc:"lastFiveClosedTickets,omitempty"`

	// Each customer account is listed under a single individual. This is that individual's last name.
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// Whether an account has late fee protection.
	LateFeeProtectionFlag *bool `json:"lateFeeProtectionFlag,omitempty" xmlrpc:"lateFeeProtectionFlag,omitempty"`

	// An account's most recent billing date.
	LatestBillDate *Time `json:"latestBillDate,omitempty" xmlrpc:"latestBillDate,omitempty"`

	// An account's latest recurring invoice.
	LatestRecurringInvoice *Billing_Invoice `json:"latestRecurringInvoice,omitempty" xmlrpc:"latestRecurringInvoice,omitempty"`

	// An account's latest recurring pending invoice.
	LatestRecurringPendingInvoice *Billing_Invoice `json:"latestRecurringPendingInvoice,omitempty" xmlrpc:"latestRecurringPendingInvoice,omitempty"`

	// A count of the legacy bandwidth allotments for an account.
	LegacyBandwidthAllotmentCount *uint `json:"legacyBandwidthAllotmentCount,omitempty" xmlrpc:"legacyBandwidthAllotmentCount,omitempty"`

	// The legacy bandwidth allotments for an account.
	LegacyBandwidthAllotments []Network_Bandwidth_Version1_Allotment `json:"legacyBandwidthAllotments,omitempty" xmlrpc:"legacyBandwidthAllotments,omitempty"`

	// The total capacity of Legacy iSCSI Volumes on an account, in GB.
	LegacyIscsiCapacityGB *uint `json:"legacyIscsiCapacityGB,omitempty" xmlrpc:"legacyIscsiCapacityGB,omitempty"`

	// A count of an account's associated load balancers.
	LoadBalancerCount *uint `json:"loadBalancerCount,omitempty" xmlrpc:"loadBalancerCount,omitempty"`

	// An account's associated load balancers.
	LoadBalancers []Network_LoadBalancer_VirtualIpAddress `json:"loadBalancers,omitempty" xmlrpc:"loadBalancers,omitempty"`

	// The total capacity of Legacy lockbox Volumes on an account, in GB.
	LockboxCapacityGB *uint `json:"lockboxCapacityGB,omitempty" xmlrpc:"lockboxCapacityGB,omitempty"`

	// An account's associated Lockbox storage volumes.
	LockboxNetworkStorage []Network_Storage `json:"lockboxNetworkStorage,omitempty" xmlrpc:"lockboxNetworkStorage,omitempty"`

	// A count of an account's associated Lockbox storage volumes.
	LockboxNetworkStorageCount *uint `json:"lockboxNetworkStorageCount,omitempty" xmlrpc:"lockboxNetworkStorageCount,omitempty"`

	// no documentation yet
	ManualPaymentsUnderReview []Billing_Payment_Card_ManualPayment `json:"manualPaymentsUnderReview,omitempty" xmlrpc:"manualPaymentsUnderReview,omitempty"`

	// A count of
	ManualPaymentsUnderReviewCount *uint `json:"manualPaymentsUnderReviewCount,omitempty" xmlrpc:"manualPaymentsUnderReviewCount,omitempty"`

	// An account's master user.
	MasterUser *User_Customer `json:"masterUser,omitempty" xmlrpc:"masterUser,omitempty"`

	// A count of an account's media transfer service requests.
	MediaDataTransferRequestCount *uint `json:"mediaDataTransferRequestCount,omitempty" xmlrpc:"mediaDataTransferRequestCount,omitempty"`

	// An account's media transfer service requests.
	MediaDataTransferRequests []Account_Media_Data_Transfer_Request `json:"mediaDataTransferRequests,omitempty" xmlrpc:"mediaDataTransferRequests,omitempty"`

	// A count of an account's associated Message Queue accounts.
	MessageQueueAccountCount *uint `json:"messageQueueAccountCount,omitempty" xmlrpc:"messageQueueAccountCount,omitempty"`

	// An account's associated Message Queue accounts.
	MessageQueueAccounts []Network_Message_Queue `json:"messageQueueAccounts,omitempty" xmlrpc:"messageQueueAccounts,omitempty"`

	// The date an account was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A count of an account's associated monthly bare metal server objects.
	MonthlyBareMetalInstanceCount *uint `json:"monthlyBareMetalInstanceCount,omitempty" xmlrpc:"monthlyBareMetalInstanceCount,omitempty"`

	// An account's associated monthly bare metal server objects.
	MonthlyBareMetalInstances []Hardware `json:"monthlyBareMetalInstances,omitempty" xmlrpc:"monthlyBareMetalInstances,omitempty"`

	// A count of an account's associated monthly virtual guest objects.
	MonthlyVirtualGuestCount *uint `json:"monthlyVirtualGuestCount,omitempty" xmlrpc:"monthlyVirtualGuestCount,omitempty"`

	// An account's associated monthly virtual guest objects.
	MonthlyVirtualGuests []Virtual_Guest `json:"monthlyVirtualGuests,omitempty" xmlrpc:"monthlyVirtualGuests,omitempty"`

	// An account's associated NAS storage volumes.
	NasNetworkStorage []Network_Storage `json:"nasNetworkStorage,omitempty" xmlrpc:"nasNetworkStorage,omitempty"`

	// A count of an account's associated NAS storage volumes.
	NasNetworkStorageCount *uint `json:"nasNetworkStorageCount,omitempty" xmlrpc:"nasNetworkStorageCount,omitempty"`

	// Whether or not this account can define their own networks.
	NetworkCreationFlag *bool `json:"networkCreationFlag,omitempty" xmlrpc:"networkCreationFlag,omitempty"`

	// A count of all network gateway devices on this account.
	NetworkGatewayCount *uint `json:"networkGatewayCount,omitempty" xmlrpc:"networkGatewayCount,omitempty"`

	// All network gateway devices on this account.
	NetworkGateways []Network_Gateway `json:"networkGateways,omitempty" xmlrpc:"networkGateways,omitempty"`

	// An account's associated network hardware.
	NetworkHardware []Hardware `json:"networkHardware,omitempty" xmlrpc:"networkHardware,omitempty"`

	// A count of an account's associated network hardware.
	NetworkHardwareCount *uint `json:"networkHardwareCount,omitempty" xmlrpc:"networkHardwareCount,omitempty"`

	// A count of
	NetworkMessageDeliveryAccountCount *uint `json:"networkMessageDeliveryAccountCount,omitempty" xmlrpc:"networkMessageDeliveryAccountCount,omitempty"`

	// no documentation yet
	NetworkMessageDeliveryAccounts []Network_Message_Delivery `json:"networkMessageDeliveryAccounts,omitempty" xmlrpc:"networkMessageDeliveryAccounts,omitempty"`

	// Hardware which is currently experiencing a service failure.
	NetworkMonitorDownHardware []Hardware `json:"networkMonitorDownHardware,omitempty" xmlrpc:"networkMonitorDownHardware,omitempty"`

	// A count of hardware which is currently experiencing a service failure.
	NetworkMonitorDownHardwareCount *uint `json:"networkMonitorDownHardwareCount,omitempty" xmlrpc:"networkMonitorDownHardwareCount,omitempty"`

	// A count of virtual guest which is currently experiencing a service failure.
	NetworkMonitorDownVirtualGuestCount *uint `json:"networkMonitorDownVirtualGuestCount,omitempty" xmlrpc:"networkMonitorDownVirtualGuestCount,omitempty"`

	// Virtual guest which is currently experiencing a service failure.
	NetworkMonitorDownVirtualGuests []Virtual_Guest `json:"networkMonitorDownVirtualGuests,omitempty" xmlrpc:"networkMonitorDownVirtualGuests,omitempty"`

	// Hardware which is currently recovering from a service failure.
	NetworkMonitorRecoveringHardware []Hardware `json:"networkMonitorRecoveringHardware,omitempty" xmlrpc:"networkMonitorRecoveringHardware,omitempty"`

	// A count of hardware which is currently recovering from a service failure.
	NetworkMonitorRecoveringHardwareCount *uint `json:"networkMonitorRecoveringHardwareCount,omitempty" xmlrpc:"networkMonitorRecoveringHardwareCount,omitempty"`

	// A count of virtual guest which is currently recovering from a service failure.
	NetworkMonitorRecoveringVirtualGuestCount *uint `json:"networkMonitorRecoveringVirtualGuestCount,omitempty" xmlrpc:"networkMonitorRecoveringVirtualGuestCount,omitempty"`

	// Virtual guest which is currently recovering from a service failure.
	NetworkMonitorRecoveringVirtualGuests []Virtual_Guest `json:"networkMonitorRecoveringVirtualGuests,omitempty" xmlrpc:"networkMonitorRecoveringVirtualGuests,omitempty"`

	// Hardware which is currently online.
	NetworkMonitorUpHardware []Hardware `json:"networkMonitorUpHardware,omitempty" xmlrpc:"networkMonitorUpHardware,omitempty"`

	// A count of hardware which is currently online.
	NetworkMonitorUpHardwareCount *uint `json:"networkMonitorUpHardwareCount,omitempty" xmlrpc:"networkMonitorUpHardwareCount,omitempty"`

	// A count of virtual guest which is currently online.
	NetworkMonitorUpVirtualGuestCount *uint `json:"networkMonitorUpVirtualGuestCount,omitempty" xmlrpc:"networkMonitorUpVirtualGuestCount,omitempty"`

	// Virtual guest which is currently online.
	NetworkMonitorUpVirtualGuests []Virtual_Guest `json:"networkMonitorUpVirtualGuests,omitempty" xmlrpc:"networkMonitorUpVirtualGuests,omitempty"`

	// An account's associated storage volumes. This includes Lockbox, NAS, EVault, and iSCSI volumes.
	NetworkStorage []Network_Storage `json:"networkStorage,omitempty" xmlrpc:"networkStorage,omitempty"`

	// A count of an account's associated storage volumes. This includes Lockbox, NAS, EVault, and iSCSI volumes.
	NetworkStorageCount *uint `json:"networkStorageCount,omitempty" xmlrpc:"networkStorageCount,omitempty"`

	// A count of an account's Network Storage groups.
	NetworkStorageGroupCount *uint `json:"networkStorageGroupCount,omitempty" xmlrpc:"networkStorageGroupCount,omitempty"`

	// An account's Network Storage groups.
	NetworkStorageGroups []Network_Storage_Group `json:"networkStorageGroups,omitempty" xmlrpc:"networkStorageGroups,omitempty"`

	// A count of iPSec network tunnels for an account.
	NetworkTunnelContextCount *uint `json:"networkTunnelContextCount,omitempty" xmlrpc:"networkTunnelContextCount,omitempty"`

	// IPSec network tunnels for an account.
	NetworkTunnelContexts []Network_Tunnel_Module_Context `json:"networkTunnelContexts,omitempty" xmlrpc:"networkTunnelContexts,omitempty"`

	// A count of all network VLANs assigned to an account.
	NetworkVlanCount *uint `json:"networkVlanCount,omitempty" xmlrpc:"networkVlanCount,omitempty"`

	// Whether or not an account has automatic private VLAN spanning enabled.
	NetworkVlanSpan *Account_Network_Vlan_Span `json:"networkVlanSpan,omitempty" xmlrpc:"networkVlanSpan,omitempty"`

	// All network VLANs assigned to an account.
	NetworkVlans []Network_Vlan `json:"networkVlans,omitempty" xmlrpc:"networkVlans,omitempty"`

	// A count of dEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers for the next billing cycle. The public inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
	NextBillingPublicAllotmentHardwareBandwidthDetailCount *uint `json:"nextBillingPublicAllotmentHardwareBandwidthDetailCount,omitempty" xmlrpc:"nextBillingPublicAllotmentHardwareBandwidthDetailCount,omitempty"`

	// DEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers for the next billing cycle. The public inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
	NextBillingPublicAllotmentHardwareBandwidthDetails []Network_Bandwidth_Version1_Allotment `json:"nextBillingPublicAllotmentHardwareBandwidthDetails,omitempty" xmlrpc:"nextBillingPublicAllotmentHardwareBandwidthDetails,omitempty"`

	// The pre-tax total amount exempt from incubator credit for the account's next invoice. This field is now deprecated and will soon be removed. Please update all references to instead use nextInvoiceTotalAmount
	NextInvoiceIncubatorExemptTotal *Float64 `json:"nextInvoiceIncubatorExemptTotal,omitempty" xmlrpc:"nextInvoiceIncubatorExemptTotal,omitempty"`

	// A count of the billing items that will be on an account's next invoice.
	NextInvoiceTopLevelBillingItemCount *uint `json:"nextInvoiceTopLevelBillingItemCount,omitempty" xmlrpc:"nextInvoiceTopLevelBillingItemCount,omitempty"`

	// The billing items that will be on an account's next invoice.
	NextInvoiceTopLevelBillingItems []Billing_Item `json:"nextInvoiceTopLevelBillingItems,omitempty" xmlrpc:"nextInvoiceTopLevelBillingItems,omitempty"`

	// The pre-tax total amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
	NextInvoiceTotalAmount *Float64 `json:"nextInvoiceTotalAmount,omitempty" xmlrpc:"nextInvoiceTotalAmount,omitempty"`

	// The total one-time charge amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
	NextInvoiceTotalOneTimeAmount *Float64 `json:"nextInvoiceTotalOneTimeAmount,omitempty" xmlrpc:"nextInvoiceTotalOneTimeAmount,omitempty"`

	// The total one-time tax amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
	NextInvoiceTotalOneTimeTaxAmount *Float64 `json:"nextInvoiceTotalOneTimeTaxAmount,omitempty" xmlrpc:"nextInvoiceTotalOneTimeTaxAmount,omitempty"`

	// The total recurring charge amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
	NextInvoiceTotalRecurringAmount *Float64 `json:"nextInvoiceTotalRecurringAmount,omitempty" xmlrpc:"nextInvoiceTotalRecurringAmount,omitempty"`

	// The total recurring charge amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
	NextInvoiceTotalRecurringAmountBeforeAccountDiscount *Float64 `json:"nextInvoiceTotalRecurringAmountBeforeAccountDiscount,omitempty" xmlrpc:"nextInvoiceTotalRecurringAmountBeforeAccountDiscount,omitempty"`

	// The total recurring tax amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
	NextInvoiceTotalRecurringTaxAmount *Float64 `json:"nextInvoiceTotalRecurringTaxAmount,omitempty" xmlrpc:"nextInvoiceTotalRecurringTaxAmount,omitempty"`

	// The total recurring charge amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
	NextInvoiceTotalTaxableRecurringAmount *Float64 `json:"nextInvoiceTotalTaxableRecurringAmount,omitempty" xmlrpc:"nextInvoiceTotalTaxableRecurringAmount,omitempty"`

	// A count of
	NotificationSubscriberCount *uint `json:"notificationSubscriberCount,omitempty" xmlrpc:"notificationSubscriberCount,omitempty"`

	// no documentation yet
	NotificationSubscribers []Notification_Subscriber `json:"notificationSubscribers,omitempty" xmlrpc:"notificationSubscribers,omitempty"`

	// An office phone number assigned to an account.
	OfficePhone *string `json:"officePhone,omitempty" xmlrpc:"officePhone,omitempty"`

	// A count of the open abuse tickets associated with an account.
	OpenAbuseTicketCount *uint `json:"openAbuseTicketCount,omitempty" xmlrpc:"openAbuseTicketCount,omitempty"`

	// The open abuse tickets associated with an account.
	OpenAbuseTickets []Ticket `json:"openAbuseTickets,omitempty" xmlrpc:"openAbuseTickets,omitempty"`

	// A count of the open accounting tickets associated with an account.
	OpenAccountingTicketCount *uint `json:"openAccountingTicketCount,omitempty" xmlrpc:"openAccountingTicketCount,omitempty"`

	// The open accounting tickets associated with an account.
	OpenAccountingTickets []Ticket `json:"openAccountingTickets,omitempty" xmlrpc:"openAccountingTickets,omitempty"`

	// A count of the open billing tickets associated with an account.
	OpenBillingTicketCount *uint `json:"openBillingTicketCount,omitempty" xmlrpc:"openBillingTicketCount,omitempty"`

	// The open billing tickets associated with an account.
	OpenBillingTickets []Ticket `json:"openBillingTickets,omitempty" xmlrpc:"openBillingTickets,omitempty"`

	// A count of an open ticket requesting cancellation of this server, if one exists.
	OpenCancellationRequestCount *uint `json:"openCancellationRequestCount,omitempty" xmlrpc:"openCancellationRequestCount,omitempty"`

	// An open ticket requesting cancellation of this server, if one exists.
	OpenCancellationRequests []Billing_Item_Cancellation_Request `json:"openCancellationRequests,omitempty" xmlrpc:"openCancellationRequests,omitempty"`

	// A count of the open tickets that do not belong to the abuse, accounting, sales, or support groups associated with an account.
	OpenOtherTicketCount *uint `json:"openOtherTicketCount,omitempty" xmlrpc:"openOtherTicketCount,omitempty"`

	// The open tickets that do not belong to the abuse, accounting, sales, or support groups associated with an account.
	OpenOtherTickets []Ticket `json:"openOtherTickets,omitempty" xmlrpc:"openOtherTickets,omitempty"`

	// A count of an account's recurring invoices.
	OpenRecurringInvoiceCount *uint `json:"openRecurringInvoiceCount,omitempty" xmlrpc:"openRecurringInvoiceCount,omitempty"`

	// An account's recurring invoices.
	OpenRecurringInvoices []Billing_Invoice `json:"openRecurringInvoices,omitempty" xmlrpc:"openRecurringInvoices,omitempty"`

	// A count of the open sales tickets associated with an account.
	OpenSalesTicketCount *uint `json:"openSalesTicketCount,omitempty" xmlrpc:"openSalesTicketCount,omitempty"`

	// The open sales tickets associated with an account.
	OpenSalesTickets []Ticket `json:"openSalesTickets,omitempty" xmlrpc:"openSalesTickets,omitempty"`

	// A count of
	OpenStackAccountLinkCount *uint `json:"openStackAccountLinkCount,omitempty" xmlrpc:"openStackAccountLinkCount,omitempty"`

	// no documentation yet
	OpenStackAccountLinks []Account_Link `json:"openStackAccountLinks,omitempty" xmlrpc:"openStackAccountLinks,omitempty"`

	// An account's associated Openstack related Object Storage accounts.
	OpenStackObjectStorage []Network_Storage `json:"openStackObjectStorage,omitempty" xmlrpc:"openStackObjectStorage,omitempty"`

	// A count of an account's associated Openstack related Object Storage accounts.
	OpenStackObjectStorageCount *uint `json:"openStackObjectStorageCount,omitempty" xmlrpc:"openStackObjectStorageCount,omitempty"`

	// A count of the open support tickets associated with an account.
	OpenSupportTicketCount *uint `json:"openSupportTicketCount,omitempty" xmlrpc:"openSupportTicketCount,omitempty"`

	// The open support tickets associated with an account.
	OpenSupportTickets []Ticket `json:"openSupportTickets,omitempty" xmlrpc:"openSupportTickets,omitempty"`

	// A count of all open tickets associated with an account.
	OpenTicketCount *uint `json:"openTicketCount,omitempty" xmlrpc:"openTicketCount,omitempty"`

	// All open tickets associated with an account.
	OpenTickets []Ticket `json:"openTickets,omitempty" xmlrpc:"openTickets,omitempty"`

	// All open tickets associated with an account last edited by an employee.
	OpenTicketsWaitingOnCustomer []Ticket `json:"openTicketsWaitingOnCustomer,omitempty" xmlrpc:"openTicketsWaitingOnCustomer,omitempty"`

	// A count of all open tickets associated with an account last edited by an employee.
	OpenTicketsWaitingOnCustomerCount *uint `json:"openTicketsWaitingOnCustomerCount,omitempty" xmlrpc:"openTicketsWaitingOnCustomerCount,omitempty"`

	// A count of an account's associated billing orders excluding upgrades.
	OrderCount *uint `json:"orderCount,omitempty" xmlrpc:"orderCount,omitempty"`

	// An account's associated billing orders excluding upgrades.
	Orders []Billing_Order `json:"orders,omitempty" xmlrpc:"orders,omitempty"`

	// A count of the billing items that have no parent billing item. These are items that don't necessarily belong to a single server.
	OrphanBillingItemCount *uint `json:"orphanBillingItemCount,omitempty" xmlrpc:"orphanBillingItemCount,omitempty"`

	// The billing items that have no parent billing item. These are items that don't necessarily belong to a single server.
	OrphanBillingItems []Billing_Item `json:"orphanBillingItems,omitempty" xmlrpc:"orphanBillingItems,omitempty"`

	// A count of
	OwnedBrandCount *uint `json:"ownedBrandCount,omitempty" xmlrpc:"ownedBrandCount,omitempty"`

	// no documentation yet
	OwnedBrands []Brand `json:"ownedBrands,omitempty" xmlrpc:"ownedBrands,omitempty"`

	// A count of
	OwnedHardwareGenericComponentModelCount *uint `json:"ownedHardwareGenericComponentModelCount,omitempty" xmlrpc:"ownedHardwareGenericComponentModelCount,omitempty"`

	// no documentation yet
	OwnedHardwareGenericComponentModels []Hardware_Component_Model_Generic `json:"ownedHardwareGenericComponentModels,omitempty" xmlrpc:"ownedHardwareGenericComponentModels,omitempty"`

	// A count of
	PaymentProcessorCount *uint `json:"paymentProcessorCount,omitempty" xmlrpc:"paymentProcessorCount,omitempty"`

	// no documentation yet
	PaymentProcessors []Billing_Payment_Processor `json:"paymentProcessors,omitempty" xmlrpc:"paymentProcessors,omitempty"`

	// A count of
	PendingEventCount *uint `json:"pendingEventCount,omitempty" xmlrpc:"pendingEventCount,omitempty"`

	// no documentation yet
	PendingEvents []Notification_Occurrence_Event `json:"pendingEvents,omitempty" xmlrpc:"pendingEvents,omitempty"`

	// An account's latest open (pending) invoice.
	PendingInvoice *Billing_Invoice `json:"pendingInvoice,omitempty" xmlrpc:"pendingInvoice,omitempty"`

	// A count of a list of top-level invoice items that are on an account's currently pending invoice.
	PendingInvoiceTopLevelItemCount *uint `json:"pendingInvoiceTopLevelItemCount,omitempty" xmlrpc:"pendingInvoiceTopLevelItemCount,omitempty"`

	// A list of top-level invoice items that are on an account's currently pending invoice.
	PendingInvoiceTopLevelItems []Billing_Invoice_Item `json:"pendingInvoiceTopLevelItems,omitempty" xmlrpc:"pendingInvoiceTopLevelItems,omitempty"`

	// The total amount of an account's pending invoice, if one exists.
	PendingInvoiceTotalAmount *Float64 `json:"pendingInvoiceTotalAmount,omitempty" xmlrpc:"pendingInvoiceTotalAmount,omitempty"`

	// The total one-time charges for an account's pending invoice, if one exists. In other words, it is the sum of one-time charges, setup fees, and labor fees. It does not include taxes.
	PendingInvoiceTotalOneTimeAmount *Float64 `json:"pendingInvoiceTotalOneTimeAmount,omitempty" xmlrpc:"pendingInvoiceTotalOneTimeAmount,omitempty"`

	// The sum of all the taxes related to one time charges for an account's pending invoice, if one exists.
	PendingInvoiceTotalOneTimeTaxAmount *Float64 `json:"pendingInvoiceTotalOneTimeTaxAmount,omitempty" xmlrpc:"pendingInvoiceTotalOneTimeTaxAmount,omitempty"`

	// The total recurring amount of an account's pending invoice, if one exists.
	PendingInvoiceTotalRecurringAmount *Float64 `json:"pendingInvoiceTotalRecurringAmount,omitempty" xmlrpc:"pendingInvoiceTotalRecurringAmount,omitempty"`

	// The total amount of the recurring taxes on an account's pending invoice, if one exists.
	PendingInvoiceTotalRecurringTaxAmount *Float64 `json:"pendingInvoiceTotalRecurringTaxAmount,omitempty" xmlrpc:"pendingInvoiceTotalRecurringTaxAmount,omitempty"`

	// A count of an account's permission groups.
	PermissionGroupCount *uint `json:"permissionGroupCount,omitempty" xmlrpc:"permissionGroupCount,omitempty"`

	// An account's permission groups.
	PermissionGroups []User_Permission_Group `json:"permissionGroups,omitempty" xmlrpc:"permissionGroups,omitempty"`

	// A count of an account's user roles.
	PermissionRoleCount *uint `json:"permissionRoleCount,omitempty" xmlrpc:"permissionRoleCount,omitempty"`

	// An account's user roles.
	PermissionRoles []User_Permission_Role `json:"permissionRoles,omitempty" xmlrpc:"permissionRoles,omitempty"`

	// A count of
	PortableStorageVolumeCount *uint `json:"portableStorageVolumeCount,omitempty" xmlrpc:"portableStorageVolumeCount,omitempty"`

	// no documentation yet
	PortableStorageVolumes []Virtual_Disk_Image `json:"portableStorageVolumes,omitempty" xmlrpc:"portableStorageVolumes,omitempty"`

	// A count of customer specified URIs that are downloaded onto a newly provisioned or reloaded server. If the URI is sent over https it will be executed directly on the server.
	PostProvisioningHookCount *uint `json:"postProvisioningHookCount,omitempty" xmlrpc:"postProvisioningHookCount,omitempty"`

	// Customer specified URIs that are downloaded onto a newly provisioned or reloaded server. If the URI is sent over https it will be executed directly on the server.
	PostProvisioningHooks []Provisioning_Hook `json:"postProvisioningHooks,omitempty" xmlrpc:"postProvisioningHooks,omitempty"`

	// The postal code of the mailing address belonging to an account.
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// A count of an account's associated portal users with PPTP VPN access.
	PptpVpnUserCount *uint `json:"pptpVpnUserCount,omitempty" xmlrpc:"pptpVpnUserCount,omitempty"`

	// An account's associated portal users with PPTP VPN access.
	PptpVpnUsers []User_Customer `json:"pptpVpnUsers,omitempty" xmlrpc:"pptpVpnUsers,omitempty"`

	// The total recurring amount for an accounts previous revenue.
	PreviousRecurringRevenue *Float64 `json:"previousRecurringRevenue,omitempty" xmlrpc:"previousRecurringRevenue,omitempty"`

	// A count of the item price that an account is restricted to.
	PriceRestrictionCount *uint `json:"priceRestrictionCount,omitempty" xmlrpc:"priceRestrictionCount,omitempty"`

	// The item price that an account is restricted to.
	PriceRestrictions []Product_Item_Price_Account_Restriction `json:"priceRestrictions,omitempty" xmlrpc:"priceRestrictions,omitempty"`

	// A count of all priority one tickets associated with an account.
	PriorityOneTicketCount *uint `json:"priorityOneTicketCount,omitempty" xmlrpc:"priorityOneTicketCount,omitempty"`

	// All priority one tickets associated with an account.
	PriorityOneTickets []Ticket `json:"priorityOneTickets,omitempty" xmlrpc:"priorityOneTickets,omitempty"`

	// A count of dEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers. The private inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
	PrivateAllotmentHardwareBandwidthDetailCount *uint `json:"privateAllotmentHardwareBandwidthDetailCount,omitempty" xmlrpc:"privateAllotmentHardwareBandwidthDetailCount,omitempty"`

	// DEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers. The private inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
	PrivateAllotmentHardwareBandwidthDetails []Network_Bandwidth_Version1_Allotment `json:"privateAllotmentHardwareBandwidthDetails,omitempty" xmlrpc:"privateAllotmentHardwareBandwidthDetails,omitempty"`

	// A count of private and shared template group objects (parent only) for an account.
	PrivateBlockDeviceTemplateGroupCount *uint `json:"privateBlockDeviceTemplateGroupCount,omitempty" xmlrpc:"privateBlockDeviceTemplateGroupCount,omitempty"`

	// Private and shared template group objects (parent only) for an account.
	PrivateBlockDeviceTemplateGroups []Virtual_Guest_Block_Device_Template_Group `json:"privateBlockDeviceTemplateGroups,omitempty" xmlrpc:"privateBlockDeviceTemplateGroups,omitempty"`

	// A count of
	PrivateIpAddressCount *uint `json:"privateIpAddressCount,omitempty" xmlrpc:"privateIpAddressCount,omitempty"`

	// no documentation yet
	PrivateIpAddresses []Network_Subnet_IpAddress `json:"privateIpAddresses,omitempty" xmlrpc:"privateIpAddresses,omitempty"`

	// A count of the private network VLANs assigned to an account.
	PrivateNetworkVlanCount *uint `json:"privateNetworkVlanCount,omitempty" xmlrpc:"privateNetworkVlanCount,omitempty"`

	// The private network VLANs assigned to an account.
	PrivateNetworkVlans []Network_Vlan `json:"privateNetworkVlans,omitempty" xmlrpc:"privateNetworkVlans,omitempty"`

	// A count of all private subnets associated with an account.
	PrivateSubnetCount *uint `json:"privateSubnetCount,omitempty" xmlrpc:"privateSubnetCount,omitempty"`

	// All private subnets associated with an account.
	PrivateSubnets []Network_Subnet `json:"privateSubnets,omitempty" xmlrpc:"privateSubnets,omitempty"`

	// A count of dEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers. The public inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
	PublicAllotmentHardwareBandwidthDetailCount *uint `json:"publicAllotmentHardwareBandwidthDetailCount,omitempty" xmlrpc:"publicAllotmentHardwareBandwidthDetailCount,omitempty"`

	// DEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers. The public inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
	PublicAllotmentHardwareBandwidthDetails []Network_Bandwidth_Version1_Allotment `json:"publicAllotmentHardwareBandwidthDetails,omitempty" xmlrpc:"publicAllotmentHardwareBandwidthDetails,omitempty"`

	// A count of
	PublicIpAddressCount *uint `json:"publicIpAddressCount,omitempty" xmlrpc:"publicIpAddressCount,omitempty"`

	// no documentation yet
	PublicIpAddresses []Network_Subnet_IpAddress `json:"publicIpAddresses,omitempty" xmlrpc:"publicIpAddresses,omitempty"`

	// A count of the public network VLANs assigned to an account.
	PublicNetworkVlanCount *uint `json:"publicNetworkVlanCount,omitempty" xmlrpc:"publicNetworkVlanCount,omitempty"`

	// The public network VLANs assigned to an account.
	PublicNetworkVlans []Network_Vlan `json:"publicNetworkVlans,omitempty" xmlrpc:"publicNetworkVlans,omitempty"`

	// A count of all public network subnets associated with an account.
	PublicSubnetCount *uint `json:"publicSubnetCount,omitempty" xmlrpc:"publicSubnetCount,omitempty"`

	// All public network subnets associated with an account.
	PublicSubnets []Network_Subnet `json:"publicSubnets,omitempty" xmlrpc:"publicSubnets,omitempty"`

	// A count of an account's quotes.
	QuoteCount *uint `json:"quoteCount,omitempty" xmlrpc:"quoteCount,omitempty"`

	// An account's quotes.
	Quotes []Billing_Order_Quote `json:"quotes,omitempty" xmlrpc:"quotes,omitempty"`

	// A count of
	RecentEventCount *uint `json:"recentEventCount,omitempty" xmlrpc:"recentEventCount,omitempty"`

	// no documentation yet
	RecentEvents []Notification_Occurrence_Event `json:"recentEvents,omitempty" xmlrpc:"recentEvents,omitempty"`

	// The Referral Partner for this account, if any.
	ReferralPartner *Account `json:"referralPartner,omitempty" xmlrpc:"referralPartner,omitempty"`

	// A count of if this is a account is a referral partner, the accounts this referral partner has referred
	ReferredAccountCount *uint `json:"referredAccountCount,omitempty" xmlrpc:"referredAccountCount,omitempty"`

	// If this is a account is a referral partner, the accounts this referral partner has referred
	ReferredAccounts []Account `json:"referredAccounts,omitempty" xmlrpc:"referredAccounts,omitempty"`

	// A count of
	RegulatedWorkloadCount *uint `json:"regulatedWorkloadCount,omitempty" xmlrpc:"regulatedWorkloadCount,omitempty"`

	// no documentation yet
	RegulatedWorkloads []Legal_RegulatedWorkload `json:"regulatedWorkloads,omitempty" xmlrpc:"regulatedWorkloads,omitempty"`

	// A count of remote management command requests for an account
	RemoteManagementCommandRequestCount *uint `json:"remoteManagementCommandRequestCount,omitempty" xmlrpc:"remoteManagementCommandRequestCount,omitempty"`

	// Remote management command requests for an account
	RemoteManagementCommandRequests []Hardware_Component_RemoteManagement_Command_Request `json:"remoteManagementCommandRequests,omitempty" xmlrpc:"remoteManagementCommandRequests,omitempty"`

	// A count of the Replication events for all Network Storage volumes on an account.
	ReplicationEventCount *uint `json:"replicationEventCount,omitempty" xmlrpc:"replicationEventCount,omitempty"`

	// The Replication events for all Network Storage volumes on an account.
	ReplicationEvents []Network_Storage_Event `json:"replicationEvents,omitempty" xmlrpc:"replicationEvents,omitempty"`

	// A count of an account's associated top-level resource groups.
	ResourceGroupCount *uint `json:"resourceGroupCount,omitempty" xmlrpc:"resourceGroupCount,omitempty"`

	// An account's associated top-level resource groups.
	ResourceGroups []Resource_Group `json:"resourceGroups,omitempty" xmlrpc:"resourceGroups,omitempty"`

	// A count of all Routers that an accounts VLANs reside on
	RouterCount *uint `json:"routerCount,omitempty" xmlrpc:"routerCount,omitempty"`

	// All Routers that an accounts VLANs reside on
	Routers []Hardware `json:"routers,omitempty" xmlrpc:"routers,omitempty"`

	// An account's reverse WHOIS data. This data is used when making SWIP requests.
	RwhoisData *Network_Subnet_Rwhois_Data `json:"rwhoisData,omitempty" xmlrpc:"rwhoisData,omitempty"`

	// no documentation yet
	SalesforceAccountLink *Account_Link `json:"salesforceAccountLink,omitempty" xmlrpc:"salesforceAccountLink,omitempty"`

	// The SAML configuration for this account.
	SamlAuthentication *Account_Authentication_Saml `json:"samlAuthentication,omitempty" xmlrpc:"samlAuthentication,omitempty"`

	// A count of all scale groups on this account.
	ScaleGroupCount *uint `json:"scaleGroupCount,omitempty" xmlrpc:"scaleGroupCount,omitempty"`

	// All scale groups on this account.
	ScaleGroups []Scale_Group `json:"scaleGroups,omitempty" xmlrpc:"scaleGroups,omitempty"`

	// A count of the secondary DNS records for a SoftLayer customer account.
	SecondaryDomainCount *uint `json:"secondaryDomainCount,omitempty" xmlrpc:"secondaryDomainCount,omitempty"`

	// The secondary DNS records for a SoftLayer customer account.
	SecondaryDomains []Dns_Secondary `json:"secondaryDomains,omitempty" xmlrpc:"secondaryDomains,omitempty"`

	// A count of stored security certificates (ie. SSL)
	SecurityCertificateCount *uint `json:"securityCertificateCount,omitempty" xmlrpc:"securityCertificateCount,omitempty"`

	// Stored security certificates (ie. SSL)
	SecurityCertificates []Security_Certificate `json:"securityCertificates,omitempty" xmlrpc:"securityCertificates,omitempty"`

	// A count of an account's vulnerability scan requests.
	SecurityScanRequestCount *uint `json:"securityScanRequestCount,omitempty" xmlrpc:"securityScanRequestCount,omitempty"`

	// An account's vulnerability scan requests.
	SecurityScanRequests []Network_Security_Scanner_Request `json:"securityScanRequests,omitempty" xmlrpc:"securityScanRequests,omitempty"`

	// A count of the service billing items that will be on an account's next invoice.
	ServiceBillingItemCount *uint `json:"serviceBillingItemCount,omitempty" xmlrpc:"serviceBillingItemCount,omitempty"`

	// The service billing items that will be on an account's next invoice.
	ServiceBillingItems []Billing_Item `json:"serviceBillingItems,omitempty" xmlrpc:"serviceBillingItems,omitempty"`

	// A count of shipments that belong to the customer's account.
	ShipmentCount *uint `json:"shipmentCount,omitempty" xmlrpc:"shipmentCount,omitempty"`

	// Shipments that belong to the customer's account.
	Shipments []Account_Shipment `json:"shipments,omitempty" xmlrpc:"shipments,omitempty"`

	// A count of customer specified SSH keys that can be implemented onto a newly provisioned or reloaded server.
	SshKeyCount *uint `json:"sshKeyCount,omitempty" xmlrpc:"sshKeyCount,omitempty"`

	// Customer specified SSH keys that can be implemented onto a newly provisioned or reloaded server.
	SshKeys []Security_Ssh_Key `json:"sshKeys,omitempty" xmlrpc:"sshKeys,omitempty"`

	// A count of an account's associated portal users with SSL VPN access.
	SslVpnUserCount *uint `json:"sslVpnUserCount,omitempty" xmlrpc:"sslVpnUserCount,omitempty"`

	// An account's associated portal users with SSL VPN access.
	SslVpnUsers []User_Customer `json:"sslVpnUsers,omitempty" xmlrpc:"sslVpnUsers,omitempty"`

	// A count of an account's virtual guest objects that are hosted on a user provisioned hypervisor.
	StandardPoolVirtualGuestCount *uint `json:"standardPoolVirtualGuestCount,omitempty" xmlrpc:"standardPoolVirtualGuestCount,omitempty"`

	// An account's virtual guest objects that are hosted on a user provisioned hypervisor.
	StandardPoolVirtualGuests []Virtual_Guest `json:"standardPoolVirtualGuests,omitempty" xmlrpc:"standardPoolVirtualGuests,omitempty"`

	// A two-letter abbreviation of the state in the mailing address belonging to an account. If an account does not reside in a province then this is typically blank.
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// The date of an account's last status change.
	StatusDate *Time `json:"statusDate,omitempty" xmlrpc:"statusDate,omitempty"`

	// A count of all network subnets associated with an account.
	SubnetCount *uint `json:"subnetCount,omitempty" xmlrpc:"subnetCount,omitempty"`

	// A count of
	SubnetRegistrationCount *uint `json:"subnetRegistrationCount,omitempty" xmlrpc:"subnetRegistrationCount,omitempty"`

	// A count of
	SubnetRegistrationDetailCount *uint `json:"subnetRegistrationDetailCount,omitempty" xmlrpc:"subnetRegistrationDetailCount,omitempty"`

	// no documentation yet
	SubnetRegistrationDetails []Account_Regional_Registry_Detail `json:"subnetRegistrationDetails,omitempty" xmlrpc:"subnetRegistrationDetails,omitempty"`

	// no documentation yet
	SubnetRegistrations []Network_Subnet_Registration `json:"subnetRegistrations,omitempty" xmlrpc:"subnetRegistrations,omitempty"`

	// All network subnets associated with an account.
	Subnets []Network_Subnet `json:"subnets,omitempty" xmlrpc:"subnets,omitempty"`

	// A count of the SoftLayer employees that an account is assigned to.
	SupportRepresentativeCount *uint `json:"supportRepresentativeCount,omitempty" xmlrpc:"supportRepresentativeCount,omitempty"`

	// The SoftLayer employees that an account is assigned to.
	SupportRepresentatives []User_Employee `json:"supportRepresentatives,omitempty" xmlrpc:"supportRepresentatives,omitempty"`

	// A count of the active support subscriptions for this account.
	SupportSubscriptionCount *uint `json:"supportSubscriptionCount,omitempty" xmlrpc:"supportSubscriptionCount,omitempty"`

	// The active support subscriptions for this account.
	SupportSubscriptions []Billing_Item `json:"supportSubscriptions,omitempty" xmlrpc:"supportSubscriptions,omitempty"`

	// no documentation yet
	SupportTier *string `json:"supportTier,omitempty" xmlrpc:"supportTier,omitempty"`

	// A flag indicating to suppress invoices.
	SuppressInvoicesFlag *bool `json:"suppressInvoicesFlag,omitempty" xmlrpc:"suppressInvoicesFlag,omitempty"`

	// A count of
	TagCount *uint `json:"tagCount,omitempty" xmlrpc:"tagCount,omitempty"`

	// no documentation yet
	Tags []Tag `json:"tags,omitempty" xmlrpc:"tags,omitempty"`

	// A count of an account's associated tickets.
	TicketCount *uint `json:"ticketCount,omitempty" xmlrpc:"ticketCount,omitempty"`

	// An account's associated tickets.
	Tickets []Ticket `json:"tickets,omitempty" xmlrpc:"tickets,omitempty"`

	// Tickets closed within the last 72 hours or last 10 tickets, whichever is less, associated with an account.
	TicketsClosedInTheLastThreeDays []Ticket `json:"ticketsClosedInTheLastThreeDays,omitempty" xmlrpc:"ticketsClosedInTheLastThreeDays,omitempty"`

	// A count of tickets closed within the last 72 hours or last 10 tickets, whichever is less, associated with an account.
	TicketsClosedInTheLastThreeDaysCount *uint `json:"ticketsClosedInTheLastThreeDaysCount,omitempty" xmlrpc:"ticketsClosedInTheLastThreeDaysCount,omitempty"`

	// Tickets closed today associated with an account.
	TicketsClosedToday []Ticket `json:"ticketsClosedToday,omitempty" xmlrpc:"ticketsClosedToday,omitempty"`

	// A count of tickets closed today associated with an account.
	TicketsClosedTodayCount *uint `json:"ticketsClosedTodayCount,omitempty" xmlrpc:"ticketsClosedTodayCount,omitempty"`

	// A count of an account's associated Transcode account.
	TranscodeAccountCount *uint `json:"transcodeAccountCount,omitempty" xmlrpc:"transcodeAccountCount,omitempty"`

	// An account's associated Transcode account.
	TranscodeAccounts []Network_Media_Transcode_Account `json:"transcodeAccounts,omitempty" xmlrpc:"transcodeAccounts,omitempty"`

	// A count of an account's associated upgrade requests.
	UpgradeRequestCount *uint `json:"upgradeRequestCount,omitempty" xmlrpc:"upgradeRequestCount,omitempty"`

	// An account's associated upgrade requests.
	UpgradeRequests []Product_Upgrade_Request `json:"upgradeRequests,omitempty" xmlrpc:"upgradeRequests,omitempty"`

	// A count of an account's portal users.
	UserCount *uint `json:"userCount,omitempty" xmlrpc:"userCount,omitempty"`

	// An account's portal users.
	Users []User_Customer `json:"users,omitempty" xmlrpc:"users,omitempty"`

	// A count of stored security certificates that are not expired (ie. SSL)
	ValidSecurityCertificateCount *uint `json:"validSecurityCertificateCount,omitempty" xmlrpc:"validSecurityCertificateCount,omitempty"`

	// Stored security certificates that are not expired (ie. SSL)
	ValidSecurityCertificates []Security_Certificate `json:"validSecurityCertificates,omitempty" xmlrpc:"validSecurityCertificates,omitempty"`

	// Return 0 if vpn updates are currently in progress on this account otherwise 1.
	VdrUpdatesInProgressFlag *bool `json:"vdrUpdatesInProgressFlag,omitempty" xmlrpc:"vdrUpdatesInProgressFlag,omitempty"`

	// A count of the bandwidth pooling for this account.
	VirtualDedicatedRackCount *uint `json:"virtualDedicatedRackCount,omitempty" xmlrpc:"virtualDedicatedRackCount,omitempty"`

	// The bandwidth pooling for this account.
	VirtualDedicatedRacks []Network_Bandwidth_Version1_Allotment `json:"virtualDedicatedRacks,omitempty" xmlrpc:"virtualDedicatedRacks,omitempty"`

	// A count of an account's associated virtual server virtual disk images.
	VirtualDiskImageCount *uint `json:"virtualDiskImageCount,omitempty" xmlrpc:"virtualDiskImageCount,omitempty"`

	// An account's associated virtual server virtual disk images.
	VirtualDiskImages []Virtual_Disk_Image `json:"virtualDiskImages,omitempty" xmlrpc:"virtualDiskImages,omitempty"`

	// A count of an account's associated virtual guest objects.
	VirtualGuestCount *uint `json:"virtualGuestCount,omitempty" xmlrpc:"virtualGuestCount,omitempty"`

	// An account's associated virtual guest objects.
	VirtualGuests []Virtual_Guest `json:"virtualGuests,omitempty" xmlrpc:"virtualGuests,omitempty"`

	// An account's associated virtual guest objects currently over bandwidth allocation.
	VirtualGuestsOverBandwidthAllocation []Virtual_Guest `json:"virtualGuestsOverBandwidthAllocation,omitempty" xmlrpc:"virtualGuestsOverBandwidthAllocation,omitempty"`

	// A count of an account's associated virtual guest objects currently over bandwidth allocation.
	VirtualGuestsOverBandwidthAllocationCount *uint `json:"virtualGuestsOverBandwidthAllocationCount,omitempty" xmlrpc:"virtualGuestsOverBandwidthAllocationCount,omitempty"`

	// An account's associated virtual guest objects currently over bandwidth allocation.
	VirtualGuestsProjectedOverBandwidthAllocation []Virtual_Guest `json:"virtualGuestsProjectedOverBandwidthAllocation,omitempty" xmlrpc:"virtualGuestsProjectedOverBandwidthAllocation,omitempty"`

	// A count of an account's associated virtual guest objects currently over bandwidth allocation.
	VirtualGuestsProjectedOverBandwidthAllocationCount *uint `json:"virtualGuestsProjectedOverBandwidthAllocationCount,omitempty" xmlrpc:"virtualGuestsProjectedOverBandwidthAllocationCount,omitempty"`

	// All virtual guests associated with an account that has the cPanel web hosting control panel installed.
	VirtualGuestsWithCpanel []Virtual_Guest `json:"virtualGuestsWithCpanel,omitempty" xmlrpc:"virtualGuestsWithCpanel,omitempty"`

	// A count of all virtual guests associated with an account that has the cPanel web hosting control panel installed.
	VirtualGuestsWithCpanelCount *uint `json:"virtualGuestsWithCpanelCount,omitempty" xmlrpc:"virtualGuestsWithCpanelCount,omitempty"`

	// All virtual guests associated with an account that have McAfee Secure software components.
	VirtualGuestsWithMcafee []Virtual_Guest `json:"virtualGuestsWithMcafee,omitempty" xmlrpc:"virtualGuestsWithMcafee,omitempty"`

	// All virtual guests associated with an account that have McAfee Secure AntiVirus for Redhat software components.
	VirtualGuestsWithMcafeeAntivirusRedhat []Virtual_Guest `json:"virtualGuestsWithMcafeeAntivirusRedhat,omitempty" xmlrpc:"virtualGuestsWithMcafeeAntivirusRedhat,omitempty"`

	// A count of all virtual guests associated with an account that have McAfee Secure AntiVirus for Redhat software components.
	VirtualGuestsWithMcafeeAntivirusRedhatCount *uint `json:"virtualGuestsWithMcafeeAntivirusRedhatCount,omitempty" xmlrpc:"virtualGuestsWithMcafeeAntivirusRedhatCount,omitempty"`

	// A count of all virtual guests associated with an account that has McAfee Secure AntiVirus for Windows software components.
	VirtualGuestsWithMcafeeAntivirusWindowCount *uint `json:"virtualGuestsWithMcafeeAntivirusWindowCount,omitempty" xmlrpc:"virtualGuestsWithMcafeeAntivirusWindowCount,omitempty"`

	// All virtual guests associated with an account that has McAfee Secure AntiVirus for Windows software components.
	VirtualGuestsWithMcafeeAntivirusWindows []Virtual_Guest `json:"virtualGuestsWithMcafeeAntivirusWindows,omitempty" xmlrpc:"virtualGuestsWithMcafeeAntivirusWindows,omitempty"`

	// A count of all virtual guests associated with an account that have McAfee Secure software components.
	VirtualGuestsWithMcafeeCount *uint `json:"virtualGuestsWithMcafeeCount,omitempty" xmlrpc:"virtualGuestsWithMcafeeCount,omitempty"`

	// All virtual guests associated with an account that has McAfee Secure Intrusion Detection System software components.
	VirtualGuestsWithMcafeeIntrusionDetectionSystem []Virtual_Guest `json:"virtualGuestsWithMcafeeIntrusionDetectionSystem,omitempty" xmlrpc:"virtualGuestsWithMcafeeIntrusionDetectionSystem,omitempty"`

	// A count of all virtual guests associated with an account that has McAfee Secure Intrusion Detection System software components.
	VirtualGuestsWithMcafeeIntrusionDetectionSystemCount *uint `json:"virtualGuestsWithMcafeeIntrusionDetectionSystemCount,omitempty" xmlrpc:"virtualGuestsWithMcafeeIntrusionDetectionSystemCount,omitempty"`

	// All virtual guests associated with an account that has the Plesk web hosting control panel installed.
	VirtualGuestsWithPlesk []Virtual_Guest `json:"virtualGuestsWithPlesk,omitempty" xmlrpc:"virtualGuestsWithPlesk,omitempty"`

	// A count of all virtual guests associated with an account that has the Plesk web hosting control panel installed.
	VirtualGuestsWithPleskCount *uint `json:"virtualGuestsWithPleskCount,omitempty" xmlrpc:"virtualGuestsWithPleskCount,omitempty"`

	// All virtual guests associated with an account that have the QuantaStor storage system installed.
	VirtualGuestsWithQuantastor []Virtual_Guest `json:"virtualGuestsWithQuantastor,omitempty" xmlrpc:"virtualGuestsWithQuantastor,omitempty"`

	// A count of all virtual guests associated with an account that have the QuantaStor storage system installed.
	VirtualGuestsWithQuantastorCount *uint `json:"virtualGuestsWithQuantastorCount,omitempty" xmlrpc:"virtualGuestsWithQuantastorCount,omitempty"`

	// All virtual guests associated with an account that has the Urchin web traffic analytics package installed.
	VirtualGuestsWithUrchin []Virtual_Guest `json:"virtualGuestsWithUrchin,omitempty" xmlrpc:"virtualGuestsWithUrchin,omitempty"`

	// A count of all virtual guests associated with an account that has the Urchin web traffic analytics package installed.
	VirtualGuestsWithUrchinCount *uint `json:"virtualGuestsWithUrchinCount,omitempty" xmlrpc:"virtualGuestsWithUrchinCount,omitempty"`

	// The bandwidth pooling for this account.
	VirtualPrivateRack *Network_Bandwidth_Version1_Allotment `json:"virtualPrivateRack,omitempty" xmlrpc:"virtualPrivateRack,omitempty"`

	// An account's associated virtual server archived storage repositories.
	VirtualStorageArchiveRepositories []Virtual_Storage_Repository `json:"virtualStorageArchiveRepositories,omitempty" xmlrpc:"virtualStorageArchiveRepositories,omitempty"`

	// A count of an account's associated virtual server archived storage repositories.
	VirtualStorageArchiveRepositoryCount *uint `json:"virtualStorageArchiveRepositoryCount,omitempty" xmlrpc:"virtualStorageArchiveRepositoryCount,omitempty"`

	// An account's associated virtual server public storage repositories.
	VirtualStoragePublicRepositories []Virtual_Storage_Repository `json:"virtualStoragePublicRepositories,omitempty" xmlrpc:"virtualStoragePublicRepositories,omitempty"`

	// A count of an account's associated virtual server public storage repositories.
	VirtualStoragePublicRepositoryCount *uint `json:"virtualStoragePublicRepositoryCount,omitempty" xmlrpc:"virtualStoragePublicRepositoryCount,omitempty"`
}

// An unfortunate facet of the hosting business is the necessity of with legal and network abuse inquiries. As these types of inquiries frequently contain sensitive information SoftLayer keeps a separate account contact email address for direct contact about legal and abuse matters, modeled by the SoftLayer_Account_AbuseEmail data type. SoftLayer will typically email an account's abuse email addresses in these types of cases, and an email is automatically sent to an account's abuse email addresses when a legal or abuse ticket is created or updated.
type Account_AbuseEmail struct {
	Entity

	// The account associated with an abuse email address.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A valid email address.
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`
}

// The SoftLayer_Account_Address data type contains information on an address associated with a SoftLayer account.
type Account_Address struct {
	Entity

	// The account to which this address belongs.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// Line 1 of the address (normally the street address).
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// Line 2 of the address.
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// The city of the address.
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// The contact name (person, office) of the address.
	ContactName *string `json:"contactName,omitempty" xmlrpc:"contactName,omitempty"`

	// The country of the address.
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// The customer user who created this address.
	CreateUser *User_Customer `json:"createUser,omitempty" xmlrpc:"createUser,omitempty"`

	// The description of the address.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique id of the address.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Flag to show whether the address is active.
	IsActive *int `json:"isActive,omitempty" xmlrpc:"isActive,omitempty"`

	// The location of this address.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// The location id of the address.
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// The employee who last modified this address.
	ModifyEmployee *User_Employee `json:"modifyEmployee,omitempty" xmlrpc:"modifyEmployee,omitempty"`

	// The customer user who last modified this address.
	ModifyUser *User_Customer `json:"modifyUser,omitempty" xmlrpc:"modifyUser,omitempty"`

	// The postal (zip) code of the address.
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// The state of the address.
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// An account address' type.
	Type *Account_Address_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// no documentation yet
type Account_Address_Type struct {
	Entity

	// DEPRECATED
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// This service allows for a unique identifier to be associated to an existing customer account.
type Account_Affiliation struct {
	Entity

	// The account that an affiliation belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A customer account's internal identifier.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// An affiliate identifier associated with the customer account.
	AffiliateId *string `json:"affiliateId,omitempty" xmlrpc:"affiliateId,omitempty"`

	// The date an account affiliation was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A customer affiliation internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date an account affiliation was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`
}

// no documentation yet
type Account_Agreement struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The type of agreement.
	AgreementType *Account_Agreement_Type `json:"agreementType,omitempty" xmlrpc:"agreementType,omitempty"`

	// The type of agreement identifier.
	AgreementTypeId *int `json:"agreementTypeId,omitempty" xmlrpc:"agreementTypeId,omitempty"`

	// A count of the files attached to an agreement.
	AttachedBillingAgreementFileCount *uint `json:"attachedBillingAgreementFileCount,omitempty" xmlrpc:"attachedBillingAgreementFileCount,omitempty"`

	// The files attached to an agreement.
	AttachedBillingAgreementFiles []Account_MasterServiceAgreement `json:"attachedBillingAgreementFiles,omitempty" xmlrpc:"attachedBillingAgreementFiles,omitempty"`

	// no documentation yet
	AutoRenew *int `json:"autoRenew,omitempty" xmlrpc:"autoRenew,omitempty"`

	// A count of the billing items associated with an agreement.
	BillingItemCount *uint `json:"billingItemCount,omitempty" xmlrpc:"billingItemCount,omitempty"`

	// The billing items associated with an agreement.
	BillingItems []Billing_Item `json:"billingItems,omitempty" xmlrpc:"billingItems,omitempty"`

	// no documentation yet
	CancellationFee *int `json:"cancellationFee,omitempty" xmlrpc:"cancellationFee,omitempty"`

	// The date an agreement was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The duration in months of an agreement.
	DurationMonths *int `json:"durationMonths,omitempty" xmlrpc:"durationMonths,omitempty"`

	// The end date of an agreement.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// An agreement's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The effective start date of an agreement.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// The status of the agreement.
	Status *Account_Agreement_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The status identifier for an agreement.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// The title of an agreement.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`

	// A count of the top level billing item associated with an agreement.
	TopLevelBillingItemCount *uint `json:"topLevelBillingItemCount,omitempty" xmlrpc:"topLevelBillingItemCount,omitempty"`

	// The top level billing item associated with an agreement.
	TopLevelBillingItems []Billing_Item `json:"topLevelBillingItems,omitempty" xmlrpc:"topLevelBillingItems,omitempty"`
}

// no documentation yet
type Account_Agreement_Status struct {
	Entity

	// The name of the agreement status.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Account_Agreement_Type struct {
	Entity

	// The name of the agreement type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// A SoftLayer_Account_Attachment_Employee models an assignment of a single [[SoftLayer_User_Employee|employee]] with a single [[SoftLayer_Account|account]]
type Account_Attachment_Employee struct {
	Entity

	// A [[SoftLayer_Account|account]] that is assigned to a [[SoftLayer_User_Employee|employee]].
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A [[SoftLayer_User_Employee|employee]] that is assigned to a [[SoftLayer_Account|account]].
	Employee *User_Employee `json:"employee,omitempty" xmlrpc:"employee,omitempty"`

	// A [[SoftLayer_User_Employee|employee]] that is assigned to a [[SoftLayer_Account|account]].
	EmployeeRole *Account_Attachment_Employee_Role `json:"employeeRole,omitempty" xmlrpc:"employeeRole,omitempty"`

	// Role identifier.
	RoleId *int `json:"roleId,omitempty" xmlrpc:"roleId,omitempty"`
}

// no documentation yet
type Account_Attachment_Employee_Role struct {
	Entity

	// no documentation yet
	Keyname *string `json:"keyname,omitempty" xmlrpc:"keyname,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Many SoftLayer customer accounts have individual attributes assigned to them that describe features or special features for that account, such as special pricing, account statuses, and ordering instructions. The SoftLayer_Account_Attribute data type contains information relating to a single SoftLayer_Account attribute.
type Account_Attribute struct {
	Entity

	// The SoftLayer customer account that has an attribute.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The type of attribute assigned to a SoftLayer customer account.
	AccountAttributeType *Account_Attribute_Type `json:"accountAttributeType,omitempty" xmlrpc:"accountAttributeType,omitempty"`

	// The internal identifier of the type of attribute that a SoftLayer customer account attribute belongs to.
	AccountAttributeTypeId *int `json:"accountAttributeTypeId,omitempty" xmlrpc:"accountAttributeTypeId,omitempty"`

	// The internal identifier of the SoftLayer customer account that is assigned an account attribute.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A SoftLayer customer account attribute's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A SoftLayer account attribute's value.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// SoftLayer_Account_Attribute_Type models the type of attribute that can be assigned to a SoftLayer customer account.
type Account_Attribute_Type struct {
	Entity

	// A brief description of a SoftLayer account attribute type.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A SoftLayer account attribute type's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A SoftLayer account attribute type's key name. This is typically a shorter version of an attribute type's name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A SoftLayer account attribute type's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Account authentication has many different settings that can be set. This class allows the customer or employee to set these settigns.
type Account_Authentication_Attribute struct {
	Entity

	// The SoftLayer customer account.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The internal identifier of the SoftLayer customer account that is assigned an account authenction attribute.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The SoftLayer account authentication that has an attribute.
	AuthenticationRecord *Account_Authentication_Saml `json:"authenticationRecord,omitempty" xmlrpc:"authenticationRecord,omitempty"`

	// A SoftLayer account authenction attribute's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The type of attribute assigned to a SoftLayer account authentication.
	Type *Account_Authentication_Attribute_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The internal identifier of the type of attribute that a SoftLayer account authenction attribute belongs to.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// A SoftLayer account authenction attribute's value.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// SoftLayer_Account_Authentication_Attribute_Type models the type of attribute that can be assigned to a SoftLayer customer account authentication.
type Account_Authentication_Attribute_Type struct {
	Entity

	// A brief description of a SoftLayer account authentication attribute type.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A SoftLayer account authentication attribute type's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A SoftLayer account authentication attribute type's key name. This is typically a shorter version of an attribute type's name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A SoftLayer account authentication attribute type's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// An example of what you can put in as your value.
	ValueExample *string `json:"valueExample,omitempty" xmlrpc:"valueExample,omitempty"`
}

// no documentation yet
type Account_Authentication_OpenIdConnect_Option struct {
	Entity

	// no documentation yet
	Key *string `json:"key,omitempty" xmlrpc:"key,omitempty"`

	// no documentation yet
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Account_Authentication_OpenIdConnect_RegistrationInformation struct {
	Entity

	// no documentation yet
	ExistingBlueIdFlag *bool `json:"existingBlueIdFlag,omitempty" xmlrpc:"existingBlueIdFlag,omitempty"`

	// no documentation yet
	FederatedEmailDomainFlag *bool `json:"federatedEmailDomainFlag,omitempty" xmlrpc:"federatedEmailDomainFlag,omitempty"`

	// no documentation yet
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`
}

// no documentation yet
type Account_Authentication_Saml struct {
	Entity

	// The account associated with this saml configuration.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The saml account id.
	AccountId *string `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of the saml attribute values for a SoftLayer customer account.
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// The saml attribute values for a SoftLayer customer account.
	Attributes []Account_Authentication_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// The identity provider x509 certificate.
	Certificate *string `json:"certificate,omitempty" xmlrpc:"certificate,omitempty"`

	// The identity provider x509 certificate fingerprint.
	CertificateFingerprint *string `json:"certificateFingerprint,omitempty" xmlrpc:"certificateFingerprint,omitempty"`

	// The identity provider entity ID.
	EntityId *string `json:"entityId,omitempty" xmlrpc:"entityId,omitempty"`

	// The saml internal identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The service provider x509 certificate.
	ServiceProviderCertificate *string `json:"serviceProviderCertificate,omitempty" xmlrpc:"serviceProviderCertificate,omitempty"`

	// The service provider entity IDs.
	ServiceProviderEntityId *string `json:"serviceProviderEntityId,omitempty" xmlrpc:"serviceProviderEntityId,omitempty"`

	// The service provider public key.
	ServiceProviderPublicKey *string `json:"serviceProviderPublicKey,omitempty" xmlrpc:"serviceProviderPublicKey,omitempty"`

	// The service provider signle logout encoding.
	ServiceProviderSingleLogoutEncoding *string `json:"serviceProviderSingleLogoutEncoding,omitempty" xmlrpc:"serviceProviderSingleLogoutEncoding,omitempty"`

	// The service provider signle logout address.
	ServiceProviderSingleLogoutUrl *string `json:"serviceProviderSingleLogoutUrl,omitempty" xmlrpc:"serviceProviderSingleLogoutUrl,omitempty"`

	// The service provider signle sign on encoding.
	ServiceProviderSingleSignOnEncoding *string `json:"serviceProviderSingleSignOnEncoding,omitempty" xmlrpc:"serviceProviderSingleSignOnEncoding,omitempty"`

	// The service provider signle sign on address.
	ServiceProviderSingleSignOnUrl *string `json:"serviceProviderSingleSignOnUrl,omitempty" xmlrpc:"serviceProviderSingleSignOnUrl,omitempty"`

	// The identity provider single logout encoding.
	SingleLogoutEncoding *string `json:"singleLogoutEncoding,omitempty" xmlrpc:"singleLogoutEncoding,omitempty"`

	// The identity provider sigle logout address.
	SingleLogoutUrl *string `json:"singleLogoutUrl,omitempty" xmlrpc:"singleLogoutUrl,omitempty"`

	// The identity provider single sign on encoding.
	SingleSignOnEncoding *string `json:"singleSignOnEncoding,omitempty" xmlrpc:"singleSignOnEncoding,omitempty"`

	// The identity provider signle sign on address.
	SingleSignOnUrl *string `json:"singleSignOnUrl,omitempty" xmlrpc:"singleSignOnUrl,omitempty"`
}

// no documentation yet
type Account_Classification_Group_Type struct {
	Entity

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`
}

// no documentation yet
type Account_Contact struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// no documentation yet
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// no documentation yet
	AlternatePhone *string `json:"alternatePhone,omitempty" xmlrpc:"alternatePhone,omitempty"`

	// no documentation yet
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// no documentation yet
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// no documentation yet
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// no documentation yet
	FaxPhone *string `json:"faxPhone,omitempty" xmlrpc:"faxPhone,omitempty"`

	// no documentation yet
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	JobTitle *string `json:"jobTitle,omitempty" xmlrpc:"jobTitle,omitempty"`

	// no documentation yet
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	OfficePhone *string `json:"officePhone,omitempty" xmlrpc:"officePhone,omitempty"`

	// no documentation yet
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// no documentation yet
	ProfileName *string `json:"profileName,omitempty" xmlrpc:"profileName,omitempty"`

	// no documentation yet
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// no documentation yet
	Type *Account_Contact_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// no documentation yet
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// no documentation yet
	Url *string `json:"url,omitempty" xmlrpc:"url,omitempty"`
}

// no documentation yet
type Account_Contact_Type struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Account_Historical_Report struct {
	Entity
}

// no documentation yet
type Account_Link struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	DestinationAccountAlphanumericId *string `json:"destinationAccountAlphanumericId,omitempty" xmlrpc:"destinationAccountAlphanumericId,omitempty"`

	// no documentation yet
	DestinationAccountId *int `json:"destinationAccountId,omitempty" xmlrpc:"destinationAccountId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`

	// no documentation yet
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`
}

// no documentation yet
type Account_Link_Bluemix struct {
	Account_Link
}

// no documentation yet
type Account_Link_OpenStack struct {
	Account_Link

	// Pseudonym for destinationAccountAlphanumericId
	DomainId *string `json:"domainId,omitempty" xmlrpc:"domainId,omitempty"`
}

// OpenStack domain creation details
type Account_Link_OpenStack_DomainCreationDetails struct {
	Entity

	// Id for the domain this user was added to.
	DomainId *string `json:"domainId,omitempty" xmlrpc:"domainId,omitempty"`

	// Id for the user given the Cloud Admin role for this domain.
	UserId *string `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// Name for the user given the Cloud Admin role for this domain.
	UserName *string `json:"userName,omitempty" xmlrpc:"userName,omitempty"`
}

// Details required for OpenStack link request
type Account_Link_OpenStack_LinkRequest struct {
	Entity

	// Optional password
	DesiredPassword *string `json:"desiredPassword,omitempty" xmlrpc:"desiredPassword,omitempty"`

	// Optional projectName
	DesiredProjectName *string `json:"desiredProjectName,omitempty" xmlrpc:"desiredProjectName,omitempty"`

	// Required username
	DesiredUsername *string `json:"desiredUsername,omitempty" xmlrpc:"desiredUsername,omitempty"`
}

// OpenStack project creation details
type Account_Link_OpenStack_ProjectCreationDetails struct {
	Entity

	// Id for the domain this project was added to.
	DomainId *string `json:"domainId,omitempty" xmlrpc:"domainId,omitempty"`

	// Id for this project.
	ProjectId *string `json:"projectId,omitempty" xmlrpc:"projectId,omitempty"`

	// Name for this project.
	ProjectName *string `json:"projectName,omitempty" xmlrpc:"projectName,omitempty"`

	// Id for the user given the Project Admin role for this project.
	UserId *string `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// Name for the user given the Project Admin role for this project.
	UserName *string `json:"userName,omitempty" xmlrpc:"userName,omitempty"`
}

// OpenStack project details
type Account_Link_OpenStack_ProjectDetails struct {
	Entity

	// Id for this project.
	ProjectId *string `json:"projectId,omitempty" xmlrpc:"projectId,omitempty"`

	// Name for this project.
	ProjectName *string `json:"projectName,omitempty" xmlrpc:"projectName,omitempty"`
}

// no documentation yet
type Account_Link_ThePlanet struct {
	Account_Link
}

// no documentation yet
type Account_Link_Vendor struct {
	Entity

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Account_Lockdown_Request data type holds information on API requests from brand customers.
type Account_Lockdown_Request struct {
	Entity

	// Account ID associated with this lockdown request.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// Type of request.
	Action *string `json:"action,omitempty" xmlrpc:"action,omitempty"`

	// Timestamp when the lockdown request was initially made.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// ID of this lockdown request.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Timestamp when the lockdown request was modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Status of the lockdown request denoting whether it's been completed.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// no documentation yet
type Account_MasterServiceAgreement struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	Guid *string `json:"guid,omitempty" xmlrpc:"guid,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Account_Media data type contains information on a single piece of media associated with a Data Transfer Service request.
type Account_Media struct {
	Entity

	// The account to which the media belongs.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The customer user who created the media object.
	CreateUser *User_Customer `json:"createUser,omitempty" xmlrpc:"createUser,omitempty"`

	// The datacenter where the media resides.
	Datacenter *Location `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// The description of the media.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique id of the media.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The employee who last modified the media.
	ModifyEmployee *User_Employee `json:"modifyEmployee,omitempty" xmlrpc:"modifyEmployee,omitempty"`

	// The customer user who last modified the media.
	ModifyUser *User_Customer `json:"modifyUser,omitempty" xmlrpc:"modifyUser,omitempty"`

	// The request to which the media belongs.
	Request *Account_Media_Data_Transfer_Request `json:"request,omitempty" xmlrpc:"request,omitempty"`

	// The request id of the media.
	RequestId *int `json:"requestId,omitempty" xmlrpc:"requestId,omitempty"`

	// The manufacturer's serial number of the media.
	SerialNumber *string `json:"serialNumber,omitempty" xmlrpc:"serialNumber,omitempty"`

	// The media's type.
	Type *Account_Media_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The type id of the media.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// A guest's associated EVault network storage service account.
	Volume *Network_Storage `json:"volume,omitempty" xmlrpc:"volume,omitempty"`
}

// The SoftLayer_Account_Media_Data_Transfer_Request data type contains information on a single Data Transfer Service request. Creation of these requests is limited to SoftLayer customers through the SoftLayer Customer Portal.
type Account_Media_Data_Transfer_Request struct {
	Entity

	// The account to which the request belongs.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The account id of the request.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of the active tickets that are attached to the data transfer request.
	ActiveTicketCount *uint `json:"activeTicketCount,omitempty" xmlrpc:"activeTicketCount,omitempty"`

	// The active tickets that are attached to the data transfer request.
	ActiveTickets []Ticket `json:"activeTickets,omitempty" xmlrpc:"activeTickets,omitempty"`

	// The billing item for the original request.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The customer user who created the request.
	CreateUser *User_Customer `json:"createUser,omitempty" xmlrpc:"createUser,omitempty"`

	// The create user id of the request.
	CreateUserId *int `json:"createUserId,omitempty" xmlrpc:"createUserId,omitempty"`

	// The end date of the request.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// The unique id of the request.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The media of the request.
	Media *Account_Media `json:"media,omitempty" xmlrpc:"media,omitempty"`

	// The employee who last modified the request.
	ModifyEmployee *User_Employee `json:"modifyEmployee,omitempty" xmlrpc:"modifyEmployee,omitempty"`

	// The customer user who last modified the request.
	ModifyUser *User_Customer `json:"modifyUser,omitempty" xmlrpc:"modifyUser,omitempty"`

	// The modify user id of the request.
	ModifyUserId *int `json:"modifyUserId,omitempty" xmlrpc:"modifyUserId,omitempty"`

	// A count of the shipments of the request.
	ShipmentCount *uint `json:"shipmentCount,omitempty" xmlrpc:"shipmentCount,omitempty"`

	// The shipments of the request.
	Shipments []Account_Shipment `json:"shipments,omitempty" xmlrpc:"shipments,omitempty"`

	// The start date of the request.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// The status of the request.
	Status *Account_Media_Data_Transfer_Request_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The status id of the request.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// A count of all tickets that are attached to the data transfer request.
	TicketCount *uint `json:"ticketCount,omitempty" xmlrpc:"ticketCount,omitempty"`

	// All tickets that are attached to the data transfer request.
	Tickets []Ticket `json:"tickets,omitempty" xmlrpc:"tickets,omitempty"`
}

// The SoftLayer_Account_Media_Data_Transfer_Request_Status data type contains general information relating to the statuses to which a Data Transfer Request may be set.
type Account_Media_Data_Transfer_Request_Status struct {
	Entity

	// The description of the request status.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique id of the request status.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The unique keyname of the request status.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of the request status.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Account_Media_Type data type contains general information relating to the different types of media devices that SoftLayer currently supports, as part of the Data Transfer Request Service. Such devices as USB hard drives and flash drives, as well as optical media such as CD and DVD are currently supported.
type Account_Media_Type struct {
	Entity

	// The description of the media type.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique id of the media type.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The unique keyname of the media type.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of the media type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Account_Network_Vlan_Span data type exposes the setting which controls the automatic spanning of private VLANs attached to a given customers account.
type Account_Network_Vlan_Span struct {
	Entity

	// The SoftLayer customer account associated with a VLAN.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// Flag indicating whether the customer wishes to have all private network VLANs associated with account automatically joined [0 or 1]
	EnabledFlag *bool `json:"enabledFlag,omitempty" xmlrpc:"enabledFlag,omitempty"`

	// The unique internal identifier of the SoftLayer_Account_Network_Vlan_Span object.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Timestamp of the last time the ACL for this account was applied.
	LastAppliedDate *Time `json:"lastAppliedDate,omitempty" xmlrpc:"lastAppliedDate,omitempty"`

	// Timestamp of the last time the subnet hash was verified for this VLAN span record.
	LastVerifiedDate *Time `json:"lastVerifiedDate,omitempty" xmlrpc:"lastVerifiedDate,omitempty"`

	// Timestamp of the last edit of the record.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`
}

// no documentation yet
type Account_Note struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Customer *User_Customer `json:"customer,omitempty" xmlrpc:"customer,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	// no documentation yet
	NoteHistory []Account_Note_History `json:"noteHistory,omitempty" xmlrpc:"noteHistory,omitempty"`

	// A count of
	NoteHistoryCount *uint `json:"noteHistoryCount,omitempty" xmlrpc:"noteHistoryCount,omitempty"`

	// no documentation yet
	NoteType *Account_Note_Type `json:"noteType,omitempty" xmlrpc:"noteType,omitempty"`

	// no documentation yet
	NoteTypeId *int `json:"noteTypeId,omitempty" xmlrpc:"noteTypeId,omitempty"`

	// no documentation yet
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// no documentation yet
type Account_Note_History struct {
	Entity

	// no documentation yet
	AccountNote *Account_Note `json:"accountNote,omitempty" xmlrpc:"accountNote,omitempty"`

	// no documentation yet
	AccountNoteId *int `json:"accountNoteId,omitempty" xmlrpc:"accountNoteId,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Customer *User_Customer `json:"customer,omitempty" xmlrpc:"customer,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	// no documentation yet
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// no documentation yet
type Account_Note_Type struct {
	Entity

	// no documentation yet
	BrandId *int `json:"brandId,omitempty" xmlrpc:"brandId,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	ValueExpression *string `json:"valueExpression,omitempty" xmlrpc:"valueExpression,omitempty"`
}

// no documentation yet
type Account_Partner_Referral_Prospect struct {
	User_Customer_Prospect

	// no documentation yet
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// no documentation yet
	EmailAddress *string `json:"emailAddress,omitempty" xmlrpc:"emailAddress,omitempty"`

	// no documentation yet
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`
}

// The SoftLayer_Account_Password contains username, passwords and notes for services that may require for external applications such the Webcc interface for the EVault Storage service.
type Account_Password struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The SoftLayer customer account id that a username/password combination is associated with.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A username/password combination's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A simple description of a username/password combination. These notes don't affect portal functionality.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The password portion of a username/password combination.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// The service that an account/password combination is tied to.
	Type *Account_Password_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// An identifier relating to a username/password combinations's associated service.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// The username portion of a username/password combination.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// Every username and password combination associated with a SoftLayer customer account belongs to a service that SoftLayer provides. The relationship between a username/password and it's service is provided by the SoftLayer_Account_Password_Type data type. Each username/password belongs to a single service type.
type Account_Password_Type struct {
	Entity

	// A description of the use for the account username/password combination.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`
}

//
//
//
//
//
type Account_Regional_Registry_Detail struct {
	Entity

	// The account that this detail object belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The detail object's associated [[SoftLayer_Account|account]] id
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The date and time the detail object was created
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A count of references to the [[SoftLayer_Network_Subnet_Registration|registration objects]] that consume this detail object.
	DetailCount *uint `json:"detailCount,omitempty" xmlrpc:"detailCount,omitempty"`

	// The associated type of this detail object.
	DetailType *Account_Regional_Registry_Detail_Type `json:"detailType,omitempty" xmlrpc:"detailType,omitempty"`

	// The detail object's associated [[SoftLayer_Account_Regional_Registry_Detail_Type|type]] id
	DetailTypeId *int `json:"detailTypeId,omitempty" xmlrpc:"detailTypeId,omitempty"`

	// References to the [[SoftLayer_Network_Subnet_Registration|registration objects]] that consume this detail object.
	Details []Network_Subnet_Registration_Details `json:"details,omitempty" xmlrpc:"details,omitempty"`

	// Unique ID of the detail object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date and time the detail object was last modified
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The individual properties that define this detail object's values.
	Properties []Account_Regional_Registry_Detail_Property `json:"properties,omitempty" xmlrpc:"properties,omitempty"`

	// A count of the individual properties that define this detail object's values.
	PropertyCount *uint `json:"propertyCount,omitempty" xmlrpc:"propertyCount,omitempty"`

	// The associated RWhois handle of this detail object. Used only when detailed reassignments are necessary.
	RegionalInternetRegistryHandle *Account_Rwhois_Handle `json:"regionalInternetRegistryHandle,omitempty" xmlrpc:"regionalInternetRegistryHandle,omitempty"`

	// The detail object's associated [[SoftLayer_Account_Rwhois_Handle|RIR handle]] id
	RegionalInternetRegistryHandleId *int `json:"regionalInternetRegistryHandleId,omitempty" xmlrpc:"regionalInternetRegistryHandleId,omitempty"`
}

// Subnet registration properties are used to define various attributes of the [[SoftLayer_Account_Regional_Registry_Detail|detail objects]]. These properties are defined by the [[SoftLayer_Account_Regional_Registry_Detail_Property_Type]] objects, which describe the available value formats.
type Account_Regional_Registry_Detail_Property struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The [[SoftLayer_Account_Regional_Registry_Detail]] object this property belongs to
	Detail *Account_Regional_Registry_Detail `json:"detail,omitempty" xmlrpc:"detail,omitempty"`

	// Unique ID of the property object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The [[SoftLayer_Account_Regional_Registry_Detail_Property_Type]] object this property belongs to
	PropertyType *Account_Regional_Registry_Detail_Property_Type `json:"propertyType,omitempty" xmlrpc:"propertyType,omitempty"`

	// The numeric ID of the related [[SoftLayer_Account_Regional_Registry_Detail_Property_Type|property type object]]
	PropertyTypeId *int `json:"propertyTypeId,omitempty" xmlrpc:"propertyTypeId,omitempty"`

	// The numeric ID of the related [[SoftLayer_Account_Regional_Registry_Detail|detail object]]
	RegistrationDetailId *int `json:"registrationDetailId,omitempty" xmlrpc:"registrationDetailId,omitempty"`

	// When multiple properties exist for a property type, defines the position in the sequence of those properties
	SequencePosition *int `json:"sequencePosition,omitempty" xmlrpc:"sequencePosition,omitempty"`

	// The value of the property
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// Subnet Registration Detail Property Type objects describe the nature of a [[SoftLayer_Account_Regional_Registry_Detail_Property]] object. These types use [http://php.net/pcre.pattern.php Perl-Compatible Regular Expressions] to validate the value of a property object.
type Account_Regional_Registry_Detail_Property_Type struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Unique numeric ID of the property type object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Code-friendly string name of the property type
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Human-readable name of the property type
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A Perl-compatible regular expression used to describe the valid format of the property
	ValueExpression *string `json:"valueExpression,omitempty" xmlrpc:"valueExpression,omitempty"`
}

// Subnet Registration Detail Type objects describe the nature of a [[SoftLayer_Account_Regional_Registry_Detail]] object.
//
// The standard values for these objects are as follows: <ul> <li><strong>NETWORK</strong> - The detail object represents the information for a [[SoftLayer_Network_Subnet|subnet]]</li> <li><strong>NETWORK6</strong> - The detail object represents the information for an [[SoftLayer_Network_Subnet_Version6|IPv6 subnet]]</li> <li><strong>PERSON</strong> - The detail object represents the information for a customer with the RIR</li> </ul>
type Account_Regional_Registry_Detail_Type struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Unique numeric ID of the detail type object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Code-friendly string name of the detail type
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Human-readable name of the detail type
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Account_Regional_Registry_Detail_Version4_Person_Default data type contains general information relating to a single SoftLayer RIR account. RIR account information in this type such as names, addresses, and phone numbers are assigned to the registry only and not to users belonging to the account.
type Account_Regional_Registry_Detail_Version4_Person_Default struct {
	Account_Regional_Registry_Detail
}

// no documentation yet
type Account_Reports_Request struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A request's corresponding external contact, if one exists.
	AccountContact *Account_Contact `json:"accountContact,omitempty" xmlrpc:"accountContact,omitempty"`

	// no documentation yet
	AccountContactId *int `json:"accountContactId,omitempty" xmlrpc:"accountContactId,omitempty"`

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	ComplianceReportTypeId *string `json:"complianceReportTypeId,omitempty" xmlrpc:"complianceReportTypeId,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	EmployeeRecordId *int `json:"employeeRecordId,omitempty" xmlrpc:"employeeRecordId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Nda *string `json:"nda,omitempty" xmlrpc:"nda,omitempty"`

	// no documentation yet
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// no documentation yet
	Report *string `json:"report,omitempty" xmlrpc:"report,omitempty"`

	// Type of the report customer is requesting for.
	ReportType *Compliance_Report_Type `json:"reportType,omitempty" xmlrpc:"reportType,omitempty"`

	// no documentation yet
	RequestKey *string `json:"requestKey,omitempty" xmlrpc:"requestKey,omitempty"`

	// no documentation yet
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// no documentation yet
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// no documentation yet
	TicketId *int `json:"ticketId,omitempty" xmlrpc:"ticketId,omitempty"`

	// The customer user that initiated a report request.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// no documentation yet
	UsrRecordId *int `json:"usrRecordId,omitempty" xmlrpc:"usrRecordId,omitempty"`
}

// Provides a means of tracking handle identifiers at the various regional internet registries (RIRs). These objects are used by the [[SoftLayer_Network_Subnet_Registration (type)|SoftLayer_Network_Subnet_Registration]] objects to identify a customer or organization when a subnet is registered.
type Account_Rwhois_Handle struct {
	Entity

	// The account that this handle belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The handle object's associated [[SoftLayer_Account|account]] id
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The handle object's unique identifier as assigned by the RIR.
	Handle *string `json:"handle,omitempty" xmlrpc:"handle,omitempty"`

	// Unique ID of the handle object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`
}

// The SoftLayer_Account_Shipment data type contains information relating to a shipment. Basic information such as addresses, the shipment courier, and any tracking information for as shipment is accessible with this data type.
type Account_Shipment struct {
	Entity

	// The account to which the shipment belongs.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The account id of the shipment.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The courier handling the shipment.
	Courier *Auxiliary_Shipping_Courier `json:"courier,omitempty" xmlrpc:"courier,omitempty"`

	// The courier id of the shipment.
	CourierId *int `json:"courierId,omitempty" xmlrpc:"courierId,omitempty"`

	// The courier name of the shipment.
	CourierName *string `json:"courierName,omitempty" xmlrpc:"courierName,omitempty"`

	// The employee who created the shipment.
	CreateEmployee *User_Employee `json:"createEmployee,omitempty" xmlrpc:"createEmployee,omitempty"`

	// The customer user who created the shipment.
	CreateUser *User_Customer `json:"createUser,omitempty" xmlrpc:"createUser,omitempty"`

	// The create user id of the shipment.
	CreateUserId *int `json:"createUserId,omitempty" xmlrpc:"createUserId,omitempty"`

	// The address at which the shipment is received.
	DestinationAddress *Account_Address `json:"destinationAddress,omitempty" xmlrpc:"destinationAddress,omitempty"`

	// The destination address id of the shipment.
	DestinationAddressId *int `json:"destinationAddressId,omitempty" xmlrpc:"destinationAddressId,omitempty"`

	// The destination date of the shipment.
	DestinationDate *Time `json:"destinationDate,omitempty" xmlrpc:"destinationDate,omitempty"`

	// The unique id of the shipment.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The employee who last modified the shipment.
	ModifyEmployee *User_Employee `json:"modifyEmployee,omitempty" xmlrpc:"modifyEmployee,omitempty"`

	// The customer user who last modified the shipment.
	ModifyUser *User_Customer `json:"modifyUser,omitempty" xmlrpc:"modifyUser,omitempty"`

	// The modify user id of the shipment.
	ModifyUserId *int `json:"modifyUserId,omitempty" xmlrpc:"modifyUserId,omitempty"`

	// The shipment note (special handling instructions).
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	// The address from which the shipment is sent.
	OriginationAddress *Account_Address `json:"originationAddress,omitempty" xmlrpc:"originationAddress,omitempty"`

	// The origination address id of the shipment.
	OriginationAddressId *int `json:"originationAddressId,omitempty" xmlrpc:"originationAddressId,omitempty"`

	// The origination date of the shipment.
	OriginationDate *Time `json:"originationDate,omitempty" xmlrpc:"originationDate,omitempty"`

	// A count of the items in the shipment.
	ShipmentItemCount *uint `json:"shipmentItemCount,omitempty" xmlrpc:"shipmentItemCount,omitempty"`

	// The items in the shipment.
	ShipmentItems []Account_Shipment_Item `json:"shipmentItems,omitempty" xmlrpc:"shipmentItems,omitempty"`

	// The status of the shipment.
	Status *Account_Shipment_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The status id of the shipment.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// The tracking data for the shipment.
	TrackingData []Account_Shipment_Tracking_Data `json:"trackingData,omitempty" xmlrpc:"trackingData,omitempty"`

	// A count of the tracking data for the shipment.
	TrackingDataCount *uint `json:"trackingDataCount,omitempty" xmlrpc:"trackingDataCount,omitempty"`

	// The type of shipment (e.g. for Data Transfer Service or Colocation Service).
	Type *Account_Shipment_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The type id of the shipment.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`
}

// The SoftLayer_Account_Shipment_Item data type contains information relating to a shipment's item. Basic information such as addresses, the shipment courier, and any tracking information for as shipment is accessible with this data type.
type Account_Shipment_Item struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The description of the shipping item.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique id of the shipping item.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The package id of the shipping item.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`

	// The shipment to which this item belongs.
	Shipment *Account_Shipment `json:"shipment,omitempty" xmlrpc:"shipment,omitempty"`

	// The shipment id of the shipping item.
	ShipmentId *int `json:"shipmentId,omitempty" xmlrpc:"shipmentId,omitempty"`

	// The item id of the shipping item.
	ShipmentItemId *int `json:"shipmentItemId,omitempty" xmlrpc:"shipmentItemId,omitempty"`

	// The type of this shipment item.
	ShipmentItemType *Account_Shipment_Item_Type `json:"shipmentItemType,omitempty" xmlrpc:"shipmentItemType,omitempty"`

	// The item type id of the shipping item.
	ShipmentItemTypeId *int `json:"shipmentItemTypeId,omitempty" xmlrpc:"shipmentItemTypeId,omitempty"`
}

// no documentation yet
type Account_Shipment_Item_Type struct {
	Entity

	// DEPRECATED
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Account_Shipment_Resource_Type struct {
	Entity
}

// no documentation yet
type Account_Shipment_Status struct {
	Entity

	// DEPRECATED
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Account_Shipment_Tracking_Data data type contains information on a single piece of tracking information pertaining to a shipment. This tracking information tracking numbers by which the shipment may be tracked through the shipping courier.
type Account_Shipment_Tracking_Data struct {
	Entity

	// The employee who created the tracking datum.
	CreateEmployee *User_Employee `json:"createEmployee,omitempty" xmlrpc:"createEmployee,omitempty"`

	// The customer user who created the tracking datum.
	CreateUser *User_Customer `json:"createUser,omitempty" xmlrpc:"createUser,omitempty"`

	// The create user id of the tracking data.
	CreateUserId *int `json:"createUserId,omitempty" xmlrpc:"createUserId,omitempty"`

	// The unique id of the tracking data.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The employee who last modified the tracking datum.
	ModifyEmployee *User_Employee `json:"modifyEmployee,omitempty" xmlrpc:"modifyEmployee,omitempty"`

	// The customer user who last modified the tracking datum.
	ModifyUser *User_Customer `json:"modifyUser,omitempty" xmlrpc:"modifyUser,omitempty"`

	// The user id of the tracking data.
	ModifyUserId *int `json:"modifyUserId,omitempty" xmlrpc:"modifyUserId,omitempty"`

	// The package id of the tracking data.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`

	// The sequence of the tracking data.
	Sequence *int `json:"sequence,omitempty" xmlrpc:"sequence,omitempty"`

	// The shipment of the tracking datum.
	Shipment *Account_Shipment `json:"shipment,omitempty" xmlrpc:"shipment,omitempty"`

	// The shipment id of the tracking data.
	ShipmentId *int `json:"shipmentId,omitempty" xmlrpc:"shipmentId,omitempty"`

	// The tracking data (tracking number/reference number).
	TrackingData *string `json:"trackingData,omitempty" xmlrpc:"trackingData,omitempty"`
}

// no documentation yet
type Account_Shipment_Type struct {
	Entity

	// DEPRECATED
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

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
type Account_Status struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}
