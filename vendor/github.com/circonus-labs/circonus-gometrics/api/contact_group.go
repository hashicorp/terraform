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

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// ContactGroupAlertFormats define alert formats
type ContactGroupAlertFormats struct {
	LongMessage  *string `json:"long_message"`
	LongSubject  *string `json:"long_subject"`
	LongSummary  *string `json:"long_summary"`
	ShortMessage *string `json:"short_message"`
	ShortSummary *string `json:"short_summary"`
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
	After           uint   `json:"after"`
	ContactGroupCID string `json:"contact_group"`
}

// ContactGroup defines a contactGroup
type ContactGroup struct {
	CID               string                    `json:"_cid,omitempty"`
	LastModified      uint                      `json:"_last_modified,omitempty"`
	LastModifiedBy    string                    `json:"_last_modified_by,omitempty"`
	AggregationWindow uint                      `json:"aggregation_window,omitempty"`
	AlertFormats      ContactGroupAlertFormats  `json:"alert_formats,omitempty"`
	Contacts          ContactGroupContacts      `json:"contacts,omitempty"`
	Escalations       []*ContactGroupEscalation `json:"escalations,omitempty"`
	Name              string                    `json:"name,omitempty"`
	Reminders         []uint                    `json:"reminders,omitempty"`
	Tags              []string                  `json:"tags,omitempty"`
}

// NewContactGroup returns a ContactGroup
func (a *API) NewContactGroup() *ContactGroup {
	return &ContactGroup{
		Escalations: make([]*ContactGroupEscalation, config.NumSeverityLevels),
		Reminders:   make([]uint, config.NumSeverityLevels),
		Contacts: ContactGroupContacts{
			External: []ContactGroupContactsExternal{},
			Users:    []ContactGroupContactsUser{},
		},
	}
}

// FetchContactGroup retrieves a contact group definition
func (a *API) FetchContactGroup(cid CIDType) (*ContactGroup, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid contact group CID [none]")
	}

	groupCID := string(*cid)

	matched, err := regexp.MatchString(config.ContactGroupCIDRegex, groupCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid contact group CID [%s]", groupCID)
	}

	result, err := a.Get(groupCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch contact group, received JSON: %s", string(result))
	}

	group := new(ContactGroup)
	if err := json.Unmarshal(result, group); err != nil {
		return nil, err
	}

	return group, nil
}

// FetchContactGroups retrieves all contact groups
func (a *API) FetchContactGroups() (*[]ContactGroup, error) {
	result, err := a.Get(config.ContactGroupPrefix)
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
func (a *API) UpdateContactGroup(cfg *ContactGroup) (*ContactGroup, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid contact group config [nil]")
	}

	groupCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.ContactGroupCIDRegex, groupCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid contact group CID [%s]", groupCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update contact group, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(groupCID, jsonCfg)
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
func (a *API) CreateContactGroup(cfg *ContactGroup) (*ContactGroup, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid contact group config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create contact group, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.ContactGroupPrefix, jsonCfg)
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
func (a *API) DeleteContactGroup(cfg *ContactGroup) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid contact group config [nil]")
	}
	return a.DeleteContactGroupByCID(CIDType(&cfg.CID))
}

// DeleteContactGroupByCID delete a contact group by cid
func (a *API) DeleteContactGroupByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid contact group CID [none]")
	}

	groupCID := string(*cid)

	matched, err := regexp.MatchString(config.ContactGroupCIDRegex, groupCID)
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

// SearchContactGroups returns list of contact groups matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) SearchContactGroups(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]ContactGroup, error) {
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
		Path:     config.ContactGroupPrefix,
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
