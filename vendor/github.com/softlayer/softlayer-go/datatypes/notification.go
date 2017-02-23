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

// Details provided for the notification are basic.  Details such as the related preferences, name and keyname for the notification can be retrieved.  The keyname property for the notification can be used to refer to a notification when integrating into the SoftLayer Notification system.  The name property can used more for display purposes.
type Notification struct {
	Entity

	// Unique identifier for the notification.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Name that can be used by external systems to refer to a notification.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// Friendly name for the notification.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of the preferences related to the notification. These are preferences are configurable and optional for subscribers to use.
	PreferenceCount *uint `json:"preferenceCount,omitempty" xmlrpc:"preferenceCount,omitempty"`

	// The preferences related to the notification. These are preferences are configurable and optional for subscribers to use.
	Preferences []Notification_Preference `json:"preferences,omitempty" xmlrpc:"preferences,omitempty"`

	// A count of the required preferences related to the notification. While configurable, the subscriber does not have the option whether to use the preference.
	RequiredPreferenceCount *uint `json:"requiredPreferenceCount,omitempty" xmlrpc:"requiredPreferenceCount,omitempty"`

	// The required preferences related to the notification. While configurable, the subscriber does not have the option whether to use the preference.
	RequiredPreferences []Notification_Preference `json:"requiredPreferences,omitempty" xmlrpc:"requiredPreferences,omitempty"`
}

// Provides details for the delivery methods available.
type Notification_Delivery_Method struct {
	Entity

	// Determines if the delivery method is still used by the system.
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// Description used for the delivery method.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Unique identifier for the various notification delivery methods.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Name that can be used by external systems to refer to delivery method.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// Friendly name used for the delivery method.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// This is an extension of the SoftLayer_Notification class.  These are implementation details specific to those notifications which can be subscribed to and received on a mobile device.
type Notification_Mobile struct {
	Notification
}

// no documentation yet
type Notification_Occurrence_Account struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// no documentation yet
	LastNotificationUpdate *Notification_Occurrence_Update `json:"lastNotificationUpdate,omitempty" xmlrpc:"lastNotificationUpdate,omitempty"`

	// no documentation yet
	NotificationOccurrenceEvent *Notification_Occurrence_Event `json:"notificationOccurrenceEvent,omitempty" xmlrpc:"notificationOccurrenceEvent,omitempty"`
}

