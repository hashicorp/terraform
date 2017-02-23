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

// Details provided for the notification are basic.  Details such as the related preferences, name and keyname for the notification can be retrieved.  The keyname property for the notification can be used to refer to a notification when integrating into the SoftLayer Notification system.  The name property can used more for display purposes.
type Notification struct {
	Session *session.Session
	Options sl.Options
}

// GetNotificationService returns an instance of the Notification SoftLayer service
func GetNotificationService(sess *session.Session) Notification {
	return Notification{Session: sess}
}

func (r Notification) Id(id int) Notification {
	r.Options.Id = &id
	return r
}

func (r Notification) Mask(mask string) Notification {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Notification) Filter(filter string) Notification {
	r.Options.Filter = filter
	return r
}

func (r Notification) Limit(limit int) Notification {
	r.Options.Limit = &limit
	return r
}

func (r Notification) Offset(offset int) Notification {
	r.Options.Offset = &offset
	return r
}

// Use this method to retrieve all active notifications that can be subscribed to.
func (r Notification) GetAllObjects() (resp []datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification) GetObject() (resp datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The preferences related to the notification. These are preferences are configurable and optional for subscribers to use.
func (r Notification) GetPreferences() (resp []datatypes.Notification_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification", "getPreferences", nil, &r.Options, &resp)
	return
}

// Retrieve The required preferences related to the notification. While configurable, the subscriber does not have the option whether to use the preference.
func (r Notification) GetRequiredPreferences() (resp []datatypes.Notification_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification", "getRequiredPreferences", nil, &r.Options, &resp)
	return
}

// This is an extension of the SoftLayer_Notification class.  These are implementation details specific to those notifications which can be subscribed to and received on a mobile device.
type Notification_Mobile struct {
	Session *session.Session
	Options sl.Options
}

// GetNotificationMobileService returns an instance of the Notification_Mobile SoftLayer service
func GetNotificationMobileService(sess *session.Session) Notification_Mobile {
	return Notification_Mobile{Session: sess}
}

func (r Notification_Mobile) Id(id int) Notification_Mobile {
	r.Options.Id = &id
	return r
}

func (r Notification_Mobile) Mask(mask string) Notification_Mobile {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Notification_Mobile) Filter(filter string) Notification_Mobile {
	r.Options.Filter = filter
	return r
}

func (r Notification_Mobile) Limit(limit int) Notification_Mobile {
	r.Options.Limit = &limit
	return r
}

func (r Notification_Mobile) Offset(offset int) Notification_Mobile {
	r.Options.Offset = &offset
	return r
}

// Create a new subscriber for a given resource.
func (r Notification_Mobile) CreateSubscriberForMobileDevice(keyName *string, resourceTableId *int, userRecordId *int) (resp bool, err error) {
	params := []interface{}{
		keyName,
		resourceTableId,
		userRecordId,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_Mobile", "createSubscriberForMobileDevice", params, &r.Options, &resp)
	return
}

// Use this method to retrieve all active notifications that can be subscribed to.
func (r Notification_Mobile) GetAllObjects() (resp []datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Mobile", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_Mobile) GetObject() (resp datatypes.Notification_Mobile, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Mobile", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The preferences related to the notification. These are preferences are configurable and optional for subscribers to use.
func (r Notification_Mobile) GetPreferences() (resp []datatypes.Notification_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Mobile", "getPreferences", nil, &r.Options, &resp)
	return
}

// Retrieve The required preferences related to the notification. While configurable, the subscriber does not have the option whether to use the preference.
func (r Notification_Mobile) GetRequiredPreferences() (resp []datatypes.Notification_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Mobile", "getRequiredPreferences", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Notification_Occurrence_Event struct {
	Session *session.Session
	Options sl.Options
}

// GetNotificationOccurrenceEventService returns an instance of the Notification_Occurrence_Event SoftLayer service
func GetNotificationOccurrenceEventService(sess *session.Session) Notification_Occurrence_Event {
	return Notification_Occurrence_Event{Session: sess}
}

func (r Notification_Occurrence_Event) Id(id int) Notification_Occurrence_Event {
	r.Options.Id = &id
	return r
}

func (r Notification_Occurrence_Event) Mask(mask string) Notification_Occurrence_Event {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Notification_Occurrence_Event) Filter(filter string) Notification_Occurrence_Event {
	r.Options.Filter = filter
	return r
}

func (r Notification_Occurrence_Event) Limit(limit int) Notification_Occurrence_Event {
	r.Options.Limit = &limit
	return r
}

func (r Notification_Occurrence_Event) Offset(offset int) Notification_Occurrence_Event {
	r.Options.Offset = &offset
	return r
}

// <<<< EOT
func (r Notification_Occurrence_Event) AcknowledgeNotification() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "acknowledgeNotification", nil, &r.Options, &resp)
	return
}

// Retrieve Indicates whether or not this event has been acknowledged by the user.
func (r Notification_Occurrence_Event) GetAcknowledgedFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getAcknowledgedFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_Occurrence_Event) GetAllObjects() (resp []datatypes.Notification_Occurrence_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve the contents of the file attached to a SoftLayer event by it's given identifier.
func (r Notification_Occurrence_Event) GetAttachedFile(attachmentId *int) (resp []byte, err error) {
	params := []interface{}{
		attachmentId,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getAttachedFile", params, &r.Options, &resp)
	return
}

// Retrieve A collection of attachments for this event which provide supplementary information to impacted users some examples are RFO (Reason For Outage) and root cause analysis documents.
func (r Notification_Occurrence_Event) GetAttachments() (resp []datatypes.Notification_Occurrence_Event_Attachment, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getAttachments", nil, &r.Options, &resp)
	return
}

// Retrieve The first update for this event.
func (r Notification_Occurrence_Event) GetFirstUpdate() (resp datatypes.Notification_Occurrence_Update, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getFirstUpdate", nil, &r.Options, &resp)
	return
}

// This method will return the number of impacted owned accounts associated with this event for the current user.
func (r Notification_Occurrence_Event) GetImpactedAccountCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getImpactedAccountCount", nil, &r.Options, &resp)
	return
}

// Retrieve A collection of accounts impacted by this event. Each impacted account record relates directly to a [[SoftLayer_Account]].
func (r Notification_Occurrence_Event) GetImpactedAccounts() (resp []datatypes.Notification_Occurrence_Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getImpactedAccounts", nil, &r.Options, &resp)
	return
}

// This method will return the number of impacted devices associated with this event for the current user.
func (r Notification_Occurrence_Event) GetImpactedDeviceCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getImpactedDeviceCount", nil, &r.Options, &resp)
	return
}

