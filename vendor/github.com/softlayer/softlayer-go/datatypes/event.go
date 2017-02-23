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

// The SoftLayer_Event_Log data type contains an event detail occurred upon various SoftLayer resources.
type Event_Log struct {
	Entity

	// Account id with which the event is associated
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// Event creation date in millisecond precision
	EventCreateDate *Time `json:"eventCreateDate,omitempty" xmlrpc:"eventCreateDate,omitempty"`

	// Event name such as "reboot", "cancel", "update host" and so on.
	EventName *string `json:"eventName,omitempty" xmlrpc:"eventName,omitempty"`

	// The remote IP Address that made the request
	IpAddress *string `json:"ipAddress,omitempty" xmlrpc:"ipAddress,omitempty"`

	// Label or description of the event object
	Label *string `json:"label,omitempty" xmlrpc:"label,omitempty"`

	// Meta data for an event in JSON string
	MetaData *string `json:"metaData,omitempty" xmlrpc:"metaData,omitempty"`

	// Event object id
	ObjectId *int `json:"objectId,omitempty" xmlrpc:"objectId,omitempty"`

	// Event object name such as "server", "dns" and so on.
	ObjectName *string `json:"objectName,omitempty" xmlrpc:"objectName,omitempty"`

	// A resource object that is associated with the event
	Resource *Entity `json:"resource,omitempty" xmlrpc:"resource,omitempty"`

	// A unique trace id. Multiple event can be grouped by a trace id.
	TraceId *string `json:"traceId,omitempty" xmlrpc:"traceId,omitempty"`

	// no documentation yet
	User *User_Customer `json:"user,omitempty" xmlrpc:"user,omitempty"`

	// Id of customer who initiated the event
	UserId *int `json:"userId,omitempty" xmlrpc:"userId,omitempty"`

	// Type of user that triggered the event. User type can be CUSTOMER, EMPLOYEE or SYSTEM.
	UserType *string `json:"userType,omitempty" xmlrpc:"userType,omitempty"`

	// Customer username who initiated the event
	Username *string `json:"username,omitempty" xmlrpc:"username,omitempty"`
}
