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

// A monitoring agent object contains information describing the agent.
type Monitoring_Agent struct {
	Session *session.Session
	Options sl.Options
}

// GetMonitoringAgentService returns an instance of the Monitoring_Agent SoftLayer service
func GetMonitoringAgentService(sess *session.Session) Monitoring_Agent {
	return Monitoring_Agent{Session: sess}
}

func (r Monitoring_Agent) Id(id int) Monitoring_Agent {
	r.Options.Id = &id
	return r
}

func (r Monitoring_Agent) Mask(mask string) Monitoring_Agent {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Monitoring_Agent) Filter(filter string) Monitoring_Agent {
	r.Options.Filter = filter
	return r
}

func (r Monitoring_Agent) Limit(limit int) Monitoring_Agent {
	r.Options.Limit = &limit
	return r
}

func (r Monitoring_Agent) Offset(offset int) Monitoring_Agent {
	r.Options.Offset = &offset
	return r
}

// This method activates a SoftLayer_Monitoring_Agent.
func (r Monitoring_Agent) Activate() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "activate", nil, &r.Options, &resp)
	return
}

// This method is used to apply changes to a monitoring agent's configuration for SoftLayer_Configuration_Template_Section with the property sectionType that has a keyName of 'TEMPLATE_SECTION'. Configuration values that are passed in can be new or updated objects but must have a definitionId and profileId defined for both. Existing SoftLayer_Monitoring_Agent_Configuration_Value values can be retrieved as a property of the SoftLayer_Configuration_Template_Section_Definition's from the monitoring agent's configurationTemplate property. New values will follow the structure of SoftLayer_Monitoring_Agent_Configuration_Value. It returns a SoftLayer_Provisioning_Version1_Transaction object to track the progress of the update being applied. Some configuration sections act as a template which helps to create additional monitoring configurations. For instance, Core Resource monitoring agent lets you create monitoring configurations for different disk volumes or disk path.
func (r Monitoring_Agent) AddConfigurationProfile(configurationValues []datatypes.Monitoring_Agent_Configuration_Value) (resp datatypes.Provisioning_Version1_Transaction, err error) {
	params := []interface{}{
		configurationValues,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "addConfigurationProfile", params, &r.Options, &resp)
	return
}

// This method creates a transaction used to apply changes to a monitoring agent's configuration for an array of SoftLayer_Configuration_Template_Section that have the property sectionType with a name of 'Fixed section'. Configuration values that are passed in can be new or updated objects but must have a configurationDefinitionId defined for both. Existing SoftLayer_Monitoring_Agent_Configuration_Value values can be retrieved as a property of the SoftLayer_Configuration_Template_Section_Definition from the monitoring agent's configurationTemplate property. New values will follow the structure of SoftLayer_Monitoring_Agent_Configuration_Value. This method returns a SoftLayer_Provisioning_Version1_Transaction object to track the progress of the update being applied.
func (r Monitoring_Agent) ApplyConfigurationValues(configurationValues []datatypes.Monitoring_Agent_Configuration_Value) (resp datatypes.Provisioning_Version1_Transaction, err error) {
	params := []interface{}{
		configurationValues,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "applyConfigurationValues", params, &r.Options, &resp)
	return
}

// This method will deactivate the monitoring agent, preventing it from generating any further alarms.
func (r Monitoring_Agent) Deactivate() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "deactivate", nil, &r.Options, &resp)
	return
}

// This method will remove a SoftLayer_Configuration_Template_Section_Profile from a SoftLayer_Configuration_Template_Section by passing in the sectionId of the profile object and identifier of the profile. This will execute the action immediately on the server and the SoftLayer_Configuration_Template_Section returning a boolean true if successful.
func (r Monitoring_Agent) DeleteConfigurationProfile(sectionId *int, profileId *int) (resp bool, err error) {
	params := []interface{}{
		sectionId,
		profileId,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "deleteConfigurationProfile", params, &r.Options, &resp)
	return
}