// This method will return a collection of SoftLayer_Notification_Occurrence_Resource objects which is a listing of the current users' impacted devices that are associated with this event.
func (r Notification_Occurrence_Event) GetImpactedDevices() (resp []datatypes.Notification_Occurrence_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getImpactedDevices", nil, &r.Options, &resp)
	return
}

// Retrieve A collection of resources impacted by this event. Each record will relate to some physical resource that the user has access to such as [[SoftLayer_Hardware]] or [[SoftLayer_Virtual_Guest]].
func (r Notification_Occurrence_Event) GetImpactedResources() (resp []datatypes.Notification_Occurrence_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getImpactedResources", nil, &r.Options, &resp)
	return
}

// Retrieve A collection of users impacted by this event. Each impacted user record relates directly to a [[SoftLayer_User_Customer]].
func (r Notification_Occurrence_Event) GetImpactedUsers() (resp []datatypes.Notification_Occurrence_User, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getImpactedUsers", nil, &r.Options, &resp)
	return
}

// Retrieve The last update for this event.
func (r Notification_Occurrence_Event) GetLastUpdate() (resp datatypes.Notification_Occurrence_Update, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getLastUpdate", nil, &r.Options, &resp)
	return
}

// Retrieve The type of event such as planned or unplanned maintenance.
func (r Notification_Occurrence_Event) GetNotificationOccurrenceEventType() (resp datatypes.Notification_Occurrence_Event_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getNotificationOccurrenceEventType", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_Occurrence_Event) GetObject() (resp datatypes.Notification_Occurrence_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Notification_Occurrence_Event) GetStatusCode() (resp datatypes.Notification_Occurrence_Status_Code, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getStatusCode", nil, &r.Options, &resp)
	return
}

