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

// The SoftLayer_User_Customer data type contains general information relating to a single SoftLayer customer portal user. Personal information in this type such as names, addresses, and phone numbers are not necessarily associated with the customer account the user is assigned to.
type User_Customer struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerService returns an instance of the User_Customer SoftLayer service
func GetUserCustomerService(sess *session.Session) User_Customer {
	return User_Customer{Session: sess}
}

func (r User_Customer) Id(id int) User_Customer {
	r.Options.Id = &id
	return r
}

func (r User_Customer) Mask(mask string) User_Customer {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer) Filter(filter string) User_Customer {
	r.Options.Filter = filter
	return r
}

func (r User_Customer) Limit(limit int) User_Customer {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer) Offset(offset int) User_Customer {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Customer) AcknowledgeSupportPolicy() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_User_Customer", "acknowledgeSupportPolicy", nil, &r.Options, &resp)
	return
}

// Create a user's API authentication key, allowing that user access to query the SoftLayer API. addApiAuthenticationKey() returns the users new API key. Each portal user is allowed a maximum of two API keys.
func (r User_Customer) AddApiAuthenticationKey() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addApiAuthenticationKey", nil, &r.Options, &resp)
	return
}

// Add multiple hardware to a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. Hardware does not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. addBulkHardwareAccess() does not attempt to add hardware access if the given user already has access to that hardware object.
//
// Users can assign hardware access to their child users, but not to themselves. An account's master has access to all hardware on their customer account and can set hardware access for any of the other users on their account.
func (r User_Customer) AddBulkHardwareAccess(hardwareIds []int) (resp bool, err error) {
	params := []interface{}{
		hardwareIds,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addBulkHardwareAccess", params, &r.Options, &resp)
	return
}

// Add multiple permissions to a portal user's permission set. [[Permissions]] control which features in the SoftLayer customer portal and API a user may use. addBulkPortalPermission() does not attempt to add permissions already assigned to the user.
//
// Users can assign permissions to their child users, but not to themselves. An account's master has all portal permissions and can set permissions for any of the other users on their account.
//
// Use the [[SoftLayer_User_Customer_CustomerPermission_Permission::getAllObjects]] method to retrieve a list of all permissions available in the SoftLayer customer portal and API. Permissions are removed based on the keyName property of the permission objects within the permissions parameter.
func (r User_Customer) AddBulkPortalPermission(permissions []datatypes.User_Customer_CustomerPermission_Permission) (resp bool, err error) {
	params := []interface{}{
		permissions,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addBulkPortalPermission", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) AddBulkRoles(roles []datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		roles,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addBulkRoles", params, &r.Options, &resp)
	return
}

// Add multiple CloudLayer Computing Instances to a portal user's access list. A user's CloudLayer Computing Instance access list controls which of an account's CloudLayer Computing Instance objects a user has access to in the SoftLayer customer portal and API. CloudLayer Computing Instances do not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. addBulkVirtualGuestAccess() does not attempt to add CloudLayer Computing Instance access if the given user already has access to that CloudLayer Computing Instance object.
//
// Users can assign CloudLayer Computing Instance access to their child users, but not to themselves. An account's master has access to all CloudLayer Computing Instances on their customer account and can set CloudLayer Computing Instance access for any of the other users on their account.
func (r User_Customer) AddBulkVirtualGuestAccess(virtualGuestIds []int) (resp bool, err error) {
	params := []interface{}{
		virtualGuestIds,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addBulkVirtualGuestAccess", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) AddExternalBinding(externalBinding *datatypes.User_External_Binding) (resp datatypes.User_Customer_External_Binding, err error) {
	params := []interface{}{
		externalBinding,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addExternalBinding", params, &r.Options, &resp)
	return
}

// Add hardware to a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. Hardware does not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user already has access to the hardware you're attempting to add then addHardwareAccess() returns true.
//
// Users can assign hardware access to their child users, but not to themselves. An account's master has access to all hardware on their customer account and can set hardware access for any of the other users on their account.
func (r User_Customer) AddHardwareAccess(hardwareId *int) (resp bool, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addHardwareAccess", params, &r.Options, &resp)
	return
}

// Create a notification subscription record for the user. If a subscription record exists for the notification, the record will be set to active, if currently inactive.
func (r User_Customer) AddNotificationSubscriber(notificationKeyName *string) (resp bool, err error) {
	params := []interface{}{
		notificationKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addNotificationSubscriber", params, &r.Options, &resp)
	return
}

// Add a permission to a portal user's permission set. [[Permissions]] control which features in the SoftLayer customer portal and API a user may use. If the user already has the permission you're attempting to add then addPortalPermission() returns true.
//
// Users can assign permissions to their child users, but not to themselves. An account's master has all portal permissions and can set permissions for any of the other users on their account.
//
// Use the [[SoftLayer_User_Customer_CustomerPermission_Permission::getAllObjects]] method to retrieve a list of all permissions available in the SoftLayer customer portal and API. Permissions are added based on the keyName property of the permission parameter.
func (r User_Customer) AddPortalPermission(permission *datatypes.User_Customer_CustomerPermission_Permission) (resp bool, err error) {
	params := []interface{}{
		permission,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addPortalPermission", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) AddRole(role *datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		role,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addRole", params, &r.Options, &resp)
	return
}

// Add a CloudLayer Computing Instance to a portal user's access list. A user's CloudLayer Computing Instance access list controls which of an account's CloudLayer Computing Instance objects a user has access to in the SoftLayer customer portal and API. CloudLayer Computing Instances do not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user already has access to the CloudLayer Computing Instance you're attempting to add then addVirtualGuestAccess() returns true.
//
// Users can assign CloudLayer Computing Instance access to their child users, but not to themselves. An account's master has access to all CloudLayer Computing Instances on their customer account and can set CloudLayer Computing Instance access for any of the other users on their account.
func (r User_Customer) AddVirtualGuestAccess(virtualGuestId *int) (resp bool, err error) {
	params := []interface{}{
		virtualGuestId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "addVirtualGuestAccess", params, &r.Options, &resp)
	return
}

// Select a type of preference you would like to modify using [[SoftLayer_User_Customer::getPreferenceTypes|getPreferenceTypes]] and invoke this method using that preference type key name.
func (r User_Customer) ChangePreference(preferenceTypeKeyName *string, value *string) (resp []datatypes.User_Preference, err error) {
	params := []interface{}{
		preferenceTypeKeyName,
		value,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "changePreference", params, &r.Options, &resp)
	return
}

// This service checks the result of a previously requested external authentication. [[SoftLayer_Container_User_Customer_External_Binding_Phone|Phone external binding]] container can be used for this service. Make sure to set the [[SoftLayer_Container_User_Customer_External_Binding_Phone::authenticationToken|authenticationToken]] that is generated by [[SoftLayer_User_Customer|initiateExternalAuthentication]] service.
func (r User_Customer) CheckExternalAuthenticationStatus(authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "checkExternalAuthenticationStatus", params, &r.Options, &resp)
	return
}

// Add a description here
//
//
func (r User_Customer) CheckPhoneFactorAuthenticationForPasswordSet(passwordSet *datatypes.Container_User_Customer_PasswordSet, authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp bool, err error) {
	params := []interface{}{
		passwordSet,
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "checkPhoneFactorAuthenticationForPasswordSet", params, &r.Options, &resp)
	return
}

// Create a new subscriber for a given resource.
func (r User_Customer) CreateNotificationSubscriber(keyName *string, resourceTableId *int) (resp bool, err error) {
	params := []interface{}{
		keyName,
		resourceTableId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "createNotificationSubscriber", params, &r.Options, &resp)
	return
}

// Create a new user in the SoftLayer customer portal. createObject() creates a user's portal record and adds them into the SoftLayer community forums. It is no longer possible to set up the SSL or PPTP enable flag in this call since the manage permissions have not yet been set.  You will need to make a subsequent call to edit object in order to enable VPN access. An account's master user and sub-users who have the User Manage permission can add new users. createObject() creates users with a default permission set. After adding a user it may be helpful to set their permissions and hardware access.
//
// Note, neither password nor vpnPassword parameters are required.
//
// Password When a new user is created, an email will be sent to the new user's email address with a link to a url that will allow the new user to create or change their password for the SoftLayer customer portal.
//
// If the password parameter is provided and is not null, then that value will be validated. If it is a valid password, then the user will be created with this password.  This user will still receive a portal password email.  It can be used within 24 hours to change their password, or it can be allowed to expire, and the password provided during user creation will remain as the user's password.
//
// If the password parameter is not provided or the value is null, the user must set their portal password using the link sent in email within 24 hours.  If the user fails to set their password within 24 hours, then a non-master user can use the "Reset Password" link on the login page of the portal to request a new email.  A master user can use the link to retrieve a phone number to call to assist in resetting their password.
//
// The password parameter is ignored for VPN_ONLY users or for IBMid authenticated users.
//
// vpnPassword If the vpnPassword is provided, then the user's vpnPassword will be set to the provided password.  When creating a vpn only user, the vpnPassword MUST be supplied.  If the vpnPassword is not provided, then the user will need to use the portal to edit their profile and set the vpnPassword.
//
//
func (r User_Customer) CreateObject(templateObject *datatypes.User_Customer, password *string, vpnPassword *string) (resp datatypes.User_Customer, err error) {
	params := []interface{}{
		templateObject,
		password,
		vpnPassword,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "createObject", params, &r.Options, &resp)
	return
}

// Create delivery methods for a notification that the user is subscribed to. Multiple delivery method keyNames can be supplied to create multiple delivery methods for the specified notification. Available delivery methods - 'EMAIL'. Available notifications - 'PLANNED_MAINTENANCE', 'UNPLANNED_INCIDENT'.
func (r User_Customer) CreateSubscriberDeliveryMethods(notificationKeyName *string, deliveryMethodKeyNames []string) (resp bool, err error) {
	params := []interface{}{
		notificationKeyName,
		deliveryMethodKeyNames,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "createSubscriberDeliveryMethods", params, &r.Options, &resp)
	return
}

// Create a new subscriber for a given resource.
func (r User_Customer) DeactivateNotificationSubscriber(keyName *string, resourceTableId *int) (resp bool, err error) {
	params := []interface{}{
		keyName,
		resourceTableId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "deactivateNotificationSubscriber", params, &r.Options, &resp)
	return
}

// Account master users and sub-users who have the User Manage permission in the SoftLayer customer portal can update other user's information. Use editObject() if you wish to edit a single user account. Users who do not have the User Manage permission can only update their own information.
func (r User_Customer) EditObject(templateObject *datatypes.User_Customer) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "editObject", params, &r.Options, &resp)
	return
}

// Account master users and sub-users who have the User Manage permission in the SoftLayer customer portal can update other user's information. Use editObjects() if you wish to edit multiple users at once. Users who do not have the User Manage permission can only update their own information.
func (r User_Customer) EditObjects(templateObjects []datatypes.User_Customer) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "editObjects", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) FindUserPreference(profileName *string, containerKeyname *string, preferenceKeyname *string) (resp []datatypes.Layout_Profile, err error) {
	params := []interface{}{
		profileName,
		containerKeyname,
		preferenceKeyname,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "findUserPreference", params, &r.Options, &resp)
	return
}

// Retrieve The customer account that a user belongs to.
func (r User_Customer) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer) GetActions() (resp []datatypes.User_Permission_Action, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getActions", nil, &r.Options, &resp)
	return
}

