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
type Search struct {
	Session *session.Session
	Options sl.Options
}

// GetSearchService returns an instance of the Search SoftLayer service
func GetSearchService(sess *session.Session) Search {
	return Search{Session: sess}
}

func (r Search) Id(id int) Search {
	r.Options.Id = &id
	return r
}

func (r Search) Mask(mask string) Search {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Search) Filter(filter string) Search {
	r.Options.Filter = filter
	return r
}

func (r Search) Limit(limit int) Search {
	r.Options.Limit = &limit
	return r
}

func (r Search) Offset(offset int) Search {
	r.Options.Offset = &offset
	return r
}

// This method allows for searching for SoftLayer resources by simple terms and operators.  Fields that are used for searching will be available at sldn.softlayer.com. It returns a collection or array of <b>[[SoftLayer_Container_Search_Result (type)|SoftLayer_Container_Search_Result]]</b> objects that have search metadata for each result and the resulting resource found.
//
// The advancedSearch() method recognizes the special <b><code>_objectType:</code></b> quantifier in search strings.  See the documentation for the <b>[[SoftLayer_Search/search|search()]]</b> method on how to restrict searches using object types.
//
// The advancedSearch() method recognizes <b>[[SoftLayer_Container_Search_ObjectType_Property (type)|object properties]]</b>, which can also be used to limit searches.  Example:
//
// <code>_objectType:Type_1 propertyA:</code><i><code>value</code></i>
//
// A search string can specify multiple properties, separated with spaces. Example:
//
// <code>_objectType:Type_1 propertyA:</code><i><code>value</code></i> <code>propertyB:</code><i><code>value</code></i>
//
// A collection of available object types and their properties can be retrieved by calling the <b>[[SoftLayer_Search/getObjectTypes|getObjectTypes()]]</b> method.
func (r Search) AdvancedSearch(searchString *string) (resp []datatypes.Container_Search_Result, err error) {
	params := []interface{}{
		searchString,
	}
	err = r.Session.DoRequest("SoftLayer_Search", "advancedSearch", params, &r.Options, &resp)
	return
}

// This method returns a collection of <b>[[SoftLayer_Container_Search_ObjectType (type)|SoftLayer_Container_Search_ObjectType]]</b> containers that specify which indexed object types and properties are exposed for the current user.  These object types can be used to discover searchable data and to create or validate object index search strings.
//
// <p> Refer to the <b>[[SoftLayer_Search/search|search()]]</b> and <b>[[SoftLayer_Search/advancedSearch|advancedSearch()]]</b> methods for information on using object types and properties in search strings.
func (r Search) GetObjectTypes() (resp []datatypes.Container_Search_ObjectType, err error) {
	err = r.Session.DoRequest("SoftLayer_Search", "getObjectTypes", nil, &r.Options, &resp)
	return
}

// This method allows for searching for SoftLayer resources by simple phrase. It returns a collection or array of <b>[[SoftLayer_Container_Search_Result (type)|SoftLayer_Container_Search_Result]]</b> objects that have search metadata for each result and the resulting resource found.
//
// This method recognizes the special <b><code>_objectType:</code></b> quantifier in search strings.  This quantifier can be used to restrict a search to specific object types.  Example usage:
//
// <code>_objectType:Type_1 </code><i><code>(other search terms...)</code></i>
//
// A search string can specify multiple object types, separated by commas (no spaces are permitted between the type names).  Example:
//
// <code>_objectType:Type_1,Type_2,Type_3 </code><i><code>(other search terms...)</code></i>
//
// If the list of object types is prefixed with a hyphen or minus sign (-), then the specified types are excluded from the search.  Example:
//
// <code>_objectType:-Type_4,Type_5 </code><i><code>(other search terms...)</code></i>
//
// A collection of available object types can be retrieved by calling the <b>[[SoftLayer_Search/getObjectTypes|getObjectTypes()]]</b> method.
func (r Search) Search(searchString *string) (resp []datatypes.Container_Search_Result, err error) {
	params := []interface{}{
		searchString,
	}
	err = r.Session.DoRequest("SoftLayer_Search", "search", params, &r.Options, &resp)
	return
}
