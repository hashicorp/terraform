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

// The SoftLayer_Provisioning_Hook contains all the information needed to add a hook into a server/Virtual provision and os reload.
type Provisioning_Hook struct {
	Session *session.Session
	Options sl.Options
}

// GetProvisioningHookService returns an instance of the Provisioning_Hook SoftLayer service
func GetProvisioningHookService(sess *session.Session) Provisioning_Hook {
	return Provisioning_Hook{Session: sess}
}

func (r Provisioning_Hook) Id(id int) Provisioning_Hook {
	r.Options.Id = &id
	return r
}

func (r Provisioning_Hook) Mask(mask string) Provisioning_Hook {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Provisioning_Hook) Filter(filter string) Provisioning_Hook {
	r.Options.Filter = filter
	return r
}

func (r Provisioning_Hook) Limit(limit int) Provisioning_Hook {
	r.Options.Limit = &limit
	return r
}

func (r Provisioning_Hook) Offset(offset int) Provisioning_Hook {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Provisioning_Hook) CreateObject(templateObject *datatypes.Provisioning_Hook) (resp datatypes.Provisioning_Hook, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Hook", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Provisioning_Hook) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Hook", "deleteObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Provisioning_Hook) EditObject(templateObject *datatypes.Provisioning_Hook) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Hook", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Provisioning_Hook) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Hook", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Provisioning_Hook) GetHookType() (resp datatypes.Provisioning_Hook_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Hook", "getHookType", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Provisioning_Hook) GetObject() (resp datatypes.Provisioning_Hook, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Hook", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Provisioning_Hook_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetProvisioningHookTypeService returns an instance of the Provisioning_Hook_Type SoftLayer service
func GetProvisioningHookTypeService(sess *session.Session) Provisioning_Hook_Type {
	return Provisioning_Hook_Type{Session: sess}
}

func (r Provisioning_Hook_Type) Id(id int) Provisioning_Hook_Type {
	r.Options.Id = &id
	return r
}

func (r Provisioning_Hook_Type) Mask(mask string) Provisioning_Hook_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Provisioning_Hook_Type) Filter(filter string) Provisioning_Hook_Type {
	r.Options.Filter = filter
	return r
}

func (r Provisioning_Hook_Type) Limit(limit int) Provisioning_Hook_Type {
	r.Options.Limit = &limit
	return r
}