// The getActiveExternalAuthenticationVendors method will return a list of available external vendors that a SoftLayer user can authenticate against.  The list will only contain vendors for which the user has at least one active external binding.
func (r User_Customer) GetActiveExternalAuthenticationVendors() (resp []datatypes.Container_User_Customer_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getActiveExternalAuthenticationVendors", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's additional email addresses. These email addresses are contacted when updates are made to support tickets.
func (r User_Customer) GetAdditionalEmails() (resp []datatypes.User_Customer_AdditionalEmail, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getAdditionalEmails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) GetAllowedHardwareIds() (resp []int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getAllowedHardwareIds", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) GetAllowedVirtualGuestIds() (resp []int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getAllowedVirtualGuestIds", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's API Authentication keys. There is a max limit of two API keys per user.
func (r User_Customer) GetApiAuthenticationKeys() (resp []datatypes.User_Customer_ApiAuthentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getApiAuthenticationKeys", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) GetAuthenticationToken(token *datatypes.Container_User_Authentication_Token) (resp datatypes.Container_User_Authentication_Token, err error) {
	params := []interface{}{
		token,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getAuthenticationToken", params, &r.Options, &resp)
	return
}

// Retrieve The CDN accounts associated with a portal user.
func (r User_Customer) GetCdnAccounts() (resp []datatypes.Network_ContentDelivery_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getCdnAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's child users. Some portal users may not have child users.
func (r User_Customer) GetChildUsers() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getChildUsers", nil, &r.Options, &resp)
	return
}

// Retrieve An user's associated closed tickets.
func (r User_Customer) GetClosedTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getClosedTickets", nil, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong>
func (r User_Customer) GetDefaultSecurityQuestions(key *string) (resp []datatypes.User_Security_Question, err error) {
	params := []interface{}{
		key,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getDefaultSecurityQuestions", params, &r.Options, &resp)
	return
}

// Retrieve The external authentication bindings that link an external identifier to a SoftLayer user.
func (r User_Customer) GetExternalBindings() (resp []datatypes.User_External_Binding, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getExternalBindings", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's accessible hardware. These permissions control which hardware a user has access to in the SoftLayer customer portal.
func (r User_Customer) GetHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve the number of servers that a portal user has access to. Portal users can have restrictions set to limit services for and to perform actions on hardware. You can set these permissions in the portal by clicking the "administrative" then "user admin" links.
func (r User_Customer) GetHardwareCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getHardwareCount", nil, &r.Options, &resp)
	return
}

// Retrieve Hardware notifications associated with this user. A hardware notification links a user to a piece of hardware, and that user will be notified if any monitors on that hardware fail, if the monitors have a status of 'Notify User'.
func (r User_Customer) GetHardwareNotifications() (resp []datatypes.User_Customer_Notification_Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getHardwareNotifications", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a user has acknowledged the support policy.
func (r User_Customer) GetHasAcknowledgedSupportPolicyFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getHasAcknowledgedSupportPolicyFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a portal user has access to all hardware on their account.
func (r User_Customer) GetHasFullHardwareAccessFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getHasFullHardwareAccessFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a portal user has access to all hardware on their account.
func (r User_Customer) GetHasFullVirtualGuestAccessFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getHasFullVirtualGuestAccessFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) GetImpersonationToken() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getImpersonationToken", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer) GetLayoutProfiles() (resp []datatypes.Layout_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getLayoutProfiles", nil, &r.Options, &resp)
	return
}

// Retrieve A user's locale. Locale holds user's language and region information.
func (r User_Customer) GetLocale() (resp datatypes.Locale, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getLocale", nil, &r.Options, &resp)
	return
}

// Retrieve A user's attempts to log into the SoftLayer customer portal.
func (r User_Customer) GetLoginAttempts() (resp []datatypes.User_Customer_Access_Authentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getLoginAttempts", nil, &r.Options, &resp)
	return
}

// Attempt to authenticate a user to the SoftLayer customer portal using the provided authentication container. Depending on the specific type of authentication container that is used, this API will leverage the appropriate authentication protocol. If authentication is successful then the API returns a list of linked accounts for the user, a token containing the ID of the authenticated user and a hash key used by the SoftLayer customer portal to maintain authentication.
func (r User_Customer) GetLoginToken(request *datatypes.Container_Authentication_Request_Contract) (resp datatypes.Container_Authentication_Response_Common, err error) {
	params := []interface{}{
		request,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getLoginToken", params, &r.Options, &resp)
	return
}

// Retrieve A portal user's associated mobile device profiles.
func (r User_Customer) GetMobileDevices() (resp []datatypes.User_Customer_MobileDevice, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getMobileDevices", nil, &r.Options, &resp)
	return
}

// Retrieve Notification subscription records for the user.
func (r User_Customer) GetNotificationSubscribers() (resp []datatypes.Notification_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getNotificationSubscribers", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_User_Customer object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_User_Customer service. You can only retrieve users that are assigned to the customer account belonging to the user making the API call.
func (r User_Customer) GetObject() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getObject", nil, &r.Options, &resp)
	return
}

// This API returns a SoftLayer_Container_User_Customer_OpenIdConnect_MigrationState object containing the necessary information to determine what migration state the user is in. If the account is not OpenIdConnect authenticated, then an exception is thrown.
func (r User_Customer) GetOpenIdConnectMigrationState() (resp datatypes.Container_User_Customer_OpenIdConnect_MigrationState, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getOpenIdConnectMigrationState", nil, &r.Options, &resp)
	return
}

// Retrieve An user's associated open tickets.
func (r User_Customer) GetOpenTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getOpenTickets", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's vpn accessible subnets.
func (r User_Customer) GetOverrides() (resp []datatypes.Network_Service_Vpn_Overrides, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getOverrides", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's parent user. If a SoftLayer_User_Customer has a null parentId property then it doesn't have a parent user.
func (r User_Customer) GetParent() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getParent", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's permissions. These permissions control that user's access to functions within the SoftLayer customer portal and API.
func (r User_Customer) GetPermissions() (resp []datatypes.User_Customer_CustomerPermission_Permission, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getPermissions", nil, &r.Options, &resp)
	return
}

// Attempt to authenticate a username and password to the SoftLayer customer portal. Many portal user accounts are configured to require answering a security question on login. In this case getPortalLoginToken() also verifies the given security question ID and answer. If authentication is successful then the API returns a token containing the ID of the authenticated user and a hash key used by the SoftLayer customer portal to maintain authentication.
func (r User_Customer) GetPortalLoginToken(username *string, password *string, securityQuestionId *int, securityQuestionAnswer *string) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		username,
		password,
		securityQuestionId,
		securityQuestionAnswer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getPortalLoginToken", params, &r.Options, &resp)
	return
}

// Select a type of preference you would like to get using [[SoftLayer_User_Customer::getPreferenceTypes|getPreferenceTypes]] and invoke this method using that preference type key name.
func (r User_Customer) GetPreference(preferenceTypeKeyName *string) (resp datatypes.User_Preference, err error) {
	params := []interface{}{
		preferenceTypeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getPreference", params, &r.Options, &resp)
	return
}

// Use any of the preference types to fetch or modify user preferences using [[SoftLayer_User_Customer::getPreference|getPreference]] or [[SoftLayer_User_Customer::changePreference|changePreference]], respectively.
func (r User_Customer) GetPreferenceTypes() (resp []datatypes.User_Preference_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getPreferenceTypes", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer) GetPreferences() (resp []datatypes.User_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getPreferences", nil, &r.Options, &resp)
	return
}

// Retrieve the authentication requirements for an outstanding password set/reset request.  The password key provided to the user in an email generated by the [[SoftLayer_User_Customer::newUserPassword|newUserPassword]]. Password recovery keys are valid for 24 hours after they're generated.
func (r User_Customer) GetRequirementsForPasswordSet(passwordSet *datatypes.Container_User_Customer_PasswordSet) (resp datatypes.Container_User_Customer_PasswordSet, err error) {
	params := []interface{}{
		passwordSet,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getRequirementsForPasswordSet", params, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer) GetRoles() (resp []datatypes.User_Permission_Role, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getRoles", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer) GetSalesforceUserLink() (resp datatypes.User_Customer_Link, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSalesforceUserLink", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's security question answers. Some portal users may not have security answers or may not be configured to require answering a security question on login.
func (r User_Customer) GetSecurityAnswers() (resp []datatypes.User_Customer_Security_Answer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSecurityAnswers", nil, &r.Options, &resp)
	return
}

// Retrieve A user's notification subscription records.
func (r User_Customer) GetSubscribers() (resp []datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSubscribers", nil, &r.Options, &resp)
	return
}

// Retrieve A user's successful attempts to log into the SoftLayer customer portal.
func (r User_Customer) GetSuccessfulLogins() (resp []datatypes.User_Customer_Access_Authentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSuccessfulLogins", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a user is required to acknowledge the support policy for portal access.
func (r User_Customer) GetSupportPolicyAcknowledgementRequiredFlag() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSupportPolicyAcknowledgementRequiredFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) GetSupportPolicyDocument() (resp []byte, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSupportPolicyDocument", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) GetSupportPolicyName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSupportPolicyName", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) GetSupportedLocales() (resp []datatypes.Locale, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSupportedLocales", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a user must take a brief survey the next time they log into the SoftLayer customer portal.
func (r User_Customer) GetSurveyRequiredFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSurveyRequiredFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The surveys that a user has taken in the SoftLayer customer portal.
func (r User_Customer) GetSurveys() (resp []datatypes.Survey, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getSurveys", nil, &r.Options, &resp)
	return
}

// Retrieve An user's associated tickets.
func (r User_Customer) GetTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getTickets", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's time zone.
func (r User_Customer) GetTimezone() (resp datatypes.Locale_Timezone, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getTimezone", nil, &r.Options, &resp)
	return
}

// Retrieve A user's unsuccessful attempts to log into the SoftLayer customer portal.
func (r User_Customer) GetUnsuccessfulLogins() (resp []datatypes.User_Customer_Access_Authentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getUnsuccessfulLogins", nil, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong> Retrieve a user object using a password recovery key received in an email generated by the [[SoftLayer_User_Customer::lostPassword|lostPassword]] method. The SoftLayer customer portal uses getUserFromLostPasswordRequest() to retrieve user security questions. Password recovery keys are valid for 24 hours after they're generated.
func (r User_Customer) GetUserFromLostPasswordRequest(key *string) (resp []datatypes.User_Security_Question, err error) {
	params := []interface{}{
		key,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getUserFromLostPasswordRequest", params, &r.Options, &resp)
	return
}

// Retrieve a user object using a password token. When a new user is created or when a user has requested a password change using initiatePortalPasswordChange, they will have received an email that contains a url with a token.  That token is used as the parameter for getUserIdForPasswordSet.
func (r User_Customer) GetUserIdForPasswordSet(key *string) (resp int, err error) {
	params := []interface{}{
		key,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getUserIdForPasswordSet", params, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer) GetUserLinks() (resp []datatypes.User_Customer_Link, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getUserLinks", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) GetUserPreferences(profileName *string, containerKeyname *string) (resp []datatypes.Layout_Profile, err error) {
	params := []interface{}{
		profileName,
		containerKeyname,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getUserPreferences", params, &r.Options, &resp)
	return
}

// Retrieve A portal user's status, which controls overall access to the SoftLayer customer portal and VPN access to the private network.
func (r User_Customer) GetUserStatus() (resp datatypes.User_Customer_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getUserStatus", nil, &r.Options, &resp)
	return
}

// Retrieve the number of CloudLayer Computing Instances that a portal user has access to. Portal users can have restrictions set to limit services for and to perform actions on CloudLayer Computing Instances. You can set these permissions in the portal by clicking the "administrative" then "user admin" links.
func (r User_Customer) GetVirtualGuestCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getVirtualGuestCount", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's accessible CloudLayer Computing Instances. These permissions control which CloudLayer Computing Instances a user has access to in the SoftLayer customer portal.
func (r User_Customer) GetVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "getVirtualGuests", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) InTerminalStatus() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "inTerminalStatus", nil, &r.Options, &resp)
	return
}

// The service initiates an external authentication with the given external authentication vendor. The authentication container and its content will be verified before an attempt is made to initiate an external authentication. [[SoftLayer_Container_User_Customer_External_Binding_Phone|Phone external binding]] container can be used for this service.
//
// This service returns a unique authentication request token. You can use [[SoftLayer_User_Customer::checkExternalAuthenticationStatus|checkExternalAuthenticationStatus]] service to check if the authentication request is complete or not.
func (r User_Customer) InitiateExternalAuthentication(authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp string, err error) {
	params := []interface{}{
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "initiateExternalAuthentication", params, &r.Options, &resp)
	return
}

// Sends password change email to the user containing url that allows the user the change their password. This is the first step when a user wishes to change their password.  The url that is generated contains a one-time use token that is valid for only 24-hours.
func (r User_Customer) InitiatePortalPasswordChange(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "initiatePortalPasswordChange", params, &r.Options, &resp)
	return
}

// A Brand Agent that has permissions to Add Customer Accounts will be able to request the password email be sent to the Master User of a Customer Account created by the same Brand as the agent making the request.
func (r User_Customer) InitiatePortalPasswordChangeByBrandAgent(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "initiatePortalPasswordChangeByBrandAgent", params, &r.Options, &resp)
	return
}

// Send email invitation to a user to join a SoftLayer account and authenticate with OpenIdConnect. Throws an exception on error.
func (r User_Customer) InviteUserToLinkOpenIdConnect(providerType *string, linkedBluemixAccountMatchAlreadyChecked *bool) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		providerType,
		linkedBluemixAccountMatchAlreadyChecked,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "inviteUserToLinkOpenIdConnect", params, &r.Options, &resp)
	return
}

// Portal users are considered master users if they don't have an associated parent user. The only users who don't have parent users are users whose username matches their SoftLayer account name. Master users have special permissions throughout the SoftLayer customer portal.
func (r User_Customer) IsMasterUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "isMasterUser", nil, &r.Options, &resp)
	return
}

// This method is deprecated! SoftLayer Community Forums no longer exist, therefore, any password verified will return false.
//
// Determine if a string is the given user's login password to the SoftLayer community forums.
func (r User_Customer) IsValidForumPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "isValidForumPassword", params, &r.Options, &resp)
	return
}

// Determine if a string is the given user's login password to the SoftLayer customer portal.
func (r User_Customer) IsValidPortalPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "isValidPortalPassword", params, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong> SoftLayer provides a way for users of it's customer portal to recover lost passwords. The lostPassword() method is the first step in this process. Given a valid username and email address, the SoftLayer API will email the address provided with a URL to visit to begin the password recovery process. The last part of this URL is a hash key that's used as an identifier throughout this process. Use this hash key in the [[SoftLayer_User_Customer::setPasswordFromLostPasswordRequest|setPasswordFromLostPasswordRequest]] method to reset a user's password. Password recovery hash keys are valid for 24 hours after they're generated.
func (r User_Customer) LostPassword(username *string, email *string) (resp bool, err error) {
	params := []interface{}{
		username,
		email,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "lostPassword", params, &r.Options, &resp)
	return
}

// The perform external authentication method will authenticate the given external authentication container with an external vendor.  The authentication container and its contents will be verified before an attempt is made to authenticate the contents of the container with an external vendor.
func (r User_Customer) PerformExternalAuthentication(authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "performExternalAuthentication", params, &r.Options, &resp)
	return
}

// Set the password for a user who has an outstanding password request. A user with an outstanding password request will have an unused and unexpired password key.  The password key is part of the url provided to the user in the email sent to the user with information on how to set their password.  The email was generated by the [[SoftLayer_User_Customer::setNewUserPassword|setNewUserPassword]] method. Password recovery keys are valid for 24 hours after they're generated.
//
// User portal passwords must match the following restrictions. Portal passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ + =
// * ...not match your username
// * ...not match your forum password
func (r User_Customer) ProcessPasswordSetRequest(passwordSet *datatypes.Container_User_Customer_PasswordSet, authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp bool, err error) {
	params := []interface{}{
		passwordSet,
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "processPasswordSetRequest", params, &r.Options, &resp)
	return
}

// Remove all hardware from a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. If the current user does not have administrative privileges over this user, an inadequate permissions exception will get thrown.
//
// Users can call this function on child users, but not to themselves. An account's master has access to all users permissions on their account.
func (r User_Customer) RemoveAllHardwareAccessForThisUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeAllHardwareAccessForThisUser", nil, &r.Options, &resp)
	return
}

// Remove all cloud computing instances from a portal user's instance access list. A user's instance access list controls which of an account's computing instance objects a user has access to in the SoftLayer customer portal and API. If the current user does not have administrative privileges over this user, an inadequate permissions exception will get thrown.
//
// Users can call this function on child users, but not to themselves. An account's master has access to all users permissions on their account.
func (r User_Customer) RemoveAllVirtualAccessForThisUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeAllVirtualAccessForThisUser", nil, &r.Options, &resp)
	return
}

// Remove a user's API authentication key, removing that user's access to query the SoftLayer API.
func (r User_Customer) RemoveApiAuthenticationKey(keyId *int) (resp bool, err error) {
	params := []interface{}{
		keyId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeApiAuthenticationKey", params, &r.Options, &resp)
	return
}

// Remove multiple hardware from a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. Hardware does not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user does not has access to the hardware you're attempting remove add then removeBulkHardwareAccess() returns true.
//
// Users can assign hardware access to their child users, but not to themselves. An account's master has access to all hardware on their customer account and can set hardware access for any of the other users on their account.
//
// If the user has full hardware access, then it will provide access to "ALL but passed in" hardware ids.
func (r User_Customer) RemoveBulkHardwareAccess(hardwareIds []int) (resp bool, err error) {
	params := []interface{}{
		hardwareIds,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeBulkHardwareAccess", params, &r.Options, &resp)
	return
}

// Remove multiple permissions from a portal user's permission set. [[Permissions]] control which features in the SoftLayer customer portal and API a user may use. Removing a user's permission will affect that user's portal and API access. removePortalPermission() does not attempt to remove permissions that are not assigned to the user.
//
// Users can assign permissions to their child users, but not to themselves. An account's master has all portal permissions and can set permissions for any of the other users on their account.
//
// Use the [[SoftLayer_User_Customer_CustomerPermission_Permission::getAllObjects]] method to retrieve a list of all permissions available in the SoftLayer customer portal and API. Permissions are removed based on the keyName property of the permission objects within the permissions parameter.
func (r User_Customer) RemoveBulkPortalPermission(permissions []datatypes.User_Customer_CustomerPermission_Permission) (resp bool, err error) {
	params := []interface{}{
		permissions,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeBulkPortalPermission", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) RemoveBulkRoles(roles []datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		roles,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeBulkRoles", params, &r.Options, &resp)
	return
}

// Remove multiple CloudLayer Computing Instances from a portal user's access list. A user's CloudLayer Computing Instance access list controls which of an account's CloudLayer Computing Instance objects a user has access to in the SoftLayer customer portal and API. CloudLayer Computing Instances do not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user does not has access to the CloudLayer Computing Instance you're attempting remove add then removeBulkVirtualGuestAccess() returns true.
//
// Users can assign CloudLayer Computing Instance access to their child users, but not to themselves. An account's master has access to all CloudLayer Computing Instances on their customer account and can set hardware access for any of the other users on their account.
func (r User_Customer) RemoveBulkVirtualGuestAccess(virtualGuestIds []int) (resp bool, err error) {
	params := []interface{}{
		virtualGuestIds,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeBulkVirtualGuestAccess", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) RemoveExternalBinding(externalBinding *datatypes.User_External_Binding) (resp bool, err error) {
	params := []interface{}{
		externalBinding,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeExternalBinding", params, &r.Options, &resp)
	return
}

// Remove hardware from a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. Hardware does not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user does not has access to the hardware you're attempting remove add then removeHardwareAccess() returns true.
//
// Users can assign hardware access to their child users, but not to themselves. An account's master has access to all hardware on their customer account and can set hardware access for any of the other users on their account.
func (r User_Customer) RemoveHardwareAccess(hardwareId *int) (resp bool, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeHardwareAccess", params, &r.Options, &resp)
	return
}

// Remove a permission from a portal user's permission set. [[Permissions]] control which features in the SoftLayer customer portal and API a user may use. Removing a user's permission will affect that user's portal and API access. If the user does not have the permission you're attempting to remove then removePortalPermission() returns true.
//
// Users can assign permissions to their child users, but not to themselves. An account's master has all portal permissions and can set permissions for any of the other users on their account.
//
// Use the [[SoftLayer_User_Customer_CustomerPermission_Permission::getAllObjects]] method to retrieve a list of all permissions available in the SoftLayer customer portal and API. Permissions are removed based on the keyName property of the permission parameter.
func (r User_Customer) RemovePortalPermission(permission *datatypes.User_Customer_CustomerPermission_Permission) (resp bool, err error) {
	params := []interface{}{
		permission,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removePortalPermission", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) RemoveRole(role *datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		role,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeRole", params, &r.Options, &resp)
	return
}

// Remove a CloudLayer Computing Instance from a portal user's access list. A user's CloudLayer Computing Instance access list controls which of an account's computing instances a user has access to in the SoftLayer customer portal and API. CloudLayer Computing Instances do not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user does not has access to the CloudLayer Computing Instance you're attempting remove add then removeVirtualGuestAccess() returns true.
//
// Users can assign CloudLayer Computing Instance access to their child users, but not to themselves. An account's master has access to all CloudLayer Computing Instances on their customer account and can set instance access for any of the other users on their account.
func (r User_Customer) RemoveVirtualGuestAccess(virtualGuestId *int) (resp bool, err error) {
	params := []interface{}{
		virtualGuestId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "removeVirtualGuestAccess", params, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong> Attempt to authenticate a username and password to the SoftLayer customer portal and reset there password. If authentication and password reset is successful then the API returns true.
func (r User_Customer) ResetExpiredPassword(username *string, password *string, newPassword *string, securityQuestionId *int, securityQuestionAnswer *string) (resp bool, err error) {
	params := []interface{}{
		username,
		password,
		newPassword,
		securityQuestionId,
		securityQuestionAnswer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "resetExpiredPassword", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) SamlAuthenticate(accountId *string, samlResponse *string) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		accountId,
		samlResponse,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "samlAuthenticate", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) SamlBeginAuthentication(accountId *int) (resp string, err error) {
	params := []interface{}{
		accountId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "samlBeginAuthentication", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) SamlBeginLogout() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "samlBeginLogout", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) SamlLogout(samlResponse *string) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		samlResponse,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "samlLogout", params, &r.Options, &resp)
	return
}

// Set a user's password via the lost password recovery system, using a password recovery key received in an email generated by the [[SoftLayer_User_Customer::lostPassword|lostPassword]] method. Password recovery keys are valid for 24 hours after they're generated.
//
// User portal passwords must match the following restrictions. Portal passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ + =
// * ...not match your username
// * ...not match your forum password
func (r User_Customer) SetPasswordFromLostPasswordRequest(key *string, password *string, securityAnswers []datatypes.User_Customer_Security_Answer) (resp bool, err error) {
	params := []interface{}{
		key,
		password,
		securityAnswers,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "setPasswordFromLostPasswordRequest", params, &r.Options, &resp)
	return
}

// As master user, calling this api for the IBMid provider type when there is an existing IBMid for the email on the SL account will silently (without sending an invitation email) create a link for the IBMid. If the SoftLayer user is already linked to IBMid, this will reset the existing link. If the IBMid specified by the email of this user, is already used in a link to another user in this account, this call will fail. If there is already an open invitation from this SoftLayer user to this or any IBMid, this call will fail. If there is already an open invitation from some other SoftLayer user in this account to this IBMid, then this call will fail.
func (r User_Customer) SilentlyMigrateUserOpenIdConnect(providerType *string) (resp bool, err error) {
	params := []interface{}{
		providerType,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "silentlyMigrateUserOpenIdConnect", params, &r.Options, &resp)
	return
}

// This method is deprecated! SoftLayer Community Forums no longer exist, therefore, this method will return false.
//
// Update a user's password on the SoftLayer community forums. As with portal passwords, user forum passwords must match the following restrictions. Forum passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ + =
// * ...not match your username
// * ...not match your portal password
// Finally, users can only update their own password.
func (r User_Customer) UpdateForumPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "updateForumPassword", params, &r.Options, &resp)
	return
}

// Update the active status for a notification that the user is subscribed to. A notification along with an active flag can be supplied to update the active status for a particular notification subscription.
func (r User_Customer) UpdateNotificationSubscriber(notificationKeyName *string, active *int) (resp bool, err error) {
	params := []interface{}{
		notificationKeyName,
		active,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "updateNotificationSubscriber", params, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong> Update a user's password on the SoftLayer customer portal. As with forum passwords, user portal passwords must match the following restrictions. Portal passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ + =
// * ...not match your username
// * ...not match your forum password
// Finally, users can only update their own password. An account's master user can update any of their account users' passwords.
func (r User_Customer) UpdatePassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "updatePassword", params, &r.Options, &resp)
	return
}

// Update a user's login security questions and answers on the SoftLayer customer portal. These questions and answers are used to optionally log into the SoftLayer customer portal using two-factor authentication. Each user must have three distinct questions set with a unique answer for each question, and each answer may only contain alphanumeric or the . , - _ ( ) [ ] : ; > < characters. Existing user security questions and answers are deleted before new ones are set, and users may only update their own security questions and answers.
func (r User_Customer) UpdateSecurityAnswers(questions []datatypes.User_Security_Question, answers []string) (resp bool, err error) {
	params := []interface{}{
		questions,
		answers,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "updateSecurityAnswers", params, &r.Options, &resp)
	return
}

// Update a delivery method for a notification that the user is subscribed to. A delivery method keyName along with an active flag can be supplied to update the active status of the delivery methods for the specified notification. Available delivery methods - 'EMAIL'. Available notifications - 'PLANNED_MAINTENANCE', 'UNPLANNED_INCIDENT'.
func (r User_Customer) UpdateSubscriberDeliveryMethod(notificationKeyName *string, deliveryMethodKeyNames []string, active *int) (resp bool, err error) {
	params := []interface{}{
		notificationKeyName,
		deliveryMethodKeyNames,
		active,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "updateSubscriberDeliveryMethod", params, &r.Options, &resp)
	return
}

// Update a user's VPN password on the SoftLayer customer portal. As with portal passwords, VPN passwords must match the following restrictions. VPN passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ =
// * ...not match your username
// * ...not match your forum password
// Finally, users can only update their own VPN password. An account's master user can update any of their account users' VPN passwords.
func (r User_Customer) UpdateVpnPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "updateVpnPassword", params, &r.Options, &resp)
	return
}

