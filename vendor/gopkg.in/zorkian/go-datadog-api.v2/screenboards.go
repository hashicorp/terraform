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

// Screenboard represents a user created screenboard. This is the full screenboard
// struct when we load a screenboard in detail.
type Screenboard struct {
	Id                *int               `json:"id,omitempty"`
	Title             *string            `json:"board_title,omitempty"`
	Height            *string            `json:"height,omitempty"`
	Width             *string            `json:"width,omitempty"`
	Shared            *bool              `json:"shared,omitempty"`
	Templated         *bool              `json:"templated,omitempty"`
	TemplateVariables []TemplateVariable `json:"template_variables,omitempty"`
	Widgets           []Widget           `json:"widgets,omitempty"`
	ReadOnly          *bool              `json:"read_only,omitempty"`
}

//type Widget struct {
type Widget struct {
	Default             *string              `json:"default,omitempty"`
	Name                *string              `json:"name,omitempty"`
	Prefix              *string              `json:"prefix,omitempty"`
	TimeseriesWidget    *TimeseriesWidget    `json:"timeseries,omitempty"`
	QueryValueWidget    *QueryValueWidget    `json:"query_value,omitempty"`
	EventStreamWidget   *EventStreamWidget   `json:"event_stream,omitempty"`
	FreeTextWidget      *FreeTextWidget      `json:"free_text,omitempty"`
	ToplistWidget       *ToplistWidget       `json:"toplist,omitempty"`
	ImageWidget         *ImageWidget         `json:"image,omitempty"`
	ChangeWidget        *ChangeWidget        `json:"change,omitempty"`
	GraphWidget         *GraphWidget         `json:"graph,omitempty"`
	EventTimelineWidget *EventTimelineWidget `json:"event_timeline,omitempty"`
	AlertValueWidget    *AlertValueWidget    `json:"alert_value,omitempty"`
	AlertGraphWidget    *AlertGraphWidget    `json:"alert_graph,omitempty"`
	HostMapWidget       *HostMapWidget       `json:"hostmap,omitempty"`
	CheckStatusWidget   *CheckStatusWidget   `json:"check_status,omitempty"`
	IFrameWidget        *IFrameWidget        `json:"iframe,omitempty"`
	NoteWidget          *NoteWidget          `json:"frame,omitempty"`
}

// ScreenboardLite represents a user created screenboard. This is the mini
// struct when we load the summaries.
type ScreenboardLite struct {
	Id       *int    `json:"id,omitempty"`
	Resource *string `json:"resource,omitempty"`
	Title    *string `json:"title,omitempty"`
}

// reqGetScreenboards from /api/v1/screen
type reqGetScreenboards struct {
	Screenboards []*ScreenboardLite `json:"screenboards,omitempty"`
}

// GetScreenboard returns a single screenboard created on this account.
func (client *Client) GetScreenboard(id int) (*Screenboard, error) {
	out := &Screenboard{}
	if err := client.doJsonRequest("GET", fmt.Sprintf("/v1/screen/%d", id), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetScreenboards returns a list of all screenboards created on this account.
func (client *Client) GetScreenboards() ([]*ScreenboardLite, error) {
	var out reqGetScreenboards
	if err := client.doJsonRequest("GET", "/v1/screen", nil, &out); err != nil {
		return nil, err
	}
	return out.Screenboards, nil
}

// DeleteScreenboard deletes a screenboard by the identifier.
func (client *Client) DeleteScreenboard(id int) error {
	return client.doJsonRequest("DELETE", fmt.Sprintf("/v1/screen/%d", id), nil, nil)
}

// CreateScreenboard creates a new screenboard when given a Screenboard struct. Note
// that the Id, Resource, Url and similar elements are not used in creation.
func (client *Client) CreateScreenboard(board *Screenboard) (*Screenboard, error) {
	out := &Screenboard{}
	if err := client.doJsonRequest("POST", "/v1/screen", board, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateScreenboard in essence takes a Screenboard struct and persists it back to
// the server. Use this if you've updated your local and need to push it back.
func (client *Client) UpdateScreenboard(board *Screenboard) error {
	return client.doJsonRequest("PUT", fmt.Sprintf("/v1/screen/%d", *board.Id), board, nil)
}

type ScreenShareResponse struct {
	BoardId   int    `json:"board_id"`
	PublicUrl string `json:"public_url"`
}

// ShareScreenboard shares an existing screenboard, it takes and updates ScreenShareResponse
func (client *Client) ShareScreenboard(id int, response *ScreenShareResponse) error {
	return client.doJsonRequest("GET", fmt.Sprintf("/v1/screen/share/%d", id), nil, response)
}

// RevokeScreenboard revokes a currently shared screenboard
func (client *Client) RevokeScreenboard(id int) error {
	return client.doJsonRequest("DELETE", fmt.Sprintf("/v1/screen/share/%d", id), nil, nil)
}