// Retrieve All updates for this event.
func (r Notification_Occurrence_Event) GetUpdates() (resp []datatypes.Notification_Occurrence_Update, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_Event", "getUpdates", nil, &r.Options, &resp)
	return
}

// This type contains general information relating to a user that may be impacted by a [[SoftLayer_Notification_Occurrence_Event]].
type Notification_Occurrence_User struct {
	Session *session.Session
	Options sl.Options
}

// GetNotificationOccurrenceUserService returns an instance of the Notification_Occurrence_User SoftLayer service
func GetNotificationOccurrenceUserService(sess *session.Session) Notification_Occurrence_User {
	return Notification_Occurrence_User{Session: sess}
}

func (r Notification_Occurrence_User) Id(id int) Notification_Occurrence_User {
	r.Options.Id = &id
	return r
}

func (r Notification_Occurrence_User) Mask(mask string) Notification_Occurrence_User {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Notification_Occurrence_User) Filter(filter string) Notification_Occurrence_User {
	r.Options.Filter = filter
	return r
}

func (r Notification_Occurrence_User) Limit(limit int) Notification_Occurrence_User {
	r.Options.Limit = &limit
	return r
}

func (r Notification_Occurrence_User) Offset(offset int) Notification_Occurrence_User {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Notification_Occurrence_User) Acknowledge() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_User", "acknowledge", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_Occurrence_User) GetAllObjects() (resp []datatypes.Notification_Occurrence_User, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_User", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_Occurrence_User) GetImpactedDeviceCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_User", "getImpactedDeviceCount", nil, &r.Options, &resp)
	return
}

// Retrieve A collection of resources impacted by the associated event.
func (r Notification_Occurrence_User) GetImpactedResources() (resp []datatypes.Notification_Occurrence_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_User", "getImpactedResources", nil, &r.Options, &resp)
	return
}

// Retrieve The associated event.
func (r Notification_Occurrence_User) GetNotificationOccurrenceEvent() (resp datatypes.Notification_Occurrence_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_User", "getNotificationOccurrenceEvent", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_Occurrence_User) GetObject() (resp datatypes.Notification_Occurrence_User, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_User", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The impacted user.
func (r Notification_Occurrence_User) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_Occurrence_User", "getUser", nil, &r.Options, &resp)
	return
}

// A notification subscriber will have details pertaining to the subscriber's notification subscription.  You can receive details such as preferences, details of the preferences, delivery methods and the delivery methods for the subscriber.
//
// NOTE: There are preferences and delivery methods that cannot be modified.  Also, there are some subscriptions that are required.
type Notification_User_Subscriber struct {
	Session *session.Session
	Options sl.Options
}

// GetNotificationUserSubscriberService returns an instance of the Notification_User_Subscriber SoftLayer service
func GetNotificationUserSubscriberService(sess *session.Session) Notification_User_Subscriber {
	return Notification_User_Subscriber{Session: sess}
}

func (r Notification_User_Subscriber) Id(id int) Notification_User_Subscriber {
	r.Options.Id = &id
	return r
}

func (r Notification_User_Subscriber) Mask(mask string) Notification_User_Subscriber {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Notification_User_Subscriber) Filter(filter string) Notification_User_Subscriber {
	r.Options.Filter = filter
	return r
}

func (r Notification_User_Subscriber) Limit(limit int) Notification_User_Subscriber {
	r.Options.Limit = &limit
	return r
}

func (r Notification_User_Subscriber) Offset(offset int) Notification_User_Subscriber {
	r.Options.Offset = &offset
	return r
}

