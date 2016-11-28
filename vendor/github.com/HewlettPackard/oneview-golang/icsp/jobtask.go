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
	"errors"
	"time"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// ODSUri  returned from create server for job uri task
type ODSUri struct {
	URI utils.Nstring `json:"uri,omitempty"` // uri of job
}

// JobTask holds a Job ODSUri and task status
type JobTask struct {
	Job                    // copy of the original job
	JobURI   ODSUri        // link to the job
	IsDone   bool          // when true, task are done
	Timeout  int           // time before timeout on Executor
	WaitTime time.Duration // time between task checks
	Client   *ICSPClient   // reference to a client
}

// NewJobTask create a new job task
func (jt *JobTask) NewJobTask(c *ICSPClient) *JobTask {
	return &JobTask{
		IsDone:   false,
		Client:   c,
		Timeout:  360, // default 1hr
		WaitTime: 10}  // default 10sec, impacts Timeout
}

// Reset - reset job task
func (jt *JobTask) Reset() {
	jt.IsDone = false
}

// GetCurrentStatus - Get the current status
func (jt *JobTask) GetCurrentStatus() error {
	log.Debugf("Working on getting current job status")
	if jt.JobURI.URI != "" {
		log.Debugf(jt.JobURI.URI.String())
		data, err := jt.Client.RestAPICall(rest.GET, jt.JobURI.URI.String(), nil)
		if err != nil {
			return err
		}
		log.Debugf("data: %s", data)
		if err := json.Unmarshal([]byte(data), &jt); err != nil {
			return err
		}
	} else {
		log.Debugf("Unable to get current job, no URI found")
	}
	if JOB_STATUS_ERROR.Equal(jt.Status) {
		var errmsg string
		errmsg = ""
		for _, je := range jt.JobResult {
			if je.JobResultErrorDetails != "" {
				errmsg += je.JobMessage + " \n" + je.JobResultErrorDetails + "\n"
			}
		}
		return errors.New(errmsg)
	}
	return nil
}

// GetLastStatusUpdate get the last status from JobProgress
func (jt *JobTask) GetLastStatusUpdate() string {
	lastjobstep := len(jt.JobProgress)
	if lastjobstep > 0 {
		return jt.JobProgress[lastjobstep-1].CurrentStepName
	}
	return ""
}

// GetComplettedStatus  get the message from JobResult
func (jt *JobTask) GetComplettedStatus() string {
	lastjobstep := len(jt.JobResult)
	if lastjobstep > 0 {
		return jt.JobResult[lastjobstep-1].JobMessage
	}
	return ""

}

// GetPercentProgress get the progress as a percentage
func (jt *JobTask) GetPercentProgress() float64 {
	var progress float64
	lastjobstep := len(jt.JobProgress)
	stepscompleted := jt.JobProgress[lastjobstep-1].JobCompletedSteps + 1
	totalcompleted := jt.JobProgress[lastjobstep-1].JobTotalSteps + 1
	if totalcompleted > 1 {
		progress = (float64(stepscompleted) / float64(totalcompleted)) * 100
	}
	log.Debugf("steps => %d, totalsteps => %d, progress => %0.0f", stepscompleted, totalcompleted, progress)
	return progress
}

// Wait - wait on job task to complete
func (jt *JobTask) Wait() error {
	var (
		currenttime int
	)
	log.Debugf("task : %+v", jt)
	if err := jt.GetCurrentStatus(); err != nil {
		jt.IsDone = true
		return err
	}

	for JOB_RUNNING_YES.Equal(jt.Running) && (currenttime < jt.Timeout) {
		log.Debugf("jt => %+v", jt)
		if jt.JobURI.URI.String() != "" {
			log.Debugf("Waiting for job to complete, %s ", jt.Description)
			lastjobstep := len(jt.JobProgress)
			if lastjobstep > 0 {
				statusmessage := jt.GetLastStatusUpdate()
				if statusmessage == "" {
					log.Infof("Waiting on, %s, %0.0f%%", jt.Description, jt.GetPercentProgress())
				} else {
					log.Infof("Waiting on, %s, %0.0f%%, %s", jt.Description, jt.GetPercentProgress(), statusmessage)
				}
			}
		} else {
			log.Info("Waiting on job creation.")
		}

		// wait time before next check
		time.Sleep(time.Millisecond * (1000 * jt.WaitTime)) // wait 10sec before checking the status again
		currenttime++

		// get the current status
		if err := jt.GetCurrentStatus(); err != nil {
			jt.IsDone = true
			return err
		}
	}
	if !(currenttime < jt.Timeout) {
		log.Warn("Task timed out.")
	}

	if JOB_RUNNING_NO.Equal(jt.Running) {
		log.Infof("Job, %s, completed", jt.GetComplettedStatus())
	} else {
		log.Warn("Job still running un-expected.")
	}
	jt.IsDone = true
	return nil
}
