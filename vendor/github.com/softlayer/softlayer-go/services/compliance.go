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

// no documentation yet
type Compliance_Report_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetComplianceReportTypeService returns an instance of the Compliance_Report_Type SoftLayer service
func GetComplianceReportTypeService(sess *session.Session) Compliance_Report_Type {
	return Compliance_Report_Type{Session: sess}
}

func (r Compliance_Report_Type) Id(id int) Compliance_Report_Type {
	r.Options.Id = &id
	return r
}

func (r Compliance_Report_Type) Mask(mask string) Compliance_Report_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Compliance_Report_Type) Filter(filter string) Compliance_Report_Type {
	r.Options.Filter = filter
	return r
}

func (r Compliance_Report_Type) Limit(limit int) Compliance_Report_Type {
	r.Options.Limit = &limit
	return r
}

func (r Compliance_Report_Type) Offset(offset int) Compliance_Report_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Compliance_Report_Type) GetAllObjects() (resp []datatypes.Compliance_Report_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Compliance_Report_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Compliance_Report_Type) GetObject() (resp datatypes.Compliance_Report_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Compliance_Report_Type", "getObject", nil, &r.Options, &resp)
	return
}
