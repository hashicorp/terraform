/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

import (
	"fmt"
)

// Alert represents the data of an alert: a query that can fire and send a
// message to the users.
type Alert struct {
	Id           *int    `json:"id,omitempty"`
	Creator      *int    `json:"creator,omitempty"`
	Query        *string `json:"query,omitempty"`
	Name         *string `json:"name,omitempty"`
	Message      *string `json:"message,omitempty"`
	Silenced     *bool   `json:"silenced,omitempty"`
	NotifyNoData *bool   `json:"notify_no_data,omitempty"`
	State        *string `json:"state,omitempty"`
}

// reqAlerts receives a slice of all alerts.
type reqAlerts struct {
	Alerts []Alert `json:"alerts,omitempty"`
}

// CreateAlert adds a new alert to the system. This returns a pointer to an
// Alert so you can pass that to UpdateAlert later if needed.
func (client *Client) CreateAlert(alert *Alert) (*Alert, error) {
	var out Alert
	if err := client.doJsonRequest("POST", "/v1/alert", alert, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateAlert takes an alert that was previously retrieved through some method
// and sends it back to the server.
func (client *Client) UpdateAlert(alert *Alert) error {
	return client.doJsonRequest("PUT", fmt.Sprintf("/v1/alert/%d", alert.Id),
		alert, nil)
}

// GetAlert retrieves an alert by identifier.
func (client *Client) GetAlert(id int) (*Alert, error) {
	var out Alert
	if err := client.doJsonRequest("GET", fmt.Sprintf("/v1/alert/%d", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAlert removes an alert from the system.
func (client *Client) DeleteAlert(id int) error {
	return client.doJsonRequest("DELETE", fmt.Sprintf("/v1/alert/%d", id),
		nil, nil)
}

// GetAlerts returns a slice of all alerts.
func (client *Client) GetAlerts() ([]Alert, error) {
	var out reqAlerts
	if err := client.doJsonRequest("GET", "/v1/alert", nil, &out); err != nil {
		return nil, err
	}
	return out.Alerts, nil
}

// MuteAlerts turns off alerting notifications.
func (client *Client) MuteAlerts() error {
	return client.doJsonRequest("POST", "/v1/mute_alerts", nil, nil)
}

// UnmuteAlerts turns on alerting notifications.
func (client *Client) UnmuteAlerts() error {
	return client.doJsonRequest("POST", "/v1/unmute_alerts", nil, nil)
}