// Initialize a monitoring agent and deploy it with the SoftLayer_Configuration_Template with the same identifier as the $configurationTemplateId parameter. If the configuration template ID is not provided, the current configuration template will be used. When executing this method, the existing configuration values will be lost. If no configuration template identifier is provided, the current configuration template will be used. '''Warning''' Reporting data may be lost as a result of executing this method.
func (r Monitoring_Agent) DeployMonitoringAgent(configurationTemplateId *int) (resp datatypes.Provisioning_Version1_Transaction, err error) {
	params := []interface{}{
		configurationTemplateId,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "deployMonitoringAgent", params, &r.Options, &resp)
	return
}

// This method retrieves an array of SoftLayer_Notification_User_Subscriber objects belonging to the SoftLayer_Monitoring_Agent which are able to receive alarm notifications.
func (r Monitoring_Agent) GetActiveAlarmSubscribers() (resp []datatypes.Notification_User_Subscriber, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getActiveAlarmSubscribers", nil, &r.Options, &resp)
	return
}

// Retrieve The current status of the corresponding agent
func (r Monitoring_Agent) GetAgentStatus() (resp datatypes.Monitoring_Agent_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getAgentStatus", nil, &r.Options, &resp)
	return
}

// This method returns an array of available SoftLayer_Configuration_Template objects for this monitoring agent.
func (r Monitoring_Agent) GetAvailableConfigurationTemplates() (resp []datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getAvailableConfigurationTemplates", nil, &r.Options, &resp)
	return
}

// Returns an array of available configuration values that are specific to a server or a Virtual that this monitoring agent is running on. For example, invoking this method against "Network Traffic Monitoring Agent" will return all available network adapters on your system.
func (r Monitoring_Agent) GetAvailableConfigurationValues(configurationDefinitionId *int, configValues []datatypes.Monitoring_Agent_Configuration_Value) (resp []datatypes.Monitoring_Agent_Configuration_Value, err error) {
	params := []interface{}{
		configurationDefinitionId,
		configValues,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getAvailableConfigurationValues", params, &r.Options, &resp)
	return
}

// Retrieve All custom configuration profiles associated with the corresponding agent
func (r Monitoring_Agent) GetConfigurationProfiles() (resp []datatypes.Configuration_Template_Section_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getConfigurationProfiles", nil, &r.Options, &resp)
	return
}

// Retrieve A template of an agent's current configuration which contains information about the structure of the configuration values.
func (r Monitoring_Agent) GetConfigurationTemplate() (resp datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getConfigurationTemplate", nil, &r.Options, &resp)
	return
}

// Retrieve The values associated with the corresponding Agent configuration.
func (r Monitoring_Agent) GetConfigurationValues() (resp []datatypes.Monitoring_Agent_Configuration_Value, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getConfigurationValues", nil, &r.Options, &resp)
	return
}

// This method returns an array of SoftLayer_User_Customer objects, representing those who are allowed to be used as alarm subscribers.
func (r Monitoring_Agent) GetEligibleAlarmSubscibers() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getEligibleAlarmSubscibers", nil, &r.Options, &resp)
	return
}

// This method returns a SoftLayer_Container_Bandwidth_GraphOutputs object containing a base64 PNG string graph of the provided configuration values for the given begin and end dates.
func (r Monitoring_Agent) GetGraph(configurationValues []datatypes.Monitoring_Agent_Configuration_Value, beginDate *datatypes.Time, endDate *datatypes.Time) (resp datatypes.Container_Monitoring_Graph_Outputs, err error) {
	params := []interface{}{
		configurationValues,
		beginDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getGraph", params, &r.Options, &resp)
	return
}

// This method returns the metric data for each of the configuration values provided during the given time range.
func (r Monitoring_Agent) GetGraphData(metricDataTypes []datatypes.Container_Metric_Data_Type, startDate *datatypes.Time, endDate *datatypes.Time) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		metricDataTypes,
		startDate,
		endDate,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getGraphData", params, &r.Options, &resp)
	return
}

// Retrieve SoftLayer hardware related to the agent.
func (r Monitoring_Agent) GetHardware() (resp datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getHardware", nil, &r.Options, &resp)
	return
}