// Always call this function to enable changes when manually configuring VPN subnet access.
func (r User_Customer) UpdateVpnUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer", "updateVpnUser", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer) ValidateAuthenticationToken(authenticationToken *datatypes.Container_User_Authentication_Token) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		authenticationToken,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer", "validateAuthenticationToken", params, &r.Options, &resp)
	return
}

// The SoftLayer_User_Customer_ApiAuthentication type contains user's authentication key(s).
type User_Customer_ApiAuthentication struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerApiAuthenticationService returns an instance of the User_Customer_ApiAuthentication SoftLayer service
func GetUserCustomerApiAuthenticationService(sess *session.Session) User_Customer_ApiAuthentication {
	return User_Customer_ApiAuthentication{Session: sess}
}

func (r User_Customer_ApiAuthentication) Id(id int) User_Customer_ApiAuthentication {
	r.Options.Id = &id
	return r
}

func (r User_Customer_ApiAuthentication) Mask(mask string) User_Customer_ApiAuthentication {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_ApiAuthentication) Filter(filter string) User_Customer_ApiAuthentication {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_ApiAuthentication) Limit(limit int) User_Customer_ApiAuthentication {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_ApiAuthentication) Offset(offset int) User_Customer_ApiAuthentication {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Customer_ApiAuthentication) EditObject(templateObject *datatypes.User_Customer_ApiAuthentication) (resp datatypes.User_Customer_ApiAuthentication, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_ApiAuthentication", "editObject", params, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_User_Customer_ApiAuthentication object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_User_Customer_ApiAuthentication service.
func (r User_Customer_ApiAuthentication) GetObject() (resp datatypes.User_Customer_ApiAuthentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_ApiAuthentication", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The user who owns the api authentication key.
func (r User_Customer_ApiAuthentication) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_ApiAuthentication", "getUser", nil, &r.Options, &resp)
	return
}

// Each SoftLayer portal account is assigned a series of permissions that determine what access the user has to functions within the SoftLayer customer portal. This status is reflected in the SoftLayer_User_Customer_Status data type. Permissions differ from user status in that user status applies globally to the portal while user permissions are applied to specific portal functions.
type User_Customer_CustomerPermission_Permission struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerCustomerPermissionPermissionService returns an instance of the User_Customer_CustomerPermission_Permission SoftLayer service
func GetUserCustomerCustomerPermissionPermissionService(sess *session.Session) User_Customer_CustomerPermission_Permission {
	return User_Customer_CustomerPermission_Permission{Session: sess}
}

func (r User_Customer_CustomerPermission_Permission) Id(id int) User_Customer_CustomerPermission_Permission {
	r.Options.Id = &id
	return r
}

func (r User_Customer_CustomerPermission_Permission) Mask(mask string) User_Customer_CustomerPermission_Permission {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_CustomerPermission_Permission) Filter(filter string) User_Customer_CustomerPermission_Permission {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_CustomerPermission_Permission) Limit(limit int) User_Customer_CustomerPermission_Permission {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_CustomerPermission_Permission) Offset(offset int) User_Customer_CustomerPermission_Permission {
	r.Options.Offset = &offset
	return r
}

// Retrieve all available permissions.
func (r User_Customer_CustomerPermission_Permission) GetAllObjects() (resp []datatypes.User_Customer_CustomerPermission_Permission, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_CustomerPermission_Permission", "getAllObjects", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_User_Customer_CustomerPermission_Permission object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_User_Customer_CustomerPermission_Permission service.
func (r User_Customer_CustomerPermission_Permission) GetObject() (resp datatypes.User_Customer_CustomerPermission_Permission, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_CustomerPermission_Permission", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_User_Customer_External_Binding data type contains general information for a single external binding.  This includes the 3rd party vendor, type of binding, and a unique identifier and password that is used to authenticate against the 3rd party service.
type User_Customer_External_Binding struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerExternalBindingService returns an instance of the User_Customer_External_Binding SoftLayer service
func GetUserCustomerExternalBindingService(sess *session.Session) User_Customer_External_Binding {
	return User_Customer_External_Binding{Session: sess}
}

func (r User_Customer_External_Binding) Id(id int) User_Customer_External_Binding {
	r.Options.Id = &id
	return r
}

func (r User_Customer_External_Binding) Mask(mask string) User_Customer_External_Binding {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_External_Binding) Filter(filter string) User_Customer_External_Binding {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_External_Binding) Limit(limit int) User_Customer_External_Binding {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_External_Binding) Offset(offset int) User_Customer_External_Binding {
	r.Options.Offset = &offset
	return r
}

// Delete an external authentication binding.  If the external binding currently has an active billing item associated you will be prevented from deleting the binding.  The alternative method to remove an external authentication binding is to use the service cancellation form.
func (r User_Customer_External_Binding) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "deleteObject", nil, &r.Options, &resp)
	return
}

// Disabling an external binding will allow you to keep the external binding on your SoftLayer account, but will not require you to authentication with our trusted 2 form factor vendor when logging into the SoftLayer customer portal.
//
// You may supply one of the following reason when you disable an external binding:
// *Unspecified
// *TemporarilyUnavailable
// *Lost
// *Stolen
func (r User_Customer_External_Binding) Disable(reason *string) (resp bool, err error) {
	params := []interface{}{
		reason,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "disable", params, &r.Options, &resp)
	return
}

// Enabling an external binding will activate the binding on your account and require you to authenticate with our trusted 3rd party 2 form factor vendor when logging into the SoftLayer customer portal.
//
// Please note that API access will be disabled for users that have an active external binding.
func (r User_Customer_External_Binding) Enable() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "enable", nil, &r.Options, &resp)
	return
}

// Retrieve Attributes of an external authentication binding.
func (r User_Customer_External_Binding) GetAttributes() (resp []datatypes.User_External_Binding_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "getAttributes", nil, &r.Options, &resp)
	return
}

// Retrieve Information regarding the billing item for external authentication.
func (r User_Customer_External_Binding) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve An optional note for identifying the external binding.
func (r User_Customer_External_Binding) GetNote() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "getNote", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding) GetObject() (resp datatypes.User_Customer_External_Binding, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The type of external authentication binding.
func (r User_Customer_External_Binding) GetType() (resp datatypes.User_External_Binding_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer user that the external authentication binding belongs to.
func (r User_Customer_External_Binding) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "getUser", nil, &r.Options, &resp)
	return
}

// Retrieve The vendor of an external authentication binding.
func (r User_Customer_External_Binding) GetVendor() (resp datatypes.User_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "getVendor", nil, &r.Options, &resp)
	return
}

// Update the note of an external binding.  The note is an optional property that is used to store information about a binding.
func (r User_Customer_External_Binding) UpdateNote(text *string) (resp bool, err error) {
	params := []interface{}{
		text,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding", "updateNote", params, &r.Options, &resp)
	return
}

// The SoftLayer_User_Customer_External_Binding_Phone data type contains information about an external binding that uses a phone call, SMS or mobile app for 2 form factor authentication. The external binding information is used when a SoftLayer customer logs into the SoftLayer customer portal or VPN to authenticate them against a trusted 3rd party, in this case using a mobile phone, mobile phone application or land-line phone.
//
// SoftLayer users with an active external binding will be prohibited from using the API for security reasons.
type User_Customer_External_Binding_Phone struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerExternalBindingPhoneService returns an instance of the User_Customer_External_Binding_Phone SoftLayer service
func GetUserCustomerExternalBindingPhoneService(sess *session.Session) User_Customer_External_Binding_Phone {
	return User_Customer_External_Binding_Phone{Session: sess}
}

func (r User_Customer_External_Binding_Phone) Id(id int) User_Customer_External_Binding_Phone {
	r.Options.Id = &id
	return r
}

func (r User_Customer_External_Binding_Phone) Mask(mask string) User_Customer_External_Binding_Phone {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_External_Binding_Phone) Filter(filter string) User_Customer_External_Binding_Phone {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_External_Binding_Phone) Limit(limit int) User_Customer_External_Binding_Phone {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_External_Binding_Phone) Offset(offset int) User_Customer_External_Binding_Phone {
	r.Options.Offset = &offset
	return r
}

// Return a phone validation result.
func (r User_Customer_External_Binding_Phone) CheckPhoneValidationResult(token *string) (resp bool, err error) {
	params := []interface{}{
		token,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "checkPhoneValidationResult", params, &r.Options, &resp)
	return
}

// Delete an external authentication binding.  If the external binding currently has an active billing item associated you will be prevented from deleting the binding.  The alternative method to remove an external authentication binding is to use the service cancellation form.
func (r User_Customer_External_Binding_Phone) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "deleteObject", nil, &r.Options, &resp)
	return
}

// Disabling an external binding will allow you to keep the external binding on your SoftLayer account, but will not require you to authentication with our trusted 2 form factor vendor when logging into the SoftLayer customer portal.
//
// You may supply one of the following reason when you disable an external binding:
// *Unspecified
// *TemporarilyUnavailable
// *Lost
// *Stolen
func (r User_Customer_External_Binding_Phone) Disable(reason *string) (resp bool, err error) {
	params := []interface{}{
		reason,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "disable", params, &r.Options, &resp)
	return
}

// Enabling an external binding will activate the binding on your account and require you to authenticate with our trusted 3rd party 2 form factor vendor when logging into the SoftLayer customer portal.
//
// Please note that API access will be disabled for users that have an active external binding.
func (r User_Customer_External_Binding_Phone) Enable() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "enable", nil, &r.Options, &resp)
	return
}

// This service returns key names of all available authentication modes. See [[SoftLayer_Container_User_Customer_External_Binding_Phone_Mode|authentication mode]] container for details.
func (r User_Customer_External_Binding_Phone) GetAllAuthenticationModes() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getAllAuthenticationModes", nil, &r.Options, &resp)
	return
}

// This service returns key names of all available authentication modes. Refer to [[SoftLayer_User_Customer_External_Binding_Phone::getAllAuthenticationModes|getAllAuthenticationModes]] to retrieve authentication mode key names.
func (r User_Customer_External_Binding_Phone) GetAllAuthenticationPinModes(authenticationModeKeyName *string) (resp []string, err error) {
	params := []interface{}{
		authenticationModeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getAllAuthenticationPinModes", params, &r.Options, &resp)
	return
}

// Retrieve Attributes of an external authentication binding.
func (r User_Customer_External_Binding_Phone) GetAttributes() (resp []datatypes.User_External_Binding_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getAttributes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding_Phone) GetAuthenticationMode() (resp datatypes.Container_User_Customer_External_Binding_Phone_Mode, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getAuthenticationMode", nil, &r.Options, &resp)
	return
}

// Retrieve Information regarding the billing item for external authentication.
func (r User_Customer_External_Binding_Phone) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The current external binding status. It can be "ACTIVE" or "BLOCKED".
func (r User_Customer_External_Binding_Phone) GetBindingStatus() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getBindingStatus", nil, &r.Options, &resp)
	return
}

// Retrieve An optional note for identifying the external binding.
func (r User_Customer_External_Binding_Phone) GetNote() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getNote", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding_Phone) GetObject() (resp datatypes.User_Customer_External_Binding_Phone, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getObject", nil, &r.Options, &resp)
	return
}

// Some vendor's mobile app requires an activation code. Use this method to get an activation data.
func (r User_Customer_External_Binding_Phone) GetPhoneAppActivationCode() (resp []datatypes.User_External_Binding_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getPhoneAppActivationCode", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding_Phone) GetPhoneData() (resp []datatypes.Container_User_Data_Phone, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getPhoneData", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer_External_Binding_Phone) GetPinLength() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getPinLength", nil, &r.Options, &resp)
	return
}

// Retrieve The type of external authentication binding.
func (r User_Customer_External_Binding_Phone) GetType() (resp datatypes.User_External_Binding_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer user that the external authentication binding belongs to.
func (r User_Customer_External_Binding_Phone) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getUser", nil, &r.Options, &resp)
	return
}

// Retrieve The vendor of an external authentication binding.
func (r User_Customer_External_Binding_Phone) GetVendor() (resp datatypes.User_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "getVendor", nil, &r.Options, &resp)
	return
}

// Initiates a phone validation requests and returns a unique token. Use [[SoftLayer_User_Customer_External_Binding_Phone::checkPhoneValidationResult|checkPhoneValidationResult]] to find the phone validation result.
func (r User_Customer_External_Binding_Phone) RequestPhoneValidation(phoneData *datatypes.Container_User_Data_Phone) (resp string, err error) {
	params := []interface{}{
		phoneData,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "requestPhoneValidation", params, &r.Options, &resp)
	return
}

// This service allow you to change your phone authentication mode. See [[SoftLayer_Container_User_Customer_External_Binding_Phone_Mode|authentication mode]] container for available modes.
func (r User_Customer_External_Binding_Phone) UpdateAuthenticationMode(mode *datatypes.Container_User_Customer_External_Binding_Phone_Mode) (resp bool, err error) {
	params := []interface{}{
		mode,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "updateAuthenticationMode", params, &r.Options, &resp)
	return
}

// Update the note of an external binding.  The note is an optional property that is used to store information about a binding.
func (r User_Customer_External_Binding_Phone) UpdateNote(text *string) (resp bool, err error) {
	params := []interface{}{
		text,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "updateNote", params, &r.Options, &resp)
	return
}

