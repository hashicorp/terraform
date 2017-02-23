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

// The SoftLayer_Account data type contains general information relating to a single SoftLayer customer account. Personal information in this type such as names, addresses, and phone numbers are assigned to the account only and not to users belonging to the account. The SoftLayer_Account data type contains a number of relational properties that are used by the SoftLayer customer portal to quickly present a variety of account related services to it's users.
//
// SoftLayer customers are unable to change their company account information in the portal or the API. If you need to change this information please open a sales ticket in our customer portal and our account management staff will assist you.
type Account struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountService returns an instance of the Account SoftLayer service
func GetAccountService(sess *session.Session) Account {
	return Account{Session: sess}
}

func (r Account) Id(id int) Account {
	r.Options.Id = &id
	return r
}

func (r Account) Mask(mask string) Account {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account) Filter(filter string) Account {
	r.Options.Filter = filter
	return r
}

func (r Account) Limit(limit int) Account {
	r.Options.Limit = &limit
	return r
}

func (r Account) Offset(offset int) Account {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account) ActivatePartner(accountId *string, hashCode *string) (resp datatypes.Account, err error) {
	params := []interface{}{
		accountId,
		hashCode,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "activatePartner", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) AddAchInformation(achInformation *datatypes.Container_Billing_Info_Ach) (resp bool, err error) {
	params := []interface{}{
		achInformation,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "addAchInformation", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) AddReferralPartnerPaymentOption(paymentOption *datatypes.Container_Referral_Partner_Payment_Option) (resp bool, err error) {
	params := []interface{}{
		paymentOption,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "addReferralPartnerPaymentOption", params, &r.Options, &resp)
	return
}

// This method indicates whether or not Bandwidth Pooling updates are blocked for the account so the billing cycle can run.  Generally, accounts are restricted from moving servers in or out of Bandwidth Pools from 12:00 CST on the day prior to billing, until the billing batch completes, sometime after midnight the day of actual billing for the account.
func (r Account) AreVdrUpdatesBlockedForBilling() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "areVdrUpdatesBlockedForBilling", nil, &r.Options, &resp)
	return
}

// Cancel the PayPal Payment Request process. During the process of submitting a PayPal payment request, the customer is redirected to PayPal to confirm the request.  If the customer elects to cancel the payment from PayPal, they are returned to SoftLayer where the manual payment record is updated to a status of canceled.
func (r Account) CancelPayPalTransaction(token *string, payerId *string) (resp bool, err error) {
	params := []interface{}{
		token,
		payerId,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "cancelPayPalTransaction", params, &r.Options, &resp)
	return
}

// Complete the PayPal Payment Request process and receive confirmation message. During the process of submitting a PayPal payment request, the customer is redirected to PayPal to confirm the request.  Once confirmed, PayPal returns the customer to SoftLayer where an attempt is made to finalize the transaction.  A status message regarding the attempt is returned to the calling function.
func (r Account) CompletePayPalTransaction(token *string, payerId *string) (resp string, err error) {
	params := []interface{}{
		token,
		payerId,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "completePayPalTransaction", params, &r.Options, &resp)
	return
}

// Retrieve the number of hourly services on an account that are active, plus any pending orders with hourly services attached.
func (r Account) CountHourlyInstances() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "countHourlyInstances", nil, &r.Options, &resp)
	return
}

// Retrieve An email address that is responsible for abuse and legal inquiries on behalf of an account. For instance, new legal and abuse tickets are sent to this address.
func (r Account) GetAbuseEmail() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAbuseEmail", nil, &r.Options, &resp)
	return
}

// Retrieve Email addresses that are responsible for abuse and legal inquiries on behalf of an account. For instance, new legal and abuse tickets are sent to these addresses.
func (r Account) GetAbuseEmails() (resp []datatypes.Account_AbuseEmail, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAbuseEmails", nil, &r.Options, &resp)
	return
}

// This method returns an array of SoftLayer_Container_Network_Storage_Evault_WebCc_JobDetails objects for the given start and end dates. Start and end dates should be be valid ISO 8601 dates. The backupStatus can be one of null, 'success', 'failed', or 'conflict'. The 'success' backupStatus returns jobs with a status of 'COMPLETED', the 'failed' backupStatus returns jobs with a status of 'FAILED', while the 'conflict' backupStatus will return jobs that are not 'COMPLETED' or 'FAILED'.
func (r Account) GetAccountBackupHistory(startDate *datatypes.Time, endDate *datatypes.Time, backupStatus *string) (resp []datatypes.Container_Network_Storage_Evault_WebCc_JobDetails, err error) {
	params := []interface{}{
		startDate,
		endDate,
		backupStatus,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getAccountBackupHistory", params, &r.Options, &resp)
	return
}

// Retrieve The account contacts on an account.
func (r Account) GetAccountContacts() (resp []datatypes.Account_Contact, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAccountContacts", nil, &r.Options, &resp)
	return
}

// Retrieve The account software licenses owned by an account
func (r Account) GetAccountLicenses() (resp []datatypes.Software_AccountLicense, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAccountLicenses", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetAccountLinks() (resp []datatypes.Account_Link, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAccountLinks", nil, &r.Options, &resp)
	return
}

// Retrieve An account's status presented in a more detailed data type.
func (r Account) GetAccountStatus() (resp datatypes.Account_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAccountStatus", nil, &r.Options, &resp)
	return
}

// This method pulls an account trait by its key.
func (r Account) GetAccountTraitValue(keyName *string) (resp string, err error) {
	params := []interface{}{
		keyName,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getAccountTraitValue", params, &r.Options, &resp)
	return
}

// Retrieve The billing item associated with an account's monthly discount.
func (r Account) GetActiveAccountDiscountBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveAccountDiscountBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The active account software licenses owned by an account
func (r Account) GetActiveAccountLicenses() (resp []datatypes.Software_AccountLicense, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveAccountLicenses", nil, &r.Options, &resp)
	return
}

// Retrieve The active address(es) that belong to an account.
func (r Account) GetActiveAddresses() (resp []datatypes.Account_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveAddresses", nil, &r.Options, &resp)
	return
}

// Return all currently active alarms on this account.  Only alarms on hardware and virtual servers accessible to the current user will be returned.
func (r Account) GetActiveAlarms() (resp []datatypes.Container_Monitoring_Alarm_History, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveAlarms", nil, &r.Options, &resp)
	return
}

// Retrieve All billing agreements for an account
func (r Account) GetActiveBillingAgreements() (resp []datatypes.Account_Agreement, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveBillingAgreements", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetActiveCatalystEnrollment() (resp datatypes.Catalyst_Enrollment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveCatalystEnrollment", nil, &r.Options, &resp)
	return
}

// Retrieve The account's active top level colocation containers.
func (r Account) GetActiveColocationContainers() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveColocationContainers", nil, &r.Options, &resp)
	return
}

// Retrieve Account's currently active Flexible Credit enrollment.
func (r Account) GetActiveFlexibleCreditEnrollment() (resp datatypes.FlexibleCredit_Enrollment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveFlexibleCreditEnrollment", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetActiveNotificationSubscribers() (resp []datatypes.Notification_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveNotificationSubscribers", nil, &r.Options, &resp)
	return
}

// This method pulls all the active packages. This will give you a basic description of the packages within the SoftLayer Outlet store that are currently active and from which you can order a server or additional services.
func (r Account) GetActiveOutletPackages() (resp []datatypes.Product_Package, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveOutletPackages", nil, &r.Options, &resp)
	return
}

// This method will return the [[SoftLayer_Product_Package]] objects from which you can order a bare metal server, virtual server, service (such as CDN or Object Storage) or other software. Once you have the package you want to order from, you may query one of various endpoints from that package to get specific information about its products and pricing. See [[SoftLayer_Product_Package/getCategories|getCategories]] or [[SoftLayer_Product_Package/getItems|getItems]] for more information.
//
// Packages that have been retired will not appear in this result set.
func (r Account) GetActivePackages() (resp []datatypes.Product_Package, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActivePackages", nil, &r.Options, &resp)
	return
}

// <strong>This method is deprecated and should not be used in production code.</strong>
//
// This method will return the [[SoftLayer_Product_Package]] objects from which you can order a bare metal server, virtual server, service (such as CDN or Object Storage) or other software filtered by an attribute type associated with the package. Once you have the package you want to order from, you may query one of various endpoints from that package to get specific information about its products and pricing. See [[SoftLayer_Product_Package/getCategories|getCategories]] or [[SoftLayer_Product_Package/getItems|getItems]] for more information.
func (r Account) GetActivePackagesByAttribute(attributeKeyName *string) (resp []datatypes.Product_Package, err error) {
	params := []interface{}{
		attributeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getActivePackagesByAttribute", params, &r.Options, &resp)
	return
}

// This method pulls all the active private hosted cloud packages. This will give you a basic description of the packages that are currently active and from which you can order private hosted cloud configurations.
func (r Account) GetActivePrivateHostedCloudPackages() (resp []datatypes.Product_Package, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActivePrivateHostedCloudPackages", nil, &r.Options, &resp)
	return
}

// Retrieve An account's non-expired quotes.
func (r Account) GetActiveQuotes() (resp []datatypes.Billing_Order_Quote, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveQuotes", nil, &r.Options, &resp)
	return
}

// Retrieve The virtual software licenses controlled by an account
func (r Account) GetActiveVirtualLicenses() (resp []datatypes.Software_VirtualLicense, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getActiveVirtualLicenses", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated load balancers.
func (r Account) GetAdcLoadBalancers() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAdcLoadBalancers", nil, &r.Options, &resp)
	return
}

// Retrieve All the address(es) that belong to an account.
func (r Account) GetAddresses() (resp []datatypes.Account_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve An affiliate identifier associated with the customer account.
func (r Account) GetAffiliateId() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAffiliateId", nil, &r.Options, &resp)
	return
}

// Returns URL uptime data for your account
func (r Account) GetAggregatedUptimeGraph(startDate *datatypes.Time, endDate *datatypes.Time) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getAggregatedUptimeGraph", params, &r.Options, &resp)
	return
}

// Retrieve The billing items that will be on an account's next invoice.
func (r Account) GetAllBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllBillingItems", nil, &r.Options, &resp)
	return
}

// Retrieve The billing items that will be on an account's next invoice.
func (r Account) GetAllCommissionBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllCommissionBillingItems", nil, &r.Options, &resp)
	return
}

// Retrieve The billing items that will be on an account's next invoice.
func (r Account) GetAllRecurringTopLevelBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllRecurringTopLevelBillingItems", nil, &r.Options, &resp)
	return
}

// Retrieve The billing items that will be on an account's next invoice. Does not consider associated items.
func (r Account) GetAllRecurringTopLevelBillingItemsUnfiltered() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllRecurringTopLevelBillingItemsUnfiltered", nil, &r.Options, &resp)
	return
}

// Retrieve The billing items that will be on an account's next invoice.
func (r Account) GetAllSubnetBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllSubnetBillingItems", nil, &r.Options, &resp)
	return
}

// Retrieve All billing items of an account.
func (r Account) GetAllTopLevelBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllTopLevelBillingItems", nil, &r.Options, &resp)
	return
}

