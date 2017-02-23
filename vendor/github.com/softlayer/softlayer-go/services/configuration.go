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

// Supported hardware raid modes
type Configuration_Storage_Group_Array_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationStorageGroupArrayTypeService returns an instance of the Configuration_Storage_Group_Array_Type SoftLayer service
func GetConfigurationStorageGroupArrayTypeService(sess *session.Session) Configuration_Storage_Group_Array_Type {
	return Configuration_Storage_Group_Array_Type{Session: sess}
}

func (r Configuration_Storage_Group_Array_Type) Id(id int) Configuration_Storage_Group_Array_Type {
	r.Options.Id = &id
	return r
}

func (r Configuration_Storage_Group_Array_Type) Mask(mask string) Configuration_Storage_Group_Array_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Storage_Group_Array_Type) Filter(filter string) Configuration_Storage_Group_Array_Type {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Storage_Group_Array_Type) Limit(limit int) Configuration_Storage_Group_Array_Type {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Storage_Group_Array_Type) Offset(offset int) Configuration_Storage_Group_Array_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Configuration_Storage_Group_Array_Type) GetAllObjects() (resp []datatypes.Configuration_Storage_Group_Array_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Storage_Group_Array_Type", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Storage_Group_Array_Type) GetHardwareComponentModels() (resp []datatypes.Hardware_Component_Model, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Storage_Group_Array_Type", "getHardwareComponentModels", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Configuration_Storage_Group_Array_Type) GetObject() (resp datatypes.Configuration_Storage_Group_Array_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Storage_Group_Array_Type", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Configuration_Template data type contains general information of an arbitrary resource.
type Configuration_Template struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateService returns an instance of the Configuration_Template SoftLayer service
func GetConfigurationTemplateService(sess *session.Session) Configuration_Template {
	return Configuration_Template{Session: sess}
}

func (r Configuration_Template) Id(id int) Configuration_Template {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template) Mask(mask string) Configuration_Template {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template) Filter(filter string) Configuration_Template {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template) Limit(limit int) Configuration_Template {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template) Offset(offset int) Configuration_Template {
	r.Options.Offset = &offset
	return r
}

// Copy a configuration template and returns a newly created template copy
func (r Configuration_Template) CopyTemplate(templateObject *datatypes.Configuration_Template) (resp datatypes.Configuration_Template, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "copyTemplate", params, &r.Options, &resp)
	return
}

// Deletes a customer configuration template.
func (r Configuration_Template) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "deleteObject", nil, &r.Options, &resp)
	return
}

