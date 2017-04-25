/*
The gomanta/manta package interacts with the Manta API (http://apidocs.joyent.com/manta/api.html).

This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at http://mozilla.org/MPL/2.0/.

Copyright (c) 2016 Joyent Inc.
Written by Daniele Stroppa <daniele.stroppa@joyent.com>

*/
package manta

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"

	jh "github.com/joyent/gocommon/http"
)

const (
	// The default version of the Manta API to use
	DefaultAPIVersion = "7.1"

	// Manta API URL parts
	apiStorage    = "stor"
	apiJobs       = "jobs"
	apiJobsLive   = "live"
	apiJobsIn     = "in"
	apiJobsOut    = "out"
	apiJobsFail   = "fail"
	apiJobsErr    = "err"
	apiJobsEnd    = "end"
	apiJobsCancel = "cancel"
	apiJobsStatus = "status"
)

// Client provides a means to access Joyent Manta
type Client struct {
	client client.Client
}

// New creates a new Client.
func New(client client.Client) *Client {
	return &Client{client}
}

// request represents an API request
type request struct {
	method         string
	url            string
	reqValue       interface{}
	reqHeader      http.Header
	reqReader      io.Reader
	reqLength      int
	resp           interface{}
	respHeader     *http.Header
	expectedStatus int
}

// Helper method to send an API request
func (c *Client) sendRequest(req request) (*jh.ResponseData, error) {
	request := jh.RequestData{
		ReqValue:   req.reqValue,
		ReqHeaders: req.reqHeader,
		ReqReader:  req.reqReader,
		ReqLength:  req.reqLength,
	}
	if req.expectedStatus == 0 {
		req.expectedStatus = http.StatusOK
	}
	respData := jh.ResponseData{
		RespValue:      req.resp,
		RespHeaders:    req.respHeader,
		ExpectedStatus: []int{req.expectedStatus},
	}
	err := c.client.SendRequest(req.method, req.url, "", &request, &respData)
	return &respData, err
}

// Helper method to create the API URL
func makeURL(parts ...string) string {
	return path.Join(parts...)
}

// ListDirectoryOpts represent the option that can be specified
// when listing a directory.
type ListDirectoryOpts struct {
	Limit  int    `json:"limit"`  // Limit to the number of records returned (default and max is 1000)
	Marker string `json:"marker"` // Key name at which to start the next listing
}

// Entry represents an object stored in Manta, either a file or a directory
type Entry struct {
	Name  string `json:"name"`           // Entry name
	Etag  string `json:"etag,omitempty"` // If type is 'object', object UUID
	Size  int    `json:"size,omitempty"` // If type is 'object', object size (content-length)
	Type  string `json:"type"`           // Entry type, one of 'directory' or 'object'
	Mtime string `json:"mtime"`          // ISO8601 timestamp of the last update
}

// Creates a directory at the specified path. Any parent directory must exist.
// See API docs: http://apidocs.joyent.com/manta/api.html#PutDirectory
func (c *Client) PutDirectory(path string) error {
	requestHeaders := make(http.Header)
	requestHeaders.Set("Content-Type", "application/json; type=directory")
	requestHeaders.Set("Accept", "*/*")
	req := request{
		method:         client.PUT,
		url:            makeURL(apiStorage, path),
		reqHeader:      requestHeaders,
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to create directory: %s", path)
	}
	return nil
}

// Returns the content of the specified directory, using the specified options.
// See API docs: http://apidocs.joyent.com/manta/api.html#ListDirectory
func (c *Client) ListDirectory(directory string, opts ListDirectoryOpts) ([]Entry, error) {
	var resp []Entry
	requestHeaders := make(http.Header)
	requestHeaders.Set("Accept", "*/*")
	req := request{
		method:    client.GET,
		url:       makeURL(apiStorage, directory),
		reqHeader: requestHeaders,
		resp:      &resp,
		reqValue:  opts,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to list directory %s", directory)
	}
	return resp, nil
}

// Deletes the specified directory. Directory must be empty.
// See API docs: http://apidocs.joyent.com/manta/api.html#DeleteDirectory
func (c *Client) DeleteDirectory(path string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiStorage, path),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete directory %s", path)
	}
	return nil
}

// Creates an object at the specified path. Any parent directory must exist.
// See API docs: http://apidocs.joyent.com/manta/api.html#PutObject
func (c *Client) PutObject(path, objectName string, object []byte) error {
	r := bytes.NewReader(object)
	req := request{
		method:         client.PUT,
		url:            makeURL(apiStorage, path, objectName),
		reqReader:      r,
		reqLength:      len(object),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to create object: %s/%s", path, objectName)
	}
	return nil
}

