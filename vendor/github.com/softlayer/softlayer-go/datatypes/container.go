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

// SoftLayer_Container_Account_Discount_Program models a single outbound object for a graph of given data sets.
type Container_Account_Discount_Program struct {
	Entity

	// The credit allowance that has already been applied during the current billing cycle. If the lifetime limit has been or soon will be reached, this amount may included credit applied in previous billing cycles.
	AppliedCredit *Float64 `json:"appliedCredit,omitempty" xmlrpc:"appliedCredit,omitempty"`

	// Flag to signify whether the account is a participant in the discount program.
	IsParticipant *bool `json:"isParticipant,omitempty" xmlrpc:"isParticipant,omitempty"`

	// Credit allowance applied over the course of the entire program enrollment. For enrollments without a lifetime restriction, this property will not be populated as credit will be tracked on a purely monthly basis.
	LifetimeAppliedCredit *Float64 `json:"lifetimeAppliedCredit,omitempty" xmlrpc:"lifetimeAppliedCredit,omitempty"`

	// Credit allowance available over the course of the entire program enrollment. If null, enrollment credit is applied on a strictly monthly basis and there is no lifetime maximum. Enrollments with non-null lifetime credit will receive the lesser of the remaining monthly credit or the remaining lifetime credit.
	LifetimeCredit *Float64 `json:"lifetimeCredit,omitempty" xmlrpc:"lifetimeCredit,omitempty"`

	// Remaining credit allowance available over the remaining duration of the program enrollment. If null, enrollment credit is applied on a strictly monthly basis and there is no lifetime maximum. Enrollments with non-null remaining lifetime credit will receive the lesser of the remaining monthly credit or the remaining lifetime credit.
	LifetimeRemainingCredit *Float64 `json:"lifetimeRemainingCredit,omitempty" xmlrpc:"lifetimeRemainingCredit,omitempty"`

	// Maximum number of orders the enrolled account is allowed to have open at one time. If null, then the Flexible Credit Program does not impose an order limit.
	MaximumActiveOrders *Float64 `json:"maximumActiveOrders,omitempty" xmlrpc:"maximumActiveOrders,omitempty"`

	// The monthly credit allowance that is available at the beginning of the billing cycle.
	MonthlyCredit *Float64 `json:"monthlyCredit,omitempty" xmlrpc:"monthlyCredit,omitempty"`

	// DEPRECATED: Taxes are calculated in real time and discount amounts are shown pre-tax in all cases. Tax values in the SoftLayer_Container_Account_Discount_Program container are now populated with the related pre-tax values.
	PostTaxRemainingCredit *Float64 `json:"postTaxRemainingCredit,omitempty" xmlrpc:"postTaxRemainingCredit,omitempty"`

	// The date at which the program expires in MM/DD/YYYY format.
	ProgramEndDate *Time `json:"programEndDate,omitempty" xmlrpc:"programEndDate,omitempty"`

	// Name of the Flexible Credit Program the account is enrolled in.
	ProgramName *string `json:"programName,omitempty" xmlrpc:"programName,omitempty"`

	// The credit allowance that is available during the current billing cycle. If the lifetime limit has been or soon will be reached, this amount may be reduced by credit applied in previous billing cycles.
	RemainingCredit *Float64 `json:"remainingCredit,omitempty" xmlrpc:"remainingCredit,omitempty"`

	// DEPRECATED: Taxes are calculated in real time and discount amounts are shown pre-tax in all cases. Tax values in the SoftLayer_Container_Account_Discount_Program container are now populated with the related pre-tax values.
	RemainingCreditTax *Float64 `json:"remainingCreditTax,omitempty" xmlrpc:"remainingCreditTax,omitempty"`
}

// SoftLayer_Container_Account_Graph_Outputs <<< EOT
type Container_Account_Graph_Outputs struct {
	Entity

	// The count of closed tickets included in this graph.
	ClosedTickets *string `json:"closedTickets,omitempty" xmlrpc:"closedTickets,omitempty"`

	// The count of completed backups included in this graph.
	CompletedBackupCount *string `json:"completedBackupCount,omitempty" xmlrpc:"completedBackupCount,omitempty"`

	// The count of conflicted backups included in this graph.
	ConflictBackupCount *string `json:"conflictBackupCount,omitempty" xmlrpc:"conflictBackupCount,omitempty"`

	// The maximum date included in this graph.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// The count of failed backups included in this graph.
	FailedBackupCount *string `json:"failedBackupCount,omitempty" xmlrpc:"failedBackupCount,omitempty"`

	// Error message encountered during graphing
	GraphError *string `json:"graphError,omitempty" xmlrpc:"graphError,omitempty"`

	// The raw PNG binary data to be displayed once the graph is drawn.
	GraphImage *[]byte `json:"graphImage,omitempty" xmlrpc:"graphImage,omitempty"`

	// The average of hardware uptime included in this graph.
	HardwareUptime *string `json:"hardwareUptime,omitempty" xmlrpc:"hardwareUptime,omitempty"`

	// The inbound bandwidth usage shown in this graph.
	InboundUsage *string `json:"inboundUsage,omitempty" xmlrpc:"inboundUsage,omitempty"`

	// The count of open tickets included in this graph.
	OpenTickets *string `json:"openTickets,omitempty" xmlrpc:"openTickets,omitempty"`

	// The outbound bandwidth usage shown in this graph.
	OutboundUsage *string `json:"outboundUsage,omitempty" xmlrpc:"outboundUsage,omitempty"`

	// The count of tickets included in this graph.
	PendingCustomerResponseTicketCount *string `json:"pendingCustomerResponseTicketCount,omitempty" xmlrpc:"pendingCustomerResponseTicketCount,omitempty"`

	// The minimum date included in this graph.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// The average of url uptime included in this graph.
	UrlUptime *string `json:"urlUptime,omitempty" xmlrpc:"urlUptime,omitempty"`

	// The count of tickets included in this graph.
	WaitingEmployeeResponseTicketCount *string `json:"waitingEmployeeResponseTicketCount,omitempty" xmlrpc:"waitingEmployeeResponseTicketCount,omitempty"`
}

