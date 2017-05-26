// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// User API support - Fetch, Update, and Search
// See: https://login.circonus.com/resources/api/calls/user
// Note: Create and Delete are not supported directly via the User API
// endpoint. See the Account endpoint for inviting and removing users
// from specific accounts.

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// UserContactInfo defines known contact details
type UserContactInfo struct {
	SMS  string `json:"sms,omitempty"`  // string
	XMPP string `json:"xmpp,omitempty"` // string
}

// User defines a user. See https://login.circonus.com/resources/api/calls/user for more information.
type User struct {
	CID         string          `json:"_cid,omitempty"`         // string
	ContactInfo UserContactInfo `json:"contact_info,omitempty"` // UserContactInfo
	Email       string          `json:"email"`                  // string
	Firstname   string          `json:"firstname"`              // string
	Lastname    string          `json:"lastname"`               // string
}

// FetchUser retrieves user with passed cid. Pass nil for '/user/current'.
func (a *API) FetchUser(cid CIDType) (*User, error) {
	var userCID string

	if cid == nil || *cid == "" {
		userCID = config.UserPrefix + "/current"
	} else {
		userCID = string(*cid)
	}

	matched, err := regexp.MatchString(config.UserCIDRegex, userCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid user CID [%s]", userCID)
	}

	result, err := a.Get(userCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch user, received JSON: %s", string(result))
	}

	user := new(User)
	if err := json.Unmarshal(result, user); err != nil {
		return nil, err
	}

	return user, nil
}

// FetchUsers retrieves all users available to API Token.
func (a *API) FetchUsers() (*[]User, error) {
	result, err := a.Get(config.UserPrefix)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(result, &users); err != nil {
		return nil, err
	}

	return &users, nil
}

// UpdateUser updates passed user.
func (a *API) UpdateUser(cfg *User) (*User, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid user config [nil]")
	}

	userCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.UserCIDRegex, userCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid user CID [%s]", userCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update user, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(userCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	user := &User{}
	if err := json.Unmarshal(result, user); err != nil {
		return nil, err
	}

	return user, nil
}

// SearchUsers returns users matching a filter (search queries
// are not suppoted by the user endpoint). Pass nil as filter for all
// users available to the API Token.
func (a *API) SearchUsers(filterCriteria *SearchFilterType) (*[]User, error) {
	q := url.Values{}

	if filterCriteria != nil && len(*filterCriteria) > 0 {
		for filter, criteria := range *filterCriteria {
			for _, val := range criteria {
				q.Add(filter, val)
			}
		}
	}

	if q.Encode() == "" {
		return a.FetchUsers()
	}

	reqURL := url.URL{
		Path:     config.UserPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var users []User
	if err := json.Unmarshal(result, &users); err != nil {
		return nil, err
	}

	return &users, nil
}