// This method retrieves a monitoring agent whose identifier corresponds to the value provided in the initialization parameter passed to the SoftLayer_Monitoring_Agent service.
func (r Monitoring_Agent) GetObject() (resp datatypes.Monitoring_Agent, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve Contains general information relating to a single SoftLayer product.
func (r Monitoring_Agent) GetProductItem() (resp datatypes.Product_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getProductItem", nil, &r.Options, &resp)
	return
}

// Retrieve A description for a specific installation of a Software Component
func (r Monitoring_Agent) GetSoftwareDescription() (resp datatypes.Software_Description, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getSoftwareDescription", nil, &r.Options, &resp)
	return
}

// Retrieve Monitoring agent status name.
func (r Monitoring_Agent) GetStatusName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getStatusName", nil, &r.Options, &resp)
	return
}

// Retrieve Softlayer_Virtual_Guest object related to the monitoring agent, which this virtual guest object and hardware is on the server of the running agent.
func (r Monitoring_Agent) GetVirtualGuest() (resp datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "getVirtualGuest", nil, &r.Options, &resp)
	return
}

// Use of this method will allow removing active subscribers from the monitoring agent. The agent subscribers can be managed within the portal from the "Alarm Subscribers" tab of the monitoring agent configuration.
func (r Monitoring_Agent) RemoveActiveAlarmSubscriber(userRecordId *int) (resp bool, err error) {
	params := []interface{}{
		userRecordId,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "removeActiveAlarmSubscriber", params, &r.Options, &resp)
	return
}

// Use of this method will allow removing all subscribers from the monitoring agent. The agent subscribers can be managed within the portal from the "Alarm Subscribers" tab of the monitoring agent configuration.
func (r Monitoring_Agent) RemoveAllAlarmSubscribers() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "removeAllAlarmSubscribers", nil, &r.Options, &resp)
	return
}

// This method restarts a monitoring agent and sets the agent's status to 'ACTIVE'.
func (r Monitoring_Agent) RestartMonitoringAgent() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "restartMonitoringAgent", nil, &r.Options, &resp)
	return
}

// This method assigns a user to receive the alerts generated by this SoftLayer_Monitoring_Agent.
func (r Monitoring_Agent) SetActiveAlarmSubscriber(userRecordId *int) (resp bool, err error) {
	params := []interface{}{
		userRecordId,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent", "setActiveAlarmSubscriber", params, &r.Options, &resp)
	return
}

// The SoftLayer_Monitoring_Agent_Configuration_Template_Group class is consisted of configuration templates for agents in a monitoring package.
type Monitoring_Agent_Configuration_Template_Group struct {
	Session *session.Session
	Options sl.Options
}

// GetMonitoringAgentConfigurationTemplateGroupService returns an instance of the Monitoring_Agent_Configuration_Template_Group SoftLayer service
func GetMonitoringAgentConfigurationTemplateGroupService(sess *session.Session) Monitoring_Agent_Configuration_Template_Group {
	return Monitoring_Agent_Configuration_Template_Group{Session: sess}
}

func (r Monitoring_Agent_Configuration_Template_Group) Id(id int) Monitoring_Agent_Configuration_Template_Group {
	r.Options.Id = &id
	return r
}

func (r Monitoring_Agent_Configuration_Template_Group) Mask(mask string) Monitoring_Agent_Configuration_Template_Group {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Monitoring_Agent_Configuration_Template_Group) Filter(filter string) Monitoring_Agent_Configuration_Template_Group {
	r.Options.Filter = filter
	return r
}

func (r Monitoring_Agent_Configuration_Template_Group) Limit(limit int) Monitoring_Agent_Configuration_Template_Group {
	r.Options.Limit = &limit
	return r
}

func (r Monitoring_Agent_Configuration_Template_Group) Offset(offset int) Monitoring_Agent_Configuration_Template_Group {
	r.Options.Offset = &offset
	return r
}

// This method creates a SoftLayer_Monitoring_Agent_Configuration_Template_Group using the values provided in the template object. The template objects accountId will be overridden to use the active user's accountId as it shows on their associated SoftLayer_User_Customer object.
func (r Monitoring_Agent_Configuration_Template_Group) CreateObject(templateObject *datatypes.Monitoring_Agent_Configuration_Template_Group) (resp datatypes.Monitoring_Agent_Configuration_Template_Group, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "createObject", params, &r.Options, &resp)
	return
}

// Deletes a customer configuration template group.
func (r Monitoring_Agent_Configuration_Template_Group) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "deleteObject", nil, &r.Options, &resp)
	return
}

