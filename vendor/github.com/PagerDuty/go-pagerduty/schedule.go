package pagerduty

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// Restriction limits on-call responsibility for a layer to certain times of the day or week.
type Restriction struct {
	Type            string `json:"type,omitempty"`
	StartTimeOfDay  string `json:"start_time_of_day,omitempty"`
	StartDayOfWeek  uint   `json:"start_day_of_week,omitempty"`
	DurationSeconds uint   `json:"duration_seconds,omitempty"`
}

// RenderedScheduleEntry represents the computed set of schedule layer entries that put users on call for a schedule, and cannot be modified directly.
type RenderedScheduleEntry struct {
	Start string    `json:"start,omitempty"`
	End   string    `json:"end,omitempty"`
	User  APIObject `json:"user,omitempty"`
}

// ScheduleLayer is an entry that puts users on call for a schedule.
type ScheduleLayer struct {
	APIObject
	Name                       string                  `json:"name,omitempty"`
	Start                      string                  `json:"start,omitempty"`
	End                        string                  `json:"end,omitempty"`
	RotationVirtualStart       string                  `json:"rotation_virtual_start,omitempty"`
	RotationTurnLengthSeconds  uint                    `json:"rotation_turn_length_seconds,omitempty"`
	Users                      []UserReference         `json:"users,omitempty"`
	Restrictions               []Restriction           `json:"restrictions,omitempty"`
	RenderedScheduleEntries    []RenderedScheduleEntry `json:"rendered_schedule_entries,omitempty"`
	RenderedCoveragePercentage float64                 `json:"rendered_coverage_percentage,omitempty"`
}

// Schedule determines the time periods that users are on call.
type Schedule struct {
	APIObject
	Name                string          `json:"name,omitempty"`
	TimeZone            string          `json:"time_zone,omitempty"`
	Description         string          `json:"description,omitempty"`
	EscalationPolicies  []APIObject     `json:"escalation_policies,omitempty"`
	Users               []APIObject     `json:"users,omitempty"`
	ScheduleLayers      []ScheduleLayer `json:"schedule_layers,omitempty"`
	OverrideSubschedule ScheduleLayer   `json:"override_subschedule,omitempty"`
	FinalSchedule       ScheduleLayer   `json:"final_schedule,omitempty"`
}

// ListSchedulesOptions is the data structure used when calling the ListSchedules API endpoint.
type ListSchedulesOptions struct {
	APIListObject
	Query string `url:"query,omitempty"`
}

// ListSchedulesResponse is the data structure returned from calling the ListSchedules API endpoint.
type ListSchedulesResponse struct {
	APIListObject
	Schedules []Schedule
}

// UserReference is a reference to an authorized PagerDuty user.
type UserReference struct {
	User APIObject `json:"user"`
}

// ListSchedules lists the on-call schedules.
func (c *Client) ListSchedules(o ListSchedulesOptions) (*ListSchedulesResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}
	resp, err := c.get("/schedules?" + v.Encode())
	if err != nil {
		return nil, err
	}
	var result ListSchedulesResponse
	return &result, c.decodeJSON(resp, &result)
}

// CreateSchedule creates a new on-call schedule.
func (c *Client) CreateSchedule(s Schedule) (*Schedule, error) {
	data := make(map[string]Schedule)
	data["schedule"] = s
	resp, err := c.post("/schedules", data)
	if err != nil {
		return nil, err
	}
	return getScheduleFromResponse(c, resp)
}

// PreviewScheduleOptions is the data structure used when calling the PreviewSchedule API endpoint.
type PreviewScheduleOptions struct {
	APIListObject
	Since    string `url:"since,omitempty"`
	Until    string `url:"until,omitempty"`
	Overflow bool   `url:"overflow,omitempty"`
}

// PreviewSchedule previews what an on-call schedule would look like without saving it.
func (c *Client) PreviewSchedule(s Schedule, o PreviewScheduleOptions) error {
	v, err := query.Values(o)
	if err != nil {
		return err
	}
	var data map[string]Schedule
	data["schedule"] = s
	_, e := c.post("/schedules/preview?"+v.Encode(), data)
	return e
}

