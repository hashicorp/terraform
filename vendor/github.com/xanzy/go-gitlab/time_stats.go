package gitlab

import (
	"fmt"
	"net/url"
)

// TimeStatsService handles communication with the time tracking related
// methods of the GitLab API.
//
// GitLab docs: https://docs.gitlab.com/ce/workflow/time_tracking.html
// GitLab API docs: https://docs.gitlab.com/ce/api/issues.html
type TimeStatsService struct {
	client *Client
}

// TimeStats represents the time estimates and time spent for an issue.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/issues.html
type TimeStats struct {
	HumanTimeEstimate   string `json:"human_time_estimate"`
	HumanTotalTimeSpent string `json:"human_total_time_spent"`
	TimeEstimate        int    `json:"time_estimate"`
	TotalTimeSpent      int    `json:"total_time_spent"`
}

func (t TimeStats) String() string {
	return Stringify(t)
}

// SetTimeEstimateOptions represents the available SetTimeEstimate()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/issues.html#set-a-time-estimate-for-an-issue
type SetTimeEstimateOptions struct {
	Duration *string `url:"duration,omitempty" json:"duration,omitempty"`
}

// SetTimeEstimate sets the time estimate for a single project issue.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/issues.html#set-a-time-estimate-for-an-issue
func (s *TimeStatsService) SetTimeEstimate(pid interface{}, issue int, opt *SetTimeEstimateOptions, options ...OptionFunc) (*TimeStats, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/time_estimate", url.QueryEscape(project), issue)

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(TimeStats)
	resp, err := s.client.Do(req, t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, err
}

// ResetTimeEstimate resets the time estimate for a single project issue.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/issues.html#reset-the-time-estimate-for-an-issue
func (s *TimeStatsService) ResetTimeEstimate(pid interface{}, issue int, options ...OptionFunc) (*TimeStats, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/reset_time_estimate", url.QueryEscape(project), issue)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(TimeStats)
	resp, err := s.client.Do(req, t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, err
}

// AddSpentTimeOptions represents the available AddSpentTime() options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/issues.html#add-spent-time-for-an-issue
type AddSpentTimeOptions struct {
	Duration *string `url:"duration,omitempty" json:"duration,omitempty"`
}

// AddSpentTime adds spent time for a single project issue.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/issues.html#add-spent-time-for-an-issue
func (s *TimeStatsService) AddSpentTime(pid interface{}, issue int, opt *AddSpentTimeOptions, options ...OptionFunc) (*TimeStats, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/add_spent_time", url.QueryEscape(project), issue)

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(TimeStats)
	resp, err := s.client.Do(req, t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, err
}

// ResetSpentTime resets the spent time for a single project issue.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/issues.html#reset-spent-time-for-an-issue
func (s *TimeStatsService) ResetSpentTime(pid interface{}, issue int, options ...OptionFunc) (*TimeStats, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/reset_spent_time", url.QueryEscape(project), issue)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(TimeStats)
	resp, err := s.client.Do(req, t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, err
}

// GetTimeSpent gets the spent time for a single project issue.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/issues.html#get-time-tracking-stats
func (s *TimeStatsService) GetTimeSpent(pid interface{}, issue int, options ...OptionFunc) (*TimeStats, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/time_stats", url.QueryEscape(project), issue)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(TimeStats)
	resp, err := s.client.Do(req, t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, err
}
