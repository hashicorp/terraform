/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * AUTOMATICALLY GENERATED CODE - DO NOT MODIFY
 */

package services

import (
	"fmt"
	"strings"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
)

// The SoftLayer_Event_Log data type contains an event detail occurred upon various SoftLayer resources.
type Event_Log struct {
	Session *session.Session
	Options sl.Options
}

// GetEventLogService returns an instance of the Event_Log SoftLayer service
func GetEventLogService(sess *session.Session) Event_Log {
	return Event_Log{Session: sess}
}

func (r Event_Log) Id(id int) Event_Log {
	r.Options.Id = &id
	return r
}

func (r Event_Log) Mask(mask string) Event_Log {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Event_Log) Filter(filter string) Event_Log {
	r.Options.Filter = filter
	return r
}

func (r Event_Log) Limit(limit int) Event_Log {
	r.Options.Limit = &limit
	return r
}

func (r Event_Log) Offset(offset int) Event_Log {
	r.Options.Offset = &offset
	return r
}

// This all indexed event names.
func (r Event_Log) GetAllEventNames(objectName *string) (resp []string, err error) {
	params := []interface{}{
		objectName,
	}
	err = r.Session.DoRequest("SoftLayer_Event_Log", "getAllEventNames", params, &r.Options, &resp)
	return
}

// This all indexed event object names.
func (r Event_Log) GetAllEventObjectNames() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Event_Log", "getAllEventObjectNames", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Event_Log) GetAllObjects() (resp []datatypes.Event_Log, err error) {
	err = r.Session.DoRequest("SoftLayer_Event_Log", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Event_Log) GetAllUserTypes() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Event_Log", "getAllUserTypes", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Event_Log) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Event_Log", "getUser", nil, &r.Options, &resp)
	return
}
