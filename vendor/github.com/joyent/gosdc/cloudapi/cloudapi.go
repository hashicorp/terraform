/*
Package cloudapi interacts with the Cloud API (http://apidocs.joyent.com/cloudapi/).

Licensed under the Mozilla Public License version 2.0

Copyright (c) Joyent Inc.
*/
package cloudapi

import (
	"net/http"
	"net/url"
	"path"

	"github.com/joyent/gocommon/client"
	jh "github.com/joyent/gocommon/http"
)

const (
	// DefaultAPIVersion defines the default version of the Cloud API to use
	DefaultAPIVersion = "~7.3"

	// CloudAPI URL parts
	apiKeys                    = "keys"
	apiPackages                = "packages"
	apiImages                  = "images"
	apiDatacenters             = "datacenters"
	apiMachines                = "machines"
	apiMetadata                = "metadata"
	apiSnapshots               = "snapshots"
	apiTags                    = "tags"
	apiAnalytics               = "analytics"
	apiInstrumentations        = "instrumentations"
	apiInstrumentationsValue   = "value"
	apiInstrumentationsRaw     = "raw"
	apiInstrumentationsHeatmap = "heatmap"
	apiInstrumentationsImage   = "image"
	apiInstrumentationsDetails = "details"
	apiUsage                   = "usage"
	apiAudit                   = "audit"
	apiFirewallRules           = "fwrules"
	apiFirewallRulesEnable     = "enable"
	apiFirewallRulesDisable    = "disable"
	apiNetworks                = "networks"
	apiFabricVLANs             = "fabrics/default/vlans"
	apiFabricNetworks          = "networks"
	apiNICs                    = "nics"
	apiServices                = "services"

	// CloudAPI actions
	actionExport    = "export"
	actionStop      = "stop"
	actionStart     = "start"
	actionReboot    = "reboot"
	actionResize    = "resize"
	actionRename    = "rename"
	actionEnableFw  = "enable_firewall"
	actionDisableFw = "disable_firewall"
)

// Client provides a means to access the Joyent CloudAPI
type Client struct {
	client client.Client
}

// New creates a new Client.
func New(client client.Client) *Client {
	return &Client{client}
}

// Filter represents a filter that can be applied to an API request.
type Filter struct {
	v url.Values
}

// NewFilter creates a new Filter.
func NewFilter() *Filter {
	return &Filter{make(url.Values)}
}

// Set a value for the specified filter.
func (f *Filter) Set(filter, value string) {
	f.v.Set(filter, value)
}

// Add a value for the specified filter.
func (f *Filter) Add(filter, value string) {
	f.v.Add(filter, value)
}

// request represents an API request
type request struct {
	method         string
	url            string
	filter         *Filter
	reqValue       interface{}
	reqHeader      http.Header
	resp           interface{}
	respHeader     *http.Header
	expectedStatus int
}

// Helper method to send an API request
func (c *Client) sendRequest(req request) (*jh.ResponseData, error) {
	request := jh.RequestData{
		ReqValue:   req.reqValue,
		ReqHeaders: req.reqHeader,
	}
	if req.filter != nil {
		request.Params = &req.filter.v
	}
	if req.expectedStatus == 0 {
		req.expectedStatus = http.StatusOK
	}
	respData := jh.ResponseData{
		RespValue:      req.resp,
		RespHeaders:    req.respHeader,
		ExpectedStatus: []int{req.expectedStatus},
	}
	err := c.client.SendRequest(req.method, req.url, "", &request, &respData)
	return &respData, err
}

// Helper method to create the API URL
func makeURL(parts ...string) string {
	return path.Join(parts...)
}