// Phone external binding supports a primary and a backup phone number. You can use this method to update your phone number used for the phone authentication. You can provide an array of [[SoftLayer_Container_User_Data_Phone|User Phone]] objects. You have to mark one as the primary phone number by setting "phoneType" to "PRIMARY".
//
//
// *countryCode: Country code number for the phone number. Default: 1 (United States & Canada +1)
// *phone: Phone number that 2 Form Factor system will call or text for user authentication. The phone number format must match the format selected in the Country Code.
// *extension: Specify the extension that will be dialed after the call is answered. Digits, commas, *, and # are allowed.  Commas can be used for a one second pause to navigate phone system menus.
// *phoneType: Specify the primary and backup phone number by setting this value to "PRIMARY" or "BACKUP". If omitted, it will be considered to be the primary phone number. If you are passing two Phone objects, you must specify the phone type of each phone number.
//
//
func (r User_Customer_External_Binding_Phone) UpdatePhone(phoneData []datatypes.Container_User_Data_Phone) (resp bool, err error) {
	params := []interface{}{
		phoneData,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Phone", "updatePhone", params, &r.Options, &resp)
	return
}

// The SoftLayer_User_Customer_External_Binding_Totp data type contains information about a single time-based one time password external binding.  The external binding information is used when a SoftLayer customer logs into the SoftLayer customer portal to authenticate them.
//
// The information provided by this external binding data type includes:
// * The type of credential
// * The current state of the credential
// ** Active
// ** Inactive
//
//
// SoftLayer users with an active external binding will be prohibited from using the API for security reasons.
type User_Customer_External_Binding_Totp struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerExternalBindingTotpService returns an instance of the User_Customer_External_Binding_Totp SoftLayer service
func GetUserCustomerExternalBindingTotpService(sess *session.Session) User_Customer_External_Binding_Totp {
	return User_Customer_External_Binding_Totp{Session: sess}
}

func (r User_Customer_External_Binding_Totp) Id(id int) User_Customer_External_Binding_Totp {
	r.Options.Id = &id
	return r
}

func (r User_Customer_External_Binding_Totp) Mask(mask string) User_Customer_External_Binding_Totp {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_External_Binding_Totp) Filter(filter string) User_Customer_External_Binding_Totp {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_External_Binding_Totp) Limit(limit int) User_Customer_External_Binding_Totp {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_External_Binding_Totp) Offset(offset int) User_Customer_External_Binding_Totp {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Customer_External_Binding_Totp) Activate() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "activate", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding_Totp) Deactivate() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "deactivate", nil, &r.Options, &resp)
	return
}

// Delete an external authentication binding.  If the external binding currently has an active billing item associated you will be prevented from deleting the binding.  The alternative method to remove an external authentication binding is to use the service cancellation form.
func (r User_Customer_External_Binding_Totp) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "deleteObject", nil, &r.Options, &resp)
	return
}

// Disabling an external binding will allow you to keep the external binding on your SoftLayer account, but will not require you to authentication with our trusted 2 form factor vendor when logging into the SoftLayer customer portal.
//
// You may supply one of the following reason when you disable an external binding:
// *Unspecified
// *TemporarilyUnavailable
// *Lost
// *Stolen
func (r User_Customer_External_Binding_Totp) Disable(reason *string) (resp bool, err error) {
	params := []interface{}{
		reason,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "disable", params, &r.Options, &resp)
	return
}

// Enabling an external binding will activate the binding on your account and require you to authenticate with our trusted 3rd party 2 form factor vendor when logging into the SoftLayer customer portal.
//
// Please note that API access will be disabled for users that have an active external binding.
func (r User_Customer_External_Binding_Totp) Enable() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "enable", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding_Totp) GenerateSecretKey() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "generateSecretKey", nil, &r.Options, &resp)
	return
}

// Retrieve Attributes of an external authentication binding.
func (r User_Customer_External_Binding_Totp) GetAttributes() (resp []datatypes.User_External_Binding_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "getAttributes", nil, &r.Options, &resp)
	return
}

// Retrieve Information regarding the billing item for external authentication.
func (r User_Customer_External_Binding_Totp) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve An optional note for identifying the external binding.
func (r User_Customer_External_Binding_Totp) GetNote() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "getNote", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding_Totp) GetObject() (resp datatypes.User_Customer_External_Binding_Totp, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The type of external authentication binding.
func (r User_Customer_External_Binding_Totp) GetType() (resp datatypes.User_External_Binding_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer user that the external authentication binding belongs to.
func (r User_Customer_External_Binding_Totp) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "getUser", nil, &r.Options, &resp)
	return
}

// Retrieve The vendor of an external authentication binding.
func (r User_Customer_External_Binding_Totp) GetVendor() (resp datatypes.User_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "getVendor", nil, &r.Options, &resp)
	return
}

// Update the note of an external binding.  The note is an optional property that is used to store information about a binding.
func (r User_Customer_External_Binding_Totp) UpdateNote(text *string) (resp bool, err error) {
	params := []interface{}{
		text,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Totp", "updateNote", params, &r.Options, &resp)
	return
}

// The SoftLayer_User_Customer_External_Binding_Vendor data type contains information for a single external binding vendor.  This information includes a user friendly vendor name, a unique version of the vendor name, and a unique internal identifier that can be used when creating a new external binding.
type User_Customer_External_Binding_Vendor struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerExternalBindingVendorService returns an instance of the User_Customer_External_Binding_Vendor SoftLayer service
func GetUserCustomerExternalBindingVendorService(sess *session.Session) User_Customer_External_Binding_Vendor {
	return User_Customer_External_Binding_Vendor{Session: sess}
}

func (r User_Customer_External_Binding_Vendor) Id(id int) User_Customer_External_Binding_Vendor {
	r.Options.Id = &id
	return r
}

func (r User_Customer_External_Binding_Vendor) Mask(mask string) User_Customer_External_Binding_Vendor {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_External_Binding_Vendor) Filter(filter string) User_Customer_External_Binding_Vendor {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_External_Binding_Vendor) Limit(limit int) User_Customer_External_Binding_Vendor {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_External_Binding_Vendor) Offset(offset int) User_Customer_External_Binding_Vendor {
	r.Options.Offset = &offset
	return r
}

// getAllObjects() will return a list of the available external binding vendors that SoftLayer supports.  Use this list to select the appropriate vendor when creating a new external binding.
func (r User_Customer_External_Binding_Vendor) GetAllObjects() (resp []datatypes.User_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Vendor", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding_Vendor) GetObject() (resp datatypes.User_Customer_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Vendor", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_User_Customer_External_Binding_Verisign data type contains information about a single VeriSign external binding.  The external binding information is used when a SoftLayer customer logs into the SoftLayer customer portal to authenticate them against a 3rd party, in this case VeriSign.
//
// The information provided by the VeriSign external binding data type includes:
// * The type of credential
// * The current state of the credential
// ** Enabled
// ** Disabled
// ** Locked
// * The credential's expiration date
// * The last time the credential was updated
//
//
// SoftLayer users with an active external binding will be prohibited from using the API for security reasons.
type User_Customer_External_Binding_Verisign struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerExternalBindingVerisignService returns an instance of the User_Customer_External_Binding_Verisign SoftLayer service
func GetUserCustomerExternalBindingVerisignService(sess *session.Session) User_Customer_External_Binding_Verisign {
	return User_Customer_External_Binding_Verisign{Session: sess}
}

func (r User_Customer_External_Binding_Verisign) Id(id int) User_Customer_External_Binding_Verisign {
	r.Options.Id = &id
	return r
}

func (r User_Customer_External_Binding_Verisign) Mask(mask string) User_Customer_External_Binding_Verisign {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_External_Binding_Verisign) Filter(filter string) User_Customer_External_Binding_Verisign {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_External_Binding_Verisign) Limit(limit int) User_Customer_External_Binding_Verisign {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_External_Binding_Verisign) Offset(offset int) User_Customer_External_Binding_Verisign {
	r.Options.Offset = &offset
	return r
}

// Delete a VeriSign external binding.  The only VeriSign external binding that can be deleted through this method is the free VeriSign external binding for the master user of a SoftLayer account. All other external bindings must be canceled using the SoftLayer service cancellation form.
//
// When a VeriSign external binding is deleted the credential is deactivated in VeriSign's system for use on the SoftLayer site and the $0 billing item associated with the free VeriSign external binding is cancelled.
func (r User_Customer_External_Binding_Verisign) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "deleteObject", nil, &r.Options, &resp)
	return
}

// Disabling an external binding will allow you to keep the external binding on your SoftLayer account, but will not require you to authentication with our trusted 2 form factor vendor when logging into the SoftLayer customer portal.
//
// You may supply one of the following reason when you disable an external binding:
// *Unspecified
// *TemporarilyUnavailable
// *Lost
// *Stolen
func (r User_Customer_External_Binding_Verisign) Disable(reason *string) (resp bool, err error) {
	params := []interface{}{
		reason,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "disable", params, &r.Options, &resp)
	return
}

// Enabling an external binding will activate the binding on your account and require you to authenticate with our trusted 3rd party 2 form factor vendor when logging into the SoftLayer customer portal.
//
// Please note that API access will be disabled for users that have an active external binding.
func (r User_Customer_External_Binding_Verisign) Enable() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "enable", nil, &r.Options, &resp)
	return
}

// An activation code is required when provisioning a new mobile credential from Verisign.  This method will return the required activation code.
func (r User_Customer_External_Binding_Verisign) GetActivationCodeForMobileClient() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getActivationCodeForMobileClient", nil, &r.Options, &resp)
	return
}

// Retrieve Attributes of an external authentication binding.
func (r User_Customer_External_Binding_Verisign) GetAttributes() (resp []datatypes.User_External_Binding_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getAttributes", nil, &r.Options, &resp)
	return
}

// Retrieve Information regarding the billing item for external authentication.
func (r User_Customer_External_Binding_Verisign) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve The date that a VeriSign credential expires.
func (r User_Customer_External_Binding_Verisign) GetCredentialExpirationDate() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getCredentialExpirationDate", nil, &r.Options, &resp)
	return
}

// Retrieve The last time a VeriSign credential was updated.
func (r User_Customer_External_Binding_Verisign) GetCredentialLastUpdateDate() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getCredentialLastUpdateDate", nil, &r.Options, &resp)
	return
}

// Retrieve The current state of a VeriSign credential. This can be 'Enabled', 'Disabled', or 'Locked'.
func (r User_Customer_External_Binding_Verisign) GetCredentialState() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getCredentialState", nil, &r.Options, &resp)
	return
}

// Retrieve The type of VeriSign credential. This can be either 'Hardware' or 'Software'.
func (r User_Customer_External_Binding_Verisign) GetCredentialType() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getCredentialType", nil, &r.Options, &resp)
	return
}

// Retrieve An optional note for identifying the external binding.
func (r User_Customer_External_Binding_Verisign) GetNote() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getNote", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_External_Binding_Verisign) GetObject() (resp datatypes.User_Customer_External_Binding_Verisign, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The type of external authentication binding.
func (r User_Customer_External_Binding_Verisign) GetType() (resp datatypes.User_External_Binding_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer user that the external authentication binding belongs to.
func (r User_Customer_External_Binding_Verisign) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getUser", nil, &r.Options, &resp)
	return
}

// Retrieve The vendor of an external authentication binding.
func (r User_Customer_External_Binding_Verisign) GetVendor() (resp datatypes.User_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "getVendor", nil, &r.Options, &resp)
	return
}

// If a VeriSign credential becomes locked because of too many failed login attempts the unlock method can be used to unlock a VeriSign credential. As a security precaution a valid security code generated by the credential will be required before the credential is unlocked.
func (r User_Customer_External_Binding_Verisign) Unlock(securityCode *string) (resp bool, err error) {
	params := []interface{}{
		securityCode,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "unlock", params, &r.Options, &resp)
	return
}

// Update the note of an external binding.  The note is an optional property that is used to store information about a binding.
func (r User_Customer_External_Binding_Verisign) UpdateNote(text *string) (resp bool, err error) {
	params := []interface{}{
		text,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "updateNote", params, &r.Options, &resp)
	return
}

// Validate the user id and VeriSign credential id used to create an external authentication binding.
func (r User_Customer_External_Binding_Verisign) ValidateCredentialId(userId *int, externalId *string) (resp bool, err error) {
	params := []interface{}{
		userId,
		externalId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_External_Binding_Verisign", "validateCredentialId", params, &r.Options, &resp)
	return
}

// no documentation yet
type User_Customer_Invitation struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerInvitationService returns an instance of the User_Customer_Invitation SoftLayer service
func GetUserCustomerInvitationService(sess *session.Session) User_Customer_Invitation {
	return User_Customer_Invitation{Session: sess}
}

func (r User_Customer_Invitation) Id(id int) User_Customer_Invitation {
	r.Options.Id = &id
	return r
}

func (r User_Customer_Invitation) Mask(mask string) User_Customer_Invitation {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_Invitation) Filter(filter string) User_Customer_Invitation {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_Invitation) Limit(limit int) User_Customer_Invitation {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_Invitation) Offset(offset int) User_Customer_Invitation {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Customer_Invitation) GetObject() (resp datatypes.User_Customer_Invitation, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Invitation", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer_Invitation) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Invitation", "getUser", nil, &r.Options, &resp)
	return
}

// This class represents a mobile device belonging to a user.  The device can be a phone, tablet, or possibly even some Android based net books.  The purpose is to tie just enough info with the device and the user to enable push notifications through non-softlayer entities (Google, Apple, RIM).
type User_Customer_MobileDevice struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerMobileDeviceService returns an instance of the User_Customer_MobileDevice SoftLayer service
func GetUserCustomerMobileDeviceService(sess *session.Session) User_Customer_MobileDevice {
	return User_Customer_MobileDevice{Session: sess}
}

func (r User_Customer_MobileDevice) Id(id int) User_Customer_MobileDevice {
	r.Options.Id = &id
	return r
}

func (r User_Customer_MobileDevice) Mask(mask string) User_Customer_MobileDevice {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_MobileDevice) Filter(filter string) User_Customer_MobileDevice {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_MobileDevice) Limit(limit int) User_Customer_MobileDevice {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_MobileDevice) Offset(offset int) User_Customer_MobileDevice {
	r.Options.Offset = &offset
	return r
}

// Create a new mobile device association for a user.
func (r User_Customer_MobileDevice) CreateObject(templateObject *datatypes.User_Customer_MobileDevice) (resp datatypes.User_Customer_MobileDevice, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "createObject", params, &r.Options, &resp)
	return
}

// Delete a mobile device association for a user.
func (r User_Customer_MobileDevice) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "deleteObject", nil, &r.Options, &resp)
	return
}

// Edit the object by passing in a modified instance of the object
func (r User_Customer_MobileDevice) EditObject(templateObject *datatypes.User_Customer_MobileDevice) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve Notification subscriptions available to a mobile device.
func (r User_Customer_MobileDevice) GetAvailablePushNotificationSubscriptions() (resp []datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "getAvailablePushNotificationSubscriptions", nil, &r.Options, &resp)
	return
}

// Retrieve The user this mobile device belongs to.
func (r User_Customer_MobileDevice) GetCustomer() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "getCustomer", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_MobileDevice) GetObject() (resp datatypes.User_Customer_MobileDevice, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The operating system this device is using
func (r User_Customer_MobileDevice) GetOperatingSystem() (resp datatypes.User_Customer_MobileDevice_OperatingSystem, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "getOperatingSystem", nil, &r.Options, &resp)
	return
}

// Retrieve Notification subscriptions attached to a mobile device.
func (r User_Customer_MobileDevice) GetPushNotificationSubscriptions() (resp []datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "getPushNotificationSubscriptions", nil, &r.Options, &resp)
	return
}

// Retrieve The type of device this user is using
func (r User_Customer_MobileDevice) GetType() (resp datatypes.User_Customer_MobileDevice_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice", "getType", nil, &r.Options, &resp)
	return
}

// This class represents the mobile operating system installed on a user's registered mobile device. It assists us when determining the how to get a push notification to the user.
type User_Customer_MobileDevice_OperatingSystem struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerMobileDeviceOperatingSystemService returns an instance of the User_Customer_MobileDevice_OperatingSystem SoftLayer service
func GetUserCustomerMobileDeviceOperatingSystemService(sess *session.Session) User_Customer_MobileDevice_OperatingSystem {
	return User_Customer_MobileDevice_OperatingSystem{Session: sess}
}

func (r User_Customer_MobileDevice_OperatingSystem) Id(id int) User_Customer_MobileDevice_OperatingSystem {
	r.Options.Id = &id
	return r
}

func (r User_Customer_MobileDevice_OperatingSystem) Mask(mask string) User_Customer_MobileDevice_OperatingSystem {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_MobileDevice_OperatingSystem) Filter(filter string) User_Customer_MobileDevice_OperatingSystem {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_MobileDevice_OperatingSystem) Limit(limit int) User_Customer_MobileDevice_OperatingSystem {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_MobileDevice_OperatingSystem) Offset(offset int) User_Customer_MobileDevice_OperatingSystem {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Customer_MobileDevice_OperatingSystem) GetAllObjects() (resp []datatypes.User_Customer_MobileDevice_OperatingSystem, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice_OperatingSystem", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_MobileDevice_OperatingSystem) GetObject() (resp datatypes.User_Customer_MobileDevice_OperatingSystem, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice_OperatingSystem", "getObject", nil, &r.Options, &resp)
	return
}

// Describes a supported class of mobile device. In this the word class is used in the context of classes of consumer electronic devices, the two most prominent examples being mobile phones and tablets.
type User_Customer_MobileDevice_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerMobileDeviceTypeService returns an instance of the User_Customer_MobileDevice_Type SoftLayer service
func GetUserCustomerMobileDeviceTypeService(sess *session.Session) User_Customer_MobileDevice_Type {
	return User_Customer_MobileDevice_Type{Session: sess}
}

func (r User_Customer_MobileDevice_Type) Id(id int) User_Customer_MobileDevice_Type {
	r.Options.Id = &id
	return r
}

