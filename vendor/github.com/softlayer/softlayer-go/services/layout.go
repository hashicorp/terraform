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

// The SoftLayer_Layout_Container contains definitions for default page layouts
type Layout_Container struct {
	Session *session.Session
	Options sl.Options
}

// GetLayoutContainerService returns an instance of the Layout_Container SoftLayer service
func GetLayoutContainerService(sess *session.Session) Layout_Container {
	return Layout_Container{Session: sess}
}

func (r Layout_Container) Id(id int) Layout_Container {
	r.Options.Id = &id
	return r
}

func (r Layout_Container) Mask(mask string) Layout_Container {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Layout_Container) Filter(filter string) Layout_Container {
	r.Options.Filter = filter
	return r
}

func (r Layout_Container) Limit(limit int) Layout_Container {
	r.Options.Limit = &limit
	return r
}

func (r Layout_Container) Offset(offset int) Layout_Container {
	r.Options.Offset = &offset
	return r
}

// Use this method to retrieve all active layout containers that can be customized.
func (r Layout_Container) GetAllObjects() (resp []datatypes.Layout_Container, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Container", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve The type of the layout container object
func (r Layout_Container) GetLayoutContainerType() (resp datatypes.Layout_Container_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Container", "getLayoutContainerType", nil, &r.Options, &resp)
	return
}

// Retrieve The layout items assigned to this layout container
func (r Layout_Container) GetLayoutItems() (resp []datatypes.Layout_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Container", "getLayoutItems", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Layout_Container) GetObject() (resp datatypes.Layout_Container, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Container", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Layout_Item contains definitions for default layout items
type Layout_Item struct {
	Session *session.Session
	Options sl.Options
}

// GetLayoutItemService returns an instance of the Layout_Item SoftLayer service
func GetLayoutItemService(sess *session.Session) Layout_Item {
	return Layout_Item{Session: sess}
}

func (r Layout_Item) Id(id int) Layout_Item {
	r.Options.Id = &id
	return r
}

func (r Layout_Item) Mask(mask string) Layout_Item {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Layout_Item) Filter(filter string) Layout_Item {
	r.Options.Filter = filter
	return r
}

func (r Layout_Item) Limit(limit int) Layout_Item {
	r.Options.Limit = &limit
	return r
}

func (r Layout_Item) Offset(offset int) Layout_Item {
	r.Options.Offset = &offset
	return r
}

// Retrieve The layout preferences assigned to this layout item
func (r Layout_Item) GetLayoutItemPreferences() (resp []datatypes.Layout_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Item", "getLayoutItemPreferences", nil, &r.Options, &resp)
	return
}

// Retrieve The type of the layout item object
func (r Layout_Item) GetLayoutItemType() (resp datatypes.Layout_Item_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Item", "getLayoutItemType", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Layout_Item) GetObject() (resp datatypes.Layout_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Item", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Layout_Profile contains the definition of the layout profile
type Layout_Profile struct {
	Session *session.Session
	Options sl.Options
}

// GetLayoutProfileService returns an instance of the Layout_Profile SoftLayer service
func GetLayoutProfileService(sess *session.Session) Layout_Profile {
	return Layout_Profile{Session: sess}
}

func (r Layout_Profile) Id(id int) Layout_Profile {
	r.Options.Id = &id
	return r
}

func (r Layout_Profile) Mask(mask string) Layout_Profile {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Layout_Profile) Filter(filter string) Layout_Profile {
	r.Options.Filter = filter
	return r
}

func (r Layout_Profile) Limit(limit int) Layout_Profile {
	r.Options.Limit = &limit
	return r
}

func (r Layout_Profile) Offset(offset int) Layout_Profile {
	r.Options.Offset = &offset
	return r
}

// This method creates a new layout profile object.
func (r Layout_Profile) CreateObject(templateObject *datatypes.Layout_Profile) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile", "createObject", params, &r.Options, &resp)
	return
}

// This method deletes an existing layout profile and associated custom preferences
func (r Layout_Profile) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method edits an existing layout profile object by passing in a modified instance of the object.
func (r Layout_Profile) EditObject(templateObject *datatypes.Layout_Profile) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Layout_Profile) GetLayoutContainers() (resp []datatypes.Layout_Container, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile", "getLayoutContainers", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Layout_Profile) GetLayoutPreferences() (resp []datatypes.Layout_Profile_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile", "getLayoutPreferences", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Layout_Profile) GetObject() (resp datatypes.Layout_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile", "getObject", nil, &r.Options, &resp)
	return
}

// This method modifies an existing associated [[SoftLayer_Layout_Profile_Preference]] object. If the preference object being modified is a default value object, a new record is created to override the default value.
//
// Only preferences that are assigned to a profile may be updated. Attempts to update a non-existent preference object will result in an exception being thrown.
func (r Layout_Profile) ModifyPreference(templateObject *datatypes.Layout_Profile_Preference) (resp datatypes.Layout_Profile_Preference, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile", "modifyPreference", params, &r.Options, &resp)
	return
}

// Using this method, multiple [[SoftLayer_Layout_Profile_Preference]] objects may be updated at once.
//
// Refer to [[SoftLayer_Layout_Profile::modifyPreference()]] for more information.
func (r Layout_Profile) ModifyPreferences(layoutPreferenceObjects []datatypes.Layout_Profile_Preference) (resp []datatypes.Layout_Profile_Preference, err error) {
	params := []interface{}{
		layoutPreferenceObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile", "modifyPreferences", params, &r.Options, &resp)
	return
}

// no documentation yet
type Layout_Profile_Containers struct {
	Session *session.Session
	Options sl.Options
}

// GetLayoutProfileContainersService returns an instance of the Layout_Profile_Containers SoftLayer service
func GetLayoutProfileContainersService(sess *session.Session) Layout_Profile_Containers {
	return Layout_Profile_Containers{Session: sess}
}

func (r Layout_Profile_Containers) Id(id int) Layout_Profile_Containers {
	r.Options.Id = &id
	return r
}

func (r Layout_Profile_Containers) Mask(mask string) Layout_Profile_Containers {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Layout_Profile_Containers) Filter(filter string) Layout_Profile_Containers {
	r.Options.Filter = filter
	return r
}

func (r Layout_Profile_Containers) Limit(limit int) Layout_Profile_Containers {
	r.Options.Limit = &limit
	return r
}

func (r Layout_Profile_Containers) Offset(offset int) Layout_Profile_Containers {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Layout_Profile_Containers) CreateObject(templateObject *datatypes.Layout_Profile_Containers) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Containers", "createObject", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Layout_Profile_Containers) EditObject(templateObject *datatypes.Layout_Profile_Containers) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Containers", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve The container to be contained
func (r Layout_Profile_Containers) GetLayoutContainerType() (resp datatypes.Layout_Container, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Containers", "getLayoutContainerType", nil, &r.Options, &resp)
	return
}

// Retrieve The profile containing this container
func (r Layout_Profile_Containers) GetLayoutProfile() (resp datatypes.Layout_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Containers", "getLayoutProfile", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Layout_Profile_Containers) GetObject() (resp datatypes.Layout_Profile_Containers, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Containers", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Layout_Profile_Customer struct {
	Session *session.Session
	Options sl.Options
}

// GetLayoutProfileCustomerService returns an instance of the Layout_Profile_Customer SoftLayer service
func GetLayoutProfileCustomerService(sess *session.Session) Layout_Profile_Customer {
	return Layout_Profile_Customer{Session: sess}
}

func (r Layout_Profile_Customer) Id(id int) Layout_Profile_Customer {
	r.Options.Id = &id
	return r
}

func (r Layout_Profile_Customer) Mask(mask string) Layout_Profile_Customer {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Layout_Profile_Customer) Filter(filter string) Layout_Profile_Customer {
	r.Options.Filter = filter
	return r
}

func (r Layout_Profile_Customer) Limit(limit int) Layout_Profile_Customer {
	r.Options.Limit = &limit
	return r
}

func (r Layout_Profile_Customer) Offset(offset int) Layout_Profile_Customer {
	r.Options.Offset = &offset
	return r
}

// This method creates a new layout profile object.
func (r Layout_Profile_Customer) CreateObject(templateObject *datatypes.Layout_Profile) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "createObject", params, &r.Options, &resp)
	return
}

// This method deletes an existing layout profile and associated custom preferences
func (r Layout_Profile_Customer) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method edits an existing layout profile object by passing in a modified instance of the object.
func (r Layout_Profile_Customer) EditObject(templateObject *datatypes.Layout_Profile) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Layout_Profile_Customer) GetLayoutContainers() (resp []datatypes.Layout_Container, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "getLayoutContainers", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Layout_Profile_Customer) GetLayoutPreferences() (resp []datatypes.Layout_Profile_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "getLayoutPreferences", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Layout_Profile_Customer) GetObject() (resp datatypes.Layout_Profile_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Layout_Profile_Customer) GetUserRecord() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "getUserRecord", nil, &r.Options, &resp)
	return
}

