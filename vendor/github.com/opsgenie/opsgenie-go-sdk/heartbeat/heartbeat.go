/*
Copyright 2015 OpsGenie. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package heartbeat provides requests and response structures to achieve Heartbeat API actions.
package heartbeat

// AddHeartbeatRequest provides necessary parameter structure to Create an Heartbeat at OpsGenie.
type AddHeartbeatRequest struct {
	APIKey       string `json:"apiKey,omitempty"`
	Name         string `json:"name,omitempty"`
	Interval     int    `json:"interval,omitempty"`
	IntervalUnit string `json:"intervalUnit,omitempty"`
	Description  string `json:"description,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

// UpdateHeartbeatRequest provides necessary parameter structure to Update an existing Heartbeat at OpsGenie.
type UpdateHeartbeatRequest struct {
	APIKey       string `json:"apiKey,omitempty"`
	Name         string `json:"name,omitempty"`
	Interval     int    `json:"interval,omitempty"`
	IntervalUnit string `json:"intervalUnit,omitempty"`
	Description  string `json:"description,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

// EnableHeartbeatRequest provides necessary parameter structure to Enable an Heartbeat at OpsGenie.
type EnableHeartbeatRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Name   string `json:"name,omitempty"`
}

// DisableHeartbeatRequest provides necessary parameter structure to Disable an Heartbeat at OpsGenie.
type DisableHeartbeatRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Name   string `json:"name,omitempty"`
}

// DeleteHeartbeatRequest provides necessary parameter structure to Delete an Heartbeat from OpsGenie.
type DeleteHeartbeatRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Name   string `url:"name,omitempty"`
}

// GetHeartbeatRequest provides necessary parameter structure to Retrieve an Heartbeat with details from OpsGenie.
type GetHeartbeatRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Name   string `url:"name,omitempty"`
}

// ListHeartbeatsRequest provides necessary parameter structure to Retrieve Heartbeats from OpsGenie.
type ListHeartbeatsRequest struct {
	APIKey string `url:"apiKey,omitempty"`
}

// SendHeartbeatRequest provides necessary parameter structure to Send an Heartbeat Signal to OpsGenie.
type SendHeartbeatRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Name   string `json:"name,omitempty"`
}