// Retrieve The billing items that will be on an account's next invoice. Does not consider associated items.
func (r Account) GetAllTopLevelBillingItemsUnfiltered() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllTopLevelBillingItemsUnfiltered", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates whether this account is allowed to silently migrate to use IBMid Authentication.
func (r Account) GetAllowIbmIdSilentMigrationFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllowIbmIdSilentMigrationFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Flag indicating if this account can be linked with Bluemix.
func (r Account) GetAllowsBluemixAccountLinkingFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAllowsBluemixAccountLinkingFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetAlternateCreditCardData() (resp datatypes.Container_Account_Payment_Method_CreditCard, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAlternateCreditCardData", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated application delivery controller records.
func (r Account) GetApplicationDeliveryControllers() (resp []datatypes.Network_Application_Delivery_Controller, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getApplicationDeliveryControllers", nil, &r.Options, &resp)
	return
}

// Retrieve a single [[SoftLayer_Account_Attribute]] record by its [[SoftLayer_Account_Attribute_Type|types's]] key name.
func (r Account) GetAttributeByType(attributeType *string) (resp datatypes.Account_Attribute, err error) {
	params := []interface{}{
		attributeType,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getAttributeByType", params, &r.Options, &resp)
	return
}

// Retrieve The account attribute values for a SoftLayer customer account.
func (r Account) GetAttributes() (resp []datatypes.Account_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAttributes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetAuxiliaryNotifications() (resp []datatypes.Container_Utility_Message, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAuxiliaryNotifications", nil, &r.Options, &resp)
	return
}

// Retrieve The public network VLANs assigned to an account.
func (r Account) GetAvailablePublicNetworkVlans() (resp []datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getAvailablePublicNetworkVlans", nil, &r.Options, &resp)
	return
}

// Returns the average disk space usage for all archive repositories.
func (r Account) GetAverageArchiveUsageMetricDataByDate(startDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp datatypes.Float64, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getAverageArchiveUsageMetricDataByDate", params, &r.Options, &resp)
	return
}

// Returns the average disk space usage for all public repositories.
func (r Account) GetAveragePublicUsageMetricDataByDate(startDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp datatypes.Float64, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getAveragePublicUsageMetricDataByDate", params, &r.Options, &resp)
	return
}

// Retrieve The account balance of a SoftLayer customer account. An account's balance is the amount of money owed to SoftLayer by the account holder, returned as a floating point number with two decimal places, measured in US Dollars ($USD). A negative account balance means the account holder has overpaid and is owed money by SoftLayer.
func (r Account) GetBalance() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBalance", nil, &r.Options, &resp)
	return
}

// Retrieve The bandwidth allotments for an account.
func (r Account) GetBandwidthAllotments() (resp []datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBandwidthAllotments", nil, &r.Options, &resp)
	return
}

// Retrieve The bandwidth allotments for an account currently over allocation.
func (r Account) GetBandwidthAllotmentsOverAllocation() (resp []datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBandwidthAllotmentsOverAllocation", nil, &r.Options, &resp)
	return
}

// Retrieve The bandwidth allotments for an account projected to go over allocation.
func (r Account) GetBandwidthAllotmentsProjectedOverAllocation() (resp []datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBandwidthAllotmentsProjectedOverAllocation", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated bare metal server objects.
func (r Account) GetBareMetalInstances() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBareMetalInstances", nil, &r.Options, &resp)
	return
}

// Retrieve All billing agreements for an account
func (r Account) GetBillingAgreements() (resp []datatypes.Account_Agreement, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBillingAgreements", nil, &r.Options, &resp)
	return
}

// Retrieve An account's billing information.
func (r Account) GetBillingInfo() (resp datatypes.Billing_Info, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBillingInfo", nil, &r.Options, &resp)
	return
}

// Retrieve Private template group objects (parent and children) and the shared template group objects (parent only) for an account.
func (r Account) GetBlockDeviceTemplateGroups() (resp []datatypes.Virtual_Guest_Block_Device_Template_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBlockDeviceTemplateGroups", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates whether this account requires blue id authentication.
func (r Account) GetBlueIdAuthenticationRequiredFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBlueIdAuthenticationRequiredFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Returns true if this account is linked to IBM Bluemix, false if not.
func (r Account) GetBluemixLinkedFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBluemixLinkedFlag", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetBrand() (resp datatypes.Brand, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBrand", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetBrandAccountFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBrandAccountFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The brand keyName.
func (r Account) GetBrandKeyName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getBrandKeyName", nil, &r.Options, &resp)
	return
}

// Retrieve Indicating whether this account can order additional Vlans.
func (r Account) GetCanOrderAdditionalVlansFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getCanOrderAdditionalVlansFlag", nil, &r.Options, &resp)
	return
}

// Retrieve An account's active carts.
func (r Account) GetCarts() (resp []datatypes.Billing_Order_Quote, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getCarts", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetCatalystEnrollments() (resp []datatypes.Catalyst_Enrollment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getCatalystEnrollments", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated CDN accounts.
func (r Account) GetCdnAccounts() (resp []datatypes.Network_ContentDelivery_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getCdnAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve All closed tickets associated with an account.
func (r Account) GetClosedTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getClosedTickets", nil, &r.Options, &resp)
	return
}

// This method returns a SoftLayer_Container_Account_Graph_Outputs containing a base64 string PNG image. The optional parameter, detailedGraph, can be passed to get a more detailed graph.
func (r Account) GetCurrentBackupStatisticsGraph(detailedGraph *bool) (resp datatypes.Container_Account_Graph_Outputs, err error) {
	params := []interface{}{
		detailedGraph,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getCurrentBackupStatisticsGraph", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetCurrentTicketStatisticsGraph(detailedGraph *bool) (resp datatypes.Container_Account_Graph_Outputs, err error) {
	params := []interface{}{
		detailedGraph,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getCurrentTicketStatisticsGraph", params, &r.Options, &resp)
	return
}

// Retrieve the user record of the user calling the SoftLayer API.
func (r Account) GetCurrentUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getCurrentUser", nil, &r.Options, &resp)
	return
}

// Retrieve Datacenters which contain subnets that the account has access to route.
func (r Account) GetDatacentersWithSubnetAllocations() (resp []datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getDatacentersWithSubnetAllocations", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating whether payments are processed for this account.
func (r Account) GetDisablePaymentProcessingFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getDisablePaymentProcessingFlag", nil, &r.Options, &resp)
	return
}

// Retrieve disk usage data on a [[SoftLayer_Virtual_Guest|Cloud Computing Instance]] image for the time range you provide from the Metric Tracking Object System and Legacy Data Warehouse. Each data entry objects contain ''dateTime'' and ''counter'' properties. ''dateTime'' property indicates the time that the disk usage data was measured and ''counter'' property holds the disk usage in bytes.
func (r Account) GetDiskUsageMetricDataByDate(startDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getDiskUsageMetricDataByDate", params, &r.Options, &resp)
	return
}

// Retrieve disk usage data on a [[SoftLayer_Virtual_Guest|Cloud Computing Instance]] image for the time range you provide from the Legacy Data Warehouse.  Each data entry objects contain ''dateTime'' and ''counter'' properties. ''dateTime'' property indicates the time that the disk usage data was measured and ''counter'' property holds the disk usage in bytes.
func (r Account) GetDiskUsageMetricDataFromLegacyByDate(startDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getDiskUsageMetricDataFromLegacyByDate", params, &r.Options, &resp)
	return
}

// Retrieve disk usage data on a [[SoftLayer_Virtual_Guest|Cloud Computing Instance]] image for the time range you provide from the Metric Tracking Object System.  Each data entry object contains ''dateTime'' and ''counter'' properties.  ''dateTime'' property indicates the time that the disk usage data was measured and ''counter'' property holds the disk usage in bytes.
func (r Account) GetDiskUsageMetricDataFromMetricTrackingObjectSystemByDate(startDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getDiskUsageMetricDataFromMetricTrackingObjectSystemByDate", params, &r.Options, &resp)
	return
}

// Returns a disk usage image based on disk usage specified by the input parameters.
func (r Account) GetDiskUsageMetricImageByDate(startDateTime *datatypes.Time, endDateTime *datatypes.Time) (resp datatypes.Container_Account_Graph_Outputs, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getDiskUsageMetricImageByDate", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer employees that an account is assigned to.
func (r Account) GetDisplaySupportRepresentativeAssignments() (resp []datatypes.Account_Attachment_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getDisplaySupportRepresentativeAssignments", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetDomainRegistrations() (resp []datatypes.Dns_Domain_Registration, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getDomainRegistrations", nil, &r.Options, &resp)
	return
}

// Retrieve The DNS domains associated with an account.
func (r Account) GetDomains() (resp []datatypes.Dns_Domain, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getDomains", nil, &r.Options, &resp)
	return
}

// Retrieve The DNS domains associated with an account that were not created as a result of a secondary DNS zone transfer.
func (r Account) GetDomainsWithoutSecondaryDnsRecords() (resp []datatypes.Dns_Domain, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getDomainsWithoutSecondaryDnsRecords", nil, &r.Options, &resp)
	return
}

// Retrieve The total capacity of Legacy EVault Volumes on an account, in GB.
func (r Account) GetEvaultCapacityGB() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getEvaultCapacityGB", nil, &r.Options, &resp)
	return
}

// Retrieve An account's master EVault user. This is only used when an account has EVault service.
func (r Account) GetEvaultMasterUsers() (resp []datatypes.Account_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getEvaultMasterUsers", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated EVault storage volumes.
func (r Account) GetEvaultNetworkStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getEvaultNetworkStorage", nil, &r.Options, &resp)
	return
}

// This method will return a PDF of the specified report, with the specified period within the start and end dates. The pdfType must be one of 'snapshot', or 'historical'. Possible historicalType parameters are 'monthly', 'yearly', and 'quarterly'. Start and end dates should be in ISO 8601 date format.
func (r Account) GetExecutiveSummaryPdf(pdfType *string, historicalType *string, startDate *string, endDate *string) (resp []byte, err error) {
	params := []interface{}{
		pdfType,
		historicalType,
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getExecutiveSummaryPdf", params, &r.Options, &resp)
	return
}

// Retrieve Stored security certificates that are expired (ie. SSL)
func (r Account) GetExpiredSecurityCertificates() (resp []datatypes.Security_Certificate, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getExpiredSecurityCertificates", nil, &r.Options, &resp)
	return
}

// Retrieve Logs of who entered a colocation area which is assigned to this account, or when a user under this account enters a datacenter.
func (r Account) GetFacilityLogs() (resp []datatypes.User_Access_Facility_Log, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getFacilityLogs", nil, &r.Options, &resp)
	return
}

// Retrieve All of the account's current and former Flexible Credit enrollments.
func (r Account) GetFlexibleCreditEnrollments() (resp []datatypes.FlexibleCredit_Enrollment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getFlexibleCreditEnrollments", nil, &r.Options, &resp)
	return
}

// This method will return a [[SoftLayer_Container_Account_Discount_Program]] object containing the Flexible Credit Program information for this account. To be considered an active participant, the account must have an enrollment record with a monthly credit amount set and the current date must be within the range defined by the enrollment and graduation date. The forNextBillCycle parameter can be set to true to return a SoftLayer_Container_Account_Discount_Program object with information with relation to the next bill cycle. The forNextBillCycle parameter defaults to false. Please note that all discount amount entries are reported as pre-tax amounts and the legacy tax fields in the [[SoftLayer_Container_Account_Discount_Program]] are deprecated.
func (r Account) GetFlexibleCreditProgramInfo(forNextBillCycle *bool) (resp datatypes.Container_Account_Discount_Program, err error) {
	params := []interface{}{
		forNextBillCycle,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getFlexibleCreditProgramInfo", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetGlobalIpRecords() (resp []datatypes.Network_Subnet_IpAddress_Global, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getGlobalIpRecords", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetGlobalIpv4Records() (resp []datatypes.Network_Subnet_IpAddress_Global, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getGlobalIpv4Records", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetGlobalIpv6Records() (resp []datatypes.Network_Subnet_IpAddress_Global, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getGlobalIpv6Records", nil, &r.Options, &resp)
	return
}

// Retrieve The global load balancer accounts for a softlayer customer account.
func (r Account) GetGlobalLoadBalancerAccounts() (resp []datatypes.Network_LoadBalancer_Global_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getGlobalLoadBalancerAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated hardware objects.
func (r Account) GetHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated hardware objects currently over bandwidth allocation.
func (r Account) GetHardwareOverBandwidthAllocation() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareOverBandwidthAllocation", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated hardware objects projected to go over bandwidth allocation.
func (r Account) GetHardwareProjectedOverBandwidthAllocation() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareProjectedOverBandwidthAllocation", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has the cPanel web hosting control panel installed.
func (r Account) GetHardwareWithCpanel() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithCpanel", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has the Helm web hosting control panel installed.
func (r Account) GetHardwareWithHelm() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithHelm", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has McAfee Secure software components.
func (r Account) GetHardwareWithMcafee() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithMcafee", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has McAfee Secure AntiVirus for Redhat software components.
func (r Account) GetHardwareWithMcafeeAntivirusRedhat() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithMcafeeAntivirusRedhat", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has McAfee Secure AntiVirus for Windows software components.
func (r Account) GetHardwareWithMcafeeAntivirusWindows() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithMcafeeAntivirusWindows", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has McAfee Secure Intrusion Detection System software components.
func (r Account) GetHardwareWithMcafeeIntrusionDetectionSystem() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithMcafeeIntrusionDetectionSystem", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has the Plesk web hosting control panel installed.
func (r Account) GetHardwareWithPlesk() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithPlesk", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has the QuantaStor storage system installed.
func (r Account) GetHardwareWithQuantastor() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithQuantastor", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that has the Urchin web traffic analytics package installed.
func (r Account) GetHardwareWithUrchin() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithUrchin", nil, &r.Options, &resp)
	return
}

// Retrieve All hardware associated with an account that is running a version of the Microsoft Windows operating system.
func (r Account) GetHardwareWithWindows() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHardwareWithWindows", nil, &r.Options, &resp)
	return
}

// Retrieve Return 1 if one of the account's hardware has the EVault Bare Metal Server Restore Plugin otherwise 0.
func (r Account) GetHasEvaultBareMetalRestorePluginFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHasEvaultBareMetalRestorePluginFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Return 1 if one of the account's hardware has an installation of Idera Server Backup otherwise 0.
func (r Account) GetHasIderaBareMetalRestorePluginFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHasIderaBareMetalRestorePluginFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The number of orders in a PENDING status for a SoftLayer customer account.
func (r Account) GetHasPendingOrder() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHasPendingOrder", nil, &r.Options, &resp)
	return
}

// Retrieve Return 1 if one of the account's hardware has an installation of R1Soft CDP otherwise 0.
func (r Account) GetHasR1softBareMetalRestorePluginFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHasR1softBareMetalRestorePluginFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetHistoricalBackupGraph(startDate *datatypes.Time, endDate *datatypes.Time) (resp datatypes.Container_Account_Graph_Outputs, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getHistoricalBackupGraph", params, &r.Options, &resp)
	return
}

// This method will return a SoftLayer_Container_Account_Graph_Outputs object containing a base64 string PNG image of a line graph of bandwidth statistics given the start and end dates. The start and end dates should be valid ISO 8601 date formatted strings.
func (r Account) GetHistoricalBandwidthGraph(startDate *datatypes.Time, endDate *datatypes.Time) (resp datatypes.Container_Account_Graph_Outputs, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getHistoricalBandwidthGraph", params, &r.Options, &resp)
	return
}

// Given the start and end dates, this method will return a pie chart of ticket statistics in the form of SoftLayer_Container_Account_Graph_Outputs object with a base64 PNG string. If an error occurs the graphError parameter will be populated. Possible errors include: SoftLayer_Exception_Public Thrown if an invalid start or end date is provided. SoftLayer_Exception Thrown if there is an error connecting to HBase. SoftLayer_Exception Thrown if there is no data available for the specified date range. SoftLayer_Exception Thrown if there is an error retrieving data or generating the graph.
func (r Account) GetHistoricalTicketGraph(startDate *datatypes.Time, endDate *datatypes.Time) (resp datatypes.Container_Account_Graph_Outputs, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getHistoricalTicketGraph", params, &r.Options, &resp)
	return
}

// The graph image is returned as a base64 PNG string. Start and end dates should be formatted using the ISO 8601 date standard. If there is an error retrieving graph data or generating the graph string a graphError attribute will be returned. The graphError attribute may contain any of the following error messages: SoftLayer_Exception_Public Thrown if an invalid start or end date is provided. SoftLayer_Exception Thrown if there is an error connecting to HBase. SoftLayer_Exception Thrown if there is no data available for the specified date range. SoftLayer_Exception Thrown if there is an error retrieving data or generating the graph.
func (r Account) GetHistoricalUptimeGraph(startDate *datatypes.Time, endDate *datatypes.Time) (resp datatypes.Container_Account_Graph_Outputs, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getHistoricalUptimeGraph", params, &r.Options, &resp)
	return
}

// Retrieve An account's associated hourly bare metal server objects.
func (r Account) GetHourlyBareMetalInstances() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHourlyBareMetalInstances", nil, &r.Options, &resp)
	return
}

// Retrieve Hourly service billing items that will be on an account's next invoice.
func (r Account) GetHourlyServiceBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHourlyServiceBillingItems", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated hourly virtual guest objects.
func (r Account) GetHourlyVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHourlyVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated Virtual Storage volumes.
func (r Account) GetHubNetworkStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getHubNetworkStorage", nil, &r.Options, &resp)
	return
}

// Retrieve Timestamp representing the point in time when an account is required to use IBMid authentication.
func (r Account) GetIbmIdMigrationExpirationTimestamp() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getIbmIdMigrationExpirationTimestamp", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetInternalNotes() (resp []datatypes.Account_Note, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getInternalNotes", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated billing invoices.
func (r Account) GetInvoices() (resp []datatypes.Billing_Invoice, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getInvoices", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated iSCSI storage volumes.
func (r Account) GetIscsiNetworkStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getIscsiNetworkStorage", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetLargestAllowedSubnetCidr(numberOfHosts *int, locationId *int) (resp int, err error) {
	params := []interface{}{
		numberOfHosts,
		locationId,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getLargestAllowedSubnetCidr", params, &r.Options, &resp)
	return
}

// Retrieve The most recently canceled billing item.
func (r Account) GetLastCanceledBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLastCanceledBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The most recent cancelled server billing item.
func (r Account) GetLastCancelledServerBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLastCancelledServerBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The five most recently closed abuse tickets associated with an account.
func (r Account) GetLastFiveClosedAbuseTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLastFiveClosedAbuseTickets", nil, &r.Options, &resp)
	return
}

// Retrieve The five most recently closed accounting tickets associated with an account.
func (r Account) GetLastFiveClosedAccountingTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLastFiveClosedAccountingTickets", nil, &r.Options, &resp)
	return
}

// Retrieve The five most recently closed tickets that do not belong to the abuse, accounting, sales, or support groups associated with an account.
func (r Account) GetLastFiveClosedOtherTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLastFiveClosedOtherTickets", nil, &r.Options, &resp)
	return
}

// Retrieve The five most recently closed sales tickets associated with an account.
func (r Account) GetLastFiveClosedSalesTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLastFiveClosedSalesTickets", nil, &r.Options, &resp)
	return
}

// Retrieve The five most recently closed support tickets associated with an account.
func (r Account) GetLastFiveClosedSupportTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLastFiveClosedSupportTickets", nil, &r.Options, &resp)
	return
}

// Retrieve The five most recently closed tickets associated with an account.
func (r Account) GetLastFiveClosedTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLastFiveClosedTickets", nil, &r.Options, &resp)
	return
}

// Retrieve An account's most recent billing date.
func (r Account) GetLatestBillDate() (resp datatypes.Time, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLatestBillDate", nil, &r.Options, &resp)
	return
}

// Retrieve An account's latest recurring invoice.
func (r Account) GetLatestRecurringInvoice() (resp datatypes.Billing_Invoice, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLatestRecurringInvoice", nil, &r.Options, &resp)
	return
}

// Retrieve An account's latest recurring pending invoice.
func (r Account) GetLatestRecurringPendingInvoice() (resp datatypes.Billing_Invoice, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLatestRecurringPendingInvoice", nil, &r.Options, &resp)
	return
}

// Retrieve The legacy bandwidth allotments for an account.
func (r Account) GetLegacyBandwidthAllotments() (resp []datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLegacyBandwidthAllotments", nil, &r.Options, &resp)
	return
}

// Retrieve The total capacity of Legacy iSCSI Volumes on an account, in GB.
func (r Account) GetLegacyIscsiCapacityGB() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLegacyIscsiCapacityGB", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated load balancers.
func (r Account) GetLoadBalancers() (resp []datatypes.Network_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLoadBalancers", nil, &r.Options, &resp)
	return
}