// Retrieves the specified object from the specified location.
// See API docs: http://apidocs.joyent.com/manta/api.html#GetObject
func (c *Client) GetObject(path, objectName string) ([]byte, error) {
	var resp []byte
	requestHeaders := make(http.Header)
	requestHeaders.Set("Accept", "*/*")
	req := request{
		method:    client.GET,
		url:       makeURL(apiStorage, path, objectName),
		reqHeader: requestHeaders,
		resp:      &resp,
	}
	respData, err := c.sendRequest(req)
	if err != nil {
		return nil, errors.Newf(err, "failed to get object %s/%s", path, objectName)
	}
	res, ok := respData.RespValue.(*[]byte)
	if !ok {
		return nil, errors.Newf(err, "failed to assert downloaded data as type *[]byte for object %s/%s", path, objectName)
	}
	return *res, nil
}

// Deletes the specified object from the specified location.
// See API docs: http://apidocs.joyent.com/manta/api.html#DeleteObject
func (c *Client) DeleteObject(path, objectName string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiStorage, path, objectName),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete object %s/%s", path, objectName)
	}
	return nil
}

// Creates a link (similar to a Unix hard link) from location to path/linkName.
// See API docs: http://apidocs.joyent.com/manta/api.html#PutSnapLink
func (c *Client) PutSnapLink(path, linkName, location string) error {
	requestHeaders := make(http.Header)
	requestHeaders.Set("Accept", "application/json; type=link")
	requestHeaders.Set("Location", location)
	req := request{
		method:         client.PUT,
		url:            makeURL(apiStorage, path, linkName),
		reqHeader:      requestHeaders,
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to create snap link: %s/%s", path, linkName)
	}
	return nil
}

// CreateJobOpts represent the option that can be specified
// when creating a job.
type CreateJobOpts struct {
	Name   string  `json:"name,omitempty"` // Job Name (optional)
	Phases []Phase `json:"phases"`         // Tasks to execute as part of this job
}

// Job represents the status of a job.
type Job struct {
	Id                 string      // Job unique identifier
	Name               string      `json:"name,omitempty"` // Job Name
	State              string      // Job state
	Cancelled          bool        // Whether the job has been cancelled or not
	InputDone          bool        // Whether the inputs for the job is still open or not
	Stats              JobStats    `json:"stats,omitempty"` // Job statistics
	TimeCreated        string      // Time the job was created at
	TimeDone           string      `json:"timeDone,omitempty"`           // Time the job was completed
	TimeArchiveStarted string      `json:"timeArchiveStarted,omitempty"` // Time the job archiving started
	TimeArchiveDone    string      `json:"timeArchiveDone,omitempty"`    // Time the job archiving completed
	Phases             []Phase     `json:"phases"`                       // Job tasks
	Options            interface{} // Job options
}

// JobStats represents statistics about a job
type JobStats struct {
	Errors    int // Number or errors
	Outputs   int // Number of output produced
	Retries   int // Number of retries
	Tasks     int // Total number of task in the job
	TasksDone int // number of tasks done
}

// Phase represents a task to be executed as part of a Job
type Phase struct {
	Type   string   `json:"type,omitempty"`   // Task type, one of 'map' or 'reduce' (optional)
	Assets []string `json:"assets,omitempty"` // An array of objects to be placed in the compute zones (optional)
	Exec   string   `json:"exec"`             // The actual shell statement to execute
	Init   string   `json:"init"`             // Shell statement to execute in each compute zone before any tasks are executed
	Count  int      `json:"count,omitempty"`  // If type is 'reduce', an optional number of reducers for this phase (default is 1)
	Memory int      `json:"memory,omitempty"` // Amount of DRAM to give to your compute zone (in Mb, optional)
	Disk   int      `json:"disk,omitempty"`   // Amount of disk space to give to your compute zone (in Gb, optional)
}

// JobError represents an error occurred during a job execution
type JobError struct {
	Id      string // Job Id
	Phase   string // Phase number of the failure
	What    string // A human readable summary of what failed
	Code    string // Error code
	Message string // Human readable error message
	Stderr  string // A key that saved the stderr for the given command (optional)
	Key     string // The input key being processed when the task failed (optional)
}

