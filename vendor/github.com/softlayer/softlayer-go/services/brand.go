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

// The SoftLayer_Brand data type contains brand information relating to the single SoftLayer customer account.
//
// SoftLayer customers are unable to change their brand information in the portal or the API.
type Brand struct {
	Session *session.Session
	Options sl.Options
}

// GetBrandService returns an instance of the Brand SoftLayer service
func GetBrandService(sess *session.Session) Brand {
	return Brand{Session: sess}
}

func (r Brand) Id(id int) Brand {
	r.Options.Id = &id
	return r
}

func (r Brand) Mask(mask string) Brand {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Brand) Filter(filter string) Brand {
	r.Options.Filter = filter
	return r
}

func (r Brand) Limit(limit int) Brand {
	r.Options.Limit = &limit
	return r
}

func (r Brand) Offset(offset int) Brand {
	r.Options.Offset = &offset
	return r
}

// Create a new customer account record.
func (r Brand) CreateCustomerAccount(account *datatypes.Account, bypassDuplicateAccountCheck *bool) (resp datatypes.Account, err error) {
	params := []interface{}{
		account,
		bypassDuplicateAccountCheck,
	}
	err = r.Session.DoRequest("SoftLayer_Brand", "createCustomerAccount", params, &r.Options, &resp)
	return
}

// Create a new brand record.
func (r Brand) CreateObject(templateObject *datatypes.Brand) (resp datatypes.Brand, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Brand", "createObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve All accounts owned by the brand.
func (r Brand) GetAllOwnedAccounts() (resp []datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getAllOwnedAccounts", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Brand) GetAllTicketSubjects(account *datatypes.Account) (resp []datatypes.Ticket_Subject, err error) {
	params := []interface{}{
		account,
	}
	err = r.Session.DoRequest("SoftLayer_Brand", "getAllTicketSubjects", params, &r.Options, &resp)
	return
}

// Retrieve This flag indicates if creation of accounts is allowed.
func (r Brand) GetAllowAccountCreationFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getAllowAccountCreationFlag", nil, &r.Options, &resp)
	return
}

// Retrieve The Product Catalog for the Brand
func (r Brand) GetCatalog() (resp datatypes.Product_Catalog, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getCatalog", nil, &r.Options, &resp)
	return
}

// Retrieve the contact information for the brand such as the corporate or support contact.  This will include the contact name, telephone number, fax number, email address, and mailing address of the contact.
func (r Brand) GetContactInformation() (resp []datatypes.Brand_Contact, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getContactInformation", nil, &r.Options, &resp)
	return
}

// Retrieve The contacts for the brand.
func (r Brand) GetContacts() (resp []datatypes.Brand_Contact, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getContacts", nil, &r.Options, &resp)
	return
}

// Retrieve This references relationship between brands, locations and countries associated with a user's account that are ineligible when ordering products. For example, the India datacenter may not be available on this brand for customers that live in Great Britain.
func (r Brand) GetCustomerCountryLocationRestrictions() (resp []datatypes.Brand_Restriction_Location_CustomerCountry, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getCustomerCountryLocationRestrictions", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetDistributor() (resp datatypes.Brand, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getDistributor", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetDistributorChildFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getDistributorChildFlag", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetDistributorFlag() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getDistributorFlag", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated hardware objects.
func (r Brand) GetHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getHardware", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetHasAgentSupportFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getHasAgentSupportFlag", nil, &r.Options, &resp)
	return
}

// Get the payment processor merchant name.
func (r Brand) GetMerchantName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getMerchantName", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Brand) GetObject() (resp datatypes.Brand, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetOpenTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getOpenTickets", nil, &r.Options, &resp)
	return
}

// Retrieve Active accounts owned by the brand.
func (r Brand) GetOwnedAccounts() (resp []datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getOwnedAccounts", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetTicketGroups() (resp []datatypes.Ticket_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getTicketGroups", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetTickets() (resp []datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getTickets", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Brand) GetToken(userId *int) (resp string, err error) {
	params := []interface{}{
		userId,
	}
	err = r.Session.DoRequest("SoftLayer_Brand", "getToken", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Brand) GetUsers() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getUsers", nil, &r.Options, &resp)
	return
}

// Retrieve An account's associated virtual guest objects.
func (r Brand) GetVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand", "getVirtualGuests", nil, &r.Options, &resp)
	return
}

// The [[SoftLayer_Brand_Restriction_Location_CustomerCountry]] data type defines the relationship between brands, locations and countries associated with a user's account that are ineligible when ordering products. For example, the India datacenter may not be available on the SoftLayer US brand for customers that live in Great Britain.
type Brand_Restriction_Location_CustomerCountry struct {
	Session *session.Session
	Options sl.Options
}

// GetBrandRestrictionLocationCustomerCountryService returns an instance of the Brand_Restriction_Location_CustomerCountry SoftLayer service
func GetBrandRestrictionLocationCustomerCountryService(sess *session.Session) Brand_Restriction_Location_CustomerCountry {
	return Brand_Restriction_Location_CustomerCountry{Session: sess}
}

func (r Brand_Restriction_Location_CustomerCountry) Id(id int) Brand_Restriction_Location_CustomerCountry {
	r.Options.Id = &id
	return r
}

func (r Brand_Restriction_Location_CustomerCountry) Mask(mask string) Brand_Restriction_Location_CustomerCountry {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Brand_Restriction_Location_CustomerCountry) Filter(filter string) Brand_Restriction_Location_CustomerCountry {
	r.Options.Filter = filter
	return r
}

func (r Brand_Restriction_Location_CustomerCountry) Limit(limit int) Brand_Restriction_Location_CustomerCountry {
	r.Options.Limit = &limit
	return r
}

func (r Brand_Restriction_Location_CustomerCountry) Offset(offset int) Brand_Restriction_Location_CustomerCountry {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Brand_Restriction_Location_CustomerCountry) GetAllObjects() (resp []datatypes.Brand_Restriction_Location_CustomerCountry, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand_Restriction_Location_CustomerCountry", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve This references the brand that has a brand-location-country restriction setup.
func (r Brand_Restriction_Location_CustomerCountry) GetBrand() (resp datatypes.Brand, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand_Restriction_Location_CustomerCountry", "getBrand", nil, &r.Options, &resp)
	return
}

// Retrieve This references the datacenter that has a brand-location-country restriction setup. For example, if a datacenter is listed with a restriction for Canada, a Canadian customer may not be eligible to order services at that location.
func (r Brand_Restriction_Location_CustomerCountry) GetLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand_Restriction_Location_CustomerCountry", "getLocation", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Brand_Restriction_Location_CustomerCountry) GetObject() (resp datatypes.Brand_Restriction_Location_CustomerCountry, err error) {
	err = r.Session.DoRequest("SoftLayer_Brand_Restriction_Location_CustomerCountry", "getObject", nil, &r.Options, &resp)
	return
}