// Retrieve The total capacity of Legacy lockbox Volumes on an account, in GB.
func (r Account) GetLockboxCapacityGB() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLockboxCapacityGB", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated Lockbox storage volumes.
func (r Account) GetLockboxNetworkStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getLockboxNetworkStorage", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetManualPaymentsUnderReview() (resp []datatypes.Billing_Payment_Card_ManualPayment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getManualPaymentsUnderReview", nil, &r.Options, &resp)
	return
}

// Retrieve An account's master user.
func (r Account) GetMasterUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getMasterUser", nil, &r.Options, &resp)
	return
}

// Retrieve An account's media transfer service requests.
func (r Account) GetMediaDataTransferRequests() (resp []datatypes.Account_Media_Data_Transfer_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getMediaDataTransferRequests", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated Message Queue accounts.
func (r Account) GetMessageQueueAccounts() (resp []datatypes.Network_Message_Queue, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getMessageQueueAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated monthly bare metal server objects.
func (r Account) GetMonthlyBareMetalInstances() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getMonthlyBareMetalInstances", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated monthly virtual guest objects.
func (r Account) GetMonthlyVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getMonthlyVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated NAS storage volumes.
func (r Account) GetNasNetworkStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNasNetworkStorage", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not this account can define their own networks.
func (r Account) GetNetworkCreationFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkCreationFlag", nil, &r.Options, &resp)
	return
}

// Retrieve All network gateway devices on this account.
func (r Account) GetNetworkGateways() (resp []datatypes.Network_Gateway, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkGateways", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated network hardware.
func (r Account) GetNetworkHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkHardware", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetNetworkMessageDeliveryAccounts() (resp []datatypes.Network_Message_Delivery, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkMessageDeliveryAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve Hardware which is currently experiencing a service failure.
func (r Account) GetNetworkMonitorDownHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkMonitorDownHardware", nil, &r.Options, &resp)
	return
}

// Retrieve Virtual guest which is currently experiencing a service failure.
func (r Account) GetNetworkMonitorDownVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkMonitorDownVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve Hardware which is currently recovering from a service failure.
func (r Account) GetNetworkMonitorRecoveringHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkMonitorRecoveringHardware", nil, &r.Options, &resp)
	return
}

// Retrieve Virtual guest which is currently recovering from a service failure.
func (r Account) GetNetworkMonitorRecoveringVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkMonitorRecoveringVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve Hardware which is currently online.
func (r Account) GetNetworkMonitorUpHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkMonitorUpHardware", nil, &r.Options, &resp)
	return
}

// Retrieve Virtual guest which is currently online.
func (r Account) GetNetworkMonitorUpVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkMonitorUpVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated storage volumes. This includes Lockbox, NAS, EVault, and iSCSI volumes.
func (r Account) GetNetworkStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkStorage", nil, &r.Options, &resp)
	return
}

// Retrieve An account's Network Storage groups.
func (r Account) GetNetworkStorageGroups() (resp []datatypes.Network_Storage_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkStorageGroups", nil, &r.Options, &resp)
	return
}

// Retrieve IPSec network tunnels for an account.
func (r Account) GetNetworkTunnelContexts() (resp []datatypes.Network_Tunnel_Module_Context, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkTunnelContexts", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not an account has automatic private VLAN spanning enabled.
func (r Account) GetNetworkVlanSpan() (resp datatypes.Account_Network_Vlan_Span, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkVlanSpan", nil, &r.Options, &resp)
	return
}

// Retrieve All network VLANs assigned to an account.
func (r Account) GetNetworkVlans() (resp []datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNetworkVlans", nil, &r.Options, &resp)
	return
}

// Retrieve DEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers for the next billing cycle. The public inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
func (r Account) GetNextBillingPublicAllotmentHardwareBandwidthDetails() (resp []datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextBillingPublicAllotmentHardwareBandwidthDetails", nil, &r.Options, &resp)
	return
}

// Return an account's next invoice in a Microsoft excel format. The "next invoice" is what a customer will be billed on their next invoice, assuming no changes are made. Currently this does not include Bandwidth Pooling charges.
func (r Account) GetNextInvoiceExcel(documentCreateDate *datatypes.Time) (resp []byte, err error) {
	params := []interface{}{
		documentCreateDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceExcel", params, &r.Options, &resp)
	return
}

// Retrieve The pre-tax total amount exempt from incubator credit for the account's next invoice. This field is now deprecated and will soon be removed. Please update all references to instead use nextInvoiceTotalAmount
func (r Account) GetNextInvoiceIncubatorExemptTotal() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceIncubatorExemptTotal", nil, &r.Options, &resp)
	return
}

// Return an account's next invoice in PDF format. The "next invoice" is what a customer will be billed on their next invoice, assuming no changes are made. Currently this does not include Bandwidth Pooling charges.
func (r Account) GetNextInvoicePdf(documentCreateDate *datatypes.Time) (resp []byte, err error) {
	params := []interface{}{
		documentCreateDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoicePdf", params, &r.Options, &resp)
	return
}

// Return an account's next invoice detailed portion in PDF format. The "next invoice" is what a customer will be billed on their next invoice, assuming no changes are made. Currently this does not include Bandwidth Pooling charges.
func (r Account) GetNextInvoicePdfDetailed(documentCreateDate *datatypes.Time) (resp []byte, err error) {
	params := []interface{}{
		documentCreateDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoicePdfDetailed", params, &r.Options, &resp)
	return
}

// Retrieve The billing items that will be on an account's next invoice.
func (r Account) GetNextInvoiceTopLevelBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceTopLevelBillingItems", nil, &r.Options, &resp)
	return
}

// Retrieve The pre-tax total amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
func (r Account) GetNextInvoiceTotalAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceTotalAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The total one-time charge amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
func (r Account) GetNextInvoiceTotalOneTimeAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceTotalOneTimeAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The total one-time tax amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
func (r Account) GetNextInvoiceTotalOneTimeTaxAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceTotalOneTimeTaxAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The total recurring charge amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
func (r Account) GetNextInvoiceTotalRecurringAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceTotalRecurringAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The total recurring charge amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
func (r Account) GetNextInvoiceTotalRecurringAmountBeforeAccountDiscount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceTotalRecurringAmountBeforeAccountDiscount", nil, &r.Options, &resp)
	return
}

// Retrieve The total recurring tax amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
func (r Account) GetNextInvoiceTotalRecurringTaxAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceTotalRecurringTaxAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The total recurring charge amount of an account's next invoice measured in US Dollars ($USD), assuming no changes or charges occur between now and time of billing.
func (r Account) GetNextInvoiceTotalTaxableRecurringAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceTotalTaxableRecurringAmount", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetNextInvoiceZeroFeeItemCounts() (resp []datatypes.Container_Product_Item_Category_ZeroFee_Count, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNextInvoiceZeroFeeItemCounts", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetNotificationSubscribers() (resp []datatypes.Notification_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getNotificationSubscribers", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Account object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Account service. You can only retrieve the account that your portal user is assigned to.
func (r Account) GetObject() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The open abuse tickets associated with an account.
func (r Account) GetOpenAbuseTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenAbuseTickets", nil, &r.Options, &resp)
	return
}

// Retrieve The open accounting tickets associated with an account.
func (r Account) GetOpenAccountingTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenAccountingTickets", nil, &r.Options, &resp)
	return
}

// Retrieve The open billing tickets associated with an account.
func (r Account) GetOpenBillingTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenBillingTickets", nil, &r.Options, &resp)
	return
}

// Retrieve An open ticket requesting cancellation of this server, if one exists.
func (r Account) GetOpenCancellationRequests() (resp []datatypes.Billing_Item_Cancellation_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenCancellationRequests", nil, &r.Options, &resp)
	return
}

// Retrieve The open tickets that do not belong to the abuse, accounting, sales, or support groups associated with an account.
func (r Account) GetOpenOtherTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenOtherTickets", nil, &r.Options, &resp)
	return
}

// Retrieve An account's recurring invoices.
func (r Account) GetOpenRecurringInvoices() (resp []datatypes.Billing_Invoice, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenRecurringInvoices", nil, &r.Options, &resp)
	return
}

// Retrieve The open sales tickets associated with an account.
func (r Account) GetOpenSalesTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenSalesTickets", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetOpenStackAccountLinks() (resp []datatypes.Account_Link, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenStackAccountLinks", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated Openstack related Object Storage accounts.
func (r Account) GetOpenStackObjectStorage() (resp []datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenStackObjectStorage", nil, &r.Options, &resp)
	return
}

// Retrieve The open support tickets associated with an account.
func (r Account) GetOpenSupportTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenSupportTickets", nil, &r.Options, &resp)
	return
}

// Retrieve All open tickets associated with an account.
func (r Account) GetOpenTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenTickets", nil, &r.Options, &resp)
	return
}

// Retrieve All open tickets associated with an account last edited by an employee.
func (r Account) GetOpenTicketsWaitingOnCustomer() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOpenTicketsWaitingOnCustomer", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated billing orders excluding upgrades.
func (r Account) GetOrders() (resp []datatypes.Billing_Order, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOrders", nil, &r.Options, &resp)
	return
}

// Retrieve The billing items that have no parent billing item. These are items that don't necessarily belong to a single server.
func (r Account) GetOrphanBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOrphanBillingItems", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetOwnedBrands() (resp []datatypes.Brand, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOwnedBrands", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetOwnedHardwareGenericComponentModels() (resp []datatypes.Hardware_Component_Model_Generic, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getOwnedHardwareGenericComponentModels", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetPaymentProcessors() (resp []datatypes.Billing_Payment_Processor, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPaymentProcessors", nil, &r.Options, &resp)
	return
}

// Before being approved for general use, a credit card must be approved by a SoftLayer agent. Once a credit card change request has been either approved or denied, the change request will no longer appear in the list of pending change requests. This method will return a list of all pending change requests as well as a portion of the data from the original request.
func (r Account) GetPendingCreditCardChangeRequestData() (resp []datatypes.Container_Account_Payment_Method_CreditCard, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingCreditCardChangeRequestData", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetPendingEvents() (resp []datatypes.Notification_Occurrence_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingEvents", nil, &r.Options, &resp)
	return
}

// Retrieve An account's latest open (pending) invoice.
func (r Account) GetPendingInvoice() (resp datatypes.Billing_Invoice, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingInvoice", nil, &r.Options, &resp)
	return
}

// Retrieve A list of top-level invoice items that are on an account's currently pending invoice.
func (r Account) GetPendingInvoiceTopLevelItems() (resp []datatypes.Billing_Invoice_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingInvoiceTopLevelItems", nil, &r.Options, &resp)
	return
}

// Retrieve The total amount of an account's pending invoice, if one exists.
func (r Account) GetPendingInvoiceTotalAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingInvoiceTotalAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The total one-time charges for an account's pending invoice, if one exists. In other words, it is the sum of one-time charges, setup fees, and labor fees. It does not include taxes.
func (r Account) GetPendingInvoiceTotalOneTimeAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingInvoiceTotalOneTimeAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The sum of all the taxes related to one time charges for an account's pending invoice, if one exists.
func (r Account) GetPendingInvoiceTotalOneTimeTaxAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingInvoiceTotalOneTimeTaxAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The total recurring amount of an account's pending invoice, if one exists.
func (r Account) GetPendingInvoiceTotalRecurringAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingInvoiceTotalRecurringAmount", nil, &r.Options, &resp)
	return
}

// Retrieve The total amount of the recurring taxes on an account's pending invoice, if one exists.
func (r Account) GetPendingInvoiceTotalRecurringTaxAmount() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPendingInvoiceTotalRecurringTaxAmount", nil, &r.Options, &resp)
	return
}

// Retrieve An account's permission groups.
func (r Account) GetPermissionGroups() (resp []datatypes.User_Permission_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPermissionGroups", nil, &r.Options, &resp)
	return
}

// Retrieve An account's user roles.
func (r Account) GetPermissionRoles() (resp []datatypes.User_Permission_Role, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPermissionRoles", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetPortableStorageVolumes() (resp []datatypes.Virtual_Disk_Image, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPortableStorageVolumes", nil, &r.Options, &resp)
	return
}

// Retrieve Customer specified URIs that are downloaded onto a newly provisioned or reloaded server. If the URI is sent over https it will be executed directly on the server.
func (r Account) GetPostProvisioningHooks() (resp []datatypes.Provisioning_Hook, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPostProvisioningHooks", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated portal users with PPTP VPN access.
func (r Account) GetPptpVpnUsers() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPptpVpnUsers", nil, &r.Options, &resp)
	return
}

// Retrieve The total recurring amount for an accounts previous revenue.
func (r Account) GetPreviousRecurringRevenue() (resp datatypes.Float64, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPreviousRecurringRevenue", nil, &r.Options, &resp)
	return
}

// Retrieve The item price that an account is restricted to.
func (r Account) GetPriceRestrictions() (resp []datatypes.Product_Item_Price_Account_Restriction, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPriceRestrictions", nil, &r.Options, &resp)
	return
}

// Retrieve All priority one tickets associated with an account.
func (r Account) GetPriorityOneTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPriorityOneTickets", nil, &r.Options, &resp)
	return
}

// Retrieve DEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers. The private inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
func (r Account) GetPrivateAllotmentHardwareBandwidthDetails() (resp []datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPrivateAllotmentHardwareBandwidthDetails", nil, &r.Options, &resp)
	return
}

// Retrieve Private and shared template group objects (parent only) for an account.
func (r Account) GetPrivateBlockDeviceTemplateGroups() (resp []datatypes.Virtual_Guest_Block_Device_Template_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPrivateBlockDeviceTemplateGroups", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetPrivateIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPrivateIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The private network VLANs assigned to an account.
func (r Account) GetPrivateNetworkVlans() (resp []datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPrivateNetworkVlans", nil, &r.Options, &resp)
	return
}

// Retrieve All private subnets associated with an account.
func (r Account) GetPrivateSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPrivateSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve DEPRECATED - This information can be pulled directly through tapping keys now - DEPRECATED. The allotments for this account and their servers. The public inbound and outbound bandwidth is calculated for each server in addition to the daily average network traffic since the last billing date.
func (r Account) GetPublicAllotmentHardwareBandwidthDetails() (resp []datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPublicAllotmentHardwareBandwidthDetails", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetPublicIpAddresses() (resp []datatypes.Network_Subnet_IpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPublicIpAddresses", nil, &r.Options, &resp)
	return
}

// Retrieve The public network VLANs assigned to an account.
func (r Account) GetPublicNetworkVlans() (resp []datatypes.Network_Vlan, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPublicNetworkVlans", nil, &r.Options, &resp)
	return
}

// Retrieve All public network subnets associated with an account.
func (r Account) GetPublicSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getPublicSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve An account's quotes.
func (r Account) GetQuotes() (resp []datatypes.Billing_Order_Quote, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getQuotes", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetRecentEvents() (resp []datatypes.Notification_Occurrence_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getRecentEvents", nil, &r.Options, &resp)
	return
}

// Retrieve The Referral Partner for this account, if any.
func (r Account) GetReferralPartner() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getReferralPartner", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetReferralPartnerCommissionForecast() (resp []datatypes.Container_Referral_Partner_Commission, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getReferralPartnerCommissionForecast", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetReferralPartnerCommissionHistory() (resp []datatypes.Container_Referral_Partner_Commission, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getReferralPartnerCommissionHistory", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) GetReferralPartnerCommissionPending() (resp []datatypes.Container_Referral_Partner_Commission, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getReferralPartnerCommissionPending", nil, &r.Options, &resp)
	return
}

