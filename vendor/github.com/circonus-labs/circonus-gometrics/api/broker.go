// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Broker API support - Fetch and Search
// See: https://login.circonus.com/resources/api/calls/broker

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// BrokerDetail defines instance attributes
type BrokerDetail struct {
	CN           string   `json:"cn"`                       // string
	ExternalHost *string  `json:"external_host"`            // string or null
	ExternalPort uint16   `json:"external_port"`            // uint16
	IP           *string  `json:"ipaddress"`                // string or null
	MinVer       uint     `json:"minimum_version_required"` // uint
	Modules      []string `json:"modules"`                  // [] len >= 0
	Port         *uint16  `json:"port"`                     // uint16 or null
	Skew         *string  `json:"skew"`                     // BUG doc: floating point number, api object: string or null
	Status       string   `json:"status"`                   // string
	Version      *uint    `json:"version"`                  // uint or null
}

// Broker defines a broker. See https://login.circonus.com/resources/api/calls/broker for more information.
type Broker struct {
	CID       string         `json:"_cid"`       // string
	Details   []BrokerDetail `json:"_details"`   // [] len >= 1
	Latitude  *string        `json:"_latitude"`  // string or null
	Longitude *string        `json:"_longitude"` // string or null
	Name      string         `json:"_name"`      // string
	Tags      []string       `json:"_tags"`      // [] len >= 0
	Type      string         `json:"_type"`      // string
}

// FetchBroker retrieves broker with passed cid.
func (a *API) FetchBroker(cid CIDType) (*Broker, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid broker CID [none]")
	}

	brokerCID := string(*cid)

	matched, err := regexp.MatchString(config.BrokerCIDRegex, brokerCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid broker CID [%s]", brokerCID)
	}

	result, err := a.Get(brokerCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch broker, received JSON: %s", string(result))
	}

	response := new(Broker)
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}

	return response, nil

}

// FetchBrokers returns all brokers available to the API Token.
func (a *API) FetchBrokers() (*[]Broker, error) {
	result, err := a.Get(config.BrokerPrefix)
	if err != nil {
		return nil, err
	}

	var response []Broker
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// SearchBrokers returns brokers matching the specified search
// query and/or filter. If nil is passed for both parameters
// all brokers will be returned.
func (a *API) SearchBrokers(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Broker, error) {
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
		return a.FetchBrokers()
	}

	reqURL := url.URL{
		Path:     config.BrokerPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var brokers []Broker
	if err := json.Unmarshal(result, &brokers); err != nil {
		return nil, err
	}

	return &brokers, nil
}
