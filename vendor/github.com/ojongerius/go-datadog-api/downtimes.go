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
	Period           int      `json:"period,omitempty"`
	Type             string   `json:"type,omitempty"`
	UntilDate        int      `json:"until_date,omitempty"`
	UntilOccurrences int      `json:"until_occurrences,omitempty"`
	WeekDays         []string `json:"week_days,omitempty"`
}

type Downtime struct {
	Active     bool        `json:"active,omitempty"`
	Canceled   int         `json:"canceled,omitempty"`
	Disabled   bool        `json:"disabled,omitempty"`
	End        int         `json:"end,omitempty"`
	Id         int         `json:"id,omitempty"`
	Message    string      `json:"message,omitempty"`
	Recurrence *Recurrence `json:"recurrence,omitempty"`
	Scope      []string    `json:"scope,omitempty"`
	Start      int         `json:"start,omitempty"`
}

// reqDowntimes retrieves a slice of all Downtimes.
type reqDowntimes struct {
	Downtimes []Downtime `json:"downtimes,omitempty"`
}

// CreateDowntime adds a new downtme to the system. This returns a pointer
// to a Downtime so you can pass that to UpdateDowntime or CancelDowntime
// later if needed.
func (self *Client) CreateDowntime(downtime *Downtime) (*Downtime, error) {
	var out Downtime
	err := self.doJsonRequest("POST", "/v1/downtime", downtime, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateDowntime takes a downtime that was previously retrieved through some method
// and sends it back to the server.
func (self *Client) UpdateDowntime(downtime *Downtime) error {
	return self.doJsonRequest("PUT", fmt.Sprintf("/v1/downtime/%d", downtime.Id),
		downtime, nil)
}

// Getdowntime retrieves an downtime by identifier.
func (self *Client) GetDowntime(id int) (*Downtime, error) {
	var out Downtime
	err := self.doJsonRequest("GET", fmt.Sprintf("/v1/downtime/%d", id), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDowntime removes an downtime from the system.
func (self *Client) DeleteDowntime(id int) error {
	return self.doJsonRequest("DELETE", fmt.Sprintf("/v1/downtime/%d", id),
		nil, nil)
}

// GetDowntimes returns a slice of all downtimes.
func (self *Client) GetDowntimes() ([]Downtime, error) {
	var out reqDowntimes
	err := self.doJsonRequest("GET", "/v1/downtime", nil, &out.Downtimes)
	if err != nil {
		return nil, err
	}
	return out.Downtimes, nil
}