func (r User_Customer_MobileDevice_Type) Mask(mask string) User_Customer_MobileDevice_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_MobileDevice_Type) Filter(filter string) User_Customer_MobileDevice_Type {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_MobileDevice_Type) Limit(limit int) User_Customer_MobileDevice_Type {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_MobileDevice_Type) Offset(offset int) User_Customer_MobileDevice_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Customer_MobileDevice_Type) GetAllObjects() (resp []datatypes.User_Customer_MobileDevice_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_MobileDevice_Type) GetObject() (resp datatypes.User_Customer_MobileDevice_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_MobileDevice_Type", "getObject", nil, &r.Options, &resp)
	return
}

// The Customer_Notification_Hardware object stores links between customers and the hardware devices they wish to monitor.  This link is not enough, the user must be sure to also create SoftLayer_Network_Monitor_Version1_Query_Host instance with the response action set to "notify users" in order for the users linked to that hardware object to be notified on failure.
type User_Customer_Notification_Hardware struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerNotificationHardwareService returns an instance of the User_Customer_Notification_Hardware SoftLayer service
func GetUserCustomerNotificationHardwareService(sess *session.Session) User_Customer_Notification_Hardware {
	return User_Customer_Notification_Hardware{Session: sess}
}

func (r User_Customer_Notification_Hardware) Id(id int) User_Customer_Notification_Hardware {
	r.Options.Id = &id
	return r
}

func (r User_Customer_Notification_Hardware) Mask(mask string) User_Customer_Notification_Hardware {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_Notification_Hardware) Filter(filter string) User_Customer_Notification_Hardware {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_Notification_Hardware) Limit(limit int) User_Customer_Notification_Hardware {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_Notification_Hardware) Offset(offset int) User_Customer_Notification_Hardware {
	r.Options.Offset = &offset
	return r
}

// Passing in an unsaved instances of a Customer_Notification_Hardware object into this function will create the object and return the results to the user.
func (r User_Customer_Notification_Hardware) CreateObject(templateObject *datatypes.User_Customer_Notification_Hardware) (resp datatypes.User_Customer_Notification_Hardware, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Hardware", "createObject", params, &r.Options, &resp)
	return
}

// Passing in a collection of unsaved instances of Customer_Notification_Hardware objects into this function will create all objects and return the results to the user.
func (r User_Customer_Notification_Hardware) CreateObjects(templateObjects []datatypes.User_Customer_Notification_Hardware) (resp []datatypes.Dns_Domain, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Hardware", "createObjects", params, &r.Options, &resp)
	return
}

// Like any other API object, the customer notification objects can be deleted by passing an instance of them into this function.  The ID on the object must be set.
func (r User_Customer_Notification_Hardware) DeleteObjects(templateObjects []datatypes.User_Customer_Notification_Hardware) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Hardware", "deleteObjects", params, &r.Options, &resp)
	return
}

// This method returns all Customer_Notification_Hardware objects associated with the passed in hardware ID as long as that hardware ID is owned by the current user's account.
//
// This behavior can also be accomplished by simply tapping monitoringUserNotification on the Hardware_Server object.
func (r User_Customer_Notification_Hardware) FindByHardwareId(hardwareId *int) (resp []datatypes.User_Customer_Notification_Hardware, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Hardware", "findByHardwareId", params, &r.Options, &resp)
	return
}

// Retrieve The hardware object that will be monitored.
func (r User_Customer_Notification_Hardware) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Hardware", "getHardware", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_User_Customer_Notification_Hardware object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_User_Customer_Notification_Hardware service. You can only retrieve hardware notifications attached to hardware and users that belong to your account
func (r User_Customer_Notification_Hardware) GetObject() (resp datatypes.User_Customer_Notification_Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Hardware", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The user that will be notified when the associated hardware object fails a monitoring instance.
func (r User_Customer_Notification_Hardware) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Hardware", "getUser", nil, &r.Options, &resp)
	return
}

// The SoftLayer_User_Customer_Notification_Virtual_Guest object stores links between customers and the virtual guests they wish to monitor.  This link is not enough, the user must be sure to also create SoftLayer_Network_Monitor_Version1_Query_Host instance with the response action set to "notify users" in order for the users linked to that hardware object to be notified on failure.
type User_Customer_Notification_Virtual_Guest struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerNotificationVirtualGuestService returns an instance of the User_Customer_Notification_Virtual_Guest SoftLayer service
func GetUserCustomerNotificationVirtualGuestService(sess *session.Session) User_Customer_Notification_Virtual_Guest {
	return User_Customer_Notification_Virtual_Guest{Session: sess}
}

func (r User_Customer_Notification_Virtual_Guest) Id(id int) User_Customer_Notification_Virtual_Guest {
	r.Options.Id = &id
	return r
}

func (r User_Customer_Notification_Virtual_Guest) Mask(mask string) User_Customer_Notification_Virtual_Guest {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_Notification_Virtual_Guest) Filter(filter string) User_Customer_Notification_Virtual_Guest {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_Notification_Virtual_Guest) Limit(limit int) User_Customer_Notification_Virtual_Guest {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_Notification_Virtual_Guest) Offset(offset int) User_Customer_Notification_Virtual_Guest {
	r.Options.Offset = &offset
	return r
}

// Passing in an unsaved instance of a SoftLayer_Customer_Notification_Virtual_Guest object into this function will create the object and return the results to the user.
func (r User_Customer_Notification_Virtual_Guest) CreateObject(templateObject *datatypes.User_Customer_Notification_Virtual_Guest) (resp datatypes.User_Customer_Notification_Virtual_Guest, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Virtual_Guest", "createObject", params, &r.Options, &resp)
	return
}

// Passing in a collection of unsaved instances of SoftLayer_Customer_Notification_Virtual_Guest objects into this function will create all objects and return the results to the user.
func (r User_Customer_Notification_Virtual_Guest) CreateObjects(templateObjects []datatypes.User_Customer_Notification_Virtual_Guest) (resp []datatypes.User_Customer_Notification_Virtual_Guest, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Virtual_Guest", "createObjects", params, &r.Options, &resp)
	return
}

// Like any other API object, the customer notification objects can be deleted by passing an instance of them into this function.  The ID on the object must be set.
func (r User_Customer_Notification_Virtual_Guest) DeleteObjects(templateObjects []datatypes.User_Customer_Notification_Virtual_Guest) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Virtual_Guest", "deleteObjects", params, &r.Options, &resp)
	return
}

// This method returns all SoftLayer_User_Customer_Notification_Virtual_Guest objects associated with the passed in ID as long as that hardware ID is owned by the current user's account.
//
// This behavior can also be accomplished by simply tapping monitoringUserNotification on the Virtual_Guest object.
func (r User_Customer_Notification_Virtual_Guest) FindByGuestId(id *int) (resp []datatypes.User_Customer_Notification_Virtual_Guest, err error) {
	params := []interface{}{
		id,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Virtual_Guest", "findByGuestId", params, &r.Options, &resp)
	return
}

// Retrieve The virtual guest object that will be monitored.
func (r User_Customer_Notification_Virtual_Guest) GetGuest() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Virtual_Guest", "getGuest", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_User_Customer_Notification_Virtual_Guest object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_User_Customer_Notification_Virtual_Guest service. You can only retrieve guest notifications attached to virtual guests and users that belong to your account
func (r User_Customer_Notification_Virtual_Guest) GetObject() (resp datatypes.User_Customer_Notification_Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Virtual_Guest", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The user that will be notified when the associated virtual guest object fails a monitoring instance.
func (r User_Customer_Notification_Virtual_Guest) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Notification_Virtual_Guest", "getUser", nil, &r.Options, &resp)
	return
}

// no documentation yet
type User_Customer_OpenIdConnect struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerOpenIdConnectService returns an instance of the User_Customer_OpenIdConnect SoftLayer service
func GetUserCustomerOpenIdConnectService(sess *session.Session) User_Customer_OpenIdConnect {
	return User_Customer_OpenIdConnect{Session: sess}
}

func (r User_Customer_OpenIdConnect) Id(id int) User_Customer_OpenIdConnect {
	r.Options.Id = &id
	return r
}

func (r User_Customer_OpenIdConnect) Mask(mask string) User_Customer_OpenIdConnect {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_OpenIdConnect) Filter(filter string) User_Customer_OpenIdConnect {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_OpenIdConnect) Limit(limit int) User_Customer_OpenIdConnect {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_OpenIdConnect) Offset(offset int) User_Customer_OpenIdConnect {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Customer_OpenIdConnect) AcknowledgeSupportPolicy() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "acknowledgeSupportPolicy", nil, &r.Options, &resp)
	return
}

// Create a user's API authentication key, allowing that user access to query the SoftLayer API. addApiAuthenticationKey() returns the users new API key. Each portal user is allowed a maximum of two API keys.
func (r User_Customer_OpenIdConnect) AddApiAuthenticationKey() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addApiAuthenticationKey", nil, &r.Options, &resp)
	return
}

// Add multiple hardware to a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. Hardware does not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. addBulkHardwareAccess() does not attempt to add hardware access if the given user already has access to that hardware object.
//
// Users can assign hardware access to their child users, but not to themselves. An account's master has access to all hardware on their customer account and can set hardware access for any of the other users on their account.
func (r User_Customer_OpenIdConnect) AddBulkHardwareAccess(hardwareIds []int) (resp bool, err error) {
	params := []interface{}{
		hardwareIds,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addBulkHardwareAccess", params, &r.Options, &resp)
	return
}

// Add multiple permissions to a portal user's permission set. [[Permissions]] control which features in the SoftLayer customer portal and API a user may use. addBulkPortalPermission() does not attempt to add permissions already assigned to the user.
//
// Users can assign permissions to their child users, but not to themselves. An account's master has all portal permissions and can set permissions for any of the other users on their account.
//
// Use the [[SoftLayer_User_Customer_CustomerPermission_Permission::getAllObjects]] method to retrieve a list of all permissions available in the SoftLayer customer portal and API. Permissions are removed based on the keyName property of the permission objects within the permissions parameter.
func (r User_Customer_OpenIdConnect) AddBulkPortalPermission(permissions []datatypes.User_Customer_CustomerPermission_Permission) (resp bool, err error) {
	params := []interface{}{
		permissions,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addBulkPortalPermission", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) AddBulkRoles(roles []datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		roles,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addBulkRoles", params, &r.Options, &resp)
	return
}

// Add multiple CloudLayer Computing Instances to a portal user's access list. A user's CloudLayer Computing Instance access list controls which of an account's CloudLayer Computing Instance objects a user has access to in the SoftLayer customer portal and API. CloudLayer Computing Instances do not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. addBulkVirtualGuestAccess() does not attempt to add CloudLayer Computing Instance access if the given user already has access to that CloudLayer Computing Instance object.
//
// Users can assign CloudLayer Computing Instance access to their child users, but not to themselves. An account's master has access to all CloudLayer Computing Instances on their customer account and can set CloudLayer Computing Instance access for any of the other users on their account.
func (r User_Customer_OpenIdConnect) AddBulkVirtualGuestAccess(virtualGuestIds []int) (resp bool, err error) {
	params := []interface{}{
		virtualGuestIds,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addBulkVirtualGuestAccess", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) AddExternalBinding(externalBinding *datatypes.User_External_Binding) (resp datatypes.User_Customer_External_Binding, err error) {
	params := []interface{}{
		externalBinding,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addExternalBinding", params, &r.Options, &resp)
	return
}

// Add hardware to a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. Hardware does not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user already has access to the hardware you're attempting to add then addHardwareAccess() returns true.
//
// Users can assign hardware access to their child users, but not to themselves. An account's master has access to all hardware on their customer account and can set hardware access for any of the other users on their account.
func (r User_Customer_OpenIdConnect) AddHardwareAccess(hardwareId *int) (resp bool, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addHardwareAccess", params, &r.Options, &resp)
	return
}

// Create a notification subscription record for the user. If a subscription record exists for the notification, the record will be set to active, if currently inactive.
func (r User_Customer_OpenIdConnect) AddNotificationSubscriber(notificationKeyName *string) (resp bool, err error) {
	params := []interface{}{
		notificationKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addNotificationSubscriber", params, &r.Options, &resp)
	return
}

// Add a permission to a portal user's permission set. [[Permissions]] control which features in the SoftLayer customer portal and API a user may use. If the user already has the permission you're attempting to add then addPortalPermission() returns true.
//
// Users can assign permissions to their child users, but not to themselves. An account's master has all portal permissions and can set permissions for any of the other users on their account.
//
// Use the [[SoftLayer_User_Customer_CustomerPermission_Permission::getAllObjects]] method to retrieve a list of all permissions available in the SoftLayer customer portal and API. Permissions are added based on the keyName property of the permission parameter.
func (r User_Customer_OpenIdConnect) AddPortalPermission(permission *datatypes.User_Customer_CustomerPermission_Permission) (resp bool, err error) {
	params := []interface{}{
		permission,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addPortalPermission", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) AddRole(role *datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		role,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addRole", params, &r.Options, &resp)
	return
}

// Add a CloudLayer Computing Instance to a portal user's access list. A user's CloudLayer Computing Instance access list controls which of an account's CloudLayer Computing Instance objects a user has access to in the SoftLayer customer portal and API. CloudLayer Computing Instances do not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user already has access to the CloudLayer Computing Instance you're attempting to add then addVirtualGuestAccess() returns true.
//
// Users can assign CloudLayer Computing Instance access to their child users, but not to themselves. An account's master has access to all CloudLayer Computing Instances on their customer account and can set CloudLayer Computing Instance access for any of the other users on their account.
func (r User_Customer_OpenIdConnect) AddVirtualGuestAccess(virtualGuestId *int) (resp bool, err error) {
	params := []interface{}{
		virtualGuestId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "addVirtualGuestAccess", params, &r.Options, &resp)
	return
}

// Select a type of preference you would like to modify using [[SoftLayer_User_Customer::getPreferenceTypes|getPreferenceTypes]] and invoke this method using that preference type key name.
func (r User_Customer_OpenIdConnect) ChangePreference(preferenceTypeKeyName *string, value *string) (resp []datatypes.User_Preference, err error) {
	params := []interface{}{
		preferenceTypeKeyName,
		value,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "changePreference", params, &r.Options, &resp)
	return
}

// This service checks the result of a previously requested external authentication. [[SoftLayer_Container_User_Customer_External_Binding_Phone|Phone external binding]] container can be used for this service. Make sure to set the [[SoftLayer_Container_User_Customer_External_Binding_Phone::authenticationToken|authenticationToken]] that is generated by [[SoftLayer_User_Customer|initiateExternalAuthentication]] service.
func (r User_Customer_OpenIdConnect) CheckExternalAuthenticationStatus(authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "checkExternalAuthenticationStatus", params, &r.Options, &resp)
	return
}

// Add a description here
//
//
func (r User_Customer_OpenIdConnect) CheckPhoneFactorAuthenticationForPasswordSet(passwordSet *datatypes.Container_User_Customer_PasswordSet, authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp bool, err error) {
	params := []interface{}{
		passwordSet,
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "checkPhoneFactorAuthenticationForPasswordSet", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) CompleteInvitationAfterLogin(providerType *string, accessToken *string, emailRegistrationCode *string) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		providerType,
		accessToken,
		emailRegistrationCode,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "completeInvitationAfterLogin", params, &r.Options, &resp)
	return
}

// Create a new subscriber for a given resource.
func (r User_Customer_OpenIdConnect) CreateNotificationSubscriber(keyName *string, resourceTableId *int) (resp bool, err error) {
	params := []interface{}{
		keyName,
		resourceTableId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "createNotificationSubscriber", params, &r.Options, &resp)
	return
}

// Create a new user in the SoftLayer customer portal. createObject() creates a user's portal record and adds them into the SoftLayer community forums. It is no longer possible to set up the SSL or PPTP enable flag in this call since the manage permissions have not yet been set.  You will need to make a subsequent call to edit object in order to enable VPN access. An account's master user and sub-users who have the User Manage permission can add new users. createObject() creates users with a default permission set. After adding a user it may be helpful to set their permissions and hardware access.
//
// Note, neither password nor vpnPassword parameters are required.
//
// Password When a new user is created, an email will be sent to the new user's email address with a link to a url that will allow the new user to create or change their password for the SoftLayer customer portal.
//
// If the password parameter is provided and is not null, then that value will be validated. If it is a valid password, then the user will be created with this password.  This user will still receive a portal password email.  It can be used within 24 hours to change their password, or it can be allowed to expire, and the password provided during user creation will remain as the user's password.
//
// If the password parameter is not provided or the value is null, the user must set their portal password using the link sent in email within 24 hours.  If the user fails to set their password within 24 hours, then a non-master user can use the "Reset Password" link on the login page of the portal to request a new email.  A master user can use the link to retrieve a phone number to call to assist in resetting their password.
//
// The password parameter is ignored for VPN_ONLY users or for IBMid authenticated users.
//
// vpnPassword If the vpnPassword is provided, then the user's vpnPassword will be set to the provided password.  When creating a vpn only user, the vpnPassword MUST be supplied.  If the vpnPassword is not provided, then the user will need to use the portal to edit their profile and set the vpnPassword.
//
//
func (r User_Customer_OpenIdConnect) CreateObject(templateObject *datatypes.User_Customer, password *string, vpnPassword *string) (resp datatypes.User_Customer, err error) {
	params := []interface{}{
		templateObject,
		password,
		vpnPassword,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) CreateOpenIdConnectUserAndCompleteInvitation(providerType *string, user *datatypes.User_Customer, password *string, registrationCode *string) (resp string, err error) {
	params := []interface{}{
		providerType,
		user,
		password,
		registrationCode,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "createOpenIdConnectUserAndCompleteInvitation", params, &r.Options, &resp)
	return
}

// Create delivery methods for a notification that the user is subscribed to. Multiple delivery method keyNames can be supplied to create multiple delivery methods for the specified notification. Available delivery methods - 'EMAIL'. Available notifications - 'PLANNED_MAINTENANCE', 'UNPLANNED_INCIDENT'.
func (r User_Customer_OpenIdConnect) CreateSubscriberDeliveryMethods(notificationKeyName *string, deliveryMethodKeyNames []string) (resp bool, err error) {
	params := []interface{}{
		notificationKeyName,
		deliveryMethodKeyNames,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "createSubscriberDeliveryMethods", params, &r.Options, &resp)
	return
}

// Create a new subscriber for a given resource.
func (r User_Customer_OpenIdConnect) DeactivateNotificationSubscriber(keyName *string, resourceTableId *int) (resp bool, err error) {
	params := []interface{}{
		keyName,
		resourceTableId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "deactivateNotificationSubscriber", params, &r.Options, &resp)
	return
}

// Declines an invitation to link an OpenIdConnect identity to a SoftLayer (Atlas) identity and account. Note that this uses a registration code that is likely a one-time-use-only token, so if an invitation has already been processed (accepted or previously declined) it will not be possible to process it a second time.
func (r User_Customer_OpenIdConnect) DeclineInvitation(providerType *string, registrationCode *string) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		providerType,
		registrationCode,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "declineInvitation", params, &r.Options, &resp)
	return
}

// Account master users and sub-users who have the User Manage permission in the SoftLayer customer portal can update other user's information. Use editObject() if you wish to edit a single user account. Users who do not have the User Manage permission can only update their own information.
func (r User_Customer_OpenIdConnect) EditObject(templateObject *datatypes.User_Customer) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "editObject", params, &r.Options, &resp)
	return
}

