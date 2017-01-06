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
	CN   string `json:"cn,omitempty"`
	Host string `json:"host,omitempty"`
	Port string `json:"port,omitempty"`
}

// ProvisionBroker defines a broker
type ProvisionBroker struct {
	CID                     string           `json:"_cid,omitempty"`
	Cert                    string           `json:"_cert,omitempty"`
	Stratcons               []BrokerStratcon `json:"_stratcons,omitempty"`
	CSR                     string           `json:"_csr,omitempty"`
	ExternalHost            string           `json:"external_host,omitempty"`
	ExternalPort            string           `json:"external_port,omitempty"`
	IPAddress               string           `json:"ipaddress,omitempty"`
	Latitude                string           `json:"latitude,omitempty"`
	Longitude               string           `json:"longitude,omitempty"`
	NoitName                string           `json:"noit_name,omitempty"`
	Port                    string           `json:"port,omitempty"`
	PreferReverseConnection bool             `json:"prefer_reverse_connection,omitempty"`
	Rebuild                 bool             `json:"rebuild,omitempty"`
	Tags                    []string         `json:"tags,omitempty"`
}

// FetchProvisionBroker retrieves a broker definition
func (a *API) FetchProvisionBroker(cid CIDType) (*ProvisionBroker, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid broker CID [none]")
	}

	brokerCID := string(*cid)

	matched, err := regexp.MatchString(config.ProvisionBrokerCIDRegex, brokerCID)
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

	broker := &ProvisionBroker{}
	if err := json.Unmarshal(result, broker); err != nil {
		return nil, err
	}

	return broker, nil
}

// UpdateProvisionBroker update broker definition
func (a *API) UpdateProvisionBroker(cid CIDType, cfg *ProvisionBroker) (*ProvisionBroker, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid broker config [nil]")
	}

	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid broker CID [none]")
	}

	brokerCID := string(*cid)

	matched, err := regexp.MatchString(config.ProvisionBrokerCIDRegex, brokerCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid broker CID [%s]", brokerCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
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

// CreateProvisionBroker create a new broker
func (a *API) CreateProvisionBroker(cfg *ProvisionBroker) (*ProvisionBroker, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid broker config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
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
