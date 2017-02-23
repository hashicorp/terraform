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

// Metric tracking objects provides a common interface to all metrics provided by SoftLayer. These metrics range from network component traffic for a server to aggregated Bandwidth Pooling traffic and more. Every object within SoftLayer's range of objects that has data that can be tracked over time has an associated tracking object. Use the [[SoftLayer_Metric_Tracking_Object]] service to retrieve raw and graph data from a tracking object.
type Metric_Tracking_Object struct {
	Session *session.Session
	Options sl.Options
}

// GetMetricTrackingObjectService returns an instance of the Metric_Tracking_Object SoftLayer service
func GetMetricTrackingObjectService(sess *session.Session) Metric_Tracking_Object {
	return Metric_Tracking_Object{Session: sess}
}

func (r Metric_Tracking_Object) Id(id int) Metric_Tracking_Object {
	r.Options.Id = &id
	return r
}

func (r Metric_Tracking_Object) Mask(mask string) Metric_Tracking_Object {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Metric_Tracking_Object) Filter(filter string) Metric_Tracking_Object {
	r.Options.Filter = filter
	return r
}

func (r Metric_Tracking_Object) Limit(limit int) Metric_Tracking_Object {
	r.Options.Limit = &limit
	return r
}

func (r Metric_Tracking_Object) Offset(offset int) Metric_Tracking_Object {
	r.Options.Offset = &offset
	return r
}

// Retrieve a PNG image of the last 24 hours of bandwidth usage of one of SoftLayer's network backbones.
func (r Metric_Tracking_Object) GetBackboneBandwidthGraph(graphTitle *string) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		graphTitle,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getBackboneBandwidthGraph", params, &r.Options, &resp)
	return
}

// Retrieve a collection of raw bandwidth data from an individual public or private network tracking object. Raw data is ideal if you with to employ your own traffic storage and graphing systems.
func (r Metric_Tracking_Object) GetBandwidthData(startDateTime *datatypes.Time, endDateTime *datatypes.Time, typ *string, rollupSeconds *int) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
		typ,
		rollupSeconds,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getBandwidthData", params, &r.Options, &resp)
	return
}

// Retrieve a PNG image of a bandwidth graph representing the bandwidth usage over time recorded by SofTLayer's bandwidth pollers.
func (r Metric_Tracking_Object) GetBandwidthGraph(startDateTime *datatypes.Time, endDateTime *datatypes.Time, graphType *string, fontSize *int, graphWidth *int, graphHeight *int, doNotShowTimeZone *bool) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
		graphType,
		fontSize,
		graphWidth,
		graphHeight,
		doNotShowTimeZone,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getBandwidthGraph", params, &r.Options, &resp)
	return
}

// Retrieve the total amount of bandwidth recorded by a tracking object within the given date range. This method will only work on SoftLayer_Metric_Tracking_Object for SoftLayer_Hardware objects, and SoftLayer_Virtual_Guest objects.
func (r Metric_Tracking_Object) GetBandwidthTotal(startDateTime *datatypes.Time, endDateTime *datatypes.Time, direction *string, typ *string) (resp uint, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
		direction,
		typ,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getBandwidthTotal", params, &r.Options, &resp)
	return
}

