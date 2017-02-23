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

// The presale event data types indicate the information regarding an individual presale event. The '''locationId''' will indicate the datacenter associated with the presale event. The '''itemId''' will indicate the product item associated with a particular presale event - however these are more rare. The '''startDate''' and '''endDate''' will provide information regarding when the presale event is available for use. At the end of the presale event, the server or services purchased will be available once approved and provisioned.
type Sales_Presale_Event struct {
	Entity

	// A flag to indicate that the presale event is currently active. A presale event is active if the current time is between the start and end dates.
	ActiveFlag *bool `json:"activeFlag,omitempty" xmlrpc:"activeFlag,omitempty"`

	// Description of the presale event.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// End date of the presale event. Orders can be approved and provisioned after this date.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// A flag to indicate that the presale event is expired. A presale event is expired if the current time is after the end date.
	ExpiredFlag *bool `json:"expiredFlag,omitempty" xmlrpc:"expiredFlag,omitempty"`

	// Presale event unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The [[SoftLayer_Product_Item]] associated with the presale event.
	Item *Product_Item `json:"item,omitempty" xmlrpc:"item,omitempty"`

	// [[SoftLayer_Product_Item]] id associated with the presale event.
	ItemId *int `json:"itemId,omitempty" xmlrpc:"itemId,omitempty"`

	// The [[SoftLayer_Location]] associated with the presale event.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// [[SoftLayer_Location]] id for the presale event.
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// A count of the orders ([[SoftLayer_Billing_Order]]) associated with this presale event that were created for the customer's account.
	OrderCount *uint `json:"orderCount,omitempty" xmlrpc:"orderCount,omitempty"`

	// The orders ([[SoftLayer_Billing_Order]]) associated with this presale event that were created for the customer's account.
	Orders []Billing_Order `json:"orders,omitempty" xmlrpc:"orders,omitempty"`

	// Start date of the presale event. Orders cannot be approved before this date.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`
}
