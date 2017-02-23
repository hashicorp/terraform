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

// The SoftLayer_Tag data type is an optional type associated with hardware. The account ID that the tag is tied to, and the tag itself are stored in this data type. There is also a flag to denote whether the tag is internal or not.
type Tag struct {
	Session *session.Session
	Options sl.Options
}

// GetTagService returns an instance of the Tag SoftLayer service
func GetTagService(sess *session.Session) Tag {
	return Tag{Session: sess}
}

func (r Tag) Id(id int) Tag {
	r.Options.Id = &id
	return r
}

func (r Tag) Mask(mask string) Tag {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Tag) Filter(filter string) Tag {
	r.Options.Filter = filter
	return r
}

func (r Tag) Limit(limit int) Tag {
	r.Options.Limit = &limit
	return r
}

func (r Tag) Offset(offset int) Tag {
	r.Options.Offset = &offset
	return r
}

// This function is responsible for setting the Tags values. The internal flag is set to 0 if the user is a customer, and 1 otherwise. AccountId is set to the account bound to the user, and the tags name is set to the clean version of the tag inputted by the user.
func (r Tag) AutoComplete(tag *string) (resp []datatypes.Tag, err error) {
	params := []interface{}{
		tag,
	}
	err = r.Session.DoRequest("SoftLayer_Tag", "autoComplete", params, &r.Options, &resp)
	return
}

// Retrieve The account to which the tag is tied.
func (r Tag) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Tag", "getAccount", nil, &r.Options, &resp)
	return
}

// Returns all tags of a given object type.
func (r Tag) GetAllTagTypes() (resp []datatypes.Tag_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Tag", "getAllTagTypes", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Tag) GetObject() (resp datatypes.Tag, err error) {
	err = r.Session.DoRequest("SoftLayer_Tag", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve References that tie object to the tag.
func (r Tag) GetReferences() (resp []datatypes.Tag_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Tag", "getReferences", nil, &r.Options, &resp)
	return
}

// Returns the Tag object with a given name. The user types in the tag name and this method returns the tag with that name.
func (r Tag) GetTagByTagName(tagList *string) (resp []datatypes.Tag, err error) {
	params := []interface{}{
		tagList,
	}
	err = r.Session.DoRequest("SoftLayer_Tag", "getTagByTagName", params, &r.Options, &resp)
	return
}

// Tag an object by passing in one or more tags separated by a comma.  Tag references are cleared out every time this method is called. If your object is already tagged you will need to pass the current tags along with any new ones.  To remove all tag references pass an empty string. To remove one or more tags omit them from the tag list.  The characters permitted are A-Z, 0-9, whitespace, _ (underscore), - (hypen), . (period), and : (colon). All other characters will be stripped away.
func (r Tag) SetTags(tags *string, keyName *string, resourceTableId *int) (resp bool, err error) {
	params := []interface{}{
		tags,
		keyName,
		resourceTableId,
	}
	err = r.Session.DoRequest("SoftLayer_Tag", "setTags", params, &r.Options, &resp)
	return
}
