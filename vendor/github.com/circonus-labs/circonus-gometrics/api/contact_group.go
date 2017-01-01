// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Contact Group API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/contact_group

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
)

// ContactGroupAlertFormats define alert formats
type ContactGroupAlertFormats struct {
	LongMessage  string `json:"long_message"`
	LongSubject  string `json:"long_subject"`
	LongSummary  string `json:"long_summary"`
	ShortMessage string `json:"short_message"`
	ShortSummary string `json:"short_summary"`
}

// ContactGroupContactsExternal external contacts
type ContactGroupContactsExternal struct {
	Info   string `json:"contact_info"`
	Method string `json:"method"`
}

// ContactGroupContactsUser user contacts
type ContactGroupContactsUser struct {
	Info    string `json:"_contact_info,omitempty"`
	Method  string `json:"method"`
	UserCID string `json:"user"`
}

// ContactGroupContacts list of contacts
type ContactGroupContacts struct {
	External []ContactGroupContactsExternal `json:"external"`
	Users    []ContactGroupContactsUser     `json:"users"`
}

// ContactGroupEscalation defines escalations for severity levels
type ContactGroupEscalation struct {
	After           int    `json:"after"`
	ContactGroupCID string `json:"contact_group"`
}

// ContactGroup defines a contactGroup
type ContactGroup struct {
	CID               string                     `json:"_cid,omitempty"`
	LastModified      int                        `json:"_last_modified,omitempty"`
	LastModfiedBy     string                     `json:"_last_modified_by,omitempty"`
	AggregationWindow int                        `json:"aggregation_window"`
	AlertFormats      []ContactGroupAlertFormats `json:"alert_formats"`
	Contacts          ContactGroupContacts       `json:"contacts"`
	Escalations       []ContactGroupEscalation   `json:"escalations"`
	Name              string                     `json:"name"`
	Reminders         []int                      `json:"reminders"`
	Tags              []string                   `json:"tags"`
}

const (
	baseContactGroupPath = "/contact_group"
	contactGroupCIDRegex = "^" + baseContactGroupPath + "/[0-9]+$"
)

// FetchContactGroup retrieves a contact group definition
func (a *API) FetchContactGroup(cid CIDType) (*ContactGroup, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid contact group CID [none]")
	}

	groupCID := string(*cid)

	if matched, err := regexp.MatchString(contactGroupCIDRegex, groupCID); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid contact group CID [%s]", groupCID)
	}

	result, err := a.Get(groupCID)
	if err != nil {
		return nil, err
	}

	group := new(ContactGroup)
	if err := json.Unmarshal(result, group); err != nil {
		return nil, err
	}

	return group, nil
}

// FetchContactGroups retrieves all contact groups
func (a *API) FetchContactGroups() (*[]ContactGroup, error) {
	result, err := a.Get(baseContactGroupPath)
	if err != nil {
		return nil, err
	}

	var groups []ContactGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, err
	}

	return &groups, nil
}

// UpdateContactGroup update contact group definition
func (a *API) UpdateContactGroup(config *ContactGroup) (*ContactGroup, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid contact group config [nil]")
	}

	groupCID := string(config.CID)

	if matched, err := regexp.MatchString(contactGroupCIDRegex, groupCID); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid contact group CID [%s]", groupCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(groupCID, cfg)
	if err != nil {
		return nil, err
	}

	group := &ContactGroup{}
	if err := json.Unmarshal(result, group); err != nil {
		return nil, err
	}

	return group, nil
}

// CreateContactGroup create a new contact group
func (a *API) CreateContactGroup(config *ContactGroup) (*ContactGroup, error) {
	reqURL := url.URL{
		Path: baseContactGroupPath,
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Post(reqURL.String(), cfg)
	if err != nil {
		return nil, err
	}

	group := &ContactGroup{}
	if err := json.Unmarshal(result, group); err != nil {
		return nil, err
	}

	return group, nil
}

// DeleteContactGroup delete a contact group
func (a *API) DeleteContactGroup(config *ContactGroup) (bool, error) {
	if config == nil {
		return false, fmt.Errorf("Invalid contact group config [nil]")
	}
	return a.DeleteContactGroupByCID(CIDType(&config.CID))
}

// DeleteContactGroupByCID delete a contact group by cid
func (a *API) DeleteContactGroupByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid contact group CID [none]")
	}

	groupCID := string(*cid)

	matched, err := regexp.MatchString(contactGroupCIDRegex, groupCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid contact group CID [%s]", groupCID)
	}

	_, err = a.Delete(groupCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// ContactGroupSearch returns list of contact groups matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) ContactGroupSearch(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]ContactGroup, error) {
	q := url.Values{}

	if searchCriteria != nil && *searchCriteria != "" {
		q.Set("search", string(*searchCriteria))
	}

	if filterCriteria != nil && len(*filterCriteria) > 0 {
		for filter, criteria := range *filterCriteria {
			for _, val := range criteria {
				q.Add(filter, val)
			}
		}
	}

	if q.Encode() == "" {
		return a.FetchContactGroups()
	}

	reqURL := url.URL{
		Path:     baseContactGroupPath,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var groups []ContactGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, err
	}

	return &groups, nil
}
