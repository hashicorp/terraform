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

// no documentation yet
type Auxiliary_Marketing_Event struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryMarketingEventService returns an instance of the Auxiliary_Marketing_Event SoftLayer service
func GetAuxiliaryMarketingEventService(sess *session.Session) Auxiliary_Marketing_Event {
	return Auxiliary_Marketing_Event{Session: sess}
}

func (r Auxiliary_Marketing_Event) Id(id int) Auxiliary_Marketing_Event {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Marketing_Event) Mask(mask string) Auxiliary_Marketing_Event {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Marketing_Event) Filter(filter string) Auxiliary_Marketing_Event {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Marketing_Event) Limit(limit int) Auxiliary_Marketing_Event {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Marketing_Event) Offset(offset int) Auxiliary_Marketing_Event {
	r.Options.Offset = &offset
	return r
}

// This method will return a collection of SoftLayer_Auxiliary_Marketing_Event objects ordered in ascending order by start date.
func (r Auxiliary_Marketing_Event) GetMarketingEvents() (resp []datatypes.Auxiliary_Marketing_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Marketing_Event", "getMarketingEvents", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Auxiliary_Marketing_Event) GetObject() (resp datatypes.Auxiliary_Marketing_Event, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Marketing_Event", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Network_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryNetworkStatusService returns an instance of the Auxiliary_Network_Status SoftLayer service
func GetAuxiliaryNetworkStatusService(sess *session.Session) Auxiliary_Network_Status {
	return Auxiliary_Network_Status{Session: sess}
}

func (r Auxiliary_Network_Status) Id(id int) Auxiliary_Network_Status {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Network_Status) Mask(mask string) Auxiliary_Network_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Network_Status) Filter(filter string) Auxiliary_Network_Status {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Network_Status) Limit(limit int) Auxiliary_Network_Status {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Network_Status) Offset(offset int) Auxiliary_Network_Status {
	r.Options.Offset = &offset
	return r
}

// Return the current network status of and latency information for a given target from numerous points around the world. Valid Targets:
// * ALL
// * NETWORK_DALLAS
// * NETWORK_SEATTLE
// * NETWORK_PUBLIC
// * NETWORK_PUBLIC_DALLAS
// * NETWORK_PUBLIC_SEATTLE
// * NETWORK_PUBLIC_WDC
// * NETWORK_PRIVATE
// * NETWORK_PRIVATE_DALLAS
// * NETWORK_PRIVATE_SEATTLE
// * NETWORK_PRIVATE_WDC
func (r Auxiliary_Network_Status) GetNetworkStatus(target *string) (resp []datatypes.Container_Auxiliary_Network_Status_Reading, err error) {
	params := []interface{}{
		target,
	}
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Network_Status", "getNetworkStatus", params, &r.Options, &resp)
	return
}

// A SoftLayer_Auxiliary_Notification_Emergency data object represents a notification event being broadcast to the SoftLayer customer base. It is used to provide information regarding outages or current known issues.
type Auxiliary_Notification_Emergency struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryNotificationEmergencyService returns an instance of the Auxiliary_Notification_Emergency SoftLayer service
func GetAuxiliaryNotificationEmergencyService(sess *session.Session) Auxiliary_Notification_Emergency {
	return Auxiliary_Notification_Emergency{Session: sess}
}

func (r Auxiliary_Notification_Emergency) Id(id int) Auxiliary_Notification_Emergency {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Notification_Emergency) Mask(mask string) Auxiliary_Notification_Emergency {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Notification_Emergency) Filter(filter string) Auxiliary_Notification_Emergency {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Notification_Emergency) Limit(limit int) Auxiliary_Notification_Emergency {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Notification_Emergency) Offset(offset int) Auxiliary_Notification_Emergency {
	r.Options.Offset = &offset
	return r
}