// Use the method to create a new subscription for a notification.  This method is the entry method to the notification system. Certain properties are required to create a subscription while others are optional.
//
// The required property is the resourceRecord property which is type SoftLayer_Notification_User_Subscriber_Resource.  For the resourceRecord property, the only property that needs to be populated is the resourceTableId.  The resourceTableId is the unique identifier of a SoftLayer service to create the subscription for.  For example, the unique identifier of the Storage Evault service to create the subscription on.
//
// Optional properties that can be set is the preferences property.  The preference property is an array SoftLayer_Notification_User_Subscriber_Preference. By default, the system will populate the preferences with the default values if no preferences are passed in.  The preferences passed in must be the preferences related to the notification subscribing to.  The notification preferences and preference details (such as minimum and maximum values) can be retrieved using the SoftLayer_Notification service.  The properties that need to be populated for preferences are the notificationPreferenceId and value.
//
// For example to create a subscriber for a Storage EVault service to be notified 15 times during a billing cycle and to be notified when the vault usage reaches 85% of its allowed capacity use the following structure:
//
//
// *userRecordId = 1111
// *notificationId = 3
// *resourceRecord
// **resourceTableId = 1234
// *preferences[1]
// **notificationPreferenceId = 2
// **value = 85
// *preference[2]
// **notificationPreferenceId = 3
// **value = 15
//
//
func (r Notification_User_Subscriber) CreateObject(templateObject *datatypes.Notification_User_Subscriber) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "createObject", params, &r.Options, &resp)
	return
}

// The subscriber's subscription status can be "turned off" or "turned on" if the subscription is not required.
//
// Subscriber preferences may also be edited.  To edit the preferences, you must pass in the id off the preferences to edit.  Here is an example of structure to pass in.  In this example, the structure will set the subscriber status to active and the threshold preference to 90 and the limit preference to 20
//
//
// *id = 1111
// *active = 1
// *preferences[1]
// **id = 11
// **value = 90
// *preference[2]
// **id = 12
// **value = 20
func (r Notification_User_Subscriber) EditObject(templateObject *datatypes.Notification_User_Subscriber) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The delivery methods used to send the subscribed notification.
func (r Notification_User_Subscriber) GetDeliveryMethods() (resp []datatypes.Notification_Delivery_Method, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "getDeliveryMethods", nil, &r.Options, &resp)
	return
}

// Retrieve Notification subscribed to.
func (r Notification_User_Subscriber) GetNotification() (resp datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "getNotification", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_User_Subscriber) GetObject() (resp datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve Associated subscriber preferences used for the notification subscription. For example, preferences include number of deliveries (limit) and threshold.
func (r Notification_User_Subscriber) GetPreferences() (resp []datatypes.Notification_User_Subscriber_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "getPreferences", nil, &r.Options, &resp)
	return
}

// Retrieve Preference details such as description, minimum and maximum limits, default value and unit of measure.
func (r Notification_User_Subscriber) GetPreferencesDetails() (resp []datatypes.Notification_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "getPreferencesDetails", nil, &r.Options, &resp)
	return
}

// Retrieve The subscriber id to resource id mapping.
func (r Notification_User_Subscriber) GetResourceRecord() (resp datatypes.Notification_User_Subscriber_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "getResourceRecord", nil, &r.Options, &resp)
	return
}

// Retrieve User record for the subscription.
func (r Notification_User_Subscriber) GetUserRecord() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber", "getUserRecord", nil, &r.Options, &resp)
	return
}

// A notification subscriber will have details pertaining to the subscriber's notification subscription.  You can receive details such as preferences, details of the preferences, delivery methods and the delivery methods for the subscriber.
//
// NOTE: There are preferences and delivery methods that cannot be modified.  Also, there are some subscriptions that are required.
type Notification_User_Subscriber_Billing struct {
	Session *session.Session
	Options sl.Options
}

// GetNotificationUserSubscriberBillingService returns an instance of the Notification_User_Subscriber_Billing SoftLayer service
func GetNotificationUserSubscriberBillingService(sess *session.Session) Notification_User_Subscriber_Billing {
	return Notification_User_Subscriber_Billing{Session: sess}
}