// Retrieve If this is a account is a referral partner, the accounts this referral partner has referred
func (r Account) GetReferredAccounts() (resp []datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getReferredAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetRegulatedWorkloads() (resp []datatypes.Legal_RegulatedWorkload, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getRegulatedWorkloads", nil, &r.Options, &resp)
	return
}

// Retrieve Remote management command requests for an account
func (r Account) GetRemoteManagementCommandRequests() (resp []datatypes.Hardware_Component_RemoteManagement_Command_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getRemoteManagementCommandRequests", nil, &r.Options, &resp)
	return
}

// Retrieve The Replication events for all Network Storage volumes on an account.
func (r Account) GetReplicationEvents() (resp []datatypes.Network_Storage_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getReplicationEvents", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated top-level resource groups.
func (r Account) GetResourceGroups() (resp []datatypes.Resource_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getResourceGroups", nil, &r.Options, &resp)
	return
}

// Retrieve All Routers that an accounts VLANs reside on
func (r Account) GetRouters() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getRouters", nil, &r.Options, &resp)
	return
}

// Retrieve An account's reverse WHOIS data. This data is used when making SWIP requests.
func (r Account) GetRwhoisData() (resp datatypes.Network_Subnet_Rwhois_Data, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getRwhoisData", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetSalesforceAccountLink() (resp datatypes.Account_Link, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSalesforceAccountLink", nil, &r.Options, &resp)
	return
}

// Retrieve The SAML configuration for this account.
func (r Account) GetSamlAuthentication() (resp datatypes.Account_Authentication_Saml, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSamlAuthentication", nil, &r.Options, &resp)
	return
}

// Retrieve All scale groups on this account.
func (r Account) GetScaleGroups() (resp []datatypes.Scale_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getScaleGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The secondary DNS records for a SoftLayer customer account.
func (r Account) GetSecondaryDomains() (resp []datatypes.Dns_Secondary, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSecondaryDomains", nil, &r.Options, &resp)
	return
}

// Retrieve Stored security certificates (ie. SSL)
func (r Account) GetSecurityCertificates() (resp []datatypes.Security_Certificate, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSecurityCertificates", nil, &r.Options, &resp)
	return
}

// Retrieve An account's vulnerability scan requests.
func (r Account) GetSecurityScanRequests() (resp []datatypes.Network_Security_Scanner_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSecurityScanRequests", nil, &r.Options, &resp)
	return
}

// Retrieve The service billing items that will be on an account's next invoice.
func (r Account) GetServiceBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getServiceBillingItems", nil, &r.Options, &resp)
	return
}

// This method returns the [[SoftLayer_Virtual_Guest_Block_Device_Template_Group]] objects that have been shared with this account
func (r Account) GetSharedBlockDeviceTemplateGroups() (resp []datatypes.Virtual_Guest_Block_Device_Template_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSharedBlockDeviceTemplateGroups", nil, &r.Options, &resp)
	return
}

// Retrieve Shipments that belong to the customer's account.
func (r Account) GetShipments() (resp []datatypes.Account_Shipment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getShipments", nil, &r.Options, &resp)
	return
}

// Retrieve Customer specified SSH keys that can be implemented onto a newly provisioned or reloaded server.
func (r Account) GetSshKeys() (resp []datatypes.Security_Ssh_Key, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSshKeys", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated portal users with SSL VPN access.
func (r Account) GetSslVpnUsers() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSslVpnUsers", nil, &r.Options, &resp)
	return
}

// Retrieve An account's virtual guest objects that are hosted on a user provisioned hypervisor.
func (r Account) GetStandardPoolVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getStandardPoolVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetSubnetRegistrationDetails() (resp []datatypes.Account_Regional_Registry_Detail, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSubnetRegistrationDetails", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetSubnetRegistrations() (resp []datatypes.Network_Subnet_Registration, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSubnetRegistrations", nil, &r.Options, &resp)
	return
}

// Retrieve All network subnets associated with an account.
func (r Account) GetSubnets() (resp []datatypes.Network_Subnet, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSubnets", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer employees that an account is assigned to.
func (r Account) GetSupportRepresentatives() (resp []datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSupportRepresentatives", nil, &r.Options, &resp)
	return
}

// Retrieve The active support subscriptions for this account.
func (r Account) GetSupportSubscriptions() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSupportSubscriptions", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetSupportTier() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSupportTier", nil, &r.Options, &resp)
	return
}

// Retrieve A flag indicating to suppress invoices.
func (r Account) GetSuppressInvoicesFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getSuppressInvoicesFlag", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account) GetTags() (resp []datatypes.Tag, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getTags", nil, &r.Options, &resp)
	return
}

// This method will return a SoftLayer_Container_Account_Discount_Program object containing the Technology Incubator Program information for this account. To be considered an active participant, the account must have an enrollment record with a monthly credit amount set and the current date must be within the range defined by the enrollment and graduation date. The forNextBillCycle parameter can be set to true to return a SoftLayer_Container_Account_Discount_Program object with information with relation to the next bill cycle. The forNextBillCycle parameter defaults to false.
func (r Account) GetTechIncubatorProgramInfo(forNextBillCycle *bool) (resp datatypes.Container_Account_Discount_Program, err error) {
	params := []interface{}{
		forNextBillCycle,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "getTechIncubatorProgramInfo", params, &r.Options, &resp)
	return
}

// Returns multiple [[SoftLayer_Container_Policy_Acceptance]] that represent the acceptance status of the applicable third-party policies for this account.
func (r Account) GetThirdPartyPoliciesAcceptanceStatus() (resp []datatypes.Container_Policy_Acceptance, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getThirdPartyPoliciesAcceptanceStatus", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated tickets.
func (r Account) GetTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getTickets", nil, &r.Options, &resp)
	return
}

// Retrieve Tickets closed within the last 72 hours or last 10 tickets, whichever is less, associated with an account.
func (r Account) GetTicketsClosedInTheLastThreeDays() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getTicketsClosedInTheLastThreeDays", nil, &r.Options, &resp)
	return
}

// Retrieve Tickets closed today associated with an account.
func (r Account) GetTicketsClosedToday() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getTicketsClosedToday", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated Transcode account.
func (r Account) GetTranscodeAccounts() (resp []datatypes.Network_Media_Transcode_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getTranscodeAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated upgrade requests.
func (r Account) GetUpgradeRequests() (resp []datatypes.Product_Upgrade_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getUpgradeRequests", nil, &r.Options, &resp)
	return
}

// Retrieve An account's portal users.
func (r Account) GetUsers() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getUsers", nil, &r.Options, &resp)
	return
}

// Retrieve a list of valid (non-expired) security certificates without the sensitive certificate information. This allows non-privileged users to view and select security certificates when configuring associated services.
func (r Account) GetValidSecurityCertificateEntries() (resp []datatypes.Security_Certificate_Entry, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getValidSecurityCertificateEntries", nil, &r.Options, &resp)
	return
}

// Retrieve Stored security certificates that are not expired (ie. SSL)
func (r Account) GetValidSecurityCertificates() (resp []datatypes.Security_Certificate, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getValidSecurityCertificates", nil, &r.Options, &resp)
	return
}

// Retrieve Return 0 if vpn updates are currently in progress on this account otherwise 1.
func (r Account) GetVdrUpdatesInProgressFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVdrUpdatesInProgressFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The bandwidth pooling for this account.
func (r Account) GetVirtualDedicatedRacks() (resp []datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualDedicatedRacks", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated virtual server virtual disk images.
func (r Account) GetVirtualDiskImages() (resp []datatypes.Virtual_Disk_Image, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualDiskImages", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated virtual guest objects.
func (r Account) GetVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated virtual guest objects currently over bandwidth allocation.
func (r Account) GetVirtualGuestsOverBandwidthAllocation() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsOverBandwidthAllocation", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated virtual guest objects currently over bandwidth allocation.
func (r Account) GetVirtualGuestsProjectedOverBandwidthAllocation() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsProjectedOverBandwidthAllocation", nil, &r.Options, &resp)
	return
}

// Retrieve All virtual guests associated with an account that has the cPanel web hosting control panel installed.
func (r Account) GetVirtualGuestsWithCpanel() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsWithCpanel", nil, &r.Options, &resp)
	return
}

// Retrieve All virtual guests associated with an account that have McAfee Secure software components.
func (r Account) GetVirtualGuestsWithMcafee() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsWithMcafee", nil, &r.Options, &resp)
	return
}

// Retrieve All virtual guests associated with an account that have McAfee Secure AntiVirus for Redhat software components.
func (r Account) GetVirtualGuestsWithMcafeeAntivirusRedhat() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsWithMcafeeAntivirusRedhat", nil, &r.Options, &resp)
	return
}

// Retrieve All virtual guests associated with an account that has McAfee Secure AntiVirus for Windows software components.
func (r Account) GetVirtualGuestsWithMcafeeAntivirusWindows() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsWithMcafeeAntivirusWindows", nil, &r.Options, &resp)
	return
}

// Retrieve All virtual guests associated with an account that has McAfee Secure Intrusion Detection System software components.
func (r Account) GetVirtualGuestsWithMcafeeIntrusionDetectionSystem() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsWithMcafeeIntrusionDetectionSystem", nil, &r.Options, &resp)
	return
}

// Retrieve All virtual guests associated with an account that has the Plesk web hosting control panel installed.
func (r Account) GetVirtualGuestsWithPlesk() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsWithPlesk", nil, &r.Options, &resp)
	return
}

// Retrieve All virtual guests associated with an account that have the QuantaStor storage system installed.
func (r Account) GetVirtualGuestsWithQuantastor() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsWithQuantastor", nil, &r.Options, &resp)
	return
}

// Retrieve All virtual guests associated with an account that has the Urchin web traffic analytics package installed.
func (r Account) GetVirtualGuestsWithUrchin() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualGuestsWithUrchin", nil, &r.Options, &resp)
	return
}

// Retrieve The bandwidth pooling for this account.
func (r Account) GetVirtualPrivateRack() (resp datatypes.Network_Bandwidth_Version1_Allotment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualPrivateRack", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated virtual server archived storage repositories.
func (r Account) GetVirtualStorageArchiveRepositories() (resp []datatypes.Virtual_Storage_Repository, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualStorageArchiveRepositories", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated virtual server public storage repositories.
func (r Account) GetVirtualStoragePublicRepositories() (resp []datatypes.Virtual_Storage_Repository, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVirtualStoragePublicRepositories", nil, &r.Options, &resp)
	return
}

// This returns a collection of active VMware software account license keys.
func (r Account) GetVmWareActiveAccountLicenseKeys() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getVmWareActiveAccountLicenseKeys", nil, &r.Options, &resp)
	return
}

// Retrieve a list of an account's hardware's Windows Update status. This list includes which servers have available updates, which servers require rebooting due to updates, which servers have failed retrieving updates, and which servers have failed to communicate with the SoftLayer private Windows Software Update Services server.
func (r Account) GetWindowsUpdateStatus() (resp []datatypes.Container_Utility_Microsoft_Windows_UpdateServices_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "getWindowsUpdateStatus", nil, &r.Options, &resp)
	return
}

// Determine if an account has an [[SoftLayer_Account_Attribute|attribute]] associated with it. hasAttribute() returns false if the attribute does not exist or if it does not have a value.
func (r Account) HasAttribute(attributeType *string) (resp bool, err error) {
	params := []interface{}{
		attributeType,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "hasAttribute", params, &r.Options, &resp)
	return
}

// This method will return the limit (number) of hourly services the account is allowed to have.
func (r Account) HourlyInstanceLimit() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "hourlyInstanceLimit", nil, &r.Options, &resp)
	return
}

// This method will return the limit (number) of hourly bare metal servers the account is allowed to have.
func (r Account) HourlyServerLimit() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "hourlyServerLimit", nil, &r.Options, &resp)
	return
}

// This method will link this SoftLayer account with the provided external account.
func (r Account) LinkExternalAccount(externalAccountId *string, authorizationToken *string, externalServiceProviderKey *string) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		externalAccountId,
		authorizationToken,
		externalServiceProviderKey,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "linkExternalAccount", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) RemoveAlternateCreditCard() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "removeAlternateCreditCard", nil, &r.Options, &resp)
	return
}

// Retrieve the record data associated with the submission of a Credit Card Change Request. Softlayer customers are permitted to request a change in Credit Card information. Part of the process calls for an attempt by SoftLayer to submit at $1.00 charge to the financial institution backing the credit card as a means of verifying that the information provided in the change request is valid.  The data associated with this change request returned to the calling function.
//
// If the onlyChangeNicknameFlag parameter is set to true, the nickname of the credit card will be changed immediately without requiring approval by an agent.  To change the nickname of the active payment method, pass the empty string for paymentRoleName.  To change the nickname for the alternate credit card, pass ALTERNATE_CREDIT_CARD as the paymentRoleName.  vatId must be set, but the value will not be used and the empty string is acceptable.
func (r Account) RequestCreditCardChange(request *datatypes.Billing_Payment_Card_ChangeRequest, vatId *string, paymentRoleName *string, onlyChangeNicknameFlag *bool) (resp datatypes.Billing_Payment_Card_ChangeRequest, err error) {
	params := []interface{}{
		request,
		vatId,
		paymentRoleName,
		onlyChangeNicknameFlag,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "requestCreditCardChange", params, &r.Options, &resp)
	return
}

// Retrieve the record data associated with the submission of a Manual Payment Request. Softlayer customers are permitted to request a manual one-time payment at a minimum amount of $2.00. Customers may submit a Credit Card Payment (Mastercard, Visa, American Express) or a PayPal payment. For Credit Card Payments, SoftLayer engages the credit card financial institution to submit the payment request.  The financial institution's response and other data associated with the transaction are returned to the calling function.  In the case of PayPal Payments, SoftLayer engages the PayPal system to initiate the PayPal payment sequence.  The applicable data generated during the request is returned to the calling function.
func (r Account) RequestManualPayment(request *datatypes.Billing_Payment_Card_ManualPayment) (resp datatypes.Billing_Payment_Card_ManualPayment, err error) {
	params := []interface{}{
		request,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "requestManualPayment", params, &r.Options, &resp)
	return
}

// Retrieve the record data associated with the submission of a Manual Payment Request for a manual payment using a credit card which is on file and does not require an approval process.  Softlayer customers are permitted to request a manual one-time payment at a minimum amount of $2.00.  Customers may use an existing Credit Card on file (Mastercard, Visa, American Express).  SoftLayer engages the credit card financial institution to submit the payment request.  The financial institution's response and other data associated with the transaction are returned to the calling function.  The applicable data generated during the request is returned to the calling function.
func (r Account) RequestManualPaymentUsingCreditCardOnFile(amount *string, payWithAlternateCardFlag *bool, note *string) (resp datatypes.Billing_Payment_Card_ManualPayment, err error) {
	params := []interface{}{
		amount,
		payWithAlternateCardFlag,
		note,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "requestManualPaymentUsingCreditCardOnFile", params, &r.Options, &resp)
	return
}

// Set this account's abuse emails. Takes an array of email addresses as strings.
func (r Account) SetAbuseEmails(emails []string) (resp bool, err error) {
	params := []interface{}{
		emails,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "setAbuseEmails", params, &r.Options, &resp)
	return
}

// Set the flag that enables or disables automatic private network VLAN spanning for a SoftLayer customer account. Enabling VLAN spanning allows an account's servers to talk on the same broadcast domain even if they reside within different private vlans.
func (r Account) SetVlanSpan(enabled *bool) (resp bool, err error) {
	params := []interface{}{
		enabled,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "setVlanSpan", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account) SwapCreditCards() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account", "swapCreditCards", nil, &r.Options, &resp)
	return
}

// Some larger SoftLayer customer accounts may have servers and virtual servers on more subnets than SoftLayer's private network VPN devices can assign routes for. In those cases routes for individual servers and virtual servers may be assigned individually to an account's servers via this method.
//
// Always call this method to enable changes when manually configuring VPN subnet access.
func (r Account) UpdateVpnUsersForResource(objectId *int, objectType *string) (resp bool, err error) {
	params := []interface{}{
		objectId,
		objectType,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "updateVpnUsersForResource", params, &r.Options, &resp)
	return
}