// no documentation yet
type Notification_Occurrence_Event struct {
	Entity

	// Indicates whether or not this event has been acknowledged by the user.
	AcknowledgedFlag *bool `json:"acknowledgedFlag,omitempty" xmlrpc:"acknowledgedFlag,omitempty"`

	// A count of a collection of attachments for this event which provide supplementary information to impacted users some examples are RFO (Reason For Outage) and root cause analysis documents.
	AttachmentCount *uint `json:"attachmentCount,omitempty" xmlrpc:"attachmentCount,omitempty"`

	// A collection of attachments for this event which provide supplementary information to impacted users some examples are RFO (Reason For Outage) and root cause analysis documents.
	Attachments []Notification_Occurrence_Event_Attachment `json:"attachments,omitempty" xmlrpc:"attachments,omitempty"`

	// When this event will end.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// The first update for this event.
	FirstUpdate *Notification_Occurrence_Update `json:"firstUpdate,omitempty" xmlrpc:"firstUpdate,omitempty"`

	// Unique identifier for this event.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of a collection of accounts impacted by this event. Each impacted account record relates directly to a [[SoftLayer_Account]].
	ImpactedAccountCount *uint `json:"impactedAccountCount,omitempty" xmlrpc:"impactedAccountCount,omitempty"`

	// A collection of accounts impacted by this event. Each impacted account record relates directly to a [[SoftLayer_Account]].
	ImpactedAccounts []Notification_Occurrence_Account `json:"impactedAccounts,omitempty" xmlrpc:"impactedAccounts,omitempty"`

	// A count of a collection of resources impacted by this event. Each record will relate to some physical resource that the user has access to such as [[SoftLayer_Hardware]] or [[SoftLayer_Virtual_Guest]].
	ImpactedResourceCount *uint `json:"impactedResourceCount,omitempty" xmlrpc:"impactedResourceCount,omitempty"`

	// A collection of resources impacted by this event. Each record will relate to some physical resource that the user has access to such as [[SoftLayer_Hardware]] or [[SoftLayer_Virtual_Guest]].
	ImpactedResources []Notification_Occurrence_Resource `json:"impactedResources,omitempty" xmlrpc:"impactedResources,omitempty"`

	// A count of a collection of users impacted by this event. Each impacted user record relates directly to a [[SoftLayer_User_Customer]].
	ImpactedUserCount *uint `json:"impactedUserCount,omitempty" xmlrpc:"impactedUserCount,omitempty"`

	// A collection of users impacted by this event. Each impacted user record relates directly to a [[SoftLayer_User_Customer]].
	ImpactedUsers []Notification_Occurrence_User `json:"impactedUsers,omitempty" xmlrpc:"impactedUsers,omitempty"`

	// Latest count of users impacted by this event.
	LastImpactedUserCount *int `json:"lastImpactedUserCount,omitempty" xmlrpc:"lastImpactedUserCount,omitempty"`

	// The last update for this event.
	LastUpdate *Notification_Occurrence_Update `json:"lastUpdate,omitempty" xmlrpc:"lastUpdate,omitempty"`

	// When this event was last updated.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The type of event such as planned or unplanned maintenance.
	NotificationOccurrenceEventType *Notification_Occurrence_Event_Type `json:"notificationOccurrenceEventType,omitempty" xmlrpc:"notificationOccurrenceEventType,omitempty"`

	// no documentation yet
	RecoveryTime *int `json:"recoveryTime,omitempty" xmlrpc:"recoveryTime,omitempty"`

	// When this event started.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// no documentation yet
	StatusCode *Notification_Occurrence_Status_Code `json:"statusCode,omitempty" xmlrpc:"statusCode,omitempty"`

	// Brief description of this event.
	Subject *string `json:"subject,omitempty" xmlrpc:"subject,omitempty"`

	// Details of this event.
	Summary *string `json:"summary,omitempty" xmlrpc:"summary,omitempty"`

	// Unique identifier for the [[SoftLayer_Ticket]] associated with this event.
	SystemTicketId *int `json:"systemTicketId,omitempty" xmlrpc:"systemTicketId,omitempty"`

	// A count of all updates for this event.
	UpdateCount *uint `json:"updateCount,omitempty" xmlrpc:"updateCount,omitempty"`

	// All updates for this event.
	Updates []Notification_Occurrence_Update `json:"updates,omitempty" xmlrpc:"updates,omitempty"`
}

// SoftLayer events can have have files attached to them by a SoftLayer employee. Attaching a file to a event is a way to provide supplementary information such as a RFO (reason for outage) document or root cause analysis. The SoftLayer_Notification_Occurrence_Event_Attachment data type models a single file attached to a event.
type Notification_Occurrence_Event_Attachment struct {
	Entity

	// The date the file was attached to the event.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The name of the file attached to the event.
	FileName *string `json:"fileName,omitempty" xmlrpc:"fileName,omitempty"`

	// The size of the file, measured in bytes.
	FileSize *string `json:"fileSize,omitempty" xmlrpc:"fileSize,omitempty"`

	// A event attachments' unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	NotificationOccurrenceEvent *Notification_Occurrence_Event `json:"notificationOccurrenceEvent,omitempty" xmlrpc:"notificationOccurrenceEvent,omitempty"`

	// The unique event identifier that the file is attached to.
	NotificationOccurrenceEventId *int `json:"notificationOccurrenceEventId,omitempty" xmlrpc:"notificationOccurrenceEventId,omitempty"`
}