// Edit the object by passing in a modified instance of the object. Use this method to modify configuration template name or description.
func (r Configuration_Template) EditObject(templateObject *datatypes.Configuration_Template) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieves all available configuration templates
func (r Configuration_Template) GetAllObjects() (resp []datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetConfigurationSections() (resp []datatypes.Configuration_Template_Section, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getConfigurationSections", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetConfigurationTemplateReference() (resp []datatypes.Monitoring_Agent_Configuration_Template_Group_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getConfigurationTemplateReference", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetDefaultValues() (resp []datatypes.Configuration_Template_Section_Definition_Value, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getDefaultValues", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetDefinitions() (resp []datatypes.Configuration_Template_Section_Definition, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getDefinitions", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetItem() (resp datatypes.Product_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getItem", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetLinkedSectionReferences() (resp datatypes.Configuration_Template_Section_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getLinkedSectionReferences", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Configuration_Template) GetObject() (resp datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetParent() (resp datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getParent", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template) GetUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "getUser", nil, &r.Options, &resp)
	return
}

// Updates default configuration values.
func (r Configuration_Template) UpdateDefaultValues(configurationValues []datatypes.Configuration_Template_Section_Definition_Value) (resp bool, err error) {
	params := []interface{}{
		configurationValues,
	}
	err = r.Session.DoRequest("SoftLayer_Configuration_Template", "updateDefaultValues", params, &r.Options, &resp)
	return
}

// The SoftLayer_Configuration_Template_Section data type contains information of a configuration section.
//
// Configuration can contain sub-sections.
type Configuration_Template_Section struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateSectionService returns an instance of the Configuration_Template_Section SoftLayer service
func GetConfigurationTemplateSectionService(sess *session.Session) Configuration_Template_Section {
	return Configuration_Template_Section{Session: sess}
}

func (r Configuration_Template_Section) Id(id int) Configuration_Template_Section {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Section) Mask(mask string) Configuration_Template_Section {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Section) Filter(filter string) Configuration_Template_Section {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Section) Limit(limit int) Configuration_Template_Section {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Section) Offset(offset int) Configuration_Template_Section {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Configuration_Template_Section) GetDefinitions() (resp []datatypes.Configuration_Template_Section_Definition, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getDefinitions", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section) GetDisallowedDeletionFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getDisallowedDeletionFlag", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section) GetLinkedTemplate() (resp datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getLinkedTemplate", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section) GetLinkedTemplateReference() (resp datatypes.Configuration_Template_Section_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getLinkedTemplateReference", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Configuration_Template_Section) GetObject() (resp datatypes.Configuration_Template_Section, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section) GetProfiles() (resp []datatypes.Configuration_Template_Section_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getProfiles", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section) GetSectionType() (resp datatypes.Configuration_Template_Section_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getSectionType", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section) GetSectionTypeName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getSectionTypeName", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section) GetSubSections() (resp []datatypes.Configuration_Template_Section, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getSubSections", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section) GetTemplate() (resp datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "getTemplate", nil, &r.Options, &resp)
	return
}

// Returns true if the object has sub-sections
func (r Configuration_Template_Section) HasSubSections() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section", "hasSubSections", nil, &r.Options, &resp)
	return
}

// Configuration definition gives you details of the value that you're setting.
//
// Some monitoring agents requires values unique to your system. If value type is defined as "Resource Specific Values", you will have to make an additional API call to retrieve your system specific values.
//
// See [[SoftLayer_Monitoring_Agent::getAvailableConfigurationValues|Monitoring Agent]] service to retrieve your system specific values.
type Configuration_Template_Section_Definition struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateSectionDefinitionService returns an instance of the Configuration_Template_Section_Definition SoftLayer service
func GetConfigurationTemplateSectionDefinitionService(sess *session.Session) Configuration_Template_Section_Definition {
	return Configuration_Template_Section_Definition{Session: sess}
}

func (r Configuration_Template_Section_Definition) Id(id int) Configuration_Template_Section_Definition {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Section_Definition) Mask(mask string) Configuration_Template_Section_Definition {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Section_Definition) Filter(filter string) Configuration_Template_Section_Definition {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Section_Definition) Limit(limit int) Configuration_Template_Section_Definition {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Section_Definition) Offset(offset int) Configuration_Template_Section_Definition {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Configuration_Template_Section_Definition) GetAttributes() (resp []datatypes.Configuration_Template_Section_Definition_Attribute, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition", "getAttributes", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Definition) GetDefaultValue() (resp datatypes.Configuration_Template_Section_Definition_Value, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition", "getDefaultValue", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Definition) GetGroup() (resp datatypes.Configuration_Template_Section_Definition_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition", "getGroup", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Definition) GetMonitoringDataFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition", "getMonitoringDataFlag", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Configuration_Template_Section_Definition) GetObject() (resp datatypes.Configuration_Template_Section_Definition, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Definition) GetSection() (resp datatypes.Configuration_Template_Section, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition", "getSection", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Definition) GetValueType() (resp datatypes.Configuration_Template_Section_Definition_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition", "getValueType", nil, &r.Options, &resp)
	return
}

// Configuration definition group gives you details of the definition and allows extra functionality.
//
//
type Configuration_Template_Section_Definition_Group struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateSectionDefinitionGroupService returns an instance of the Configuration_Template_Section_Definition_Group SoftLayer service
func GetConfigurationTemplateSectionDefinitionGroupService(sess *session.Session) Configuration_Template_Section_Definition_Group {
	return Configuration_Template_Section_Definition_Group{Session: sess}
}