// This method will validate the following account fields. Included are the allowed characters for each field. Email Address*: letters, numbers, space, period, dash, parenthesis, exclamation point, at sign, ampersand, colon, comma, underscore, apostrophe, octothorpe. Company Name*: alphabet, numbers, space, period, dash, octothorpe, forward slash, backward slash, comma, colon, at sign, ampersand, underscore, apostrophe, parenthesis, exclamation point. (Note: may not contain an email address) First Name*: alphabet, space, period, dash, comma, apostrophe. Last Name*: alphabet, space, period, dash, comma, apostrophe. Address 1*: alphabet, numbers, space, period, dash, octothorpe, forward slash, backward slash, comma, colon, at sign, ampersand, underscore, apostrophe. Address 2: alphabet, numbers, space, period, dash, octothorpe, forward slash, backward slash, comma, colon, at sign, ampersand, underscore, apostrophe. City*: alphabet, space, period, dash, apostrophe. State*: Required if country is US or Canada. Must be valid two-letter state code for that country. Postal Code*: alphabet, numbers, dash, space. Country*: alphabet, numbers. Office Phone*: alphabet, numbers, space, period, dash, parenthesis, plus sign. Alternate Phone: alphabet, numbers, space, period, dash, parenthesis, plus sign. Fax Phone: alphabet, numbers, space, period, dash, parenthesis, plus sign.
// * denotes a required field.
func (r Account) Validate(account *datatypes.Account) (resp []string, err error) {
	params := []interface{}{
		account,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "validate", params, &r.Options, &resp)
	return
}

// This method checks global and account specific requirements and returns true if the dollar amount entered is acceptable for this account and false otherwise. Please note the dollar amount is in USD.
func (r Account) ValidateManualPaymentAmount(amount *string) (resp bool, err error) {
	params := []interface{}{
		amount,
	}
	err = r.Session.DoRequest("SoftLayer_Account", "validateManualPaymentAmount", params, &r.Options, &resp)
	return
}

// The SoftLayer_Account_Address data type contains information on an address associated with a SoftLayer account.
type Account_Address struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountAddressService returns an instance of the Account_Address SoftLayer service
func GetAccountAddressService(sess *session.Session) Account_Address {
	return Account_Address{Session: sess}
}

func (r Account_Address) Id(id int) Account_Address {
	r.Options.Id = &id
	return r
}

func (r Account_Address) Mask(mask string) Account_Address {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Address) Filter(filter string) Account_Address {
	r.Options.Filter = filter
	return r
}

func (r Account_Address) Limit(limit int) Account_Address {
	r.Options.Limit = &limit
	return r
}

func (r Account_Address) Offset(offset int) Account_Address {
	r.Options.Offset = &offset
	return r
}

// Create a new address record. The ''typeId'', ''accountId'', ''description'', ''address1'', ''city'', ''state'', ''country'', and ''postalCode'' properties in the templateObject parameter are required properties and may not be null or empty. Users will be restricted to creating addresses for their account.
func (r Account_Address) CreateObject(templateObject *datatypes.Account_Address) (resp datatypes.Account_Address, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Address", "createObject", params, &r.Options, &resp)
	return
}

// Edit the properties of an address record by passing in a modified instance of a SoftLayer_Account_Address object. Users will be restricted to modifying addresses for their account.
func (r Account_Address) EditObject(templateObject *datatypes.Account_Address) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Address", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account to which this address belongs.
func (r Account_Address) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve a list of SoftLayer datacenter addresses.
func (r Account_Address) GetAllDataCenters() (resp []datatypes.Account_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getAllDataCenters", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who created this address.
func (r Account_Address) GetCreateUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getCreateUser", nil, &r.Options, &resp)
	return
}

// Retrieve The location of this address.
func (r Account_Address) GetLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getLocation", nil, &r.Options, &resp)
	return
}

// Retrieve The employee who last modified this address.
func (r Account_Address) GetModifyEmployee() (resp datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getModifyEmployee", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who last modified this address.
func (r Account_Address) GetModifyUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getModifyUser", nil, &r.Options, &resp)
	return
}

// Retrieve a list of SoftLayer datacenter addresses.
func (r Account_Address) GetNetworkAddress(name *string) (resp []datatypes.Account_Address, err error) {
	params := []interface{}{
		name,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getNetworkAddress", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Address) GetObject() (resp datatypes.Account_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve An account address' type.
func (r Account_Address) GetType() (resp datatypes.Account_Address_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address", "getType", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Address_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountAddressTypeService returns an instance of the Account_Address_Type SoftLayer service
func GetAccountAddressTypeService(sess *session.Session) Account_Address_Type {
	return Account_Address_Type{Session: sess}
}

func (r Account_Address_Type) Id(id int) Account_Address_Type {
	r.Options.Id = &id
	return r
}

func (r Account_Address_Type) Mask(mask string) Account_Address_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Address_Type) Filter(filter string) Account_Address_Type {
	r.Options.Filter = filter
	return r
}

func (r Account_Address_Type) Limit(limit int) Account_Address_Type {
	r.Options.Limit = &limit
	return r
}

func (r Account_Address_Type) Offset(offset int) Account_Address_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Address_Type) GetObject() (resp datatypes.Account_Address_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Address_Type", "getObject", nil, &r.Options, &resp)
	return
}

// This service allows for a unique identifier to be associated to an existing customer account.
type Account_Affiliation struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountAffiliationService returns an instance of the Account_Affiliation SoftLayer service
func GetAccountAffiliationService(sess *session.Session) Account_Affiliation {
	return Account_Affiliation{Session: sess}
}

func (r Account_Affiliation) Id(id int) Account_Affiliation {
	r.Options.Id = &id
	return r
}

func (r Account_Affiliation) Mask(mask string) Account_Affiliation {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Affiliation) Filter(filter string) Account_Affiliation {
	r.Options.Filter = filter
	return r
}

func (r Account_Affiliation) Limit(limit int) Account_Affiliation {
	r.Options.Limit = &limit
	return r
}

func (r Account_Affiliation) Offset(offset int) Account_Affiliation {
	r.Options.Offset = &offset
	return r
}

// Create a new affiliation to associate with an existing account.
func (r Account_Affiliation) CreateObject(templateObject *datatypes.Account_Affiliation) (resp datatypes.Account_Affiliation, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Affiliation", "createObject", params, &r.Options, &resp)
	return
}

// deleteObject permanently removes an account affiliation
func (r Account_Affiliation) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Affiliation", "deleteObject", nil, &r.Options, &resp)
	return
}

// Edit an affiliation that is associated to an existing account.
func (r Account_Affiliation) EditObject(templateObject *datatypes.Account_Affiliation) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Affiliation", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account that an affiliation belongs to.
func (r Account_Affiliation) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Affiliation", "getAccount", nil, &r.Options, &resp)
	return
}

// Get account affiliation information associated with affiliate id.
func (r Account_Affiliation) GetAccountAffiliationsByAffiliateId(affiliateId *string) (resp []datatypes.Account_Affiliation, err error) {
	params := []interface{}{
		affiliateId,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Affiliation", "getAccountAffiliationsByAffiliateId", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Affiliation) GetObject() (resp datatypes.Account_Affiliation, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Affiliation", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Agreement struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountAgreementService returns an instance of the Account_Agreement SoftLayer service
func GetAccountAgreementService(sess *session.Session) Account_Agreement {
	return Account_Agreement{Session: sess}
}

func (r Account_Agreement) Id(id int) Account_Agreement {
	r.Options.Id = &id
	return r
}

func (r Account_Agreement) Mask(mask string) Account_Agreement {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Agreement) Filter(filter string) Account_Agreement {
	r.Options.Filter = filter
	return r
}

func (r Account_Agreement) Limit(limit int) Account_Agreement {
	r.Options.Limit = &limit
	return r
}

func (r Account_Agreement) Offset(offset int) Account_Agreement {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Account_Agreement) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Agreement", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The type of agreement.
func (r Account_Agreement) GetAgreementType() (resp datatypes.Account_Agreement_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Agreement", "getAgreementType", nil, &r.Options, &resp)
	return
}

// Retrieve The files attached to an agreement.
func (r Account_Agreement) GetAttachedBillingAgreementFiles() (resp []datatypes.Account_MasterServiceAgreement, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Agreement", "getAttachedBillingAgreementFiles", nil, &r.Options, &resp)
	return
}

// Retrieve The billing items associated with an agreement.
func (r Account_Agreement) GetBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Agreement", "getBillingItems", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Agreement) GetObject() (resp datatypes.Account_Agreement, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Agreement", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The status of the agreement.
func (r Account_Agreement) GetStatus() (resp datatypes.Account_Agreement_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Agreement", "getStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The top level billing item associated with an agreement.
func (r Account_Agreement) GetTopLevelBillingItems() (resp []datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Agreement", "getTopLevelBillingItems", nil, &r.Options, &resp)
	return
}

// Account authentication has many different settings that can be set. This class allows the customer or employee to set these settigns.
type Account_Authentication_Attribute struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountAuthenticationAttributeService returns an instance of the Account_Authentication_Attribute SoftLayer service
func GetAccountAuthenticationAttributeService(sess *session.Session) Account_Authentication_Attribute {
	return Account_Authentication_Attribute{Session: sess}
}

func (r Account_Authentication_Attribute) Id(id int) Account_Authentication_Attribute {
	r.Options.Id = &id
	return r
}

func (r Account_Authentication_Attribute) Mask(mask string) Account_Authentication_Attribute {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Authentication_Attribute) Filter(filter string) Account_Authentication_Attribute {
	r.Options.Filter = filter
	return r
}

func (r Account_Authentication_Attribute) Limit(limit int) Account_Authentication_Attribute {
	r.Options.Limit = &limit
	return r
}

func (r Account_Authentication_Attribute) Offset(offset int) Account_Authentication_Attribute {
	r.Options.Offset = &offset
	return r
}

// Retrieve The SoftLayer customer account.
func (r Account_Authentication_Attribute) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Attribute", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer account authentication that has an attribute.
func (r Account_Authentication_Attribute) GetAuthenticationRecord() (resp datatypes.Account_Authentication_Saml, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Attribute", "getAuthenticationRecord", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Authentication_Attribute) GetObject() (resp datatypes.Account_Authentication_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Attribute", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The type of attribute assigned to a SoftLayer account authentication.
func (r Account_Authentication_Attribute) GetType() (resp datatypes.Account_Authentication_Attribute_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Attribute", "getType", nil, &r.Options, &resp)
	return
}

// SoftLayer_Account_Authentication_Attribute_Type models the type of attribute that can be assigned to a SoftLayer customer account authentication.
type Account_Authentication_Attribute_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountAuthenticationAttributeTypeService returns an instance of the Account_Authentication_Attribute_Type SoftLayer service
func GetAccountAuthenticationAttributeTypeService(sess *session.Session) Account_Authentication_Attribute_Type {
	return Account_Authentication_Attribute_Type{Session: sess}
}

func (r Account_Authentication_Attribute_Type) Id(id int) Account_Authentication_Attribute_Type {
	r.Options.Id = &id
	return r
}

func (r Account_Authentication_Attribute_Type) Mask(mask string) Account_Authentication_Attribute_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Authentication_Attribute_Type) Filter(filter string) Account_Authentication_Attribute_Type {
	r.Options.Filter = filter
	return r
}

func (r Account_Authentication_Attribute_Type) Limit(limit int) Account_Authentication_Attribute_Type {
	r.Options.Limit = &limit
	return r
}

func (r Account_Authentication_Attribute_Type) Offset(offset int) Account_Authentication_Attribute_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Authentication_Attribute_Type) GetAllObjects() (resp []datatypes.Account_Attribute_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Attribute_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Authentication_Attribute_Type) GetObject() (resp datatypes.Account_Authentication_Attribute_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Attribute_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Authentication_Saml struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountAuthenticationSamlService returns an instance of the Account_Authentication_Saml SoftLayer service
func GetAccountAuthenticationSamlService(sess *session.Session) Account_Authentication_Saml {
	return Account_Authentication_Saml{Session: sess}
}

func (r Account_Authentication_Saml) Id(id int) Account_Authentication_Saml {
	r.Options.Id = &id
	return r
}

func (r Account_Authentication_Saml) Mask(mask string) Account_Authentication_Saml {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Authentication_Saml) Filter(filter string) Account_Authentication_Saml {
	r.Options.Filter = filter
	return r
}

func (r Account_Authentication_Saml) Limit(limit int) Account_Authentication_Saml {
	r.Options.Limit = &limit
	return r
}

func (r Account_Authentication_Saml) Offset(offset int) Account_Authentication_Saml {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Authentication_Saml) CreateObject(templateObject *datatypes.Account_Authentication_Saml) (resp datatypes.Account_Authentication_Saml, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Saml", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Authentication_Saml) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Saml", "deleteObject", nil, &r.Options, &resp)
	return
}

// Edit the object by passing in a modified instance of the object
func (r Account_Authentication_Saml) EditObject(templateObject *datatypes.Account_Authentication_Saml) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Saml", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account associated with this saml configuration.
func (r Account_Authentication_Saml) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Saml", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The saml attribute values for a SoftLayer customer account.
func (r Account_Authentication_Saml) GetAttributes() (resp []datatypes.Account_Authentication_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Saml", "getAttributes", nil, &r.Options, &resp)
	return
}

// This method will return the service provider metadata in XML format.
func (r Account_Authentication_Saml) GetMetadata() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Saml", "getMetadata", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Authentication_Saml) GetObject() (resp datatypes.Account_Authentication_Saml, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Authentication_Saml", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Contact struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountContactService returns an instance of the Account_Contact SoftLayer service
func GetAccountContactService(sess *session.Session) Account_Contact {
	return Account_Contact{Session: sess}
}

func (r Account_Contact) Id(id int) Account_Contact {
	r.Options.Id = &id
	return r
}

func (r Account_Contact) Mask(mask string) Account_Contact {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Contact) Filter(filter string) Account_Contact {
	r.Options.Filter = filter
	return r
}

func (r Account_Contact) Limit(limit int) Account_Contact {
	r.Options.Limit = &limit
	return r
}

func (r Account_Contact) Offset(offset int) Account_Contact {
	r.Options.Offset = &offset
	return r
}

// This method creates an account contact. The accountId is fixed, other properties can be set during creation. The typeId indicates the SoftLayer_Account_Contact_Type for the contact. This method returns the SoftLayer_Account_Contact object that is created.
func (r Account_Contact) CreateObject(templateObject *datatypes.Account_Contact) (resp datatypes.Account_Contact, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Contact", "createObject", params, &r.Options, &resp)
	return
}

// deleteObject permanently removes an account contact
func (r Account_Contact) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Contact", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method allows you to modify an account contact. Only master users are permitted to modify an account contact.
func (r Account_Contact) EditObject(templateObject *datatypes.Account_Contact) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Contact", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Contact) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Contact", "getAccount", nil, &r.Options, &resp)
	return
}