// This represents the type of SoftLayer_Notification_Occurrence_Event.
type Notification_Occurrence_Event_Type struct {
	Entity

	// The friendly unique identifier for this event type.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`
}

// This type contains general information relating to any hardware or services that may be impacted by a SoftLayer_Notification_Occurrence_Event.
type Notification_Occurrence_Resource struct {
	Entity

	// no documentation yet
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// <<< EOT A label which gives some background as to what piece of
	FilterLabel *string `json:"filterLabel,omitempty" xmlrpc:"filterLabel,omitempty"`

	// The associated event.
	NotificationOccurrenceEvent *Notification_Occurrence_Event `json:"notificationOccurrenceEvent,omitempty" xmlrpc:"notificationOccurrenceEvent,omitempty"`

	// <<< EOT The unique identifier for the associated
	NotificationOccurrenceEventId *int `json:"notificationOccurrenceEventId,omitempty" xmlrpc:"notificationOccurrenceEventId,omitempty"`

	// The physical resource.
	Resource *Entity `json:"resource,omitempty" xmlrpc:"resource,omitempty"`

	// <<< EOT The unique identifier for the [[SoftLayer_Account]] associated with
	ResourceAccountId *int `json:"resourceAccountId,omitempty" xmlrpc:"resourceAccountId,omitempty"`

	// no documentation yet
	ResourceName *string `json:"resourceName,omitempty" xmlrpc:"resourceName,omitempty"`

	// <<< EOT The unique identifier for the physical resource that is associated
	ResourceTableId *int `json:"resourceTableId,omitempty" xmlrpc:"resourceTableId,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Hardware]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Hardware struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PrivateIp *string `json:"privateIp,omitempty" xmlrpc:"privateIp,omitempty"`

	// no documentation yet
	PublicIp *string `json:"publicIp,omitempty" xmlrpc:"publicIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Network_Application_Delivery_Controller]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Network_Application_Delivery_Controller struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PrivateIp *string `json:"privateIp,omitempty" xmlrpc:"privateIp,omitempty"`

	// no documentation yet
	PublicIp *string `json:"publicIp,omitempty" xmlrpc:"publicIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PublicIp *string `json:"publicIp,omitempty" xmlrpc:"publicIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Network_Storage_Iscsi_EqualLogic]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Network_Storage_Iscsi_EqualLogic struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PrivateIp *string `json:"privateIp,omitempty" xmlrpc:"privateIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Network_Storage_Iscsi_NetApp]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Network_Storage_Iscsi_NetApp struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PrivateIp *string `json:"privateIp,omitempty" xmlrpc:"privateIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Network_Storage_Lockbox]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Network_Storage_Lockbox struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PrivateIp *string `json:"privateIp,omitempty" xmlrpc:"privateIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Network_Storage_Nas]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Network_Storage_Nas struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PrivateIp *string `json:"privateIp,omitempty" xmlrpc:"privateIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Network_Storage_NetApp_Volume]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Network_Storage_NetApp_Volume struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PrivateIp *string `json:"privateIp,omitempty" xmlrpc:"privateIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// This type contains general information related to a [[SoftLayer_Virtual_Guest]] resource that is impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_Resource_Virtual struct {
	Notification_Occurrence_Resource

	// no documentation yet
	Hostname *string `json:"hostname,omitempty" xmlrpc:"hostname,omitempty"`

	// no documentation yet
	PrivateIp *string `json:"privateIp,omitempty" xmlrpc:"privateIp,omitempty"`

	// no documentation yet
	PublicIp *string `json:"publicIp,omitempty" xmlrpc:"publicIp,omitempty"`

	// no documentation yet
	ResourceType *string `json:"resourceType,omitempty" xmlrpc:"resourceType,omitempty"`
}

// no documentation yet
type Notification_Occurrence_Status_Code struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Notification_Occurrence_Update struct {
	Entity

	// no documentation yet
	Contents *string `json:"contents,omitempty" xmlrpc:"contents,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	Employee *User_Employee `json:"employee,omitempty" xmlrpc:"employee,omitempty"`

	// no documentation yet
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// no documentation yet
	NotificationOccurrenceEvent *Notification_Occurrence_Event `json:"notificationOccurrenceEvent,omitempty" xmlrpc:"notificationOccurrenceEvent,omitempty"`

	// no documentation yet
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}

