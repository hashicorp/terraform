/*
Copyright 2016. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package schedule provides requests and response structures to achieve Schedule API actions.
package schedule

// Restrictions defines the structure for each rotation restrictions
type Restriction struct {
	StartDay string `json:"startDay,omitempty"`
	StartTime string `json:"startTime,omitempty"`
	EndDay string `json:"endDay,omitempty"`
	EndTime string `json:"endTime,omitempty"`
}

// Rotation defines the structure for each rotation definition
type Rotation struct {
	StartDate string `json:"startDate,omitempty"`
	EndDate string `json:"endDate,omitempty"`
	RotationType string `json:"rotationType,omitempty"`
	Participants []string `json:"participants,omitempty"`
	Name string `json:"name,omitempty"`
	RotationLength int `json:"rotationLength,omitempty"`
	Restrictions []Restriction `json:"restrictions,omitempty"`
}

// CreateScheduleRequest provides necessary parameter structure for creating Schedule
type CreateScheduleRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Name   string `json:"name,omitempty"`
	Timezone string `json:"timezone,omitempty"`
	Enabled *bool `json:"enabled,omitempty"`
        Rotations []Rotation `json:"rotations,omitempty"`
}

// UpdateScheduleRequest provides necessary parameter structure for updating an Schedule
type UpdateScheduleRequest struct {
	Id     string `json:"id,omitempty"`
	APIKey string `json:"apiKey,omitempty"`
	Name   string `json:"name,omitempty"`
	Timezone string `json:"timezone,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
        Rotations []Rotation `json:"rotations,omitempty"`
}

// DeleteScheduleRequest provides necessary parameter structure for deleting an Schedule
type DeleteScheduleRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Id     string `url:"id,omitempty"`
        Name   string `url:"name,omitempty"`
}

// GetScheduleRequest provides necessary parameter structure for requesting Schedule information
type GetScheduleRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Id     string `url:"id,omitempty"`
        Name   string `url:"name,omitempty"`
}

// ListScheduleRequest provides necessary parameter structure for listing Schedules
type ListSchedulesRequest struct {
	APIKey string `url:"apiKey,omitempty"`
}
