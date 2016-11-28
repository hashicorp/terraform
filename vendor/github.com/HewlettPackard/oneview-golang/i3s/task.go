/*
(c) Copyright [2015] Hewlett Packard Enterprise Development LP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package i3s

import (
	"encoding/json"
	"errors"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
	"strings"
	"time"
)

type Task struct {
	AssociatedRes           AssociatedResource `json:"associatedResource,omitempty"`      // "associatedResource": { },
	AssociatedTaskURI       utils.Nstring      `json:"associatedTaskUri,omitempty"`       // "associatedTaskUri": null,
	Category                string             `json:"category,omitempty"`                // "category": "tasks",
	CompletedSteps          int                `json:"completedSteps,omitempty"`          // "completedSteps": 0,
	ComputedPercentComplete int                `json:"computedPercentComplete,omitempty"` // "computedPercentComplete": 0,
	Created                 string             `json:"created,omitempty"`                 // "created": "2015-09-07T03:25:54.844Z",
	Data                    TaskData           `json:"data,omitempty"`                    // "data": null,
	ETAG                    string             `json:"eTag,omitempty"`                    // "eTag": "0",
	ExpectedDuration        int                `json:"expectedDuration,omitempty"`        // "expectedDuration": 300,
	Hidden                  bool               `json:"hidden,omitempty"`                  // "hidden": false,
	Modified                string             `json:"modified,omitempty"`                // "modified": "2015-09-07T03:25:54.844Z",
	Name                    string             `json:"name,omitempty"`                    // "name": "Power off",
	Owner                   string             `json:"owner,omitempty"`                   // "owner": "i3sUser",
	ParentTaskURI           utils.Nstring      `json:"parentTaskUri,omitempty"`           // "parentTaskUri": null,
	PercentComplete         int                `json:"percentComplete,omitempty"`         // "percentComplete": 0,
	ProgressUpdates         []ProgressUpdate   `json:"progressUpdates,omitempty"`         // "progressUpdates": [],
	StateReason             string             `json:"stateReason,omitempty"`             // "stateReason": null,
	TaskErrors              []TaskError        `json:"taskErrors,omitempty"`              // "taskErrors": [],
	TaskOutput              []string           `json:"taskOutput,omitempty"`              // "taskOutput": [],
	TaskState               string             `json:"taskState,omitempty"`               // "taskState": "New",
	TaskStatus              string             `json:"taskStatus,omitempty"`              // "taskStatus": "Power off Server: se05, bay 16",
	TaskType                string             `json:"taskType,omitempty"`                // "taskType": "Appliance",
	TotalSteps              int                `json:"totalSteps,omitempty"`              // "totalSteps": 0,
	Type                    string             `json:"type,omitempty"`                    // "type": "TaskResourceV2",
	URI                     utils.Nstring      `json:"uri,omitempty"`                     // "uri": "/rest/tasks/145F808A-A8DD-4E1B-8C86-C2379C97B3B2"
	UserInitiated           bool               `json:"userInitiated,omitempty"`           // "userInitiated": true,
	TaskIsDone              bool               // when true, task are done
	Timeout                 int                // time before timeout on Executor
	WaitTime                time.Duration      // time between task checks
	Client                  *I3SClient
}

// AssociatedResource associated resource
type AssociatedResource struct {
	AssociationType  string        `json:"associationType,omitempty"`  // "associationType": "MANAGED_BY",
	ResourceCateogry string        `json:"resourceCategory,omitempty"` // "resourceCategory": "deployment-plan",
	ResourceName     utils.Nstring `json:"resourceName,omitempty"`     // "resourceName": "ubuntu 16.04",
	ResourceURI      utils.Nstring `json:"resourceUri,omitempty"`      // "resourceUri": "/rest/deployment-plans/30373237-3132-4D32-3235-303930524D57"
}

// TaskServer Example:
// {"name":"deployment plan name", "uri":"/rest/deployment-plans/30373237-3132-4D32-3235-303930524D52"}
type TaskServer struct {
	Name string `json:"name,omitempty"` // "Name to server
	URI  string `json:"uri,omitempty"`  // "URI to server
}

type TaskData struct {
	TaskCategory string `json:"task-category,omitempty"`
}

// ProgressUpdate - Task Progress Updates
type ProgressUpdate struct {
	ID           int    `json:"id,omitempty"`           // "id":12566
	StatusUpdate string `json:"statusUpdate,omitempty"` // "statusUpdate":"Apply server settings.",
	TimeStamp    string `json:"timestamp,omitempty"`    // "timestamp":"2015-09-10T22:50:14.250Z",
}

// TaskError struct
type TaskError struct {
	Data               map[string]interface{} `json:"data,omitempty"`               // "data":{},
	Details            string                 `json:"details,omitempty"`            // "details":"",
	ErrorCode          string                 `json:"errorCode,omitempty"`          // "errorCode":"MacTypeDiffGlobalMacType",
	ErrorSource        utils.Nstring          `json:"errorSource,omitempty"`        // "errorSource":null,
	Message            string                 `json:"message,omitempty"`            // "message":"When macType is not user defined, mac type should be same as the global Mac assignment Virtual."
	NestedErrors       []TaskError            `json:"nestedErrors,omitempty"`       // "nestedErrors":[],
	RecommendedActions []string               `json:"recommendedActions,omitempty"` // "recommendedActions":["Verify parameters and try again."],
}

// TaskState task state
type TaskState int

const (
	T_COMPLETED TaskState = 1 + iota
	T_ERROR
	T_INERRUPTED
	T_KILLED
	T_NEW
	T_PENDING
	T_RUNNING
	T_STARTING
	T_STOPPING
	T_SUSPENDED
	T_TERMINATED
	T_UNKNOWN
	T_WARNING
)

var taskstate = [...]string{
	"Completed",   // Completed Task has been completed.
	"Error",       // Error Task has terminated with an error.
	"Interrupted", // Interrupted Task has been interrupted.
	"Killed",      // Killed Task has been killed.
	"New",         // New Task is new.
	"Pending",     // Pending Task is in pending state.
	"Running",     // Running Task is running.
	"Starting",    // Starting Task is starting.
	"Stopping",    // Stopping Task is stopping.
	"Suspended",   // Suspended Task is suspended.
	"Terminated",  // Terminated Task has been terminated.
	"Unknown",     // Unknown State of task is unknown.
	"Warning",     // Warning Task has terminated with a warning.
}

// String for type
func (ts TaskState) String() string { return taskstate[ts-1] }

// Equal for type
func (ts TaskState) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(ts.String())) }

// TaskType - task type
type TaskType int

const (
	T_APPLIANCE TaskType = 1 + iota
	T_BACKGROUND
	T_USER
)

var tasktype = [...]string{
	"Applicance", // Appliance Task is appliance initiated and shows in notification panel.
	"Background", // Background Task is appliance initiated and does not show in notification panel.
	"User",       // User Task is user initiated and shows in notification panel.
}

// String return
func (tt TaskType) String() string { return tasktype[tt-1] }

// Equal type
func (tt TaskType) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(tt.String())) }

// NewTask - Create New Task
func (t *Task) NewTask(c *I3SClient) *Task {
	return &Task{TaskIsDone: false,
		Client:   c,
		URI:      "",
		Name:     "",
		Owner:    "",
		Timeout:  144, // default 24min
		WaitTime: 10}  // default 10sec, impacts Timeout
}

// ResetTask - reset the power task back to off
func (t *Task) ResetTask() {
	t.TaskIsDone = false
	t.URI = ""
	t.Name = ""
	t.Owner = ""
}

// GetCurrentTaskStatus - Get the current status
func (t *Task) GetCurrentTaskStatus() error {
	log.Debugf("Working on getting current task status")
	var (
		uri = t.URI
	)
	if uri != "" {
		log.Debugf(uri.String())
		data, err := t.Client.RestAPICall(rest.GET, uri.String(), nil)
		if err != nil {
			return err
		}
		log.Debugf("data: %s", data)
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			return err
		}
	} else {
		log.Debugf("Unable to get current task, no URI found")
	}
	if len(t.TaskErrors) > 0 {
		var errmsg string
		errmsg = ""
		for _, te := range t.TaskErrors {
			errmsg += te.Message + " \n" + strings.Join(te.RecommendedActions, " ")
		}
		return errors.New(errmsg)
	}
	return nil
}

// GetLastStatusUpdate - get last detail updates from task
func (t *Task) GetLastStatusUpdate() string {
	if len(t.ProgressUpdates) > 0 {
		lastupdate := len(t.ProgressUpdates) - 1
		// sanatize a little by removing json
		message := utils.StringRemoveJSON(t.ProgressUpdates[lastupdate].StatusUpdate)
		// parse out server name
		servernamejson := utils.StringGetJSON(t.ProgressUpdates[lastupdate].StatusUpdate)
		var ts *TaskServer
		if err := json.Unmarshal([]byte(servernamejson), &ts); err == nil {
			message += ts.Name
		}
		return t.TaskStatus + ", " + message
	}
	return t.TaskStatus
}

// Wait - wait on task to complete
func (t *Task) Wait() error {
	var (
		currenttime int
	)
	log.Debugf("task : %+v", t)
	if t.Timeout < t.ExpectedDuration {
		t.Timeout = t.ExpectedDuration
		log.Debugf("assign timeout %d", t.Timeout)
	}
	log.Debugf("task timeout is : %d", t.Timeout)
	for !t.TaskIsDone && (currenttime < t.Timeout) {
		if err := t.GetCurrentTaskStatus(); err != nil {
			t.TaskIsDone = true
			return err
		}
		if t.URI != "" && T_COMPLETED.Equal(t.TaskState) {
			t.TaskIsDone = true
		}
		if t.URI != "" {
			log.Debugf("Waiting for task to complete, for %s ", t.Name)
			log.Debugf("Waiting on, %s, %d%%, %s, %d, %d", t.Name, t.ComputedPercentComplete, t.GetLastStatusUpdate(), currenttime, t.ExpectedDuration)
			log.Infof("Waiting on, %s, %d%%, %s", t.Name, t.ComputedPercentComplete, t.GetLastStatusUpdate())
		} else {
			log.Info("Waiting on task creation.")
		}

		// wait time before next check
		time.Sleep(time.Millisecond * (1000 * t.WaitTime)) // wait 10sec before checking the status again
		currenttime++
		if t.Timeout < t.ExpectedDuration {
			t.Timeout = t.ExpectedDuration
		}
	}
	if currenttime > t.Timeout {
		log.Warnf("Task timed out, %d.", currenttime)
	}

	if t.Name != "" {
		log.Infof("Task, %s, completed", t.Name)
	}
	return nil
}