// This type contains general information relating to a user that may be impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_User struct {
	Entity

	// no documentation yet
	AcknowledgedFlag *int `json:"acknowledgedFlag,omitempty" xmlrpc:"acknowledgedFlag,omitempty"`

	// no documentation yet
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of a collection of resources impacted by the associated event.
	ImpactedResourceCount *uint `json:"impactedResourceCount,omitempty" xmlrpc:"impactedResourceCount,omitempty"`

	// A collection of resources impacted by the associated event.
	ImpactedResources []Notification_Occurrence_Resource `json:"impactedResources,omitempty" xmlrpc:"impactedResources,omitempty"`

	// The associated event.
	NotificationOccurrenceEvent *Notification_Occurrence_Event `json:"notificationOccurrenceEvent,omitempty" xmlrpc:"notificationOccurrenceEvent,omitempty"`

	// The impacted user.
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// no documentation yet
	UsrRecordId *int `json:"usrRecordId,omitempty" xmlrpc:"usrRecordId,omitempty"`
}

// Retrieve details for preferences.  Preferences are used to allow the subscriber to modify their subscription in various ways.  Details such as friendly name, keyname maximum and minimum values can be retrieved.  These provide details to help configure subscriber preferences correctly.
type Notification_Preference struct {
	Entity

	// A description of what the preference is used for.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Unique identifier for the notification preference.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Name that can be used by external systems to refer to preference.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// Largest value allowed for the preference.
	MaximumValue *string `json:"maximumValue,omitempty" xmlrpc:"maximumValue,omitempty"`

	// Smallest value allowed for the preference.
	MinimumValue *string `json:"minimumValue,omitempty" xmlrpc:"minimumValue,omitempty"`

	// Friendly name for the notification.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The unit of measure used for the preference's value, minimum and maximum as well.
	Units *string `json:"units,omitempty" xmlrpc:"units,omitempty"`

	// Default value used when setting up preferences for a new subscriber.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// no documentation yet
type Notification_Subscriber struct {
	Entity

	// no documentation yet
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A count of
	DeliveryMethodCount *uint `json:"deliveryMethodCount,omitempty" xmlrpc:"deliveryMethodCount,omitempty"`

	// no documentation yet
	DeliveryMethods []Notification_Subscriber_Delivery_Method `json:"deliveryMethods,omitempty" xmlrpc:"deliveryMethods,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Notification *Notification `json:"notification,omitempty" xmlrpc:"notification,omitempty"`

	// no documentation yet
	NotificationId *int `json:"notificationId,omitempty" xmlrpc:"notificationId,omitempty"`

	// no documentation yet
	NotificationSubscriberTypeId *int `json:"notificationSubscriberTypeId,omitempty" xmlrpc:"notificationSubscriberTypeId,omitempty"`

	// no documentation yet
	NotificationSubscriberTypeResourceId *int `json:"notificationSubscriberTypeResourceId,omitempty" xmlrpc:"notificationSubscriberTypeResourceId,omitempty"`
}

// no documentation yet
type Notification_Subscriber_Customer struct {
	Notification_Subscriber

	// no documentation yet
	SubscriberRecord *User_Customer `json:"subscriberRecord,omitempty" xmlrpc:"subscriberRecord,omitempty"`
}

// Provides details for the subscriber's delivery methods.
type Notification_Subscriber_Delivery_Method struct {
	Entity

	// Indicates the subscriber's delivery method availability for notifications.
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// Date the subscriber's delivery method was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// Date the subscriber's delivery method was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	NotificationDeliveryMethod *Notification_Delivery_Method `json:"notificationDeliveryMethod,omitempty" xmlrpc:"notificationDeliveryMethod,omitempty"`

	// Identifier for the notification delivery method.
	NotificationDeliveryMethodId *int `json:"notificationDeliveryMethodId,omitempty" xmlrpc:"notificationDeliveryMethodId,omitempty"`

	// no documentation yet
	NotificationSubscriber *Notification_Subscriber `json:"notificationSubscriber,omitempty" xmlrpc:"notificationSubscriber,omitempty"`

	// Identifier for the subscriber.
	NotificationSubscriberId *int `json:"notificationSubscriberId,omitempty" xmlrpc:"notificationSubscriberId,omitempty"`
}

// A notification subscriber will have details pertaining to the subscriber's notification subscription.  You can receive details such as preferences, details of the preferences, delivery methods and the delivery methods for the subscriber.
//
// NOTE: There are preferences and delivery methods that cannot be modified.  Also, there are some subscriptions that are required.
type Notification_User_Subscriber struct {
	Entity

	// The current status of the subscription.
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// A count of the delivery methods used to send the subscribed notification.
	DeliveryMethodCount *uint `json:"deliveryMethodCount,omitempty" xmlrpc:"deliveryMethodCount,omitempty"`

	// The delivery methods used to send the subscribed notification.
	DeliveryMethods []Notification_Delivery_Method `json:"deliveryMethods,omitempty" xmlrpc:"deliveryMethods,omitempty"`

	// Unique identifier of the subscriber that will receive the alerts.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Notification subscribed to.
	Notification *Notification `json:"notification,omitempty" xmlrpc:"notification,omitempty"`

	// Unique identifier of the notification subscribed to.
	NotificationId *int `json:"notificationId,omitempty" xmlrpc:"notificationId,omitempty"`

	// A count of associated subscriber preferences used for the notification subscription. For example, preferences include number of deliveries (limit) and threshold.
	PreferenceCount *uint `json:"preferenceCount,omitempty" xmlrpc:"preferenceCount,omitempty"`

	// Associated subscriber preferences used for the notification subscription. For example, preferences include number of deliveries (limit) and threshold.
	Preferences []Notification_User_Subscriber_Preference `json:"preferences,omitempty" xmlrpc:"preferences,omitempty"`

	// A count of preference details such as description, minimum and maximum limits, default value and unit of measure.
	PreferencesDetailCount *uint `json:"preferencesDetailCount,omitempty" xmlrpc:"preferencesDetailCount,omitempty"`

	// Preference details such as description, minimum and maximum limits, default value and unit of measure.
	PreferencesDetails []Notification_Preference `json:"preferencesDetails,omitempty" xmlrpc:"preferencesDetails,omitempty"`

	// The subscriber id to resource id mapping.
	ResourceRecord *Notification_User_Subscriber_Resource `json:"resourceRecord,omitempty" xmlrpc:"resourceRecord,omitempty"`

	// User record for the subscription.
	UserRecord *User_Customer `json:"userRecord,omitempty" xmlrpc:"userRecord,omitempty"`

	// Unique identifier of the user the subscription is for.
	UserRecordId *int `json:"userRecordId,omitempty" xmlrpc:"userRecordId,omitempty"`
}

// A notification subscriber will have details pertaining to the subscriber's notification subscription.  You can receive details such as preferences, details of the preferences, delivery methods and the delivery methods for the subscriber.
//
// NOTE: There are preferences and delivery methods that cannot be modified.  Also, there are some subscriptions that are required.
type Notification_User_Subscriber_Billing struct {
	Notification_User_Subscriber
}

// Provides mapping details of how the subscriber's notification will be delivered.  This maps the subscriber's id with all the delivery method ids used to delivery the notification.
type Notification_User_Subscriber_Delivery_Method struct {
	Entity

	// Determines if the delivery method is active for the user.
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// Provides details for the method used to deliver the notification (email, sms, ticket).
	DeliveryMethod *Notification_Delivery_Method `json:"deliveryMethod,omitempty" xmlrpc:"deliveryMethod,omitempty"`

	// Unique identifier of the method used to deliver notification.
	NotificationMethodId *int `json:"notificationMethodId,omitempty" xmlrpc:"notificationMethodId,omitempty"`

	// The Subscriber information tied to the delivery method.
	NotificationUserSubscriber *Notification_User_Subscriber `json:"notificationUserSubscriber,omitempty" xmlrpc:"notificationUserSubscriber,omitempty"`

	// Unique identifier of the subscriber tied to the delivery method.
	NotificationUserSubscriberId *int `json:"notificationUserSubscriberId,omitempty" xmlrpc:"notificationUserSubscriberId,omitempty"`
}

// A notification subscriber will have details pertaining to the subscriber's notification subscription.  You can receive details such as preferences, details of the preferences, delivery methods and the delivery methods for the subscriber.
//
// NOTE: There are preferences and delivery methods that cannot be modified.  Also, there are some subscriptions that are required.
type Notification_User_Subscriber_Mobile struct {
	Notification_User_Subscriber
}

// Preferences are settings that can be modified to change the behavior of the subscription.  For example, modify the limit preference to only receive notifications 10 times instead of 1 during a billing cycle.
//
// NOTE: Some preferences have certain restrictions on values that can be set.
type Notification_User_Subscriber_Preference struct {
	Entity

	// Details such name, keyname, minimum and maximum values for the preference.
	DefaultPreference *Notification_Preference `json:"defaultPreference,omitempty" xmlrpc:"defaultPreference,omitempty"`

	// Unique identifier for the subscriber's preferences.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Unique identifier of the default preference for which the subscriber preference is based on.  For example, if no preferences are supplied during the creation of a subscriber.  The default values are pulled using this property.
	NotificationPreferenceId *int `json:"notificationPreferenceId,omitempty" xmlrpc:"notificationPreferenceId,omitempty"`

	// Details of the subscriber tied to the preference.
	NotificationUserSubscriber *Notification_User_Subscriber `json:"notificationUserSubscriber,omitempty" xmlrpc:"notificationUserSubscriber,omitempty"`

	// Unique identifier of the subscriber tied to the subscriber preference.
	NotificationUserSubscriberId *int `json:"notificationUserSubscriberId,omitempty" xmlrpc:"notificationUserSubscriberId,omitempty"`

	// The user supplied value to "override" the "default" preference's value.
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// Retrieve identifier cross-reference information.  SoftLayer_Notification_User_Subscriber_Resource provides the resource table id and subscriber id relation. The resource table id is the id of the service the subscriber receives alerts for.  This resource table id could be the unique identifier for a Storage Evault service, Global Load Balancer or CDN service.
type Notification_User_Subscriber_Resource struct {
	Entity

	// The Subscriber information tied to the resource service.
	NotificationUserSubscriber *Notification_User_Subscriber `json:"notificationUserSubscriber,omitempty" xmlrpc:"notificationUserSubscriber,omitempty"`

	// Unique identifier of the subscriber that will receive the alerts for the resource subscribed to a notification.
	NotificationUserSubscriberId *int `json:"notificationUserSubscriberId,omitempty" xmlrpc:"notificationUserSubscriberId,omitempty"`

	// Unique identifier for a SoftLayer service that is subscribed to a notification.  Currently, the SoftLayer services that can be subscribed to notifications are:
	//
	// Storage EVault CDN Global Load Balancer
	//
	//
	ResourceTableId *int `json:"resourceTableId,omitempty" xmlrpc:"resourceTableId,omitempty"`
}
