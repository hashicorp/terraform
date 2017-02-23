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

// This class represents a login/logout sheet for facility visitors.
type User_Access_Facility_Log struct {
	Entity

	// This is the account associated with the log entry. For users under a customer's account, it is the customer's account. For contractors and others visiting a colocation area, it is the account associated with the area they visited.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// This is the account associated with a log record. For a customer logging into a datacenter, this is the customer's account. For a contractor or any other guest logging into a customer's cabinet or colocation cage, this is the customer's account.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// This is the location of the facility.
	Datacenter *Location `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// This is a short description of why the person is at the location.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// This is the colocation hardware that was visited.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// no documentation yet
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// This is the type of person entering the facility.
	LogType *User_Access_Facility_Log_Type `json:"logType,omitempty" xmlrpc:"logType,omitempty"`

	// This is the date and time the person arrived.
	TimeIn *Time `json:"timeIn,omitempty" xmlrpc:"timeIn,omitempty"`

	// no documentation yet
	TimeOut *Time `json:"timeOut,omitempty" xmlrpc:"timeOut,omitempty"`

	// no documentation yet
	Visitor *Entity `json:"visitor,omitempty" xmlrpc:"visitor,omitempty"`
}

// no documentation yet
type User_Access_Facility_Log_Type struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// This class represents a facility visitor that is not an active employee or customer.
type User_Access_Facility_Visitor struct {
	Entity

	// no documentation yet
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// no documentation yet
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// no documentation yet
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// no documentation yet
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// no documentation yet
	VisitorType *User_Access_Facility_Visitor_Type `json:"visitorType,omitempty" xmlrpc:"visitorType,omitempty"`
}

// no documentation yet
type User_Access_Facility_Visitor_Type struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_User_Customer data type contains general information relating to a single SoftLayer customer portal user. Personal information in this type such as names, addresses, and phone numbers are not necessarily associated with the customer account the user is assigned to.
type User_Customer struct {
	User_Interface

	// The customer account that a user belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A portal user's associated [[SoftLayer_Account|customer account]] id.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of
	ActionCount *uint `json:"actionCount,omitempty" xmlrpc:"actionCount,omitempty"`

	// no documentation yet
	Actions []User_Permission_Action `json:"actions,omitempty" xmlrpc:"actions,omitempty"`

	// A count of a portal user's additional email addresses. These email addresses are contacted when updates are made to support tickets.
	AdditionalEmailCount *uint `json:"additionalEmailCount,omitempty" xmlrpc:"additionalEmailCount,omitempty"`

	// A portal user's additional email addresses. These email addresses are contacted when updates are made to support tickets.
	AdditionalEmails []User_Customer_AdditionalEmail `json:"additionalEmails,omitempty" xmlrpc:"additionalEmails,omitempty"`

	// The first line of the mailing address belonging to a portal user.
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// The second line of the mailing address belonging to a portal user.
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// A portal user's AOL Instant Messenger screen name.
	Aim *string `json:"aim,omitempty" xmlrpc:"aim,omitempty"`

	// A portal user's secondary phone number.
	AlternatePhone *string `json:"alternatePhone,omitempty" xmlrpc:"alternatePhone,omitempty"`

	// A count of a portal user's API Authentication keys. There is a max limit of two API keys per user.
	ApiAuthenticationKeyCount *uint `json:"apiAuthenticationKeyCount,omitempty" xmlrpc:"apiAuthenticationKeyCount,omitempty"`

	// A portal user's API Authentication keys. There is a max limit of two API keys per user.
	ApiAuthenticationKeys []User_Customer_ApiAuthentication `json:"apiAuthenticationKeys,omitempty" xmlrpc:"apiAuthenticationKeys,omitempty"`

	// The authentication token used for logging into the SoftLayer customer portal.
	AuthenticationToken *Container_User_Authentication_Token `json:"authenticationToken,omitempty" xmlrpc:"authenticationToken,omitempty"`

	// A count of the CDN accounts associated with a portal user.
	CdnAccountCount *uint `json:"cdnAccountCount,omitempty" xmlrpc:"cdnAccountCount,omitempty"`

	// The CDN accounts associated with a portal user.
	CdnAccounts []Network_ContentDelivery_Account `json:"cdnAccounts,omitempty" xmlrpc:"cdnAccounts,omitempty"`

	// A count of a portal user's child users. Some portal users may not have child users.
	ChildUserCount *uint `json:"childUserCount,omitempty" xmlrpc:"childUserCount,omitempty"`

	// A portal user's child users. Some portal users may not have child users.
	ChildUsers []User_Customer `json:"childUsers,omitempty" xmlrpc:"childUsers,omitempty"`

	// The city of the mailing address belonging to a portal user.
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// A count of an user's associated closed tickets.
	ClosedTicketCount *uint `json:"closedTicketCount,omitempty" xmlrpc:"closedTicketCount,omitempty"`

	// An user's associated closed tickets.
	ClosedTickets []Ticket `json:"closedTickets,omitempty" xmlrpc:"closedTickets,omitempty"`

	// A portal user's associated company. This may not be the same company as the customer that owns this portal user.
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// A two-letter abbreviation of the country in the mailing address belonging to a portal user.
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// The date a portal user's record was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Whether a portal user's time zone is affected by Daylight Savings Time.
	DaylightSavingsTimeFlag *bool `json:"daylightSavingsTimeFlag,omitempty" xmlrpc:"daylightSavingsTimeFlag,omitempty"`

	// Flag used to deny access to all hardware and cloud computing instances upon user creation.
	DenyAllResourceAccessOnCreateFlag *bool `json:"denyAllResourceAccessOnCreateFlag,omitempty" xmlrpc:"denyAllResourceAccessOnCreateFlag,omitempty"`

	// no documentation yet
	DisplayName *string `json:"displayName,omitempty" xmlrpc:"displayName,omitempty"`

	// A portal user's email address.
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// A count of the external authentication bindings that link an external identifier to a SoftLayer user.
	ExternalBindingCount *uint `json:"externalBindingCount,omitempty" xmlrpc:"externalBindingCount,omitempty"`

	// The external authentication bindings that link an external identifier to a SoftLayer user.
	ExternalBindings []User_External_Binding `json:"externalBindings,omitempty" xmlrpc:"externalBindings,omitempty"`

	// A portal user's first name.
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// A user's password for the SoftLayer forums, hashed for auto-login capability from the SoftLayer customer portal
	ForumPasswordHash *string `json:"forumPasswordHash,omitempty" xmlrpc:"forumPasswordHash,omitempty"`

	// A portal user's accessible hardware. These permissions control which hardware a user has access to in the SoftLayer customer portal.
	Hardware []Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// A count of a portal user's accessible hardware. These permissions control which hardware a user has access to in the SoftLayer customer portal.
	HardwareCount *uint `json:"hardwareCount,omitempty" xmlrpc:"hardwareCount,omitempty"`

	// A count of hardware notifications associated with this user. A hardware notification links a user to a piece of hardware, and that user will be notified if any monitors on that hardware fail, if the monitors have a status of 'Notify User'.
	HardwareNotificationCount *uint `json:"hardwareNotificationCount,omitempty" xmlrpc:"hardwareNotificationCount,omitempty"`

	// Hardware notifications associated with this user. A hardware notification links a user to a piece of hardware, and that user will be notified if any monitors on that hardware fail, if the monitors have a status of 'Notify User'.
	HardwareNotifications []User_Customer_Notification_Hardware `json:"hardwareNotifications,omitempty" xmlrpc:"hardwareNotifications,omitempty"`

	// Whether or not a user has acknowledged the support policy.
	HasAcknowledgedSupportPolicyFlag *bool `json:"hasAcknowledgedSupportPolicyFlag,omitempty" xmlrpc:"hasAcknowledgedSupportPolicyFlag,omitempty"`

	// Whether or not a portal user has access to all hardware on their account.
	HasFullHardwareAccessFlag *bool `json:"hasFullHardwareAccessFlag,omitempty" xmlrpc:"hasFullHardwareAccessFlag,omitempty"`

	// Whether or not a portal user has access to all hardware on their account.
	HasFullVirtualGuestAccessFlag *bool `json:"hasFullVirtualGuestAccessFlag,omitempty" xmlrpc:"hasFullVirtualGuestAccessFlag,omitempty"`

	// A portal user's ICQ UIN.
	Icq *string `json:"icq,omitempty" xmlrpc:"icq,omitempty"`

	// A portal user's internal identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The IP addresses or IP ranges from which a user may login to the SoftLayer customer portal. Specify subnets in CIDR format and separate multiple addresses and subnets by commas. You may combine IPv4 and IPv6 addresses and subnets, for example: 192.168.0.0/16,fe80:021b::0/64.
	IpAddressRestriction *string `json:"ipAddressRestriction,omitempty" xmlrpc:"ipAddressRestriction,omitempty"`

	// no documentation yet
	IsMasterUserFlag *bool `json:"isMasterUserFlag,omitempty" xmlrpc:"isMasterUserFlag,omitempty"`

	// A portal user's last name.
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// A count of
	LayoutProfileCount *uint `json:"layoutProfileCount,omitempty" xmlrpc:"layoutProfileCount,omitempty"`

	// no documentation yet
	LayoutProfiles []Layout_Profile `json:"layoutProfiles,omitempty" xmlrpc:"layoutProfiles,omitempty"`

	// A user's locale. Locale holds user's language and region information.
	Locale *Locale `json:"locale,omitempty" xmlrpc:"locale,omitempty"`

	// A portal user's associated [[SoftLayer_Locale|locale]] id.
	LocaleId *int `json:"localeId,omitempty" xmlrpc:"localeId,omitempty"`

	// A count of a user's attempts to log into the SoftLayer customer portal.
	LoginAttemptCount *uint `json:"loginAttemptCount,omitempty" xmlrpc:"loginAttemptCount,omitempty"`

	// A user's attempts to log into the SoftLayer customer portal.
	LoginAttempts []User_Customer_Access_Authentication `json:"loginAttempts,omitempty" xmlrpc:"loginAttempts,omitempty"`

	// Determines if this portal user is managed by SAML federation.
	ManagedByFederationFlag *bool `json:"managedByFederationFlag,omitempty" xmlrpc:"managedByFederationFlag,omitempty"`

	// Determines if this portal user is managed by IBMid federation.
	ManagedByOpenIdConnectFlag *bool `json:"managedByOpenIdConnectFlag,omitempty" xmlrpc:"managedByOpenIdConnectFlag,omitempty"`

	// A count of a portal user's associated mobile device profiles.
	MobileDeviceCount *uint `json:"mobileDeviceCount,omitempty" xmlrpc:"mobileDeviceCount,omitempty"`

	// A portal user's associated mobile device profiles.
	MobileDevices []User_Customer_MobileDevice `json:"mobileDevices,omitempty" xmlrpc:"mobileDevices,omitempty"`

	// The date a portal user's record was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A portal user's MSN address.
	Msn *string `json:"msn,omitempty" xmlrpc:"msn,omitempty"`

	// no documentation yet
	NameId *string `json:"nameId,omitempty" xmlrpc:"nameId,omitempty"`

	// A count of notification subscription records for the user.
	NotificationSubscriberCount *uint `json:"notificationSubscriberCount,omitempty" xmlrpc:"notificationSubscriberCount,omitempty"`

	// Notification subscription records for the user.
	NotificationSubscribers []Notification_Subscriber `json:"notificationSubscribers,omitempty" xmlrpc:"notificationSubscribers,omitempty"`

	// A portal user's office phone number.
	OfficePhone *string `json:"officePhone,omitempty" xmlrpc:"officePhone,omitempty"`

	// The BlueID username associated to with this user, if the account is managed by OpenIDConnect / BlueID federation
	OpenIdConnectUserName *string `json:"openIdConnectUserName,omitempty" xmlrpc:"openIdConnectUserName,omitempty"`

	// A count of an user's associated open tickets.
	OpenTicketCount *uint `json:"openTicketCount,omitempty" xmlrpc:"openTicketCount,omitempty"`

	// An user's associated open tickets.
	OpenTickets []Ticket `json:"openTickets,omitempty" xmlrpc:"openTickets,omitempty"`

	// A count of a portal user's vpn accessible subnets.
	OverrideCount *uint `json:"overrideCount,omitempty" xmlrpc:"overrideCount,omitempty"`

	// A portal user's vpn accessible subnets.
	Overrides []Network_Service_Vpn_Overrides `json:"overrides,omitempty" xmlrpc:"overrides,omitempty"`

	// A portal user's parent user. If a SoftLayer_User_Customer has a null parentId property then it doesn't have a parent user.
	Parent *User_Customer `json:"parent,omitempty" xmlrpc:"parent,omitempty"`

	// A portal user's parent user. Id a users parentId is ''null'' then it doesn't have a parent user in the customer portal.
	ParentId *int `json:"parentId,omitempty" xmlrpc:"parentId,omitempty"`

	// The expiration date for the user's password
	PasswordExpireDate *Time `json:"passwordExpireDate,omitempty" xmlrpc:"passwordExpireDate,omitempty"`

	// A count of a portal user's permissions. These permissions control that user's access to functions within the SoftLayer customer portal and API.
	PermissionCount *uint `json:"permissionCount,omitempty" xmlrpc:"permissionCount,omitempty"`

	// no documentation yet
	PermissionSystemVersion *int `json:"permissionSystemVersion,omitempty" xmlrpc:"permissionSystemVersion,omitempty"`

	// A portal user's permissions. These permissions control that user's access to functions within the SoftLayer customer portal and API.
	Permissions []User_Customer_CustomerPermission_Permission `json:"permissions,omitempty" xmlrpc:"permissions,omitempty"`

	// The postal code of the mailing address belonging to an portal user.
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// Whether a portal user may connect to the SoftLayer private network via PPTP VPN or not.
	PptpVpnAllowedFlag *bool `json:"pptpVpnAllowedFlag,omitempty" xmlrpc:"pptpVpnAllowedFlag,omitempty"`

	// A count of
	PreferenceCount *uint `json:"preferenceCount,omitempty" xmlrpc:"preferenceCount,omitempty"`

	// no documentation yet
	Preferences []User_Preference `json:"preferences,omitempty" xmlrpc:"preferences,omitempty"`

	// A count of
	RoleCount *uint `json:"roleCount,omitempty" xmlrpc:"roleCount,omitempty"`

	// no documentation yet
	Roles []User_Permission_Role `json:"roles,omitempty" xmlrpc:"roles,omitempty"`

	// no documentation yet
	SalesforceUserLink *User_Customer_Link `json:"salesforceUserLink,omitempty" xmlrpc:"salesforceUserLink,omitempty"`

	// no documentation yet
	SavedId *string `json:"savedId,omitempty" xmlrpc:"savedId,omitempty"`

	// Whether a user may change their security options (IP restriction, password expiration, or enforce security questions on login) which were pre-selected by their account's master user.
	SecondaryLoginManagementFlag *bool `json:"secondaryLoginManagementFlag,omitempty" xmlrpc:"secondaryLoginManagementFlag,omitempty"`

	// Whether a user is required to answer a security question when logging into the SoftLayer customer portal.
	SecondaryLoginRequiredFlag *bool `json:"secondaryLoginRequiredFlag,omitempty" xmlrpc:"secondaryLoginRequiredFlag,omitempty"`

	// The date when a user's password was last updated.
	SecondaryPasswordModifyDate *Time `json:"secondaryPasswordModifyDate,omitempty" xmlrpc:"secondaryPasswordModifyDate,omitempty"`

	// The number of days for which a user's password is active.
	SecondaryPasswordTimeoutDays *int `json:"secondaryPasswordTimeoutDays,omitempty" xmlrpc:"secondaryPasswordTimeoutDays,omitempty"`

	// A count of a portal user's security question answers. Some portal users may not have security answers or may not be configured to require answering a security question on login.
	SecurityAnswerCount *uint `json:"securityAnswerCount,omitempty" xmlrpc:"securityAnswerCount,omitempty"`

	// A portal user's security question answers. Some portal users may not have security answers or may not be configured to require answering a security question on login.
	SecurityAnswers []User_Customer_Security_Answer `json:"securityAnswers,omitempty" xmlrpc:"securityAnswers,omitempty"`

	// A phone number that can receive SMS text messages for this portal user.
	Sms *string `json:"sms,omitempty" xmlrpc:"sms,omitempty"`

	// Whether a portal user may connect to the SoftLayer private network via SSL VPN or not.
	SslVpnAllowedFlag *bool `json:"sslVpnAllowedFlag,omitempty" xmlrpc:"sslVpnAllowedFlag,omitempty"`

	// A two-letter abbreviation of the state in the mailing address belonging to a portal user. If a user does not reside in a province then this is typically blank.
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// The date a portal users record's last status change.
	StatusDate *Time `json:"statusDate,omitempty" xmlrpc:"statusDate,omitempty"`

	// A count of a user's notification subscription records.
	SubscriberCount *uint `json:"subscriberCount,omitempty" xmlrpc:"subscriberCount,omitempty"`

	// A user's notification subscription records.
	Subscribers []Notification_User_Subscriber `json:"subscribers,omitempty" xmlrpc:"subscribers,omitempty"`

	// A count of a user's successful attempts to log into the SoftLayer customer portal.
	SuccessfulLoginCount *uint `json:"successfulLoginCount,omitempty" xmlrpc:"successfulLoginCount,omitempty"`

	// A user's successful attempts to log into the SoftLayer customer portal.
	SuccessfulLogins []User_Customer_Access_Authentication `json:"successfulLogins,omitempty" xmlrpc:"successfulLogins,omitempty"`

	// Whether or not a user is required to acknowledge the support policy for portal access.
	SupportPolicyAcknowledgementRequiredFlag *int `json:"supportPolicyAcknowledgementRequiredFlag,omitempty" xmlrpc:"supportPolicyAcknowledgementRequiredFlag,omitempty"`

	// A count of the surveys that a user has taken in the SoftLayer customer portal.
	SurveyCount *uint `json:"surveyCount,omitempty" xmlrpc:"surveyCount,omitempty"`

	// Whether or not a user must take a brief survey the next time they log into the SoftLayer customer portal.
	SurveyRequiredFlag *bool `json:"surveyRequiredFlag,omitempty" xmlrpc:"surveyRequiredFlag,omitempty"`

	// The surveys that a user has taken in the SoftLayer customer portal.
	Surveys []Survey `json:"surveys,omitempty" xmlrpc:"surveys,omitempty"`

	// A count of an user's associated tickets.
	TicketCount *uint `json:"ticketCount,omitempty" xmlrpc:"ticketCount,omitempty"`

	// An user's associated tickets.
	Tickets []Ticket `json:"tickets,omitempty" xmlrpc:"tickets,omitempty"`

	// A portal user's time zone.
	Timezone *Locale_Timezone `json:"timezone,omitempty" xmlrpc:"timezone,omitempty"`

	// A portal user's time zone.
	TimezoneId *int `json:"timezoneId,omitempty" xmlrpc:"timezoneId,omitempty"`

	// A count of a user's unsuccessful attempts to log into the SoftLayer customer portal.
	UnsuccessfulLoginCount *uint `json:"unsuccessfulLoginCount,omitempty" xmlrpc:"unsuccessfulLoginCount,omitempty"`

	// A user's unsuccessful attempts to log into the SoftLayer customer portal.
	UnsuccessfulLogins []User_Customer_Access_Authentication `json:"unsuccessfulLogins,omitempty" xmlrpc:"unsuccessfulLogins,omitempty"`

	// A count of
	UserLinkCount *uint `json:"userLinkCount,omitempty" xmlrpc:"userLinkCount,omitempty"`

	// no documentation yet
	UserLinks []User_Customer_Link `json:"userLinks,omitempty" xmlrpc:"userLinks,omitempty"`

	// A portal user's status, which controls overall access to the SoftLayer customer portal and VPN access to the private network.
	UserStatus *User_Customer_Status `json:"userStatus,omitempty" xmlrpc:"userStatus,omitempty"`

	// A number reflecting the state of a portal user.
	UserStatusId *int `json:"userStatusId,omitempty" xmlrpc:"userStatusId,omitempty"`

	// A portal user's username.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`

	// A count of a portal user's accessible CloudLayer Computing Instances. These permissions control which CloudLayer Computing Instances a user has access to in the SoftLayer customer portal.
	VirtualGuestCount *uint `json:"virtualGuestCount,omitempty" xmlrpc:"virtualGuestCount,omitempty"`

	// A portal user's accessible CloudLayer Computing Instances. These permissions control which CloudLayer Computing Instances a user has access to in the SoftLayer customer portal.
	VirtualGuests []Virtual_Guest `json:"virtualGuests,omitempty" xmlrpc:"virtualGuests,omitempty"`

	// Whether a portal user vpn subnets have been manual configured.
	VpnManualConfig *bool `json:"vpnManualConfig,omitempty" xmlrpc:"vpnManualConfig,omitempty"`

	// A portal user's Yahoo! Chat name.
	Yahoo *string `json:"yahoo,omitempty" xmlrpc:"yahoo,omitempty"`
}