func (r Provisioning_Hook_Type) Offset(offset int) Provisioning_Hook_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Provisioning_Hook_Type) GetAllHookTypes() (resp []datatypes.Provisioning_Hook_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Hook_Type", "getAllHookTypes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Provisioning_Hook_Type) GetObject() (resp datatypes.Provisioning_Hook_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Hook_Type", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Provisioning_Maintenance_Classification represent a maintenance type for the specific hardware maintenance desired.
type Provisioning_Maintenance_Classification struct {
	Session *session.Session
	Options sl.Options
}

// GetProvisioningMaintenanceClassificationService returns an instance of the Provisioning_Maintenance_Classification SoftLayer service
func GetProvisioningMaintenanceClassificationService(sess *session.Session) Provisioning_Maintenance_Classification {
	return Provisioning_Maintenance_Classification{Session: sess}
}

func (r Provisioning_Maintenance_Classification) Id(id int) Provisioning_Maintenance_Classification {
	r.Options.Id = &id
	return r
}

func (r Provisioning_Maintenance_Classification) Mask(mask string) Provisioning_Maintenance_Classification {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Provisioning_Maintenance_Classification) Filter(filter string) Provisioning_Maintenance_Classification {
	r.Options.Filter = filter
	return r
}

func (r Provisioning_Maintenance_Classification) Limit(limit int) Provisioning_Maintenance_Classification {
	r.Options.Limit = &limit
	return r
}

func (r Provisioning_Maintenance_Classification) Offset(offset int) Provisioning_Maintenance_Classification {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Provisioning_Maintenance_Classification) GetItemCategories() (resp []datatypes.Provisioning_Maintenance_Classification_Item_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Classification", "getItemCategories", nil, &r.Options, &resp)
	return
}

// Retrieve an array of SoftLayer_Provisioning_Maintenance_Classification data types, which contain all maintenance classifications.
func (r Provisioning_Maintenance_Classification) GetMaintenanceClassification(maintenanceClassificationId *int) (resp []datatypes.Provisioning_Maintenance_Classification, err error) {
	params := []interface{}{
		maintenanceClassificationId,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Classification", "getMaintenanceClassification", params, &r.Options, &resp)
	return
}

// Retrieve an array of SoftLayer_Provisioning_Maintenance_Classification data types, which contain all maintenance classifications.
func (r Provisioning_Maintenance_Classification) GetMaintenanceClassificationsByItemCategory() (resp []datatypes.Provisioning_Maintenance_Classification_Item_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Classification", "getMaintenanceClassificationsByItemCategory", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Provisioning_Maintenance_Classification) GetObject() (resp datatypes.Provisioning_Maintenance_Classification, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Classification", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Provisioning_Maintenance_Classification_Item_Category struct {
	Session *session.Session
	Options sl.Options
}

// GetProvisioningMaintenanceClassificationItemCategoryService returns an instance of the Provisioning_Maintenance_Classification_Item_Category SoftLayer service
func GetProvisioningMaintenanceClassificationItemCategoryService(sess *session.Session) Provisioning_Maintenance_Classification_Item_Category {
	return Provisioning_Maintenance_Classification_Item_Category{Session: sess}
}

func (r Provisioning_Maintenance_Classification_Item_Category) Id(id int) Provisioning_Maintenance_Classification_Item_Category {
	r.Options.Id = &id
	return r
}

func (r Provisioning_Maintenance_Classification_Item_Category) Mask(mask string) Provisioning_Maintenance_Classification_Item_Category {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Provisioning_Maintenance_Classification_Item_Category) Filter(filter string) Provisioning_Maintenance_Classification_Item_Category {
	r.Options.Filter = filter
	return r
}

func (r Provisioning_Maintenance_Classification_Item_Category) Limit(limit int) Provisioning_Maintenance_Classification_Item_Category {
	r.Options.Limit = &limit
	return r
}

func (r Provisioning_Maintenance_Classification_Item_Category) Offset(offset int) Provisioning_Maintenance_Classification_Item_Category {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Provisioning_Maintenance_Classification_Item_Category) GetMaintenanceClassification() (resp datatypes.Provisioning_Maintenance_Classification, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Classification_Item_Category", "getMaintenanceClassification", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Provisioning_Maintenance_Classification_Item_Category) GetObject() (resp datatypes.Provisioning_Maintenance_Classification_Item_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Classification_Item_Category", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Provisioning_Maintenance_Slots represent the available slots for a given maintenance window at a SoftLayer data center.
type Provisioning_Maintenance_Slots struct {
	Session *session.Session
	Options sl.Options
}

// GetProvisioningMaintenanceSlotsService returns an instance of the Provisioning_Maintenance_Slots SoftLayer service
func GetProvisioningMaintenanceSlotsService(sess *session.Session) Provisioning_Maintenance_Slots {
	return Provisioning_Maintenance_Slots{Session: sess}
}

func (r Provisioning_Maintenance_Slots) Id(id int) Provisioning_Maintenance_Slots {
	r.Options.Id = &id
	return r
}

func (r Provisioning_Maintenance_Slots) Mask(mask string) Provisioning_Maintenance_Slots {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Provisioning_Maintenance_Slots) Filter(filter string) Provisioning_Maintenance_Slots {
	r.Options.Filter = filter
	return r
}

func (r Provisioning_Maintenance_Slots) Limit(limit int) Provisioning_Maintenance_Slots {
	r.Options.Limit = &limit
	return r
}

func (r Provisioning_Maintenance_Slots) Offset(offset int) Provisioning_Maintenance_Slots {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Provisioning_Maintenance_Slots) GetObject() (resp datatypes.Provisioning_Maintenance_Slots, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Slots", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Provisioning_Maintenance_Ticket struct {
	Session *session.Session
	Options sl.Options
}

// GetProvisioningMaintenanceTicketService returns an instance of the Provisioning_Maintenance_Ticket SoftLayer service
func GetProvisioningMaintenanceTicketService(sess *session.Session) Provisioning_Maintenance_Ticket {
	return Provisioning_Maintenance_Ticket{Session: sess}
}

func (r Provisioning_Maintenance_Ticket) Id(id int) Provisioning_Maintenance_Ticket {
	r.Options.Id = &id
	return r
}

func (r Provisioning_Maintenance_Ticket) Mask(mask string) Provisioning_Maintenance_Ticket {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Provisioning_Maintenance_Ticket) Filter(filter string) Provisioning_Maintenance_Ticket {
	r.Options.Filter = filter
	return r
}

func (r Provisioning_Maintenance_Ticket) Limit(limit int) Provisioning_Maintenance_Ticket {
	r.Options.Limit = &limit
	return r
}

func (r Provisioning_Maintenance_Ticket) Offset(offset int) Provisioning_Maintenance_Ticket {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Provisioning_Maintenance_Ticket) GetAvailableSlots() (resp datatypes.Provisioning_Maintenance_Slots, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Ticket", "getAvailableSlots", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Provisioning_Maintenance_Ticket) GetMaintenanceClass() (resp datatypes.Provisioning_Maintenance_Classification, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Ticket", "getMaintenanceClass", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Provisioning_Maintenance_Ticket) GetObject() (resp datatypes.Provisioning_Maintenance_Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Ticket", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Provisioning_Maintenance_Ticket) GetTicket() (resp datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Ticket", "getTicket", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Provisioning_Maintenance_Window represent a time window that SoftLayer performs a hardware or software maintenance and upgrades.
type Provisioning_Maintenance_Window struct {
	Session *session.Session
	Options sl.Options
}

// GetProvisioningMaintenanceWindowService returns an instance of the Provisioning_Maintenance_Window SoftLayer service
func GetProvisioningMaintenanceWindowService(sess *session.Session) Provisioning_Maintenance_Window {
	return Provisioning_Maintenance_Window{Session: sess}
}

func (r Provisioning_Maintenance_Window) Id(id int) Provisioning_Maintenance_Window {
	r.Options.Id = &id
	return r
}

func (r Provisioning_Maintenance_Window) Mask(mask string) Provisioning_Maintenance_Window {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Provisioning_Maintenance_Window) Filter(filter string) Provisioning_Maintenance_Window {
	r.Options.Filter = filter
	return r
}

func (r Provisioning_Maintenance_Window) Limit(limit int) Provisioning_Maintenance_Window {
	r.Options.Limit = &limit
	return r
}

func (r Provisioning_Maintenance_Window) Offset(offset int) Provisioning_Maintenance_Window {
	r.Options.Offset = &offset
	return r
}

// getMaintenceWindowForTicket() returns a boolean
func (r Provisioning_Maintenance_Window) AddCustomerUpgradeWindow(customerUpgradeWindow *datatypes.Container_Provisioning_Maintenance_Window) (resp bool, err error) {
	params := []interface{}{
		customerUpgradeWindow,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Window", "addCustomerUpgradeWindow", params, &r.Options, &resp)
	return
}

// getMaintenanceClassifications() returns an object of maintenance classifications
func (r Provisioning_Maintenance_Window) GetMaintenanceClassifications() (resp []datatypes.Provisioning_Maintenance_Classification, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Window", "getMaintenanceClassifications", nil, &r.Options, &resp)
	return
}

// getMaintenanceStartEndTime() returns a specific maintenance window
func (r Provisioning_Maintenance_Window) GetMaintenanceStartEndTime(ticketId *int) (resp datatypes.Provisioning_Maintenance_Window, err error) {
	params := []interface{}{
		ticketId,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Window", "getMaintenanceStartEndTime", params, &r.Options, &resp)
	return
}

// getMaintenceWindowForTicket() returns a specific maintenance window
func (r Provisioning_Maintenance_Window) GetMaintenanceWindowForTicket(maintenanceWindowId *int) (resp []datatypes.Provisioning_Maintenance_Window, err error) {
	params := []interface{}{
		maintenanceWindowId,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Window", "getMaintenanceWindowForTicket", params, &r.Options, &resp)
	return
}

// getMaintenanceWindowTicketsByTicketId() returns a list maintenance window ticket records by ticket id
func (r Provisioning_Maintenance_Window) GetMaintenanceWindowTicketsByTicketId(ticketId *int) (resp []datatypes.Provisioning_Maintenance_Ticket, err error) {
	params := []interface{}{
		ticketId,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Window", "getMaintenanceWindowTicketsByTicketId", params, &r.Options, &resp)
	return
}

// This method returns a list of available maintenance windows
func (r Provisioning_Maintenance_Window) GetMaintenanceWindows(beginDate *datatypes.Time, endDate *datatypes.Time, locationId *int, slotsNeeded *int) (resp []datatypes.Provisioning_Maintenance_Window, err error) {
	params := []interface{}{
		beginDate,
		endDate,
		locationId,
		slotsNeeded,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Window", "getMaintenanceWindows", params, &r.Options, &resp)
	return
}

// (DEPRECATED) Use [[SoftLayer_Provisioning_Maintenance_Window::getMaintenanceWindows|getMaintenanceWindows]] method.
func (r Provisioning_Maintenance_Window) GetMaintenceWindows(beginDate *datatypes.Time, endDate *datatypes.Time, locationId *int, slotsNeeded *int) (resp []datatypes.Provisioning_Maintenance_Window, err error) {
	params := []interface{}{
		beginDate,
		endDate,
		locationId,
		slotsNeeded,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Window", "getMaintenceWindows", params, &r.Options, &resp)
	return
}

// getMaintenceWindowForTicket() returns a boolean
func (r Provisioning_Maintenance_Window) UpdateCustomerUpgradeWindow(maintenanceStartTime *datatypes.Time, newMaintenanceWindowId *int, ticketId *int) (resp bool, err error) {
	params := []interface{}{
		maintenanceStartTime,
		newMaintenanceWindowId,
		ticketId,
	}
	err = r.Session.DoRequest("SoftLayer_Provisioning_Maintenance_Window", "updateCustomerUpgradeWindow", params, &r.Options, &resp)
	return
}

// The SoftLayer_Provisioning_Version1_Transaction_Group data type contains general information relating to a single SoftLayer hardware transaction group.
//
// SoftLayer customers are unable to change their hardware transactions or the hardware transaction group.
type Provisioning_Version1_Transaction_Group struct {
	Session *session.Session
	Options sl.Options
}

// GetProvisioningVersion1TransactionGroupService returns an instance of the Provisioning_Version1_Transaction_Group SoftLayer service
func GetProvisioningVersion1TransactionGroupService(sess *session.Session) Provisioning_Version1_Transaction_Group {
	return Provisioning_Version1_Transaction_Group{Session: sess}
}

func (r Provisioning_Version1_Transaction_Group) Id(id int) Provisioning_Version1_Transaction_Group {
	r.Options.Id = &id
	return r
}

func (r Provisioning_Version1_Transaction_Group) Mask(mask string) Provisioning_Version1_Transaction_Group {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Provisioning_Version1_Transaction_Group) Filter(filter string) Provisioning_Version1_Transaction_Group {
	r.Options.Filter = filter
	return r
}

func (r Provisioning_Version1_Transaction_Group) Limit(limit int) Provisioning_Version1_Transaction_Group {
	r.Options.Limit = &limit
	return r
}

func (r Provisioning_Version1_Transaction_Group) Offset(offset int) Provisioning_Version1_Transaction_Group {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Provisioning_Version1_Transaction_Group) GetAllObjects() (resp []datatypes.Provisioning_Version1_Transaction_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Version1_Transaction_Group", "getAllObjects", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Provisioning_Version1_Transaction_Group object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Provisioning_Version1_Transaction_Group service.
func (r Provisioning_Version1_Transaction_Group) GetObject() (resp datatypes.Provisioning_Version1_Transaction_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Provisioning_Version1_Transaction_Group", "getObject", nil, &r.Options, &resp)
	return
}