func (r Notification_User_Subscriber_Billing) Id(id int) Notification_User_Subscriber_Billing {
	r.Options.Id = &id
	return r
}

func (r Notification_User_Subscriber_Billing) Mask(mask string) Notification_User_Subscriber_Billing {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Notification_User_Subscriber_Billing) Filter(filter string) Notification_User_Subscriber_Billing {
	r.Options.Filter = filter
	return r
}

func (r Notification_User_Subscriber_Billing) Limit(limit int) Notification_User_Subscriber_Billing {
	r.Options.Limit = &limit
	return r
}

func (r Notification_User_Subscriber_Billing) Offset(offset int) Notification_User_Subscriber_Billing {
	r.Options.Offset = &offset
	return r
}

// Use the method to create a new subscription for a notification.  This method is the entry method to the notification system. Certain properties are required to create a subscription while others are optional.
//
// The required property is the resourceRecord property which is type SoftLayer_Notification_User_Subscriber_Resource.  For the resourceRecord property, the only property that needs to be populated is the resourceTableId.  The resourceTableId is the unique identifier of a SoftLayer service to create the subscription for.  For example, the unique identifier of the Storage Evault service to create the subscription on.
//
// Optional properties that can be set is the preferences property.  The preference property is an array SoftLayer_Notification_User_Subscriber_Preference. By default, the system will populate the preferences with the default values if no preferences are passed in.  The preferences passed in must be the preferences related to the notification subscribing to.  The notification preferences and preference details (such as minimum and maximum values) can be retrieved using the SoftLayer_Notification service.  The properties that need to be populated for preferences are the notificationPreferenceId and value.
//
// For example to create a subscriber for a Storage EVault service to be notified 15 times during a billing cycle and to be notified when the vault usage reaches 85% of its allowed capacity use the following structure:
//
//
// *userRecordId = 1111
// *notificationId = 3
// *resourceRecord
// **resourceTableId = 1234
// *preferences[1]
// **notificationPreferenceId = 2
// **value = 85
// *preference[2]
// **notificationPreferenceId = 3
// **value = 15
//
//
func (r Notification_User_Subscriber_Billing) CreateObject(templateObject *datatypes.Notification_User_Subscriber) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "createObject", params, &r.Options, &resp)
	return
}

// The subscriber's subscription status can be "turned off" or "turned on" if the subscription is not required.
//
// Subscriber preferences may also be edited.  To edit the preferences, you must pass in the id off the preferences to edit.  Here is an example of structure to pass in.  In this example, the structure will set the subscriber status to active and the threshold preference to 90 and the limit preference to 20
//
//
// *id = 1111
// *active = 1
// *preferences[1]
// **id = 11
// **value = 90
// *preference[2]
// **id = 12
// **value = 20
func (r Notification_User_Subscriber_Billing) EditObject(templateObject *datatypes.Notification_User_Subscriber) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The delivery methods used to send the subscribed notification.
func (r Notification_User_Subscriber_Billing) GetDeliveryMethods() (resp []datatypes.Notification_Delivery_Method, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "getDeliveryMethods", nil, &r.Options, &resp)
	return
}

// Retrieve Notification subscribed to.
func (r Notification_User_Subscriber_Billing) GetNotification() (resp datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "getNotification", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_User_Subscriber_Billing) GetObject() (resp datatypes.Notification_User_Subscriber_Billing, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve Associated subscriber preferences used for the notification subscription. For example, preferences include number of deliveries (limit) and threshold.
func (r Notification_User_Subscriber_Billing) GetPreferences() (resp []datatypes.Notification_User_Subscriber_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "getPreferences", nil, &r.Options, &resp)
	return
}

// Retrieve Preference details such as description, minimum and maximum limits, default value and unit of measure.
func (r Notification_User_Subscriber_Billing) GetPreferencesDetails() (resp []datatypes.Notification_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "getPreferencesDetails", nil, &r.Options, &resp)
	return
}

// Retrieve The subscriber id to resource id mapping.
func (r Notification_User_Subscriber_Billing) GetResourceRecord() (resp datatypes.Notification_User_Subscriber_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "getResourceRecord", nil, &r.Options, &resp)
	return
}