// Account master users and sub-users who have the User Manage permission in the SoftLayer customer portal can update other user's information. Use editObjects() if you wish to edit multiple users at once. Users who do not have the User Manage permission can only update their own information.
func (r User_Customer_OpenIdConnect) EditObjects(templateObjects []datatypes.User_Customer) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "editObjects", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) FindUserPreference(profileName *string, containerKeyname *string, preferenceKeyname *string) (resp []datatypes.Layout_Profile, err error) {
	params := []interface{}{
		profileName,
		containerKeyname,
		preferenceKeyname,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "findUserPreference", params, &r.Options, &resp)
	return
}

// Retrieve The customer account that a user belongs to.
func (r User_Customer_OpenIdConnect) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer_OpenIdConnect) GetActions() (resp []datatypes.User_Permission_Action, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getActions", nil, &r.Options, &resp)
	return
}

// The getActiveExternalAuthenticationVendors method will return a list of available external vendors that a SoftLayer user can authenticate against.  The list will only contain vendors for which the user has at least one active external binding.
func (r User_Customer_OpenIdConnect) GetActiveExternalAuthenticationVendors() (resp []datatypes.Container_User_Customer_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getActiveExternalAuthenticationVendors", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's additional email addresses. These email addresses are contacted when updates are made to support tickets.
func (r User_Customer_OpenIdConnect) GetAdditionalEmails() (resp []datatypes.User_Customer_AdditionalEmail, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getAdditionalEmails", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetAllowedHardwareIds() (resp []int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getAllowedHardwareIds", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetAllowedVirtualGuestIds() (resp []int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getAllowedVirtualGuestIds", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's API Authentication keys. There is a max limit of two API keys per user.
func (r User_Customer_OpenIdConnect) GetApiAuthenticationKeys() (resp []datatypes.User_Customer_ApiAuthentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getApiAuthenticationKeys", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetAuthenticationToken(token *datatypes.Container_User_Authentication_Token) (resp datatypes.Container_User_Authentication_Token, err error) {
	params := []interface{}{
		token,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getAuthenticationToken", params, &r.Options, &resp)
	return
}

// Retrieve The CDN accounts associated with a portal user.
func (r User_Customer_OpenIdConnect) GetCdnAccounts() (resp []datatypes.Network_ContentDelivery_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getCdnAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's child users. Some portal users may not have child users.
func (r User_Customer_OpenIdConnect) GetChildUsers() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getChildUsers", nil, &r.Options, &resp)
	return
}

// Retrieve An user's associated closed tickets.
func (r User_Customer_OpenIdConnect) GetClosedTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getClosedTickets", nil, &r.Options, &resp)
	return
}

// This API gets the default account for the OpenIdConnect identity that is linked to the current SoftLayer user identity. If there is no default present, the API returns null, except in the special case where we find one active user linked to the IBMid. In that case, we will set the link from the IBMid to that user as default, and return the account of which that user is a member.
func (r User_Customer_OpenIdConnect) GetDefaultAccount(providerType *string) (resp datatypes.Account, err error) {
	params := []interface{}{
		providerType,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getDefaultAccount", params, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong>
func (r User_Customer_OpenIdConnect) GetDefaultSecurityQuestions(key *string) (resp []datatypes.User_Security_Question, err error) {
	params := []interface{}{
		key,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getDefaultSecurityQuestions", params, &r.Options, &resp)
	return
}

// Retrieve The external authentication bindings that link an external identifier to a SoftLayer user.
func (r User_Customer_OpenIdConnect) GetExternalBindings() (resp []datatypes.User_External_Binding, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getExternalBindings", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's accessible hardware. These permissions control which hardware a user has access to in the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) GetHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve the number of servers that a portal user has access to. Portal users can have restrictions set to limit services for and to perform actions on hardware. You can set these permissions in the portal by clicking the "administrative" then "user admin" links.
func (r User_Customer_OpenIdConnect) GetHardwareCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getHardwareCount", nil, &r.Options, &resp)
	return
}

// Retrieve Hardware notifications associated with this user. A hardware notification links a user to a piece of hardware, and that user will be notified if any monitors on that hardware fail, if the monitors have a status of 'Notify User'.
func (r User_Customer_OpenIdConnect) GetHardwareNotifications() (resp []datatypes.User_Customer_Notification_Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getHardwareNotifications", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a user has acknowledged the support policy.
func (r User_Customer_OpenIdConnect) GetHasAcknowledgedSupportPolicyFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getHasAcknowledgedSupportPolicyFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a portal user has access to all hardware on their account.
func (r User_Customer_OpenIdConnect) GetHasFullHardwareAccessFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getHasFullHardwareAccessFlag", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a portal user has access to all hardware on their account.
func (r User_Customer_OpenIdConnect) GetHasFullVirtualGuestAccessFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getHasFullVirtualGuestAccessFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetImpersonationToken() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getImpersonationToken", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer_OpenIdConnect) GetLayoutProfiles() (resp []datatypes.Layout_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getLayoutProfiles", nil, &r.Options, &resp)
	return
}

// Retrieve A user's locale. Locale holds user's language and region information.
func (r User_Customer_OpenIdConnect) GetLocale() (resp datatypes.Locale, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getLocale", nil, &r.Options, &resp)
	return
}

// Validates a supplied OpenIdConnect access token to the SoftLayer customer portal and returns the default account name and id for the active user. An exception will be thrown if no matching customer is found.
func (r User_Customer_OpenIdConnect) GetLoginAccountInfoOpenIdConnect(providerType *string, accessToken *string) (resp datatypes.Container_User_Customer_OpenIdConnect_LoginAccountInfo, err error) {
	params := []interface{}{
		providerType,
		accessToken,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getLoginAccountInfoOpenIdConnect", params, &r.Options, &resp)
	return
}

// Retrieve A user's attempts to log into the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) GetLoginAttempts() (resp []datatypes.User_Customer_Access_Authentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getLoginAttempts", nil, &r.Options, &resp)
	return
}

// Attempt to authenticate a user to the SoftLayer customer portal using the provided authentication container. Depending on the specific type of authentication container that is used, this API will leverage the appropriate authentication protocol. If authentication is successful then the API returns a list of linked accounts for the user, a token containing the ID of the authenticated user and a hash key used by the SoftLayer customer portal to maintain authentication.
func (r User_Customer_OpenIdConnect) GetLoginToken(request *datatypes.Container_Authentication_Request_Contract) (resp datatypes.Container_Authentication_Response_Common, err error) {
	params := []interface{}{
		request,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getLoginToken", params, &r.Options, &resp)
	return
}

// An OpenIdConnect identity, for example an IBMid, can be linked or mapped to one or more individual SoftLayer users, but no more than one SoftLayer user per account. This effectively links the OpenIdConnect identity to those accounts. This API returns a list of all the accounts for which there is a link between the OpenIdConnect identity and a SoftLayer user.
func (r User_Customer_OpenIdConnect) GetMappedAccounts(providerType *string) (resp []datatypes.Account, err error) {
	params := []interface{}{
		providerType,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getMappedAccounts", params, &r.Options, &resp)
	return
}

// Retrieve A portal user's associated mobile device profiles.
func (r User_Customer_OpenIdConnect) GetMobileDevices() (resp []datatypes.User_Customer_MobileDevice, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getMobileDevices", nil, &r.Options, &resp)
	return
}

// Retrieve Notification subscription records for the user.
func (r User_Customer_OpenIdConnect) GetNotificationSubscribers() (resp []datatypes.Notification_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getNotificationSubscribers", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetObject() (resp datatypes.User_Customer_OpenIdConnect, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getObject", nil, &r.Options, &resp)
	return
}

// This API returns a SoftLayer_Container_User_Customer_OpenIdConnect_MigrationState object containing the necessary information to determine what migration state the user is in. If the account is not OpenIdConnect authenticated, then an exception is thrown.
func (r User_Customer_OpenIdConnect) GetOpenIdConnectMigrationState() (resp datatypes.Container_User_Customer_OpenIdConnect_MigrationState, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getOpenIdConnectMigrationState", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetOpenIdRegistrationInfoFromCode(providerType *string, registrationCode *string) (resp datatypes.Account_Authentication_OpenIdConnect_RegistrationInformation, err error) {
	params := []interface{}{
		providerType,
		registrationCode,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getOpenIdRegistrationInfoFromCode", params, &r.Options, &resp)
	return
}

// Retrieve An user's associated open tickets.
func (r User_Customer_OpenIdConnect) GetOpenTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getOpenTickets", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's vpn accessible subnets.
func (r User_Customer_OpenIdConnect) GetOverrides() (resp []datatypes.Network_Service_Vpn_Overrides, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getOverrides", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's parent user. If a SoftLayer_User_Customer has a null parentId property then it doesn't have a parent user.
func (r User_Customer_OpenIdConnect) GetParent() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getParent", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's permissions. These permissions control that user's access to functions within the SoftLayer customer portal and API.
func (r User_Customer_OpenIdConnect) GetPermissions() (resp []datatypes.User_Customer_CustomerPermission_Permission, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getPermissions", nil, &r.Options, &resp)
	return
}

// Attempt to authenticate a username and password to the SoftLayer customer portal. Many portal user accounts are configured to require answering a security question on login. In this case getPortalLoginToken() also verifies the given security question ID and answer. If authentication is successful then the API returns a token containing the ID of the authenticated user and a hash key used by the SoftLayer customer portal to maintain authentication.
func (r User_Customer_OpenIdConnect) GetPortalLoginToken(username *string, password *string, securityQuestionId *int, securityQuestionAnswer *string) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		username,
		password,
		securityQuestionId,
		securityQuestionAnswer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getPortalLoginToken", params, &r.Options, &resp)
	return
}

// Attempt to authenticate a supplied OpenIdConnect access token to the SoftLayer customer portal. If authentication is successful then the API returns a token containing the ID of the authenticated user and a hash key used by the SoftLayer customer portal to maintain authentication.
func (r User_Customer_OpenIdConnect) GetPortalLoginTokenOpenIdConnect(providerType *string, accessToken *string, accountId *int, securityQuestionId *int, securityQuestionAnswer *string) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		providerType,
		accessToken,
		accountId,
		securityQuestionId,
		securityQuestionAnswer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getPortalLoginTokenOpenIdConnect", params, &r.Options, &resp)
	return
}

// Select a type of preference you would like to get using [[SoftLayer_User_Customer::getPreferenceTypes|getPreferenceTypes]] and invoke this method using that preference type key name.
func (r User_Customer_OpenIdConnect) GetPreference(preferenceTypeKeyName *string) (resp datatypes.User_Preference, err error) {
	params := []interface{}{
		preferenceTypeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getPreference", params, &r.Options, &resp)
	return
}

// Use any of the preference types to fetch or modify user preferences using [[SoftLayer_User_Customer::getPreference|getPreference]] or [[SoftLayer_User_Customer::changePreference|changePreference]], respectively.
func (r User_Customer_OpenIdConnect) GetPreferenceTypes() (resp []datatypes.User_Preference_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getPreferenceTypes", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer_OpenIdConnect) GetPreferences() (resp []datatypes.User_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getPreferences", nil, &r.Options, &resp)
	return
}

// Retrieve the authentication requirements for an outstanding password set/reset request.  The password key provided to the user in an email generated by the [[SoftLayer_User_Customer::newUserPassword|newUserPassword]]. Password recovery keys are valid for 24 hours after they're generated.
func (r User_Customer_OpenIdConnect) GetRequirementsForPasswordSet(passwordSet *datatypes.Container_User_Customer_PasswordSet) (resp datatypes.Container_User_Customer_PasswordSet, err error) {
	params := []interface{}{
		passwordSet,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getRequirementsForPasswordSet", params, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer_OpenIdConnect) GetRoles() (resp []datatypes.User_Permission_Role, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getRoles", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer_OpenIdConnect) GetSalesforceUserLink() (resp datatypes.User_Customer_Link, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSalesforceUserLink", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's security question answers. Some portal users may not have security answers or may not be configured to require answering a security question on login.
func (r User_Customer_OpenIdConnect) GetSecurityAnswers() (resp []datatypes.User_Customer_Security_Answer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSecurityAnswers", nil, &r.Options, &resp)
	return
}

// Retrieve A user's notification subscription records.
func (r User_Customer_OpenIdConnect) GetSubscribers() (resp []datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSubscribers", nil, &r.Options, &resp)
	return
}

// Retrieve A user's successful attempts to log into the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) GetSuccessfulLogins() (resp []datatypes.User_Customer_Access_Authentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSuccessfulLogins", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a user is required to acknowledge the support policy for portal access.
func (r User_Customer_OpenIdConnect) GetSupportPolicyAcknowledgementRequiredFlag() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSupportPolicyAcknowledgementRequiredFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetSupportPolicyDocument() (resp []byte, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSupportPolicyDocument", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetSupportPolicyName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSupportPolicyName", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetSupportedLocales() (resp []datatypes.Locale, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSupportedLocales", nil, &r.Options, &resp)
	return
}

// Retrieve Whether or not a user must take a brief survey the next time they log into the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) GetSurveyRequiredFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSurveyRequiredFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The surveys that a user has taken in the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) GetSurveys() (resp []datatypes.Survey, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getSurveys", nil, &r.Options, &resp)
	return
}

// Retrieve An user's associated tickets.
func (r User_Customer_OpenIdConnect) GetTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getTickets", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's time zone.
func (r User_Customer_OpenIdConnect) GetTimezone() (resp datatypes.Locale_Timezone, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getTimezone", nil, &r.Options, &resp)
	return
}

// Retrieve A user's unsuccessful attempts to log into the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) GetUnsuccessfulLogins() (resp []datatypes.User_Customer_Access_Authentication, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getUnsuccessfulLogins", nil, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong> Retrieve a user object using a password recovery key received in an email generated by the [[SoftLayer_User_Customer::lostPassword|lostPassword]] method. The SoftLayer customer portal uses getUserFromLostPasswordRequest() to retrieve user security questions. Password recovery keys are valid for 24 hours after they're generated.
func (r User_Customer_OpenIdConnect) GetUserFromLostPasswordRequest(key *string) (resp []datatypes.User_Security_Question, err error) {
	params := []interface{}{
		key,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getUserFromLostPasswordRequest", params, &r.Options, &resp)
	return
}

// Retrieve a user object using a password token. When a new user is created or when a user has requested a password change using initiatePortalPasswordChange, they will have received an email that contains a url with a token.  That token is used as the parameter for getUserIdForPasswordSet.
func (r User_Customer_OpenIdConnect) GetUserIdForPasswordSet(key *string) (resp int, err error) {
	params := []interface{}{
		key,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getUserIdForPasswordSet", params, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Customer_OpenIdConnect) GetUserLinks() (resp []datatypes.User_Customer_Link, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getUserLinks", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) GetUserPreferences(profileName *string, containerKeyname *string) (resp []datatypes.Layout_Profile, err error) {
	params := []interface{}{
		profileName,
		containerKeyname,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getUserPreferences", params, &r.Options, &resp)
	return
}

// Retrieve A portal user's status, which controls overall access to the SoftLayer customer portal and VPN access to the private network.
func (r User_Customer_OpenIdConnect) GetUserStatus() (resp datatypes.User_Customer_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getUserStatus", nil, &r.Options, &resp)
	return
}

// Retrieve the number of CloudLayer Computing Instances that a portal user has access to. Portal users can have restrictions set to limit services for and to perform actions on CloudLayer Computing Instances. You can set these permissions in the portal by clicking the "administrative" then "user admin" links.
func (r User_Customer_OpenIdConnect) GetVirtualGuestCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getVirtualGuestCount", nil, &r.Options, &resp)
	return
}

// Retrieve A portal user's accessible CloudLayer Computing Instances. These permissions control which CloudLayer Computing Instances a user has access to in the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) GetVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "getVirtualGuests", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) InTerminalStatus() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "inTerminalStatus", nil, &r.Options, &resp)
	return
}

// The service initiates an external authentication with the given external authentication vendor. The authentication container and its content will be verified before an attempt is made to initiate an external authentication. [[SoftLayer_Container_User_Customer_External_Binding_Phone|Phone external binding]] container can be used for this service.
//
// This service returns a unique authentication request token. You can use [[SoftLayer_User_Customer::checkExternalAuthenticationStatus|checkExternalAuthenticationStatus]] service to check if the authentication request is complete or not.
func (r User_Customer_OpenIdConnect) InitiateExternalAuthentication(authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp string, err error) {
	params := []interface{}{
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "initiateExternalAuthentication", params, &r.Options, &resp)
	return
}

// Sends password change email to the user containing url that allows the user the change their password. This is the first step when a user wishes to change their password.  The url that is generated contains a one-time use token that is valid for only 24-hours.
func (r User_Customer_OpenIdConnect) InitiatePortalPasswordChange(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "initiatePortalPasswordChange", params, &r.Options, &resp)
	return
}

// A Brand Agent that has permissions to Add Customer Accounts will be able to request the password email be sent to the Master User of a Customer Account created by the same Brand as the agent making the request.
func (r User_Customer_OpenIdConnect) InitiatePortalPasswordChangeByBrandAgent(username *string) (resp bool, err error) {
	params := []interface{}{
		username,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "initiatePortalPasswordChangeByBrandAgent", params, &r.Options, &resp)
	return
}

// Send email invitation to a user to join a SoftLayer account and authenticate with OpenIdConnect. Throws an exception on error.
func (r User_Customer_OpenIdConnect) InviteUserToLinkOpenIdConnect(providerType *string, linkedBluemixAccountMatchAlreadyChecked *bool) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		providerType,
		linkedBluemixAccountMatchAlreadyChecked,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "inviteUserToLinkOpenIdConnect", params, &r.Options, &resp)
	return
}

// Portal users are considered master users if they don't have an associated parent user. The only users who don't have parent users are users whose username matches their SoftLayer account name. Master users have special permissions throughout the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) IsMasterUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "isMasterUser", nil, &r.Options, &resp)
	return
}

// This method is deprecated! SoftLayer Community Forums no longer exist, therefore, any password verified will return false.
//
// Determine if a string is the given user's login password to the SoftLayer community forums.
func (r User_Customer_OpenIdConnect) IsValidForumPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "isValidForumPassword", params, &r.Options, &resp)
	return
}

// Determine if a string is the given user's login password to the SoftLayer customer portal.
func (r User_Customer_OpenIdConnect) IsValidPortalPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "isValidPortalPassword", params, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong> SoftLayer provides a way for users of it's customer portal to recover lost passwords. The lostPassword() method is the first step in this process. Given a valid username and email address, the SoftLayer API will email the address provided with a URL to visit to begin the password recovery process. The last part of this URL is a hash key that's used as an identifier throughout this process. Use this hash key in the [[SoftLayer_User_Customer::setPasswordFromLostPasswordRequest|setPasswordFromLostPasswordRequest]] method to reset a user's password. Password recovery hash keys are valid for 24 hours after they're generated.
func (r User_Customer_OpenIdConnect) LostPassword(username *string, email *string) (resp bool, err error) {
	params := []interface{}{
		username,
		email,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "lostPassword", params, &r.Options, &resp)
	return
}

// The perform external authentication method will authenticate the given external authentication container with an external vendor.  The authentication container and its contents will be verified before an attempt is made to authenticate the contents of the container with an external vendor.
func (r User_Customer_OpenIdConnect) PerformExternalAuthentication(authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "performExternalAuthentication", params, &r.Options, &resp)
	return
}

// Set the password for a user who has an outstanding password request. A user with an outstanding password request will have an unused and unexpired password key.  The password key is part of the url provided to the user in the email sent to the user with information on how to set their password.  The email was generated by the [[SoftLayer_User_Customer::setNewUserPassword|setNewUserPassword]] method. Password recovery keys are valid for 24 hours after they're generated.
//
// User portal passwords must match the following restrictions. Portal passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ + =
// * ...not match your username
// * ...not match your forum password
func (r User_Customer_OpenIdConnect) ProcessPasswordSetRequest(passwordSet *datatypes.Container_User_Customer_PasswordSet, authenticationContainer *datatypes.Container_User_Customer_External_Binding) (resp bool, err error) {
	params := []interface{}{
		passwordSet,
		authenticationContainer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "processPasswordSetRequest", params, &r.Options, &resp)
	return
}

// Remove all hardware from a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. If the current user does not have administrative privileges over this user, an inadequate permissions exception will get thrown.
//
// Users can call this function on child users, but not to themselves. An account's master has access to all users permissions on their account.
func (r User_Customer_OpenIdConnect) RemoveAllHardwareAccessForThisUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeAllHardwareAccessForThisUser", nil, &r.Options, &resp)
	return
}

// Remove all cloud computing instances from a portal user's instance access list. A user's instance access list controls which of an account's computing instance objects a user has access to in the SoftLayer customer portal and API. If the current user does not have administrative privileges over this user, an inadequate permissions exception will get thrown.
//
// Users can call this function on child users, but not to themselves. An account's master has access to all users permissions on their account.
func (r User_Customer_OpenIdConnect) RemoveAllVirtualAccessForThisUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeAllVirtualAccessForThisUser", nil, &r.Options, &resp)
	return
}

