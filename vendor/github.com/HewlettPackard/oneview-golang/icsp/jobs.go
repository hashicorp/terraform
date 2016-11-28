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

// Package icsp -
package icsp

import (
	"encoding/json"
	"strings"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// ElementJobStatus type
type ElementJobStatus int

const (
	E_STATUS_ERROR ElementJobStatus = 1 + iota
	E_STATUS_OK
	E_STATUS_PENDING
	E_STATUS_WARNING
)

var elementjobstatuslist = [...]string{
	"STATUS_ERROR",   // - Status for server that have errors;
	"STATUS_OK",      // - Status for server that have no errors;
	"STATUS_PENDING", // - Status for server that are pending;
	"STATUS_WARNING", // - Status for server that have warnings;
}

// String helper for ElementJobStatus
func (o ElementJobStatus) String() string { return elementjobstatuslist[o-1] }

// Equal helper for ElementJobStatus
func (o ElementJobStatus) Equal(s string) bool {
	return (strings.ToUpper(s) == strings.ToUpper(o.String()))
}

// JobServerInclusionStatus type
type JobServerInclusionStatus int

const (
	ADDED_INCLUSION_STATUS JobServerInclusionStatus = 1 + iota
	INCLUDED_INCLUSION_STATUS
	REMOVED_INCLUSION_STATUS
)

var jobserverinclusionstatus = [...]string{
	"ADDED_INCLUSION_STATUS",    // - Inclusion status to indicate that a server was added at run time because membership in a server group changed;
	"INCLUDED_INCLUSION_STATUS", // - Inclusion status to indicate that a server was present at schedule time and run time, either directly or because of being a member of a server group;
	"REMOVED_INCLUSION_STATUS",  // - Inclusion status to indicate that a server was present at schedule time but removed at run time because membership in a server group changed;
}

// String helper for JobServerInclusionStatus
func (o JobServerInclusionStatus) String() string { return jobserverinclusionstatus[o-1] }

// Equal helper for JobServerInclusionStatus
func (o JobServerInclusionStatus) Equal(s string) bool {
	return (strings.ToUpper(s) == strings.ToUpper(o.String()))
}

// JobStatusOnServer Status of the Job on a server
type JobStatusOnServer int

const (
	J_CANCELLED_STATUS JobStatusOnServer = 1 + iota
	J_FAILURE_STATUS
	J_SKIPPED_STATUS
	J_SUCCESS_STATUS
	J_UNKNOWN_STATUS
	J_WARNING_STATUS
)

var jobstatusonserver = [...]string{
	"CANCELLED_STATUS", // - Status value to indicate that the job did not process this server because it was terminated by user request;
	"FAILURE_STATUS",   // - Status value to indicate the job failed on this server;
	"SKIPPED_STATUS",   // - Status value to indicate the job skipped this server;
	"SUCCESS_STATUS",   // - Status value to indicate the job succeed on this server;
	"UNKNOWN_STATUS",   // - Status value before a job runs or if Opsware did not report status about the server when the job finished;
	"WARNING_STATUS",   // - Status value to indicate a job completed but had warning for this server;
}

// String helper for JobStatusOnServer
func (o JobStatusOnServer) String() string { return jobstatusonserver[o-1] }

// Equal helper for JobStatusOnServer
func (o JobStatusOnServer) Equal(s string) bool {
	return (strings.ToUpper(s) == strings.ToUpper(o.String()))
}

// JobState type
type JobState int

const (
	STATUS_ABORTED JobState = 1 + iota
	STATUS_ACTIVE
	STATUS_BLOCKED
	STATUS_CANCELLED
	STATUS_DELETED
	STATUS_EXPIRED
	STATUS_FAILURE
	STATUS_PENDING
	STATUS_RECURRING
	STATUS_STALE
	STATUS_SUCCESS
	STATUS_TERMINATED
	STATUS_TERMINATING
	STATUS_UNKNOWN
	STATUS_WARNING
	STATUS_ZOMBIE
)

var jobstatelist = [...]string{
	"STATUS_ABORTED",     // - The job has finished running and a failure has been detected;
	"STATUS_ACTIVE",      // - The job is currently running;
	"STATUS_BLOCKED",     // - The job was blocked;
	"STATUS_CANCELLED",   // - The job was scheduled but has been canceled;
	"STATUS_DELETED",     // - The job was deleted;
	"STATUS_EXPIRED",     // - The current date is later than the job schedule's end date, so the job schedule is no longer in effect;
	"STATUS_FAILURE",     // - The job has finished running and an error has been detected;
	"STATUS_PENDING",     // - The job is scheduled to run in the future;
	"STATUS_RECURRING",   // - The job is scheduled to run repeatedly in the future;
	"STATUS_STALE",       // - The job became stale;
	"STATUS_SUCCESS",     // - The job has finished running successfully	;
	"STATUS_TERMINATED",  // - The job ended early in response to a user request;
	"STATUS_TERMINATING", // - The user has requested that the job end, and it is in the process of shutting down;
	"STATUS_UNKNOWN",     // - The status of the job is unknown;
	"STATUS_WARNING",     // - The job has finished running and a warning has been detected;
	"STATUS_ZOMBIE",      // - The command engine was stopped while the job was running, leaving it in an orphaned state
}

// String helper for JobState
func (o JobState) String() string { return jobstatelist[o-1] }

// Equal helper for JobState
func (o JobState) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(o.String())) }