// Historical Summary Container for account resource details
type Container_Account_Historical_Summary struct {
	Entity

	// Array of server uptime detail containers
	Details []Container_Account_Historical_Summary_Detail `json:"details,omitempty" xmlrpc:"details,omitempty"`

	// The maximum date included in the summary.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// The minimum date included in the summary.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}

// Historical Summary Details Container for a resource's data
type Container_Account_Historical_Summary_Detail struct {
	Entity

	// The maximum date included in the detail.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// The minimum date included in the detail.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}

// Historical Summary Details Container for a host resource uptime
type Container_Account_Historical_Summary_Detail_Uptime struct {
	Container_Account_Historical_Summary_Detail

	// The hardware for uptime details.
	CloudComputingInstance *Virtual_Guest `json:"cloudComputingInstance,omitempty" xmlrpc:"cloudComputingInstance,omitempty"`

	// The configuration value for the detail's resource.
	ConfigurationValue *Monitoring_Agent_Configuration_Value `json:"configurationValue,omitempty" xmlrpc:"configurationValue,omitempty"`

	// The data associated with a host uptime details.
	Data []Metric_Tracking_Object_Data `json:"data,omitempty" xmlrpc:"data,omitempty"`

	// The hardware for uptime details.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`
}

// Historical Summary Container for account host's resource uptime details
type Container_Account_Historical_Summary_Uptime struct {
	Container_Account_Historical_Summary
}

// no documentation yet
type Container_Account_Payment_Method_CreditCard struct {
	Entity

	// no documentation yet
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// no documentation yet
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// no documentation yet
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// no documentation yet
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// no documentation yet
	CurrencyShortName *string `json:"currencyShortName,omitempty" xmlrpc:"currencyShortName,omitempty"`

	// no documentation yet
	CybersourceAssignedCardType *string `json:"cybersourceAssignedCardType,omitempty" xmlrpc:"cybersourceAssignedCardType,omitempty"`

	// no documentation yet
	ExpireMonth *string `json:"expireMonth,omitempty" xmlrpc:"expireMonth,omitempty"`

	// no documentation yet
	ExpireYear *string `json:"expireYear,omitempty" xmlrpc:"expireYear,omitempty"`

	// no documentation yet
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// no documentation yet
	LastFourDigits *string `json:"lastFourDigits,omitempty" xmlrpc:"lastFourDigits,omitempty"`

	// no documentation yet
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// no documentation yet
	Nickname *string `json:"nickname,omitempty" xmlrpc:"nickname,omitempty"`

	// no documentation yet
	PaymentMethodRoleName *string `json:"paymentMethodRoleName,omitempty" xmlrpc:"paymentMethodRoleName,omitempty"`

	// no documentation yet
	PaymentTypeId *string `json:"paymentTypeId,omitempty" xmlrpc:"paymentTypeId,omitempty"`

	// no documentation yet
	PaymentTypeName *string `json:"paymentTypeName,omitempty" xmlrpc:"paymentTypeName,omitempty"`

	// no documentation yet
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// no documentation yet
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`
}

// The SoftLayer_Container_Authentication_Request_Common data type contains common information for requests to the getPortalLogin API. This is an abstract class that serves as a base that more specialized classes will derive from. For example, a request class specific to SoftLayer Native IMS Login (username and password).
type Container_Authentication_Request_Common struct {
	Container_Authentication_Request_Contract

	// The answer to your security question.
	SecurityQuestionAnswer *string `json:"securityQuestionAnswer,omitempty" xmlrpc:"securityQuestionAnswer,omitempty"`

	// A security question you wish to answer when authenticating to the SoftLayer customer portal. This parameter isn't required if no security questions are set on your portal account or if your account is configured to not require answering a security account upon login.
	SecurityQuestionId *int `json:"securityQuestionId,omitempty" xmlrpc:"securityQuestionId,omitempty"`
}

// The SoftLayer_Container_Authentication_Request_Contract provides a common set of operations for implementing classes.
type Container_Authentication_Request_Contract struct {
	Entity
}

// The SoftLayer_Container_Authentication_Request_Native data type contains information for requests to the getPortalLogin API. This class is specific to the SoftLayer Native login (username/password). The request information will be verified to ensure it is valid, and then there will be an attempt to obtain a portal login token in authenticating the user with the provided information.
type Container_Authentication_Request_Native struct {
	Container_Authentication_Request_Common

	// Your SoftLayer customer portal user's portal password.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// The username you wish to authenticate to the SoftLayer customer portal with.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// The SoftLayer_Container_Authentication_Request_Native_External data type contains information for requests to the getPortalLogin API. This class serves as a base class for more specialized external authentication classes to the SoftLayer Native login (username/password).
type Container_Authentication_Request_Native_External struct {
	Container_Authentication_Request_Native
}

// The SoftLayer_Container_Authentication_Request_Native_External_Totp data type contains information for requests to the getPortalLogin API. This class provides information to allow the user to submit a request to the native SoftLayer (username/password) login service for a portal login token, as well as submitting a request to the TOTP 2 factor authentication service.
type Container_Authentication_Request_Native_External_Totp struct {
	Container_Authentication_Request_Native_External

	// no documentation yet
	SecondSecurityCode *string `json:"secondSecurityCode,omitempty" xmlrpc:"secondSecurityCode,omitempty"`

	// no documentation yet
	SecurityCode *string `json:"securityCode,omitempty" xmlrpc:"securityCode,omitempty"`

	// no documentation yet
	Vendor *string `json:"vendor,omitempty" xmlrpc:"vendor,omitempty"`
}

// The SoftLayer_Container_Authentication_Request_Native_External_Verisign data type contains information for requests to the getPortalLogin API. This class provides information to allow the user to submit a request to the native SoftLayer (username/password) login service for a portal login token, as well as submitting a request to the Verisign 2 factor authentication service.
type Container_Authentication_Request_Native_External_Verisign struct {
	Container_Authentication_Request_Native_External

	// no documentation yet
	SecondSecurityCode *string `json:"secondSecurityCode,omitempty" xmlrpc:"secondSecurityCode,omitempty"`

	// no documentation yet
	SecurityCode *string `json:"securityCode,omitempty" xmlrpc:"securityCode,omitempty"`

	// no documentation yet
	Vendor *string `json:"vendor,omitempty" xmlrpc:"vendor,omitempty"`
}

// The SoftLayer_Container_Authentication_Request_OpenIdConnect data type contains information for requests to the getPortalLogin API. This class is specific to the SoftLayer Cloud Token login. The request information will be verified to ensure it is valid, and then there will be an attempt to obtain a portal login token in authenticating the user with the provided information.
type Container_Authentication_Request_OpenIdConnect struct {
	Container_Authentication_Request_Common

	// no documentation yet
	OpenIdConnectAccessToken *string `json:"openIdConnectAccessToken,omitempty" xmlrpc:"openIdConnectAccessToken,omitempty"`

	// no documentation yet
	OpenIdConnectAccountId *int `json:"openIdConnectAccountId,omitempty" xmlrpc:"openIdConnectAccountId,omitempty"`

	// no documentation yet
	OpenIdConnectProvider *string `json:"openIdConnectProvider,omitempty" xmlrpc:"openIdConnectProvider,omitempty"`
}

// The SoftLayer_Container_Authentication_Request_OpenIdConnect_External data type contains information for requests to the getPortalLogin API. This class serves as a base class for more specialized external authentication classes to the SoftLayer OpenIdConnect login service.
type Container_Authentication_Request_OpenIdConnect_External struct {
	Container_Authentication_Request_OpenIdConnect
}

// The SoftLayer_Container_Authentication_Request_OpenIdConnect_External_Totp data type contains information for requests to the getPortalLogin API. This class provides information to allow the user to submit a request to the SoftLayer OpenIdConnect (token) login service for a portal login token, as well as submitting a request to the TOTP 2 factor authentication service.
type Container_Authentication_Request_OpenIdConnect_External_Totp struct {
	Container_Authentication_Request_OpenIdConnect_External

	// no documentation yet
	SecondSecurityCode *string `json:"secondSecurityCode,omitempty" xmlrpc:"secondSecurityCode,omitempty"`

	// no documentation yet
	SecurityCode *string `json:"securityCode,omitempty" xmlrpc:"securityCode,omitempty"`

	// no documentation yet
	Vendor *string `json:"vendor,omitempty" xmlrpc:"vendor,omitempty"`
}

// The SoftLayer_Container_Authentication_Request_OpenIdConnect_External_Verisign data type contains information for requests to the getPortalLogin API. This class provides information to allow the user to submit a request to the SoftLayer OpenIdConnect (token) login service for a portal login token, as well as submitting a request to the Verisign 2 factor authentication service.
type Container_Authentication_Request_OpenIdConnect_External_Verisign struct {
	Container_Authentication_Request_OpenIdConnect_External

	// no documentation yet
	SecondSecurityCode *string `json:"secondSecurityCode,omitempty" xmlrpc:"secondSecurityCode,omitempty"`

	// no documentation yet
	SecurityCode *int `json:"securityCode,omitempty" xmlrpc:"securityCode,omitempty"`

	// no documentation yet
	Vendor *string `json:"vendor,omitempty" xmlrpc:"vendor,omitempty"`
}

// The SoftLayer_Container_Authentication_Response_2FactorAuthenticationNeeded data type contains information for specific responses from the getPortalLogin API. This class is indicative of a request that is missing the appropriate 2FA information.
type Container_Authentication_Response_2FactorAuthenticationNeeded struct {
	Container_Authentication_Response_Common

	// no documentation yet
	AdditionalData *Container_Authentication_Response_Common `json:"additionalData,omitempty" xmlrpc:"additionalData,omitempty"`

	// no documentation yet
	StatusKeyName *string `json:"statusKeyName,omitempty" xmlrpc:"statusKeyName,omitempty"`
}

// The SoftLayer_Container_Authentication_Response_Account data type contains account information for responses from the getPortalLogin API.
type Container_Authentication_Response_Account struct {
	Entity

	// no documentation yet
	AccountCompanyName *string `json:"accountCompanyName,omitempty" xmlrpc:"accountCompanyName,omitempty"`

	// no documentation yet
	AccountCountry *string `json:"accountCountry,omitempty" xmlrpc:"accountCountry,omitempty"`

	// no documentation yet
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	AccountStatusName *string `json:"accountStatusName,omitempty" xmlrpc:"accountStatusName,omitempty"`

	// no documentation yet
	BluemixAccountId *string `json:"bluemixAccountId,omitempty" xmlrpc:"bluemixAccountId,omitempty"`

	// no documentation yet
	DefaultAccount *bool `json:"defaultAccount,omitempty" xmlrpc:"defaultAccount,omitempty"`

	// no documentation yet
	IsMasterUserFlag *bool `json:"isMasterUserFlag,omitempty" xmlrpc:"isMasterUserFlag,omitempty"`

	// no documentation yet
	PhoneFactorExternalAuthenticationRequired *bool `json:"phoneFactorExternalAuthenticationRequired,omitempty" xmlrpc:"phoneFactorExternalAuthenticationRequired,omitempty"`

	// no documentation yet
	SecurityQuestionRequired *bool `json:"securityQuestionRequired,omitempty" xmlrpc:"securityQuestionRequired,omitempty"`

	// no documentation yet
	TotpExternalAuthenticationRequired *bool `json:"totpExternalAuthenticationRequired,omitempty" xmlrpc:"totpExternalAuthenticationRequired,omitempty"`

	// no documentation yet
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// no documentation yet
	VerisignExternalAuthenticationRequired *bool `json:"verisignExternalAuthenticationRequired,omitempty" xmlrpc:"verisignExternalAuthenticationRequired,omitempty"`
}

// The SoftLayer_Container_Authentication_Response_AccountIdMissing data type contains information for specific responses from the getPortalLogin API. This class is indicative of a request that is missing the account id.
type Container_Authentication_Response_AccountIdMissing struct {
	Container_Authentication_Response_Common

	// no documentation yet
	StatusKeyName *string `json:"statusKeyName,omitempty" xmlrpc:"statusKeyName,omitempty"`
}

// The SoftLayer_Container_Authentication_Response_Common data type contains common information for responses from the getPortalLogin API. This is an abstract class that serves as a base that more specialized classes will derive from. For example, a response class that is specific to a successful response from the getPortalLogin API.
type Container_Authentication_Response_Common struct {
	Entity

	// The list of linked accounts for the authenticated SoftLayer customer portal user.
	Accounts []Container_Authentication_Response_Account `json:"accounts,omitempty" xmlrpc:"accounts,omitempty"`
}

// The SoftLayer_Container_Authentication_Response_LOGIN_FAILED data type contains information for specific responses from the getPortalLogin API. This class is indicative of a request where there was an inability to login based on the information that was provided.
type Container_Authentication_Response_LoginFailed struct {
	Container_Authentication_Response_Common

	// no documentation yet
	ErrorMessage *string `json:"errorMessage,omitempty" xmlrpc:"errorMessage,omitempty"`

	// no documentation yet
	StatusKeyName *string `json:"statusKeyName,omitempty" xmlrpc:"statusKeyName,omitempty"`
}

// The SoftLayer_Container_Authentication_Response_SUCCESS data type contains information for specific responses from the getPortalLogin API. This class is indicative of a request that was successful in obtaining a portal login token from the getPortalLogin API.
type Container_Authentication_Response_Success struct {
	Container_Authentication_Response_Common

	// no documentation yet
	StatusKeyName *string `json:"statusKeyName,omitempty" xmlrpc:"statusKeyName,omitempty"`

	// The token for interacting with the SoftLayer customer portal.
	Token *Container_User_Authentication_Token `json:"token,omitempty" xmlrpc:"token,omitempty"`
}

// The SoftLayer_Container_Auxiliary_Network_Status_Reading data type contains information relating to an object being monitored from outside the SoftLayer network.  It is primarily used to check the status of our edge routers from multiple locations around the world.
type Container_Auxiliary_Network_Status_Reading struct {
	Entity

	// Average packet round-trip time.
	AveragePing *Float64 `json:"averagePing,omitempty" xmlrpc:"averagePing,omitempty"`

	// Number of failures since the target was last detected to be working properly.
	Fails *int `json:"fails,omitempty" xmlrpc:"fails,omitempty"`

	// Monitoring frequency in minutes.
	Frequency *int `json:"frequency,omitempty" xmlrpc:"frequency,omitempty"`

	// The target babel.
	Label *string `json:"label,omitempty" xmlrpc:"label,omitempty"`

	// Last check date and time.
	LastCheckDate *Time `json:"lastCheckDate,omitempty" xmlrpc:"lastCheckDate,omitempty"`

	// Date and time of the last problem detected.
	LastDownDate *Time `json:"lastDownDate,omitempty" xmlrpc:"lastDownDate,omitempty"`

	// The total response time in seconds calculated during the last check.
	Latency *Float64 `json:"latency,omitempty" xmlrpc:"latency,omitempty"`

	// The monitoring location name.
	Location *string `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// Maximum packet round-trip time.
	MaximumPing *Float64 `json:"maximumPing,omitempty" xmlrpc:"maximumPing,omitempty"`

	// Minimum packet round-trip time.
	MinimumPing *Float64 `json:"minimumPing,omitempty" xmlrpc:"minimumPing,omitempty"`

	// Packet loss percentage.
	PingLoss *Float64 `json:"pingLoss,omitempty" xmlrpc:"pingLoss,omitempty"`

	// The date monitoring first began
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// Status Code - one of UP, Down, Test pending.
	StatusCode *string `json:"statusCode,omitempty" xmlrpc:"statusCode,omitempty"`

	// The status message from the last effective check.
	StatusMessage *string `json:"statusMessage,omitempty" xmlrpc:"statusMessage,omitempty"`

	// The target object.
	Target *string `json:"target,omitempty" xmlrpc:"target,omitempty"`

	// A letter indicating the target type.
	TargetType *string `json:"targetType,omitempty" xmlrpc:"targetType,omitempty"`
}

// SoftLayer_Container_Bandwidth_GraphInputs models a single inbound object for a given bandwidth graph.
type Container_Bandwidth_GraphInputs struct {
	Entity

	// This is a unix timestamp that represents the stop date/time for a graph.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// The front-end or back-end network uplink interface associated with this server.
	NetworkInterfaceId *int `json:"networkInterfaceId,omitempty" xmlrpc:"networkInterfaceId,omitempty"`

	// *
	Pod *int `json:"pod,omitempty" xmlrpc:"pod,omitempty"`

	// This is a human readable name for the server or rack being graphed.
	ServerName *string `json:"serverName,omitempty" xmlrpc:"serverName,omitempty"`

	// This is a unix timestamp that represents the begin date/time for a graph.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}

// SoftLayer_Container_Bandwidth_GraphOutputs models a single outbound object for a given bandwidth graph.
type Container_Bandwidth_GraphOutputs struct {
	Entity

	// The raw PNG binary data to be displayed once the graph is drawn.
	GraphImage *[]byte `json:"graphImage,omitempty" xmlrpc:"graphImage,omitempty"`

	// The title that ended up being displayed as part of the graph image.
	GraphTitle *string `json:"graphTitle,omitempty" xmlrpc:"graphTitle,omitempty"`

	// The maximum date included in this graph.
	MaxEndDate *Time `json:"maxEndDate,omitempty" xmlrpc:"maxEndDate,omitempty"`

	// The minimum date included in this graph.
	MinStartDate *Time `json:"minStartDate,omitempty" xmlrpc:"minStartDate,omitempty"`
}

// SoftLayer_Container_Bandwidth_GraphOutputs models an individual bandwidth graph image and certain details about that graph image.
type Container_Bandwidth_GraphOutputsExtended struct {
	Entity

	// The raw PNG binary data of a bandwidth graph image.
	GraphImage *[]byte `json:"graphImage,omitempty" xmlrpc:"graphImage,omitempty"`

	// A bandwidth graph's title.
	GraphTitle *string `json:"graphTitle,omitempty" xmlrpc:"graphTitle,omitempty"`

	// The amount of inbound traffic reported on a bandwidth graph image.
	InBoundTotalBytes *uint `json:"inBoundTotalBytes,omitempty" xmlrpc:"inBoundTotalBytes,omitempty"`

	// The ending date of the data represented in a bandwidth graph.
	MaxEndDate *Time `json:"maxEndDate,omitempty" xmlrpc:"maxEndDate,omitempty"`

	// The beginning date of the data represented in a bandwidth graph.
	MinStartDate *Time `json:"minStartDate,omitempty" xmlrpc:"minStartDate,omitempty"`

	// The amount of outbound traffic reported on a bandwidth graph image.
	OutBoundTotalBytes *uint `json:"outBoundTotalBytes,omitempty" xmlrpc:"outBoundTotalBytes,omitempty"`
}

// SoftLayer_Container_Bandwidth_Projection models projected bandwidth use over a time range.
type Container_Bandwidth_Projection struct {
	Entity

	// Bandwidth limit for this hardware.
	AllowedUsage *string `json:"allowedUsage,omitempty" xmlrpc:"allowedUsage,omitempty"`

	// Estimated bandwidth usage so far this billing cycle.
	EstimatedUsage *string `json:"estimatedUsage,omitempty" xmlrpc:"estimatedUsage,omitempty"`

	// Hardware ID of server to monitor.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// Projected usage for this hardware based on previous usage this billing cycle.
	ProjectedUsage *string `json:"projectedUsage,omitempty" xmlrpc:"projectedUsage,omitempty"`

	// the text name of the server being monitored.
	ServerName *string `json:"serverName,omitempty" xmlrpc:"serverName,omitempty"`

	// The minimum date included in this list.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}

// no documentation yet
type Container_Billing_Currency_Format struct {
	Entity

	// no documentation yet
	Currency *string `json:"currency,omitempty" xmlrpc:"currency,omitempty"`

	// no documentation yet
	Display *int `json:"display,omitempty" xmlrpc:"display,omitempty"`

	// no documentation yet
	Format *string `json:"format,omitempty" xmlrpc:"format,omitempty"`

	// no documentation yet
	Locale *string `json:"locale,omitempty" xmlrpc:"locale,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Position *int `json:"position,omitempty" xmlrpc:"position,omitempty"`

	// no documentation yet
	Precision *int `json:"precision,omitempty" xmlrpc:"precision,omitempty"`

	// no documentation yet
	Script *string `json:"script,omitempty" xmlrpc:"script,omitempty"`

	// no documentation yet
	Service *string `json:"service,omitempty" xmlrpc:"service,omitempty"`

	// no documentation yet
	Symbol *string `json:"symbol,omitempty" xmlrpc:"symbol,omitempty"`

	// no documentation yet
	Tag *string `json:"tag,omitempty" xmlrpc:"tag,omitempty"`

	// no documentation yet
	Value *Float64 `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Container_Billing_Info_Ach struct {
	Entity

	// no documentation yet
	AccountNumber *string `json:"accountNumber,omitempty" xmlrpc:"accountNumber,omitempty"`

	// no documentation yet
	AccountType *string `json:"accountType,omitempty" xmlrpc:"accountType,omitempty"`

	// no documentation yet
	BankTransitNumber *string `json:"bankTransitNumber,omitempty" xmlrpc:"bankTransitNumber,omitempty"`

	// no documentation yet
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// no documentation yet
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// no documentation yet
	FederalTaxId *string `json:"federalTaxId,omitempty" xmlrpc:"federalTaxId,omitempty"`

	// no documentation yet
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// no documentation yet
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// no documentation yet
	PhoneNumber *string `json:"phoneNumber,omitempty" xmlrpc:"phoneNumber,omitempty"`

	// no documentation yet
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// no documentation yet
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// no documentation yet
	Street1 *string `json:"street1,omitempty" xmlrpc:"street1,omitempty"`

	// no documentation yet
	Street2 *string `json:"street2,omitempty" xmlrpc:"street2,omitempty"`
}

// This container is used to provide all the options for [[SoftLayer_Billing_Invoice/emailInvoices|emailInvoices]] in order to have the necessary invoices generated and links sent to the user's email.
type Container_Billing_Invoice_Email struct {
	Entity

	// Excel Invoices to email
	ExcelInvoiceIds []int `json:"excelInvoiceIds,omitempty" xmlrpc:"excelInvoiceIds,omitempty"`

	// PDF Invoice Details to email
	PdfDetailedInvoiceIds []int `json:"pdfDetailedInvoiceIds,omitempty" xmlrpc:"pdfDetailedInvoiceIds,omitempty"`

	// PDF Invoices to email
	PdfInvoiceIds []int `json:"pdfInvoiceIds,omitempty" xmlrpc:"pdfInvoiceIds,omitempty"`

	// The type of Invoices to be emailed [current|next]. If next is selected, the account id will be used.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// SoftLayer_Container_Billing_Order_Status models an order status.
type Container_Billing_Order_Status struct {
	Entity

	// The description of the status.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The keyname of the status.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// Contains user information used to request a manual Catalyst enrollment.
type Container_Catalyst_ManualEnrollmentRequest struct {
	Entity

	// Applicant's email address
	CustomerEmail *string `json:"customerEmail,omitempty" xmlrpc:"customerEmail,omitempty"`

	// Applicant's first and last name
	CustomerName *string `json:"customerName,omitempty" xmlrpc:"customerName,omitempty"`

	// Name of applicant's startup company
	StartupName *string `json:"startupName,omitempty" xmlrpc:"startupName,omitempty"`

	// Flag indicating whether (true) or not (false) and applicant is
	VentureAffiliationFlag *bool `json:"ventureAffiliationFlag,omitempty" xmlrpc:"ventureAffiliationFlag,omitempty"`

	// Name of the venture capital fund, if any, applicant is affiliated with
	VentureFundName *string `json:"ventureFundName,omitempty" xmlrpc:"ventureFundName,omitempty"`
}

// This container is used to hold country locale information.
type Container_Collection_Locale_CountryCode struct {
	Entity

	// no documentation yet
	LongName *string `json:"longName,omitempty" xmlrpc:"longName,omitempty"`

	// no documentation yet
	ShortName *string `json:"shortName,omitempty" xmlrpc:"shortName,omitempty"`

	// no documentation yet
	StateCodes []Container_Collection_Locale_StateCode `json:"stateCodes,omitempty" xmlrpc:"stateCodes,omitempty"`
}

// This container is used to hold information regarding a state or province.
type Container_Collection_Locale_StateCode struct {
	Entity

	// no documentation yet
	LongName *string `json:"longName,omitempty" xmlrpc:"longName,omitempty"`

	// no documentation yet
	ShortName *string `json:"shortName,omitempty" xmlrpc:"shortName,omitempty"`
}

// no documentation yet
type Container_Disk_Image_Capture_Template struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Summary *string `json:"summary,omitempty" xmlrpc:"summary,omitempty"`

	// no documentation yet
	Volumes []Container_Disk_Image_Capture_Template_Volume `json:"volumes,omitempty" xmlrpc:"volumes,omitempty"`
}

// no documentation yet
type Container_Disk_Image_Capture_Template_Volume struct {
	Entity

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Partitions []Container_Disk_Image_Capture_Template_Volume_Partition `json:"partitions,omitempty" xmlrpc:"partitions,omitempty"`
}

// no documentation yet
type Container_Disk_Image_Capture_Template_Volume_Partition struct {
	Entity

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Contact information container for domain registration
type Container_Dns_Domain_Registration_Contact struct {
	Entity

	// The street address of the contact.
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// The second line in the address of the contact.
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// The third line in the address of the contact.
	Address3 *string `json:"address3,omitempty" xmlrpc:"address3,omitempty"`

	// The city of the contact.
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// The 2-character Country code. (i.e. US)
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// The email address of the contact.
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// The fax number of the contact.
	Fax *string `json:"fax,omitempty" xmlrpc:"fax,omitempty"`

	// The first name of the contact.
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// The last name of the contact.
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// The organization name of the contact.
	OrganizationName *string `json:"organizationName,omitempty" xmlrpc:"organizationName,omitempty"`

	// The phone number of the contact.
	Phone *string `json:"phone,omitempty" xmlrpc:"phone,omitempty"`

	// The postal code of the contact.
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// The state of the contact.
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// The type of contact. The following are the valid types of contacts:
	// * admin
	// * owner
	// * billing
	// * tech
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// This container data type contains extended attributes information for a domain of country code TLD.
type Container_Dns_Domain_Registration_ExtendedAttribute struct {
	Entity

	// Indicates if this is a child of another extended attribute.
	ChildFlag *bool `json:"childFlag,omitempty" xmlrpc:"childFlag,omitempty"`

	// The description of an extended attribute.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The name of an extended attribute.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The collection of options for an extended attribute.
	Options []Container_Dns_Domain_Registration_ExtendedAttribute_Option `json:"options,omitempty" xmlrpc:"options,omitempty"`

	// Indicates if extended attribute is required.
	RequiredFlag *int `json:"requiredFlag,omitempty" xmlrpc:"requiredFlag,omitempty"`

	// User defined indicates that the value is required from outside sources.
	UserDefinedFlag *bool `json:"userDefinedFlag,omitempty" xmlrpc:"userDefinedFlag,omitempty"`
}

// This is the data type that may need to be populated to complete registraton for domains that are country code TLD's.
type Container_Dns_Domain_Registration_ExtendedAttribute_Configuration struct {
	Entity

	// The extended attribute name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The extended attribute option value.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// This container data type contains extended attribute options information for a domain of country code TLD.
type Container_Dns_Domain_Registration_ExtendedAttribute_Option struct {
	Entity

	// The description of an option.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Extended Attribute that is required for an option.
	RequireExtendedAttributes []Container_Dns_Domain_Registration_ExtendedAttribute_Option_Require `json:"requireExtendedAttributes,omitempty" xmlrpc:"requireExtendedAttributes,omitempty"`

	// The title of an option.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`

	// The value of an option.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// This container data type contains the extended attribute name that is required by an extended attribute option.
type Container_Dns_Domain_Registration_ExtendedAttribute_Option_Require struct {
	Entity

	// The name of an extended attribute that is required by an extended attribute option.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Information container for domain registration
type Container_Dns_Domain_Registration_Information struct {
	Entity

	// The information of the registered domain.
	Contacts []Container_Dns_Domain_Registration_Contact `json:"contacts,omitempty" xmlrpc:"contacts,omitempty"`

	// The date that a domain is set to expire.
	ExpireDate *Time `json:"expireDate,omitempty" xmlrpc:"expireDate,omitempty"`

	// The list of nameservers for the domain.
	Nameservers []Container_Dns_Domain_Registration_Nameserver `json:"nameservers,omitempty" xmlrpc:"nameservers,omitempty"`

	// no documentation yet
	RegistryCreateDate *Time `json:"registryCreateDate,omitempty" xmlrpc:"registryCreateDate,omitempty"`

	// no documentation yet
	RegistryExpireDate *Time `json:"registryExpireDate,omitempty" xmlrpc:"registryExpireDate,omitempty"`

	// no documentation yet
	RegistryUpdateDate *Time `json:"registryUpdateDate,omitempty" xmlrpc:"registryUpdateDate,omitempty"`
}

// no documentation yet
type Container_Dns_Domain_Registration_List struct {
	Entity

	// The domain name.
	DomainName *string `json:"domainName,omitempty" xmlrpc:"domainName,omitempty"`

	// Three-character language tag for the IDN domain that you're trying to register. This is only required for IDN domains.
	EncodingType *string `json:"encodingType,omitempty" xmlrpc:"encodingType,omitempty"`

	// Data required by the Registry for some country code top level domains (i.e. example.us).
	//
	// In order to determine if a domain requires extended attributes, use [[SoftLayer_Dns_Domain_Registration::getExtendedAttributes|domain registration]] service.
	ExtendedAttributeConfiguration []Container_Dns_Domain_Registration_ExtendedAttribute_Configuration `json:"extendedAttributeConfiguration,omitempty" xmlrpc:"extendedAttributeConfiguration,omitempty"`

	// The length of the registration period in years. Valid values are 1 – 10.
	RegistrationPeriod *int `json:"registrationPeriod,omitempty" xmlrpc:"registrationPeriod,omitempty"`
}

// Lookup domain container for domain registration
type Container_Dns_Domain_Registration_Lookup struct {
	Entity

	// The list of available and taken domain names.
	Items []Container_Dns_Domain_Registration_Lookup_Items `json:"items,omitempty" xmlrpc:"items,omitempty"`
}

// Lookup items container for domain registration
type Container_Dns_Domain_Registration_Lookup_Items struct {
	Entity

	// The domain name.
	DomainName *string `json:"domainName,omitempty" xmlrpc:"domainName,omitempty"`

	// The status of the domain name if available and can be registered.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// Nameserver container for domain registration
type Container_Dns_Domain_Registration_Nameserver struct {
	Entity

	// The list of fully qualified names of the nameserver.
	Nameservers []Container_Dns_Domain_Registration_Nameserver_List `json:"nameservers,omitempty" xmlrpc:"nameservers,omitempty"`
}

// Nameservers list container for domain registration
type Container_Dns_Domain_Registration_Nameserver_List struct {
	Entity

	// The IPv4 address of the nameserver.
	Ipv4Address *string `json:"ipv4Address,omitempty" xmlrpc:"ipv4Address,omitempty"`

	// The IPv6 address of the nameserver.
	Ipv6Address *string `json:"ipv6Address,omitempty" xmlrpc:"ipv6Address,omitempty"`

	// The fully qualified name of the nameserver
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The sort order of the nameserver
	SortOrder *int `json:"sortOrder,omitempty" xmlrpc:"sortOrder,omitempty"`
}

// no documentation yet
type Container_Dns_Domain_Registration_Registrant_Verification_StatusDetail struct {
	Entity

	// The current status of the verification.
	Status *Dns_Domain_Registration_Registrant_Verification_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The adate when the domain will be suspended.
	VerificationDeadlineDate *Time `json:"verificationDeadlineDate,omitempty" xmlrpc:"verificationDeadlineDate,omitempty"`
}

// Transfer Information container for domain registration
type Container_Dns_Domain_Registration_Transfer_Information struct {
	Entity

	// The reason why a domain is not transferable.
	Reason *string `json:"reason,omitempty" xmlrpc:"reason,omitempty"`

	// The registrant email.
	RegistrantEmail *string `json:"registrantEmail,omitempty" xmlrpc:"registrantEmail,omitempty"`

	// The status of the latest transfer on the domain.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The date and time of the most recent update to the state of the transfer.
	TimeStamp *Time `json:"timeStamp,omitempty" xmlrpc:"timeStamp,omitempty"`

	// Indicates if the domain can be transferred.
	Transferrable *int `json:"transferrable,omitempty" xmlrpc:"transferrable,omitempty"`
}

// The SoftLayer_Container_Exception data type represents a SoftLayer_Exception.
type Container_Exception struct {
	Entity

	// The SoftLayer_Exception class that the error is.
	ExceptionClass *string `json:"exceptionClass,omitempty" xmlrpc:"exceptionClass,omitempty"`

	// The exception message.
	ExceptionMessage *string `json:"exceptionMessage,omitempty" xmlrpc:"exceptionMessage,omitempty"`
}

// no documentation yet
type Container_Graph struct {
	Entity

	// base units associated with the graph.
	BaseUnit *string `json:"baseUnit,omitempty" xmlrpc:"baseUnit,omitempty"`

	// Graph range end datetime.
	EndDatetime *string `json:"endDatetime,omitempty" xmlrpc:"endDatetime,omitempty"`

	// The height of the graph image.
	Height *int `json:"height,omitempty" xmlrpc:"height,omitempty"`

	// The graph image.
	Image *[]byte `json:"image,omitempty" xmlrpc:"image,omitempty"`

	// The graph interval in seconds.
	Interval *int `json:"interval,omitempty" xmlrpc:"interval,omitempty"`

	// Metric types associated with the graph.
	Metrics []Container_Metric_Data_Type `json:"metrics,omitempty" xmlrpc:"metrics,omitempty"`

	// Indicator to control whether the graph data is normalized.
	NormalizeFlag *[]byte `json:"normalizeFlag,omitempty" xmlrpc:"normalizeFlag,omitempty"`

	// The options used to control the graph appearance.
	Options []Container_Graph_Option `json:"options,omitempty" xmlrpc:"options,omitempty"`

	// A collection of graph plots.
	Plots []Container_Graph_Plot `json:"plots,omitempty" xmlrpc:"plots,omitempty"`

	// option to not return the image.
	ReturnImage *bool `json:"returnImage,omitempty" xmlrpc:"returnImage,omitempty"`

	// Graph range start datetime.
	StartDatetime *string `json:"startDatetime,omitempty" xmlrpc:"startDatetime,omitempty"`

	// The name of the template to use; may be null.
	Template *string `json:"template,omitempty" xmlrpc:"template,omitempty"`

	// The title of the graph image.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`

	// The width of the graph image.
	Width *int `json:"width,omitempty" xmlrpc:"width,omitempty"`
}

// no documentation yet
type Container_Graph_Option struct {
	Entity

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Container_Graph_Plot struct {
	Entity

	// no documentation yet
	Data []Container_Graph_Plot_Coordinate `json:"data,omitempty" xmlrpc:"data,omitempty"`

	// no documentation yet
	Metric *Container_Metric_Data_Type `json:"metric,omitempty" xmlrpc:"metric,omitempty"`

	// no documentation yet
	Unit *string `json:"unit,omitempty" xmlrpc:"unit,omitempty"`
}

// no documentation yet
type Container_Graph_Plot_Coordinate struct {
	Entity

	// no documentation yet
	XValue *Float64 `json:"xValue,omitempty" xmlrpc:"xValue,omitempty"`

	// no documentation yet
	YValue *Float64 `json:"yValue,omitempty" xmlrpc:"yValue,omitempty"`

	// no documentation yet
	ZValue *Float64 `json:"zValue,omitempty" xmlrpc:"zValue,omitempty"`
}

// The hardware configuration container is used to provide configuration options for servers.
//
// Each configuration option will include both an <code>itemPrice</code> and a <code>template</code>.
//
// The <code>itemPrice</code> value will provide hourly and monthly costs (if either are applicable), and a description of the option.
//
// The <code>template</code> will provide a fragment of the request with the properties and values that must be sent when creating a server with the option.
//
// The [[SoftLayer_Hardware/getCreateObjectOptions|getCreateObjectOptions]] method returns this data structure.
//
// <style type="text/css">#properties .views-field-body p { margin-top: 1.5em; };</style>
type Container_Hardware_Configuration struct {
	Entity

	//
	// <div style="width: 200%">
	// Available datacenter options.
	//
	//
	// The <code>datacenter.name</code> value in the template represents which datacenter the server will be provisioned in.
	// </div>
	Datacenters []Container_Hardware_Configuration_Option `json:"datacenters,omitempty" xmlrpc:"datacenters,omitempty"`

	//
	// <div style="width: 200%">
	// Available fixed configuration preset options.
	//
	//
	// The <code>fixedConfigurationPreset.keyName</code> value in the template is an identifier for a particular fixed configuration. When provided exactly as shown in the template, that fixed configuration will be used.
	//
	//
	// When providing a <code>fixedConfigurationPreset.keyName</code> while ordering a server the <code>processors</code> and <code>hardDrives</code> configuration options cannot be used.
	// </div>
	FixedConfigurationPresets []Container_Hardware_Configuration_Option `json:"fixedConfigurationPresets,omitempty" xmlrpc:"fixedConfigurationPresets,omitempty"`

	//
	// <div style="width: 200%">
	// Available hard drive options.
	//
	//
	// A server will have at least one hard drive.
	//
	//
	// The <code>hardDrives.capacity</code> value in the template represents the size, in gigabytes, of the disk.
	// </div>
	HardDrives []Container_Hardware_Configuration_Option `json:"hardDrives,omitempty" xmlrpc:"hardDrives,omitempty"`

	//
	// <div style="width: 200%">
	// Available network component options.
	//
	//
	// The <code>networkComponent.maxSpeed</code> value in the template represents the link speed, in megabits per second, of the network connections for a server.
	// </div>
	NetworkComponents []Container_Hardware_Configuration_Option `json:"networkComponents,omitempty" xmlrpc:"networkComponents,omitempty"`

	//
	// <div style="width: 200%">
	// Available operating system options.
	//
	//
	// The <code>operatingSystemReferenceCode</code> value in the template is an identifier for a particular operating system. When provided exactly as shown in the template, that operating system will be used.
	//
	//
	// A reference code is structured as three tokens separated by underscores. The first token represents the product, the second is the version of the product, and the third is whether the OS is 32 or 64bit.
	//
	//
	// When providing an <code>operatingSystemReferenceCode</code> while ordering a server the only token required to match exactly is the product. The version token may be given as 'LATEST', else it will require an exact match as well. When the bits token is not provided, 64 bits will be assumed.
	//
	//
	// Providing the value of 'LATEST' for a version will select the latest release of that product for the operating system. As this may change over time, you should be sure that the release version is irrelevant for your applications.
	//
	//
	// For Windows based operating systems the version will represent both the release version (2008, 2012, etc) and the edition (Standard, Enterprise, etc). For all other operating systems the version will represent the major version (Centos 6, Ubuntu 12, etc) of that operating system, minor versions are represented in few reference codes where they are significant.
	// </div>
	OperatingSystems []Container_Hardware_Configuration_Option `json:"operatingSystems,omitempty" xmlrpc:"operatingSystems,omitempty"`

	//
	// <div style="width: 200%">
	// Available processor options.
	//
	//
	// The <code>processorCoreAmount</code> value in the template represents the number of cores allocated to the server.
	// The <code>memoryCapacity</code> value in the template represents the amount of memory, in gigabytes, allocated to the server.
	// </div>
	Processors []Container_Hardware_Configuration_Option `json:"processors,omitempty" xmlrpc:"processors,omitempty"`
}

// An option found within a [[SoftLayer_Container_Hardware_Configuration (type)]] structure.
type Container_Hardware_Configuration_Option struct {
	Entity

	//
	// Provides hourly and monthly costs (if either are applicable), and a description of the option.
	ItemPrice *Product_Item_Price `json:"itemPrice,omitempty" xmlrpc:"itemPrice,omitempty"`

	//
	// Provides a description of a fixed configuration preset with monthly and hourly costs.
	Preset *Product_Package_Preset `json:"preset,omitempty" xmlrpc:"preset,omitempty"`

	//
	// Provides a fragment of the request with the properties and values that must be sent when creating a server with the option.
	Template *Hardware `json:"template,omitempty" xmlrpc:"template,omitempty"`
}

// no documentation yet
type Container_Hardware_MassUpdate struct {
	Entity

	// The hardwares updated by the mass update tool
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// Errors encountered while mass updating hardwares
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// The hardwares that failed to update
	SuccessFlag *string `json:"successFlag,omitempty" xmlrpc:"successFlag,omitempty"`
}

// The SoftLayer_Container_Hardware_Server_Configuration data type contains information relating to a server's item price information, and hard drive partition information.
type Container_Hardware_Server_Configuration struct {
	Entity

	// A flag indicating that the server will be moved into the spare pool after an Operating system reload.
	AddToSparePoolAfterOsReload *int `json:"addToSparePoolAfterOsReload,omitempty" xmlrpc:"addToSparePoolAfterOsReload,omitempty"`

	// The customer provision uri will be used to download and execute a customer defined script on the host at the end of provisioning.
	CustomProvisionScriptUri *string `json:"customProvisionScriptUri,omitempty" xmlrpc:"customProvisionScriptUri,omitempty"`

	// A flag indicating that the primary drive will be converted to a portable storage volume during an Operating System reload.
	DriveRetentionFlag *bool `json:"driveRetentionFlag,omitempty" xmlrpc:"driveRetentionFlag,omitempty"`

	// A flag indicating that all data will be erased from drives during an Operating System reload.
	EraseHardDrives *int `json:"eraseHardDrives,omitempty" xmlrpc:"eraseHardDrives,omitempty"`

	// The hard drive partitions that a server can be partitioned with.
	HardDrives []Hardware_Component `json:"hardDrives,omitempty" xmlrpc:"hardDrives,omitempty"`

	// An Image Template ID [[SoftLayer_Virtual_Guest_Block_Device_Template_Group]] that will be deployed to the host.  If provided no item prices are required.
	ImageTemplateId *int `json:"imageTemplateId,omitempty" xmlrpc:"imageTemplateId,omitempty"`

	// The item prices that a server can be configured with.
	ItemPrices []Product_Item_Price `json:"itemPrices,omitempty" xmlrpc:"itemPrices,omitempty"`

	// A flag indicating that the remote management cards password will be reset.
	ResetIpmiPassword *int `json:"resetIpmiPassword,omitempty" xmlrpc:"resetIpmiPassword,omitempty"`

	// IDs to SoftLayer_Security_Ssh_Key objects on the current account which will be added to the server for authentication. SSH Keys will not be added to servers with Microsoft Windows.
	SshKeyIds []int `json:"sshKeyIds,omitempty" xmlrpc:"sshKeyIds,omitempty"`

	// A flag indicating that the BIOS will be updated when installing the operating system.
	UpgradeBios *int `json:"upgradeBios,omitempty" xmlrpc:"upgradeBios,omitempty"`

	// A flag indicating that the firmware on all hard drives will be updated when installing the operating system.
	UpgradeHardDriveFirmware *int `json:"upgradeHardDriveFirmware,omitempty" xmlrpc:"upgradeHardDriveFirmware,omitempty"`
}

// The SoftLayer_Container_Hardware_Server_Details data type contains information relating to a server's component information, network information, and software information.
type Container_Hardware_Server_Details struct {
	Entity

	// The components that belong to a piece of hardware.
	Components []Hardware_Component `json:"components,omitempty" xmlrpc:"components,omitempty"`

	// The network components that belong to a piece of hardware.
	NetworkComponents []Network_Component `json:"networkComponents,omitempty" xmlrpc:"networkComponents,omitempty"`

	// The software that belong to a piece of hardware.
	Software []Software_Component `json:"software,omitempty" xmlrpc:"software,omitempty"`
}

// SoftLayer_Container_KnowledgeLayer_QuestionAnswer models a single question and answer pair from SoftLayer's KnowledgeLayer knowledge base. SoftLayer's backend network interfaces with the KnowledgeLayer to recommend helpful articles when support tickets are created.
type Container_KnowledgeLayer_QuestionAnswer struct {
	Entity

	// The answer to a question asked on the SoftLayer KnowledgeLayer.
	Answer *string `json:"answer,omitempty" xmlrpc:"answer,omitempty"`

	// The link to a question asked on the SoftLayer KnowledgeLayer.
	Link *string `json:"link,omitempty" xmlrpc:"link,omitempty"`

	// A question asked on the SoftLayer KnowledgeLayer.
	Question *string `json:"question,omitempty" xmlrpc:"question,omitempty"`
}

// no documentation yet
type Container_Message struct {
	Entity

	// no documentation yet
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// no documentation yet
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// no documentation yet
type Container_Metric_Data_Type struct {
	Entity

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	SummaryType *string `json:"summaryType,omitempty" xmlrpc:"summaryType,omitempty"`

	// no documentation yet
	Unit *string `json:"unit,omitempty" xmlrpc:"unit,omitempty"`
}

// SoftLayer_Container_Metric_Tracking_Object_Details This container is a parent class for detailing diverse metrics.
type Container_Metric_Tracking_Object_Details struct {
	Entity

	// The name that best describes the metric being collected.
	MetricName *string `json:"metricName,omitempty" xmlrpc:"metricName,omitempty"`
}

// SoftLayer_Container_Metric_Tracking_Object_Summary This container is a parent class for summarizing diverse metrics.
type Container_Metric_Tracking_Object_Summary struct {
	Entity

	// The name that best describes the metric being collected.
	MetricName *string `json:"metricName,omitempty" xmlrpc:"metricName,omitempty"`
}

// SoftLayer_Container_Metric_Tracking_Object_Virtual_Host_Details This container details a virtual host's metric data.
type Container_Metric_Tracking_Object_Virtual_Host_Details struct {
	Container_Metric_Tracking_Object_Details

	// The day this metric was collected.
	Day *Time `json:"day,omitempty" xmlrpc:"day,omitempty"`

	// The maximum number of guests hosted by this platform for the given day.
	MaxInstances *int `json:"maxInstances,omitempty" xmlrpc:"maxInstances,omitempty"`

	// The maximum amount of memory utilized by this platform for the given day.
	MaxMemoryUsage *int `json:"maxMemoryUsage,omitempty" xmlrpc:"maxMemoryUsage,omitempty"`

	// The mean number of guests hosted by this platform for the given day.
	MeanInstances *Float64 `json:"meanInstances,omitempty" xmlrpc:"meanInstances,omitempty"`

	// The mean amount of memory utilized by this platform for the given day.
	MeanMemoryUsage *Float64 `json:"meanMemoryUsage,omitempty" xmlrpc:"meanMemoryUsage,omitempty"`

	// The minimum number of guests hosted by this platform for the given day.
	MinInstances *int `json:"minInstances,omitempty" xmlrpc:"minInstances,omitempty"`

	// The minimum amount of memory utilized by this platform for the given day.
	MinMemoryUsage *int `json:"minMemoryUsage,omitempty" xmlrpc:"minMemoryUsage,omitempty"`
}

// SoftLayer_Container_Metric_Tracking_Object_Virtual_Host_Summary This container summarizes a virtual host's metric data.
type Container_Metric_Tracking_Object_Virtual_Host_Summary struct {
	Container_Metric_Tracking_Object_Summary

	// The average amount of memory usage thus far in this billing cycle.
	AvgMemoryUsageInBillingCycle *int `json:"avgMemoryUsageInBillingCycle,omitempty" xmlrpc:"avgMemoryUsageInBillingCycle,omitempty"`

	// Current bill cycle end date.
	CurrentBillCycleEnd *Time `json:"currentBillCycleEnd,omitempty" xmlrpc:"currentBillCycleEnd,omitempty"`

	// Current bill cycle start date.
	CurrentBillCycleStart *Time `json:"currentBillCycleStart,omitempty" xmlrpc:"currentBillCycleStart,omitempty"`

	// The last count of instances this platform was hosting.
	LastInstanceCount *int `json:"lastInstanceCount,omitempty" xmlrpc:"lastInstanceCount,omitempty"`

	// The last amount of memory this platform was using.
	LastMemoryUsageAmount *int `json:"lastMemoryUsageAmount,omitempty" xmlrpc:"lastMemoryUsageAmount,omitempty"`

	// The last time this virtual host was polled for metrics.
	LastPollTime *Time `json:"lastPollTime,omitempty" xmlrpc:"lastPollTime,omitempty"`

	// The max number of instances hosted thus far in this billing cycle.
	MaxInstanceInBillingCycle *int `json:"maxInstanceInBillingCycle,omitempty" xmlrpc:"maxInstanceInBillingCycle,omitempty"`

	// Previous bill cycle end date.
	PreviousBillCycleEnd *Time `json:"previousBillCycleEnd,omitempty" xmlrpc:"previousBillCycleEnd,omitempty"`

	// Previous bill cycle start date.
	PreviousBillCycleStart *Time `json:"previousBillCycleStart,omitempty" xmlrpc:"previousBillCycleStart,omitempty"`

	// This virtual hosting platform name.
	VirtualPlatformName *string `json:"virtualPlatformName,omitempty" xmlrpc:"virtualPlatformName,omitempty"`
}

// The SoftLayer_Container_Monitoring_Alarm_History data type contains information relating to SoftLayer monitoring alarm history.
type Container_Monitoring_Alarm_History struct {
	Entity

	// Account ID that this alarm belongs to
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// ID of the monitoring agent that triggered this alarm
	AgentId *int `json:"agentId,omitempty" xmlrpc:"agentId,omitempty"`

	// Alarm ID
	AlarmId *string `json:"alarmId,omitempty" xmlrpc:"alarmId,omitempty"`

	// Time that an alarm was closed.
	ClosedDate *Time `json:"closedDate,omitempty" xmlrpc:"closedDate,omitempty"`

	// Time that an alarm was triggered
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Alarm message
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// Robot ID
	RobotId *int `json:"robotId,omitempty" xmlrpc:"robotId,omitempty"`

	// Severity of an alarm
	Severity *string `json:"severity,omitempty" xmlrpc:"severity,omitempty"`
}

// SoftLayer_Container_Monitoring_Graph_Outputs models a single outbound object for a graph of given data sets.
type Container_Monitoring_Graph_Outputs struct {
	Entity

	// The maximum date included in this graph.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// Error message encountered during graphing
	GraphError *string `json:"graphError,omitempty" xmlrpc:"graphError,omitempty"`

	// The raw PNG binary data to be displayed once the graph is drawn.
	GraphImage *[]byte `json:"graphImage,omitempty" xmlrpc:"graphImage,omitempty"`

	// The minimum date included in this graph.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}

// This object holds authentication data to a server.
type Container_Network_Authentication_Data struct {
	Entity

	// The name of a host
	Host *string `json:"host,omitempty" xmlrpc:"host,omitempty"`

	// The authentication password
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// The port number
	Port *int `json:"port,omitempty" xmlrpc:"port,omitempty"`

	// The type of network protocol. This can be ftp, ssh and so on.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The authentication username
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}

// SoftLayer_Container_Network_Bandwidth_Data_Summary models an interface's overall bandwidth usage during it's current billing cycle.
type Container_Network_Bandwidth_Data_Summary struct {
	Entity

	// The amount of bandwidth a server has allocated to it in it's current billing period.
	AllowedUsage *Float64 `json:"allowedUsage,omitempty" xmlrpc:"allowedUsage,omitempty"`

	// The amount of bandwidth that a server has used within it's current billing period.
	EstimatedUsage *Float64 `json:"estimatedUsage,omitempty" xmlrpc:"estimatedUsage,omitempty"`

	// The amount of bandwidth a server is projected to use within its billing period, based on it's current usage.
	ProjectedUsage *Float64 `json:"projectedUsage,omitempty" xmlrpc:"projectedUsage,omitempty"`

	// The unit of measurement used in a bandwidth data summary.
	UsageUnits *string `json:"usageUnits,omitempty" xmlrpc:"usageUnits,omitempty"`
}

// SoftLayer_Container_Network_Bandwidth_Version1_Usage models an hourly bandwidth record.
type Container_Network_Bandwidth_Version1_Usage struct {
	Entity

	// The amount of incoming bandwidth that a server has used within the hour of the recordedDate.
	IncomingAmount *Float64 `json:"incomingAmount,omitempty" xmlrpc:"incomingAmount,omitempty"`

	// The amount of outgoing bandwidth that a server has used within the hour of the recordedDate.
	OutgoingAmount *Float64 `json:"outgoingAmount,omitempty" xmlrpc:"outgoingAmount,omitempty"`

	// The date and time that the bandwidth was used by a piece of hardware
	RecordedDate *Time `json:"recordedDate,omitempty" xmlrpc:"recordedDate,omitempty"`
}

// SoftLayer_Container_Network_ContentDelivery_Authentication_Directory represents a token authentication directory on your CDN FTP or on your origin server.
type Container_Network_ContentDelivery_Authentication_Directory struct {
	Entity

	// The date that a token authentication directory was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The name of a directory or a file within a directory listing.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The type of platform that a token authentication directory is defined for. Possible types are HTTP Large, HTTP Small, Flash and Windows Media
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// This container is used for CDN content authentication service.
type Container_Network_ContentDelivery_Authentication_Parameter struct {
	Entity

	// A CDN account name
	CdnAccountName *string `json:"cdnAccountName,omitempty" xmlrpc:"cdnAccountName,omitempty"`

	// A client IP address
	ClientIp *string `json:"clientIp,omitempty" xmlrpc:"clientIp,omitempty"`

	// A client referrer information
	Referrer *string `json:"referrer,omitempty" xmlrpc:"referrer,omitempty"`

	// A source URL
	SourceUrl *string `json:"sourceUrl,omitempty" xmlrpc:"sourceUrl,omitempty"`

	// An authentication token string
	Token *string `json:"token,omitempty" xmlrpc:"token,omitempty"`
}

// CDN supports the content authentication service. With the content authentication service, customers can control access to their contents. There are several scenarios where this authentication capability could be useful. Websites can prevent other rogue websites from linking to their videos. Content owners can prevent users from passing around http links, thus forcing them to login to view contents. It is also possible to authenticate via the client IP address. Referrer information is also checked if provided by a client's browser. servers will invoke a web service method to validate a content authentication token.
//
// CDN uses the default authentication web service provided by SoftLayer to validate a token. A customer can use their own implementation of the token authentication web service by using [[SoftLayer_Network_ContentDelivery_Account::setAuthenticationServiceEndpoint|setAuthenticationServiceEndpoint]] method.
//
// This container class holds the token validation web service endpoint information. CDN supports 3 different protocols: HTTP, RTMP (streaming Flash), and MMS (streaming Windows Media)
type Container_Network_ContentDelivery_Authentication_ServiceEndpoint struct {
	Entity

	// The authentication web service endpoint that CDN servers will use to validate a token
	Endpoint *string `json:"endpoint,omitempty" xmlrpc:"endpoint,omitempty"`

	// The protocol that the WSDL will be used for.  This can be HTTP, WINDOWSMEDIA, or FLASH
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`
}

// SoftLayer_Container_Network_ContentDelivery_Bandwidth_PointsOfPresence_Summary models an individual CDN point of presence's bandwidth usage for a CDN account within a given date range. CDN POPs are located throughout the world, so individual POP usage may be beneficial in determining who is downloading your CDN hosted content.
type Container_Network_ContentDelivery_Bandwidth_PointsOfPresence_Summary struct {
	Entity

	// The amount of bandwidth used by a CDN POP.
	Bandwidth *uint `json:"bandwidth,omitempty" xmlrpc:"bandwidth,omitempty"`

	// The ending date of a CDN POP bandwidth summary.
	EndDateTime *Time `json:"endDateTime,omitempty" xmlrpc:"endDateTime,omitempty"`

	// A CDN POP's name. This is typically the city the POP resides in.
	PopName *string `json:"popName,omitempty" xmlrpc:"popName,omitempty"`

	// The starting date of a CDN POP bandwidth summary.
	StartDateTime *Time `json:"startDateTime,omitempty" xmlrpc:"startDateTime,omitempty"`

	// The unit of measurement used in a CDN POP bandwidth summary.
	UsageUnits *string `json:"usageUnits,omitempty" xmlrpc:"usageUnits,omitempty"`

	// The view count
	ViewCount *uint `json:"viewCount,omitempty" xmlrpc:"viewCount,omitempty"`
}

// SoftLayer_Container_Network_ContentDelivery_Bandwidth_Summary models a CDN account's overall bandwidth usage and overages within a given date range.
type Container_Network_ContentDelivery_Bandwidth_Summary struct {
	Entity

	// The CDN account id
	CdnAccountId *int `json:"cdnAccountId,omitempty" xmlrpc:"cdnAccountId,omitempty"`

	// The ending date of a CDN bandwidth summary.
	EndDateTime *Time `json:"endDateTime,omitempty" xmlrpc:"endDateTime,omitempty"`

	// The name of a file that is requested by visitors
	FileName *string `json:"fileName,omitempty" xmlrpc:"fileName,omitempty"`

	// The media type
	MediaType *string `json:"mediaType,omitempty" xmlrpc:"mediaType,omitempty"`

	// The starting date of a CDN bandwidth summary.
	StartDateTime *Time `json:"startDateTime,omitempty" xmlrpc:"startDateTime,omitempty"`

	// The amount of bandwidth used by a CDN account in between a given starting and ending date.
	Usage *Float64 `json:"usage,omitempty" xmlrpc:"usage,omitempty"`

	// The unit of measurement used in a CDN bandwidth summary.
	UsageUnits *string `json:"usageUnits,omitempty" xmlrpc:"usageUnits,omitempty"`
}

// SoftLayer_Container_Network_ContentDelivery_Bandwidth_Summary_File models a CDN account's overall bandwidth usage and overages within a given date range.
type Container_Network_ContentDelivery_Bandwidth_Summary_Detail struct {
	Container_Network_ContentDelivery_Bandwidth_Summary

	// The duration of a file that is viewed.
	Duration *Float64 `json:"duration,omitempty" xmlrpc:"duration,omitempty"`

	// The number of times that a file is viewed.
	ViewCount *int `json:"viewCount,omitempty" xmlrpc:"viewCount,omitempty"`
}

// SoftLayer's CDN allows for multiple origin pull domains and CNAME records. This container holds the origin pull configuration details. CDN currently supports origin pull method for HTTP content.
type Container_Network_ContentDelivery_OriginPull_Mapping struct {
	Entity

	// The CNAME record.
	Cname *string `json:"cname,omitempty" xmlrpc:"cname,omitempty"`

	// The unique identifier of an origin pull configuration
	Id *string `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// This indicates if an origin pull mapping is for the secure content or not.
	IsSecureContent *bool `json:"isSecureContent,omitempty" xmlrpc:"isSecureContent,omitempty"`

	// The type of a media supported by CDN. Supported media types are: "HTTP", "FLASH" and "WM"
	MediaType *string `json:"mediaType,omitempty" xmlrpc:"mediaType,omitempty"`

	// The URL of a origin server.  A URL can contain a directory path.
	OriginUrl *string `json:"originUrl,omitempty" xmlrpc:"originUrl,omitempty"`
}

// SoftLayer's CDN content delivery network offering replicates your data to a number of Points of Presence (POP's) around the world. SoftLayer_Container_Network_ContentDelivery_PointsOfPresence models one of these POP locations.
type Container_Network_ContentDelivery_PointsOfPresence struct {
	Entity

	// A CDN Point of Presence's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A CDN Point of Presence's name. This is typically the city that the POP is located in.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// This container holds information on a purge request. [[SoftLayer_Network_ContentDelivery_Account::purgeCache|Purge method]] for more details.
//
// Status code can be "SUCCESS", "FAILED", or "INVALID_URL" "INVALID_URL" code is returned when a URL is malformed or does not belong to customer. "FAILED" is returned in case there was an internal error.
type Container_Network_ContentDelivery_PurgeService_Response struct {
	Entity

	// A status code indicates whether your request was successful or not
	StatusCode *string `json:"statusCode,omitempty" xmlrpc:"statusCode,omitempty"`

	// A URL that you wish to purge its cache object
	Url *string `json:"url,omitempty" xmlrpc:"url,omitempty"`
}

// no documentation yet
type Container_Network_ContentDelivery_Report_Usage struct {
	Entity

	// no documentation yet
	ApplicationDeliveryNetwork *Float64 `json:"applicationDeliveryNetwork,omitempty" xmlrpc:"applicationDeliveryNetwork,omitempty"`

	// no documentation yet
	ApplicationDeliveryNetworkSsl *Float64 `json:"applicationDeliveryNetworkSsl,omitempty" xmlrpc:"applicationDeliveryNetworkSsl,omitempty"`

	// no documentation yet
	DiskSpace *Float64 `json:"diskSpace,omitempty" xmlrpc:"diskSpace,omitempty"`

	// no documentation yet
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// no documentation yet
	Flash *Float64 `json:"flash,omitempty" xmlrpc:"flash,omitempty"`

	// no documentation yet
	Http *Float64 `json:"http,omitempty" xmlrpc:"http,omitempty"`

	// no documentation yet
	HttpSmall *Float64 `json:"httpSmall,omitempty" xmlrpc:"httpSmall,omitempty"`

	// no documentation yet
	Https *Float64 `json:"https,omitempty" xmlrpc:"https,omitempty"`

	// no documentation yet
	HttpsSmall *Float64 `json:"httpsSmall,omitempty" xmlrpc:"httpsSmall,omitempty"`

	// no documentation yet
	Region *string `json:"region,omitempty" xmlrpc:"region,omitempty"`

	// no documentation yet
	SslTotal *Float64 `json:"sslTotal,omitempty" xmlrpc:"sslTotal,omitempty"`

	// no documentation yet
	StandardTotal *Float64 `json:"standardTotal,omitempty" xmlrpc:"standardTotal,omitempty"`

	// no documentation yet
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// no documentation yet
	WindowsMedia *Float64 `json:"windowsMedia,omitempty" xmlrpc:"windowsMedia,omitempty"`
}

// SoftLayer's CDN content delivery network allows for multiple types of media hosting in addition to traditional HTTP hosting. Each of these media types are accessible form a different URL. SoftLayer_Container_Network_ContentDelivery_SupportedProtocol holds details about CDN supported media types and their associated URLs.
//
// CDN media URLs follow the standard <protocol>://<cdn-name>.<platform-name>.cdn.softlayer.net
//
// Flash streaming, Windows Media streaming and HTTP protocols are supported: Flash streaming: <nowiki>rtmp://<cdn-name>.flash.cdn.softlayer.net</nowiki> Windows Media streaming: <nowiki>mms://<cdn-name>.wm.cdn.softlayer.net</nowiki> HTTP: <nowiki>http://<cdn-name>.http.cdn.softlayer.net</nowiki>
type Container_Network_ContentDelivery_SupportedProtocol struct {
	Entity

	// The host name related to CDN supported media, and is represented in the hostname portion of a CDN URL.
	Host *string `json:"host,omitempty" xmlrpc:"host,omitempty"`

	// The type of a media supported by CDN such as "FLASH", "WINDOWSMEDIA" or "HTTP"
	MediaType *string `json:"mediaType,omitempty" xmlrpc:"mediaType,omitempty"`

	// The platform name. It's a friendly name of media type.
	Platform *string `json:"platform,omitempty" xmlrpc:"platform,omitempty"`

	// The media protocol supported by CDN. This represents the media portion of a CDN URL.  Currently supported protocols are: rtmp, mms and http
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`
}

// SoftLayer_Container_Network_Directory_Listing represents a single entry in a listing of files within a remote directory. API methods that return remote directory listings typically return arrays of SoftLayer_Container_Network_Directory_Listing objects.
type Container_Network_Directory_Listing struct {
	Entity

	// If the file in a directory listing is a directory itself then fileCount is the number of files within the directory.
	FileCount *int `json:"fileCount,omitempty" xmlrpc:"fileCount,omitempty"`

	// The name of a directory or a file within a directory listing.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The type of file in a directory listing. If a directory listing entry is a directory itself then type is set to "directory". Otherwise, type is a blank string.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// The IntrusionProtection_Event object stores information about individual intrusion protection events.
//
// It is a data container that cannot be edited, deleted, or saved.
//
// It is returned by many methods in the TippingPointReporting object, but never directly, always as a child of another container object.
type Container_Network_IntrusionProtection_Event struct {
	Entity

	// The CVE ID(s), if any, associated with this attack signature.
	CVEId *string `json:"CVEId,omitempty" xmlrpc:"CVEId,omitempty"`

	// The action that was taken when this attack was discovered.  Can be either "Block" or "Permit"
	ActionTaken *string `json:"actionTaken,omitempty" xmlrpc:"actionTaken,omitempty"`

	// The number of attacks in this block.  Attacks are grouped differently based on the query performed on the tippingPointReporting object.
	AttackCount *int `json:"attackCount,omitempty" xmlrpc:"attackCount,omitempty"`

	// Long description of the attack.  May contain links to more information
	AttackLongDescription *string `json:"attackLongDescription,omitempty" xmlrpc:"attackLongDescription,omitempty"`

	// Name of the attack
	AttackName *string `json:"attackName,omitempty" xmlrpc:"attackName,omitempty"`

	// The starting timestamp of the attack recorded, in Y-m-d H:i:s format.  May not be set, depending on the type of query performed.
	BeginTime *string `json:"beginTime,omitempty" xmlrpc:"beginTime,omitempty"`

	// The BugTraq ID(s), if any, associated with this attack signature.
	BugtraqId *string `json:"bugtraqId,omitempty" xmlrpc:"bugtraqId,omitempty"`

	// The human-readable classification of the attack
	Classification *string `json:"classification,omitempty" xmlrpc:"classification,omitempty"`

	// The IP Address (as a dotted decimal string) of the machine that was the target of the attack
	DestinationIpAddress *string `json:"destinationIpAddress,omitempty" xmlrpc:"destinationIpAddress,omitempty"`

	// The port the attack was directed at
	DestinationPort *int `json:"destinationPort,omitempty" xmlrpc:"destinationPort,omitempty"`

	// The ending timestamp of the attack recorded, in Y-m-d H:i:s format.  May not be set, depending on the type of query performed.
	EndTime *string `json:"endTime,omitempty" xmlrpc:"endTime,omitempty"`

	// The platform affected by the attack
	Platform *string `json:"platform,omitempty" xmlrpc:"platform,omitempty"`

	// The protocol used in the attack
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`

	// The human-readable severity of this attack, from "Low" to "Critical"
	Severity *string `json:"severity,omitempty" xmlrpc:"severity,omitempty"`

	// Unique ID of the "Signature" in question.  The signature determines the type of attack recorded.  SignatureId is used in the drillDown() function on the TippingPointReporting service
	SignatureId *string `json:"signatureId,omitempty" xmlrpc:"signatureId,omitempty"`

	// The IP Address (as a dotted decimal string) of the machine originating the attack
	SourceIpAddress *string `json:"sourceIpAddress,omitempty" xmlrpc:"sourceIpAddress,omitempty"`

	// The port the attack originated from
	SourcePort *int `json:"sourcePort,omitempty" xmlrpc:"sourcePort,omitempty"`
}

// The IntrusionProtection_Statistic is used exclusively by the getMainStatistics method on the TippingPointReporting service, and serves mainly as a pair object, storing a name and an attack count.  Name is usually the name of an attack, but it can also be an attacking IP Address
type Container_Network_IntrusionProtection_Statistic struct {
	Entity

	// The number of attacks effecting this name over the time period
	AttackCount *int `json:"attackCount,omitempty" xmlrpc:"attackCount,omitempty"`

	// Either the name of the attack in question, or the attacking IP Address
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The IntrusionProtection_Statistics Type is used as a container for SoftLayer_Container_Network_IntrusionProtection_Statistic objects.  The SoftLayer_Container_Network_IntrusionProtection_Statistics class holds the "header" information, like the item being queried (either account or data center), the time frame, and the grand total of the attacks.
type Container_Network_IntrusionProtection_Statistics struct {
	Entity

	// The actual target, either a datacenter name, an account ID, or a subnet IP
	Target *string `json:"target,omitempty" xmlrpc:"target,omitempty"`

	// The type of the target, right now either "datacenter", "account", or "subnet"
	TargetType *string `json:"targetType,omitempty" xmlrpc:"targetType,omitempty"`

	// The time frame of the attack, in string form, like "Last 24 hours"
	TimeFrame *string `json:"timeFrame,omitempty" xmlrpc:"timeFrame,omitempty"`

	// The top attacks for this target over this time frame
	TopAttacks []Container_Network_IntrusionProtection_Statistic `json:"topAttacks,omitempty" xmlrpc:"topAttacks,omitempty"`

	// Total attacks for this $target over this time frame
	TotalAttacks *int `json:"totalAttacks,omitempty" xmlrpc:"totalAttacks,omitempty"`
}

// The IntrusionProtection_SubnetReport object is the container that holds the SoftLayer_Container_Network_IntrusionProtection_Event objects for a particular subnet, or "All Subnets", whatever the case may be.  Subnet, subnet mask, direction, and the individual events are returned by this object.
type Container_Network_IntrusionProtection_SubnetReport struct {
	Entity

	// cidr for this report.  If the subnetIpAddress is "All Subnets", this is set to 32 and should be ignored.
	Cidr *int `json:"cidr,omitempty" xmlrpc:"cidr,omitempty"`

	// Direction of the attack, either 'Inbound' or 'Outbound'
	Direction *string `json:"direction,omitempty" xmlrpc:"direction,omitempty"`

	// The class SoftLayer_Container_Network_IntrusionProtection_Event objects on this report.
	Events []Container_Network_IntrusionProtection_Event `json:"events,omitempty" xmlrpc:"events,omitempty"`

	// The "target" of this report, could be an IP address, a subnet's network identifier, or "All Subnets"
	SubnetIpAddress *string `json:"subnetIpAddress,omitempty" xmlrpc:"subnetIpAddress,omitempty"`
}

// The LoadBalancer_StatusEntry object stores information about the current status of a particular load balancer service.
//
// It is a data container that cannot be edited, deleted, or saved.
//
// It is returned exclusively by the getStatus method on the [[SoftLayer_Network_LoadBalancer_Service]] service
type Container_Network_LoadBalancer_StatusEntry struct {
	Entity

	// The value of the entry.
	Content *string `json:"content,omitempty" xmlrpc:"content,omitempty"`

	// Text description of the status entry
	Label *string `json:"label,omitempty" xmlrpc:"label,omitempty"`
}

// This container class holds information on a media file such as file name, codec, frame rate and so on
type Container_Network_Media_Information struct {
	Entity

	// The audio bit rate
	AudioBitRate *int `json:"audioBitRate,omitempty" xmlrpc:"audioBitRate,omitempty"`

	// The audio channel mode
	AudioChannelMode *string `json:"audioChannelMode,omitempty" xmlrpc:"audioChannelMode,omitempty"`

	// The number of audio channels
	AudioChannels *int `json:"audioChannels,omitempty" xmlrpc:"audioChannels,omitempty"`

	// The audio codec name
	AudioCodec *string `json:"audioCodec,omitempty" xmlrpc:"audioCodec,omitempty"`

	// The audio sample rate
	AudioSampleRate *int `json:"audioSampleRate,omitempty" xmlrpc:"audioSampleRate,omitempty"`

	// The duration of a media
	Duration *Float64 `json:"duration,omitempty" xmlrpc:"duration,omitempty"`

	// The error message if any.
	ErrorMessage *string `json:"errorMessage,omitempty" xmlrpc:"errorMessage,omitempty"`

	// The name of a media file
	File *string `json:"file,omitempty" xmlrpc:"file,omitempty"`

	// The file format
	FileFormat *string `json:"fileFormat,omitempty" xmlrpc:"fileFormat,omitempty"`

	// The size of a media file in byte
	FileSize *uint `json:"fileSize,omitempty" xmlrpc:"fileSize,omitempty"`

	// The frame rate
	FrameRate *Float64 `json:"frameRate,omitempty" xmlrpc:"frameRate,omitempty"`

	// The width of a media in pixel
	SizeX *int `json:"sizeX,omitempty" xmlrpc:"sizeX,omitempty"`

	// The height of a media in pixel
	SizeY *int `json:"sizeY,omitempty" xmlrpc:"sizeY,omitempty"`

	// The total of frames
	TotalFrames *uint `json:"totalFrames,omitempty" xmlrpc:"totalFrames,omitempty"`

	// The width in a video's width to height aspect ratio
	VideoAspectX *Float64 `json:"videoAspectX,omitempty" xmlrpc:"videoAspectX,omitempty"`

	// The height in a video's width to height aspect ratio
	VideoAspectY *int `json:"videoAspectY,omitempty" xmlrpc:"videoAspectY,omitempty"`

	// The video codec name
	VideoCodec *string `json:"videoCodec,omitempty" xmlrpc:"videoCodec,omitempty"`
}

// no documentation yet
type Container_Network_Media_Transcode_Job_Watermark struct {
	Entity

	// Time to stop showing watermark in milliseconds
	EndTime *int `json:"endTime,omitempty" xmlrpc:"endTime,omitempty"`

	// Filename of image to use as watermark in transcoding job
	FileName *string `json:"fileName,omitempty" xmlrpc:"fileName,omitempty"`

	// Position to place watermark at
	Position *Container_Network_Media_Transcode_Job_Watermark_Position `json:"position,omitempty" xmlrpc:"position,omitempty"`

	// Time to start showing watermark in milliseconds
	StartTime *int `json:"startTime,omitempty" xmlrpc:"startTime,omitempty"`

	// Text to Place in Watermark
	Text *string `json:"text,omitempty" xmlrpc:"text,omitempty"`

	// Percentage Transparent watermark should be
	TransparencyPercentage *int `json:"transparencyPercentage,omitempty" xmlrpc:"transparencyPercentage,omitempty"`
}

// no documentation yet
type Container_Network_Media_Transcode_Job_Watermark_Position struct {
	Entity

	// X Coordinate of Watermark
	X *int `json:"x,omitempty" xmlrpc:"x,omitempty"`

	// vertical Coordinate of Watermark
	Y *int `json:"y,omitempty" xmlrpc:"y,omitempty"`
}

// Transcode preset is a set of configuration parameters that defines a Transcode output format. SoftLayer_Container_Network_Media_Transcode_Preset contains a preset information defined on a Transcode server
type Container_Network_Media_Transcode_Preset struct {
	Entity

	// The unique id that is used by a Transcode server
	GUID *string `json:"GUID,omitempty" xmlrpc:"GUID,omitempty"`

	// The category that a preset belongs to
	Category *string `json:"category,omitempty" xmlrpc:"category,omitempty"`

	// The description of the preset
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The friendly name of a preset
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Transcode preset element
type Container_Network_Media_Transcode_Preset_Element struct {
	Entity

	// The additional elements for DROPDOWNLIST element
	AdditionalElements []Container_Network_Media_Transcode_Preset_Element_Option `json:"additionalElements,omitempty" xmlrpc:"additionalElements,omitempty"`

	// The default value of an element.
	DefaultValue *string `json:"defaultValue,omitempty" xmlrpc:"defaultValue,omitempty"`

	// The description of a preset element
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The flag that indicates whether an element is enabled or not
	Enabled *bool `json:"enabled,omitempty" xmlrpc:"enabled,omitempty"`

	// The extended description of a preset element
	ExtendedDescription *string `json:"extendedDescription,omitempty" xmlrpc:"extendedDescription,omitempty"`

	// The flag that indicates whether an element is hidden or not
	Hidden *bool `json:"hidden,omitempty" xmlrpc:"hidden,omitempty"`

	// The maximum value of an element
	MaximumValue *int `json:"maximumValue,omitempty" xmlrpc:"maximumValue,omitempty"`

	// The minimum value of an element
	MinimumValue *int `json:"minimumValue,omitempty" xmlrpc:"minimumValue,omitempty"`

	// The name of an preset element
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The name of a parent element
	ParentName *string `json:"parentName,omitempty" xmlrpc:"parentName,omitempty"`

	// The type of an preset element.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// Transcode preset element
type Container_Network_Media_Transcode_Preset_Element_Option struct {
	Entity

	// The name of a additional preset element
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The value of a additional preset element
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Container_Network_Message_Delivery_Email struct {
	Entity

	// no documentation yet
	Body *string `json:"body,omitempty" xmlrpc:"body,omitempty"`

	// no documentation yet
	ContainsHtml *bool `json:"containsHtml,omitempty" xmlrpc:"containsHtml,omitempty"`

	// no documentation yet
	From *string `json:"from,omitempty" xmlrpc:"from,omitempty"`

	// no documentation yet
	Subject *string `json:"subject,omitempty" xmlrpc:"subject,omitempty"`

	// no documentation yet
	To *string `json:"to,omitempty" xmlrpc:"to,omitempty"`
}

// no documentation yet
type Container_Network_Message_Delivery_Email_Sendgrid_Account_Overview struct {
	Entity

	// no documentation yet
	CreditsAllowed *int `json:"creditsAllowed,omitempty" xmlrpc:"creditsAllowed,omitempty"`

	// no documentation yet
	CreditsOverage *int `json:"creditsOverage,omitempty" xmlrpc:"creditsOverage,omitempty"`

	// no documentation yet
	CreditsRemain *int `json:"creditsRemain,omitempty" xmlrpc:"creditsRemain,omitempty"`

	// no documentation yet
	CreditsUsed *int `json:"creditsUsed,omitempty" xmlrpc:"creditsUsed,omitempty"`

	// no documentation yet
	Package *string `json:"package,omitempty" xmlrpc:"package,omitempty"`

	// no documentation yet
	Reputation *int `json:"reputation,omitempty" xmlrpc:"reputation,omitempty"`

	// no documentation yet
	Requests *int `json:"requests,omitempty" xmlrpc:"requests,omitempty"`
}

// no documentation yet
type Container_Network_Message_Delivery_Email_Sendgrid_Customer_Profile struct {
	Entity

	// no documentation yet
	Address *string `json:"address,omitempty" xmlrpc:"address,omitempty"`

	// no documentation yet
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// no documentation yet
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// no documentation yet
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// no documentation yet
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// no documentation yet
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// no documentation yet
	Phone *string `json:"phone,omitempty" xmlrpc:"phone,omitempty"`

	// no documentation yet
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// no documentation yet
	Website *string `json:"website,omitempty" xmlrpc:"website,omitempty"`

	// no documentation yet
	Zip *string `json:"zip,omitempty" xmlrpc:"zip,omitempty"`
}

// no documentation yet
type Container_Network_Message_Delivery_Email_Sendgrid_List_Entry struct {
	Entity

	// no documentation yet
	Created *string `json:"created,omitempty" xmlrpc:"created,omitempty"`

	// no documentation yet
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// no documentation yet
	Reason *string `json:"reason,omitempty" xmlrpc:"reason,omitempty"`

	// no documentation yet
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// no documentation yet
type Container_Network_Message_Delivery_Email_Sendgrid_Statistics struct {
	Entity

	// no documentation yet
	Blocks *int `json:"blocks,omitempty" xmlrpc:"blocks,omitempty"`

	// no documentation yet
	Bounces *int `json:"bounces,omitempty" xmlrpc:"bounces,omitempty"`

	// no documentation yet
	Clicks *int `json:"clicks,omitempty" xmlrpc:"clicks,omitempty"`

	// no documentation yet
	Date *string `json:"date,omitempty" xmlrpc:"date,omitempty"`

	// no documentation yet
	Delivered *int `json:"delivered,omitempty" xmlrpc:"delivered,omitempty"`

	// no documentation yet
	InvalidEmail *int `json:"invalidEmail,omitempty" xmlrpc:"invalidEmail,omitempty"`

	// no documentation yet
	Opens *int `json:"opens,omitempty" xmlrpc:"opens,omitempty"`

	// no documentation yet
	RepeatBounces *int `json:"repeatBounces,omitempty" xmlrpc:"repeatBounces,omitempty"`

	// no documentation yet
	RepeatSpamReports *int `json:"repeatSpamReports,omitempty" xmlrpc:"repeatSpamReports,omitempty"`

	// no documentation yet
	RepeatUnsubscribes *int `json:"repeatUnsubscribes,omitempty" xmlrpc:"repeatUnsubscribes,omitempty"`

	// no documentation yet
	Requests *int `json:"requests,omitempty" xmlrpc:"requests,omitempty"`

	// no documentation yet
	SpamReports *int `json:"spamReports,omitempty" xmlrpc:"spamReports,omitempty"`

	// no documentation yet
	UniqueClicks *int `json:"uniqueClicks,omitempty" xmlrpc:"uniqueClicks,omitempty"`

	// no documentation yet
	UniqueOpens *int `json:"uniqueOpens,omitempty" xmlrpc:"uniqueOpens,omitempty"`

	// no documentation yet
	Unsubscribes *int `json:"unsubscribes,omitempty" xmlrpc:"unsubscribes,omitempty"`
}

// no documentation yet
type Container_Network_Message_Delivery_Email_Sendgrid_Statistics_Graph struct {
	Entity

	// no documentation yet
	GraphError *string `json:"graphError,omitempty" xmlrpc:"graphError,omitempty"`

	// no documentation yet
	GraphImage *[]byte `json:"graphImage,omitempty" xmlrpc:"graphImage,omitempty"`

	// no documentation yet
	GraphTitle *string `json:"graphTitle,omitempty" xmlrpc:"graphTitle,omitempty"`
}

// no documentation yet
type Container_Network_Message_Delivery_Email_Sendgrid_Statistics_Options struct {
	Entity

	// no documentation yet
	AggregatesOnly *bool `json:"aggregatesOnly,omitempty" xmlrpc:"aggregatesOnly,omitempty"`

	// no documentation yet
	Category *string `json:"category,omitempty" xmlrpc:"category,omitempty"`

	// no documentation yet
	Days *int `json:"days,omitempty" xmlrpc:"days,omitempty"`

	// no documentation yet
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// no documentation yet
	SelectedStatistics []string `json:"selectedStatistics,omitempty" xmlrpc:"selectedStatistics,omitempty"`

	// no documentation yet
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}

// no documentation yet
type Container_Network_Port_Statistic struct {
	Entity

	// no documentation yet
	AdministrativeStatus *int `json:"administrativeStatus,omitempty" xmlrpc:"administrativeStatus,omitempty"`

	// no documentation yet
	InDiscardPackets *uint `json:"inDiscardPackets,omitempty" xmlrpc:"inDiscardPackets,omitempty"`

	// no documentation yet
	InErrorPackets *uint `json:"inErrorPackets,omitempty" xmlrpc:"inErrorPackets,omitempty"`

	// no documentation yet
	InOctets *uint `json:"inOctets,omitempty" xmlrpc:"inOctets,omitempty"`

	// no documentation yet
	InUnicastPackets *uint `json:"inUnicastPackets,omitempty" xmlrpc:"inUnicastPackets,omitempty"`

	// no documentation yet
	MaximumTransmissionUnit *uint `json:"maximumTransmissionUnit,omitempty" xmlrpc:"maximumTransmissionUnit,omitempty"`

	// no documentation yet
	OperationalStatus *int `json:"operationalStatus,omitempty" xmlrpc:"operationalStatus,omitempty"`

	// no documentation yet
	OutDiscardPackets *uint `json:"outDiscardPackets,omitempty" xmlrpc:"outDiscardPackets,omitempty"`

	// no documentation yet
	OutErrorPackets *uint `json:"outErrorPackets,omitempty" xmlrpc:"outErrorPackets,omitempty"`

	// no documentation yet
	OutOctets *uint `json:"outOctets,omitempty" xmlrpc:"outOctets,omitempty"`

	// no documentation yet
	OutUnicastPackets *uint `json:"outUnicastPackets,omitempty" xmlrpc:"outUnicastPackets,omitempty"`

	// no documentation yet
	PortDuplex *uint `json:"portDuplex,omitempty" xmlrpc:"portDuplex,omitempty"`

	// no documentation yet
	Speed *uint `json:"speed,omitempty" xmlrpc:"speed,omitempty"`
}

// no documentation yet
type Container_Network_Service_Resource_ObjectStorage_ConnectionInformation struct {
	Entity

	// no documentation yet
	Datacenter *string `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// no documentation yet
	DatacenterShortName *string `json:"datacenterShortName,omitempty" xmlrpc:"datacenterShortName,omitempty"`

	// no documentation yet
	PrivateEndpoint *string `json:"privateEndpoint,omitempty" xmlrpc:"privateEndpoint,omitempty"`

	// no documentation yet
	PublicEndpoint *string `json:"publicEndpoint,omitempty" xmlrpc:"publicEndpoint,omitempty"`
}

// no documentation yet
type Container_Network_Storage_Backup_Evault_WebCc_Authentication_Details struct {
	Entity

	// no documentation yet
	EventValidation *string `json:"eventValidation,omitempty" xmlrpc:"eventValidation,omitempty"`

	// no documentation yet
	ViewState *string `json:"viewState,omitempty" xmlrpc:"viewState,omitempty"`

	// no documentation yet
	WebCcUrl *string `json:"webCcUrl,omitempty" xmlrpc:"webCcUrl,omitempty"`
}

// SoftLayer's StorageLayer Evault services provides details regarding the the purchased vault.
//
// When a job is created using the Webcc Console, the job created is identified as a task on the vault. Using this service, information regarding the task can be retrieved.
//
//
type Container_Network_Storage_Evault_Vault_Task struct {
	Entity

	// Unique identifier for the task.
	Id *uint `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The hostname provided at time of agent registration.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// Total compressed bytes used for the task.
	UsedPoolsize *uint `json:"usedPoolsize,omitempty" xmlrpc:"usedPoolsize,omitempty"`
}

// The SoftLayer_Container_Network_Storage_Evault_WebCc_AgentStatus will contain the timestamp of the last backup performed by the EVault agent.  The agent status will also be returned.
type Container_Network_Storage_Evault_WebCc_AgentStatus struct {
	Entity

	// Timestamp of last backup performed by the EVault backup agent
	LastBackup *Time `json:"lastBackup,omitempty" xmlrpc:"lastBackup,omitempty"`

	// Status indicating the accumulative status result of all jobs performed by the evault agent.  For example, if one job out three jobs failed agent status will by "Failed Backup(s)".
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`
}

// The SoftLayer_Container_Network_Storage_Evault_WebCc_BackupResults will contain the timeframe of backups and the results will also be returned.
type Container_Network_Storage_Evault_WebCc_BackupResults struct {
	Entity

	// Timestamp of begin time
	BeginTime *Time `json:"beginTime,omitempty" xmlrpc:"beginTime,omitempty"`

	// Count of backups with conflicts.
	Conflict *string `json:"conflict,omitempty" xmlrpc:"conflict,omitempty"`

	// Timestamp of end time
	EndTime *Time `json:"endTime,omitempty" xmlrpc:"endTime,omitempty"`

	// Count of failed backups.
	Failed *string `json:"failed,omitempty" xmlrpc:"failed,omitempty"`

	// Count of successfull backups.
	Success *string `json:"success,omitempty" xmlrpc:"success,omitempty"`
}

// The SoftLayer_Container_Network_Storage_Evault_WebCc_JobDetails will contain basic details for all backup and restore jobs performed by the StorageLayer EVault service offering.
type Container_Network_Storage_Evault_WebCc_JobDetails struct {
	Entity

	// The number of bytes currently used by the backup job. (provided only for backup jobs)
	BytesUsed *uint `json:"bytesUsed,omitempty" xmlrpc:"bytesUsed,omitempty"`

	// Description of the backup/restore job
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// hardware id
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// Date of the last jobrun.
	LastRunDate *Time `json:"lastRunDate,omitempty" xmlrpc:"lastRunDate,omitempty"`

	// Name of the backup/restore job
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// Size of backup job when it was first run. (provided only for backup jobs)
	OriginalSize *uint `json:"originalSize,omitempty" xmlrpc:"originalSize,omitempty"`

	// Percentage of overall used space allocated by the job. (provided only for backup jobs)
	PercentageOfTotalUsage *int `json:"percentageOfTotalUsage,omitempty" xmlrpc:"percentageOfTotalUsage,omitempty"`

	// Result of the latest jobrun.
	Result *string `json:"result,omitempty" xmlrpc:"result,omitempty"`

	// virtual guest id
	VirtualGuestId *int `json:"virtualGuestId,omitempty" xmlrpc:"virtualGuestId,omitempty"`
}

// The SoftLayer_Container_Network_Storage_Host will contain the reference id field for the object associated with the host.  The host object type will also be returned.
type Container_Network_Storage_Host struct {
	Entity

	// Reference id field for object associated with host.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Type for the object associated with host
	ObjectType *string `json:"objectType,omitempty" xmlrpc:"objectType,omitempty"`
}

// SoftLayer_Container_Network_Storage_Hub_ObjectStorage_ContentDeliveryUrl provides specific details is a container which contains the cdn urls associated with an object storage account
type Container_Network_Storage_Hub_ObjectStorage_ContentDeliveryUrl struct {
	Entity

	// no documentation yet
	Datacenter *string `json:"datacenter,omitempty" xmlrpc:"datacenter,omitempty"`

	// no documentation yet
	FlashUrl *string `json:"flashUrl,omitempty" xmlrpc:"flashUrl,omitempty"`

	// no documentation yet
	HttpUrl *string `json:"httpUrl,omitempty" xmlrpc:"httpUrl,omitempty"`
}

// SoftLayer_Container_Network_Storage_Hub_ObjectStorage_Endpoint provides specific details on available endpoint URLs and locations.
type Container_Network_Storage_Hub_ObjectStorage_Endpoint struct {
	Entity

	// no documentation yet
	Location *string `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	Region *string `json:"region,omitempty" xmlrpc:"region,omitempty"`

	// no documentation yet
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// no documentation yet
	Url *string `json:"url,omitempty" xmlrpc:"url,omitempty"`
}

// SoftLayer_Container_Network_Storage_Hub_ObjectStorage_File provides specific details that only apply to files that are sent or received from CloudLayer storage resources.
type Container_Network_Storage_Hub_ObjectStorage_File struct {
	Container_Utility_File_Entity

	// no documentation yet
	Folder *string `json:"folder,omitempty" xmlrpc:"folder,omitempty"`

	// no documentation yet
	Hash *string `json:"hash,omitempty" xmlrpc:"hash,omitempty"`
}

// SoftLayer_Container_Network_Storage_Hub_Container provides details about containers which store collections of files.
type Container_Network_Storage_Hub_ObjectStorage_Folder struct {
	Entity

	// no documentation yet
	Bytes *uint `json:"bytes,omitempty" xmlrpc:"bytes,omitempty"`

	// no documentation yet
	Count *uint `json:"count,omitempty" xmlrpc:"count,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// SoftLayer_Container_Network_Storage_Hub_ObjectStorage_Node provides detailed information for a particular object storage node
type Container_Network_Storage_Hub_ObjectStorage_Node struct {
	Entity

	// no documentation yet
	DeviceName *string `json:"deviceName,omitempty" xmlrpc:"deviceName,omitempty"`

	// no documentation yet
	ResourceName *string `json:"resourceName,omitempty" xmlrpc:"resourceName,omitempty"`

	// no documentation yet
	UserAuthUrl *string `json:"userAuthUrl,omitempty" xmlrpc:"userAuthUrl,omitempty"`
}

// SoftLayer_Container_Network_Storage_Hub_ObjectStorage_Policy provides specific details on available storage policies.
type Container_Network_Storage_Hub_ObjectStorage_Policy struct {
	Entity

	// no documentation yet
	PolicyCode *string `json:"policyCode,omitempty" xmlrpc:"policyCode,omitempty"`
}

// no documentation yet
type Container_Network_Storage_NetworkConnectionInformation struct {
	Entity

	// no documentation yet
	Id *string `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// no documentation yet
	StorageType *string `json:"storageType,omitempty" xmlrpc:"storageType,omitempty"`
}

// SoftLayer_Container_Subnet_IPAddress models an IP v4 address as it exists as a member of it's subnet, letting the user know if it is a network identifier, gateway, broadcast, or useable address. Addresses that are neither the network identifier nor the gateway nor the broadcast addresses are usable by SoftLayer servers.
type Container_Network_Subnet_IpAddress struct {
	Entity

	// The hardware that an IP address is associated with.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// An IP address expressed in dotted-quad notation.
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// Whether an IP address is its subnet's broadcast address.
	IsBroadcastAddress *bool `json:"isBroadcastAddress,omitempty" xmlrpc:"isBroadcastAddress,omitempty"`

	// Whether an IP address is its subnet's gateway address. Gateway addresses exist on SoftLayer's routers and many not be assigned to servers.
	IsGatewayAddress *bool `json:"isGatewayAddress,omitempty" xmlrpc:"isGatewayAddress,omitempty"`

	// Whether an IP address is its subnet's network identifier address.
	IsNetworkAddress *bool `json:"isNetworkAddress,omitempty" xmlrpc:"isNetworkAddress,omitempty"`
}

// SoftLayer_Container_Network_Subnet_Registration_SubnetReference is provided to reference [[SoftLayer_Network_Subnet_Registration]] object and the [[SoftLayer_Network_Subnet]] it references, in CIDR form.
type Container_Network_Subnet_Registration_SubnetReference struct {
	Entity

	// The ID of the [[SoftLayer_Network_Subnet_Registration]] object.
	RegistrationId *int `json:"registrationId,omitempty" xmlrpc:"registrationId,omitempty"`

	// The subnet address in CIDR form.
	SubnetCidr *string `json:"subnetCidr,omitempty" xmlrpc:"subnetCidr,omitempty"`
}

// SoftLayer_Container_Subnet_Registration_TransactionDetails is provided to return details of a newly created Subnet Registration Transaction.
type Container_Network_Subnet_Registration_TransactionDetails struct {
	Entity

	// The IDs and Subnets of the [[SoftLayer_Network_Subnet_Registration]] object.
	SubnetReferences []Container_Network_Subnet_Registration_SubnetReference `json:"subnetReferences,omitempty" xmlrpc:"subnetReferences,omitempty"`

	// The ID of the Transaction object.
	TransactionId *int `json:"transactionId,omitempty" xmlrpc:"transactionId,omitempty"`
}

// no documentation yet
type Container_Notification_Mass_Filter_TemplateKey struct {
	Entity
}

// no documentation yet
type Container_Notification_Mass_Filter_TemplateValue struct {
	Entity
}

// Represents the acceptance status of a Policy.
type Container_Policy_Acceptance struct {
	Entity

	// Flag to indicate if a policy has been previously accepted.
	AcceptedFlag *bool `json:"acceptedFlag,omitempty" xmlrpc:"acceptedFlag,omitempty"`

	// Name of the policy for which we are representing it's acceptance status.
	PolicyName *string `json:"policyName,omitempty" xmlrpc:"policyName,omitempty"`

	// ID of the [[SoftLayer_Product_Item_Policy_Assignment]].
	ProductPolicyAssignmentId *int `json:"productPolicyAssignmentId,omitempty" xmlrpc:"productPolicyAssignmentId,omitempty"`
}

// The SoftLayer_Container_Product_Item_Category data type represents a single product item category.
type Container_Product_Item_Category struct {
	Entity

	// identifier for category.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`
}

// The SoftLayer_Container_Product_Item_Category_Question_Answer data type represents an answer to an item category question.  It contains the category, the question being answered, and the answer.
type Container_Product_Item_Category_Question_Answer struct {
	Entity

	// The answer to the question.
	Answer *string `json:"answer,omitempty" xmlrpc:"answer,omitempty"`

	// The product item category code.
	CategoryCode *string `json:"categoryCode,omitempty" xmlrpc:"categoryCode,omitempty"`

	// The product item category id.
	CategoryId *int `json:"categoryId,omitempty" xmlrpc:"categoryId,omitempty"`

	// The product item category question id.
	QuestionId *int `json:"questionId,omitempty" xmlrpc:"questionId,omitempty"`
}

// The SoftLayer_Container_Product_Item_Category_ZeroFee_Count data type represents a count of zero fee billing/invoice items.
type Container_Product_Item_Category_ZeroFee_Count struct {
	Entity

	// The product item category code.
	CategoryCode *string `json:"categoryCode,omitempty" xmlrpc:"categoryCode,omitempty"`

	// The product item category id.
	CategoryId *int `json:"categoryId,omitempty" xmlrpc:"categoryId,omitempty"`

	// The product item category name.
	CategoryName *string `json:"categoryName,omitempty" xmlrpc:"categoryName,omitempty"`

	// The count of zero fee items for this category.
	Count *int `json:"count,omitempty" xmlrpc:"count,omitempty"`
}

// The SoftLayer_Container_Product_Item_Discount_Program data type represents the information about a discount that is related to a specific product item.
type Container_Product_Item_Discount_Program struct {
	Entity

	// The number of times the item discount(s) may be applied for that order container.  At a minimum the number will be 1 and at most, it will match the quantity of the order container.
	ApplicableQuantity *int `json:"applicableQuantity,omitempty" xmlrpc:"applicableQuantity,omitempty"`

	// The product item that the discount applies to.
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// The sum of the one time fees (one time, setup and labor) of the prices of this container multiplied by the applicable quantity of this container.
	OneTimeAmount *Float64 `json:"oneTimeAmount,omitempty" xmlrpc:"oneTimeAmount,omitempty"`

	// The tax amount on the one time fees (one time, setup and labor) of the prices of this container mulitiplied by the applicable quantity of this container.
	OneTimeTax *Float64 `json:"oneTimeTax,omitempty" xmlrpc:"oneTimeTax,omitempty"`

	// The item prices that contain the amount of the discount in the recurringFee field.  There may be one or more prices.
	Prices []Product_Item_Price `json:"prices,omitempty" xmlrpc:"prices,omitempty"`

	// The sum of the one time fees (one time, setup and labor) of the prices of this container multiplied by the applicable quantity of this container with the proration factor applied.
	ProratedOneTimeAmount *Float64 `json:"proratedOneTimeAmount,omitempty" xmlrpc:"proratedOneTimeAmount,omitempty"`

	// The tax amount on the one time fees (one time, setup and labor) of the prices of this container mulitiplied by the applicable quantity of this container with the proration factor applied.
	ProratedOneTimeTax *Float64 `json:"proratedOneTimeTax,omitempty" xmlrpc:"proratedOneTimeTax,omitempty"`

	// The sum of the recurring fees of the prices of this container multiplied by the applicable quantity of this container with the proration factor applied.
	ProratedRecurringAmount *Float64 `json:"proratedRecurringAmount,omitempty" xmlrpc:"proratedRecurringAmount,omitempty"`

	// The tax amount on the recurring fees of the prices of this container mulitiplied by the applicable quantity of this container with the proration factor applied.
	ProratedRecurringTax *Float64 `json:"proratedRecurringTax,omitempty" xmlrpc:"proratedRecurringTax,omitempty"`

	// The sum of the recurring fees of the prices of this container multiplied by the applicable quantity of this container.
	RecurringAmount *Float64 `json:"recurringAmount,omitempty" xmlrpc:"recurringAmount,omitempty"`

	// The tax amount on the recurring fees of the prices of this container mulitiplied by the applicable quantity of this container.
	RecurringTax *Float64 `json:"recurringTax,omitempty" xmlrpc:"recurringTax,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order with SoftLayer.
type Container_Product_Order struct {
	Entity

	// Flag for identifying an order for Big Data Deployment.
	BigDataOrderFlag *bool `json:"bigDataOrderFlag,omitempty" xmlrpc:"bigDataOrderFlag,omitempty"`

	// Billing Information associated with an order. For existing customers this information is completely ignored. Do not send this information for existing customers.
	BillingInformation *Container_Product_Order_Billing_Information `json:"billingInformation,omitempty" xmlrpc:"billingInformation,omitempty"`

	// This is the ID of the [[SoftLayer_Billing_Order_Item]] of this configuration/container. It is used for rebuilding an order container from a quote and is set automatically.
	BillingOrderItemId *int `json:"billingOrderItemId,omitempty" xmlrpc:"billingOrderItemId,omitempty"`

	// The URL to which PayPal redirects browser after checkout has been canceled before completion of a payment.
	CancelUrl *string `json:"cancelUrl,omitempty" xmlrpc:"cancelUrl,omitempty"`

	// Added by Gopherlayer. This hints to the API what kind of product order this is.
	ComplexType *string `json:"complexType,omitempty" xmlrpc:"complexType,omitempty"`

	// User-specified description to identify a particular order container. This is useful if you have a multi-configuration order (multiple <code>orderContainers</code>) and you want to be able to easily determine one from another. Populating this value may be helpful if an exception is thrown when placing an order and it's tied to a specific order container.
	ContainerIdentifier *string `json:"containerIdentifier,omitempty" xmlrpc:"containerIdentifier,omitempty"`

	// This hash is internally-generated and is used to for tracking order containers.
	ContainerSplHash *string `json:"containerSplHash,omitempty" xmlrpc:"containerSplHash,omitempty"`

	// The currency type chosen at checkout.
	CurrencyShortName *string `json:"currencyShortName,omitempty" xmlrpc:"currencyShortName,omitempty"`

	// Device Fingerprint Identifier - Optional.
	DeviceFingerprintId *string `json:"deviceFingerprintId,omitempty" xmlrpc:"deviceFingerprintId,omitempty"`

	// This is the configuration identifier for tracking orders on the HTML order forms.
	DisplayLayerSessionId *string `json:"displayLayerSessionId,omitempty" xmlrpc:"displayLayerSessionId,omitempty"`

	// no documentation yet
	ExtendedHardwareTesting *bool `json:"extendedHardwareTesting,omitempty" xmlrpc:"extendedHardwareTesting,omitempty"`

	// The [[SoftLayer_Product_Item_Price]] for the Flexible Credit Program discount.  The <code>oneTimeFee</code> field contains the calculated discount being applied to the order.
	FlexibleCreditProgramPrice *Product_Item_Price `json:"flexibleCreditProgramPrice,omitempty" xmlrpc:"flexibleCreditProgramPrice,omitempty"`

	// For orders that contain servers (bare metal, virtual server, big data, etc.), the hardware property is required. This property is an array of [[SoftLayer_Hardware]] objects. The <code>hostname</code> and <code>domain</code> properties are required for each hardware object. Note that virtual server ([[SoftLayer_Container_Product_Order_Virtual_Guest]]) orders may populate this field instead of the <code>virtualGuests</code> property.
	Hardware []Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// An optional virtual disk image template identifier to be used as an installation base for a computing instance order
	ImageTemplateGlobalIdentifier *string `json:"imageTemplateGlobalIdentifier,omitempty" xmlrpc:"imageTemplateGlobalIdentifier,omitempty"`

	// An optional virtual disk image template identifier to be used as an installation base for a computing instance order
	ImageTemplateId *int `json:"imageTemplateId,omitempty" xmlrpc:"imageTemplateId,omitempty"`

	// Flag to identify a "managed" order. This value is set internally.
	IsManagedOrder *int `json:"isManagedOrder,omitempty" xmlrpc:"isManagedOrder,omitempty"`

	// The collection of [[SoftLayer_Container_Product_Item_Category_Question_Answer]] for any product category that has additional questions requiring user input.
	ItemCategoryQuestionAnswers []Container_Product_Item_Category_Question_Answer `json:"itemCategoryQuestionAnswers,omitempty" xmlrpc:"itemCategoryQuestionAnswers,omitempty"`

	// The [[SoftLayer_Location_Region]] keyname or specific [[SoftLayer_Location_Datacenter]] id where the order should be provisioned. If this value is provided and the <code>regionalGroup</code> property is also specified, an exception will be thrown indicating that only 1 is allowed.
	Location *string `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// This [[SoftLayer_Location]] object will be determined from the <code>location</code> property and will be returned in the order verification or placement response. Any value specified here will get overwritten by the verification process.
	LocationObject *Location `json:"locationObject,omitempty" xmlrpc:"locationObject,omitempty"`

	// A generic message about the order. Does not need to be sent in with any orders.
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// Orders may contain an array of configurations. Populating this property allows you to purchase multiple configurations within a single order. Each order container will have its own individual settings independent of the other order containers. For example, it is possible to order a bare metal server in one configuration and a virtual server in another.
	//
	// If <code>orderContainers</code> is populated on the base order container, most of the configuration-specific properties are ignored on the base container. For example, <code>prices</code>, <code>location</code> and <code>packageId</code> will be ignored on the base container, but since the <code>billingInformation</code> is a property that's not specific to a single order container (but the order as a whole) it must be populated on the base container.
	OrderContainers []Container_Product_Order `json:"orderContainers,omitempty" xmlrpc:"orderContainers,omitempty"`

	// This is deprecated and does not do anything.
	OrderHostnames []string `json:"orderHostnames,omitempty" xmlrpc:"orderHostnames,omitempty"`

	// Collection of exceptions resulting from the verification of the order. This value is set internally and is not required for end users when placing an order. When placing API orders, users can use this value to determine the container-specific exception that was thrown.
	OrderVerificationExceptions []Container_Exception `json:"orderVerificationExceptions,omitempty" xmlrpc:"orderVerificationExceptions,omitempty"`

	// The [[SoftLayer_Product_Package]] id for an order container. This is required to place an order.
	PackageId *int `json:"packageId,omitempty" xmlrpc:"packageId,omitempty"`

	// The Payment Type is Optional. If nothing is sent in, then the normal method of payment will be used. For paypal customers, this means a paypalToken will be returned in the receipt. This token is to be used on the paypal website to complete the order. For Credit Card customers, the card on file in our system will be used to make an initial authorization. To force the order to use a payment type, use one of the following: CARD_ON_FILE or PAYPAL
	PaymentType *string `json:"paymentType,omitempty" xmlrpc:"paymentType,omitempty"`

	// The post-tax recurring charge for the order. This is the sum of preTaxRecurring + totalRecurringTax.
	PostTaxRecurring *Float64 `json:"postTaxRecurring,omitempty" xmlrpc:"postTaxRecurring,omitempty"`

	// The post-tax recurring hourly charge for the order. Since taxes are not calculated for hourly orders, this value will be the same as preTaxRecurringHourly.
	PostTaxRecurringHourly *Float64 `json:"postTaxRecurringHourly,omitempty" xmlrpc:"postTaxRecurringHourly,omitempty"`

	// The post-tax recurring monthly charge for the order. This is the sum of preTaxRecurringMonthly + totalRecurringTax.
	PostTaxRecurringMonthly *Float64 `json:"postTaxRecurringMonthly,omitempty" xmlrpc:"postTaxRecurringMonthly,omitempty"`

	// The post-tax setup fees of the order. This is the sum of preTaxSetup + totalSetupTax;
	PostTaxSetup *Float64 `json:"postTaxSetup,omitempty" xmlrpc:"postTaxSetup,omitempty"`

	// The pre-tax recurring total of the order. If there are mixed monthly and hourly prices on the order, this will be the sum of preTaxRecurringHourly and preTaxRecurringMonthly.
	PreTaxRecurring *Float64 `json:"preTaxRecurring,omitempty" xmlrpc:"preTaxRecurring,omitempty"`

	// The pre-tax hourly recurring total of the order. If there are only monthly prices on the order, this value will be 0.
	PreTaxRecurringHourly *Float64 `json:"preTaxRecurringHourly,omitempty" xmlrpc:"preTaxRecurringHourly,omitempty"`

	// The pre-tax monthly recurring total of the order. If there are only hourly prices on the order, this value will be 0.
	PreTaxRecurringMonthly *Float64 `json:"preTaxRecurringMonthly,omitempty" xmlrpc:"preTaxRecurringMonthly,omitempty"`

	// The pre-tax setup fee total of the order.
	PreTaxSetup *Float64 `json:"preTaxSetup,omitempty" xmlrpc:"preTaxSetup,omitempty"`

	// If there are any presale events available for an order, this value will be populated. It is set internally and is not required for end users when placing an order. See [[SoftLayer_Sales_Presale_Event]] for more info.
	PresaleEvent *Sales_Presale_Event `json:"presaleEvent,omitempty" xmlrpc:"presaleEvent,omitempty"`

	// A preset configuration id for the package. Is required if not submitting any prices.
	PresetId *int `json:"presetId,omitempty" xmlrpc:"presetId,omitempty"`

	// This is a collection of [[SoftLayer_Product_Item_Price]] objects. The only required property to populate for an item price object when ordering is its <code>id</code> - all other supplied information about the price (e.g., recurringFee, setupFee, etc.) will be ignored. Unless the [[SoftLayer_Product_Package]] associated with the order allows for preset prices, this property is required to place an order.
	Prices []Product_Item_Price `json:"prices,omitempty" xmlrpc:"prices,omitempty"`

	// The id of a [[SoftLayer_Hardware_Component_Partition_Template]]. This property is optional. If no partition template is provided, a default will be used according to the operating system chosen with the order. Using the [[SoftLayer_Hardware_Component_Partition_OperatingSystem]] service, getPartitionTemplates will return those available for the particular operating system.
	PrimaryDiskPartitionId *int `json:"primaryDiskPartitionId,omitempty" xmlrpc:"primaryDiskPartitionId,omitempty"`

	// Priorities to set on replication set servers.
	Priorities []string `json:"priorities,omitempty" xmlrpc:"priorities,omitempty"`

	// Flag for identifying a container as Virtual Server (Private Node).
	PrivateCloudOrderFlag *bool `json:"privateCloudOrderFlag,omitempty" xmlrpc:"privateCloudOrderFlag,omitempty"`

	// Type of Virtual Server (Private Node) order. Potential values: INITIAL, ADDHOST, ADDIPS, ADDZONE
	PrivateCloudOrderType *string `json:"privateCloudOrderType,omitempty" xmlrpc:"privateCloudOrderType,omitempty"`

	// Optional promotion code for an order.
	PromotionCode *string `json:"promotionCode,omitempty" xmlrpc:"promotionCode,omitempty"`

	// Generic properties.
	Properties []Container_Product_Order_Property `json:"properties,omitempty" xmlrpc:"properties,omitempty"`

	// The Prorated Initial Charge plus the balance on the account. Only the recurring fees are prorated. Here's how the calculation works: We take the postTaxRecurring value and we prorate it based on the time between now and the next bill date for this account. After this, we add in the setup fee since this is not prorated. Then, if there is a balance on the account, we add that to the account. In the event that there is a credit balance on the account, we will subtract this amount from the order total. If the credit balance on the account is greater than the prorated initial charge, the order will go through without a charge to the credit card on the account or requiring a paypal payment. The credit on the account will be reduced by the order total, and the order will await approval from sales, as normal. If there is a pending order already in the system, We will ignore the balance on the account completely, in the calculation of the initial charge. This is to protect against two orders coming into the system and getting the benefit of a credit balance, or worse, both orders being charged the order amount + the balance on the account.
	ProratedInitialCharge *Float64 `json:"proratedInitialCharge,omitempty" xmlrpc:"proratedInitialCharge,omitempty"`

	// This is the same as the proratedInitialCharge, except the balance on the account is ignored. This is the prorated total amount of the order.
	ProratedOrderTotal *Float64 `json:"proratedOrderTotal,omitempty" xmlrpc:"proratedOrderTotal,omitempty"`

	// The URLs for scripts to execute on their respective servers after they have been provisioned. Provision scripts are not available for Microsoft Windows servers.
	ProvisionScripts []string `json:"provisionScripts,omitempty" xmlrpc:"provisionScripts,omitempty"`

	// The quantity of the item being ordered
	Quantity *int `json:"quantity,omitempty" xmlrpc:"quantity,omitempty"`

	// A custom name to be assigned to the quote.
	QuoteName *string `json:"quoteName,omitempty" xmlrpc:"quoteName,omitempty"`

	// Specifying a regional group name allows you to not worry about placing your server or service at a specific datacenter, but to any datacenter within that regional group. See [[SoftLayer_Location_Group_Regional]] to get a list of available regional group names.
	//
	// <code>location</code> and <code>regionalGroup</code> are mutually exclusive on an order container. If both location and regionalGroup are provided, an exception will be thrown indicating that only 1 is allowed.
	//
	// If a regional group is provided and VLANs are specified (within the <code>hardware</code> or <code>virtualGuests</code> properties), we will use the datacenter where the VLANs are located. If no VLANs are specified, we will use the preferred datacenter on the regional group object.
	RegionalGroup *string `json:"regionalGroup,omitempty" xmlrpc:"regionalGroup,omitempty"`

	// An optional resource group identifier specifying the resource group to attach the order to
	ResourceGroupId *int `json:"resourceGroupId,omitempty" xmlrpc:"resourceGroupId,omitempty"`

	// This variable specifies the name of the resource group the server configuration belongs to. For MongoDB Replica sets, it would be the replica set name.
	ResourceGroupName *string `json:"resourceGroupName,omitempty" xmlrpc:"resourceGroupName,omitempty"`

	// An optional resource group template identifier to be used as a deployment base for a Virtual Server (Private Node) order.
	ResourceGroupTemplateId *int `json:"resourceGroupTemplateId,omitempty" xmlrpc:"resourceGroupTemplateId,omitempty"`

	// The URL to which PayPal redirects browser after a payment is completed.
	ReturnUrl *string `json:"returnUrl,omitempty" xmlrpc:"returnUrl,omitempty"`

	// This flag indicates that the quote should be sent to the email address associated with the account or order.
	SendQuoteEmailFlag *bool `json:"sendQuoteEmailFlag,omitempty" xmlrpc:"sendQuoteEmailFlag,omitempty"`

	// The number of cores for the server being ordered. This value is set internally.
	ServerCoreCount *int `json:"serverCoreCount,omitempty" xmlrpc:"serverCoreCount,omitempty"`

	// An optional computing instance identifier to be used as an installation base for a computing instance order
	SourceVirtualGuestId *int `json:"sourceVirtualGuestId,omitempty" xmlrpc:"sourceVirtualGuestId,omitempty"`

	// The containers which hold SoftLayer_Security_Ssh_Key IDs to add to their respective servers. The order of containers passed in needs to match the order they are assigned to either hardware or virtualGuests. SSH Keys will not be assigned for servers with Microsoft Windows.
	SshKeys []Container_Product_Order_SshKeys `json:"sshKeys,omitempty" xmlrpc:"sshKeys,omitempty"`

	// An optional parameter for step-based order processing.
	StepId *int `json:"stepId,omitempty" xmlrpc:"stepId,omitempty"`

	//
	//
	// For orders that want to add storage groups such as RAID across multiple disks, simply add [[SoftLayer_Container_Product_Order_Storage_Group]] objects to this array. Storage groups will only be used if the 'RAID' disk controller price is selected. Any other disk controller types will ignore the storage groups set here.
	//
	// The first storage group in this array will be considered the primary storage group, which is used for the OS. Any other storage groups will act as data storage.
	//
	//
	StorageGroups []Container_Product_Order_Storage_Group `json:"storageGroups,omitempty" xmlrpc:"storageGroups,omitempty"`

	// The order container may not contain the final tax rates when it is returned from [[SoftLayer_Product_Order/verifyOrder|verifyOrder]]. This hash will facilitate checking if the tax rates have finished being calculated and retrieving the accurate tax rate values.
	TaxCacheHash *string `json:"taxCacheHash,omitempty" xmlrpc:"taxCacheHash,omitempty"`

	// Flag to indicate if the order container has the final tax rates for the order. Some tax rates are calculated in the background because they take longer, and they might not be finished when the container is returned from [[SoftLayer_Product_Order/verifyOrder|verifyOrder]].
	TaxCompletedFlag *bool `json:"taxCompletedFlag,omitempty" xmlrpc:"taxCompletedFlag,omitempty"`

	// The SoftLayer_Product_Item_Price for the Tech Incubator discount.  The oneTimeFee field contain the calculated discount being applied to the order.
	TechIncubatorItemPrice *Product_Item_Price `json:"techIncubatorItemPrice,omitempty" xmlrpc:"techIncubatorItemPrice,omitempty"`

	// The total tax portion of the recurring fees.
	TotalRecurringTax *Float64 `json:"totalRecurringTax,omitempty" xmlrpc:"totalRecurringTax,omitempty"`

	// The tax amount of the setup fees.
	TotalSetupTax *Float64 `json:"totalSetupTax,omitempty" xmlrpc:"totalSetupTax,omitempty"`

	// An optional flag to use hourly pricing instead of standard monthly pricing.
	UseHourlyPricing *bool `json:"useHourlyPricing,omitempty" xmlrpc:"useHourlyPricing,omitempty"`

	// For virtual guest (virtual server) orders, this property is required if you did not specify data in the <code>hardware</code> property. This is an array of [[SoftLayer_Virtual_Guest]] objects. The <code>hostname</code> and <code>domain</code> properties are required for each virtual guest object. There is no need to specify data in this property and the <code>hardware</code> property - only one is required for virtual server orders.
	VirtualGuests []Virtual_Guest `json:"virtualGuests,omitempty" xmlrpc:"virtualGuests,omitempty"`
}

// This datatype is to be used for data transfer requests.
type Container_Product_Order_Account_Media_Data_Transfer_Request struct {
	Container_Product_Order

	// An instance of [[SoftLayer_Account_Media_Data_Transfer_Request]]
	Request *Account_Media_Data_Transfer_Request `json:"request,omitempty" xmlrpc:"request,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. The SoftLayer_Container_Product_Order_Attribute_Address datatype contains the address information.
type Container_Product_Order_Attribute_Address struct {
	Entity

	// The physical street address.
	AddressLine1 *string `json:"addressLine1,omitempty" xmlrpc:"addressLine1,omitempty"`

	// The second line in the address. Information such as suite number goes here.
	AddressLine2 *string `json:"addressLine2,omitempty" xmlrpc:"addressLine2,omitempty"`

	// The city name
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// The 2-character Country code. (i.e. US)
	CountryCode *string `json:"countryCode,omitempty" xmlrpc:"countryCode,omitempty"`

	// The non US/Canadian state or region.
	NonUsState *string `json:"nonUsState,omitempty" xmlrpc:"nonUsState,omitempty"`

	// The Zip or Postal Code.
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// The state or region.
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. The SoftLayer_Container_Product_Order_Attribute_Contact datatype contains the contact information.
type Container_Product_Order_Attribute_Contact struct {
	Entity

	// The address information of the contact.
	Address *Container_Product_Order_Attribute_Address `json:"address,omitempty" xmlrpc:"address,omitempty"`

	// The email address of the contact.
	EmailAddress *string `json:"emailAddress,omitempty" xmlrpc:"emailAddress,omitempty"`

	// The fax number associated with a contact. This is an optional value.
	FaxNumber *string `json:"faxNumber,omitempty" xmlrpc:"faxNumber,omitempty"`

	// The first name of the contact.
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// The last name of the contact.
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// The organization name of the contact.
	OrganizationName *string `json:"organizationName,omitempty" xmlrpc:"organizationName,omitempty"`

	// The phone number associated with a contact.
	PhoneNumber *string `json:"phoneNumber,omitempty" xmlrpc:"phoneNumber,omitempty"`

	// The title of the contact.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. The SoftLayer_Container_Product_Order_Attribute_Organization datatype contains the organization information.
type Container_Product_Order_Attribute_Organization struct {
	Entity

	// The address information of the contact.
	Address *Container_Product_Order_Attribute_Address `json:"address,omitempty" xmlrpc:"address,omitempty"`

	// The fax number associated with an organization. This is an optional value.
	FaxNumber *string `json:"faxNumber,omitempty" xmlrpc:"faxNumber,omitempty"`

	// The name of an organization.
	OrganizationName *string `json:"organizationName,omitempty" xmlrpc:"organizationName,omitempty"`

	// The phone number associated with an organization.
	PhoneNumber *string `json:"phoneNumber,omitempty" xmlrpc:"phoneNumber,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order with SoftLayer.
type Container_Product_Order_Billing_Information struct {
	Entity

	// The physical street address. Reserve information such as "apartment #123" or "Suite 2" for line 1.
	BillingAddressLine1 *string `json:"billingAddressLine1,omitempty" xmlrpc:"billingAddressLine1,omitempty"`

	// The second line in the address. Information such as suite number goes here.
	BillingAddressLine2 *string `json:"billingAddressLine2,omitempty" xmlrpc:"billingAddressLine2,omitempty"`

	// The city in which a customer's account resides.
	BillingCity *string `json:"billingCity,omitempty" xmlrpc:"billingCity,omitempty"`

	// The 2-character Country code for an account's address. (i.e. US)
	BillingCountryCode *string `json:"billingCountryCode,omitempty" xmlrpc:"billingCountryCode,omitempty"`

	// The email address associated with a customer account.
	BillingEmail *string `json:"billingEmail,omitempty" xmlrpc:"billingEmail,omitempty"`

	// the company name for an account.
	BillingNameCompany *string `json:"billingNameCompany,omitempty" xmlrpc:"billingNameCompany,omitempty"`

	// The first name of the customer account owner.
	BillingNameFirst *string `json:"billingNameFirst,omitempty" xmlrpc:"billingNameFirst,omitempty"`

	// The last name of the customer account owner
	BillingNameLast *string `json:"billingNameLast,omitempty" xmlrpc:"billingNameLast,omitempty"`

	// The fax number associated with a customer account.
	BillingPhoneFax *string `json:"billingPhoneFax,omitempty" xmlrpc:"billingPhoneFax,omitempty"`

	// The phone number associated with a customer account.
	BillingPhoneVoice *string `json:"billingPhoneVoice,omitempty" xmlrpc:"billingPhoneVoice,omitempty"`

	// The Zip or Postal Code for the billing address on an account.
	BillingPostalCode *string `json:"billingPostalCode,omitempty" xmlrpc:"billingPostalCode,omitempty"`

	// The State for the account.
	BillingState *string `json:"billingState,omitempty" xmlrpc:"billingState,omitempty"`

	// The credit card number to use.
	CardAccountNumber *string `json:"cardAccountNumber,omitempty" xmlrpc:"cardAccountNumber,omitempty"`

	// The payment card expiration month
	CardExpirationMonth *int `json:"cardExpirationMonth,omitempty" xmlrpc:"cardExpirationMonth,omitempty"`

	// The payment card expiration year
	CardExpirationYear *int `json:"cardExpirationYear,omitempty" xmlrpc:"cardExpirationYear,omitempty"`

	// The Card Verification Value Code (CVV) number
	CreditCardVerificationNumber *string `json:"creditCardVerificationNumber,omitempty" xmlrpc:"creditCardVerificationNumber,omitempty"`

	// Tax exempt status. 1 = exempt (not taxable),  0 = not exempt (taxable)
	TaxExempt *int `json:"taxExempt,omitempty" xmlrpc:"taxExempt,omitempty"`

	// The VAT ID entered at checkout
	VatId *string `json:"vatId,omitempty" xmlrpc:"vatId,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. The SoftLayer_Container_Product_Order_Dns_Domain_Registration datatype contains everything required to place a domain registration order with SoftLayer.
type Container_Product_Order_Dns_Domain_Registration struct {
	Container_Product_Order

	// Administrative contact information associated with an registraton or transfer. This is required if registration type is 'new' or 'transfer'.
	AdministrativeContact *Container_Dns_Domain_Registration_Contact `json:"administrativeContact,omitempty" xmlrpc:"administrativeContact,omitempty"`

	// Billing contact information associated with an registraton or transfer. This is required if registration type is 'new' or 'transfer'.
	BillingContact *Container_Dns_Domain_Registration_Contact `json:"billingContact,omitempty" xmlrpc:"billingContact,omitempty"`

	// The list of domains to be registered. This is required if registration type is 'new', 'renew', or 'transfer'.
	DomainRegistrationList []Container_Dns_Domain_Registration_List `json:"domainRegistrationList,omitempty" xmlrpc:"domainRegistrationList,omitempty"`

	// Owner contact information associated with an registraton or transfer. This is required if registration type is 'new' or 'transfer'.
	OwnerContact *Container_Dns_Domain_Registration_Contact `json:"ownerContact,omitempty" xmlrpc:"ownerContact,omitempty"`

	// The type of a domain registration order. The registration type is Required. Allowed values are new, transfer, and renew
	RegistrationType *string `json:"registrationType,omitempty" xmlrpc:"registrationType,omitempty"`

	// Technical contact information associated with an registraton or transfer. This is required if registration type is 'new' or 'transfer'.
	TechnicalContact *Container_Dns_Domain_Registration_Contact `json:"technicalContact,omitempty" xmlrpc:"technicalContact,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. The SoftLayer_Container_Product_Order_Dns_Domain_Reseller datatype contains everything required to place a domain reseller credit order with SoftLayer.
type Container_Product_Order_Dns_Domain_Reseller struct {
	Container_Product_Order

	// Amount to be credited to the domain reseller account.
	CreditAmount *Float64 `json:"creditAmount,omitempty" xmlrpc:"creditAmount,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a Gateway Appliance Cluster order with SoftLayer.
type Container_Product_Order_Gateway_Appliance_Cluster struct {
	Container_Product_Order

	// Used to identify which items on an order belong in the same cluster.
	ClusterIdentifier *string `json:"clusterIdentifier,omitempty" xmlrpc:"clusterIdentifier,omitempty"`

	// Indicates what type of cluster order is being placed (HA, Provision).
	ClusterOrderType *string `json:"clusterOrderType,omitempty" xmlrpc:"clusterOrderType,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a hardware security module order with SoftLayer.
type Container_Product_Order_Hardware_Security_Module struct {
	Container_Product_Order_Hardware_Server
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order with SoftLayer.
type Container_Product_Order_Hardware_Server struct {
	Container_Product_Order

	// Used to identify which items on an order belong in the same cluster.
	ClusterIdentifier *string `json:"clusterIdentifier,omitempty" xmlrpc:"clusterIdentifier,omitempty"`

	// Indicates what type of cluster order is being placed (HA, Provision).
	ClusterOrderType *string `json:"clusterOrderType,omitempty" xmlrpc:"clusterOrderType,omitempty"`

	// Used to identify which gateway is being upgraded to HA.
	ClusterResourceId *int `json:"clusterResourceId,omitempty" xmlrpc:"clusterResourceId,omitempty"`

	// Id of the [[SoftLayer_Monitoring_Agent_Configuration_Template_Group]] to be used with the monitoring package
	MonitoringAgentConfigurationTemplateGroupId *int `json:"monitoringAgentConfigurationTemplateGroupId,omitempty" xmlrpc:"monitoringAgentConfigurationTemplateGroupId,omitempty"`

	// When ordering Virtual Server (Private Node), this variable specifies the role of the server configuration. (Deprecated)
	PrivateCloudServerRole *string `json:"privateCloudServerRole,omitempty" xmlrpc:"privateCloudServerRole,omitempty"`

	// Used to identify which device the new server should be attached to.
	RequiredUpstreamDeviceId *int `json:"requiredUpstreamDeviceId,omitempty" xmlrpc:"requiredUpstreamDeviceId,omitempty"`

	// tags (used in MongoDB deployments). (Deprecated)
	Tags []Container_Product_Order_Property `json:"tags,omitempty" xmlrpc:"tags,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order with SoftLayer.
type Container_Product_Order_Hardware_Server_Colocation struct {
	Container_Product_Order_Hardware_Server
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a Gateway Appliance order.
type Container_Product_Order_Hardware_Server_Gateway_Appliance struct {
	Container_Product_Order_Hardware_Server
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order with SoftLayer.
type Container_Product_Order_Hardware_Server_Upgrade struct {
	Container_Product_Order_Hardware_Server
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a Monitoring Package order with SoftLayer.
type Container_Product_Order_Monitoring_Package struct {
	Container_Product_Order

	// no documentation yet
	ConfigurationTemplateGroups []Monitoring_Agent_Configuration_Template_Group `json:"configurationTemplateGroups,omitempty" xmlrpc:"configurationTemplateGroups,omitempty"`

	// no documentation yet
	ServerType *string `json:"serverType,omitempty" xmlrpc:"serverType,omitempty"`
}

// This is a datatype used with multi-configuration deployments. Multi-configuration deployments also have a deployment specific datatype that should be used in lieu of this one.
type Container_Product_Order_MultiConfiguration struct {
	Container_Product_Order
}

// no documentation yet
type Container_Product_Order_MultiConfiguration_Tornado struct {
	Container_Product_Order_MultiConfiguration
}

// This type contains the structure of network-related objects that may be specified when ordering services.
type Container_Product_Order_Network struct {
	Entity

	// The [[SoftLayer_Network]] object.
	Network *Network `json:"network,omitempty" xmlrpc:"network,omitempty"`

	// The list of public [[SoftLayer_Container_Product_Order_Network_Vlan|vlans]] available for ordering. Each VLAN will have list of public subnets that are accessible to the VLAN.
	PublicVlans []Container_Product_Order `json:"publicVlans,omitempty" xmlrpc:"publicVlans,omitempty"`

	// The list of private [[SoftLayer_Container_Product_Order_Network_Subnet|subnets]] available for ordering with a description of their available IP space.
	Subnets []Container_Product_Order `json:"subnets,omitempty" xmlrpc:"subnets,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an application delivery controller order with SoftLayer.
type Container_Product_Order_Network_Application_Delivery_Controller struct {
	Container_Product_Order

	// An optional [[SoftLayer_Network_Application_Delivery_Controller]] identifier that is used for upgrading an existing application delivery controller.
	ApplicationDeliveryControllerId *int `json:"applicationDeliveryControllerId,omitempty" xmlrpc:"applicationDeliveryControllerId,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a CDN order with SoftLayer.
type Container_Product_Order_Network_ContentDelivery_Account struct {
	Container_Product_Order

	// The CDN account name
	CdnAccountName *string `json:"cdnAccountName,omitempty" xmlrpc:"cdnAccountName,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a CDN order with SoftLayer.
type Container_Product_Order_Network_ContentDelivery_Account_Upgrade struct {
	Container_Product_Order

	// ID of an existing CDN account. You can use this to upgrade an existing CDN account.
	CdnAccountId *string `json:"cdnAccountId,omitempty" xmlrpc:"cdnAccountId,omitempty"`
}

// This is the default container type for network load balancer orders.
type Container_Product_Order_Network_LoadBalancer struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a global load balancer order with SoftLayer.
type Container_Product_Order_Network_LoadBalancer_Global struct {
	Container_Product_Order

	// The domain name that will be load balanced.
	Domain *string `json:"domain,omitempty" xmlrpc:"domain,omitempty"`

	// The hostname that will be load balanced.
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a network message delivery order with SoftLayer.
type Container_Product_Order_Network_Message_Delivery struct {
	Container_Product_Order

	// The account password for SendGrid enrollment.
	AccountPassword *string `json:"accountPassword,omitempty" xmlrpc:"accountPassword,omitempty"`

	// The username for SendGrid enrollment.
	AccountUsername *string `json:"accountUsername,omitempty" xmlrpc:"accountUsername,omitempty"`

	// The email address for SendGrid enrollment.
	EmailAddress *string `json:"emailAddress,omitempty" xmlrpc:"emailAddress,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a Message Queue order with SoftLayer.
type Container_Product_Order_Network_Message_Queue struct {
	Container_Product_Order
}

// This is the base data type for Performance storage order containers. If you wish to place an order you must not use this class and instead use the appropriate child container for the type of storage you would like to order: [[SoftLayer_Container_Product_Order_Network_PerformanceStorage_Nfs]] for File and [[SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi]] for Block storage.
type Container_Product_Order_Network_PerformanceStorage struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order for iSCSI (Block) Performance Storage
type Container_Product_Order_Network_PerformanceStorage_Iscsi struct {
	Container_Product_Order_Network_PerformanceStorage

	// OS Type to be used when formatting the storage space, this should match the OS type that will be connecting to the LUN. The only required property its the keyName of the OS type.
	OsFormatType *Network_Storage_Iscsi_OS_Type `json:"osFormatType,omitempty" xmlrpc:"osFormatType,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order for NFS (File) Performance Storage
type Container_Product_Order_Network_PerformanceStorage_Nfs struct {
	Container_Product_Order_Network_PerformanceStorage
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a hardware firewall order with SoftLayer.
type Container_Product_Order_Network_Protection_Firewall struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a hardware (dedicated) firewall order with SoftLayer.
type Container_Product_Order_Network_Protection_Firewall_Dedicated struct {
	Container_Product_Order

	// generic properties.
	Vlan *Network_Vlan `json:"vlan,omitempty" xmlrpc:"vlan,omitempty"`

	// generic properties.
	VlanId *int `json:"vlanId,omitempty" xmlrpc:"vlanId,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order for additional Evault plugins.
type Container_Product_Order_Network_Storage_Backup_Evault_Plugin struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an Evault order with SoftLayer.
type Container_Product_Order_Network_Storage_Backup_Evault_Vault struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order for Enterprise Storage
type Container_Product_Order_Network_Storage_Enterprise struct {
	Container_Product_Order

	// This must be populated only for replicant volume ordering. It represents the identifier of the origin [[SoftLayer_Network_Storage]].
	OriginVolumeId *int `json:"originVolumeId,omitempty" xmlrpc:"originVolumeId,omitempty"`

	// This must be populated only for replicant volume ordering. It represents the [[SoftLayer_Network_Storage_Schedule]] that will be be used to replicate the origin [[SoftLayer_Network_Storage]] volume.
	OriginVolumeScheduleId *int `json:"originVolumeScheduleId,omitempty" xmlrpc:"originVolumeScheduleId,omitempty"`

	// This must be populated for block storage orders. This should match the OS type of the host(s) that will connect to the volume. The only required property is the keyName of the OS type. This property is ignored for file storage orders.
	OsFormatType *Network_Storage_Iscsi_OS_Type `json:"osFormatType,omitempty" xmlrpc:"osFormatType,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order for Enterprise Storage Snapshot Space.
type Container_Product_Order_Network_Storage_Enterprise_SnapshotSpace struct {
	Container_Product_Order

	// The [[SoftLayer_Network_Storage]] id for which snapshot space is being ordered for.
	VolumeId *int `json:"volumeId,omitempty" xmlrpc:"volumeId,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an upgrade order for Enterprise Storage Snapshot Space.
type Container_Product_Order_Network_Storage_Enterprise_SnapshotSpace_Upgrade struct {
	Container_Product_Order_Network_Storage_Enterprise_SnapshotSpace
}

// This datatype is to be used for object storage orders.
type Container_Product_Order_Network_Storage_Hub struct {
	Container_Product_Order
}

// This class is used to contain a datacenter location and its associated active usage rate prices for object storage ordering.
type Container_Product_Order_Network_Storage_Hub_Datacenter struct {
	Entity

	// The datacenter location where object storage is available.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// The collection of active usage rate item prices.
	UsageRatePrices []Product_Item_Price `json:"usageRatePrices,omitempty" xmlrpc:"usageRatePrices,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an ISCSI order with SoftLayer.
type Container_Product_Order_Network_Storage_Iscsi struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an ISCSI Replication order with SoftLayer.
type Container_Product_Order_Network_Storage_Iscsi_Replication struct {
	Container_Product_Order

	// the [[SoftLayer_Network_Storage_Iscsi_EqualLogic_Version3]] Id.
	VolumeId *int `json:"volumeId,omitempty" xmlrpc:"volumeId,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an ISCSI Snapshot Space order with SoftLayer.
type Container_Product_Order_Network_Storage_Iscsi_SnapshotSpace struct {
	Container_Product_Order

	// the [[SoftLayer_Network_Storage_Iscsi_EqualLogic_Version3]] Id.
	VolumeId *int `json:"volumeId,omitempty" xmlrpc:"volumeId,omitempty"`
}

// The SoftLayer_Container_Product_Order_Network_Storage_Modification datatype has everything required to place a modification to an existing StorageLayer account with SoftLayer. Modifications, at present time, include upgrade and downgrades only. The ''volumeId'' property must be set to the network storage volume id to be upgraded. Once populated send this container to the [[SoftLayer_Product_Order::placeOrder]] method.
//
// The ''packageId'' property passed in for CloudLayer storage accounts must be set to 0 (zero) and the ''quantity'' property must be set to 1. The location does not have to be set. Please use the [[SoftLayer_Product_Package]] service to retrieve a list of CloudLayer items.
//
// NOTE: When upgrading CloudLayer storage service from a metered plan (pay as you go) to a non-metered plan, make sure the chosen plan's storage allotment has enough space to cover the current usage. If the chosen plan's usage allotment is less than the CloudLayer storage's usage the order will be rejected.
type Container_Product_Order_Network_Storage_Modification struct {
	Container_Product_Order

	// The id of the StorageLayer account to modify.
	VolumeId *int `json:"volumeId,omitempty" xmlrpc:"volumeId,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder when placing network attached storage orders.
type Container_Product_Order_Network_Storage_Nas struct {
	Container_Product_Order
}

// This datatype is to be used for ordering object storage products using the object_storage [[SoftLayer_Product_Item_Category|category]]. For object storage products using hub [[SoftLayer_Product_Item_Category|category]] use the [[SoftLayer_Container_Product_Order_Network_Storage_Hub]] order container.
type Container_Product_Order_Network_Storage_Object struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a subnet order with SoftLayer.
type Container_Product_Order_Network_Subnet struct {
	Container_Product_Order

	// The description which includes the network identifier, Classless Inter-Domain Routing prefix and the available slot count.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The [[SoftLayer_Network_Subnet_IpAddress]] id.
	EndPointIpAddressId *int `json:"endPointIpAddressId,omitempty" xmlrpc:"endPointIpAddressId,omitempty"`

	// The [[SoftLayer_Network_Vlan]] id.
	EndPointVlanId *int `json:"endPointVlanId,omitempty" xmlrpc:"endPointVlanId,omitempty"`

	// The [[SoftLayer_Network_Subnet]] id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// This is the hostname for the router associated with the [[SoftLayer_Network_Subnet|subnet]]. This is a readonly property.
	RouterHostname *string `json:"routerHostname,omitempty" xmlrpc:"routerHostname,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a network ipsec vpn order with SoftLayer.
type Container_Product_Order_Network_Tunnel_Ipsec struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a network vlan order with SoftLayer.
type Container_Product_Order_Network_Vlan struct {
	Container_Product_Order

	// The description which includes the primary router's hostname plus the vlan number.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The datacenter portion of the hostname.
	HostnameDatacenter *string `json:"hostnameDatacenter,omitempty" xmlrpc:"hostnameDatacenter,omitempty"`

	// The router portion of the hostname.
	HostnameRouter *string `json:"hostnameRouter,omitempty" xmlrpc:"hostnameRouter,omitempty"`

	// The [[SoftLayer_Network_Vlan]] id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The optional name for this VLAN
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The router object on which the new VLAN should be created.
	Router *Hardware `json:"router,omitempty" xmlrpc:"router,omitempty"`

	// The ID of the [[SoftLayer_Hardware_Router]] object on which the new VLAN should be created.
	RouterId *int `json:"routerId,omitempty" xmlrpc:"routerId,omitempty"`

	// The collection of subnets associated with this vlan.
	Subnets []Container_Product_Order `json:"subnets,omitempty" xmlrpc:"subnets,omitempty"`

	// The vlan number.
	VlanNumber *int `json:"vlanNumber,omitempty" xmlrpc:"vlanNumber,omitempty"`
}

// This class contains the collections of public and private VLANs that are available during the ordering process.
type Container_Product_Order_Network_Vlans struct {
	Entity

	// The collection of private vlans available during ordering.
	PrivateVlans []Container_Product_Order `json:"privateVlans,omitempty" xmlrpc:"privateVlans,omitempty"`

	// The collection of public vlans available during ordering.
	PublicVlans []Container_Product_Order `json:"publicVlans,omitempty" xmlrpc:"publicVlans,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder when linking a Bluemix account to a newly created SoftLayer account.
type Container_Product_Order_NewCustomerSetup struct {
	Container_Product_Order

	// no documentation yet
	AuthorizationToken *string `json:"authorizationToken,omitempty" xmlrpc:"authorizationToken,omitempty"`

	// no documentation yet
	ExternalAccountId *string `json:"externalAccountId,omitempty" xmlrpc:"externalAccountId,omitempty"`

	// no documentation yet
	ExternalServiceProviderKey *string `json:"externalServiceProviderKey,omitempty" xmlrpc:"externalServiceProviderKey,omitempty"`
}

// This is used for storing various items about the order. Currently used for storing additional raid information when ordering servers. This is optional
type Container_Product_Order_Property struct {
	Entity

	// The property name
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The property value
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// When an order is placed (SoftLayer_Product_Order::placeOrder), a receipt is returned when the order is created successfully. The information in the receipt helps explain information about the order. It's order ID, and all the data within the order as well.
//
// For PayPal Orders, an URL is also returned to the user so that the user can complete the transaction. Users paying with PayPal must continue on to this URL, login and pay. When doing this, PayPal will redirect the user back to a SoftLayer page which will then "finalize" the authorization process. From here, Sales will verify the order by contacting the user in some way, unless sales has already spoken to the user about approving the order.
//
// For users paying with a credit card, a receipt means the order has gone to sales and is awaiting approval.
type Container_Product_Order_Receipt struct {
	Entity

	// This URL refers to the location where you will visit to complete the payment authorization for an external service, such as PayPal. This property is associated with <code>externalPaymentToken</code> and will only be populated when purchasing products with an external service.
	//
	// Once you visit this location, you will be presented with the options to confirm payment or deny payment. If you confirm payment, you will be redirected back to the receipt for your order. If you deny, you will be redirected back to the cancel order page where you do not need to take any additional action.
	//
	// Until you confirm payment with the external service, your products will not be provisioned or accessible for your consumption. Upon successfully confirming payment, our system will be notified and the order approval and provisioning systems will begin processing. After provisioning is complete, your services will be available.
	ExternalPaymentCheckoutUrl *string `json:"externalPaymentCheckoutUrl,omitempty" xmlrpc:"externalPaymentCheckoutUrl,omitempty"`

	// This token refers to the identifier for the external payment authorization. This token is associated with the <code>externalPaymentCheckoutUrl</code> and is only populated when purchasing products with an external service like PayPal.
	ExternalPaymentToken *string `json:"externalPaymentToken,omitempty" xmlrpc:"externalPaymentToken,omitempty"`

	// The date when SoftLayer received the order.
	OrderDate *Time `json:"orderDate,omitempty" xmlrpc:"orderDate,omitempty"`

	// This is a copy of the order container (SoftLayer_Container_Product_Order) which holds all the data related to an order. This will only return when an order is processed successfully. It will contain all the items in an order as well as the order totals.
	OrderDetails *Container_Product_Order `json:"orderDetails,omitempty" xmlrpc:"orderDetails,omitempty"`

	// SoftLayer's unique identifier for the order.
	OrderId *int `json:"orderId,omitempty" xmlrpc:"orderId,omitempty"`

	// Deprecation notice: use <code>externalPaymentCheckoutUrl</code> instead of this property.
	//
	// This URL refers to the location where you will visit to complete the payment authorization for PayPal. This property is associated with <code>paypalToken</code> and will only be populated when purchasing products with PayPal.
	//
	// Once you visit PayPal's site, you will be presented with the options to confirm payment or deny payment. If you confirm payment, you will be redirected back to the receipt for your order. If you deny, you will be redirected back to the cancel order page where you do not need to take any additional action.
	//
	// Until you confirm payment with PayPal, your products will not be provisioned or accessible for your consumption. Upon successfully confirming payment, our system will be notified and the order approval and provisioning systems will begin processing. After provisioning is complete, your services will be available.
	PaypalCheckoutUrl *string `json:"paypalCheckoutUrl,omitempty" xmlrpc:"paypalCheckoutUrl,omitempty"`

	// Deprecation notice: use <code>externalPaymentToken</code> instead of this property.
	//
	// This token refers to the identifier provided when payment is processed via PayPal. This token is associated with the <code>paypalCheckoutUrl</code>.
	PaypalToken *string `json:"paypalToken,omitempty" xmlrpc:"paypalToken,omitempty"`

	// This is a copy of the order that was successfully placed (SoftLayer_Billing_Order). This will only return when an order is processed successfully.
	PlacedOrder *Billing_Order `json:"placedOrder,omitempty" xmlrpc:"placedOrder,omitempty"`

	// This is a copy of the quote container (SoftLayer_Billing_Order_Quote) which holds all the data related to a quote. This will only return when a quote is processed successfully.
	Quote *Billing_Order_Quote `json:"quote,omitempty" xmlrpc:"quote,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype contains everything required to place a secure certificate order with SoftLayer.
type Container_Product_Order_Security_Certificate struct {
	Container_Product_Order

	// The administrator contact associated with a SSL certificate. If the contact is not provided the technical contact will be used. If the address is not provided the organization information address will be used.
	AdministrativeContact *Container_Product_Order_Attribute_Contact `json:"administrativeContact,omitempty" xmlrpc:"administrativeContact,omitempty"`

	// The billing contact associated with a SSL certificate. If the contact is not provided the technical contact will be used. If the address is not provided the organization information address will be used.
	BillingContact *Container_Product_Order_Attribute_Contact `json:"billingContact,omitempty" xmlrpc:"billingContact,omitempty"`

	// The base64 encoded string that sent from an applicant to a certificate authority. The CSR contains information identifying the applicant and the public key chosen by the applicant. The corresponding private key should not be included.
	CertificateSigningRequest *string `json:"certificateSigningRequest,omitempty" xmlrpc:"certificateSigningRequest,omitempty"`

	// The email address that can approve a secure certificate order.
	OrderApproverEmailAddress *string `json:"orderApproverEmailAddress,omitempty" xmlrpc:"orderApproverEmailAddress,omitempty"`

	// The organization information associated with a SSL certificate.
	OrganizationInformation *Container_Product_Order_Attribute_Organization `json:"organizationInformation,omitempty" xmlrpc:"organizationInformation,omitempty"`

	// Indicates if it is an renewal order of an existing SSL certificate.
	RenewalFlag *bool `json:"renewalFlag,omitempty" xmlrpc:"renewalFlag,omitempty"`

	// The number of servers.
	ServerCount *int `json:"serverCount,omitempty" xmlrpc:"serverCount,omitempty"`

	// The server type. This is the name from a [[SoftLayer_Security_Certificate_Request_ServerType]] object.
	ServerType *string `json:"serverType,omitempty" xmlrpc:"serverType,omitempty"`

	// The technical contact associated with a SSL certificate. If the address is not provided the organization information address will be used.
	TechnicalContact *Container_Product_Order_Attribute_Contact `json:"technicalContact,omitempty" xmlrpc:"technicalContact,omitempty"`

	// The period that a SSL certificate is valid for.  For example, 12, 24, 36
	ValidityMonths *int `json:"validityMonths,omitempty" xmlrpc:"validityMonths,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder.
type Container_Product_Order_Service struct {
	Container_Product_Order
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a virtual license order with SoftLayer.
type Container_Product_Order_Software_Component_Virtual struct {
	Container_Product_Order

	// array of ip address ids for which a license should be allocated for.
	EndPointIpAddressIds []int `json:"endPointIpAddressIds,omitempty" xmlrpc:"endPointIpAddressIds,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a hardware security module order with SoftLayer.
type Container_Product_Order_Software_License struct {
	Container_Product_Order
}

// This object holds all of the ssh key ids that will allow authentication to a single server.
type Container_Product_Order_SshKeys struct {
	Entity

	// An array of SoftLayer_Security_Ssh_Key IDs to assign to a server.
	SshKeyIds []int `json:"sshKeyIds,omitempty" xmlrpc:"sshKeyIds,omitempty"`
}

// A single storage group container used for a hardware server order.
//
// This object describes a single storage group that can be added to an order container.
type Container_Product_Order_Storage_Group struct {
	Entity

	// Size of the array in gigabytes. Must be within limitations of the smallest drive assigned to the storage group and the storage group type.
	ArraySize *Float64 `json:"arraySize,omitempty" xmlrpc:"arraySize,omitempty"`

	// The array type id from a [[SoftLayer_Configuration_Storage_Group_Array_Type]] object.
	ArrayTypeId *int `json:"arrayTypeId,omitempty" xmlrpc:"arrayTypeId,omitempty"`

	// Integer array of drive indexes to use in the storage group.
	HardDrives []int `json:"hardDrives,omitempty" xmlrpc:"hardDrives,omitempty"`

	// If an array should be protected by an hotspare, the drive index of the hotspares should be here.
	//
	// If a drive is a hotspare for all arrays then a separate storage group with array type GLOBAL_HOT_SPARE should be used
	HotSpareDrives []int `json:"hotSpareDrives,omitempty" xmlrpc:"hotSpareDrives,omitempty"`

	// The id for a [[SoftLayer_Hardware_Component_Partition_Template]] object, which will determine the partitions to add to the storage group.
	//
	// If this storage group is not a primary storage group, then this will not be used.
	PartitionTemplateId *int `json:"partitionTemplateId,omitempty" xmlrpc:"partitionTemplateId,omitempty"`

	// Defines the partitions for the storage group.
	//
	// If this storage group is not a secondary storage group, then this will not be used.
	Partitions []Container_Product_Order_Storage_Group_Partition `json:"partitions,omitempty" xmlrpc:"partitions,omitempty"`
}

// A storage group partition container used for a hardware server order.
//
// This object describes the partitions for a single storage group that can be added to an order container.
type Container_Product_Order_Storage_Group_Partition struct {
	Entity

	// Is this a grow partition
	IsGrow *bool `json:"isGrow,omitempty" xmlrpc:"isGrow,omitempty"`

	// The name of this partition
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The size of this partition
	Size *Float64 `json:"size,omitempty" xmlrpc:"size,omitempty"`
}

// When ordering paid support this datatype needs to be populated and sent to SoftLayer_Product_Order::placeOrder.
type Container_Product_Order_Support struct {
	Container_Product_Order
}

// This container type is used for placing orders for external authentication, such as phone-based authentication.
type Container_Product_Order_User_Customer_External_Binding struct {
	Container_Product_Order

	// The external id that access to external authentication is being purchased for.
	ExternalId *string `json:"externalId,omitempty" xmlrpc:"externalId,omitempty"`

	// The SoftLayer [[SoftLayer_User_Customer|user]] identifier that an external binding is being purchased for.
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// The [[SoftLayer_User_Customer_External_Binding_Vendor|vendor]] identifier for the external binding being purchased.
	VendorId *int `json:"vendorId,omitempty" xmlrpc:"vendorId,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place a Portable Storage order with SoftLayer.
type Container_Product_Order_Virtual_Disk_Image struct {
	Container_Product_Order

	// Label for the portable storage volume.
	DiskDescription *string `json:"diskDescription,omitempty" xmlrpc:"diskDescription,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order with SoftLayer.
type Container_Product_Order_Virtual_Guest struct {
	Container_Product_Order_Hardware_Server

	// Identifier of the [[SoftLayer_Virtual_Disk_Image]] to boot from.
	BootableDiskId *int `json:"bootableDiskId,omitempty" xmlrpc:"bootableDiskId,omitempty"`
}

// This is the datatype that needs to be populated and sent to SoftLayer_Product_Order::placeOrder. This datatype has everything required to place an order with SoftLayer.
type Container_Product_Order_Virtual_Guest_Upgrade struct {
	Container_Product_Order_Virtual_Guest
}

// This is the datatype that needs to be populated and sent to SoftLayer_Provisioning_Maintenance_Window::addCustomerUpgradeWindow. This datatype has everything required to place an order with SoftLayer.
type Container_Provisioning_Maintenance_Window struct {
	Entity

	// Maintenance classifications.
	ClassificationIds []Provisioning_Maintenance_Classification `json:"classificationIds,omitempty" xmlrpc:"classificationIds,omitempty"`

	// Maintenance classifications.
	ItemCategoryIds []Product_Item_Category `json:"itemCategoryIds,omitempty" xmlrpc:"itemCategoryIds,omitempty"`

	// The maintenance window id
	MaintenanceWindowId *int `json:"maintenanceWindowId,omitempty" xmlrpc:"maintenanceWindowId,omitempty"`

	// Maintenance window ticket id
	TicketId *int `json:"ticketId,omitempty" xmlrpc:"ticketId,omitempty"`

	// Maintenance window date
	WindowMaintenanceDate *Time `json:"windowMaintenanceDate,omitempty" xmlrpc:"windowMaintenanceDate,omitempty"`
}

// no documentation yet
type Container_Referral_Partner_Commission struct {
	Entity

	// no documentation yet
	CommissionAmount *Float64 `json:"commissionAmount,omitempty" xmlrpc:"commissionAmount,omitempty"`

	// no documentation yet
	CommissionRate *Float64 `json:"commissionRate,omitempty" xmlrpc:"commissionRate,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	ReferralAccountId *int `json:"referralAccountId,omitempty" xmlrpc:"referralAccountId,omitempty"`

	// no documentation yet
	ReferralCompanyName *string `json:"referralCompanyName,omitempty" xmlrpc:"referralCompanyName,omitempty"`

	// no documentation yet
	ReferralPartnerAccountId *int `json:"referralPartnerAccountId,omitempty" xmlrpc:"referralPartnerAccountId,omitempty"`

	// no documentation yet
	ReferralRevenue *Float64 `json:"referralRevenue,omitempty" xmlrpc:"referralRevenue,omitempty"`
}

// no documentation yet
type Container_Referral_Partner_Payment_Option struct {
	Entity

	// no documentation yet
	AccountNumber *string `json:"accountNumber,omitempty" xmlrpc:"accountNumber,omitempty"`

	// no documentation yet
	AccountType *string `json:"accountType,omitempty" xmlrpc:"accountType,omitempty"`

	// no documentation yet
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// no documentation yet
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// no documentation yet
	BankTransitNumber *string `json:"bankTransitNumber,omitempty" xmlrpc:"bankTransitNumber,omitempty"`

	// no documentation yet
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// no documentation yet
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// no documentation yet
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// no documentation yet
	FederalTaxId *string `json:"federalTaxId,omitempty" xmlrpc:"federalTaxId,omitempty"`

	// no documentation yet
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// no documentation yet
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// no documentation yet
	PaymentType *string `json:"paymentType,omitempty" xmlrpc:"paymentType,omitempty"`

	// no documentation yet
	PaypalEmail *string `json:"paypalEmail,omitempty" xmlrpc:"paypalEmail,omitempty"`

	// no documentation yet
	PhoneNumber *string `json:"phoneNumber,omitempty" xmlrpc:"phoneNumber,omitempty"`

	// no documentation yet
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// no documentation yet
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`
}

// no documentation yet
type Container_Referral_Partner_Prospect struct {
	Entity

	// no documentation yet
	Address1 *string `json:"address1,omitempty" xmlrpc:"address1,omitempty"`

	// no documentation yet
	Address2 *string `json:"address2,omitempty" xmlrpc:"address2,omitempty"`

	// no documentation yet
	City *string `json:"city,omitempty" xmlrpc:"city,omitempty"`

	// no documentation yet
	CompanyName *string `json:"companyName,omitempty" xmlrpc:"companyName,omitempty"`

	// no documentation yet
	Country *string `json:"country,omitempty" xmlrpc:"country,omitempty"`

	// no documentation yet
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// no documentation yet
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// no documentation yet
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// no documentation yet
	OfficePhone *string `json:"officePhone,omitempty" xmlrpc:"officePhone,omitempty"`

	// no documentation yet
	PostalCode *string `json:"postalCode,omitempty" xmlrpc:"postalCode,omitempty"`

	// no documentation yet
	Questions []string `json:"questions,omitempty" xmlrpc:"questions,omitempty"`

	// no documentation yet
	Responses []Survey_Response `json:"responses,omitempty" xmlrpc:"responses,omitempty"`

	// no documentation yet
	State *string `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// no documentation yet
	SurveyId *string `json:"surveyId,omitempty" xmlrpc:"surveyId,omitempty"`
}

// The SoftLayer_Container_RemoteManagement_Graphs_SensorSpeed contains graphs to  display speed for each of the server's fans.  Fan speeds are gathered from the server's remote management card.
type Container_RemoteManagement_Graphs_SensorSpeed struct {
	Entity

	// The graph to display the server's fan speed.
	Graph *[]byte `json:"graph,omitempty" xmlrpc:"graph,omitempty"`

	// A title that may be used to display for the graph.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`
}

// The SoftLayer_Container_RemoteManagement_Graphs_SensorTemperature contains graphs to display the cpu(s) and system temperatures retrieved from the management card using thermometer graphs.
type Container_RemoteManagement_Graphs_SensorTemperature struct {
	Entity

	// The graph to display the server's cpu(s) and system temperatures.
	Graph *[]byte `json:"graph,omitempty" xmlrpc:"graph,omitempty"`

	// A title that may be used to display for the graph.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`
}

// The SoftLayer_Container_RemoteManagement_PmInfo contains pminfo information retrieved from a server's remote management card.
type Container_RemoteManagement_PmInfo struct {
	Entity

	// PmInfo ID
	PmInfoId *string `json:"pmInfoId,omitempty" xmlrpc:"pmInfoId,omitempty"`

	// PmInfo Reading
	PmInfoReading *string `json:"pmInfoReading,omitempty" xmlrpc:"pmInfoReading,omitempty"`
}

// The SoftLayer_Container_RemoteManagement_SensorReadings contains sensor information retrieved from a server's remote management card.
type Container_RemoteManagement_SensorReading struct {
	Entity

	// Lower Non-Recoverable threshold
	LowerCritical *string `json:"lowerCritical,omitempty" xmlrpc:"lowerCritical,omitempty"`

	// Lower Non-Critical threshold
	LowerNonCritical *string `json:"lowerNonCritical,omitempty" xmlrpc:"lowerNonCritical,omitempty"`

	// Lower Non-Recoverable threshold
	LowerNonRecoverable *string `json:"lowerNonRecoverable,omitempty" xmlrpc:"lowerNonRecoverable,omitempty"`

	// Sensor ID
	SensorId *string `json:"sensorId,omitempty" xmlrpc:"sensorId,omitempty"`

	// Sensor Reading
	SensorReading *string `json:"sensorReading,omitempty" xmlrpc:"sensorReading,omitempty"`

	// Sensor Units
	SensorUnits *string `json:"sensorUnits,omitempty" xmlrpc:"sensorUnits,omitempty"`

	// Sensor Status
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// Upper Critical threshold
	UpperCritical *string `json:"upperCritical,omitempty" xmlrpc:"upperCritical,omitempty"`

	// Upper Non-Critical threshold
	UpperNonCritical *string `json:"upperNonCritical,omitempty" xmlrpc:"upperNonCritical,omitempty"`

	// Upper Non-Recoverable threshold
	UpperNonRecoverable *string `json:"upperNonRecoverable,omitempty" xmlrpc:"upperNonRecoverable,omitempty"`
}

// The SoftLayer_Container_RemoteManagement_SensorReadingsWithGraphs contains the raw data retrieved from a server's remote management card.  Along with the raw data, two sets of graphs will be returned.  One set of graphs is used to display, using thermometer graphs, the temperatures (cpu(s) and system) retrieved from the management card.  The other set is used to display speed for each of the server's fans.
type Container_RemoteManagement_SensorReadingsWithGraphs struct {
	Entity

	// The raw data returned from the server's remote management card.
	RawData []Container_RemoteManagement_SensorReading `json:"rawData,omitempty" xmlrpc:"rawData,omitempty"`

	// The graph(s) to display the server's fan speeds.
	SpeedGraphs []Container_RemoteManagement_Graphs_SensorSpeed `json:"speedGraphs,omitempty" xmlrpc:"speedGraphs,omitempty"`

	// The graph(s) to display the server's cpu(s) and system temperatures.
	TemperatureGraphs []Container_RemoteManagement_Graphs_SensorTemperature `json:"temperatureGraphs,omitempty" xmlrpc:"temperatureGraphs,omitempty"`
}

// The metadata service resource container is used to store information about a single service resource.
type Container_Resource_Metadata_ServiceResource struct {
	Entity

	// The backend IP address for this resource
	BackendIpAddress *string `json:"backendIpAddress,omitempty" xmlrpc:"backendIpAddress,omitempty"`

	// The type for this resource
	Type *Network_Service_Resource_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// This data type is a container that stores information about a single indexed object type.  Object type information can be used for discovery of searchable data and for creation or validation of object index search strings.  Each of these containers holds a collection of <b>[[SoftLayer_Container_Search_ObjectType_Property (type)|SoftLayer_Container_Search_ObjectType_Property]]</b> objects, specifying which object properties are exposed for the current user.  Refer to the the documentation for the <b>[[SoftLayer_Search/search|search()]]</b> method for information on using object types in search strings.
type Container_Search_ObjectType struct {
	Entity

	// Name of object type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A collection of [[SoftLayer_Container_Search_ObjectType_Property|object properties]].
	Properties []Container_Search_ObjectType_Property `json:"properties,omitempty" xmlrpc:"properties,omitempty"`
}

// This data type is a container that stores information about a single property of a searchable object type.  Each <b>[[SoftLayer_Container_Search_ObjectType (type)|SoftLayer_Container_Search_ObjectType]]</b> object holds a collection of these properties.  Property information can be used for discovery of searchable data and for the creation or validation of object index search strings.  Note that properties are only understood by the <b>[[SoftLayer_Search/advancedSearch|advancedSearch()]]</b> method.  Refer to the <b>advancedSearch()</b> method for information on using properties in search strings.
type Container_Search_ObjectType_Property struct {
	Entity

	// Name of property.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// Indicates if this property can be sorted.
	SortableFlag *bool `json:"sortableFlag,omitempty" xmlrpc:"sortableFlag,omitempty"`

	// Property data type.  Valid values include 'boolean', 'integer', 'date', 'string' or 'text'.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// The SoftLayer_Container_Search_Result data type represents a result row from an execution of Search service.
type Container_Search_Result struct {
	Entity

	// An array of terms that were matched in the resource object.
	MatchedTerms []string `json:"matchedTerms,omitempty" xmlrpc:"matchedTerms,omitempty"`

	// The score ratio of the result for relevance to the search criteria.
	RelevanceScore *Float64 `json:"relevanceScore,omitempty" xmlrpc:"relevanceScore,omitempty"`

	// A search results resource object that matched search criteria.
	Resource *Entity `json:"resource,omitempty" xmlrpc:"resource,omitempty"`

	// The type of the resource object that matched search criteria.
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// The SoftLayer_Container_Software_Component_HostIps_Policy container holds the title and value of a current host ips policy.
type Container_Software_Component_HostIps_Policy struct {
	Entity

	// The value of a host ips category.
	Policy *string `json:"policy,omitempty" xmlrpc:"policy,omitempty"`

	// The category title of a host ips policy.
	PolicyTitle *string `json:"policyTitle,omitempty" xmlrpc:"policyTitle,omitempty"`
}

// These are the results of a tax calculation. The tax calculation was kicked off but allowed to run in the background. This type stores the results so that an interface can be updated with up-to-date information.
type Container_Tax_Cache struct {
	Entity

	// The percentage of the final total that should be tax.
	EffectiveTaxRate *Float64 `json:"effectiveTaxRate,omitempty" xmlrpc:"effectiveTaxRate,omitempty"`

	// The container that holds the four actual tax rates, one for each fee type.
	Items []Container_Tax_Cache_Item `json:"items,omitempty" xmlrpc:"items,omitempty"`

	// The status of the tax request. This should be PENDING, FAILED, or COMPLETED.
	Status *string `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The final amount of tax for the order.
	TotalTaxAmount *Float64 `json:"totalTaxAmount,omitempty" xmlrpc:"totalTaxAmount,omitempty"`
}

// This represents one order item in a tax calculation.
type Container_Tax_Cache_Item struct {
	Entity

	// The category code for the referenced product.
	CategoryCode *string `json:"categoryCode,omitempty" xmlrpc:"categoryCode,omitempty"`

	// This hash will match to the hash on an order container.
	ContainerHash *string `json:"containerHash,omitempty" xmlrpc:"containerHash,omitempty"`

	// The reference to the price for this order item.
	ItemPriceId *int `json:"itemPriceId,omitempty" xmlrpc:"itemPriceId,omitempty"`

	// This is the container containing the individual tax rates.
	TaxRates *Container_Tax_Rates `json:"taxRates,omitempty" xmlrpc:"taxRates,omitempty"`
}

// This contains the four tax rates, one for each fee type.
type Container_Tax_Rates struct {
	Entity

	// The tax rate associated with the labor fee.
	LaborTaxRate *Float64 `json:"laborTaxRate,omitempty" xmlrpc:"laborTaxRate,omitempty"`

	// A reference to a location.
	LocationId *Float64 `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// The tax rate associated with the one-time fee.
	OneTimeTaxRate *Float64 `json:"oneTimeTaxRate,omitempty" xmlrpc:"oneTimeTaxRate,omitempty"`

	// The tax rate associated with the recurring fee.
	RecurringTaxRate *Float64 `json:"recurringTaxRate,omitempty" xmlrpc:"recurringTaxRate,omitempty"`

	// The tax rate associated with the setup fee.
	SetupTaxRate *Float64 `json:"setupTaxRate,omitempty" xmlrpc:"setupTaxRate,omitempty"`
}

// SoftLayer_Container_Ticket_GraphInputs models a single inbound object for a given ticket graph.
type Container_Ticket_GraphInputs struct {
	Entity

	// This is a unix timestamp that represents the stop date/time for a graph.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// The front-end or back-end network uplink interface associated with this server.
	NetworkInterfaceId *int `json:"networkInterfaceId,omitempty" xmlrpc:"networkInterfaceId,omitempty"`

	// *
	Pod *int `json:"pod,omitempty" xmlrpc:"pod,omitempty"`

	// This is a human readable name for the server or rack being graphed.
	ServerName *string `json:"serverName,omitempty" xmlrpc:"serverName,omitempty"`

	// This is a unix timestamp that represents the begin date/time for a graph.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}

// SoftLayer_Container_Ticket_GraphOutputs models a single outbound object for a given bandwidth graph.
type Container_Ticket_GraphOutputs struct {
	Entity

	// The raw PNG binary data to be displayed once the graph is drawn.
	GraphImage *[]byte `json:"graphImage,omitempty" xmlrpc:"graphImage,omitempty"`

	// The title that ended up being displayed as part of the graph image.
	GraphTitle *string `json:"graphTitle,omitempty" xmlrpc:"graphTitle,omitempty"`

	// The maximum date included in this graph.
	MaxEndDate *Time `json:"maxEndDate,omitempty" xmlrpc:"maxEndDate,omitempty"`

	// The minimum date included in this graph.
	MinStartDate *Time `json:"minStartDate,omitempty" xmlrpc:"minStartDate,omitempty"`
}

// no documentation yet
type Container_Ticket_Priority struct {
	Entity

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// no documentation yet
	Value *int `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Container_Ticket_Survey_Preference struct {
	Entity

	// no documentation yet
	Applicable *bool `json:"applicable,omitempty" xmlrpc:"applicable,omitempty"`

	// no documentation yet
	OptedOut *bool `json:"optedOut,omitempty" xmlrpc:"optedOut,omitempty"`

	// no documentation yet
	OptedOutDate *Time `json:"optedOutDate,omitempty" xmlrpc:"optedOutDate,omitempty"`
}

// Container class used to hold user authentication token
type Container_User_Authentication_Token struct {
	Entity

	// hash that gets populated for user authentication
	Hash *string `json:"hash,omitempty" xmlrpc:"hash,omitempty"`

	// the user authenticated object
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// the id of the user to authenticate
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// Container classed used to hold external authentication information
type Container_User_Customer_External_Binding struct {
	Entity

	// The unique token that is created by an external authentication request.
	AuthenticationToken *string `json:"authenticationToken,omitempty" xmlrpc:"authenticationToken,omitempty"`

	// The OpenID Connect access token which provides access to a resource by the OpenID Connect provider.
	OpenIdConnectAccessToken *string `json:"openIdConnectAccessToken,omitempty" xmlrpc:"openIdConnectAccessToken,omitempty"`

	// The account to login to, if not provided a default will be used.
	OpenIdConnectAccountId *int `json:"openIdConnectAccountId,omitempty" xmlrpc:"openIdConnectAccountId,omitempty"`

	// The OpenID Connect provider type, as a string.
	OpenIdConnectProvider *string `json:"openIdConnectProvider,omitempty" xmlrpc:"openIdConnectProvider,omitempty"`

	// Your SoftLayer customer portal user's portal password.
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// The answer to your security question.
	SecurityQuestionAnswer *string `json:"securityQuestionAnswer,omitempty" xmlrpc:"securityQuestionAnswer,omitempty"`

	// A security question you wish to answer when authenticating to the SoftLayer customer portal. This parameter isn't required if no security questions are set on your portal account or if your account is configured to not require answering a security account upon login.
	SecurityQuestionId *int `json:"securityQuestionId,omitempty" xmlrpc:"securityQuestionId,omitempty"`

	// The username you wish to authenticate to the SoftLayer customer portal with.
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`

	// The name of the vendor that will be used for external authentication
	Vendor *string `json:"vendor,omitempty" xmlrpc:"vendor,omitempty"`
}

// Container classed used to hold portal token
type Container_User_Customer_External_Binding_Phone struct {
	Container_User_Customer_External_Binding
}

// This container can be used to configure the phone authentication mode. By default, "VOICE_CALL" in "STANDARD" mode with no Pin number will be used. With the default mode, you will have to answer a phone call from a trusted 2 form factor vendor during authentication process. You have to answer the call and follow the instruction in order to complete the authentication.
//
// You can also use SMS text message or PhoneFactor mobile app modes (in case you're using PhoneFactor). Additionally, you can set up a Pin number. By requiring you to verify your secret PIN, you can ensure that you have possession of your phone.
type Container_User_Customer_External_Binding_Phone_Mode struct {
	Entity

	// Authentication mode. Valid modes are: VOICE_CALL, SMS_TEXT, PHONE_APP
	//
	//
	// *VOICE_CALL
	// In this mode, users will receive a phone call to authenticate. Using PIN can enhance the security of the phone authentication by requiring the user to enter a PIN during the authentication call. Valid Pin modes are: PIN, VOICE_PRINT, STANDARD
	//
	//
	// **STANDARD: (default) No PIN is used.
	// **PIN: 4 to 10 digit numeric value
	// **VOICE_PRINT: The user's voice will be used to identify the user.
	//
	//
	// *SMS_TEXT
	// SMS Text mode will send a SMS text message to the user's phone to complete the authentication.  There are 2 different pin modes:
	//
	//
	// **OTP: (default) A text message containing a One-Time Passcode (OTP) is sent to the user. The user must reply to the text message entering this OTP to complete the authentication.
	// **OTP_PIN: This mode enhances the security of the authentication by requiring the user to enter the OTP + their PIN in the text reply.
	//
	//
	//
	//
	// *PHONE_APP
	// This mode is applicable for PhoneFactor. Phone App mode results in a notification being sent to the user's PhoneFactor phone app. There are 2 different pin modes for the mobile app authentication.
	// **STANDARD: (default) The first authentication is when the user signs on using a username and password.
	// The second authentication is when the user receives a notification in the PhoneFactor phone app. In Standard Mode, users will prompted to authenticate, deny, or deny and report fraud.
	// **PIN: This mode enhances the security of the authentication by requiring the user to enter their PIN in the phone app.
	Mode *string `json:"mode,omitempty" xmlrpc:"mode,omitempty"`

	// Optional authentication pin.
	Pin *string `json:"pin,omitempty" xmlrpc:"pin,omitempty"`

	// Available Pin modes are: PIN, VOICE_PRINT, STANDARD Default: STANDARD (Pin is not used)
	PinMode *string `json:"pinMode,omitempty" xmlrpc:"pinMode,omitempty"`
}

// Container classed used to hold portal token
type Container_User_Customer_External_Binding_Totp struct {
	Container_User_Customer_External_Binding

	// The security code used to validate a Totp credential.
	SecurityCode *string `json:"securityCode,omitempty" xmlrpc:"securityCode,omitempty"`
}

// Container classed used to hold details about an external authentication vendor.
type Container_User_Customer_External_Binding_Vendor struct {
	Entity

	// The keyname used to identify an external authentication vendor.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of an external authentication vendor.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Container classed used to hold portal token
type Container_User_Customer_External_Binding_Verisign struct {
	Container_User_Customer_External_Binding

	// A second security code that is only required if your credential has become unsynchronized.
	SecondSecurityCode *string `json:"secondSecurityCode,omitempty" xmlrpc:"secondSecurityCode,omitempty"`

	// The security code used to validate a VeriSign credential.
	SecurityCode *string `json:"securityCode,omitempty" xmlrpc:"securityCode,omitempty"`
}

// no documentation yet
type Container_User_Customer_OpenIdConnect_LoginAccountInfo struct {
	Entity

	// The customer account's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The company name associated with an account.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Container_User_Customer_OpenIdConnect_MigrationState struct {
	Entity

	// The number of days remaining in the grace period for this user's account to
	DaysToGracePeriodEnd *int `json:"daysToGracePeriodEnd,omitempty" xmlrpc:"daysToGracePeriodEnd,omitempty"`

	// Flag for whether the email address inside this SoftLayer_User_Customer object
	EmailAlreadyUsedForInvitationToAccount *bool `json:"emailAlreadyUsedForInvitationToAccount,omitempty" xmlrpc:"emailAlreadyUsedForInvitationToAccount,omitempty"`

	// Flag for whether the email address inside this SoftLayer_User_Customer object
	EmailAlreadyUsedForLinkToAccount *bool `json:"emailAlreadyUsedForLinkToAccount,omitempty" xmlrpc:"emailAlreadyUsedForLinkToAccount,omitempty"`

	// The IBMid email address where an invitation was sent.
	ExistingInvitationOpenIdConnectName *string `json:"existingInvitationOpenIdConnectName,omitempty" xmlrpc:"existingInvitationOpenIdConnectName,omitempty"`

	// Flag for whether the account is OpenIdConnect authenticated or not.
	IsAccountOpenIdConnectAuthenticated *bool `json:"isAccountOpenIdConnectAuthenticated,omitempty" xmlrpc:"isAccountOpenIdConnectAuthenticated,omitempty"`
}

// Container for holding information necessary for the setting and resetting of customer passwords
//
//
type Container_User_Customer_PasswordSet struct {
	Entity

	// id of SoftLayer_User_Security_Question
	AnsweredSecurityQuestionId *int `json:"answeredSecurityQuestionId,omitempty" xmlrpc:"answeredSecurityQuestionId,omitempty"`

	// the authentication methods required
	AuthenticationMethods []int `json:"authenticationMethods,omitempty" xmlrpc:"authenticationMethods,omitempty"`

	// the password key provided to user in the password set url link sent via email
	Key *string `json:"key,omitempty" xmlrpc:"key,omitempty"`

	// the user's new password
	Password *string `json:"password,omitempty" xmlrpc:"password,omitempty"`

	// answer to security question provided by the user
	SecurityAnswer *string `json:"securityAnswer,omitempty" xmlrpc:"securityAnswer,omitempty"`

	// array of SoftLayer_User_Security_Question
	SecurityQuestions []User_Security_Question `json:"securityQuestions,omitempty" xmlrpc:"securityQuestions,omitempty"`

	// the id of the user to authenticate
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// Container classed used to hold mobile portal token
type Container_User_Customer_Portal_MobileToken struct {
	Container_User_Customer_Portal_Token

	// True if this user login required an external binding.
	HasExternalBinding *bool `json:"hasExternalBinding,omitempty" xmlrpc:"hasExternalBinding,omitempty"`
}

// Container classed used to hold portal token
type Container_User_Customer_Portal_Token struct {
	Entity

	// hash of logged in user session id
	Hash *string `json:"hash,omitempty" xmlrpc:"hash,omitempty"`

	// the logged in user data
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// the id of the logged in user
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`
}

// This container holds user's phone information.
type Container_User_Data_Phone struct {
	Entity

	// Country code number for the phone number Default: 1 (United States & Canada +1)
	CountryCode *int `json:"countryCode,omitempty" xmlrpc:"countryCode,omitempty"`

	// Phone extension code. It can be digits, commas, *, and # are allowed.
	Extension *string `json:"extension,omitempty" xmlrpc:"extension,omitempty"`

	// Phone number can be a mobile phone number, desk phone number, or some other option. The phone number format must match the format selected in the country code.
	Phone *string `json:"phone,omitempty" xmlrpc:"phone,omitempty"`

	// Type of phone number such as "primary", "office" or "home"
	PhoneType *string `json:"phoneType,omitempty" xmlrpc:"phoneType,omitempty"`
}

// Container classed used to hold portal token
type Container_User_Employee_External_Binding_Verisign struct {
	Entity
}

// At times,such as when attaching files to tickets, it is necessary to send files to SoftLayer API methods. The SoftLayer_Container_Utility_File_Attachment data type models a single file to upload to the API.
type Container_Utility_File_Attachment struct {
	Entity

	// The contents of a file that is uploaded to the SoftLayer API.
	Data *[]byte `json:"data,omitempty" xmlrpc:"data,omitempty"`

	// The name of a file that is uploaded to the SoftLayer API.
	Filename *string `json:"filename,omitempty" xmlrpc:"filename,omitempty"`
}

// Used to describe a document in the file system on the file server
type Container_Utility_File_Descriptor struct {
	Entity

	// The name of a file as it exists on the file server.
	FileName *string `json:"fileName,omitempty" xmlrpc:"fileName,omitempty"`

	// The friendly name of a file as it exists on the file server.
	FriendlyName *string `json:"friendlyName,omitempty" xmlrpc:"friendlyName,omitempty"`

	// The date the file was last modified on the file server.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`
}

// SoftLayer_Container_Utility_File_Entity data type models a single entity on a storage resource. Entities can include anything within a storage volume including files, folders, directories, and CloudLayer storage projects.
type Container_Utility_File_Entity struct {
	Entity

	// A file entity's raw content.
	Content *[]byte `json:"content,omitempty" xmlrpc:"content,omitempty"`

	// A file entity's MIME content type.
	ContentType *string `json:"contentType,omitempty" xmlrpc:"contentType,omitempty"`

	// The date a file entity was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The date a CloudLayer storage file entity was moved into the recycle bin. This field applies to files that are pending deletion in the recycle bin.
	DeleteDate *Time `json:"deleteDate,omitempty" xmlrpc:"deleteDate,omitempty"`

	// Unique identifier for the file. This can be either a number or guid.
	Id *string `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Whether a CloudLayer storage file entity is shared with another CloudLayer user.
	IsShared *int `json:"isShared,omitempty" xmlrpc:"isShared,omitempty"`

	// The date a file entity was last changed.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A file entity's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The owner is usually the account who first upload or created the file on the resource or the account who is responsible for the file at the moment.
	Owner *string `json:"owner,omitempty" xmlrpc:"owner,omitempty"`

	// The size of a file entity in bytes.
	Size *uint `json:"size,omitempty" xmlrpc:"size,omitempty"`

	// A CloudLayer storage file entity's type. Types can include "file", "folder", "dir", and "project".
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The latest revision of a file on a CloudLayer storage volume. This number increments each time a new revision of the file is uploaded.
	Version *int `json:"version,omitempty" xmlrpc:"version,omitempty"`
}

// no documentation yet
type Container_Utility_Message struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Summary *string `json:"summary,omitempty" xmlrpc:"summary,omitempty"`
}

// SoftLayer customer servers that are purchased with the Microsoft Windows operating system are configured by default to retrieve updates from SoftLayer's local Windows Server Update Services (WSUS) server. Periodically, these servers synchronize and check for new updates from their local WSUS server. SoftLayer_Container_Utility_Microsoft_Windows_UpdateServices_Status models the results of a server's last synchronization attempt as queried from SoftLayer's WSUS servers.
type Container_Utility_Microsoft_Windows_UpdateServices_Status struct {
	Entity

	// The last time a server rebooted due to a Windows Update.
	LastRebootDate *Time `json:"lastRebootDate,omitempty" xmlrpc:"lastRebootDate,omitempty"`

	// The last time that SoftLayer's local WSUS server received a status update from a customer server.
	LastStatusDate *Time `json:"lastStatusDate,omitempty" xmlrpc:"lastStatusDate,omitempty"`

	// The last time a server synchronized with SoftLayer's local WSUS server.
	LastSyncDate *Time `json:"lastSyncDate,omitempty" xmlrpc:"lastSyncDate,omitempty"`

	// This is the private IP address for this server.
	PrivateIPAddress *string `json:"privateIPAddress,omitempty" xmlrpc:"privateIPAddress,omitempty"`

	// The status message returned from a server's last synchronization with SoftLayer's local WSUS server.
	SyncStatus *string `json:"syncStatus,omitempty" xmlrpc:"syncStatus,omitempty"`

	// A server's update status, as retrieved form SoftLayer's local WSUS server.
	UpdateStatus *string `json:"updateStatus,omitempty" xmlrpc:"updateStatus,omitempty"`
}

// SoftLayer_Container_Utility_Microsoft_Windows_UpdateServices_UpdateItem models a single Microsoft Update as reported by SoftLayer's private Windows Server Update Services (WSUS) services. All servers purchased with Microsoft Windows retrieve updates from SoftLayer's WSUS servers by default.
type Container_Utility_Microsoft_Windows_UpdateServices_UpdateItem struct {
	Entity

	// A short description of a Microsoft Windows Update.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Flag indicating that this patch failed to properly install
	Failed *bool `json:"failed,omitempty" xmlrpc:"failed,omitempty"`

	// A Windows Update's knowledge base article number. Every Windows Update can be referenced on the Microsoft Help and Support site at the URL <nowiki>http://support.microsoft.com/kb/<article number></nowiki>.
	KbArticleNumber *int `json:"kbArticleNumber,omitempty" xmlrpc:"kbArticleNumber,omitempty"`

	// Flag indicating that the update is entirely optionals
	Optional *bool `json:"optional,omitempty" xmlrpc:"optional,omitempty"`

	// Flag indicating that a reboot is needed for this update to be fully applied
	RequiresReboot *bool `json:"requiresReboot,omitempty" xmlrpc:"requiresReboot,omitempty"`
}

// The SoftLayer_Container_Utility_Network_Firewall_Rule_Attribute data type contains information relating to a single firewall rule.
type Container_Utility_Network_Firewall_Rule_Attribute struct {
	Entity

	// The valid actions for use with rules.
	Actions []string `json:"actions,omitempty" xmlrpc:"actions,omitempty"`

	// Maximum allowed number of rules.
	MaximumRuleCount *int `json:"maximumRuleCount,omitempty" xmlrpc:"maximumRuleCount,omitempty"`

	// The valid protocols for use with rules.
	Protocols []string `json:"protocols,omitempty" xmlrpc:"protocols,omitempty"`

	// The valid source ip subnet masks for use with rules.
	SourceIpSubnetMasks []Container_Utility_Network_Subnet_Mask_Generic_Detail `json:"sourceIpSubnetMasks,omitempty" xmlrpc:"sourceIpSubnetMasks,omitempty"`
}

// The SoftLayer_Container_Utility_Network_Subnet_Mask_Generic_Detail data type contains information relating to a subnet mask and details associated with that object.
type Container_Utility_Network_Subnet_Mask_Generic_Detail struct {
	Entity

	// The subnet cidr prefix.
	Cidr *string `json:"cidr,omitempty" xmlrpc:"cidr,omitempty"`

	// The subnet mask description.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The subnet mask.
	Mask *string `json:"mask,omitempty" xmlrpc:"mask,omitempty"`
}

// The SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration data type contains information relating to a template's external location for importing and exporting
type Container_Virtual_Guest_Block_Device_Template_Configuration struct {
	Entity

	// The group name to be applied to the imported template
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The note to be applied to the imported template
	Note *string `json:"note,omitempty" xmlrpc:"note,omitempty"`

	//
	// The referenceCode of the operating system software description for the imported VHD
	OperatingSystemReferenceCode *string `json:"operatingSystemReferenceCode,omitempty" xmlrpc:"operatingSystemReferenceCode,omitempty"`

	//
	// The URI for an object storage object (.vhd/.iso file)
	// <code>swift://<ObjectStorageAccountName>@<clusterName>/<containerName>/<fileName.(vhd|iso)></code>
	Uri *string `json:"uri,omitempty" xmlrpc:"uri,omitempty"`
}

// The guest configuration container is used to provide configuration options for creating computing instances.
//
// Each configuration option will include both an <code>itemPrice</code> and a <code>template</code>.
//
// The <code>itemPrice</code> value will provide hourly and monthly costs (if either are applicable), and a description of the option.
//
// The <code>template</code> will provide a fragment of the request with the properties and values that must be sent when creating a computing instance with the option.
//
// The [[SoftLayer_Virtual_Guest/getCreateObjectOptions|getCreateObjectOptions]] method returns this data structure.
//
// <style type="text/css">#properties .views-field-body p { margin-top: 1.5em; };</style>
type Container_Virtual_Guest_Configuration struct {
	Entity

	//
	// <div style="width: 200%">
	// Available block device options.
	//
	//
	// A computing instance will have at least one block device represented by a <code>device</code> number of <code>'0'</code>.
	//
	//
	// The <code>blockDevices.device</code> value in the template represents which device the option is for.
	// The <code>blockDevices.diskImage.capacity</code> value in the template represents the size, in gigabytes, of the disk.
	// The <code>localDiskFlag</code> value in the template represents whether the option is a local or SAN based disk.
	//
	//
	// Note: The block device number <code>'1'</code> is reserved for the SWAP disk attached to the computing instance.
	// </div>
	BlockDevices []Container_Virtual_Guest_Configuration_Option `json:"blockDevices,omitempty" xmlrpc:"blockDevices,omitempty"`

	//
	// <div style="width: 200%">
	// Available datacenter options.
	//
	//
	// The <code>datacenter.name</code> value in the template represents which datacenter the computing instance will be provisioned in.
	// </div>
	Datacenters []Container_Virtual_Guest_Configuration_Option `json:"datacenters,omitempty" xmlrpc:"datacenters,omitempty"`

	//
	// <div style="width: 200%">
	// Available memory options.
	//
	//
	// The <code>maxMemory</code> value in the template represents the amount of memory, in megabytes, allocated to the computing instance.
	// </div>
	Memory []Container_Virtual_Guest_Configuration_Option `json:"memory,omitempty" xmlrpc:"memory,omitempty"`

	//
	// <div style="width: 200%">
	// Available network component options.
	//
	//
	// The <code>networkComponent.maxSpeed</code> value in the template represents the link speed, in megabits per second, of the network connections for a computing instance.
	// </div>
	NetworkComponents []Container_Virtual_Guest_Configuration_Option `json:"networkComponents,omitempty" xmlrpc:"networkComponents,omitempty"`

	//
	// <div style="width: 200%">
	// Available operating system options.
	//
	//
	// The <code>operatingSystemReferenceCode</code> value in the template is an identifier for a particular operating system. When provided exactly as shown in the template, that operating system will be used.
	//
	//
	// A reference code is structured as three tokens separated by underscores. The first token represents the product, the second is the version of the product, and the third is whether the OS is 32 or 64bit.
	//
	//
	// When providing an <code>operatingSystemReferenceCode</code> while ordering a computing instance the only token required to match exactly is the product. The version token may be given as 'LATEST', else it will require an exact match as well. When the bits token is not provided, 64 bits will be assumed.
	//
	//
	// Providing the value of 'LATEST' for a version will select the latest release of that product for the operating system. As this may change over time, you should be sure that the release version is irrelevant for your applications.
	//
	//
	// For Windows based operating systems the version will represent both the release version (2008, 2012, etc) and the edition (Standard, Enterprise, etc). For all other operating systems the version will represent the major version (Centos 6, Ubuntu 12, etc) of that operating system, minor versions are not represented in a reference code.
	//
	//
	// <b>Notice</b> - Some operating systems are charged based on the value specified in <code>startCpus</code>. The price which is used can be determined by calling [[SoftLayer_Virtual_Guest/generateOrderTemplate|generateOrderTemplate]] with your desired device specifications.
	// </div>
	OperatingSystems []Container_Virtual_Guest_Configuration_Option `json:"operatingSystems,omitempty" xmlrpc:"operatingSystems,omitempty"`

	//
	// <div style="width: 200%">
	// Available processor options.
	//
	//
	// The <code>startCpus</code> value in the template represents the number of cores allocated to the computing instance.
	// The <code>dedicatedAccountHostOnlyFlag</code> value in the template represents whether the instance will run on hosts with instances belonging to other accounts.
	// </div>
	Processors []Container_Virtual_Guest_Configuration_Option `json:"processors,omitempty" xmlrpc:"processors,omitempty"`
}

// An option found within a [[SoftLayer_Container_Virtual_Guest_Configuration (type)]] structure.
type Container_Virtual_Guest_Configuration_Option struct {
	Entity

	//
	// Provides hourly and monthly costs (if either are applicable), and a description of the option.
	ItemPrice *Product_Item_Price `json:"itemPrice,omitempty" xmlrpc:"itemPrice,omitempty"`

	//
	// Provides a fragment of the request with the properties and values that must be sent when creating a computing instance with the option.
	Template *Virtual_Guest `json:"template,omitempty" xmlrpc:"template,omitempty"`
}