// Remove a user's API authentication key, removing that user's access to query the SoftLayer API.
func (r User_Customer_OpenIdConnect) RemoveApiAuthenticationKey(keyId *int) (resp bool, err error) {
	params := []interface{}{
		keyId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeApiAuthenticationKey", params, &r.Options, &resp)
	return
}

// Remove multiple hardware from a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. Hardware does not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user does not has access to the hardware you're attempting remove add then removeBulkHardwareAccess() returns true.
//
// Users can assign hardware access to their child users, but not to themselves. An account's master has access to all hardware on their customer account and can set hardware access for any of the other users on their account.
//
// If the user has full hardware access, then it will provide access to "ALL but passed in" hardware ids.
func (r User_Customer_OpenIdConnect) RemoveBulkHardwareAccess(hardwareIds []int) (resp bool, err error) {
	params := []interface{}{
		hardwareIds,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeBulkHardwareAccess", params, &r.Options, &resp)
	return
}

// Remove multiple permissions from a portal user's permission set. [[Permissions]] control which features in the SoftLayer customer portal and API a user may use. Removing a user's permission will affect that user's portal and API access. removePortalPermission() does not attempt to remove permissions that are not assigned to the user.
//
// Users can assign permissions to their child users, but not to themselves. An account's master has all portal permissions and can set permissions for any of the other users on their account.
//
// Use the [[SoftLayer_User_Customer_CustomerPermission_Permission::getAllObjects]] method to retrieve a list of all permissions available in the SoftLayer customer portal and API. Permissions are removed based on the keyName property of the permission objects within the permissions parameter.
func (r User_Customer_OpenIdConnect) RemoveBulkPortalPermission(permissions []datatypes.User_Customer_CustomerPermission_Permission) (resp bool, err error) {
	params := []interface{}{
		permissions,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeBulkPortalPermission", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) RemoveBulkRoles(roles []datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		roles,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeBulkRoles", params, &r.Options, &resp)
	return
}

// Remove multiple CloudLayer Computing Instances from a portal user's access list. A user's CloudLayer Computing Instance access list controls which of an account's CloudLayer Computing Instance objects a user has access to in the SoftLayer customer portal and API. CloudLayer Computing Instances do not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user does not has access to the CloudLayer Computing Instance you're attempting remove add then removeBulkVirtualGuestAccess() returns true.
//
// Users can assign CloudLayer Computing Instance access to their child users, but not to themselves. An account's master has access to all CloudLayer Computing Instances on their customer account and can set hardware access for any of the other users on their account.
func (r User_Customer_OpenIdConnect) RemoveBulkVirtualGuestAccess(virtualGuestIds []int) (resp bool, err error) {
	params := []interface{}{
		virtualGuestIds,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeBulkVirtualGuestAccess", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) RemoveExternalBinding(externalBinding *datatypes.User_External_Binding) (resp bool, err error) {
	params := []interface{}{
		externalBinding,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeExternalBinding", params, &r.Options, &resp)
	return
}

// Remove hardware from a portal user's hardware access list. A user's hardware access list controls which of an account's hardware objects a user has access to in the SoftLayer customer portal and API. Hardware does not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user does not has access to the hardware you're attempting remove add then removeHardwareAccess() returns true.
//
// Users can assign hardware access to their child users, but not to themselves. An account's master has access to all hardware on their customer account and can set hardware access for any of the other users on their account.
func (r User_Customer_OpenIdConnect) RemoveHardwareAccess(hardwareId *int) (resp bool, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeHardwareAccess", params, &r.Options, &resp)
	return
}

// Remove a permission from a portal user's permission set. [[Permissions]] control which features in the SoftLayer customer portal and API a user may use. Removing a user's permission will affect that user's portal and API access. If the user does not have the permission you're attempting to remove then removePortalPermission() returns true.
//
// Users can assign permissions to their child users, but not to themselves. An account's master has all portal permissions and can set permissions for any of the other users on their account.
//
// Use the [[SoftLayer_User_Customer_CustomerPermission_Permission::getAllObjects]] method to retrieve a list of all permissions available in the SoftLayer customer portal and API. Permissions are removed based on the keyName property of the permission parameter.
func (r User_Customer_OpenIdConnect) RemovePortalPermission(permission *datatypes.User_Customer_CustomerPermission_Permission) (resp bool, err error) {
	params := []interface{}{
		permission,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removePortalPermission", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) RemoveRole(role *datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		role,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeRole", params, &r.Options, &resp)
	return
}

// Remove a CloudLayer Computing Instance from a portal user's access list. A user's CloudLayer Computing Instance access list controls which of an account's computing instances a user has access to in the SoftLayer customer portal and API. CloudLayer Computing Instances do not exist in the SoftLayer portal and returns "not found" exceptions in the API if the user doesn't have access to it. If a user does not has access to the CloudLayer Computing Instance you're attempting remove add then removeVirtualGuestAccess() returns true.
//
// Users can assign CloudLayer Computing Instance access to their child users, but not to themselves. An account's master has access to all CloudLayer Computing Instances on their customer account and can set instance access for any of the other users on their account.
func (r User_Customer_OpenIdConnect) RemoveVirtualGuestAccess(virtualGuestId *int) (resp bool, err error) {
	params := []interface{}{
		virtualGuestId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "removeVirtualGuestAccess", params, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong> Attempt to authenticate a username and password to the SoftLayer customer portal and reset there password. If authentication and password reset is successful then the API returns true.
func (r User_Customer_OpenIdConnect) ResetExpiredPassword(username *string, password *string, newPassword *string, securityQuestionId *int, securityQuestionAnswer *string) (resp bool, err error) {
	params := []interface{}{
		username,
		password,
		newPassword,
		securityQuestionId,
		securityQuestionAnswer,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "resetExpiredPassword", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) SamlAuthenticate(accountId *string, samlResponse *string) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		accountId,
		samlResponse,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "samlAuthenticate", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) SamlBeginAuthentication(accountId *int) (resp string, err error) {
	params := []interface{}{
		accountId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "samlBeginAuthentication", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) SamlBeginLogout() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "samlBeginLogout", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) SamlLogout(samlResponse *string) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		samlResponse,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "samlLogout", params, &r.Options, &resp)
	return
}

// An OpenIdConnect identity, for example an IBMid, can be linked or mapped to one or more individual SoftLayer users, but no more than one per account. If an OpenIdConnect identity is mapped to multiple accounts in this manner, one such account should be identified as the default account for that identity.
func (r User_Customer_OpenIdConnect) SetDefaultAccount(providerType *string, accountId *int) (resp datatypes.Account, err error) {
	params := []interface{}{
		providerType,
		accountId,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "setDefaultAccount", params, &r.Options, &resp)
	return
}

// Set a user's password via the lost password recovery system, using a password recovery key received in an email generated by the [[SoftLayer_User_Customer::lostPassword|lostPassword]] method. Password recovery keys are valid for 24 hours after they're generated.
//
// User portal passwords must match the following restrictions. Portal passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ + =
// * ...not match your username
// * ...not match your forum password
func (r User_Customer_OpenIdConnect) SetPasswordFromLostPasswordRequest(key *string, password *string, securityAnswers []datatypes.User_Customer_Security_Answer) (resp bool, err error) {
	params := []interface{}{
		key,
		password,
		securityAnswers,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "setPasswordFromLostPasswordRequest", params, &r.Options, &resp)
	return
}

// As master user, calling this api for the IBMid provider type when there is an existing IBMid for the email on the SL account will silently (without sending an invitation email) create a link for the IBMid. If the SoftLayer user is already linked to IBMid, this will reset the existing link. If the IBMid specified by the email of this user, is already used in a link to another user in this account, this call will fail. If there is already an open invitation from this SoftLayer user to this or any IBMid, this call will fail. If there is already an open invitation from some other SoftLayer user in this account to this IBMid, then this call will fail.
func (r User_Customer_OpenIdConnect) SilentlyMigrateUserOpenIdConnect(providerType *string) (resp bool, err error) {
	params := []interface{}{
		providerType,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "silentlyMigrateUserOpenIdConnect", params, &r.Options, &resp)
	return
}

// This method is deprecated! SoftLayer Community Forums no longer exist, therefore, this method will return false.
//
// Update a user's password on the SoftLayer community forums. As with portal passwords, user forum passwords must match the following restrictions. Forum passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ + =
// * ...not match your username
// * ...not match your portal password
// Finally, users can only update their own password.
func (r User_Customer_OpenIdConnect) UpdateForumPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "updateForumPassword", params, &r.Options, &resp)
	return
}

// Update the active status for a notification that the user is subscribed to. A notification along with an active flag can be supplied to update the active status for a particular notification subscription.
func (r User_Customer_OpenIdConnect) UpdateNotificationSubscriber(notificationKeyName *string, active *int) (resp bool, err error) {
	params := []interface{}{
		notificationKeyName,
		active,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "updateNotificationSubscriber", params, &r.Options, &resp)
	return
}

// <strong>This method is deprecated.  Please see documentation for initiatePortalPasswordChange</strong> Update a user's password on the SoftLayer customer portal. As with forum passwords, user portal passwords must match the following restrictions. Portal passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ + =
// * ...not match your username
// * ...not match your forum password
// Finally, users can only update their own password. An account's master user can update any of their account users' passwords.
func (r User_Customer_OpenIdConnect) UpdatePassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "updatePassword", params, &r.Options, &resp)
	return
}

// Update a user's login security questions and answers on the SoftLayer customer portal. These questions and answers are used to optionally log into the SoftLayer customer portal using two-factor authentication. Each user must have three distinct questions set with a unique answer for each question, and each answer may only contain alphanumeric or the . , - _ ( ) [ ] : ; > < characters. Existing user security questions and answers are deleted before new ones are set, and users may only update their own security questions and answers.
func (r User_Customer_OpenIdConnect) UpdateSecurityAnswers(questions []datatypes.User_Security_Question, answers []string) (resp bool, err error) {
	params := []interface{}{
		questions,
		answers,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "updateSecurityAnswers", params, &r.Options, &resp)
	return
}

// Update a delivery method for a notification that the user is subscribed to. A delivery method keyName along with an active flag can be supplied to update the active status of the delivery methods for the specified notification. Available delivery methods - 'EMAIL'. Available notifications - 'PLANNED_MAINTENANCE', 'UNPLANNED_INCIDENT'.
func (r User_Customer_OpenIdConnect) UpdateSubscriberDeliveryMethod(notificationKeyName *string, deliveryMethodKeyNames []string, active *int) (resp bool, err error) {
	params := []interface{}{
		notificationKeyName,
		deliveryMethodKeyNames,
		active,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "updateSubscriberDeliveryMethod", params, &r.Options, &resp)
	return
}

// Update a user's VPN password on the SoftLayer customer portal. As with portal passwords, VPN passwords must match the following restrictions. VPN passwords must...
// * ...be over eight characters long.
// * ...be under twenty characters long.
// * ...contain at least one uppercase letter
// * ...contain at least one lowercase letter
// * ...contain at least one number
// * ...contain one of the special characters _ - | @ . , ? / ! ~ # $ % ^ & * ( ) { } [ ] \ =
// * ...not match your username
// * ...not match your forum password
// Finally, users can only update their own VPN password. An account's master user can update any of their account users' VPN passwords.
func (r User_Customer_OpenIdConnect) UpdateVpnPassword(password *string) (resp bool, err error) {
	params := []interface{}{
		password,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "updateVpnPassword", params, &r.Options, &resp)
	return
}