// JobStatus type
type JobStatus int

const (
	JOB_STATUS_OK JobStatus = 1 + iota
	JOB_STATUS_WARNING
	JOB_STATUS_ERROR
	JOB_STATUS_UNKNOWN
)

var jobstatuslist = [...]string{
	"ok",      // The job was completed successfully
	"warning", // The job completed with warnings
	"error",   // The job had errors
	"unknown", // The status of the Job is unknown
}

// String helper for JobStatus
func (o JobStatus) String() string { return jobstatuslist[o-1] }

// Equal helper for JobStatus
func (o JobStatus) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(o.String())) }

// JobRunning running state
type JobRunning int

const (
	JOB_RUNNING_YES JobRunning = 1 + iota
	JOB_RUNNING_NO
)

var jobrunninglist = [...]string{
	"TRUE",  // The job is running
	"FALSE", // The job is not running
}

// String helper for JobRunning
func (o JobRunning) String() string { return jobrunninglist[o-1] }

// Equal helper for JobStatus
func (o JobRunning) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(o.String())) }

// OSDJobServerInfo struct
type OSDJobServerInfo struct {
	DeviceType               string        `json:"deviceType,omitempty"`               // deviceType The only supported type: os-deployment-servers, string
	JobServerInclusionStatus string        `json:"jobServerInclusionStatus,omitempty"` // jobServerInclusionStatus Information about a server that was affected by a job. Each server has information about how it was associated with the job (inclusion status) and the status of the job on that server (status). Inclusion values: string
	JobServerURI             utils.Nstring `json:"jobServerUri,omitempty"`             // jobServerUri The canonical URI of a server within the Job, string
	JobStatusOnServer        string        `json:"jobStatusOnServer,omitempty"`        // jobStatusOnServer Status of the Job on a server where this Job was executed. Status values: string
	ServerName               string        `json:"serverName,omitempty"`               // serverName Name of the server string
}

// OSDJobResult struct
type OSDJobResult struct {
	JobMessage              string        `json:"jobMessage,omitempty"`              // jobMessage Job result message  , string
	JobResultCompletedSteps int           `json:"jobResultCompletedSteps,omitempty"` // jobResultCompletedSteps Total number of completed steps  , integer
	JobResultErrorDetails   string        `json:"jobResultErrorDetails,omitempty"`   // jobResultErrorDetails Error details for the Job  , string
	JobResultLogDetails     string        `json:"jobResultLogDetails,omitempty"`     // jobResultLogDetails Log details for the Job  , string
	JobResultTotalSteps     int           `json:"jobResultTotalSteps,omitempty"`     // jobResultTotalSteps Total number of steps for the Job  , integer
	JobServerURI            utils.Nstring `json:"jobServerUri,omitempty"`            // jobServerUri The canonical URI of a server within the Job  , string
}

// OSDJobProgress struct
type OSDJobProgress struct {
	CurrentStepName   string        `json:"currentStepName,omitempty"`   // currentStepName The name of the step that this Job is currently on  , string
	ElementJobStatus  string        `json:"elementJobStatus,omitempty"`  // elementJobStatus The status of an individual server within the Job, string
	JobCompletedSteps int           `json:"jobCompletedSteps,omitempty"` // jobCompletedSteps Total number of completed steps of the Job, integer
	JobServerURI      utils.Nstring `json:"jobServerUri,omitempty"`      // jobServerUri The canonical URI of a server within the Job, string
	JobTotalSteps     int           `json:"jobTotalSteps,omitempty"`     // jobTotalSteps Total number of steps that the Job has, integer
}