// Retrieve an array of SoftLayer_Auxiliary_Notification_Emergency data types, which contain all notification events regardless of status.
func (r Auxiliary_Notification_Emergency) GetAllObjects() (resp []datatypes.Auxiliary_Notification_Emergency, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Notification_Emergency", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve an array of SoftLayer_Auxiliary_Notification_Emergency data types, which contain all current notification events.
func (r Auxiliary_Notification_Emergency) GetCurrentNotifications() (resp []datatypes.Auxiliary_Notification_Emergency, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Notification_Emergency", "getCurrentNotifications", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Auxiliary_Notification_Emergency object, it can be used to check for current notifications being broadcast by SoftLayer.
func (r Auxiliary_Notification_Emergency) GetObject() (resp datatypes.Auxiliary_Notification_Emergency, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Notification_Emergency", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The signature of the SoftLayer employee department associated with this notification.
func (r Auxiliary_Notification_Emergency) GetSignature() (resp datatypes.Auxiliary_Notification_Emergency_Signature, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Notification_Emergency", "getSignature", nil, &r.Options, &resp)
	return
}

// Retrieve The status of this notification.
func (r Auxiliary_Notification_Emergency) GetStatus() (resp datatypes.Auxiliary_Notification_Emergency_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Notification_Emergency", "getStatus", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Press_Release struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryPressReleaseService returns an instance of the Auxiliary_Press_Release SoftLayer service
func GetAuxiliaryPressReleaseService(sess *session.Session) Auxiliary_Press_Release {
	return Auxiliary_Press_Release{Session: sess}
}

func (r Auxiliary_Press_Release) Id(id int) Auxiliary_Press_Release {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Press_Release) Mask(mask string) Auxiliary_Press_Release {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Press_Release) Filter(filter string) Auxiliary_Press_Release {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Press_Release) Limit(limit int) Auxiliary_Press_Release {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Press_Release) Offset(offset int) Auxiliary_Press_Release {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Auxiliary_Press_Release) GetAbout() (resp []datatypes.Auxiliary_Press_Release_About_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getAbout", nil, &r.Options, &resp)
	return
}

// Retrieve an array of SoftLayer_Auxiliary_Press_Release data types, which contain all press releases.
func (r Auxiliary_Press_Release) GetAllObjects() (resp []datatypes.Auxiliary_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Auxiliary_Press_Release) GetContacts() (resp []datatypes.Auxiliary_Press_Release_Contact_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getContacts", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Auxiliary_Press_Release) GetMediaPartners() (resp []datatypes.Auxiliary_Press_Release_Media_Partner_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getMediaPartners", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Auxiliary_Press_Release object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Auxiliary_Press_Release service.
func (r Auxiliary_Press_Release) GetObject() (resp datatypes.Auxiliary_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Auxiliary_Press_Release) GetPressReleaseContent() (resp datatypes.Auxiliary_Press_Release_Content, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getPressReleaseContent", nil, &r.Options, &resp)
	return
}

// Retrieve an array of SoftLayer_Auxiliary_Press_Release data types, which contain all press releases.
func (r Auxiliary_Press_Release) GetRenderedPressRelease() (resp []datatypes.Auxiliary_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getRenderedPressRelease", nil, &r.Options, &resp)
	return
}

// Retrieve an array of SoftLayer_Auxiliary_Press_Release data types, which contain all press releases for a given year and or result limit.
func (r Auxiliary_Press_Release) GetRenderedPressReleases(resultLimit *string, year *string) (resp []datatypes.Auxiliary_Press_Release, err error) {
	params := []interface{}{
		resultLimit,
		year,
	}
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getRenderedPressReleases", params, &r.Options, &resp)
	return
}

// Retrieve an array of SoftLayer_Auxiliary_Press_Release data types, which have the website highlight flag set.
func (r Auxiliary_Press_Release) GetWebsiteHighlightPressReleases() (resp []datatypes.Auxiliary_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release", "getWebsiteHighlightPressReleases", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Press_Release_About struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryPressReleaseAboutService returns an instance of the Auxiliary_Press_Release_About SoftLayer service
func GetAuxiliaryPressReleaseAboutService(sess *session.Session) Auxiliary_Press_Release_About {
	return Auxiliary_Press_Release_About{Session: sess}
}

func (r Auxiliary_Press_Release_About) Id(id int) Auxiliary_Press_Release_About {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Press_Release_About) Mask(mask string) Auxiliary_Press_Release_About {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Press_Release_About) Filter(filter string) Auxiliary_Press_Release_About {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Press_Release_About) Limit(limit int) Auxiliary_Press_Release_About {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Press_Release_About) Offset(offset int) Auxiliary_Press_Release_About {
	r.Options.Offset = &offset
	return r
}

// getObject retrieves the SoftLayer_Auxiliary_Press_Release_About object whose about id number corresponds to the ID number of the init parameter passed to the SoftLayer_Auxiliary_Press_Release service.
func (r Auxiliary_Press_Release_About) GetObject() (resp datatypes.Auxiliary_Press_Release_About, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_About", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Press_Release_About_Press_Release struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryPressReleaseAboutPressReleaseService returns an instance of the Auxiliary_Press_Release_About_Press_Release SoftLayer service
func GetAuxiliaryPressReleaseAboutPressReleaseService(sess *session.Session) Auxiliary_Press_Release_About_Press_Release {
	return Auxiliary_Press_Release_About_Press_Release{Session: sess}
}

func (r Auxiliary_Press_Release_About_Press_Release) Id(id int) Auxiliary_Press_Release_About_Press_Release {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Press_Release_About_Press_Release) Mask(mask string) Auxiliary_Press_Release_About_Press_Release {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Press_Release_About_Press_Release) Filter(filter string) Auxiliary_Press_Release_About_Press_Release {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Press_Release_About_Press_Release) Limit(limit int) Auxiliary_Press_Release_About_Press_Release {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Press_Release_About_Press_Release) Offset(offset int) Auxiliary_Press_Release_About_Press_Release {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Auxiliary_Press_Release_About_Press_Release) GetAboutParagraphs() (resp []datatypes.Auxiliary_Press_Release_About, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_About_Press_Release", "getAboutParagraphs", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Auxiliary_Press_Release_About_Press_Release object whose contact id number corresponds to the ID number of the init parameter passed to the SoftLayer_Auxiliary_Press_Release service.
func (r Auxiliary_Press_Release_About_Press_Release) GetObject() (resp datatypes.Auxiliary_Press_Release_About_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_About_Press_Release", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Auxiliary_Press_Release_About_Press_Release) GetPressReleases() (resp []datatypes.Auxiliary_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_About_Press_Release", "getPressReleases", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Press_Release_Contact struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryPressReleaseContactService returns an instance of the Auxiliary_Press_Release_Contact SoftLayer service
func GetAuxiliaryPressReleaseContactService(sess *session.Session) Auxiliary_Press_Release_Contact {
	return Auxiliary_Press_Release_Contact{Session: sess}
}

func (r Auxiliary_Press_Release_Contact) Id(id int) Auxiliary_Press_Release_Contact {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Press_Release_Contact) Mask(mask string) Auxiliary_Press_Release_Contact {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Press_Release_Contact) Filter(filter string) Auxiliary_Press_Release_Contact {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Press_Release_Contact) Limit(limit int) Auxiliary_Press_Release_Contact {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Press_Release_Contact) Offset(offset int) Auxiliary_Press_Release_Contact {
	r.Options.Offset = &offset
	return r
}

// getObject retrieves the SoftLayer_Auxiliary_Press_Release_Contact object whose contact id number corresponds to the ID number of the init parameter passed to the SoftLayer_Auxiliary_Press_Release service.
func (r Auxiliary_Press_Release_Contact) GetObject() (resp datatypes.Auxiliary_Press_Release_Contact, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Contact", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Press_Release_Contact_Press_Release struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryPressReleaseContactPressReleaseService returns an instance of the Auxiliary_Press_Release_Contact_Press_Release SoftLayer service
func GetAuxiliaryPressReleaseContactPressReleaseService(sess *session.Session) Auxiliary_Press_Release_Contact_Press_Release {
	return Auxiliary_Press_Release_Contact_Press_Release{Session: sess}
}

func (r Auxiliary_Press_Release_Contact_Press_Release) Id(id int) Auxiliary_Press_Release_Contact_Press_Release {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Press_Release_Contact_Press_Release) Mask(mask string) Auxiliary_Press_Release_Contact_Press_Release {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Press_Release_Contact_Press_Release) Filter(filter string) Auxiliary_Press_Release_Contact_Press_Release {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Press_Release_Contact_Press_Release) Limit(limit int) Auxiliary_Press_Release_Contact_Press_Release {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Press_Release_Contact_Press_Release) Offset(offset int) Auxiliary_Press_Release_Contact_Press_Release {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Auxiliary_Press_Release_Contact_Press_Release) GetContacts() (resp []datatypes.Auxiliary_Press_Release_Contact, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Contact_Press_Release", "getContacts", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Auxiliary_Press_Release_Contact object whose contact id number corresponds to the ID number of the init parameter passed to the SoftLayer_Auxiliary_Press_Release service.
func (r Auxiliary_Press_Release_Contact_Press_Release) GetObject() (resp datatypes.Auxiliary_Press_Release_Contact_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Contact_Press_Release", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Auxiliary_Press_Release_Contact_Press_Release) GetPressReleases() (resp []datatypes.Auxiliary_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Contact_Press_Release", "getPressReleases", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Press_Release_Content struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryPressReleaseContentService returns an instance of the Auxiliary_Press_Release_Content SoftLayer service
func GetAuxiliaryPressReleaseContentService(sess *session.Session) Auxiliary_Press_Release_Content {
	return Auxiliary_Press_Release_Content{Session: sess}
}

func (r Auxiliary_Press_Release_Content) Id(id int) Auxiliary_Press_Release_Content {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Press_Release_Content) Mask(mask string) Auxiliary_Press_Release_Content {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Press_Release_Content) Filter(filter string) Auxiliary_Press_Release_Content {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Press_Release_Content) Limit(limit int) Auxiliary_Press_Release_Content {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Press_Release_Content) Offset(offset int) Auxiliary_Press_Release_Content {
	r.Options.Offset = &offset
	return r
}

// getObject retrieves the SoftLayer_Auxiliary_Press_Release_Content object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Auxiliary_Press_Release service.
func (r Auxiliary_Press_Release_Content) GetObject() (resp datatypes.Auxiliary_Press_Release_Content, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Content", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Press_Release_Media_Partner struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryPressReleaseMediaPartnerService returns an instance of the Auxiliary_Press_Release_Media_Partner SoftLayer service
func GetAuxiliaryPressReleaseMediaPartnerService(sess *session.Session) Auxiliary_Press_Release_Media_Partner {
	return Auxiliary_Press_Release_Media_Partner{Session: sess}
}

func (r Auxiliary_Press_Release_Media_Partner) Id(id int) Auxiliary_Press_Release_Media_Partner {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Press_Release_Media_Partner) Mask(mask string) Auxiliary_Press_Release_Media_Partner {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Press_Release_Media_Partner) Filter(filter string) Auxiliary_Press_Release_Media_Partner {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Press_Release_Media_Partner) Limit(limit int) Auxiliary_Press_Release_Media_Partner {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Press_Release_Media_Partner) Offset(offset int) Auxiliary_Press_Release_Media_Partner {
	r.Options.Offset = &offset
	return r
}

// getObject retrieves the SoftLayer_Auxiliary_Press_Release_Contact object whose contact id number corresponds to the ID number of the init parameter passed to the SoftLayer_Auxiliary_Press_Release service.
func (r Auxiliary_Press_Release_Media_Partner) GetObject() (resp datatypes.Auxiliary_Press_Release_Media_Partner, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Media_Partner", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Press_Release_Media_Partner_Press_Release struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryPressReleaseMediaPartnerPressReleaseService returns an instance of the Auxiliary_Press_Release_Media_Partner_Press_Release SoftLayer service
func GetAuxiliaryPressReleaseMediaPartnerPressReleaseService(sess *session.Session) Auxiliary_Press_Release_Media_Partner_Press_Release {
	return Auxiliary_Press_Release_Media_Partner_Press_Release{Session: sess}
}

func (r Auxiliary_Press_Release_Media_Partner_Press_Release) Id(id int) Auxiliary_Press_Release_Media_Partner_Press_Release {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Press_Release_Media_Partner_Press_Release) Mask(mask string) Auxiliary_Press_Release_Media_Partner_Press_Release {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Press_Release_Media_Partner_Press_Release) Filter(filter string) Auxiliary_Press_Release_Media_Partner_Press_Release {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Press_Release_Media_Partner_Press_Release) Limit(limit int) Auxiliary_Press_Release_Media_Partner_Press_Release {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Press_Release_Media_Partner_Press_Release) Offset(offset int) Auxiliary_Press_Release_Media_Partner_Press_Release {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Auxiliary_Press_Release_Media_Partner_Press_Release) GetMediaPartners() (resp []datatypes.Auxiliary_Press_Release_Media_Partner, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Media_Partner_Press_Release", "getMediaPartners", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Auxiliary_Press_Release_Media_Partner_Press_Release object whose media partner id number corresponds to the ID number of the init parameter passed to the SoftLayer_Auxiliary_Press_Release service.
func (r Auxiliary_Press_Release_Media_Partner_Press_Release) GetObject() (resp datatypes.Auxiliary_Press_Release_Media_Partner_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Media_Partner_Press_Release", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Auxiliary_Press_Release_Media_Partner_Press_Release) GetPressReleases() (resp []datatypes.Auxiliary_Press_Release, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Press_Release_Media_Partner_Press_Release", "getPressReleases", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Auxiliary_Shipping_Courier_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetAuxiliaryShippingCourierTypeService returns an instance of the Auxiliary_Shipping_Courier_Type SoftLayer service
func GetAuxiliaryShippingCourierTypeService(sess *session.Session) Auxiliary_Shipping_Courier_Type {
	return Auxiliary_Shipping_Courier_Type{Session: sess}
}

func (r Auxiliary_Shipping_Courier_Type) Id(id int) Auxiliary_Shipping_Courier_Type {
	r.Options.Id = &id
	return r
}

func (r Auxiliary_Shipping_Courier_Type) Mask(mask string) Auxiliary_Shipping_Courier_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Auxiliary_Shipping_Courier_Type) Filter(filter string) Auxiliary_Shipping_Courier_Type {
	r.Options.Filter = filter
	return r
}

func (r Auxiliary_Shipping_Courier_Type) Limit(limit int) Auxiliary_Shipping_Courier_Type {
	r.Options.Limit = &limit
	return r
}

func (r Auxiliary_Shipping_Courier_Type) Offset(offset int) Auxiliary_Shipping_Courier_Type {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Auxiliary_Shipping_Courier_Type) GetCourier() (resp []datatypes.Auxiliary_Shipping_Courier, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Shipping_Courier_Type", "getCourier", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Auxiliary_Shipping_Courier_Type) GetObject() (resp datatypes.Auxiliary_Shipping_Courier_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Shipping_Courier_Type", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Auxiliary_Shipping_Courier_Type) GetTypeByKeyName(keyName *string) (resp datatypes.Auxiliary_Shipping_Courier_Type, err error) {
	params := []interface{}{
		keyName,
	}
	err = r.Session.DoRequest("SoftLayer_Auxiliary_Shipping_Courier_Type", "getTypeByKeyName", params, &r.Options, &resp)
	return
}
