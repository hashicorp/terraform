/*
Copyright 2016. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package team provides requests and response structures to achieve Team API actions.
package team

// Member defines the structure for each team members definition
type Member struct {
	User string `json:"user,omitempty"`
	Role string `json:"role,omitempty"`
}

// CreateTeamRequest provides necessary parameter structure for creating team
type CreateTeamRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Description string `json:"description,omitempty"`
	Name   string `json:"name,omitempty"`
        Members []Member `json:"members,omitempty"`
}

// UpdateTeamRequest provides necessary parameter structure for updating a team
type UpdateTeamRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Description string `json:"description,omitempty"`
	Id     string `json:"id,omitempty"`
        Name   string `json:"name,omitempty"`
        Members []Member `json:"members,omitempty"`
}

// DeleteTeamRequest provides necessary parameter structure for deleting a team
type DeleteTeamRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Id     string `url:"id,omitempty"`
        Name   string `url:"name,omitempty"`
}

// GetTeamRequest provides necessary parameter structure for requesting team information
type GetTeamRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Id     string `url:"id,omitempty"`
        Name   string `url:"name,omitempty"`
}

// ListTeamsRequest provides necessary parameter structure for listing teams
type ListTeamsRequest struct {
	APIKey string `url:"apiKey,omitempty"`
}

// ListTeamLogsRequest provides necessary parameter structure for listing team logs
type ListTeamLogsRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Id     string `url:"id,omitempty"`
        Name   string `url:"name,omitempty"`
	Limit  int `url:"limit,omitempty"`
	Order  string `url:"order,omitempty"`
	LastKey string `url:"lastkey,omitempty"`
}
