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
)

// RulesetRule defines a ruleset rule
type RulesetRule struct {
	Criteria          string `json:"criteria"`
	Severity          uint   `json:"severity"`
	Value             string `json:"value"`
	WindowingDuration uint   `json:"windowing_duration,omitempty"`
	WindowingFunction string `json:"windowing_function,omitempty"`
	Wait              uint   `json:"wait,omitempty"`
}

// Ruleset defines a ruleset
type Ruleset struct {
	CID           string             `json:"_cid,omitempty"`
	CheckCID      string             `json:"check"`
	ContactGroups map[uint8][]string `json:"contact_groups"`
	Derive        string             `json:"derive,omitempty"`
	Link          string             `json:"link"`
	MetricName    string             `json:"metric_name"`
	MetricTags    []string           `json:"metric_tags"`
	MetricType    string             `json:"metric_type"`
	Notes         string             `json:"notes"`
	Parent        string             `json:"parent,omitempty"`
	Rules         []RulesetRule      `json:"rules"`
	Tags          []string           `json:"tags"`
}

const (
	baseRulesetPath = "/rule_set"
	rulesetCIDRegex = "^" + baseRulesetPath + "/[0-9]+_.+$"
)

// FetchRuleset retrieves a ruleset definition
func (a *API) FetchRuleset(cid CIDType) (*Ruleset, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid rule set CID [none]")
	}

	rulesetCID := string(*cid)

	matched, err := regexp.MatchString(rulesetCIDRegex, rulesetCID)
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

	ruleset := &Ruleset{}
	if err := json.Unmarshal(result, ruleset); err != nil {
		return nil, err
	}

	return ruleset, nil
}

// FetchRulesets retrieves all rulesets
func (a *API) FetchRulesets() (*[]Ruleset, error) {
	result, err := a.Get(baseRulesetPath)
	if err != nil {
		return nil, err
	}

	var rulesets []Ruleset
	if err := json.Unmarshal(result, &rulesets); err != nil {
		return nil, err
	}

	return &rulesets, nil
}

// UpdateRuleset update ruleset definition
func (a *API) UpdateRuleset(config *Ruleset) (*Ruleset, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid rule set config [none]")
	}

	rulesetCID := string(config.CID)

	matched, err := regexp.MatchString(rulesetCIDRegex, rulesetCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid rule set CID [%s]", rulesetCID)
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	result, err := a.Put(rulesetCID, cfg)
	if err != nil {
		return nil, err
	}

	ruleset := &Ruleset{}
	if err := json.Unmarshal(result, ruleset); err != nil {
		return nil, err
	}

	return ruleset, nil
}

// CreateRuleset create a new ruleset
func (a *API) CreateRuleset(config *Ruleset) (*Ruleset, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid rule set config [none]")
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	resp, err := a.Post(baseRulesetPath, cfg)
	if err != nil {
		return nil, err
	}

	ruleset := &Ruleset{}
	if err := json.Unmarshal(resp, ruleset); err != nil {
		return nil, err
	}

	return ruleset, nil
}

// DeleteRuleset delete a ruleset
func (a *API) DeleteRuleset(config *Ruleset) (bool, error) {
	if config == nil {
		return false, fmt.Errorf("Invalid rule set config [none]")
	}

	cid := CIDType(&config.CID)
	return a.DeleteRulesetByCID(cid)
}

// DeleteRulesetByCID delete a ruleset by cid
func (a *API) DeleteRulesetByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid rule set CID [none]")
	}

	rulesetCID := string(*cid)

	matched, err := regexp.MatchString(rulesetCIDRegex, rulesetCID)
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

// SearchRulesets returns list of rule sets matching a search query and/or filter
//    - a search query (see: https://login.circonus.com/resources/api#searching)
//    - a filter (see: https://login.circonus.com/resources/api#filtering)
func (a *API) SearchRulesets(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Ruleset, error) {
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
		return a.FetchRulesets()
	}

	reqURL := url.URL{
		Path:     baseRulesetPath,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var rulesets []Ruleset
	if err := json.Unmarshal(result, &rulesets); err != nil {
		return nil, err
	}

	return &rulesets, nil
}