// SoftLayer_User_Customer_Access_Authentication models a single attempt to log into the SoftLayer customer portal. A SoftLayer_User_Customer_Access_Authentication record is created every time a user attempts to log into the portal. Use this service to audit your users' portal activity and diagnose potential security breaches of your SoftLayer portal accounts.
//
// Unsuccessful login attempts can be caused by an incorrect password, failing to answer or not answering a login security question if the user has them configured, or attempting to log in from an IP address outside of the user's IP address restriction list.
//
// SoftLayer employees periodically log into our customer portal as users to diagnose portal issues, verify settings and configuration, and to perform maintenance on your account or services. SoftLayer employees only log into customer accounts from the following IP ranges:
// * 2607:f0d0:1000::/48
// * 2607:f0d0:2000::/48
// * 2607:f0d0:3000::/48
// * 66.228.118.67/32
// * 66.228.118.86/32
type User_Customer_Access_Authentication struct {
	Entity

	// The date of an attempt to log into the SoftLayer customer portal.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The IP address of the user who attempted to log into the SoftLayer customer portal.
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// Whether an attempt to log into the SoftLayer customer portal was successful or not.
	SuccessFlag *bool `json:"successFlag,omitempty" xmlrpc:"successFlag,omitempty"`

	// The user who has attempted to log into the SoftLayer customer portal.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// The internal identifier of the user who attempted to log into the SoftLayer customer portal.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// The username used when attempting to log into the SoftLayer customer portal
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// The SoftLayer_User_Customer_AdditionalEmail data type contains the additional email for use in ticket update notifications.
type User_Customer_AdditionalEmail struct {
	Entity

	// Email assigned to user for use in ticket update notifications.
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// The portal user that owns this additional email address.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// An internal identifier for the portal user who this additional email belongs to.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// The SoftLayer_User_Customer_ApiAuthentication type contains user's authentication key(s).
type User_Customer_ApiAuthentication struct {
	Entity

	// The user's authentication key for API access.
	AuthenticationKey *string `json:"authenticationKey,omitempty" xmlrpc:"authenticationKey,omitempty"`

	// The user's API authentication identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The IP addresses or IP ranges from which this user may access the SoftLayer API. Specify subnets in CIDR format and separate multiple addresses and subnets by commas. You may combine IPv4 and IPv6 addresses and subnets, for example: 192.168.0.0/16,fe80:021b::0/64.
	IpAddressRestriction *string `json:"ipAddressRestriction,omitempty" xmlrpc:"ipAddressRestriction,omitempty"`

	// The user's authentication key modification date.
	TimestampKey *int `json:"timestampKey,omitempty" xmlrpc:"timestampKey,omitempty"`

	// The user who owns the api authentication key.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// The user's identifying number.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// Each SoftLayer portal account is assigned a series of permissions that determine what access the user has to functions within the SoftLayer customer portal. This status is reflected in the SoftLayer_User_Customer_Status data type. Permissions differ from user status in that user status applies globally to the portal while user permissions are applied to specific portal functions.
type User_Customer_CustomerPermission_Permission struct {
	Entity

	// A user permission's short name.
	Key *string `json:"key,omitempty" xmlrpc:"key,omitempty"`

	// A user permission's key name.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A user permission's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_User_Customer_External_Binding data type contains general information for a single external binding.  This includes the 3rd party vendor, type of binding, and a unique identifier and password that is used to authenticate against the 3rd party service.
type User_Customer_External_Binding struct {
	User_External_Binding

	// The SoftLayer user that the external authentication binding belongs to.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`
}

// The SoftLayer_User_Customer_External_Binding_Attribute data type contains the value for a single attribute associated with an external binding. External binding attributes contain additional information about an external binding.  An attribute can be generic or specific to a 3rd party vendor.  For example these attributes relate to Verisign:
// *Credential Type
// *Credential State
// *Credential Expiration Date
// *Credential Last Update Date
type User_Customer_External_Binding_Attribute struct {
	User_External_Binding_Attribute
}

// The SoftLayer_User_Customer_External_Binding_Phone data type contains information about an external binding that uses a phone call, SMS or mobile app for 2 form factor authentication. The external binding information is used when a SoftLayer customer logs into the SoftLayer customer portal or VPN to authenticate them against a trusted 3rd party, in this case using a mobile phone, mobile phone application or land-line phone.
//
// SoftLayer users with an active external binding will be prohibited from using the API for security reasons.
type User_Customer_External_Binding_Phone struct {
	User_Customer_External_Binding

	// The current external binding status. It can be "ACTIVE" or "BLOCKED".
	BindingStatus *string `json:"bindingStatus,omitempty" xmlrpc:"bindingStatus,omitempty"`

	// no documentation yet
	PinLength *string `json:"pinLength,omitempty" xmlrpc:"pinLength,omitempty"`
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
	User_Customer_External_Binding
}

// The SoftLayer_User_Customer_External_Binding_Type data type contains information relating to a type of external authentication binding.  It contains a user friendly name as well as a unique key name.
type User_Customer_External_Binding_Type struct {
	User_External_Binding_Type
}

// The SoftLayer_User_Customer_External_Binding_Vendor data type contains information for a single external binding vendor.  This information includes a user friendly vendor name, a unique version of the vendor name, and a unique internal identifier that can be used when creating a new external binding.
type User_Customer_External_Binding_Vendor struct {
	User_External_Binding_Vendor
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
	User_Customer_External_Binding

	// The date that a VeriSign credential expires.
	CredentialExpirationDate *string `json:"credentialExpirationDate,omitempty" xmlrpc:"credentialExpirationDate,omitempty"`

	// The last time a VeriSign credential was updated.
	CredentialLastUpdateDate *string `json:"credentialLastUpdateDate,omitempty" xmlrpc:"credentialLastUpdateDate,omitempty"`

	// The current state of a VeriSign credential. This can be 'Enabled', 'Disabled', or 'Locked'.
	CredentialState *string `json:"credentialState,omitempty" xmlrpc:"credentialState,omitempty"`

	// The type of VeriSign credential. This can be either 'Hardware' or 'Software'.
	CredentialType *string `json:"credentialType,omitempty" xmlrpc:"credentialType,omitempty"`
}

// no documentation yet
type User_Customer_Invitation struct {
	Entity

	// no documentation yet
	Code *string `json:"code,omitempty" xmlrpc:"code,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	CreatorId *int `json:"creatorId,omitempty" xmlrpc:"creatorId,omitempty"`

	// no documentation yet
	CreatorType *string `json:"creatorType,omitempty" xmlrpc:"creatorType,omitempty"`

	// no documentation yet
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// no documentation yet
	ExistingBlueIdFlag *int `json:"existingBlueIdFlag,omitempty" xmlrpc:"existingBlueIdFlag,omitempty"`

	// no documentation yet
	ExpirationDate *Time `json:"expirationDate,omitempty" xmlrpc:"expirationDate,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	IsFederatedEmailDomainFlag *int `json:"isFederatedEmailDomainFlag,omitempty" xmlrpc:"isFederatedEmailDomainFlag,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	ResponseDate *Time `json:"responseDate,omitempty" xmlrpc:"responseDate,omitempty"`

	// no documentation yet
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// no documentation yet
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// no documentation yet
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// no documentation yet
type User_Customer_Link struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	DefaultFlag *int `json:"defaultFlag,omitempty" xmlrpc:"defaultFlag,omitempty"`

	// no documentation yet
	DestinationUserAlphanumericId *string `json:"destinationUserAlphanumericId,omitempty" xmlrpc:"destinationUserAlphanumericId,omitempty"`

	// no documentation yet
	DestinationUserId *int `json:"destinationUserId,omitempty" xmlrpc:"destinationUserId,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`

	// no documentation yet
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`

	// no documentation yet
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// no documentation yet
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// no documentation yet
type User_Customer_Link_ThePlanet struct {
	User_Customer_Link
}

// This class represents a mobile device belonging to a user.  The device can be a phone, tablet, or possibly even some Android based net books.  The purpose is to tie just enough info with the device and the user to enable push notifications through non-softlayer entities (Google, Apple, RIM).
type User_Customer_MobileDevice struct {
	Entity

	// A count of notification subscriptions available to a mobile device.
	AvailablePushNotificationSubscriptionCount *uint `json:"availablePushNotificationSubscriptionCount,omitempty" xmlrpc:"availablePushNotificationSubscriptionCount,omitempty"`

	// Notification subscriptions available to a mobile device.
	AvailablePushNotificationSubscriptions []Notification `json:"availablePushNotificationSubscriptions,omitempty" xmlrpc:"availablePushNotificationSubscriptions,omitempty"`

	// Created date for the record.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The user this mobile device belongs to.
	Customer *User_Customer `json:"customer,omitempty" xmlrpc:"customer,omitempty"`

	// The device resolution formatted width x height
	DisplayResolutionXxY *string `json:"displayResolutionXxY,omitempty" xmlrpc:"displayResolutionXxY,omitempty"`

	// Record Identifier
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Device type identifier.
	MobileDeviceTypeId *int `json:"mobileDeviceTypeId,omitempty" xmlrpc:"mobileDeviceTypeId,omitempty"`

	// Mobile OS identifier.
	MobileOperatingSystemId *int `json:"mobileOperatingSystemId,omitempty" xmlrpc:"mobileOperatingSystemId,omitempty"`

	// Device model number
	ModelNumber *string `json:"modelNumber,omitempty" xmlrpc:"modelNumber,omitempty"`

	// Last modify date for the record.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of the device the user is using.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The operating system this device is using
	OperatingSystem *User_Customer_MobileDevice_OperatingSystem `json:"operatingSystem,omitempty" xmlrpc:"operatingSystem,omitempty"`

	// Device phone number
	PhoneNumber *string `json:"phoneNumber,omitempty" xmlrpc:"phoneNumber,omitempty"`

	// A count of notification subscriptions attached to a mobile device.
	PushNotificationSubscriptionCount *uint `json:"pushNotificationSubscriptionCount,omitempty" xmlrpc:"pushNotificationSubscriptionCount,omitempty"`

	// Notification subscriptions attached to a mobile device.
	PushNotificationSubscriptions []Notification_User_Subscriber `json:"pushNotificationSubscriptions,omitempty" xmlrpc:"pushNotificationSubscriptions,omitempty"`

	// Device serial number
	SerialNumber *string `json:"serialNumber,omitempty" xmlrpc:"serialNumber,omitempty"`

	// The token that is provided by the mobile device.
	Token *string `json:"token,omitempty" xmlrpc:"token,omitempty"`

	// The type of device this user is using
	Type *User_Customer_MobileDevice_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// User Identifier
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// This class represents the mobile operating system installed on a user's registered mobile device. It assists us when determining the how to get a push notification to the user.
type User_Customer_MobileDevice_OperatingSystem struct {
	Entity

	// Build revision number of the operating system.
	BuildVersion *int `json:"buildVersion,omitempty" xmlrpc:"buildVersion,omitempty"`

	// Create date of the record.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Description of the mobile operating system..
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Indentifier for the record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Major revision number of the operating system.
	MajorVersion *int `json:"majorVersion,omitempty" xmlrpc:"majorVersion,omitempty"`

	// Minor revision number of the operating system.
	MinorVersion *int `json:"minorVersion,omitempty" xmlrpc:"minorVersion,omitempty"`

	// Modify date of the record.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// Name of the mobile operating system.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Describes a supported class of mobile device. In this the word class is used in the context of classes of consumer electronic devices, the two most prominent examples being mobile phones and tablets.
type User_Customer_MobileDevice_Type struct {
	Entity

	// Record create date.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A description of the device
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Indentifier for record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Last modify date for record.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The common name of the device.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The Customer_Notification_Hardware object stores links between customers and the hardware devices they wish to monitor.  This link is not enough, the user must be sure to also create SoftLayer_Network_Monitor_Version1_Query_Host instance with the response action set to "notify users" in order for the users linked to that hardware object to be notified on failure.
type User_Customer_Notification_Hardware struct {
	Entity

	// The hardware object that will be monitored.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The ID of the Hardware object that is to be monitored.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// The unique identifier for this object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The user that will be notified when the associated hardware object fails a monitoring instance.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// The ID of the SoftLayer_User_Customer object that represents the user to be notified on monitoring failure.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// The SoftLayer_User_Customer_Notification_Virtual_Guest object stores links between customers and the virtual guests they wish to monitor.  This link is not enough, the user must be sure to also create SoftLayer_Network_Monitor_Version1_Query_Host instance with the response action set to "notify users" in order for the users linked to that hardware object to be notified on failure.
type User_Customer_Notification_Virtual_Guest struct {
	Entity

	// The virtual guest object that will be monitored.
	Guest *Virtual_Guest `json:"guest,omitempty" xmlrpc:"guest,omitempty"`

	// The ID of the virtual guest object that is to be monitored.
	GuestId *int `json:"guestId,omitempty" xmlrpc:"guestId,omitempty"`

	// The unique identifier for this object
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The user that will be notified when the associated virtual guest object fails a monitoring instance.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// The ID of the SoftLayer_User_Customer object that represents the user to be notified on monitoring failure.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// no documentation yet
type User_Customer_OpenIdConnect struct {
	User_Customer
}

// no documentation yet
type User_Customer_Prospect struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A count of
	AssignedEmployeeCount *uint `json:"assignedEmployeeCount,omitempty" xmlrpc:"assignedEmployeeCount,omitempty"`

	// no documentation yet
	AssignedEmployees []User_Employee `json:"assignedEmployees,omitempty" xmlrpc:"assignedEmployees,omitempty"`

	// A count of
	QuoteCount *uint `json:"quoteCount,omitempty" xmlrpc:"quoteCount,omitempty"`

	// no documentation yet
	Quotes []Billing_Order_Quote `json:"quotes,omitempty" xmlrpc:"quotes,omitempty"`

	// no documentation yet
	Type *User_Customer_Prospect_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// Contains user information for Service Provider Enrollment.
type User_Customer_Prospect_ServiceProvider_EnrollRequest struct {
	Entity

	// accountId of existing SoftLayer Customer
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// Service provider address1
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// Service provider address2
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// Credit card account number
	CardAccountNumber *string `json:"cardAccountNumber,omitempty" xmlrpc:"cardAccountNumber,omitempty"`

	// Credit card expiration month
	CardExpirationMonth *string `json:"cardExpirationMonth,omitempty" xmlrpc:"cardExpirationMonth,omitempty"`

	// Credit card expiration year
	CardExpirationYear *string `json:"cardExpirationYear,omitempty" xmlrpc:"cardExpirationYear,omitempty"`

	// Type of credit card being used
	CardType *string `json:"cardType,omitempty" xmlrpc:"cardType,omitempty"`

	// Credit card verification number
	CardVerificationNumber *string `json:"cardVerificationNumber,omitempty" xmlrpc:"cardVerificationNumber,omitempty"`

	// Service provider city
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// Service provider company name
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// Catalyst company types.
	CompanyType *Catalyst_Company_Type `json:"companyType,omitempty" xmlrpc:"companyType,omitempty"`

	// Id of the company type which best describes applicant's company
	CompanyTypeId *int `json:"companyTypeId,omitempty" xmlrpc:"companyTypeId,omitempty"`

	// Service provider company url
	CompanyUrl *string `json:"companyUrl,omitempty" xmlrpc:"companyUrl,omitempty"`

	// Service provider contact's email
	ContactEmail *string `json:"contactEmail,omitempty" xmlrpc:"contactEmail,omitempty"`

	// Service provider contact's first name
	ContactFirstName *string `json:"contactFirstName,omitempty" xmlrpc:"contactFirstName,omitempty"`

	// Service provider contact's last name
	ContactLastName *string `json:"contactLastName,omitempty" xmlrpc:"contactLastName,omitempty"`

	// Service provider contact's Phone
	ContactPhone *string `json:"contactPhone,omitempty" xmlrpc:"contactPhone,omitempty"`

	// Service provider country
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// Customer Prospect id
	CustomerProspectId *int `json:"customerProspectId,omitempty" xmlrpc:"customerProspectId,omitempty"`

	// Id of the device fingerprint
	DeviceFingerprintId *string `json:"deviceFingerprintId,omitempty" xmlrpc:"deviceFingerprintId,omitempty"`

	// Service provider email
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// Indicates if customer has an existing SoftLayer account
	ExistingCustomerFlag *bool `json:"existingCustomerFlag,omitempty" xmlrpc:"existingCustomerFlag,omitempty"`

	// Service provider first name
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// IBM partner world id
	IbmPartnerWorldId *string `json:"ibmPartnerWorldId,omitempty" xmlrpc:"ibmPartnerWorldId,omitempty"`

	// Indicates if the customer is IBM partner world member
	IbmPartnerWorldMemberFlag *bool `json:"ibmPartnerWorldMemberFlag,omitempty" xmlrpc:"ibmPartnerWorldMemberFlag,omitempty"`

	// Service provider last name
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// Flag indicating whether or not applicant acknowledged MSA
	MasterAgreementCompleteFlag *bool `json:"masterAgreementCompleteFlag,omitempty" xmlrpc:"masterAgreementCompleteFlag,omitempty"`

	// Service provider office phone
	OfficePhone *string `json:"officePhone,omitempty" xmlrpc:"officePhone,omitempty"`

	// Service provider postalCode
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// Flag indicating whether or not applicant acknowledged service provider addendum
	ServiceProviderAddendumFlag *bool `json:"serviceProviderAddendumFlag,omitempty" xmlrpc:"serviceProviderAddendumFlag,omitempty"`

	// Service provider state
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// Survey responses
	SurveyResponses []Survey_Response `json:"surveyResponses,omitempty" xmlrpc:"surveyResponses,omitempty"`

	// Applicant's VAT id, if one exists
	VatId *string `json:"vatId,omitempty" xmlrpc:"vatId,omitempty"`
}

// no documentation yet
type User_Customer_Prospect_Type struct {
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

// The SoftLayer_User_Customer_Security_Answer type contains user's answers to security questions.
type User_Customer_Security_Answer struct {
	Entity

	// A user's answer.
	Answer *string `json:"answer,omitempty" xmlrpc:"answer,omitempty"`

	// A user's answer identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The question the security answer is associated with.
	Question *User_Security_Question `json:"question,omitempty" xmlrpc:"question,omitempty"`

	// A user's question identifying number.
	QuestionId *int `json:"questionId,omitempty" xmlrpc:"questionId,omitempty"`

	// The user who the security answer belongs to.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// A user's identifying number.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// Each SoftLayer portal account is assigned a status code that determines how it's treated in the customer portal. This status is reflected in the SoftLayer_User_Customer_Status data type. Status differs from user permissions in that user status applies globally to the portal while user permissions are applied to specific portal functions.
type User_Customer_Status struct {
	Entity

	// A user's status identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A user's status keyname
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// A user's status. This can be either "Active" for user accounts with portal access, "Inactive" for users disabled by another portal user, "Disabled" for accounts turned off by SoftLayer, or "VPN Only" for user accounts with no access to the customer portal but VPN access to the private network.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// A SoftLayer_User_Employee models a single SoftLayer employee for the purposes of ticket updates created by SoftLayer employees. SoftLayer portal and API users cannot see individual employee names in ticket responses.  SoftLayer employees can be assigned to customer accounts as a personal support representative.  Employee names and email will be available if an employee is assigned to the account.
type User_Employee struct {
	User_Interface

	// A count of
	ActionCount *uint `json:"actionCount,omitempty" xmlrpc:"actionCount,omitempty"`

	// no documentation yet
	Actions []User_Permission_Action `json:"actions,omitempty" xmlrpc:"actions,omitempty"`

	// no documentation yet
	ChatTranscript []Ticket_Chat `json:"chatTranscript,omitempty" xmlrpc:"chatTranscript,omitempty"`

	// A count of
	ChatTranscriptCount *uint `json:"chatTranscriptCount,omitempty" xmlrpc:"chatTranscriptCount,omitempty"`

	// no documentation yet
	DisplayName *string `json:"displayName,omitempty" xmlrpc:"displayName,omitempty"`

	// A SoftLayer employee's email address. Email addresses are only visible to [[SoftLayer_Account|SoftLayer Accounts]] that are assigned to an employee
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// The department that a SoftLayer employee belongs to.
	EmployeeDepartment *User_Employee_Department `json:"employeeDepartment,omitempty" xmlrpc:"employeeDepartment,omitempty"`

	// A SoftLayer employee's [[SoftLayer_User_Employee_Department|department]] id.
	EmployeeDepartmentId *int `json:"employeeDepartmentId,omitempty" xmlrpc:"employeeDepartmentId,omitempty"`

	// A SoftLayer employee's first name. First names are only visible to [[SoftLayer_Account|SoftLayer Accounts]] that are assigned to an employee
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// A SoftLayer employee's last name. Last names are only visible to [[SoftLayer_Account|SoftLayer Accounts]] that are assigned to an employee
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// A count of
	LayoutProfileCount *uint `json:"layoutProfileCount,omitempty" xmlrpc:"layoutProfileCount,omitempty"`

	// no documentation yet
	LayoutProfiles []Layout_Profile `json:"layoutProfiles,omitempty" xmlrpc:"layoutProfiles,omitempty"`

	// no documentation yet
	MetricTrackingObject *Metric_Tracking_Object `json:"metricTrackingObject,omitempty" xmlrpc:"metricTrackingObject,omitempty"`

	// no documentation yet
	OfficePhone *string `json:"officePhone,omitempty" xmlrpc:"officePhone,omitempty"`

	// A count of
	RoleCount *uint `json:"roleCount,omitempty" xmlrpc:"roleCount,omitempty"`

	// no documentation yet
	Roles []User_Permission_Role `json:"roles,omitempty" xmlrpc:"roles,omitempty"`

	// no documentation yet
	TicketActivities []Ticket_Activity `json:"ticketActivities,omitempty" xmlrpc:"ticketActivities,omitempty"`

	// A count of
	TicketActivityCount *uint `json:"ticketActivityCount,omitempty" xmlrpc:"ticketActivityCount,omitempty"`

	// A count of
	TicketAttachmentReferenceCount *uint `json:"ticketAttachmentReferenceCount,omitempty" xmlrpc:"ticketAttachmentReferenceCount,omitempty"`

	// no documentation yet
	TicketAttachmentReferences []Ticket_Attachment `json:"ticketAttachmentReferences,omitempty" xmlrpc:"ticketAttachmentReferences,omitempty"`

	// A representation of a SoftLayer employee's username. In all cases this should simply state "Employee".
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// SoftLayer_User_Employee_Department models a department within SoftLayer's internal employee hierarchy. Common departments include Support, Sales, Accounting, Development, Systems, and Networking.
type User_Employee_Department struct {
	Entity

	// The name of one of SoftLayer's employee departments.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_User_External_Binding data type contains general information for a single external binding.  This includes the 3rd party vendor, type of binding, and a unique identifier and password that is used to authenticate against the 3rd party service.
type User_External_Binding struct {
	Entity

	// The flag that determines whether the external binding is active will be used for authentication or not.
	Active *bool `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// A count of attributes of an external authentication binding.
	AttributeCount *uint `json:"attributeCount,omitempty" xmlrpc:"attributeCount,omitempty"`

	// Attributes of an external authentication binding.
	Attributes []User_External_Binding_Attribute `json:"attributes,omitempty" xmlrpc:"attributes,omitempty"`

	// Information regarding the billing item for external authentication.
	BillingItem *Billing_Item `json:"billingItem,omitempty" xmlrpc:"billingItem,omitempty"`

	// The date that the external authentication binding was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The identifier used to identify this binding to an external authentication source.
	ExternalId *string `json:"externalId,omitempty" xmlrpc:"externalId,omitempty"`

	// An external authentication binding's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// An optional note for identifying the external binding.
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	// The password used to authenticate the external id at an external authentication source.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// The type of external authentication binding.
	Type *User_External_Binding_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The [[SoftLayer_User_External_Binding_Type|type]] identifier of an external authentication binding.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// An external authentication binding's associated [[SoftLayer_User_Customer|user account]] id.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// The vendor of an external authentication binding.
	Vendor *User_External_Binding_Vendor `json:"vendor,omitempty" xmlrpc:"vendor,omitempty"`

	// The [[SoftLayer_User_External_Binding_Vendor|vendor]] identifier of an external authentication binding.
	VendorId *int `json:"vendorId,omitempty" xmlrpc:"vendorId,omitempty"`
}

// The SoftLayer_User_External_Binding_Attribute data type contains the value for a single attribute associated with an external binding. External binding attributes contain additional information about an external binding.  An attribute can be generic or specific to a 3rd party vendor.  For example these attributes relate to Verisign:
// *Credential Type
// *Credential State
// *Credential Expiration Date
// *Credential Last Update Date
type User_External_Binding_Attribute struct {
	Entity

	// The external authentication binding an attribute belongs to.
	ExternalBinding *User_External_Binding `json:"externalBinding,omitempty" xmlrpc:"externalBinding,omitempty"`

	// The value of an external binding attribute.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The SoftLayer_User_External_Binding_Type data type contains information relating to a type of external authentication binding.  It contains a user friendly name as well as a unique key name.
type User_External_Binding_Type struct {
	Entity

	// The unique name used to identify a type of external authentication binding.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The user friendly name of a type of external authentication binding.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_User_External_Binding_Vendor data type contains information for a single external binding vendor.  This information includes a user friendly vendor name, a unique version of the vendor name, and a unique internal identifier that can be used when creating a new external binding.
type User_External_Binding_Vendor struct {
	Entity

	// The unique identifier for an external binding vendor.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A unique version of the name property.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The user friendly name of an external binding vendor.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// A SoftLayer_User_Interface represents a generic user instance within the SoftLayer API. The SoftLayer API uses SoftLayer_User_Interfaces in cases where a user object could be one of many types of users. Currently the [[SoftLayer_User_Customer]] and [[SoftLayer_User_Employee]] classes are abstracted by this type.
type User_Interface struct {
	Entity
}

// no documentation yet
type User_Permission_Action struct {
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
type User_Permission_Group struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A permission groups associated [[SoftLayer_Account|customer account]] id.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of
	ActionCount *uint `json:"actionCount,omitempty" xmlrpc:"actionCount,omitempty"`

	// no documentation yet
	Actions []User_Permission_Action `json:"actions,omitempty" xmlrpc:"actions,omitempty"`

	// The date the permission group record was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The description of the permission group.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The date the temporary group will be destroyed.
	ExpirationDate *Time `json:"expirationDate,omitempty" xmlrpc:"expirationDate,omitempty"`

	// A permission groups internal identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date the permission group record was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of the permission group.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of
	RoleCount *uint `json:"roleCount,omitempty" xmlrpc:"roleCount,omitempty"`

	// no documentation yet
	Roles []User_Permission_Role `json:"roles,omitempty" xmlrpc:"roles,omitempty"`

	// The type of the permission group.
	Type *User_Permission_Group_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The type of permission group.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`
}

// no documentation yet
type User_Permission_Group_Type struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A count of
	GroupCount *uint `json:"groupCount,omitempty" xmlrpc:"groupCount,omitempty"`

	// no documentation yet
	Groups []User_Permission_Group `json:"groups,omitempty" xmlrpc:"groups,omitempty"`

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
type User_Permission_Role struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A permission roles associated [[SoftLayer_Account|customer account]] id.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of
	ActionCount *uint `json:"actionCount,omitempty" xmlrpc:"actionCount,omitempty"`

	// no documentation yet
	Actions []User_Permission_Action `json:"actions,omitempty" xmlrpc:"actions,omitempty"`

	// The date the permission role record was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The description of the permission role.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// A count of
	GroupCount *uint `json:"groupCount,omitempty" xmlrpc:"groupCount,omitempty"`

	// no documentation yet
	Groups []User_Permission_Group `json:"groups,omitempty" xmlrpc:"groups,omitempty"`

	// A permission roles internal identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date the permission role record was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of the permission role.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A flag showing if new users should be automatically added to this role.
	NewUserDefaultFlag *int `json:"newUserDefaultFlag,omitempty" xmlrpc:"newUserDefaultFlag,omitempty"`

	// A flag showing if the permission role was created by our internal system for a single user. If this flag is set only a single user can be assigned to this permission role and it can not be deleted.
	SystemFlag *int `json:"systemFlag,omitempty" xmlrpc:"systemFlag,omitempty"`

	// A count of
	UserCount *uint `json:"userCount,omitempty" xmlrpc:"userCount,omitempty"`

	// no documentation yet
	Users []User_Customer `json:"users,omitempty" xmlrpc:"users,omitempty"`
}

// The SoftLayer_User_Preference data type contains a single user preference to a specific preference type.
type User_Preference struct {
	Entity

	// Description of the user preference
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Type of user preference
	Type *User_Preference_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The users current preference value
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// The SoftLayer_User_Preference_Type data type contains a single preference type including the accepted values.
type User_Preference_Type struct {
	Entity

	// A description of the preference type
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of the preference type
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// An example of accepted preference values
	ValueExample *string `json:"valueExample,omitempty" xmlrpc:"valueExample,omitempty"`
}

// The SoftLayer_User_Security_Question data type contains questions.
type User_Security_Question struct {
	Entity

	// A security question's display order.
	DisplayOrder *int `json:"displayOrder,omitempty" xmlrpc:"displayOrder,omitempty"`

	// A security question's internal identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A security question's question.
	Question *string `json:"question,omitempty" xmlrpc:"question,omitempty"`

	// A security question's viewable flag.
	Viewable *int `json:"viewable,omitempty" xmlrpc:"viewable,omitempty"`
}