// Creates a job with the given options.
// See API docs: http://apidocs.joyent.com/manta/api.html#CreateJob
func (c *Client) CreateJob(opts CreateJobOpts) (string, error) {
	var resp string
	var respHeader http.Header
	req := request{
		method:         client.POST,
		url:            apiJobs,
		reqValue:       opts,
		respHeader:     &respHeader,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	respData, err := c.sendRequest(req)
	if err != nil {
		return "", errors.Newf(err, "failed to create job with name: %s", opts.Name)
	}
	return respData.RespHeaders.Get("Location"), nil
}

// Submits inputs to an already created job.
// See API docs: http://apidocs.joyent.com/manta/api.html#AddJobInputs
func (c *Client) AddJobInputs(jobId string, jobInputs io.Reader) error {
	inputData, errI := ioutil.ReadAll(jobInputs)
	if errI != nil {
		return errors.Newf(errI, "failed to read inputs for job %s", jobId)
	}
	requestHeaders := make(http.Header)
	requestHeaders.Set("Accept", "*/*")
	requestHeaders.Set("Content-Type", "text/plain")
	req := request{
		method:         client.POST,
		url:            makeURL(apiJobs, jobId, apiJobsLive, apiJobsIn),
		reqValue:       string(inputData),
		reqHeader:      requestHeaders,
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to add inputs to job %s", jobId)
	}
	return nil
}

// This closes input for a job, and finalize the job.
// See API docs: http://apidocs.joyent.com/manta/api.html#EndJobInput
func (c *Client) EndJobInputs(jobId string) error {
	req := request{
		method:         client.POST,
		url:            makeURL(apiJobs, jobId, apiJobsLive, apiJobsIn, apiJobsEnd),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to end inputs for job %s", jobId)
	}
	return nil
}

// This cancels a job from doing any further work.
// Cancellation is asynchronous and "best effort"; there is no guarantee the job will actually stop
// See API docs: http://apidocs.joyent.com/manta/api.html#CancelJob
func (c *Client) CancelJob(jobId string) error {
	req := request{
		method:         client.POST,
		url:            makeURL(apiJobs, jobId, apiJobsLive, apiJobsCancel),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to cancel job %s", jobId)
	}
	return nil
}

// Returns the list of jobs.
// Note you can filter the set of jobs down to only live jobs by setting the liveOnly flag.
// See API docs: http://apidocs.joyent.com/manta/api.html#ListJobs
func (c *Client) ListJobs(liveOnly bool) ([]Entry, error) {
	var resp []Entry
	var url string
	if liveOnly {
		url = fmt.Sprintf("%s?state=running", apiJobs)
	} else {
		url = apiJobs
	}
	req := request{
		method: client.GET,
		url:    url,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to list jobs")
	}
	return resp, nil
}

// Gets the high-level job container object for a given job.
// See API docs: http://apidocs.joyent.com/manta/api.html#GetJob
func (c *Client) GetJob(jobId string) (Job, error) {
	var resp Job
	req := request{
		method: client.GET,
		url:    makeURL(apiJobs, jobId, apiJobsLive, apiJobsStatus),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return Job{}, errors.Newf(err, "failed to get job with id: %s", jobId)
	}
	return resp, nil
}

// Returns the current "live" set of outputs from a given job.
// See API docs: http://apidocs.joyent.com/manta/api.html#GetJobOutput
func (c *Client) GetJobOutput(jobId string) (string, error) {
	var resp string
	req := request{
		method: client.GET,
		url:    makeURL(apiJobs, jobId, apiJobsLive, apiJobsOut),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return "", errors.Newf(err, "failed to get output for job with id: %s", jobId)
	}
	return resp, nil
}

// Returns the submitted input objects for a given job, available while the job is running.
// See API docs: http://apidocs.joyent.com/manta/api.html#GetJobInput
func (c *Client) GetJobInput(jobId string) (string, error) {
	var resp string
	req := request{
		method: client.GET,
		url:    makeURL(apiJobs, jobId, apiJobsLive, apiJobsIn),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return "", errors.Newf(err, "failed to get input for job with id: %s", jobId)
	}
	return resp, nil
}

// Returns the current "live" set of failures from a given job.
// See API docs: http://apidocs.joyent.com/manta/api.html#GetJobFailures
func (c *Client) GetJobFailures(jobId string) (interface{}, error) {
	var resp interface{}
	req := request{
		method: client.GET,
		url:    makeURL(apiJobs, jobId, apiJobsLive, apiJobsFail),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get failures for job with id: %s", jobId)
	}
	return resp, nil
}

// Returns the current "live" set of errors from a given job.
// See API docs: http://apidocs.joyent.com/manta/api.html#GetJobErrors
func (c *Client) GetJobErrors(jobId string) ([]JobError, error) {
	var resp []JobError
	req := request{
		method: client.GET,
		url:    makeURL(apiJobs, jobId, apiJobsLive, apiJobsErr),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get errors for job with id: %s", jobId)
	}
	return resp, nil
}

// Returns a signed URL to retrieve the object at path.
func (c *Client) SignURL(path string, expires time.Time) (string, error) {
	return c.client.SignURL(path, expires)
}
