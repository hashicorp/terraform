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

// no documentation yet
type Auxiliary_Marketing_Event struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	EnabledFlag *int `json:"enabledFlag,omitempty" xmlrpc:"enabledFlag,omitempty"`

	// no documentation yet
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// no documentation yet
	Location *string `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// no documentation yet
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`

	// no documentation yet
	Url *string `json:"url,omitempty" xmlrpc:"url,omitempty"`
}

// no documentation yet
type Auxiliary_Network_Status struct {
	Entity
}

// A SoftLayer_Auxiliary_Notification_Emergency data object represents a notification event being broadcast to the SoftLayer customer base. It is used to provide information regarding outages or current known issues.
type Auxiliary_Notification_Emergency struct {
	Entity

	// The date this event was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The device (if any) effected by this event.
	Device *string `json:"device,omitempty" xmlrpc:"device,omitempty"`

	// The duration of this event.
	Duration *string `json:"duration,omitempty" xmlrpc:"duration,omitempty"`

	// The device (if any) effected by this event.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The location effected by this event.
	Location *string `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// A message describing this event.
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// The last date this event was modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The service(s) (if any) effected by this event.
	ServicesAffected *string `json:"servicesAffected,omitempty" xmlrpc:"servicesAffected,omitempty"`

	// The signature of the SoftLayer employee department associated with this notification.
	Signature *Auxiliary_Notification_Emergency_Signature `json:"signature,omitempty" xmlrpc:"signature,omitempty"`

	// The date this event will start.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// The status of this notification.
	Status *Auxiliary_Notification_Emergency_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// Current status record for this event.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`
}