// Retrieve User record for the subscription.
func (r Notification_User_Subscriber_Billing) GetUserRecord() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Billing", "getUserRecord", nil, &r.Options, &resp)
	return
}

// A notification subscriber will have details pertaining to the subscriber's notification subscription.  You can receive details such as preferences, details of the preferences, delivery methods and the delivery methods for the subscriber.
//
// NOTE: There are preferences and delivery methods that cannot be modified.  Also, there are some subscriptions that are required.
type Notification_User_Subscriber_Mobile struct {
	Session *session.Session
	Options sl.Options
}

// GetNotificationUserSubscriberMobileService returns an instance of the Notification_User_Subscriber_Mobile SoftLayer service
func GetNotificationUserSubscriberMobileService(sess *session.Session) Notification_User_Subscriber_Mobile {
	return Notification_User_Subscriber_Mobile{Session: sess}
}

func (r Notification_User_Subscriber_Mobile) Id(id int) Notification_User_Subscriber_Mobile {
	r.Options.Id = &id
	return r
}

func (r Notification_User_Subscriber_Mobile) Mask(mask string) Notification_User_Subscriber_Mobile {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Notification_User_Subscriber_Mobile) Filter(filter string) Notification_User_Subscriber_Mobile {
	r.Options.Filter = filter
	return r
}

func (r Notification_User_Subscriber_Mobile) Limit(limit int) Notification_User_Subscriber_Mobile {
	r.Options.Limit = &limit
	return r
}

func (r Notification_User_Subscriber_Mobile) Offset(offset int) Notification_User_Subscriber_Mobile {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Notification_User_Subscriber_Mobile) ClearSnoozeTimer() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "clearSnoozeTimer", nil, &r.Options, &resp)
	return
}

// Use the method to create a new subscription for a notification.  This method is the entry method to the notification system. Certain properties are required to create a subscription while others are optional.
//
// The required property is the resourceRecord property which is type SoftLayer_Notification_User_Subscriber_Resource.  For the resourceRecord property, the only property that needs to be populated is the resourceTableId.  The resourceTableId is the unique identifier of a SoftLayer service to create the subscription for.  For example, the unique identifier of the Storage Evault service to create the subscription on.
//
// Optional properties that can be set is the preferences property.  The preference property is an array SoftLayer_Notification_User_Subscriber_Preference. By default, the system will populate the preferences with the default values if no preferences are passed in.  The preferences passed in must be the preferences related to the notification subscribing to.  The notification preferences and preference details (such as minimum and maximum values) can be retrieved using the SoftLayer_Notification service.  The properties that need to be populated for preferences are the notificationPreferenceId and value.
//
// For example to create a subscriber for a Storage EVault service to be notified 15 times during a billing cycle and to be notified when the vault usage reaches 85% of its allowed capacity use the following structure:
//
//
// *userRecordId = 1111
// *notificationId = 3
// *resourceRecord
// **resourceTableId = 1234
// *preferences[1]
// **notificationPreferenceId = 2
// **value = 85
// *preference[2]
// **notificationPreferenceId = 3
// **value = 15
//
//
func (r Notification_User_Subscriber_Mobile) CreateObject(templateObject *datatypes.Notification_User_Subscriber) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "createObject", params, &r.Options, &resp)
	return
}

// The subscriber's subscription status can be "turned off" or "turned on" if the subscription is not required.
//
// Subscriber preferences may also be edited.  To edit the preferences, you must pass in the id off the preferences to edit.  Here is an example of structure to pass in.  In this example, the structure will set the subscriber status to active and the threshold preference to 90 and the limit preference to 20
//
//
// *id = 1111
// *active = 1
// *preferences[1]
// **id = 11
// **value = 90
// *preference[2]
// **id = 12
// **value = 20
func (r Notification_User_Subscriber_Mobile) EditObject(templateObject *datatypes.Notification_User_Subscriber) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The delivery methods used to send the subscribed notification.
func (r Notification_User_Subscriber_Mobile) GetDeliveryMethods() (resp []datatypes.Notification_Delivery_Method, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "getDeliveryMethods", nil, &r.Options, &resp)
	return
}

