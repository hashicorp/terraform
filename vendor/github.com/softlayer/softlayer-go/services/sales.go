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

// The presale event data types indicate the information regarding an individual presale event. The '''locationId''' will indicate the datacenter associated with the presale event. The '''itemId''' will indicate the product item associated with a particular presale event - however these are more rare. The '''startDate''' and '''endDate''' will provide information regarding when the presale event is available for use. At the end of the presale event, the server or services purchased will be available once approved and provisioned.
type Sales_Presale_Event struct {
	Session *session.Session
	Options sl.Options
}

// GetSalesPresaleEventService returns an instance of the Sales_Presale_Event SoftLayer service
func GetSalesPresaleEventService(sess *session.Session) Sales_Presale_Event {
	return Sales_Presale_Event{Session: sess}
}

func (r Sales_Presale_Event) Id(id int) Sales_Presale_Event {
	r.Options.Id = &id
	return r
}

func (r Sales_Presale_Event) Mask(mask string) Sales_Presale_Event {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Sales_Presale_Event) Filter(filter string) Sales_Presale_Event {
	r.Options.Filter = filter
	return r
}

func (r Sales_Presale_Event) Limit(limit int) Sales_Presale_Event {
	r.Options.Limit = &limit
	return r
}

func (r Sales_Presale_Event) Offset(offset int) Sales_Presale_Event {
	r.Options.Offset = &offset
	return r
}

// Retrieve A flag to indicate that the presale event is currently active. A presale event is active if the current time is between the start and end dates.
func (r Sales_Presale_Event) GetActiveFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Sales_Presale_Event", "getActiveFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Sales_Presale_Event) GetAllObjects() (resp []datatypes.Sales_Presale_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Sales_Presale_Event", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve A flag to indicate that the presale event is expired. A presale event is expired if the current time is after the end date.
func (r Sales_Presale_Event) GetExpiredFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Sales_Presale_Event", "getExpiredFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The [[SoftLayer_Product_Item]] associated with the presale event.
func (r Sales_Presale_Event) GetItem() (resp datatypes.Product_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Sales_Presale_Event", "getItem", nil, &r.Options, &resp)
	return
}

// Retrieve The [[SoftLayer_Location]] associated with the presale event.
func (r Sales_Presale_Event) GetLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Sales_Presale_Event", "getLocation", nil, &r.Options, &resp)
	return
}

// '''getObject''' retrieves the [[SoftLayer_Sales_Presale_Event]] object whose id number corresponds to the id number of the init parameter passed to the SoftLayer_Sales_Presale_Event service. Customers may only retrieve presale events that are currently active.
func (r Sales_Presale_Event) GetObject() (resp datatypes.Sales_Presale_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Sales_Presale_Event", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The orders ([[SoftLayer_Billing_Order]]) associated with this presale event that were created for the customer's account.
func (r Sales_Presale_Event) GetOrders() (resp []datatypes.Billing_Order, err error) {
	err = r.Session.DoRequest("SoftLayer_Sales_Presale_Event", "getOrders", nil, &r.Options, &resp)
	return
}