// Every SoftLayer_Auxiliary_Notification_Emergency has a signatureId that references a SoftLayer_Auxiliary_Notification_Emergency_Signature data type.  The signature is the user or group  responsible for the current event.
type Auxiliary_Notification_Emergency_Signature struct {
	Entity

	// The name or signature for the current Emergency Notification.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// Every SoftLayer_Auxiliary_Notification_Emergency has a statusId that references a SoftLayer_Auxiliary_Notification_Emergency_Status data type.  The status is used to determine the current state of the event.
type Auxiliary_Notification_Emergency_Status struct {
	Entity

	// A name describing the status of the current Emergency Notification.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Auxiliary_Press_Release struct {
	Entity

	// no documentation yet
	About []Auxiliary_Press_Release_About_Press_Release `json:"about,omitempty" xmlrpc:"about,omitempty"`

	// A count of
	AboutCount *uint `json:"aboutCount,omitempty" xmlrpc:"aboutCount,omitempty"`

	// A count of
	ContactCount *uint `json:"contactCount,omitempty" xmlrpc:"contactCount,omitempty"`

	// no documentation yet
	Contacts []Auxiliary_Press_Release_Contact_Press_Release `json:"contacts,omitempty" xmlrpc:"contacts,omitempty"`

	// A press release's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of
	MediaPartnerCount *uint `json:"mediaPartnerCount,omitempty" xmlrpc:"mediaPartnerCount,omitempty"`

	// no documentation yet
	MediaPartners []Auxiliary_Press_Release_Media_Partner_Press_Release `json:"mediaPartners,omitempty" xmlrpc:"mediaPartners,omitempty"`

	// no documentation yet
	PressReleaseContent *Auxiliary_Press_Release_Content `json:"pressReleaseContent,omitempty" xmlrpc:"pressReleaseContent,omitempty"`

	// The data a press release was published.
	PublishDate *Time `json:"publishDate,omitempty" xmlrpc:"publishDate,omitempty"`

	// A press release's location.
	ReleaseLocation *string `json:"releaseLocation,omitempty" xmlrpc:"releaseLocation,omitempty"`

	// A press release's sub-title.
	SubTitle *string `json:"subTitle,omitempty" xmlrpc:"subTitle,omitempty"`

	// A press release's title.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`

	// Whether or not a press release is highlighted on the SoftLayer Website.
	WebsiteHighlightFlag *bool `json:"websiteHighlightFlag,omitempty" xmlrpc:"websiteHighlightFlag,omitempty"`
}

// no documentation yet
type Auxiliary_Press_Release_About struct {
	Entity

	// A press release about's content.
	Content *string `json:"content,omitempty" xmlrpc:"content,omitempty"`

	// A press release about's internal
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A press release about's title.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`
}

// no documentation yet
type Auxiliary_Press_Release_About_Press_Release struct {
	Entity

	// A count of
	AboutParagraphCount *uint `json:"aboutParagraphCount,omitempty" xmlrpc:"aboutParagraphCount,omitempty"`

	// no documentation yet
	AboutParagraphs []Auxiliary_Press_Release_About `json:"aboutParagraphs,omitempty" xmlrpc:"aboutParagraphs,omitempty"`

	// A press release about cross
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A press release about's internal
	PressReleaseAboutId *int `json:"pressReleaseAboutId,omitempty" xmlrpc:"pressReleaseAboutId,omitempty"`

	// A count of
	PressReleaseCount *uint `json:"pressReleaseCount,omitempty" xmlrpc:"pressReleaseCount,omitempty"`

	// A press release internal identifier.
	PressReleaseId *int `json:"pressReleaseId,omitempty" xmlrpc:"pressReleaseId,omitempty"`

	// no documentation yet
	PressReleases []Auxiliary_Press_Release `json:"pressReleases,omitempty" xmlrpc:"pressReleases,omitempty"`

	// The number that associated an about
	SortOrder *int `json:"sortOrder,omitempty" xmlrpc:"sortOrder,omitempty"`
}

// no documentation yet
type Auxiliary_Press_Release_Contact struct {
	Entity

	// A press release contact's email
	Email *string `json:"email,omitempty" xmlrpc:"email,omitempty"`

	// A press release contact's first name.
	FirstName *string `json:"firstName,omitempty" xmlrpc:"firstName,omitempty"`

	// A press release contact's internal
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A press release contact's last name.
	LastName *string `json:"lastName,omitempty" xmlrpc:"lastName,omitempty"`

	// A press release contact's phone
	Phone *string `json:"phone,omitempty" xmlrpc:"phone,omitempty"`

	// A press release contact's
	ProfessionalTitle *string `json:"professionalTitle,omitempty" xmlrpc:"professionalTitle,omitempty"`
}

// no documentation yet
type Auxiliary_Press_Release_Contact_Press_Release struct {
	Entity

	// A count of
	ContactCount *uint `json:"contactCount,omitempty" xmlrpc:"contactCount,omitempty"`

	// no documentation yet
	Contacts []Auxiliary_Press_Release_Contact `json:"contacts,omitempty" xmlrpc:"contacts,omitempty"`

	// A press release contact cross
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A press release contact's internal
	PressReleaseContactId *int `json:"pressReleaseContactId,omitempty" xmlrpc:"pressReleaseContactId,omitempty"`

	// A count of
	PressReleaseCount *uint `json:"pressReleaseCount,omitempty" xmlrpc:"pressReleaseCount,omitempty"`

	// A press release internal identifier.
	PressReleaseId *int `json:"pressReleaseId,omitempty" xmlrpc:"pressReleaseId,omitempty"`

	// no documentation yet
	PressReleases []Auxiliary_Press_Release `json:"pressReleases,omitempty" xmlrpc:"pressReleases,omitempty"`

	// The number that associated a contact
	SortOrder *int `json:"sortOrder,omitempty" xmlrpc:"sortOrder,omitempty"`
}

// no documentation yet
type Auxiliary_Press_Release_Content struct {
	Entity

	// the id of a single press release
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// the press release id that the content
	PressReleaseId *int `json:"pressReleaseId,omitempty" xmlrpc:"pressReleaseId,omitempty"`

	// the content of a press release
	Text *string `json:"text,omitempty" xmlrpc:"text,omitempty"`
}

// no documentation yet
type Auxiliary_Press_Release_Media_Partner struct {
	Entity

	// A press release media partner's
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A press release media partner's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Auxiliary_Press_Release_Media_Partner_Press_Release struct {
	Entity

	// A press release media partner cross
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of
	MediaPartnerCount *uint `json:"mediaPartnerCount,omitempty" xmlrpc:"mediaPartnerCount,omitempty"`

	// A press release media partner's
	MediaPartnerId *int `json:"mediaPartnerId,omitempty" xmlrpc:"mediaPartnerId,omitempty"`

	// no documentation yet
	MediaPartners []Auxiliary_Press_Release_Media_Partner `json:"mediaPartners,omitempty" xmlrpc:"mediaPartners,omitempty"`

	// A count of
	PressReleaseCount *uint `json:"pressReleaseCount,omitempty" xmlrpc:"pressReleaseCount,omitempty"`

	// A press release internal identifier.
	PressReleaseId *int `json:"pressReleaseId,omitempty" xmlrpc:"pressReleaseId,omitempty"`

	// no documentation yet
	PressReleases []Auxiliary_Press_Release `json:"pressReleases,omitempty" xmlrpc:"pressReleases,omitempty"`
}

// The SoftLayer_Auxiliary_Shipping_Courier data type contains general information relating the different (major) couriers that SoftLayer may use for shipping.
type Auxiliary_Shipping_Courier struct {
	Entity

	// The unique id of the shipping courier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The unique keyname of the shipping courier.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of the shipping courier.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The url to shipping courier's website.
	Url *string `json:"url,omitempty" xmlrpc:"url,omitempty"`
}

// no documentation yet
type Auxiliary_Shipping_Courier_Type struct {
	Entity

	// no documentation yet
	Courier []Auxiliary_Shipping_Courier `json:"courier,omitempty" xmlrpc:"courier,omitempty"`

	// A count of
	CourierCount *uint `json:"courierCount,omitempty" xmlrpc:"courierCount,omitempty"`

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}
