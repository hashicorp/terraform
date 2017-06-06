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

type Recurrence struct {
	Period           *int     `json:"period,omitempty"`
	Type             *string  `json:"type,omitempty"`
	UntilDate        *int     `json:"until_date,omitempty"`
	UntilOccurrences *int     `json:"until_occurrences,omitempty"`
	WeekDays         []string `json:"week_days,omitempty"`
}

type Downtime struct {
	Active     *bool       `json:"active,omitempty"`
	Canceled   *int        `json:"canceled,omitempty"`
	Disabled   *bool       `json:"disabled,omitempty"`
	End        *int        `json:"end,omitempty"`
	Id         *int        `json:"id,omitempty"`
	MonitorId  *int        `json:"monitor_id,omitempty"`
	Message    *string     `json:"message,omitempty"`
	Recurrence *Recurrence `json:"recurrence,omitempty"`
	Scope      []string    `json:"scope,omitempty"`
	Start      *int        `json:"start,omitempty"`
}

// reqDowntimes retrieves a slice of all Downtimes.
type reqDowntimes struct {
	Downtimes []Downtime `json:"downtimes,omitempty"`
}

// CreateDowntime adds a new downtme to the system. This returns a pointer
// to a Downtime so you can pass that to UpdateDowntime or CancelDowntime
// later if needed.
func (client *Client) CreateDowntime(downtime *Downtime) (*Downtime, error) {
	var out Downtime
	if err := client.doJsonRequest("POST", "/v1/downtime", downtime, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateDowntime takes a downtime that was previously retrieved through some method
// and sends it back to the server.
func (client *Client) UpdateDowntime(downtime *Downtime) error {
	return client.doJsonRequest("PUT", fmt.Sprintf("/v1/downtime/%d", *downtime.Id),
		downtime, nil)
}

// Getdowntime retrieves an downtime by identifier.
func (client *Client) GetDowntime(id int) (*Downtime, error) {
	var out Downtime
	if err := client.doJsonRequest("GET", fmt.Sprintf("/v1/downtime/%d", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDowntime removes an downtime from the system.
func (client *Client) DeleteDowntime(id int) error {
	return client.doJsonRequest("DELETE", fmt.Sprintf("/v1/downtime/%d", id),
		nil, nil)
}

// GetDowntimes returns a slice of all downtimes.
func (client *Client) GetDowntimes() ([]Downtime, error) {
	var out reqDowntimes
	if err := client.doJsonRequest("GET", "/v1/downtime", nil, &out.Downtimes); err != nil {
		return nil, err
	}
	return out.Downtimes, nil
}