func (r Configuration_Template_Section_Definition_Group) Id(id int) Configuration_Template_Section_Definition_Group {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Section_Definition_Group) Mask(mask string) Configuration_Template_Section_Definition_Group {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Section_Definition_Group) Filter(filter string) Configuration_Template_Section_Definition_Group {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Section_Definition_Group) Limit(limit int) Configuration_Template_Section_Definition_Group {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Section_Definition_Group) Offset(offset int) Configuration_Template_Section_Definition_Group {
	r.Options.Offset = &offset
	return r
}

// Get all configuration definition group objects.
//
// ''getAllGroups'' returns an array of SoftLayer_Configuration_Template_Section_Definition_Group objects upon success.
func (r Configuration_Template_Section_Definition_Group) GetAllGroups() (resp []datatypes.Configuration_Template_Section_Definition_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition_Group", "getAllGroups", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Configuration_Template_Section_Definition_Group) GetObject() (resp datatypes.Configuration_Template_Section_Definition_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition_Group", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Definition_Group) GetParent() (resp datatypes.Configuration_Template_Section_Definition_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition_Group", "getParent", nil, &r.Options, &resp)
	return
}

// SoftLayer_Configuration_Template_Section_Definition_Type further defines the value of a configuration definition.
type Configuration_Template_Section_Definition_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateSectionDefinitionTypeService returns an instance of the Configuration_Template_Section_Definition_Type SoftLayer service
func GetConfigurationTemplateSectionDefinitionTypeService(sess *session.Session) Configuration_Template_Section_Definition_Type {
	return Configuration_Template_Section_Definition_Type{Session: sess}
}

func (r Configuration_Template_Section_Definition_Type) Id(id int) Configuration_Template_Section_Definition_Type {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Section_Definition_Type) Mask(mask string) Configuration_Template_Section_Definition_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Section_Definition_Type) Filter(filter string) Configuration_Template_Section_Definition_Type {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Section_Definition_Type) Limit(limit int) Configuration_Template_Section_Definition_Type {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Section_Definition_Type) Offset(offset int) Configuration_Template_Section_Definition_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Configuration_Template_Section_Definition_Type) GetObject() (resp datatypes.Configuration_Template_Section_Definition_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition_Type", "getObject", nil, &r.Options, &resp)
	return
}

// SoftLayer_Configuration_Section_Value is used to set the value for a configuration definition
type Configuration_Template_Section_Definition_Value struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateSectionDefinitionValueService returns an instance of the Configuration_Template_Section_Definition_Value SoftLayer service
func GetConfigurationTemplateSectionDefinitionValueService(sess *session.Session) Configuration_Template_Section_Definition_Value {
	return Configuration_Template_Section_Definition_Value{Session: sess}
}

func (r Configuration_Template_Section_Definition_Value) Id(id int) Configuration_Template_Section_Definition_Value {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Section_Definition_Value) Mask(mask string) Configuration_Template_Section_Definition_Value {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Section_Definition_Value) Filter(filter string) Configuration_Template_Section_Definition_Value {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Section_Definition_Value) Limit(limit int) Configuration_Template_Section_Definition_Value {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Section_Definition_Value) Offset(offset int) Configuration_Template_Section_Definition_Value {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Configuration_Template_Section_Definition_Value) GetDefinition() (resp datatypes.Configuration_Template_Section_Definition, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition_Value", "getDefinition", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Configuration_Template_Section_Definition_Value) GetObject() (resp datatypes.Configuration_Template_Section_Definition_Value, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition_Value", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Definition_Value) GetTemplate() (resp datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Definition_Value", "getTemplate", nil, &r.Options, &resp)
	return
}

// Some configuration templates let you create a unique configuration profiles.
//
// For example, you can create multiple configuration profiles to monitor multiple hard drives with "CPU/Memory/Disk Monitoring Agent". SoftLayer_Configuration_Template_Section_Profile help you keep track of custom configuration profiles.
type Configuration_Template_Section_Profile struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateSectionProfileService returns an instance of the Configuration_Template_Section_Profile SoftLayer service
func GetConfigurationTemplateSectionProfileService(sess *session.Session) Configuration_Template_Section_Profile {
	return Configuration_Template_Section_Profile{Session: sess}
}

