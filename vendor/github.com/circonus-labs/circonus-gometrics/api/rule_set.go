// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Rule Set API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/rule_set

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// RuleSetRule defines a ruleset rule
type RuleSetRule struct {
	Criteria          string      `json:"criteria"`                     // string
	Severity          uint        `json:"severity"`                     // uint
	Value             interface{} `json:"value"`                        // BUG doc: string, api: actual type returned switches based on Criteria
	Wait              uint        `json:"wait"`                         // uint
	WindowingDuration uint        `json:"windowing_duration,omitempty"` // uint
	WindowingFunction *string     `json:"windowing_function,omitempty"` // string or null
}

// RuleSet defines a ruleset. See https://login.circonus.com/resources/api/calls/rule_set for more information.
type RuleSet struct {
	CheckCID      string             `json:"check"`            // string
	CID           string             `json:"_cid,omitempty"`   // string
	ContactGroups map[uint8][]string `json:"contact_groups"`   // [] len 5
	Derive        *string            `json:"derive,omitempty"` // string or null
	Link          *string            `json:"link"`             // string or null
	MetricName    string             `json:"metric_name"`      // string
	MetricTags    []string           `json:"metric_tags"`      // [] len >= 0
	MetricType    string             `json:"metric_type"`      // string
	Notes         *string            `json:"notes"`            // string or null
	Parent        *string            `json:"parent,omitempty"` // string or null
	Rules         []RuleSetRule      `json:"rules"`            // [] len >= 1
	Tags          []string           `json:"tags"`             // [] len >= 0
}

// NewRuleSet returns a new RuleSet (with defaults if applicable)
func NewRuleSet() *RuleSet {
	return &RuleSet{}
}

// FetchRuleSet retrieves rule set with passed cid.
func (a *API) FetchRuleSet(cid CIDType) (*RuleSet, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid rule set CID [none]")
	}

	rulesetCID := string(*cid)

	matched, err := regexp.MatchString(config.RuleSetCIDRegex, rulesetCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid rule set CID [%s]", rulesetCID)
	}

	result, err := a.Get(rulesetCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch rule set, received JSON: %s", string(result))
	}

	ruleset := &RuleSet{}
	if err := json.Unmarshal(result, ruleset); err != nil {
		return nil, err
	}

	return ruleset, nil
}

// FetchRuleSets retrieves all rule sets available to API Token.
func (a *API) FetchRuleSets() (*[]RuleSet, error) {
	result, err := a.Get(config.RuleSetPrefix)
	if err != nil {
		return nil, err
	}

	var rulesets []RuleSet
	if err := json.Unmarshal(result, &rulesets); err != nil {
		return nil, err
	}

	return &rulesets, nil
}

// UpdateRuleSet updates passed rule set.
func (a *API) UpdateRuleSet(cfg *RuleSet) (*RuleSet, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid rule set config [nil]")
	}

	rulesetCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.RuleSetCIDRegex, rulesetCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid rule set CID [%s]", rulesetCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update rule set, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(rulesetCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	ruleset := &RuleSet{}
	if err := json.Unmarshal(result, ruleset); err != nil {
		return nil, err
	}

	return ruleset, nil
}

// CreateRuleSet creates a new rule set.
func (a *API) CreateRuleSet(cfg *RuleSet) (*RuleSet, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid rule set config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create rule set, sending JSON: %s", string(jsonCfg))
	}

	resp, err := a.Post(config.RuleSetPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	ruleset := &RuleSet{}
	if err := json.Unmarshal(resp, ruleset); err != nil {
		return nil, err
	}

	return ruleset, nil
}

// DeleteRuleSet deletes passed rule set.
func (a *API) DeleteRuleSet(cfg *RuleSet) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid rule set config [nil]")
	}
	return a.DeleteRuleSetByCID(CIDType(&cfg.CID))
}

// DeleteRuleSetByCID deletes rule set with passed cid.
func (a *API) DeleteRuleSetByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid rule set CID [none]")
	}

	rulesetCID := string(*cid)

	matched, err := regexp.MatchString(config.RuleSetCIDRegex, rulesetCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid rule set CID [%s]", rulesetCID)
	}

	_, err = a.Delete(rulesetCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchRuleSets returns rule sets matching the specified search
// query and/or filter. If nil is passed for both parameters all
// rule sets will be returned.
func (a *API) SearchRuleSets(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]RuleSet, error) {
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
		return a.FetchRuleSets()
	}

	reqURL := url.URL{
		Path:     config.RuleSetPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var rulesets []RuleSet
	if err := json.Unmarshal(result, &rulesets); err != nil {
		return nil, err
	}

	return &rulesets, nil
}