// This method modifies an existing associated [[SoftLayer_Layout_Profile_Preference]] object. If the preference object being modified is a default value object, a new record is created to override the default value.
//
// Only preferences that are assigned to a profile may be updated. Attempts to update a non-existent preference object will result in an exception being thrown.
func (r Layout_Profile_Customer) ModifyPreference(templateObject *datatypes.Layout_Profile_Preference) (resp datatypes.Layout_Profile_Preference, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "modifyPreference", params, &r.Options, &resp)
	return
}

// Using this method, multiple [[SoftLayer_Layout_Profile_Preference]] objects may be updated at once.
//
// Refer to [[SoftLayer_Layout_Profile::modifyPreference()]] for more information.
func (r Layout_Profile_Customer) ModifyPreferences(layoutPreferenceObjects []datatypes.Layout_Profile_Preference) (resp []datatypes.Layout_Profile_Preference, err error) {
	params := []interface{}{
		layoutPreferenceObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Customer", "modifyPreferences", params, &r.Options, &resp)
	return
}

// The SoftLayer_Layout_Profile_Preference contains definitions for layout preferences
type Layout_Profile_Preference struct {
	Session *session.Session
	Options sl.Options
}

// GetLayoutProfilePreferenceService returns an instance of the Layout_Profile_Preference SoftLayer service
func GetLayoutProfilePreferenceService(sess *session.Session) Layout_Profile_Preference {
	return Layout_Profile_Preference{Session: sess}
}

func (r Layout_Profile_Preference) Id(id int) Layout_Profile_Preference {
	r.Options.Id = &id
	return r
}

func (r Layout_Profile_Preference) Mask(mask string) Layout_Profile_Preference {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Layout_Profile_Preference) Filter(filter string) Layout_Profile_Preference {
	r.Options.Filter = filter
	return r
}

func (r Layout_Profile_Preference) Limit(limit int) Layout_Profile_Preference {
	r.Options.Limit = &limit
	return r
}

func (r Layout_Profile_Preference) Offset(offset int) Layout_Profile_Preference {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Layout_Profile_Preference) GetLayoutContainer() (resp datatypes.Layout_Container, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Preference", "getLayoutContainer", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Layout_Profile_Preference) GetLayoutItem() (resp datatypes.Layout_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Preference", "getLayoutItem", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Layout_Profile_Preference) GetLayoutPreference() (resp datatypes.Layout_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Preference", "getLayoutPreference", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Layout_Profile_Preference) GetLayoutProfile() (resp datatypes.Layout_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Preference", "getLayoutProfile", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Layout_Profile_Preference) GetObject() (resp datatypes.Layout_Profile_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Layout_Profile_Preference", "getObject", nil, &r.Options, &resp)
	return
}
