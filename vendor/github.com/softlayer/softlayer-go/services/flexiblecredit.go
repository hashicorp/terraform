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
type FlexibleCredit_Program struct {
	Session *session.Session
	Options sl.Options
}

// GetFlexibleCreditProgramService returns an instance of the FlexibleCredit_Program SoftLayer service
func GetFlexibleCreditProgramService(sess *session.Session) FlexibleCredit_Program {
	return FlexibleCredit_Program{Session: sess}
}

func (r FlexibleCredit_Program) Id(id int) FlexibleCredit_Program {
	r.Options.Id = &id
	return r
}

func (r FlexibleCredit_Program) Mask(mask string) FlexibleCredit_Program {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r FlexibleCredit_Program) Filter(filter string) FlexibleCredit_Program {
	r.Options.Filter = filter
	return r
}

func (r FlexibleCredit_Program) Limit(limit int) FlexibleCredit_Program {
	r.Options.Limit = &limit
	return r
}

func (r FlexibleCredit_Program) Offset(offset int) FlexibleCredit_Program {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r FlexibleCredit_Program) GetAffiliatesAvailableForSelfEnrollmentByVerificationType(verificationTypeKeyName *string) (resp []datatypes.FlexibleCredit_Affiliate, err error) {
	params := []interface{}{
		verificationTypeKeyName,
	}
	err = r.Session.DoRequest("SoftLayer_FlexibleCredit_Program", "getAffiliatesAvailableForSelfEnrollmentByVerificationType", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r FlexibleCredit_Program) GetCompanyTypes() (resp []datatypes.FlexibleCredit_Company_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_FlexibleCredit_Program", "getCompanyTypes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r FlexibleCredit_Program) GetObject() (resp datatypes.FlexibleCredit_Program, err error) {
	err = r.Session.DoRequest("SoftLayer_FlexibleCredit_Program", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r FlexibleCredit_Program) SelfEnrollNewAccount(accountTemplate *datatypes.Account) (resp datatypes.Account, err error) {
	params := []interface{}{
		accountTemplate,
	}
	err = r.Session.DoRequest("SoftLayer_FlexibleCredit_Program", "selfEnrollNewAccount", params, &r.Options, &resp)
	return
}