// This method edits an existing SoftLayer_Monitoring_Agent_Configuration_Template_Group using the values passed in the $object parameter. The $object parameter should use the same structure as a SoftLayer_Monitoring_Agent_Configuration_Template_Group object.
func (r Monitoring_Agent_Configuration_Template_Group) EditObject(templateObject *datatypes.Monitoring_Agent_Configuration_Template_Group) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Monitoring_Agent_Configuration_Template_Group) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "getAccount", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Monitoring_Agent_Configuration_Template_Group) GetAllObjects() (resp []datatypes.Monitoring_Agent_Configuration_Template_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "getAllObjects", nil, &r.Options, &resp)
	return
}

// This method retrieves an array of SoftLayer_Monitoring_Agent_Configuration_Template_Group objects that are available to the active user's account. The packageId parameter is not currently used.
func (r Monitoring_Agent_Configuration_Template_Group) GetConfigurationGroups(packageId *int) (resp []datatypes.Monitoring_Agent_Configuration_Template_Group, err error) {
	params := []interface{}{
		packageId,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "getConfigurationGroups", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Monitoring_Agent_Configuration_Template_Group) GetConfigurationTemplateReferences() (resp []datatypes.Monitoring_Agent_Configuration_Template_Group_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "getConfigurationTemplateReferences", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Monitoring_Agent_Configuration_Template_Group) GetConfigurationTemplates() (resp []datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "getConfigurationTemplates", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Monitoring_Agent_Configuration_Template_Group) GetItem() (resp datatypes.Product_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "getItem", nil, &r.Options, &resp)
	return
}

// This method retrieves a monitoring agent configuration template group whose identifier corresponds to the value provided in the initialization parameter passed to the SoftLayer_Monitoring_Agent_Configuration_Template_Group service.
func (r Monitoring_Agent_Configuration_Template_Group) GetObject() (resp datatypes.Monitoring_Agent_Configuration_Template_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group", "getObject", nil, &r.Options, &resp)
	return
}

// SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference class holds the reference information, essentially a SQL join, between a monitoring configuration group and agent configuration templates.
type Monitoring_Agent_Configuration_Template_Group_Reference struct {
	Session *session.Session
	Options sl.Options
}

// GetMonitoringAgentConfigurationTemplateGroupReferenceService returns an instance of the Monitoring_Agent_Configuration_Template_Group_Reference SoftLayer service
func GetMonitoringAgentConfigurationTemplateGroupReferenceService(sess *session.Session) Monitoring_Agent_Configuration_Template_Group_Reference {
	return Monitoring_Agent_Configuration_Template_Group_Reference{Session: sess}
}

func (r Monitoring_Agent_Configuration_Template_Group_Reference) Id(id int) Monitoring_Agent_Configuration_Template_Group_Reference {
	r.Options.Id = &id
	return r
}

func (r Monitoring_Agent_Configuration_Template_Group_Reference) Mask(mask string) Monitoring_Agent_Configuration_Template_Group_Reference {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Monitoring_Agent_Configuration_Template_Group_Reference) Filter(filter string) Monitoring_Agent_Configuration_Template_Group_Reference {
	r.Options.Filter = filter
	return r
}

func (r Monitoring_Agent_Configuration_Template_Group_Reference) Limit(limit int) Monitoring_Agent_Configuration_Template_Group_Reference {
	r.Options.Limit = &limit
	return r
}

func (r Monitoring_Agent_Configuration_Template_Group_Reference) Offset(offset int) Monitoring_Agent_Configuration_Template_Group_Reference {
	r.Options.Offset = &offset
	return r
}

// This method creates a monitoring agent configuration template group reference by passing in an object with the SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference structure as the $templateObject parameter.
func (r Monitoring_Agent_Configuration_Template_Group_Reference) CreateObject(templateObject *datatypes.Monitoring_Agent_Configuration_Template_Group_Reference) (resp datatypes.Monitoring_Agent_Configuration_Template_Group_Reference, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference", "createObject", params, &r.Options, &resp)
	return
}

