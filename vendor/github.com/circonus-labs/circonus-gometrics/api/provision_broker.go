// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// ProvisionBroker API support - Fetch, Create, and Update
// See: https://login.circonus.com/resources/api/calls/provision_broker
// Note that the provision_broker endpoint does not return standard cid format
//      of '/object/item' (e.g. /provision_broker/abc-123) it just returns 'item'

package api

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// BrokerStratcon defines stratcons for broker
type BrokerStratcon struct {
	CN   string `json:"cn,omitempty"`   // string
	Host string `json:"host,omitempty"` // string
	Port string `json:"port,omitempty"` // string
}

// ProvisionBroker defines a provision broker [request]. See https://login.circonus.com/resources/api/calls/provision_broker for more details.
type ProvisionBroker struct {
	Cert                    string           `json:"_cert,omitempty"`                     // string
	CID                     string           `json:"_cid,omitempty"`                      // string
	CSR                     string           `json:"_csr,omitempty"`                      // string
	ExternalHost            string           `json:"external_host,omitempty"`             // string
	ExternalPort            string           `json:"external_port,omitempty"`             // string
	IPAddress               string           `json:"ipaddress,omitempty"`                 // string
	Latitude                string           `json:"latitude,omitempty"`                  // string
	Longitude               string           `json:"longitude,omitempty"`                 // string
	Name                    string           `json:"noit_name,omitempty"`                 // string
	Port                    string           `json:"port,omitempty"`                      // string
	PreferReverseConnection bool             `json:"prefer_reverse_connection,omitempty"` // boolean
	Rebuild                 bool             `json:"rebuild,omitempty"`                   // boolean
	Stratcons               []BrokerStratcon `json:"_stratcons,omitempty"`                // [] len >= 1
	Tags                    []string         `json:"tags,omitempty"`                      // [] len >= 0
}

// NewProvisionBroker returns a new ProvisionBroker (with defaults, if applicable)
func NewProvisionBroker() *ProvisionBroker {
	return &ProvisionBroker{}
}

// FetchProvisionBroker retrieves provision broker [request] with passed cid.
func (a *API) FetchProvisionBroker(cid CIDType) (*ProvisionBroker, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid provision broker request CID [none]")
	}

	brokerCID := string(*cid)

	matched, err := regexp.MatchString(config.ProvisionBrokerCIDRegex, brokerCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid provision broker request CID [%s]", brokerCID)
	}

	result, err := a.Get(brokerCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch broker provision request, received JSON: %s", string(result))
	}

	broker := &ProvisionBroker{}
	if err := json.Unmarshal(result, broker); err != nil {
		return nil, err
	}

	return broker, nil
}

// UpdateProvisionBroker updates a broker definition [request].
func (a *API) UpdateProvisionBroker(cid CIDType, cfg *ProvisionBroker) (*ProvisionBroker, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid provision broker request config [nil]")
	}

	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid provision broker request CID [none]")
	}

	brokerCID := string(*cid)

	matched, err := regexp.MatchString(config.ProvisionBrokerCIDRegex, brokerCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid provision broker request CID [%s]", brokerCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update broker provision request, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(brokerCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	broker := &ProvisionBroker{}
	if err := json.Unmarshal(result, broker); err != nil {
		return nil, err
	}

	return broker, nil
}

// CreateProvisionBroker creates a new provison broker [request].
func (a *API) CreateProvisionBroker(cfg *ProvisionBroker) (*ProvisionBroker, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid provision broker request config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create broker provision request, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.ProvisionBrokerPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	broker := &ProvisionBroker{}
	if err := json.Unmarshal(result, broker); err != nil {
		return nil, err
	}

	return broker, nil
}