// Returns a graph container instance that is populated with metric data for the tracking object.
func (r Metric_Tracking_Object) GetCustomGraphData(graphContainer *datatypes.Container_Graph) (resp datatypes.Container_Graph, err error) {
	params := []interface{}{
		graphContainer,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getCustomGraphData", params, &r.Options, &resp)
	return
}

// Retrieve a collection of detailed metric data over a date range. Ideal if you want to employ your own graphing systems.  Note not all metrics support this method.  Those that do not return null.
func (r Metric_Tracking_Object) GetDetailsForDateRange(startDate *datatypes.Time, endDate *datatypes.Time, graphType []string) (resp []datatypes.Container_Metric_Tracking_Object_Details, err error) {
	params := []interface{}{
		startDate,
		endDate,
		graphType,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getDetailsForDateRange", params, &r.Options, &resp)
	return
}

// Retrieve a PNG image of a metric in graph form.
func (r Metric_Tracking_Object) GetGraph(startDateTime *datatypes.Time, endDateTime *datatypes.Time, graphType []string) (resp datatypes.Container_Bandwidth_GraphOutputs, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
		graphType,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getGraph", params, &r.Options, &resp)
	return
}

// Returns a collection of metric data types that can be retrieved for a metric tracking object.
func (r Metric_Tracking_Object) GetMetricDataTypes() (resp []datatypes.Container_Metric_Data_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getMetricDataTypes", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Metric_Tracking_Object object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Metric_Tracking_Object service. You can only tracking objects that are associated with your SoftLayer account or services.
func (r Metric_Tracking_Object) GetObject() (resp datatypes.Metric_Tracking_Object, err error) {
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve a metric summary. Ideal if you want to employ your own graphing systems.  Note not all metric types contain a summary.  These return null.
func (r Metric_Tracking_Object) GetSummary(graphType *string) (resp datatypes.Container_Metric_Tracking_Object_Summary, err error) {
	params := []interface{}{
		graphType,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getSummary", params, &r.Options, &resp)
	return
}

// Returns summarized metric data for the date range, metric type and summary period provided.
func (r Metric_Tracking_Object) GetSummaryData(startDateTime *datatypes.Time, endDateTime *datatypes.Time, validTypes []datatypes.Container_Metric_Data_Type, summaryPeriod *int) (resp []datatypes.Metric_Tracking_Object_Data, err error) {
	params := []interface{}{
		startDateTime,
		endDateTime,
		validTypes,
		summaryPeriod,
	}
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getSummaryData", params, &r.Options, &resp)
	return
}

// Retrieve The type of data that a tracking object polls.
func (r Metric_Tracking_Object) GetType() (resp datatypes.Metric_Tracking_Object_Type, err error) {
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object", "getType", nil, &r.Options, &resp)
	return
}

// This data type provides commonly used bandwidth summary components for the current billing cycle.
type Metric_Tracking_Object_Bandwidth_Summary struct {
	Session *session.Session
	Options sl.Options
}

// GetMetricTrackingObjectBandwidthSummaryService returns an instance of the Metric_Tracking_Object_Bandwidth_Summary SoftLayer service
func GetMetricTrackingObjectBandwidthSummaryService(sess *session.Session) Metric_Tracking_Object_Bandwidth_Summary {
	return Metric_Tracking_Object_Bandwidth_Summary{Session: sess}
}

func (r Metric_Tracking_Object_Bandwidth_Summary) Id(id int) Metric_Tracking_Object_Bandwidth_Summary {
	r.Options.Id = &id
	return r
}

func (r Metric_Tracking_Object_Bandwidth_Summary) Mask(mask string) Metric_Tracking_Object_Bandwidth_Summary {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Metric_Tracking_Object_Bandwidth_Summary) Filter(filter string) Metric_Tracking_Object_Bandwidth_Summary {
	r.Options.Filter = filter
	return r
}

func (r Metric_Tracking_Object_Bandwidth_Summary) Limit(limit int) Metric_Tracking_Object_Bandwidth_Summary {
	r.Options.Limit = &limit
	return r
}

func (r Metric_Tracking_Object_Bandwidth_Summary) Offset(offset int) Metric_Tracking_Object_Bandwidth_Summary {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Metric_Tracking_Object_Bandwidth_Summary) GetObject() (resp datatypes.Metric_Tracking_Object_Bandwidth_Summary, err error) {
	err = r.Session.DoRequest("SoftLayer_Metric_Tracking_Object_Bandwidth_Summary", "getObject", nil, &r.Options, &resp)
	return
}