// This method will return an array of SoftLayer_Account_Contact_Type objects which can be used when creating or editing an account contact.
func (r Account_Contact) GetAllContactTypes() (resp []datatypes.Account_Contact_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Contact", "getAllContactTypes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Contact) GetObject() (resp datatypes.Account_Contact, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Contact", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Contact) GetType() (resp datatypes.Account_Contact_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Contact", "getType", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Historical_Report struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountHistoricalReportService returns an instance of the Account_Historical_Report SoftLayer service
func GetAccountHistoricalReportService(sess *session.Session) Account_Historical_Report {
	return Account_Historical_Report{Session: sess}
}

func (r Account_Historical_Report) Id(id int) Account_Historical_Report {
	r.Options.Id = &id
	return r
}

func (r Account_Historical_Report) Mask(mask string) Account_Historical_Report {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Historical_Report) Filter(filter string) Account_Historical_Report {
	r.Options.Filter = filter
	return r
}

func (r Account_Historical_Report) Limit(limit int) Account_Historical_Report {
	r.Options.Limit = &limit
	return r
}

func (r Account_Historical_Report) Offset(offset int) Account_Historical_Report {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Historical_Report) GetAccountHostUptimeGraphData(startDate *string, endDate *string) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Historical_Report", "getAccountHostUptimeGraphData", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Historical_Report) GetAccountHostUptimeSummary(startDateTime *string, endDateTime *string) (resp datatypes.Container_Account_Historical_Summary, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Historical_Report", "getAccountHostUptimeSummary", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Historical_Report) GetAccountUrlUptimeGraphData(startDate *string, endDate *string) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Historical_Report", "getAccountUrlUptimeGraphData", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Historical_Report) GetAccountUrlUptimeSummary(startDateTime *string, endDateTime *string) (resp datatypes.Container_Account_Historical_Summary, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Historical_Report", "getAccountUrlUptimeSummary", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Historical_Report) GetHostUptimeDetail(configurationValueId *int, startDateTime *string, endDateTime *string) (resp datatypes.Container_Account_Historical_Summary_Detail, err error) {
	params := []interface{}{
		configurationValueId,
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Historical_Report", "getHostUptimeDetail", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Historical_Report) GetHostUptimeGraphData(configurationValueId *int, startDate *string, endDate *string) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		configurationValueId,
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Historical_Report", "getHostUptimeGraphData", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Historical_Report) GetUrlUptimeDetail(configurationValueId *int, startDateTime *string, endDateTime *string) (resp datatypes.Container_Account_Historical_Summary_Detail, err error) {
	params := []interface{}{
		configurationValueId,
		startDateTime,
		endDateTime,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Historical_Report", "getUrlUptimeDetail", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Historical_Report) GetUrlUptimeGraphData(configurationValueId *int, startDate *string, endDate *string) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		configurationValueId,
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Historical_Report", "getUrlUptimeGraphData", params, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Link_Bluemix struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountLinkBluemixService returns an instance of the Account_Link_Bluemix SoftLayer service
func GetAccountLinkBluemixService(sess *session.Session) Account_Link_Bluemix {
	return Account_Link_Bluemix{Session: sess}
}

func (r Account_Link_Bluemix) Id(id int) Account_Link_Bluemix {
	r.Options.Id = &id
	return r
}

func (r Account_Link_Bluemix) Mask(mask string) Account_Link_Bluemix {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Link_Bluemix) Filter(filter string) Account_Link_Bluemix {
	r.Options.Filter = filter
	return r
}

func (r Account_Link_Bluemix) Limit(limit int) Account_Link_Bluemix {
	r.Options.Limit = &limit
	return r
}

func (r Account_Link_Bluemix) Offset(offset int) Account_Link_Bluemix {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Link_Bluemix) GetObject() (resp datatypes.Account_Link_Bluemix, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Link_Bluemix", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Link_Bluemix) GetSupportTierType() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Link_Bluemix", "getSupportTierType", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Link_OpenStack struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountLinkOpenStackService returns an instance of the Account_Link_OpenStack SoftLayer service
func GetAccountLinkOpenStackService(sess *session.Session) Account_Link_OpenStack {
	return Account_Link_OpenStack{Session: sess}
}

func (r Account_Link_OpenStack) Id(id int) Account_Link_OpenStack {
	r.Options.Id = &id
	return r
}

func (r Account_Link_OpenStack) Mask(mask string) Account_Link_OpenStack {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Link_OpenStack) Filter(filter string) Account_Link_OpenStack {
	r.Options.Filter = filter
	return r
}

func (r Account_Link_OpenStack) Limit(limit int) Account_Link_OpenStack {
	r.Options.Limit = &limit
	return r
}

func (r Account_Link_OpenStack) Offset(offset int) Account_Link_OpenStack {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Link_OpenStack) CreateOSDomain(request *datatypes.Account_Link_OpenStack_LinkRequest) (resp datatypes.Account_Link_OpenStack_DomainCreationDetails, err error) {
	params := []interface{}{
		request,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Link_OpenStack", "createOSDomain", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Link_OpenStack) CreateOSProject(request *datatypes.Account_Link_OpenStack_LinkRequest) (resp datatypes.Account_Link_OpenStack_ProjectCreationDetails, err error) {
	params := []interface{}{
		request,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Link_OpenStack", "createOSProject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Link_OpenStack) DeleteOSDomain(domainId *string) (resp bool, err error) {
	params := []interface{}{
		domainId,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Link_OpenStack", "deleteOSDomain", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Link_OpenStack) DeleteOSProject(projectId *string) (resp bool, err error) {
	params := []interface{}{
		projectId,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Link_OpenStack", "deleteOSProject", params, &r.Options, &resp)
	return
}

// deleteObject permanently removes an account link and all of it's associated keystone data (including users for the associated project). '''This cannot be undone.''' Be wary of running this method. If you remove an account link in error you will need to re-create it by creating a new SoftLayer_Account_Link_OpenStack object.
func (r Account_Link_OpenStack) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Link_OpenStack", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Link_OpenStack) GetOSProject(projectId *string) (resp datatypes.Account_Link_OpenStack_ProjectDetails, err error) {
	params := []interface{}{
		projectId,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Link_OpenStack", "getOSProject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Link_OpenStack) GetObject() (resp datatypes.Account_Link_OpenStack, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Link_OpenStack", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Link_OpenStack) ListOSProjects() (resp []datatypes.Account_Link_OpenStack_ProjectDetails, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Link_OpenStack", "listOSProjects", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Account_Lockdown_Request data type holds information on API requests from brand customers.
type Account_Lockdown_Request struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountLockdownRequestService returns an instance of the Account_Lockdown_Request SoftLayer service
func GetAccountLockdownRequestService(sess *session.Session) Account_Lockdown_Request {
	return Account_Lockdown_Request{Session: sess}
}

func (r Account_Lockdown_Request) Id(id int) Account_Lockdown_Request {
	r.Options.Id = &id
	return r
}

func (r Account_Lockdown_Request) Mask(mask string) Account_Lockdown_Request {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Lockdown_Request) Filter(filter string) Account_Lockdown_Request {
	r.Options.Filter = filter
	return r
}

func (r Account_Lockdown_Request) Limit(limit int) Account_Lockdown_Request {
	r.Options.Limit = &limit
	return r
}

func (r Account_Lockdown_Request) Offset(offset int) Account_Lockdown_Request {
	r.Options.Offset = &offset
	return r
}

// Will cancel a lockdown request scheduled in the future. Once canceled, the lockdown request cannot be reconciled and new requests must be made for subsequent actions on the account.
func (r Account_Lockdown_Request) CancelRequest() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Account_Lockdown_Request", "cancelRequest", nil, &r.Options, &resp)
	return
}

// Takes the original lockdown request ID, and an optional disable date. If no date is passed with the API call, the account will be disabled immediately. Otherwise, the account will be disabled on the date given. All hardware will be reclaimed and all accounts permanently disabled.
func (r Account_Lockdown_Request) DisableLockedAccount(disableDate *string) (resp int, err error) {
	params := []interface{}{
		disableDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Lockdown_Request", "disableLockedAccount", params, &r.Options, &resp)
	return
}

// Takes an account ID and an optional disconnect date. If no disconnect date is passed into the API call, the account disconnection will happen immediately. Otherwise, the account disconnection will happen on the date given. A brand account request ID will be returned and will then be updated when the disconnection occurs.
func (r Account_Lockdown_Request) DisconnectCompute(accountId *int, disconnectDate *string) (resp int, err error) {
	params := []interface{}{
		accountId,
		disconnectDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Lockdown_Request", "disconnectCompute", params, &r.Options, &resp)
	return
}

// Provides a history of an account's lockdown requests and their status.
func (r Account_Lockdown_Request) GetAccountHistory(accountId *int) (resp []datatypes.Account_Lockdown_Request, err error) {
	params := []interface{}{
		accountId,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Lockdown_Request", "getAccountHistory", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Lockdown_Request) GetObject() (resp datatypes.Account_Lockdown_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Lockdown_Request", "getObject", nil, &r.Options, &resp)
	return
}

// Takes the original disconnected lockdown event ID, and an optional reconnect date. If no reconnect date is passed with the API call, the account reconnection will happen immediately. Otherwise, the account reconnection will happen on the date given. The associated lockdown event will be unlocked and closed at that time.
func (r Account_Lockdown_Request) ReconnectCompute(reconnectDate *string) (resp int, err error) {
	params := []interface{}{
		reconnectDate,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Lockdown_Request", "reconnectCompute", params, &r.Options, &resp)
	return
}

// no documentation yet
type Account_MasterServiceAgreement struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountMasterServiceAgreementService returns an instance of the Account_MasterServiceAgreement SoftLayer service
func GetAccountMasterServiceAgreementService(sess *session.Session) Account_MasterServiceAgreement {
	return Account_MasterServiceAgreement{Session: sess}
}

func (r Account_MasterServiceAgreement) Id(id int) Account_MasterServiceAgreement {
	r.Options.Id = &id
	return r
}

func (r Account_MasterServiceAgreement) Mask(mask string) Account_MasterServiceAgreement {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_MasterServiceAgreement) Filter(filter string) Account_MasterServiceAgreement {
	r.Options.Filter = filter
	return r
}

func (r Account_MasterServiceAgreement) Limit(limit int) Account_MasterServiceAgreement {
	r.Options.Limit = &limit
	return r
}

func (r Account_MasterServiceAgreement) Offset(offset int) Account_MasterServiceAgreement {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Account_MasterServiceAgreement) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_MasterServiceAgreement", "getAccount", nil, &r.Options, &resp)
	return
}

// Gets a File Entity container with the user's account's current MSA PDF. Gets a translation if one is available. Otherwise, gets the master document.
func (r Account_MasterServiceAgreement) GetFile() (resp datatypes.Container_Utility_File_Entity, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_MasterServiceAgreement", "getFile", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_MasterServiceAgreement) GetObject() (resp datatypes.Account_MasterServiceAgreement, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_MasterServiceAgreement", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Account_Media data type contains information on a single piece of media associated with a Data Transfer Service request.
type Account_Media struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountMediaService returns an instance of the Account_Media SoftLayer service
func GetAccountMediaService(sess *session.Session) Account_Media {
	return Account_Media{Session: sess}
}

func (r Account_Media) Id(id int) Account_Media {
	r.Options.Id = &id
	return r
}

func (r Account_Media) Mask(mask string) Account_Media {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Media) Filter(filter string) Account_Media {
	r.Options.Filter = filter
	return r
}

func (r Account_Media) Limit(limit int) Account_Media {
	r.Options.Limit = &limit
	return r
}

func (r Account_Media) Offset(offset int) Account_Media {
	r.Options.Offset = &offset
	return r
}

// Edit the properties of a media record by passing in a modified instance of a SoftLayer_Account_Media object.
func (r Account_Media) EditObject(templateObject *datatypes.Account_Media) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Media", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account to which the media belongs.
func (r Account_Media) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve a list supported media types for SoftLayer's Data Transfer Service.
func (r Account_Media) GetAllMediaTypes() (resp []datatypes.Account_Media_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getAllMediaTypes", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who created the media object.
func (r Account_Media) GetCreateUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getCreateUser", nil, &r.Options, &resp)
	return
}

// Retrieve The datacenter where the media resides.
func (r Account_Media) GetDatacenter() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getDatacenter", nil, &r.Options, &resp)
	return
}

// Retrieve The employee who last modified the media.
func (r Account_Media) GetModifyEmployee() (resp datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getModifyEmployee", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who last modified the media.
func (r Account_Media) GetModifyUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getModifyUser", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Media) GetObject() (resp datatypes.Account_Media, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The request to which the media belongs.
func (r Account_Media) GetRequest() (resp datatypes.Account_Media_Data_Transfer_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getRequest", nil, &r.Options, &resp)
	return
}

// Retrieve The media's type.
func (r Account_Media) GetType() (resp datatypes.Account_Media_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve A guest's associated EVault network storage service account.
func (r Account_Media) GetVolume() (resp datatypes.Network_Storage, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media", "getVolume", nil, &r.Options, &resp)
	return
}

// Remove a media from a SoftLayer account's list of media. The media record is not deleted.
func (r Account_Media) RemoveMediaFromList(mediaTemplate *datatypes.Account_Media) (resp int, err error) {
	params := []interface{}{
		mediaTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Media", "removeMediaFromList", params, &r.Options, &resp)
	return
}

// The SoftLayer_Account_Media_Data_Transfer_Request data type contains information on a single Data Transfer Service request. Creation of these requests is limited to SoftLayer customers through the SoftLayer Customer Portal.
type Account_Media_Data_Transfer_Request struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountMediaDataTransferRequestService returns an instance of the Account_Media_Data_Transfer_Request SoftLayer service
func GetAccountMediaDataTransferRequestService(sess *session.Session) Account_Media_Data_Transfer_Request {
	return Account_Media_Data_Transfer_Request{Session: sess}
}

func (r Account_Media_Data_Transfer_Request) Id(id int) Account_Media_Data_Transfer_Request {
	r.Options.Id = &id
	return r
}

func (r Account_Media_Data_Transfer_Request) Mask(mask string) Account_Media_Data_Transfer_Request {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Media_Data_Transfer_Request) Filter(filter string) Account_Media_Data_Transfer_Request {
	r.Options.Filter = filter
	return r
}

func (r Account_Media_Data_Transfer_Request) Limit(limit int) Account_Media_Data_Transfer_Request {
	r.Options.Limit = &limit
	return r
}

func (r Account_Media_Data_Transfer_Request) Offset(offset int) Account_Media_Data_Transfer_Request {
	r.Options.Offset = &offset
	return r
}

// Edit the properties of a data transfer request record by passing in a modified instance of a SoftLayer_Account_Media_Data_Transfer_Request object.
func (r Account_Media_Data_Transfer_Request) EditObject(templateObject *datatypes.Account_Media_Data_Transfer_Request) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account to which the request belongs.
func (r Account_Media_Data_Transfer_Request) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The active tickets that are attached to the data transfer request.
func (r Account_Media_Data_Transfer_Request) GetActiveTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getActiveTickets", nil, &r.Options, &resp)
	return
}

// Retrieves a list of all the possible statuses to which a request may be set.
func (r Account_Media_Data_Transfer_Request) GetAllRequestStatuses() (resp []datatypes.Account_Media_Data_Transfer_Request_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getAllRequestStatuses", nil, &r.Options, &resp)
	return
}

// Retrieve The billing item for the original request.
func (r Account_Media_Data_Transfer_Request) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who created the request.
func (r Account_Media_Data_Transfer_Request) GetCreateUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getCreateUser", nil, &r.Options, &resp)
	return
}

// Retrieve The media of the request.
func (r Account_Media_Data_Transfer_Request) GetMedia() (resp datatypes.Account_Media, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getMedia", nil, &r.Options, &resp)
	return
}

// Retrieve The employee who last modified the request.
func (r Account_Media_Data_Transfer_Request) GetModifyEmployee() (resp datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getModifyEmployee", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who last modified the request.
func (r Account_Media_Data_Transfer_Request) GetModifyUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getModifyUser", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Media_Data_Transfer_Request) GetObject() (resp datatypes.Account_Media_Data_Transfer_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The shipments of the request.
func (r Account_Media_Data_Transfer_Request) GetShipments() (resp []datatypes.Account_Shipment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getShipments", nil, &r.Options, &resp)
	return
}

// Retrieve The status of the request.
func (r Account_Media_Data_Transfer_Request) GetStatus() (resp datatypes.Account_Media_Data_Transfer_Request_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getStatus", nil, &r.Options, &resp)
	return
}

// Retrieve All tickets that are attached to the data transfer request.
func (r Account_Media_Data_Transfer_Request) GetTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Media_Data_Transfer_Request", "getTickets", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Note struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountNoteService returns an instance of the Account_Note SoftLayer service
func GetAccountNoteService(sess *session.Session) Account_Note {
	return Account_Note{Session: sess}
}

func (r Account_Note) Id(id int) Account_Note {
	r.Options.Id = &id
	return r
}

func (r Account_Note) Mask(mask string) Account_Note {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Note) Filter(filter string) Account_Note {
	r.Options.Filter = filter
	return r
}

func (r Account_Note) Limit(limit int) Account_Note {
	r.Options.Limit = &limit
	return r
}

func (r Account_Note) Offset(offset int) Account_Note {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Note) CreateObject(templateObject *datatypes.Account_Note) (resp datatypes.Account_Note, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Note", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Note) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Note) EditObject(templateObject *datatypes.Account_Note) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Note", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Note) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Note) GetCustomer() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note", "getCustomer", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Note) GetNoteHistory() (resp []datatypes.Account_Note_History, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note", "getNoteHistory", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Note) GetNoteType() (resp datatypes.Account_Note_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note", "getNoteType", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Note) GetObject() (resp datatypes.Account_Note, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Note_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountNoteTypeService returns an instance of the Account_Note_Type SoftLayer service
func GetAccountNoteTypeService(sess *session.Session) Account_Note_Type {
	return Account_Note_Type{Session: sess}
}

func (r Account_Note_Type) Id(id int) Account_Note_Type {
	r.Options.Id = &id
	return r
}

func (r Account_Note_Type) Mask(mask string) Account_Note_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Note_Type) Filter(filter string) Account_Note_Type {
	r.Options.Filter = filter
	return r
}

func (r Account_Note_Type) Limit(limit int) Account_Note_Type {
	r.Options.Limit = &limit
	return r
}

func (r Account_Note_Type) Offset(offset int) Account_Note_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Note_Type) CreateObject(templateObject *datatypes.Account_Note_Type) (resp datatypes.Account_Note_Type, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Note_Type", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Note_Type) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note_Type", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Note_Type) EditObject(templateObject *datatypes.Account_Note_Type) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Note_Type", "editObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Note_Type) GetAllObjects() (resp []datatypes.Account_Note_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Note_Type) GetObject() (resp datatypes.Account_Note_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Note_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Partner_Referral_Prospect struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountPartnerReferralProspectService returns an instance of the Account_Partner_Referral_Prospect SoftLayer service
func GetAccountPartnerReferralProspectService(sess *session.Session) Account_Partner_Referral_Prospect {
	return Account_Partner_Referral_Prospect{Session: sess}
}

func (r Account_Partner_Referral_Prospect) Id(id int) Account_Partner_Referral_Prospect {
	r.Options.Id = &id
	return r
}

func (r Account_Partner_Referral_Prospect) Mask(mask string) Account_Partner_Referral_Prospect {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Partner_Referral_Prospect) Filter(filter string) Account_Partner_Referral_Prospect {
	r.Options.Filter = filter
	return r
}

func (r Account_Partner_Referral_Prospect) Limit(limit int) Account_Partner_Referral_Prospect {
	r.Options.Limit = &limit
	return r
}

func (r Account_Partner_Referral_Prospect) Offset(offset int) Account_Partner_Referral_Prospect {
	r.Options.Offset = &offset
	return r
}

// Create a new Referral Partner Prospect
func (r Account_Partner_Referral_Prospect) CreateProspect(templateObject *datatypes.Container_Referral_Partner_Prospect, commit *bool) (resp datatypes.Account_Partner_Referral_Prospect, err error) {
	params := []interface{}{
		templateObject,
		commit,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Partner_Referral_Prospect", "createProspect", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Partner_Referral_Prospect) GetObject() (resp datatypes.Account_Partner_Referral_Prospect, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Partner_Referral_Prospect", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieves Questions for a Referral Partner Survey
func (r Account_Partner_Referral_Prospect) GetSurveyQuestions() (resp []datatypes.Survey_Question, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Partner_Referral_Prospect", "getSurveyQuestions", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Account_Password contains username, passwords and notes for services that may require for external applications such the Webcc interface for the EVault Storage service.
type Account_Password struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountPasswordService returns an instance of the Account_Password SoftLayer service
func GetAccountPasswordService(sess *session.Session) Account_Password {
	return Account_Password{Session: sess}
}

func (r Account_Password) Id(id int) Account_Password {
	r.Options.Id = &id
	return r
}

func (r Account_Password) Mask(mask string) Account_Password {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Password) Filter(filter string) Account_Password {
	r.Options.Filter = filter
	return r
}

func (r Account_Password) Limit(limit int) Account_Password {
	r.Options.Limit = &limit
	return r
}

func (r Account_Password) Offset(offset int) Account_Password {
	r.Options.Offset = &offset
	return r
}

// The password and/or notes may be modified.  Modifying the EVault passwords here will also update the password the Webcc interface will use.
func (r Account_Password) EditObject(templateObject *datatypes.Account_Password) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Password", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Password) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Password", "getAccount", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Account_Password object whose ID corresponds to the ID number of the init parameter passed to the SoftLayer_Account_Password service.
func (r Account_Password) GetObject() (resp datatypes.Account_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Password", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The service that an account/password combination is tied to.
func (r Account_Password) GetType() (resp datatypes.Account_Password_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Password", "getType", nil, &r.Options, &resp)
	return
}

//
//
//
//
//
type Account_Regional_Registry_Detail struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountRegionalRegistryDetailService returns an instance of the Account_Regional_Registry_Detail SoftLayer service
func GetAccountRegionalRegistryDetailService(sess *session.Session) Account_Regional_Registry_Detail {
	return Account_Regional_Registry_Detail{Session: sess}
}

func (r Account_Regional_Registry_Detail) Id(id int) Account_Regional_Registry_Detail {
	r.Options.Id = &id
	return r
}

func (r Account_Regional_Registry_Detail) Mask(mask string) Account_Regional_Registry_Detail {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Regional_Registry_Detail) Filter(filter string) Account_Regional_Registry_Detail {
	r.Options.Filter = filter
	return r
}

func (r Account_Regional_Registry_Detail) Limit(limit int) Account_Regional_Registry_Detail {
	r.Options.Limit = &limit
	return r
}

func (r Account_Regional_Registry_Detail) Offset(offset int) Account_Regional_Registry_Detail {
	r.Options.Offset = &offset
	return r
}

// <style type="text/css">.create_object > li > div { padding-top: .5em; padding-bottom: .5em}</style> This method will create a new SoftLayer_Account_Regional_Registry_Detail object.
//
// <b>Input</b> - [[SoftLayer_Account_Regional_Registry_Detail (type)|SoftLayer_Account_Regional_Registry_Detail]] <ul class="create_object"> <li><code>detailTypeId</code> <div>The [[SoftLayer_Account_Regional_Registry_Detail_Type|type id]] of this detail object</div> <ul> <li><b>Required</b></li> <li><b>Type</b> - integer</li> </ul> </li> <li><code>regionalInternetRegistryHandleId</code> <div> The id of the [[SoftLayer_Account_Rwhois_Handle|RWhois handle]] object. This is only to be used for detailed registrations, where a subnet is registered to an organization. The associated handle will be required to be a valid organization object id at the relevant registry. In this case, the detail object will only be valid for the registry the organization belongs to. </div> <ul> <li><b>Optional</b></li> <li><b>Type</b> - integer</li> </ul> </li> </ul>
func (r Account_Regional_Registry_Detail) CreateObject(templateObject *datatypes.Account_Regional_Registry_Detail) (resp datatypes.Account_Regional_Registry_Detail, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "createObject", params, &r.Options, &resp)
	return
}

// This method will delete an existing SoftLayer_Account_Regional_Registry_Detail object.
func (r Account_Regional_Registry_Detail) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method will edit an existing SoftLayer_Account_Regional_Registry_Detail object. For more detail, see [[SoftLayer_Account_Regional_Registry_Detail::createObject|createObject]].
func (r Account_Regional_Registry_Detail) EditObject(templateObject *datatypes.Account_Regional_Registry_Detail) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account that this detail object belongs to.
func (r Account_Regional_Registry_Detail) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The associated type of this detail object.
func (r Account_Regional_Registry_Detail) GetDetailType() (resp datatypes.Account_Regional_Registry_Detail_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "getDetailType", nil, &r.Options, &resp)
	return
}

// Retrieve References to the [[SoftLayer_Network_Subnet_Registration|registration objects]] that consume this detail object.
func (r Account_Regional_Registry_Detail) GetDetails() (resp []datatypes.Network_Subnet_Registration_Details, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "getDetails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Regional_Registry_Detail) GetObject() (resp datatypes.Account_Regional_Registry_Detail, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The individual properties that define this detail object's values.
func (r Account_Regional_Registry_Detail) GetProperties() (resp []datatypes.Account_Regional_Registry_Detail_Property, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "getProperties", nil, &r.Options, &resp)
	return
}

// Retrieve The associated RWhois handle of this detail object. Used only when detailed reassignments are necessary.
func (r Account_Regional_Registry_Detail) GetRegionalInternetRegistryHandle() (resp datatypes.Account_Rwhois_Handle, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "getRegionalInternetRegistryHandle", nil, &r.Options, &resp)
	return
}

// This method will create a bulk transaction to update any registrations that reference this detail object. It should only be called from a child class such as [[SoftLayer_Account_Regional_Registry_Detail_Person]] or [[SoftLayer_Account_Regional_Registry_Detail_Network]]. The registrations should be in the Open or Registration_Complete status.
func (r Account_Regional_Registry_Detail) UpdateReferencedRegistrations() (resp datatypes.Container_Network_Subnet_Registration_TransactionDetails, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail", "updateReferencedRegistrations", nil, &r.Options, &resp)
	return
}

// Subnet registration properties are used to define various attributes of the [[SoftLayer_Account_Regional_Registry_Detail|detail objects]]. These properties are defined by the [[SoftLayer_Account_Regional_Registry_Detail_Property_Type]] objects, which describe the available value formats.
type Account_Regional_Registry_Detail_Property struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountRegionalRegistryDetailPropertyService returns an instance of the Account_Regional_Registry_Detail_Property SoftLayer service
func GetAccountRegionalRegistryDetailPropertyService(sess *session.Session) Account_Regional_Registry_Detail_Property {
	return Account_Regional_Registry_Detail_Property{Session: sess}
}

func (r Account_Regional_Registry_Detail_Property) Id(id int) Account_Regional_Registry_Detail_Property {
	r.Options.Id = &id
	return r
}

func (r Account_Regional_Registry_Detail_Property) Mask(mask string) Account_Regional_Registry_Detail_Property {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Regional_Registry_Detail_Property) Filter(filter string) Account_Regional_Registry_Detail_Property {
	r.Options.Filter = filter
	return r
}

func (r Account_Regional_Registry_Detail_Property) Limit(limit int) Account_Regional_Registry_Detail_Property {
	r.Options.Limit = &limit
	return r
}

func (r Account_Regional_Registry_Detail_Property) Offset(offset int) Account_Regional_Registry_Detail_Property {
	r.Options.Offset = &offset
	return r
}

// <style type="text/css">.create_object > li > div { padding-top: .5em; padding-bottom: .5em}</style> This method will create a new SoftLayer_Account_Regional_Registry_Detail_Property object.
//
// <b>Input</b> - [[SoftLayer_Account_Regional_Registry_Detail_Property (type)|SoftLayer_Account_Regional_Registry_Detail_Property]] <ul class="create_object"> <li><code>registrationDetailId</code> <div>The numeric ID of the [[SoftLayer_Account_Regional_Registry_Detail|detail object]] this property belongs to</div> <ul> <li><b>Required</b></li> <li><b>Type</b> - integer</li> </ul> </li> <li><code>propertyTypeId</code> <div> The numeric ID of the associated [[SoftLayer_Account_Regional_Registry_Detail_Property_Type]] object </div> <ul> <li><b>Required</b></li> <li><b>Type</b> - integer</li> </ul> </li> <li><code>sequencePosition</code> <div> When more than one property of the same type exists on a detail object, this value determines the position in that collection. This can be thought of more as a sort order. </div> <ul> <li><b>Required</b></li> <li><b>Type</b> - integer</li> </ul> </li> <li><code>value</code> <div> The actual value of the property. </div> <ul> <li><b>Required</b></li> <li><b>Type</b> - string</li> </ul> </li> </ul>
func (r Account_Regional_Registry_Detail_Property) CreateObject(templateObject *datatypes.Account_Regional_Registry_Detail_Property) (resp datatypes.Account_Regional_Registry_Detail_Property, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property", "createObject", params, &r.Options, &resp)
	return
}

// Edit multiple [[SoftLayer_Account_Regional_Registry_Detail_Property]] objects.
func (r Account_Regional_Registry_Detail_Property) CreateObjects(templateObjects []datatypes.Account_Regional_Registry_Detail_Property) (resp []datatypes.Account_Regional_Registry_Detail_Property, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property", "createObjects", params, &r.Options, &resp)
	return
}

// This method will delete an existing SoftLayer_Account_Regional_Registry_Detail_Property object.
func (r Account_Regional_Registry_Detail_Property) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method will edit an existing SoftLayer_Account_Regional_Registry_Detail_Property object. For more detail, see [[SoftLayer_Account_Regional_Registry_Detail_Property::createObject|createObject]].
func (r Account_Regional_Registry_Detail_Property) EditObject(templateObject *datatypes.Account_Regional_Registry_Detail_Property) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property", "editObject", params, &r.Options, &resp)
	return
}

// Edit multiple [[SoftLayer_Account_Regional_Registry_Detail_Property]] objects.
func (r Account_Regional_Registry_Detail_Property) EditObjects(templateObjects []datatypes.Account_Regional_Registry_Detail_Property) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property", "editObjects", params, &r.Options, &resp)
	return
}

// Retrieve The [[SoftLayer_Account_Regional_Registry_Detail]] object this property belongs to
func (r Account_Regional_Registry_Detail_Property) GetDetail() (resp datatypes.Account_Regional_Registry_Detail, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property", "getDetail", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Regional_Registry_Detail_Property) GetObject() (resp datatypes.Account_Regional_Registry_Detail_Property, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The [[SoftLayer_Account_Regional_Registry_Detail_Property_Type]] object this property belongs to
func (r Account_Regional_Registry_Detail_Property) GetPropertyType() (resp datatypes.Account_Regional_Registry_Detail_Property_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property", "getPropertyType", nil, &r.Options, &resp)
	return
}

// Subnet Registration Detail Property Type objects describe the nature of a [[SoftLayer_Account_Regional_Registry_Detail_Property]] object. These types use [http://php.net/pcre.pattern.php Perl-Compatible Regular Expressions] to validate the value of a property object.
type Account_Regional_Registry_Detail_Property_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountRegionalRegistryDetailPropertyTypeService returns an instance of the Account_Regional_Registry_Detail_Property_Type SoftLayer service
func GetAccountRegionalRegistryDetailPropertyTypeService(sess *session.Session) Account_Regional_Registry_Detail_Property_Type {
	return Account_Regional_Registry_Detail_Property_Type{Session: sess}
}

func (r Account_Regional_Registry_Detail_Property_Type) Id(id int) Account_Regional_Registry_Detail_Property_Type {
	r.Options.Id = &id
	return r
}

func (r Account_Regional_Registry_Detail_Property_Type) Mask(mask string) Account_Regional_Registry_Detail_Property_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Regional_Registry_Detail_Property_Type) Filter(filter string) Account_Regional_Registry_Detail_Property_Type {
	r.Options.Filter = filter
	return r
}

func (r Account_Regional_Registry_Detail_Property_Type) Limit(limit int) Account_Regional_Registry_Detail_Property_Type {
	r.Options.Limit = &limit
	return r
}

func (r Account_Regional_Registry_Detail_Property_Type) Offset(offset int) Account_Regional_Registry_Detail_Property_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Regional_Registry_Detail_Property_Type) GetAllObjects() (resp []datatypes.Account_Regional_Registry_Detail_Property_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Regional_Registry_Detail_Property_Type) GetObject() (resp datatypes.Account_Regional_Registry_Detail_Property_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Property_Type", "getObject", nil, &r.Options, &resp)
	return
}

// Subnet Registration Detail Type objects describe the nature of a [[SoftLayer_Account_Regional_Registry_Detail]] object.
//
// The standard values for these objects are as follows: <ul> <li><strong>NETWORK</strong> - The detail object represents the information for a [[SoftLayer_Network_Subnet|subnet]]</li> <li><strong>NETWORK6</strong> - The detail object represents the information for an [[SoftLayer_Network_Subnet_Version6|IPv6 subnet]]</li> <li><strong>PERSON</strong> - The detail object represents the information for a customer with the RIR</li> </ul>
type Account_Regional_Registry_Detail_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountRegionalRegistryDetailTypeService returns an instance of the Account_Regional_Registry_Detail_Type SoftLayer service
func GetAccountRegionalRegistryDetailTypeService(sess *session.Session) Account_Regional_Registry_Detail_Type {
	return Account_Regional_Registry_Detail_Type{Session: sess}
}

func (r Account_Regional_Registry_Detail_Type) Id(id int) Account_Regional_Registry_Detail_Type {
	r.Options.Id = &id
	return r
}

func (r Account_Regional_Registry_Detail_Type) Mask(mask string) Account_Regional_Registry_Detail_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Regional_Registry_Detail_Type) Filter(filter string) Account_Regional_Registry_Detail_Type {
	r.Options.Filter = filter
	return r
}

func (r Account_Regional_Registry_Detail_Type) Limit(limit int) Account_Regional_Registry_Detail_Type {
	r.Options.Limit = &limit
	return r
}

func (r Account_Regional_Registry_Detail_Type) Offset(offset int) Account_Regional_Registry_Detail_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Regional_Registry_Detail_Type) GetAllObjects() (resp []datatypes.Account_Regional_Registry_Detail_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Regional_Registry_Detail_Type) GetObject() (resp datatypes.Account_Regional_Registry_Detail_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Regional_Registry_Detail_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Reports_Request struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountReportsRequestService returns an instance of the Account_Reports_Request SoftLayer service
func GetAccountReportsRequestService(sess *session.Session) Account_Reports_Request {
	return Account_Reports_Request{Session: sess}
}

func (r Account_Reports_Request) Id(id int) Account_Reports_Request {
	r.Options.Id = &id
	return r
}

func (r Account_Reports_Request) Mask(mask string) Account_Reports_Request {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Reports_Request) Filter(filter string) Account_Reports_Request {
	r.Options.Filter = filter
	return r
}

func (r Account_Reports_Request) Limit(limit int) Account_Reports_Request {
	r.Options.Limit = &limit
	return r
}

func (r Account_Reports_Request) Offset(offset int) Account_Reports_Request {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Reports_Request) CreateRequest(contact *datatypes.Account_Contact, reason *string, reportType *string) (resp datatypes.Account_Reports_Request, err error) {
	params := []interface{}{
		contact,
		reason,
		reportType,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "createRequest", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Reports_Request) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve A request's corresponding external contact, if one exists.
func (r Account_Reports_Request) GetAccountContact() (resp datatypes.Account_Contact, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "getAccountContact", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Reports_Request) GetAllObjects() (resp datatypes.Account_Reports_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Reports_Request) GetObject() (resp datatypes.Account_Reports_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve Type of the report customer is requesting for.
func (r Account_Reports_Request) GetReportType() (resp datatypes.Compliance_Report_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "getReportType", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Reports_Request) GetRequestByRequestKey(requestKey *string) (resp datatypes.Account_Reports_Request, err error) {
	params := []interface{}{
		requestKey,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "getRequestByRequestKey", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Account_Reports_Request) GetTicket() (resp datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "getTicket", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user that initiated a report request.
func (r Account_Reports_Request) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "getUser", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Reports_Request) SendReportEmail(request *datatypes.Account_Reports_Request) (resp bool, err error) {
	params := []interface{}{
		request,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "sendReportEmail", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Reports_Request) UpdateTicketOnDecline(request *datatypes.Account_Reports_Request) (resp bool, err error) {
	params := []interface{}{
		request,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Reports_Request", "updateTicketOnDecline", params, &r.Options, &resp)
	return
}

// The SoftLayer_Account_Shipment data type contains information relating to a shipment. Basic information such as addresses, the shipment courier, and any tracking information for as shipment is accessible with this data type.
type Account_Shipment struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountShipmentService returns an instance of the Account_Shipment SoftLayer service
func GetAccountShipmentService(sess *session.Session) Account_Shipment {
	return Account_Shipment{Session: sess}
}

func (r Account_Shipment) Id(id int) Account_Shipment {
	r.Options.Id = &id
	return r
}

func (r Account_Shipment) Mask(mask string) Account_Shipment {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Shipment) Filter(filter string) Account_Shipment {
	r.Options.Filter = filter
	return r
}

func (r Account_Shipment) Limit(limit int) Account_Shipment {
	r.Options.Limit = &limit
	return r
}

func (r Account_Shipment) Offset(offset int) Account_Shipment {
	r.Options.Offset = &offset
	return r
}

// Edit the properties of a shipment record by passing in a modified instance of a SoftLayer_Account_Shipment object.
func (r Account_Shipment) EditObject(templateObject *datatypes.Account_Shipment) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The account to which the shipment belongs.
func (r Account_Shipment) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve a list of available shipping couriers.
func (r Account_Shipment) GetAllCouriers() (resp []datatypes.Auxiliary_Shipping_Courier, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getAllCouriers", nil, &r.Options, &resp)
	return
}

// Retrieve a list of available shipping couriers.
func (r Account_Shipment) GetAllCouriersByType(courierTypeKeyName *string) (resp []datatypes.Auxiliary_Shipping_Courier, err error) {
	params := []interface{}{
		courierTypeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getAllCouriersByType", params, &r.Options, &resp)
	return
}

// Retrieve a a list of shipment statuses.
func (r Account_Shipment) GetAllShipmentStatuses() (resp []datatypes.Account_Shipment_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getAllShipmentStatuses", nil, &r.Options, &resp)
	return
}

// Retrieve a a list of shipment types.
func (r Account_Shipment) GetAllShipmentTypes() (resp []datatypes.Account_Shipment_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getAllShipmentTypes", nil, &r.Options, &resp)
	return
}

// Retrieve The courier handling the shipment.
func (r Account_Shipment) GetCourier() (resp datatypes.Auxiliary_Shipping_Courier, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getCourier", nil, &r.Options, &resp)
	return
}

// Retrieve The employee who created the shipment.
func (r Account_Shipment) GetCreateEmployee() (resp datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getCreateEmployee", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who created the shipment.
func (r Account_Shipment) GetCreateUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getCreateUser", nil, &r.Options, &resp)
	return
}

// Retrieve The address at which the shipment is received.
func (r Account_Shipment) GetDestinationAddress() (resp datatypes.Account_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getDestinationAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The employee who last modified the shipment.
func (r Account_Shipment) GetModifyEmployee() (resp datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getModifyEmployee", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who last modified the shipment.
func (r Account_Shipment) GetModifyUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getModifyUser", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Shipment) GetObject() (resp datatypes.Account_Shipment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The address from which the shipment is sent.
func (r Account_Shipment) GetOriginationAddress() (resp datatypes.Account_Address, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getOriginationAddress", nil, &r.Options, &resp)
	return
}

// Retrieve The items in the shipment.
func (r Account_Shipment) GetShipmentItems() (resp []datatypes.Account_Shipment_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getShipmentItems", nil, &r.Options, &resp)
	return
}

// Retrieve The status of the shipment.
func (r Account_Shipment) GetStatus() (resp datatypes.Account_Shipment_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The tracking data for the shipment.
func (r Account_Shipment) GetTrackingData() (resp []datatypes.Account_Shipment_Tracking_Data, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getTrackingData", nil, &r.Options, &resp)
	return
}

// Retrieve The type of shipment (e.g. for Data Transfer Service or Colocation Service).
func (r Account_Shipment) GetType() (resp datatypes.Account_Shipment_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment", "getType", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Account_Shipment_Item data type contains information relating to a shipment's item. Basic information such as addresses, the shipment courier, and any tracking information for as shipment is accessible with this data type.
type Account_Shipment_Item struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountShipmentItemService returns an instance of the Account_Shipment_Item SoftLayer service
func GetAccountShipmentItemService(sess *session.Session) Account_Shipment_Item {
	return Account_Shipment_Item{Session: sess}
}

func (r Account_Shipment_Item) Id(id int) Account_Shipment_Item {
	r.Options.Id = &id
	return r
}

func (r Account_Shipment_Item) Mask(mask string) Account_Shipment_Item {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Shipment_Item) Filter(filter string) Account_Shipment_Item {
	r.Options.Filter = filter
	return r
}

func (r Account_Shipment_Item) Limit(limit int) Account_Shipment_Item {
	r.Options.Limit = &limit
	return r
}

func (r Account_Shipment_Item) Offset(offset int) Account_Shipment_Item {
	r.Options.Offset = &offset
	return r
}

// Edit the properties of a shipment record by passing in a modified instance of a SoftLayer_Account_Shipment_Item object.
func (r Account_Shipment_Item) EditObject(templateObject *datatypes.Account_Shipment_Item) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Item", "editObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Shipment_Item) GetObject() (resp datatypes.Account_Shipment_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Item", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The shipment to which this item belongs.
func (r Account_Shipment_Item) GetShipment() (resp datatypes.Account_Shipment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Item", "getShipment", nil, &r.Options, &resp)
	return
}

// Retrieve The type of this shipment item.
func (r Account_Shipment_Item) GetShipmentItemType() (resp datatypes.Account_Shipment_Item_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Item", "getShipmentItemType", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Shipment_Item_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountShipmentItemTypeService returns an instance of the Account_Shipment_Item_Type SoftLayer service
func GetAccountShipmentItemTypeService(sess *session.Session) Account_Shipment_Item_Type {
	return Account_Shipment_Item_Type{Session: sess}
}

func (r Account_Shipment_Item_Type) Id(id int) Account_Shipment_Item_Type {
	r.Options.Id = &id
	return r
}

func (r Account_Shipment_Item_Type) Mask(mask string) Account_Shipment_Item_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Shipment_Item_Type) Filter(filter string) Account_Shipment_Item_Type {
	r.Options.Filter = filter
	return r
}

func (r Account_Shipment_Item_Type) Limit(limit int) Account_Shipment_Item_Type {
	r.Options.Limit = &limit
	return r
}

func (r Account_Shipment_Item_Type) Offset(offset int) Account_Shipment_Item_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Shipment_Item_Type) GetObject() (resp datatypes.Account_Shipment_Item_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Item_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Shipment_Resource_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountShipmentResourceTypeService returns an instance of the Account_Shipment_Resource_Type SoftLayer service
func GetAccountShipmentResourceTypeService(sess *session.Session) Account_Shipment_Resource_Type {
	return Account_Shipment_Resource_Type{Session: sess}
}

func (r Account_Shipment_Resource_Type) Id(id int) Account_Shipment_Resource_Type {
	r.Options.Id = &id
	return r
}

func (r Account_Shipment_Resource_Type) Mask(mask string) Account_Shipment_Resource_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Shipment_Resource_Type) Filter(filter string) Account_Shipment_Resource_Type {
	r.Options.Filter = filter
	return r
}

func (r Account_Shipment_Resource_Type) Limit(limit int) Account_Shipment_Resource_Type {
	r.Options.Limit = &limit
	return r
}

func (r Account_Shipment_Resource_Type) Offset(offset int) Account_Shipment_Resource_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Shipment_Resource_Type) GetObject() (resp datatypes.Account_Shipment_Resource_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Resource_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Shipment_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountShipmentStatusService returns an instance of the Account_Shipment_Status SoftLayer service
func GetAccountShipmentStatusService(sess *session.Session) Account_Shipment_Status {
	return Account_Shipment_Status{Session: sess}
}

func (r Account_Shipment_Status) Id(id int) Account_Shipment_Status {
	r.Options.Id = &id
	return r
}

func (r Account_Shipment_Status) Mask(mask string) Account_Shipment_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Shipment_Status) Filter(filter string) Account_Shipment_Status {
	r.Options.Filter = filter
	return r
}

func (r Account_Shipment_Status) Limit(limit int) Account_Shipment_Status {
	r.Options.Limit = &limit
	return r
}

func (r Account_Shipment_Status) Offset(offset int) Account_Shipment_Status {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Shipment_Status) GetObject() (resp datatypes.Account_Shipment_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Status", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Account_Shipment_Tracking_Data data type contains information on a single piece of tracking information pertaining to a shipment. This tracking information tracking numbers by which the shipment may be tracked through the shipping courier.
type Account_Shipment_Tracking_Data struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountShipmentTrackingDataService returns an instance of the Account_Shipment_Tracking_Data SoftLayer service
func GetAccountShipmentTrackingDataService(sess *session.Session) Account_Shipment_Tracking_Data {
	return Account_Shipment_Tracking_Data{Session: sess}
}

func (r Account_Shipment_Tracking_Data) Id(id int) Account_Shipment_Tracking_Data {
	r.Options.Id = &id
	return r
}

func (r Account_Shipment_Tracking_Data) Mask(mask string) Account_Shipment_Tracking_Data {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Shipment_Tracking_Data) Filter(filter string) Account_Shipment_Tracking_Data {
	r.Options.Filter = filter
	return r
}

func (r Account_Shipment_Tracking_Data) Limit(limit int) Account_Shipment_Tracking_Data {
	r.Options.Limit = &limit
	return r
}

func (r Account_Shipment_Tracking_Data) Offset(offset int) Account_Shipment_Tracking_Data {
	r.Options.Offset = &offset
	return r
}

// Create a new shipment tracking data. The ''shipmentId'', ''sequence'', and ''trackingData'' properties in the templateObject parameter are required parameters to create a tracking data record.
func (r Account_Shipment_Tracking_Data) CreateObject(templateObject *datatypes.Account_Shipment_Tracking_Data) (resp datatypes.Account_Shipment_Tracking_Data, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "createObject", params, &r.Options, &resp)
	return
}

// Create a new shipment tracking data. The ''shipmentId'', ''sequence'', and ''trackingData'' properties of each templateObject in the templateObjects array are required parameters to create a tracking data record.
func (r Account_Shipment_Tracking_Data) CreateObjects(templateObjects []datatypes.Account_Shipment_Tracking_Data) (resp []datatypes.Account_Shipment_Tracking_Data, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "createObjects", params, &r.Options, &resp)
	return
}

// deleteObject permanently removes a shipment tracking datum (number)
func (r Account_Shipment_Tracking_Data) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "deleteObject", nil, &r.Options, &resp)
	return
}

// Edit the properties of a tracking data record by passing in a modified instance of a SoftLayer_Account_Shipment_Tracking_Data object.
func (r Account_Shipment_Tracking_Data) EditObject(templateObject *datatypes.Account_Shipment_Tracking_Data) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The employee who created the tracking datum.
func (r Account_Shipment_Tracking_Data) GetCreateEmployee() (resp datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "getCreateEmployee", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who created the tracking datum.
func (r Account_Shipment_Tracking_Data) GetCreateUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "getCreateUser", nil, &r.Options, &resp)
	return
}

// Retrieve The employee who last modified the tracking datum.
func (r Account_Shipment_Tracking_Data) GetModifyEmployee() (resp datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "getModifyEmployee", nil, &r.Options, &resp)
	return
}

// Retrieve The customer user who last modified the tracking datum.
func (r Account_Shipment_Tracking_Data) GetModifyUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "getModifyUser", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Account_Shipment_Tracking_Data) GetObject() (resp datatypes.Account_Shipment_Tracking_Data, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The shipment of the tracking datum.
func (r Account_Shipment_Tracking_Data) GetShipment() (resp datatypes.Account_Shipment, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Tracking_Data", "getShipment", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Account_Shipment_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAccountShipmentTypeService returns an instance of the Account_Shipment_Type SoftLayer service
func GetAccountShipmentTypeService(sess *session.Session) Account_Shipment_Type {
	return Account_Shipment_Type{Session: sess}
}

func (r Account_Shipment_Type) Id(id int) Account_Shipment_Type {
	r.Options.Id = &id
	return r
}

func (r Account_Shipment_Type) Mask(mask string) Account_Shipment_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Account_Shipment_Type) Filter(filter string) Account_Shipment_Type {
	r.Options.Filter = filter
	return r
}

func (r Account_Shipment_Type) Limit(limit int) Account_Shipment_Type {
	r.Options.Limit = &limit
	return r
}

func (r Account_Shipment_Type) Offset(offset int) Account_Shipment_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Account_Shipment_Type) GetObject() (resp datatypes.Account_Shipment_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Account_Shipment_Type", "getObject", nil, &r.Options, &resp)
	return
}