// DeleteSchedule deletes an on-call schedule.
func (c *Client) DeleteSchedule(id string) error {
	_, err := c.delete("/schedules/" + id)
	return err
}

// GetScheduleOptions is the data structure used when calling the GetSchedule API endpoint.
type GetScheduleOptions struct {
	APIListObject
	TimeZone string `url:"time_zone,omitempty"`
	Since    string `url:"since,omitempty"`
	Until    string `url:"until,omitempty"`
}

// GetSchedule shows detailed information about a schedule, including entries for each layer and sub-schedule.
func (c *Client) GetSchedule(id string, o GetScheduleOptions) (*Schedule, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, fmt.Errorf("Could not parse values for query: %v", err)
	}
	resp, err := c.get("/schedules/" + id + "?" + v.Encode())
	if err != nil {
		return nil, err
	}
	return getScheduleFromResponse(c, resp)
}

// UpdateScheduleOptions is the data structure used when calling the UpdateSchedule API endpoint.
type UpdateScheduleOptions struct {
	Overflow bool `url:"overflow,omitempty"`
}

// UpdateSchedule updates an existing on-call schedule.
func (c *Client) UpdateSchedule(id string, s Schedule) (*Schedule, error) {
	v := make(map[string]Schedule)
	v["schedule"] = s
	resp, err := c.put("/schedules/"+id, v, nil)
	if err != nil {
		return nil, err
	}
	return getScheduleFromResponse(c, resp)
}

// ListOverridesOptions is the data structure used when calling the ListOverrides API endpoint.
type ListOverridesOptions struct {
	APIListObject
	Since    string `url:"since,omitempty"`
	Until    string `url:"until,omitempty"`
	Editable bool   `url:"editable,omitempty"`
	Overflow bool   `url:"overflow,omitempty"`
}

// Overrides are any schedule layers from the override layer.
type Override struct {
	ID    string    `json:"id,omitempty"`
	Start string    `json:"start,omitempty"`
	End   string    `json:"end,omitempty"`
	User  APIObject `json:"user,omitempty"`
}

// ListOverrides lists overrides for a given time range.
func (c *Client) ListOverrides(id string, o ListOverridesOptions) ([]Override, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}
	resp, err := c.get("/schedules/" + id + "/overrides?" + v.Encode())
	if err != nil {
		return nil, err
	}
	var result map[string][]Override
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}
	overrides, ok := result["overrides"]
	if !ok {
		return nil, fmt.Errorf("JSON response does not have overrides field")
	}
	return overrides, nil
}

// CreateOverride creates an override for a specific user covering the specified time range.
func (c *Client) CreateOverride(id string, o Override) (*Override, error) {
	data := make(map[string]Override)
	data["override"] = o
	resp, err := c.post("/schedules/"+id+"/overrides", data)
	if err != nil {
		return nil, err
	}
	return getOverrideFromResponse(c, resp)
}

// DeleteOverride removes an override.
func (c *Client) DeleteOverride(scheduleID, overrideID string) error {
	_, err := c.delete("/schedules/" + scheduleID + "/overrides/" + overrideID)
	return err
}

// ListOnCallUsersOptions is the data structure used when calling the ListOnCallUsers API endpoint.
type ListOnCallUsersOptions struct {
	APIListObject
	Since string `url:"since,omitempty"`
	Until string `url:"until,omitempty"`
}

// ListOnCallUsers lists all of the users on call in a given schedule for a given time range.
func (c *Client) ListOnCallUsers(id string, o ListOnCallUsersOptions) ([]User, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}
	resp, err := c.get("/schedules/" + id + "/users?" + v.Encode())
	if err != nil {
		return nil, err
	}
	var result map[string][]User
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}
	u, ok := result["users"]
	if !ok {
		return nil, fmt.Errorf("JSON response does not have users field")
	}
	return u, nil
}

func getScheduleFromResponse(c *Client, resp *http.Response) (*Schedule, error) {
	var target map[string]Schedule
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}
	rootNode := "schedule"
	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}
	return &t, nil
}

func getOverrideFromResponse(c *Client, resp *http.Response) (*Override, error) {
	var target map[string]Override
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}
	rootNode := "override"
	o, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}
	return &o, nil
}
