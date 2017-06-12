package runscope

import "fmt"

// Schedule determines how often a test is executed. See https://www.runscope.com/docs/api/schedules
type Schedule struct {
	ID            string `json:"id,omitempty"`
	EnvironmentID string `json:"environment_id,omitempty"`
	Interval      string `json:"interval,omitempty"`
	Note          string `json:"note,omitempty"`
}

// NewSchedule creates a new schedule struct
func NewSchedule() *Schedule {
	return &Schedule {}
}

// CreateSchedule creates a new test schedule. See https://www.runscope.com/docs/api/schedules#create
func (client *Client) CreateSchedule(schedule *Schedule, bucketKey string, testID string) (*Schedule, error) {
	newResource, error := client.createResource(schedule, "schedule", schedule.Note,
		fmt.Sprintf("/buckets/%s/tests/%s/schedules", bucketKey, testID))
	if error != nil {
		return nil, error
	}

	newSchedule, error := getScheduleFromResponse(newResource.Data)
	if error != nil {
		return nil, error
	}

	return newSchedule, nil
}

// ReadSchedule list details about an existing test schedule. See https://www.runscope.com/docs/api/schedules#detail
func (client *Client) ReadSchedule(schedule *Schedule, bucketKey string, testID string) (*Schedule, error) {
	resource, error := client.readResource("schedule", schedule.ID,
		fmt.Sprintf("/buckets/%s/tests/%s/schedules/%s", bucketKey, testID, schedule.ID))
	if error != nil {
		return nil, error
	}

	readSchedule, error := getScheduleFromResponse(resource.Data)
	if error != nil {
		return nil, error
	}

	return readSchedule, nil
}

// UpdateSchedule updates an existing test schedule. See https://www.runscope.com/docs/api/schedules#modify
func (client *Client) UpdateSchedule(schedule *Schedule, bucketKey string, testID string) (*Schedule, error) {
	resource, error := client.updateResource(schedule, "schedule", schedule.ID,
		fmt.Sprintf("/buckets/%s/tests/%s/schedules/%s", bucketKey, testID, schedule.ID))
	if error != nil {
		return nil, error
	}

	readSchedule, error := getScheduleFromResponse(resource.Data)
	if error != nil {
		return nil, error
	}

	return readSchedule, nil
}

// DeleteSchedule delete an existing test schedule. See https://www.runscope.com/docs/api/schedules#delete
func (client *Client) DeleteSchedule(schedule *Schedule, bucketKey string, testID string) error {
	return client.deleteResource("schedule", schedule.ID,
		fmt.Sprintf("/buckets/%s/tests/%s/schedules/%s", bucketKey, testID, schedule.ID))
}

func getScheduleFromResponse(response interface{}) (*Schedule, error) {
	schedule := new(Schedule)
	err := decode(schedule, response)
	return schedule, err
}