// This method creates monitoring agent configuration template group references by passing in an array of objects with the SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference structure as the $templateObjects parameter. Setting the $bulkCommit parameter to true will commit the changes in one transaction, false will commit after each object is created.
func (r Monitoring_Agent_Configuration_Template_Group_Reference) CreateObjects(templateObjects []datatypes.Monitoring_Agent_Configuration_Template_Group_Reference) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference", "createObjects", params, &r.Options, &resp)
	return
}

// This method updates a SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference record by passing in a modified instance of the object.
func (r Monitoring_Agent_Configuration_Template_Group_Reference) EditObject(templateObject *datatypes.Monitoring_Agent_Configuration_Template_Group_Reference) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference", "editObject", params, &r.Options, &resp)
	return
}

// This method updates a set of SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference records by passing in an array of modified instances of the objects. Setting the $bulkCommit parameter to true will commit the changes in one transaction, false will commit after each object is updated.
func (r Monitoring_Agent_Configuration_Template_Group_Reference) EditObjects(templateObjects []datatypes.Monitoring_Agent_Configuration_Template_Group_Reference) (resp bool, err error) {
	params := []interface{}{
		templateObjects,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference", "editObjects", params, &r.Options, &resp)
	return
}

// This method retrieves all SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference objects accessible to the active user.
func (r Monitoring_Agent_Configuration_Template_Group_Reference) GetAllObjects() (resp []datatypes.Monitoring_Agent_Configuration_Template_Group_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Monitoring_Agent_Configuration_Template_Group_Reference) GetConfigurationTemplate() (resp datatypes.Configuration_Template, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference", "getConfigurationTemplate", nil, &r.Options, &resp)
	return
}

// This method retrieves a monitoring agent configuration template group reference whose identifier corresponds to the value provided in the initialization parameter passed to the SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference service.
func (r Monitoring_Agent_Configuration_Template_Group_Reference) GetObject() (resp datatypes.Monitoring_Agent_Configuration_Template_Group_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Monitoring_Agent_Configuration_Template_Group_Reference) GetTemplateGroup() (resp datatypes.Monitoring_Agent_Configuration_Template_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Template_Group_Reference", "getTemplateGroup", nil, &r.Options, &resp)
	return
}

// Monitoring agent configuration value
type Monitoring_Agent_Configuration_Value struct {
	Session *session.Session
	Options sl.Options
}

// GetMonitoringAgentConfigurationValueService returns an instance of the Monitoring_Agent_Configuration_Value SoftLayer service
func GetMonitoringAgentConfigurationValueService(sess *session.Session) Monitoring_Agent_Configuration_Value {
	return Monitoring_Agent_Configuration_Value{Session: sess}
}

func (r Monitoring_Agent_Configuration_Value) Id(id int) Monitoring_Agent_Configuration_Value {
	r.Options.Id = &id
	return r
}

func (r Monitoring_Agent_Configuration_Value) Mask(mask string) Monitoring_Agent_Configuration_Value {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Monitoring_Agent_Configuration_Value) Filter(filter string) Monitoring_Agent_Configuration_Value {
	r.Options.Filter = filter
	return r
}

func (r Monitoring_Agent_Configuration_Value) Limit(limit int) Monitoring_Agent_Configuration_Value {
	r.Options.Limit = &limit
	return r
}

func (r Monitoring_Agent_Configuration_Value) Offset(offset int) Monitoring_Agent_Configuration_Value {
	r.Options.Offset = &offset
	return r
}

// Retrieve
func (r Monitoring_Agent_Configuration_Value) GetDefinition() (resp datatypes.Configuration_Template_Section_Definition, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Value", "getDefinition", nil, &r.Options, &resp)
	return
}

// Retrieve The metric data type used to retrieve metric data currently being tracked.
func (r Monitoring_Agent_Configuration_Value) GetMetricDataType() (resp datatypes.Container_Metric_Data_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Value", "getMetricDataType", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Monitoring_Agent_Configuration_Value) GetMonitoringAgent() (resp datatypes.Monitoring_Agent, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Value", "getMonitoringAgent", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Monitoring_Agent_Configuration_Value) GetObject() (resp datatypes.Monitoring_Agent_Configuration_Value, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Value", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Monitoring_Agent_Configuration_Value) GetProfile() (resp datatypes.Configuration_Template_Section_Profile, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Configuration_Value", "getProfile", nil, &r.Options, &resp)
	return
}