func (r Configuration_Template_Section_Profile) Id(id int) Configuration_Template_Section_Profile {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Section_Profile) Mask(mask string) Configuration_Template_Section_Profile {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Section_Profile) Filter(filter string) Configuration_Template_Section_Profile {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Section_Profile) Limit(limit int) Configuration_Template_Section_Profile {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Section_Profile) Offset(offset int) Configuration_Template_Section_Profile {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Configuration_Template_Section_Profile) GetConfigurationSection() (resp datatypes.Configuration_Template_Section, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Profile", "getConfigurationSection", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Profile) GetMonitoringAgent() (resp datatypes.Monitoring_Agent, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Profile", "getMonitoringAgent", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Configuration_Template_Section_Profile) GetObject() (resp datatypes.Configuration_Template_Section_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Profile", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Configuration_Template_Section_Reference data type contains information of a configuration section and its associated configuration template.
type Configuration_Template_Section_Reference struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateSectionReferenceService returns an instance of the Configuration_Template_Section_Reference SoftLayer service
func GetConfigurationTemplateSectionReferenceService(sess *session.Session) Configuration_Template_Section_Reference {
	return Configuration_Template_Section_Reference{Session: sess}
}

func (r Configuration_Template_Section_Reference) Id(id int) Configuration_Template_Section_Reference {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Section_Reference) Mask(mask string) Configuration_Template_Section_Reference {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Section_Reference) Filter(filter string) Configuration_Template_Section_Reference {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Section_Reference) Limit(limit int) Configuration_Template_Section_Reference {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Section_Reference) Offset(offset int) Configuration_Template_Section_Reference {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Configuration_Template_Section_Reference) GetObject() (resp datatypes.Configuration_Template_Section_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Reference", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Reference) GetSection() (resp datatypes.Configuration_Template_Section, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Reference", "getSection", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Configuration_Template_Section_Reference) GetTemplate() (resp datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Reference", "getTemplate", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Configuration_Template_Section_Type data type contains information of a configuration section type.
//
// Configuration can contain sub-sections.
type Configuration_Template_Section_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateSectionTypeService returns an instance of the Configuration_Template_Section_Type SoftLayer service
func GetConfigurationTemplateSectionTypeService(sess *session.Session) Configuration_Template_Section_Type {
	return Configuration_Template_Section_Type{Session: sess}
}

func (r Configuration_Template_Section_Type) Id(id int) Configuration_Template_Section_Type {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Section_Type) Mask(mask string) Configuration_Template_Section_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Section_Type) Filter(filter string) Configuration_Template_Section_Type {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Section_Type) Limit(limit int) Configuration_Template_Section_Type {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Section_Type) Offset(offset int) Configuration_Template_Section_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Configuration_Template_Section_Type) GetObject() (resp datatypes.Configuration_Template_Section_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Section_Type", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Configuration_Template_Type data type contains configuration template type information.
type Configuration_Template_Type struct {
	Session *session.Session
	Options sl.Options
}

// GetConfigurationTemplateTypeService returns an instance of the Configuration_Template_Type SoftLayer service
func GetConfigurationTemplateTypeService(sess *session.Session) Configuration_Template_Type {
	return Configuration_Template_Type{Session: sess}
}

func (r Configuration_Template_Type) Id(id int) Configuration_Template_Type {
	r.Options.Id = &id
	return r
}

func (r Configuration_Template_Type) Mask(mask string) Configuration_Template_Type {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Configuration_Template_Type) Filter(filter string) Configuration_Template_Type {
	r.Options.Filter = filter
	return r
}

func (r Configuration_Template_Type) Limit(limit int) Configuration_Template_Type {
	r.Options.Limit = &limit
	return r
}

func (r Configuration_Template_Type) Offset(offset int) Configuration_Template_Type {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Configuration_Template_Type) GetObject() (resp datatypes.Configuration_Template_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Configuration_Template_Type", "getObject", nil, &r.Options, &resp)
	return
}