// Always call this function to enable changes when manually configuring VPN subnet access.
func (r User_Customer_OpenIdConnect) UpdateVpnUser() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "updateVpnUser", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_OpenIdConnect) ValidateAuthenticationToken(authenticationToken *datatypes.Container_User_Authentication_Token) (resp datatypes.Container_User_Customer_Portal_Token, err error) {
	params := []interface{}{
		authenticationToken,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_OpenIdConnect", "validateAuthenticationToken", params, &r.Options, &resp)
	return
}

// Contains user information for Service Provider Enrollment.
type User_Customer_Prospect_ServiceProvider_EnrollRequest struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerProspectServiceProviderEnrollRequestService returns an instance of the User_Customer_Prospect_ServiceProvider_EnrollRequest SoftLayer service
func GetUserCustomerProspectServiceProviderEnrollRequestService(sess *session.Session) User_Customer_Prospect_ServiceProvider_EnrollRequest {
	return User_Customer_Prospect_ServiceProvider_EnrollRequest{Session: sess}
}

func (r User_Customer_Prospect_ServiceProvider_EnrollRequest) Id(id int) User_Customer_Prospect_ServiceProvider_EnrollRequest {
	r.Options.Id = &id
	return r
}

func (r User_Customer_Prospect_ServiceProvider_EnrollRequest) Mask(mask string) User_Customer_Prospect_ServiceProvider_EnrollRequest {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_Prospect_ServiceProvider_EnrollRequest) Filter(filter string) User_Customer_Prospect_ServiceProvider_EnrollRequest {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_Prospect_ServiceProvider_EnrollRequest) Limit(limit int) User_Customer_Prospect_ServiceProvider_EnrollRequest {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_Prospect_ServiceProvider_EnrollRequest) Offset(offset int) User_Customer_Prospect_ServiceProvider_EnrollRequest {
	r.Options.Offset = &offset
	return r
}

// Create a new Service Provider Enrollment
func (r User_Customer_Prospect_ServiceProvider_EnrollRequest) Enroll(templateObject *datatypes.User_Customer_Prospect_ServiceProvider_EnrollRequest) (resp datatypes.User_Customer_Prospect_ServiceProvider_EnrollRequest, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Customer_Prospect_ServiceProvider_EnrollRequest", "enroll", params, &r.Options, &resp)
	return
}

// Retrieve Catalyst company types.
func (r User_Customer_Prospect_ServiceProvider_EnrollRequest) GetCompanyType() (resp datatypes.Catalyst_Company_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Prospect_ServiceProvider_EnrollRequest", "getCompanyType", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Customer_Prospect_ServiceProvider_EnrollRequest) GetObject() (resp datatypes.User_Customer_Prospect_ServiceProvider_EnrollRequest, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Prospect_ServiceProvider_EnrollRequest", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_User_Customer_Security_Answer type contains user's answers to security questions.
type User_Customer_Security_Answer struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerSecurityAnswerService returns an instance of the User_Customer_Security_Answer SoftLayer service
func GetUserCustomerSecurityAnswerService(sess *session.Session) User_Customer_Security_Answer {
	return User_Customer_Security_Answer{Session: sess}
}

func (r User_Customer_Security_Answer) Id(id int) User_Customer_Security_Answer {
	r.Options.Id = &id
	return r
}

func (r User_Customer_Security_Answer) Mask(mask string) User_Customer_Security_Answer {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_Security_Answer) Filter(filter string) User_Customer_Security_Answer {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_Security_Answer) Limit(limit int) User_Customer_Security_Answer {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_Security_Answer) Offset(offset int) User_Customer_Security_Answer {
	r.Options.Offset = &offset
	return r
}

// getObject retrieves the SoftLayer_User_Customer_Security_Answer object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_User_Customer_Security_Answer service.
func (r User_Customer_Security_Answer) GetObject() (resp datatypes.User_Customer_Security_Answer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Security_Answer", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The question the security answer is associated with.
func (r User_Customer_Security_Answer) GetQuestion() (resp datatypes.User_Security_Question, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Security_Answer", "getQuestion", nil, &r.Options, &resp)
	return
}

// Retrieve The user who the security answer belongs to.
func (r User_Customer_Security_Answer) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Security_Answer", "getUser", nil, &r.Options, &resp)
	return
}

// Each SoftLayer portal account is assigned a status code that determines how it's treated in the customer portal. This status is reflected in the SoftLayer_User_Customer_Status data type. Status differs from user permissions in that user status applies globally to the portal while user permissions are applied to specific portal functions.
type User_Customer_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetUserCustomerStatusService returns an instance of the User_Customer_Status SoftLayer service
func GetUserCustomerStatusService(sess *session.Session) User_Customer_Status {
	return User_Customer_Status{Session: sess}
}

func (r User_Customer_Status) Id(id int) User_Customer_Status {
	r.Options.Id = &id
	return r
}

func (r User_Customer_Status) Mask(mask string) User_Customer_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Customer_Status) Filter(filter string) User_Customer_Status {
	r.Options.Filter = filter
	return r
}

func (r User_Customer_Status) Limit(limit int) User_Customer_Status {
	r.Options.Limit = &limit
	return r
}

func (r User_Customer_Status) Offset(offset int) User_Customer_Status {
	r.Options.Offset = &offset
	return r
}

// Retrieve all user status objects.
func (r User_Customer_Status) GetAllObjects() (resp []datatypes.User_Customer_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Status", "getAllObjects", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_User_Customer_Status object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_User_Customer_Status service.
func (r User_Customer_Status) GetObject() (resp datatypes.User_Customer_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Customer_Status", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_User_External_Binding data type contains general information for a single external binding.  This includes the 3rd party vendor, type of binding, and a unique identifier and password that is used to authenticate against the 3rd party service.
type User_External_Binding struct {
	Session *session.Session
	Options sl.Options
}

// GetUserExternalBindingService returns an instance of the User_External_Binding SoftLayer service
func GetUserExternalBindingService(sess *session.Session) User_External_Binding {
	return User_External_Binding{Session: sess}
}

func (r User_External_Binding) Id(id int) User_External_Binding {
	r.Options.Id = &id
	return r
}

func (r User_External_Binding) Mask(mask string) User_External_Binding {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_External_Binding) Filter(filter string) User_External_Binding {
	r.Options.Filter = filter
	return r
}

func (r User_External_Binding) Limit(limit int) User_External_Binding {
	r.Options.Limit = &limit
	return r
}

func (r User_External_Binding) Offset(offset int) User_External_Binding {
	r.Options.Offset = &offset
	return r
}

// Delete an external authentication binding.  If the external binding currently has an active billing item associated you will be prevented from deleting the binding.  The alternative method to remove an external authentication binding is to use the service cancellation form.
func (r User_External_Binding) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding", "deleteObject", nil, &r.Options, &resp)
	return
}

// Retrieve Attributes of an external authentication binding.
func (r User_External_Binding) GetAttributes() (resp []datatypes.User_External_Binding_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding", "getAttributes", nil, &r.Options, &resp)
	return
}

// Retrieve Information regarding the billing item for external authentication.
func (r User_External_Binding) GetBillingItem() (resp datatypes.Billing_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding", "getBillingItem", nil, &r.Options, &resp)
	return
}

// Retrieve An optional note for identifying the external binding.
func (r User_External_Binding) GetNote() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding", "getNote", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_External_Binding) GetObject() (resp datatypes.User_External_Binding, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The type of external authentication binding.
func (r User_External_Binding) GetType() (resp datatypes.User_External_Binding_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding", "getType", nil, &r.Options, &resp)
	return
}

// Retrieve The vendor of an external authentication binding.
func (r User_External_Binding) GetVendor() (resp datatypes.User_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding", "getVendor", nil, &r.Options, &resp)
	return
}

// Update the note of an external binding.  The note is an optional property that is used to store information about a binding.
func (r User_External_Binding) UpdateNote(text *string) (resp bool, err error) {
	params := []interface{}{
		text,
	}
	err = r.Session.DoRequest("SoftLayer_User_External_Binding", "updateNote", params, &r.Options, &resp)
	return
}

// The SoftLayer_User_External_Binding_Vendor data type contains information for a single external binding vendor.  This information includes a user friendly vendor name, a unique version of the vendor name, and a unique internal identifier that can be used when creating a new external binding.
type User_External_Binding_Vendor struct {
	Session *session.Session
	Options sl.Options
}

// GetUserExternalBindingVendorService returns an instance of the User_External_Binding_Vendor SoftLayer service
func GetUserExternalBindingVendorService(sess *session.Session) User_External_Binding_Vendor {
	return User_External_Binding_Vendor{Session: sess}
}

func (r User_External_Binding_Vendor) Id(id int) User_External_Binding_Vendor {
	r.Options.Id = &id
	return r
}

func (r User_External_Binding_Vendor) Mask(mask string) User_External_Binding_Vendor {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_External_Binding_Vendor) Filter(filter string) User_External_Binding_Vendor {
	r.Options.Filter = filter
	return r
}

func (r User_External_Binding_Vendor) Limit(limit int) User_External_Binding_Vendor {
	r.Options.Limit = &limit
	return r
}

func (r User_External_Binding_Vendor) Offset(offset int) User_External_Binding_Vendor {
	r.Options.Offset = &offset
	return r
}

// getAllObjects() will return a list of the available external binding vendors that SoftLayer supports.  Use this list to select the appropriate vendor when creating a new external binding.
func (r User_External_Binding_Vendor) GetAllObjects() (resp []datatypes.User_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding_Vendor", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_External_Binding_Vendor) GetObject() (resp datatypes.User_External_Binding_Vendor, err error) {
	err = r.Session.DoRequest("SoftLayer_User_External_Binding_Vendor", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type User_Permission_Action struct {
	Session *session.Session
	Options sl.Options
}

// GetUserPermissionActionService returns an instance of the User_Permission_Action SoftLayer service
func GetUserPermissionActionService(sess *session.Session) User_Permission_Action {
	return User_Permission_Action{Session: sess}
}

func (r User_Permission_Action) Id(id int) User_Permission_Action {
	r.Options.Id = &id
	return r
}

func (r User_Permission_Action) Mask(mask string) User_Permission_Action {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Permission_Action) Filter(filter string) User_Permission_Action {
	r.Options.Filter = filter
	return r
}

func (r User_Permission_Action) Limit(limit int) User_Permission_Action {
	r.Options.Limit = &limit
	return r
}

func (r User_Permission_Action) Offset(offset int) User_Permission_Action {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Permission_Action) GetAllObjects() (resp []datatypes.User_Permission_Action, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Action", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Action) GetObject() (resp datatypes.User_Permission_Action, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Action", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type User_Permission_Group struct {
	Session *session.Session
	Options sl.Options
}

// GetUserPermissionGroupService returns an instance of the User_Permission_Group SoftLayer service
func GetUserPermissionGroupService(sess *session.Session) User_Permission_Group {
	return User_Permission_Group{Session: sess}
}

func (r User_Permission_Group) Id(id int) User_Permission_Group {
	r.Options.Id = &id
	return r
}

func (r User_Permission_Group) Mask(mask string) User_Permission_Group {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Permission_Group) Filter(filter string) User_Permission_Group {
	r.Options.Filter = filter
	return r
}

func (r User_Permission_Group) Limit(limit int) User_Permission_Group {
	r.Options.Limit = &limit
	return r
}

func (r User_Permission_Group) Offset(offset int) User_Permission_Group {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Permission_Group) AddAction(action *datatypes.User_Permission_Action) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		action,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "addAction", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) AddBulkActions(actions []datatypes.User_Permission_Action) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		actions,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "addBulkActions", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) AddBulkResourceObjects(resourceObjects []datatypes.Entity, resourceTypeKeyName *string) (resp bool, err error) {
	params := []interface{}{
		resourceObjects,
		resourceTypeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "addBulkResourceObjects", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) AddResourceObject(resourceObject *datatypes.Entity, resourceTypeKeyName *string) (resp bool, err error) {
	params := []interface{}{
		resourceObject,
		resourceTypeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "addResourceObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) CreateObject(templateObject *datatypes.User_Permission_Group) (resp datatypes.User_Permission_Group, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) EditObject(templateObject *datatypes.User_Permission_Group) (resp datatypes.User_Permission_Group, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Permission_Group) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Permission_Group) GetActions() (resp []datatypes.User_Permission_Action, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "getActions", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) GetObject() (resp datatypes.User_Permission_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Permission_Group) GetRoles() (resp []datatypes.User_Permission_Role, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "getRoles", nil, &r.Options, &resp)
	return
}

// Retrieve The type of the permission group.
func (r User_Permission_Group) GetType() (resp datatypes.User_Permission_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "getType", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) LinkRole(role *datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		role,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "linkRole", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) RemoveAction(action *datatypes.User_Permission_Action) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		action,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "removeAction", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) RemoveBulkActions(actions []datatypes.User_Permission_Action) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		actions,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "removeBulkActions", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) RemoveBulkResourceObjects(resourceObjects []datatypes.Entity, resourceTypeKeyName *string) (resp bool, err error) {
	params := []interface{}{
		resourceObjects,
		resourceTypeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "removeBulkResourceObjects", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) RemoveResourceObject(resourceObject *datatypes.Entity, resourceTypeKeyName *string) (resp bool, err error) {
	params := []interface{}{
		resourceObject,
		resourceTypeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "removeResourceObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group) UnlinkRole(role *datatypes.User_Permission_Role) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		role,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group", "unlinkRole", params, &r.Options, &resp)
	return
}

// no documentation yet
type User_Permission_Group_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetUserPermissionGroupTypeService returns an instance of the User_Permission_Group_Type SoftLayer service
func GetUserPermissionGroupTypeService(sess *session.Session) User_Permission_Group_Type {
	return User_Permission_Group_Type{Session: sess}
}

func (r User_Permission_Group_Type) Id(id int) User_Permission_Group_Type {
	r.Options.Id = &id
	return r
}

func (r User_Permission_Group_Type) Mask(mask string) User_Permission_Group_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Permission_Group_Type) Filter(filter string) User_Permission_Group_Type {
	r.Options.Filter = filter
	return r
}

func (r User_Permission_Group_Type) Limit(limit int) User_Permission_Group_Type {
	r.Options.Limit = &limit
	return r
}

func (r User_Permission_Group_Type) Offset(offset int) User_Permission_Group_Type {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r User_Permission_Group_Type) GetGroups() (resp []datatypes.User_Permission_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group_Type", "getGroups", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Group_Type) GetObject() (resp datatypes.User_Permission_Group_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Group_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type User_Permission_Role struct {
	Session *session.Session
	Options sl.Options
}

// GetUserPermissionRoleService returns an instance of the User_Permission_Role SoftLayer service
func GetUserPermissionRoleService(sess *session.Session) User_Permission_Role {
	return User_Permission_Role{Session: sess}
}

func (r User_Permission_Role) Id(id int) User_Permission_Role {
	r.Options.Id = &id
	return r
}

func (r User_Permission_Role) Mask(mask string) User_Permission_Role {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Permission_Role) Filter(filter string) User_Permission_Role {
	r.Options.Filter = filter
	return r
}

func (r User_Permission_Role) Limit(limit int) User_Permission_Role {
	r.Options.Limit = &limit
	return r
}

func (r User_Permission_Role) Offset(offset int) User_Permission_Role {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r User_Permission_Role) AddUser(user *datatypes.User_Customer) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		user,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "addUser", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Role) CreateObject(templateObject *datatypes.User_Permission_Role) (resp datatypes.User_Permission_Role, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Role) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Role) EditObject(templateObject *datatypes.User_Permission_Role) (resp datatypes.User_Permission_Role, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Permission_Role) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Permission_Role) GetActions() (resp []datatypes.User_Permission_Action, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "getActions", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Permission_Role) GetGroups() (resp []datatypes.User_Permission_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "getGroups", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Role) GetObject() (resp datatypes.User_Permission_Role, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r User_Permission_Role) GetUsers() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "getUsers", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Role) LinkGroup(group *datatypes.User_Permission_Group) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		group,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "linkGroup", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Role) RemoveUser(user *datatypes.User_Customer) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		user,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "removeUser", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r User_Permission_Role) UnlinkGroup(group *datatypes.User_Permission_Group) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		group,
	}
	err = r.Session.DoRequest("SoftLayer_User_Permission_Role", "unlinkGroup", params, &r.Options, &resp)
	return
}

// The SoftLayer_User_Security_Question data type contains questions.
type User_Security_Question struct {
	Session *session.Session
	Options sl.Options
}

// GetUserSecurityQuestionService returns an instance of the User_Security_Question SoftLayer service
func GetUserSecurityQuestionService(sess *session.Session) User_Security_Question {
	return User_Security_Question{Session: sess}
}

func (r User_Security_Question) Id(id int) User_Security_Question {
	r.Options.Id = &id
	return r
}

func (r User_Security_Question) Mask(mask string) User_Security_Question {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r User_Security_Question) Filter(filter string) User_Security_Question {
	r.Options.Filter = filter
	return r
}

func (r User_Security_Question) Limit(limit int) User_Security_Question {
	r.Options.Limit = &limit
	return r
}

func (r User_Security_Question) Offset(offset int) User_Security_Question {
	r.Options.Offset = &offset
	return r
}

// Retrieve all viewable security questions.
func (r User_Security_Question) GetAllObjects() (resp []datatypes.User_Security_Question, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Security_Question", "getAllObjects", nil, &r.Options, &resp)
	return
}

// getAllObjects retrieves all the SoftLayer_User_Security_Question objects where it is set to be viewable.
func (r User_Security_Question) GetObject() (resp datatypes.User_Security_Question, err error) {
	err = r.Session.DoRequest("SoftLayer_User_Security_Question", "getObject", nil, &r.Options, &resp)
	return
}
