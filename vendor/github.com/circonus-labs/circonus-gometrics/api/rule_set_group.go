// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// RulesetGroup API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/rule_set_group

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// RulesetGroupRule defines a rulesetGroup rule
type RulesetGroupRule struct {
	Criteria          string `json:"criteria"`
	Severity          uint   `json:"severity"`
	Value             string `json:"value"`
	WindowingDuration uint   `json:"windowing_duration,omitempty"`
	WindowingFunction string `json:"windowing_function,omitempty"`
	Wait              uint   `json:"wait,omitempty"`
}

// RulesetGroupFormula defines a formula for raising alerts
type RulesetGroupFormula struct {
	Expression    string `json:"expression"`
	RaiseSeverity uint   `json:"raise_severity"`
	Wait          uint   `json:"wait"`
}

// RulesetGroupCondition defines conditions for raising alerts
type RulesetGroupCondition struct {
	MatchingSeverities []string `json:"matching_serverities"`
	RulesetCID         string   `json:"rule_set"`
}

// RulesetGroup defines a ruleset group
type RulesetGroup struct {
	CID               string                  `json:"_cid,omitempty"`
	ContactGroups     map[uint8][]string      `json:"contact_groups"`
	Formulas          []RulesetGroupFormula   `json:"formulas"`
	Name              string                  `json:"name"`
	RulesetConditions []RulesetGroupCondition `json:"rule_set_conditions"`
	Tags              []string                `json:"tags"`
}

// FetchRulesetGroup retrieves a rulesetGroup definition
func (a *API) FetchRulesetGroup(cid CIDType) (*RulesetGroup, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid rule set group CID [none]")
	}

	groupCID := string(*cid)

	matched, err := regexp.MatchString(config.RulesetGroupCIDRegex, groupCID)
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

	rulesetGroup := &RulesetGroup{}
	if err := json.Unmarshal(result, rulesetGroup); err != nil {
		return nil, err
	}

	return rulesetGroup, nil
}

// FetchRulesetGroups retrieves all rulesetGroups
func (a *API) FetchRulesetGroups() (*[]RulesetGroup, error) {
	result, err := a.Get(config.RuleSetGroupPrefix)
	if err != nil {
		return nil, err
	}

	var rulesetGroups []RulesetGroup
	if err := json.Unmarshal(result, &rulesetGroups); err != nil {
		return nil, err
	}

	return &rulesetGroups, nil
}

// UpdateRulesetGroup update rulesetGroup definition
func (a *API) UpdateRulesetGroup(cfg *RulesetGroup) (*RulesetGroup, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid rule set group config [nil]")
	}

	groupCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.RulesetGroupCIDRegex, groupCID)
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

	result, err := a.Put(groupCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	groups := &RulesetGroup{}
	if err := json.Unmarshal(result, groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// CreateRulesetGroup create a new rulesetGroup
func (a *API) CreateRulesetGroup(cfg *RulesetGroup) (*RulesetGroup, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid rule set group config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	result, err := a.Post(config.RuleSetGroupPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	group := &RulesetGroup{}
	if err := json.Unmarshal(result, group); err != nil {
		return nil, err
	}

	return group, nil
}

// DeleteRulesetGroup delete a rulesetGroup
func (a *API) DeleteRulesetGroup(cfg *RulesetGroup) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid rule set group config [nil]")
	}
	return a.DeleteRulesetGroupByCID(CIDType(&cfg.CID))
}

// DeleteRulesetGroupByCID delete a rulesetGroup by cid
func (a *API) DeleteRulesetGroupByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid rule set group CID [none]")
	}

	groupCID := string(*cid)

	matched, err := regexp.MatchString(config.RulesetGroupCIDRegex, groupCID)
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

// SearchRulesetGroups returns list of annotations matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) SearchRulesetGroups(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]RulesetGroup, error) {
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
		return a.FetchRulesetGroups()
	}

	reqURL := url.URL{
		Path:     config.RuleSetGroupPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var groups []RulesetGroup
	if err := json.Unmarshal(result, &groups); err != nil {
		return nil, err
	}

	return &groups, nil
}
