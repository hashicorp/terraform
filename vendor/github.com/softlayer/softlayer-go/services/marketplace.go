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
type Marketplace_Partner struct {
	Session *session.Session
	Options sl.Options
}

// GetMarketplacePartnerService returns an instance of the Marketplace_Partner SoftLayer service
func GetMarketplacePartnerService(sess *session.Session) Marketplace_Partner {
	return Marketplace_Partner{Session: sess}
}

func (r Marketplace_Partner) Id(id int) Marketplace_Partner {
	r.Options.Id = &id
	return r
}

func (r Marketplace_Partner) Mask(mask string) Marketplace_Partner {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Marketplace_Partner) Filter(filter string) Marketplace_Partner {
	r.Options.Filter = filter
	return r
}

func (r Marketplace_Partner) Limit(limit int) Marketplace_Partner {
	r.Options.Limit = &limit
	return r
}

func (r Marketplace_Partner) Offset(offset int) Marketplace_Partner {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Marketplace_Partner) GetAllObjects() (resp []datatypes.Marketplace_Partner, err error) {
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Marketplace_Partner) GetAllPublishedPartners(searchTerm *string) (resp []datatypes.Marketplace_Partner, err error) {
	params := []interface{}{
		searchTerm,
	}
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getAllPublishedPartners", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Marketplace_Partner) GetAttachments() (resp []datatypes.Marketplace_Partner_Attachment, err error) {
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getAttachments", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Marketplace_Partner) GetFeaturedPartners(non *bool) (resp []datatypes.Marketplace_Partner, err error) {
	params := []interface{}{
		non,
	}
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getFeaturedPartners", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Marketplace_Partner) GetFile(name *string) (resp datatypes.Marketplace_Partner_File, err error) {
	params := []interface{}{
		name,
	}
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getFile", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Marketplace_Partner) GetLogoMedium() (resp datatypes.Marketplace_Partner_Attachment, err error) {
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getLogoMedium", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Marketplace_Partner) GetLogoMediumTemp() (resp datatypes.Marketplace_Partner_Attachment, err error) {
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getLogoMediumTemp", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Marketplace_Partner) GetLogoSmall() (resp datatypes.Marketplace_Partner_Attachment, err error) {
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getLogoSmall", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Marketplace_Partner) GetLogoSmallTemp() (resp datatypes.Marketplace_Partner_Attachment, err error) {
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getLogoSmallTemp", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Marketplace_Partner) GetObject() (resp datatypes.Marketplace_Partner, err error) {
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getObject", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Marketplace_Partner) GetPartnerByUrlIdentifier(urlIdentifier *string) (resp datatypes.Marketplace_Partner, err error) {
	params := []interface{}{
		urlIdentifier,
	}
	err = r.Session.DoRequest("SoftLayer_Marketplace_Partner", "getPartnerByUrlIdentifier", params, &r.Options, &resp)
	return
}
