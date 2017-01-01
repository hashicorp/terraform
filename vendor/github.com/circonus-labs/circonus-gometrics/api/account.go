// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Account API support - Fetch and Update
// See: https://login.circonus.com/resources/api/calls/account
// Note: Create and Delete are not supported for Accounts via the API

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
)

// AccountLimit defines a usage limit imposed on account
type AccountLimit struct {
	Limit int    `json:"_limit,omitempty"`
	Type  string `json:"_type,omitempty"`
	Used  int    `json:"_used,omitempty"`
}

// AccountInvite defines outstanding invites
type AccountInvite struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// AccountUser defines current users
type AccountUser struct {
	Role    string `json:"role"`
	UserCID string `json:"user"`
}

// Account definition
type Account struct {
	CID           string          `json:"_cid,omitempty"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	OwnerCID      string          `json:"_owner,omitempty"`
	Address1      string          `json:"address1"`
	Address2      string          `json:"address2"`
	CCEmail       string          `json:"cc_email"`
	City          string          `json:"city"`
	StateProv     string          `json:"state_prov"`
	Country       string          `json:"country_code"`
	Timezone      string          `json:"timezone"`
	Invites       []AccountInvite `json:"invites"`
	Users         []AccountUser   `json:"users"`
	ContactGroups []string        `json:"_contact_groups,omitempty"`
	UIBaseURL     string          `json:"_ui_base_url,omitempty"`
	Usage         []AccountLimit  `json:"_usage,omitempty"`
}

const (
	baseAccountPath = "/account"
	accountCIDRegex = "^" + baseAccountPath + "/([0-9]+|current)$"
)

// FetchAccount retrieves an account definition
func (a *API) FetchAccount(cid CIDType) (*Account, error) {
	var accountCID string

	if cid == nil || *cid == "" {
		accountCID = baseAccountPath + "/current"
	} else {
		accountCID = string(*cid)
	}

	if matched, err := regexp.MatchString(accountCIDRegex, accountCID); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid account CID [%s]", accountCID)
	}

	result, err := a.Get(accountCID)
	if err != nil {
		return nil, err
	}

	account := new(Account)
	if err := json.Unmarshal(result, account); err != nil {
		return nil, err
	}

	return account, nil
}

// UpdateAccount update account configuration
func (a *API) UpdateAccount(config *Account) (*Account, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid account config [nil]")
	}

	accountCID := string(config.CID)

	if matched, err := regexp.MatchString(accountCIDRegex, accountCID); err != nil {
		return nil, err
	} else if !matched {
		return nil, fmt.Errorf("Invalid account CID [%s]", accountCID)
	}

	reqURL := url.URL{
		Path: accountCID,
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(reqURL.String(), cfg)
	if err != nil {
		return nil, err
	}

	account := &Account{}
	if err := json.Unmarshal(result, account); err != nil {
		return nil, err
	}

	return account, nil
}