// Job type
type Job struct {
	Category        string             `json:"category,omitempty"`        // category The category is used to help identify the kind of resource, string
	Created         string             `json:"created,omitempty"`         // created Date and time when the Job was created, timestamp
	Description     string             `json:"description,omitempty"`     // description Text description of the type of the Job, string
	ETAG            string             `json:"eTag,omitempty"`            // eTag Entity tag/version ID of the resource , string
	JobDeviceGroups []string           `json:"jobDeviceGroups,omitempty"` // jobDeviceGroups An array of device groups associated with this Job , array of string
	JobProgress     []OSDJobProgress   `json:"jobProgress,omitempty"`     // jobProgress An array of Job progress. A single Job can contain progress for multiple servers. Job progress is only available when the job is running. For a single Job this is the number of steps competed for this Job on the target server. When a set of jobs is run together user will see one Job listed. The progress for this Job is the number of jobs that have been completed
	JobResult       []OSDJobResult     `json:"jobResult,omitempty"`       // jobResult  An array of Job results. A single Job can contain results for multiple servers. Job result is only available once the Job completes. For a single Job this provides total steps completed, errors that happened during Job execution and logs
	JobServerInfo   []OSDJobServerInfo `json:"jobServerInfo,omitempty"`   // jobServerInfo An array of servers and their details associated with this Job
	JobUserName     string             `json:"jobUserName,omitempty"`     // jobUserName The name of the user under whose authority this Job was invoked string
	Modified        string             `json:"modified,omitempty"`        // modified Date and time when the Job was last modified timestamp
	Name            string             `json:"name,omitempty"`            // name Name of the job string
	NameOfJobType   string             `json:"nameOfJobType,omitempty"`   // nameOfJobType The name of the type of the Job. In some cases has the same value as the field "name" string
	Running         string             `json:"running,omitempty"`         // running Indicates whether the Job is running
	State           string             `json:"state,omitempty"`           // JobState  state A constant to help explain what state a Job is in. Possible values:
	Status          string             `json:"status,omitempty"`          // status Overall status of the Job. Values: string
	Type            string             `json:"type,omitempty"`            // type Uniquely identifies the type of the JSON object
	TypeOfJobType   string             `json:"typeOfJobType,omitempty"`   // typeOfJobType A constant that indicates what type of Job it is:
	URI             utils.Nstring      `json:"uri,omitempty"`             // uri The canonical URI of the resource string
	URIOfJobType    utils.Nstring      `json:"uriOfJobType,omitempty"`    // uriOfJobType The canonical URI of the OS Build Plan string
}

// determine if Running property is 'TRUE' or 'FALSE'
func (o Job) isRunning() bool {
	if strings.ToUpper(o.Running) == "TRUE" {
		return true
	}
	return false
}

// JobsList List of jobs
type JobsList struct {
	Category    string        `json:"category,omitempty"`    // Resource category used for authorizations and resource type groupings
	Count       int           `json:"count,omitempty"`       // The actual number of resources returned in the specified page
	Created     string        `json:"created,omitempty"`     // timestamp for when resource was created
	ETAG        string        `json:"eTag,omitempty"`        // entity tag version id
	Members     []Job         `json:"members,omitempty"`     // array of Server types
	Modified    string        `json:"modified,omitempty"`    // timestamp resource last modified
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // Next page resources
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // Previous page resources
	Start       int           `json:"start,omitempty"`       // starting row of resource for current page
	Total       int           `json:"total,omitempty"`       // total number of pages
	Type        string        `json:"type,omitempty"`        // type of paging
	URI         utils.Nstring `json:"uri,omitempty"`         // uri to page
}

// GetJobs get a jobs from icsp
func (c *ICSPClient) GetJobs() (JobsList, error) {
	var (
		uri  = "/rest/os-deployment-jobs"
		jobs JobsList
	)
	//TODO: need to ask icsp team how we can get limitted data set
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return jobs, err
	}

	log.Debugf("GetJobs %+v", data)
	if err := json.Unmarshal([]byte(data), &jobs); err != nil {
		return jobs, err
	}
	return jobs, nil
}

// GetJob get a job with the ODSUri
func (c *ICSPClient) GetJob(u ODSUri) (Job, error) {
	var (
		job Job
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, u.URI.String(), nil)
	if err != nil {
		return job, err
	}

	log.Debugf("GetJob %+v", data)
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		return job, err
	}
	return job, nil
}
