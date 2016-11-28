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

// Package ov -
package ov

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/docker/machine/libmachine/log"
)

// Create a PowerState type
type PowerState int

type Power struct {
	Blade      *ServerHardware
	State      PowerState
	TaskStatus bool
}

const (
	P_ON PowerState = 1 + iota
	P_OFF
	P_UKNOWN
)

var powerstates = [...]string{
	"On",
	"Off",
	"UNKNOWN",
}

func (p PowerState) String() string      { return powerstates[p-1] }
func (p PowerState) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(p.String())) }

// Power control
type PowerControl int

const (
	P_COLDBOOT PowerControl = 1 + iota
	P_MOMPRESS
	P_PRESSANDHOLD
	P_RESET
)

var powercontrols = [...]string{
	"ColdBoot", // ColdBoot       - A hard reset that immediately removes power from the server
	//                hardware and then restarts the server after approximately six seconds.
	"MomentaryPress", // MomentaryPress - Power on or a normal (soft) power off,
	"PressAndHold",   //                  depending on powerState. PressAndHold
	//                  An immediate (hard) shutdown.
	"Reset", // Reset          - A normal server reset that resets the device in an orderly sequence.
}

func (pc PowerControl) String() string { return powercontrols[pc-1] }

// Provides power execution status
type PowerTask struct {
	Blade ServerHardware
	State PowerState // current power state
	Task
}

// Create a new power task manager
// TODO: refactor PowerTask to use Task vs overloading it here.
func (pt *PowerTask) NewPowerTask(b ServerHardware) *PowerTask {
	pt = &PowerTask{Blade: b,
		State: P_UKNOWN} //,
	// TaskIsDone:  false,
	// Client:      b.Client,
	// URI:         "",
	// Name:        "",
	// Owner:       "",
	// Timeout:     36, // default 6min
	// WaitTime:    10} // default 10sec, impacts Timeout
	pt.TaskIsDone = false
	pt.Client = b.Client
	pt.URI = ""
	pt.Name = ""
	pt.Owner = ""
	pt.Timeout = 36
	pt.WaitTime = 10
	return pt
}

// get current power state
func (pt *PowerTask) GetCurrentPowerState() error {
	// Quick check to make sure we have a proper hardware blade
	if pt.Blade.URI.IsNil() {
		pt.State = P_UKNOWN
		return errors.New("Can't get power on blade without hardware")
	}

	// get the latest state based on current blade uri
	b, err := pt.Blade.Client.GetServerHardware(pt.Blade.URI)
	if err != nil {
		return err
	}
	log.Debugf("GetCurrentPowerState() blade -> %+v", b)
	// Set the current state of the blade as a constant
	if P_OFF.Equal(b.PowerState) {
		pt.State = P_OFF
	} else if P_ON.Equal(b.PowerState) {
		pt.State = P_ON
	} else {
		log.Warnf("Un-known power state detected %s, for %s.", b.PowerState, b.Name)
		pt.State = P_UKNOWN
	}
	// Reassign the current blade and state of that blade
	pt.Blade = b
	return nil
}

// PowerRequest
// { 'body' => { 'powerState' => state.capitalize, 'powerControl' => 'MomentaryPress' } })
type PowerRequest struct {
	PowerState   string `json:"powerState,omitempty"`
	PowerControl string `json:"powerControl,omitempty"`
}

// TODO: new parameter for submit power state to do P_RESET

// Submit desired power state
func (pt *PowerTask) SubmitPowerState(s PowerState) {
	if err := pt.GetCurrentPowerState(); err != nil {
		pt.TaskIsDone = true
		log.Errorf("Error getting current power state: %s", err)
		return
	}

	if s != pt.State {
		log.Infof("Powering %s server %s for %s.", s, pt.Blade.Name, pt.Blade.SerialNumber)
		var (
			body = PowerRequest{PowerState: s.String()}
			uri  = strings.Join([]string{pt.Blade.URI.String(),
				"/powerState"}, "")
		)
		if s.String() == "On" {
			body.PowerControl = P_MOMPRESS.String()
		} else {
			body.PowerControl = P_PRESSANDHOLD.String()
		}
		log.Debugf("REST : %s \n %+v\n", uri, body)
		log.Debugf("pt -> %+v", pt)
		data, err := pt.Blade.Client.RestAPICall(rest.PUT, uri, body)
		if err != nil {
			pt.TaskIsDone = true
			log.Errorf("Error with power state request: %s", err)
			return
		}

		log.Debugf("SubmitPowerState %s", data)
		if err := json.Unmarshal([]byte(data), &pt); err != nil {
			pt.TaskIsDone = true
			log.Errorf("Error with power state un-marshal: %s", err)
			return
		}
	} else {
		log.Infof("Desired Power State already set -> %s", pt.State)
		pt.TaskIsDone = true
	}

	return
}

// Submit desired power state and wait
// Most of our concurrency will happen in PowerExecutor
func (pt *PowerTask) PowerExecutor(s PowerState) error {
	currenttime := 0
	pt.State = P_UKNOWN
	pt.ResetTask()
	go pt.SubmitPowerState(s)
	for !pt.TaskIsDone && (currenttime < pt.Timeout) {
		if err := pt.GetCurrentTaskStatus(); err != nil {
			return err
		}
		if pt.URI != "" && T_COMPLETED.Equal(pt.TaskState) {
			pt.TaskIsDone = true
		}
		if pt.URI != "" {
			log.Debugf("Waiting to set power state %s for blade %s, %s", s, pt.Blade.Name)
			log.Infof("Working on power state,%d%%, %s.", pt.ComputedPercentComplete, pt.TaskStatus)
		} else {
			log.Info("Working on power state.")
		}

		// wait time before next check
		time.Sleep(time.Millisecond * (1000 * pt.WaitTime)) // wait 10sec before checking the status again
		currenttime++
	}
	if !(currenttime < pt.Timeout) {
		log.Warnf("Power %s state timed out for %s.", s, pt.Blade.Name)
	}
	log.Infof("Power Task Execution Completed")
	return nil
}