// Retrieve Notification subscribed to.
func (r Notification_User_Subscriber_Mobile) GetNotification() (resp datatypes.Notification, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "getNotification", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_User_Subscriber_Mobile) GetObject() (resp datatypes.Notification_User_Subscriber_Mobile, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve Associated subscriber preferences used for the notification subscription. For example, preferences include number of deliveries (limit) and threshold.
func (r Notification_User_Subscriber_Mobile) GetPreferences() (resp []datatypes.Notification_User_Subscriber_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "getPreferences", nil, &r.Options, &resp)
	return
}

// Retrieve Preference details such as description, minimum and maximum limits, default value and unit of measure.
func (r Notification_User_Subscriber_Mobile) GetPreferencesDetails() (resp []datatypes.Notification_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "getPreferencesDetails", nil, &r.Options, &resp)
	return
}

// Retrieve The subscriber id to resource id mapping.
func (r Notification_User_Subscriber_Mobile) GetResourceRecord() (resp datatypes.Notification_User_Subscriber_Resource, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "getResourceRecord", nil, &r.Options, &resp)
	return
}

// Retrieve User record for the subscription.
func (r Notification_User_Subscriber_Mobile) GetUserRecord() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "getUserRecord", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_User_Subscriber_Mobile) SetSnoozeTimer(start *int, end *int) (resp bool, err error) {
	params := []interface{}{
		start,
		end,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Mobile", "setSnoozeTimer", params, &r.Options, &resp)
	return
}

// Preferences are settings that can be modified to change the behavior of the subscription.  For example, modify the limit preference to only receive notifications 10 times instead of 1 during a billing cycle.
//
// NOTE: Some preferences have certain restrictions on values that can be set.
type Notification_User_Subscriber_Preference struct {
	Session *session.Session
	Options sl.Options
}

// GetNotificationUserSubscriberPreferenceService returns an instance of the Notification_User_Subscriber_Preference SoftLayer service
func GetNotificationUserSubscriberPreferenceService(sess *session.Session) Notification_User_Subscriber_Preference {
	return Notification_User_Subscriber_Preference{Session: sess}
}

func (r Notification_User_Subscriber_Preference) Id(id int) Notification_User_Subscriber_Preference {
	r.Options.Id = &id
	return r
}

func (r Notification_User_Subscriber_Preference) Mask(mask string) Notification_User_Subscriber_Preference {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Notification_User_Subscriber_Preference) Filter(filter string) Notification_User_Subscriber_Preference {
	r.Options.Filter = filter
	return r
}

func (r Notification_User_Subscriber_Preference) Limit(limit int) Notification_User_Subscriber_Preference {
	r.Options.Limit = &limit
	return r
}

func (r Notification_User_Subscriber_Preference) Offset(offset int) Notification_User_Subscriber_Preference {
	r.Options.Offset = &offset
	return r
}

// Use the method to create a new notification preference for a subscriber
func (r Notification_User_Subscriber_Preference) CreateObject(templateObject *datatypes.Notification_User_Subscriber_Preference) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Preference", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_User_Subscriber_Preference) EditObjects(templateObjects []datatypes.Notification_User_Subscriber_Preference) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Preference", "editObjects", params, &r.Options, &resp)
	return
}

// Retrieve Details such name, keyname, minimum and maximum values for the preference.
func (r Notification_User_Subscriber_Preference) GetDefaultPreference() (resp datatypes.Notification_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Preference", "getDefaultPreference", nil, &r.Options, &resp)
	return
}

// Retrieve Details of the subscriber tied to the preference.
func (r Notification_User_Subscriber_Preference) GetNotificationUserSubscriber() (resp datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Preference", "getNotificationUserSubscriber", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Notification_User_Subscriber_Preference) GetObject() (resp datatypes.Notification_User_Subscriber_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Notification_User_Subscriber_Preference", "getObject", nil, &r.Options, &resp)
	return
}
