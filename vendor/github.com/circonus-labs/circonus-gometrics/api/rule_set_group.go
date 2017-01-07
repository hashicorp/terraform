// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// RuleSetGroup API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/rule_set_group

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// RuleSetGroupRule defines a rulesetGroup rule
type RuleSetGroupRule struct {
	Criteria          string `json:"criteria"`
	Severity          uint   `json:"severity"`
	Value             string `json:"value"`
	WindowingDuration uint   `json:"windowing_duration,omitempty"`
	WindowingFunction string `json:"windowing_function,omitempty"`
	Wait              uint   `json:"wait,omitempty"`
}

// RuleSetGroupFormula defines a formula for raising alerts
type RuleSetGroupFormula struct {
	Expression    string `json:"expression"`
	RaiseSeverity uint   `json:"raise_severity"`
	Wait          uint   `json:"wait"`
}

// RuleSetGroupCondition defines conditions for raising alerts
type RuleSetGroupCondition struct {
	MatchingSeverities []string `json:"matching_serverities"`
	RuleSetCID         string   `json:"rule_set"`
}

// RuleSetGroup defines a ruleset group
type RuleSetGroup struct {
	CID               string                  `json:"_cid,omitempty"`
	ContactGroups     map[uint8][]string      `json:"contact_groups"`
	Formulas          []RuleSetGroupFormula   `json:"formulas"`
	Name              string                  `json:"name"`
	RuleSetConditions []RuleSetGroupCondition `json:"rule_set_conditions"`
	Tags              []string                `json:"tags"`
}

// NewRuleSetGroup returns a new RuleSetGroup (with defaults, if applicable)
func NewRuleSetGroup() *RuleSetGroup {
	return &RuleSetGroup{}
}

// FetchRuleSetGroup retrieves rule set group with passed cid.
func (a *API) FetchRuleSetGroup(cid CIDType) (*RuleSetGroup, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid rule set group CID [none]")
	}

	groupCID := string(*cid)

	matched, err := regexp.MatchString(config.RuleSetGroupCIDRegex, groupCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid rule set group CID [%s]", groupCID)
	}

	result, err := a.Get(groupCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch rule set group, received JSON: %s", string(result))
	}

	rulesetGroup := &RuleSetGroup{}
	if err := json.Unmarshal(result, rulesetGroup); err != nil {
		return nil, err
	}

	return rulesetGroup, nil
}

// FetchRuleSetGroups retrieves all rulesetGroups
func (a *API) FetchRuleSetGroups() (*[]RuleSetGroup, error) {
	result, err := a.Get(config.RuleSetGroupPrefix)
	if err != nil {
		return nil, err
	}

	var rulesetGroups []RuleSetGroup
	if err := json.Unmarshal(result, &rulesetGroups); err != nil {
		return nil, err
	}

	return &rulesetGroups, nil
}

// UpdateRuleSetGroup update rulesetGroup definition
func (a *API) UpdateRuleSetGroup(cfg *RuleSetGroup) (*RuleSetGroup, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid rule set group config [nil]")
	}

	groupCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.RuleSetGroupCIDRegex, groupCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid rule set group CID [%s]", groupCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update rule set group, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(groupCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	groups := &RuleSetGroup{}
	if err := json.Unmarshal(result, groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// CreateRuleSetGroup create a new rulesetGroup
func (a *API) CreateRuleSetGroup(cfg *RuleSetGroup) (*RuleSetGroup, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid rule set group config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create rule set group, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.RuleSetGroupPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	group := &RuleSetGroup{}
	if err := json.Unmarshal(result, group); err != nil {
		return nil, err
	}

	return group, nil
}

// DeleteRuleSetGroup delete a rulesetGroup
func (a *API) DeleteRuleSetGroup(cfg *RuleSetGroup) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid rule set group config [nil]")
	}
	return a.DeleteRuleSetGroupByCID(CIDType(&cfg.CID))
}

// DeleteRuleSetGroupByCID delete a rulesetGroup by cid
func (a *API) DeleteRuleSetGroupByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid rule set group CID [none]")
	}

	groupCID := string(*cid)

	matched, err := regexp.MatchString(config.RuleSetGroupCIDRegex, groupCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid rule set group CID %v", groupCID)
	}

	_, err = a.Delete(groupCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchRuleSetGroups returns list of annotations matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) SearchRuleSetGroups(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]RuleSetGroup, error) {
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
		return a.FetchRuleSetGroups()
	}

	reqURL := url.URL{
		Path:     config.RuleSetGroupPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var groups []RuleSetGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, err
	}

	return &groups, nil
}
