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

package datatypes

// The SoftLayer_Tag data type is an optional type associated with hardware. The account ID that the tag is tied to, and the tag itself are stored in this data type. There is also a flag to denote whether the tag is internal or not.
type Tag struct {
	Entity

	// The account to which the tag is tied.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// Account the tag belongs to.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// Unique identifier for a tag.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Indicates whether a tag is internal.
	Internal *int `json:"internal,omitempty" xmlrpc:"internal,omitempty"`

	// Name of the tag. The characters permitted are A-Z, 0-9, whitespace, _ (underscore),
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of references that tie object to the tag.
	ReferenceCount *uint `json:"referenceCount,omitempty" xmlrpc:"referenceCount,omitempty"`

	// References that tie object to the tag.
	References []Tag_Reference `json:"references,omitempty" xmlrpc:"references,omitempty"`
}

// no documentation yet
type Tag_Reference struct {
	Entity

	// no documentation yet
	Customer *User_Customer `json:"customer,omitempty" xmlrpc:"customer,omitempty"`

	// no documentation yet
	EmpRecordId *int `json:"empRecordId,omitempty" xmlrpc:"empRecordId,omitempty"`

	// no documentation yet
	Employee *User_Employee `json:"employee,omitempty" xmlrpc:"employee,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ResourceTableId *int `json:"resourceTableId,omitempty" xmlrpc:"resourceTableId,omitempty"`

	// no documentation yet
	Tag *Tag `json:"tag,omitempty" xmlrpc:"tag,omitempty"`

	// no documentation yet
	TagId *int `json:"tagId,omitempty" xmlrpc:"tagId,omitempty"`

	// no documentation yet
	TagType *Tag_Type `json:"tagType,omitempty" xmlrpc:"tagType,omitempty"`

	// no documentation yet
	TagTypeId *int `json:"tagTypeId,omitempty" xmlrpc:"tagTypeId,omitempty"`

	// no documentation yet
	UsrRecordId *int `json:"usrRecordId,omitempty" xmlrpc:"usrRecordId,omitempty"`
}

// no documentation yet
type Tag_Reference_Hardware struct {
	Tag_Reference

	// no documentation yet
	Resource *Hardware `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Tag_Reference_Network_Application_Delivery_Controller struct {
	Tag_Reference

	// no documentation yet
	Resource *Network_Application_Delivery_Controller `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Tag_Reference_Network_Vlan struct {
	Tag_Reference

	// no documentation yet
	Resource *Network_Vlan `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Tag_Reference_Network_Vlan_Firewall struct {
	Tag_Reference

	// no documentation yet
	Resource *Network_Vlan_Firewall `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Tag_Reference_Resource_Group struct {
	Tag_Reference

	// no documentation yet
	Resource *Resource_Group `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Tag_Reference_Virtual_Guest struct {
	Tag_Reference

	// no documentation yet
	Resource *Virtual_Guest `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Tag_Reference_Virtual_Guest_Block_Device_Template_Group struct {
	Tag_Reference

	// no documentation yet
	Resource *Virtual_Guest_Block_Device_Template_Group `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Tag_Type struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`
}
