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

// Graph represents a graph that might exist on a dashboard.
type Graph struct {
	Title      string     `json:"title"`
	Events     []struct{} `json:"events"`
	Definition struct {
		Viz      string `json:"viz"`
		Requests []struct {
			Query   string `json:"q"`
			Stacked bool   `json:"stacked"`
		} `json:"requests"`
	} `json:"definition"`
}

// Template variable represents a template variable that might exist on a dashboard
type TemplateVariable struct {
	Name    string `json:"name"`
	Prefix  string `json:"prefix"`
	Default string `json:"default"`
}

// Dashboard represents a user created dashboard. This is the full dashboard
// struct when we load a dashboard in detail.
type Dashboard struct {
	Id                int                `json:"id"`
	Description       string             `json:"description"`
	Title             string             `json:"title"`
	Graphs            []Graph            `json:"graphs"`
	TemplateVariables []TemplateVariable `json:"template_variables,omitempty"`
}

// DashboardLite represents a user created dashboard. This is the mini
// struct when we load the summaries.
type DashboardLite struct {
	Id          int    `json:"id,string"` // TODO: Remove ',string'.
	Resource    string `json:"resource"`
	Description string `json:"description"`
	Title       string `json:"title"`
}

// reqGetDashboards from /api/v1/dash
type reqGetDashboards struct {
	Dashboards []DashboardLite `json:"dashes"`
}

// reqGetDashboard from /api/v1/dash/:dashboard_id
type reqGetDashboard struct {
	Resource  string    `json:"resource"`
	Url       string    `json:"url"`
	Dashboard Dashboard `json:"dash"`
}

// GetDashboard returns a single dashboard created on this account.
func (self *Client) GetDashboard(id int) (*Dashboard, error) {
	var out reqGetDashboard
	err := self.doJsonRequest("GET", fmt.Sprintf("/v1/dash/%d", id), nil, &out)
	if err != nil {
		return nil, err
	}
	return &out.Dashboard, nil
}

// GetDashboards returns a list of all dashboards created on this account.
func (self *Client) GetDashboards() ([]DashboardLite, error) {
	var out reqGetDashboards
	err := self.doJsonRequest("GET", "/v1/dash", nil, &out)
	if err != nil {
		return nil, err
	}
	return out.Dashboards, nil
}

// DeleteDashboard deletes a dashboard by the identifier.
func (self *Client) DeleteDashboard(id int) error {
	return self.doJsonRequest("DELETE", fmt.Sprintf("/v1/dash/%d", id), nil, nil)
}

// CreateDashboard creates a new dashboard when given a Dashboard struct. Note
// that the Id, Resource, Url and similar elements are not used in creation.
func (self *Client) CreateDashboard(dash *Dashboard) (*Dashboard, error) {
	var out reqGetDashboard
	err := self.doJsonRequest("POST", "/v1/dash", dash, &out)
	if err != nil {
		return nil, err
	}
	return &out.Dashboard, nil
}

// UpdateDashboard in essence takes a Dashboard struct and persists it back to
// the server. Use this if you've updated your local and need to push it back.
func (self *Client) UpdateDashboard(dash *Dashboard) error {
	return self.doJsonRequest("PUT", fmt.Sprintf("/v1/dash/%d", dash.Id),
		dash, nil)
}