// Monitoring agent status
type Monitoring_Agent_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetMonitoringAgentStatusService returns an instance of the Monitoring_Agent_Status SoftLayer service
func GetMonitoringAgentStatusService(sess *session.Session) Monitoring_Agent_Status {
	return Monitoring_Agent_Status{Session: sess}
}

func (r Monitoring_Agent_Status) Id(id int) Monitoring_Agent_Status {
	r.Options.Id = &id
	return r
}

func (r Monitoring_Agent_Status) Mask(mask string) Monitoring_Agent_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Monitoring_Agent_Status) Filter(filter string) Monitoring_Agent_Status {
	r.Options.Filter = filter
	return r
}

func (r Monitoring_Agent_Status) Limit(limit int) Monitoring_Agent_Status {
	r.Options.Limit = &limit
	return r
}

func (r Monitoring_Agent_Status) Offset(offset int) Monitoring_Agent_Status {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Monitoring_Agent_Status) GetObject() (resp datatypes.Monitoring_Agent_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Agent_Status", "getObject", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Monitoring_Robot data type contains general information relating to a monitoring robot.
type Monitoring_Robot struct {
	Session *session.Session
	Options sl.Options
}

// GetMonitoringRobotService returns an instance of the Monitoring_Robot SoftLayer service
func GetMonitoringRobotService(sess *session.Session) Monitoring_Robot {
	return Monitoring_Robot{Session: sess}
}

func (r Monitoring_Robot) Id(id int) Monitoring_Robot {
	r.Options.Id = &id
	return r
}

func (r Monitoring_Robot) Mask(mask string) Monitoring_Robot {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Monitoring_Robot) Filter(filter string) Monitoring_Robot {
	r.Options.Filter = filter
	return r
}

func (r Monitoring_Robot) Limit(limit int) Monitoring_Robot {
	r.Options.Limit = &limit
	return r
}

func (r Monitoring_Robot) Offset(offset int) Monitoring_Robot {
	r.Options.Offset = &offset
	return r
}

// Checks if a monitoring robot can communicate with SoftLayer monitoring management system via the private network.
//
// TCP port 48000 - 48002 must be open on your server or your virtual server in order for this test to succeed.
func (r Monitoring_Robot) CheckConnection() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "checkConnection", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Monitoring_Robot) DeployMonitoringAgents(configurationTemplateGroup *datatypes.Monitoring_Agent_Configuration_Template_Group) (resp datatypes.Provisioning_Version1_Transaction, err error) {
	params := []interface{}{
		configurationTemplateGroup,
	}
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "deployMonitoringAgents", params, &r.Options, &resp)
	return
}

// Retrieve The account associated with the corresponding robot.
func (r Monitoring_Robot) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "getAccount", nil, &r.Options, &resp)
	return
}

// Returns available configuration template groups for this monitoring agent.
func (r Monitoring_Robot) GetAvailableConfigurationGroups() (resp []datatypes.Monitoring_Agent_Configuration_Template_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "getAvailableConfigurationGroups", nil, &r.Options, &resp)
	return
}

// Retrieve The program (monitoring agent) that gets details of a system or application and reporting of the metric data and triggers alarms for predefined events.
func (r Monitoring_Robot) GetMonitoringAgents() (resp []datatypes.Monitoring_Agent, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "getMonitoringAgents", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Monitoring_Robot) GetObject() (resp datatypes.Monitoring_Robot, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The current status of the robot.
func (r Monitoring_Robot) GetRobotStatus() (resp datatypes.Monitoring_Robot_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "getRobotStatus", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer_Software_Component that corresponds to the robot installation on the server.
func (r Monitoring_Robot) GetSoftwareComponent() (resp datatypes.Software_Component, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "getSoftwareComponent", nil, &r.Options, &resp)
	return
}

// If our monitoring management system is not able to connect to your monitoring robot, it sets the robot status to "Limited Connectivity". Robots in this status will not be process by our monitoring management system. You cannot manage monitoring agents either.
//
// Use this method to resets monitoring robot status to "Active" to indicate the connection issue is resolved.
func (r Monitoring_Robot) ResetStatus() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Monitoring_Robot", "resetStatus", nil, &r.Options, &resp)
	return
}
