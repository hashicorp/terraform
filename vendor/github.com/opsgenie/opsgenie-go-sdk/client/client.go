/*
Copyright 2015 OpsGenie. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package client provides clients for using the OpsGenie Web API. Also prepares and sends requests.
//API user first creates a OpsGenieClient instance.
//
//cli := new(ogcli.OpsGenieClient)
//
//Following that he/she can set APIKey and some configurations for HTTP communication layer by setting
//a proxy definition and/or transport layer options.
//
//cli.SetAPIKey(constants.APIKey)
//
//Then create the client of the API type that he/she wants to use.
//
//alertCli, cliErr := cli.Alert()
//
//if cliErr != nil {
//panic(cliErr)
//}
//
//The most fundamental and general use case is being able to access the
//OpsGenie Web API by coding a Go program.
//The program -by mean of a client application- can send OpsGenie Web API
//the requests using the 'client' package in a higher level. For the programmer
//of the client application, that reduces the number of LoCs.
//Besides it will result a less error-prone application and reduce
//the complexity by hiding the low-level networking, error-handling and
//byte-processing calls.
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/franela/goreq"
	goquery "github.com/google/go-querystring/query"
	"github.com/opsgenie/opsgenie-go-sdk/logging"
)

// endpointURL is the base URL of OpsGenie Web API.
var endpointURL = "https://api.opsgenie.com"

const (
	defaultConnectionTimeout time.Duration = 30 * time.Second
	defaultRequestTimeout    time.Duration = 60 * time.Second
	defaultMaxRetryAttempts  int           = 5
	timeSleepBetweenRequests time.Duration = 500 * time.Millisecond
)

// RequestHeaderUserAgent contains User-Agent values tool/version (OS;GO_Version;language).
type requestHeaderUserAgent struct {
	sdkName   string
	version   string
	os        string
	goVersion string
	timezone  string
}

// ToString formats and returns RequestHeaderUserAgent type's fields as string.
func (p requestHeaderUserAgent) ToString() string {
	return fmt.Sprintf("%s/%s (%s;%s;%s)", p.sdkName, p.version, p.os, p.goVersion, p.timezone)
}

var userAgentParam requestHeaderUserAgent

/*
OpsGenieClient is a general data type used for:
- authenticating callers through their API keys and
- instantiating "alert", "heartbeat", "integration" and "policy" clients
- setting HTTP transport layer configurations
- setting Proxy configurations
*/
type OpsGenieClient struct {
	proxy                 *ProxyConfiguration
	httpTransportSettings *HTTPTransportSettings
	apiKey                string
	opsGenieAPIURL        string
}

// SetProxyConfiguration sets proxy configurations of the OpsGenieClient.
func (cli *OpsGenieClient) SetProxyConfiguration(conf *ProxyConfiguration) {
	cli.proxy = conf
}

// SetHTTPTransportSettings sets HTTP transport layer configurations of the OpsGenieClient.
func (cli *OpsGenieClient) SetHTTPTransportSettings(settings *HTTPTransportSettings) {
	cli.httpTransportSettings = settings
}

// SetAPIKey sets API Key of the OpsGenieClient and authenticates callers through the API Key at OpsGenie.
func (cli *OpsGenieClient) SetAPIKey(key string) {
	cli.apiKey = key
}

// SetOpsGenieAPIUrl sets the endpoint(base URL) that requests will send. It can be used for testing purpose.
func (cli *OpsGenieClient) SetOpsGenieAPIUrl(url string) {
	if url != "" {
		cli.opsGenieAPIURL = url
	}
}

// OpsGenieAPIUrl returns the current endpoint(base URL) that requests will send.
func (cli *OpsGenieClient) OpsGenieAPIUrl() string {
	if cli.opsGenieAPIURL == "" {
		cli.opsGenieAPIURL = endpointURL
	}
	return cli.opsGenieAPIURL
}

// APIKey returns the API Key value that OpsGenieClient uses to authenticate at OpsGenie.
func (cli *OpsGenieClient) APIKey() string {
	return cli.apiKey
}

// makeHTTPTransportSettings internal method to set default values of HTTP transport layer configuration if necessary.
func (cli *OpsGenieClient) makeHTTPTransportSettings() {
	if cli.httpTransportSettings != nil {
		if cli.httpTransportSettings.MaxRetryAttempts <= 0 {
			cli.httpTransportSettings.MaxRetryAttempts = defaultMaxRetryAttempts
		}
		if cli.httpTransportSettings.ConnectionTimeout <= 0 {
			cli.httpTransportSettings.ConnectionTimeout = defaultConnectionTimeout
		}
		if cli.httpTransportSettings.RequestTimeout <= 0 {
			cli.httpTransportSettings.RequestTimeout = defaultRequestTimeout
		}
	} else {
		cli.httpTransportSettings = &HTTPTransportSettings{MaxRetryAttempts: defaultMaxRetryAttempts, ConnectionTimeout: defaultConnectionTimeout, RequestTimeout: defaultRequestTimeout}
	}
}

// Alert instantiates a new OpsGenieAlertClient.
func (cli *OpsGenieClient) Alert() (*OpsGenieAlertClient, error) {
	cli.makeHTTPTransportSettings()

	alertClient := new(OpsGenieAlertClient)
	alertClient.SetOpsGenieClient(*cli)

	if cli.opsGenieAPIURL == "" {
		alertClient.SetOpsGenieAPIUrl(endpointURL)
	}

	return alertClient, nil
}

// Heartbeat instantiates a new OpsGenieHeartbeatClient.
func (cli *OpsGenieClient) Heartbeat() (*OpsGenieHeartbeatClient, error) {
	cli.makeHTTPTransportSettings()

	heartbeatClient := new(OpsGenieHeartbeatClient)
	heartbeatClient.SetOpsGenieClient(*cli)

	if cli.opsGenieAPIURL == "" {
		heartbeatClient.SetOpsGenieAPIUrl(endpointURL)
	}

	return heartbeatClient, nil
}

// Integration instantiates a new OpsGenieIntegrationClient.
func (cli *OpsGenieClient) Integration() (*OpsGenieIntegrationClient, error) {
	cli.makeHTTPTransportSettings()

	integrationClient := new(OpsGenieIntegrationClient)
	integrationClient.SetOpsGenieClient(*cli)

	if cli.opsGenieAPIURL == "" {
		integrationClient.SetOpsGenieAPIUrl(endpointURL)
	}

	return integrationClient, nil
}

// Policy instantiates a new OpsGeniePolicyClient.
func (cli *OpsGenieClient) Policy() (*OpsGeniePolicyClient, error) {
	cli.makeHTTPTransportSettings()

	policyClient := new(OpsGeniePolicyClient)
	policyClient.SetOpsGenieClient(*cli)

	if cli.opsGenieAPIURL == "" {
		policyClient.SetOpsGenieAPIUrl(endpointURL)
	}

	return policyClient, nil
}

// Team instantiates a new OpsGenieTeamClient.
func (cli *OpsGenieClient) Team() (*OpsGenieTeamClient, error) {
	cli.makeHTTPTransportSettings()

	teamClient := new(OpsGenieTeamClient)
	teamClient.SetOpsGenieClient(*cli)

	if cli.opsGenieAPIURL == "" {
		teamClient.SetOpsGenieAPIUrl(endpointURL)
	}

	return teamClient, nil
}

// Escalation instantiates a new OpsGenieEscalationClient.
func (cli *OpsGenieClient) Escalation() (*OpsGenieEscalationClient, error) {
	cli.makeHTTPTransportSettings()

	escalationClient := new(OpsGenieEscalationClient)
	escalationClient.SetOpsGenieClient(*cli)

	if cli.opsGenieAPIURL == "" {
		escalationClient.SetOpsGenieAPIUrl(endpointURL)
	}

	return escalationClient, nil
}

// Schedule instantiates a new OpsGenieScheduleClient.
func (cli *OpsGenieClient) Schedule() (*OpsGenieScheduleClient, error) {
	cli.makeHTTPTransportSettings()

	scheduleClient := new(OpsGenieScheduleClient)
	scheduleClient.SetOpsGenieClient(*cli)

	if cli.opsGenieAPIURL == "" {
		scheduleClient.SetOpsGenieAPIUrl(endpointURL)
	}

	return scheduleClient, nil
}

// User instantiates a new OpsGenieUserClient.
func (cli *OpsGenieClient) User() (*OpsGenieUserClient, error) {
	cli.makeHTTPTransportSettings()

	userClient := new(OpsGenieUserClient)
	userClient.SetOpsGenieClient(*cli)

	if cli.opsGenieAPIURL == "" {
		userClient.SetOpsGenieAPIUrl(endpointURL)
	}

	return userClient, nil
}

// buildCommonRequestProps is an internal method to set common properties of requests that will send to OpsGenie.
func (cli *OpsGenieClient) buildCommonRequestProps() goreq.Request {
	if cli.httpTransportSettings == nil {
		cli.makeHTTPTransportSettings()
	}
	goreq.SetConnectTimeout(cli.httpTransportSettings.ConnectionTimeout)
	req := goreq.Request{}
	if cli.proxy != nil {
		req.Proxy = cli.proxy.toString()
	}
	req.UserAgent = userAgentParam.ToString()
	req.Timeout = cli.httpTransportSettings.RequestTimeout
	req.Insecure = true

	return req
}

// buildGetRequest is an internal method to prepare a "GET" request that will send to OpsGenie.
func (cli *OpsGenieClient) buildGetRequest(uri string, request interface{}) goreq.Request {
	req := cli.buildCommonRequestProps()
	req.Method = "GET"
	req.ContentType = "application/x-www-form-urlencoded; charset=UTF-8"
	uri = cli.OpsGenieAPIUrl() + uri
	if request != nil {
		v, _ := goquery.Values(request)
		req.Uri = uri + "?" + v.Encode()
	} else {
		req.Uri = uri 
	}
	logging.Logger().Info("Executing OpsGenie request to ["+uri+"] with parameters: ")
	return req
}

// buildPostRequest is an internal method to prepare a "POST" request that will send to OpsGenie.
func (cli *OpsGenieClient) buildPostRequest(uri string, request interface{}) goreq.Request {
	req := cli.buildCommonRequestProps()
	req.Method = "POST"
	req.ContentType = "application/json; charset=utf-8"
	req.Uri = cli.OpsGenieAPIUrl() + uri
	req.Body = request
	j, _ := json.Marshal(request)
	logging.Logger().Info("Executing OpsGenie request to ["+req.Uri+"] with content parameters: ", string(j))

	return req
}

// buildDeleteRequest is an internal method to prepare a "DELETE" request that will send to OpsGenie.
func (cli *OpsGenieClient) buildDeleteRequest(uri string, request interface{}) goreq.Request {
	req := cli.buildGetRequest(uri, request)
	req.Method = "DELETE"
	return req
}

// sendRequest is an internal method to send the prepared requests to OpsGenie.
func (cli *OpsGenieClient) sendRequest(req goreq.Request) (*goreq.Response, error) {
	// send the request
	var resp *goreq.Response
	var err error
	for i := 0; i < cli.httpTransportSettings.MaxRetryAttempts; i++ {
		resp, err = req.Do()
		if err == nil && resp.StatusCode < 500 {
			break
		}
		if resp != nil {
			defer resp.Body.Close()
			logging.Logger().Info(fmt.Sprintf("Retrying request [%s] ResponseCode:[%d]. RetryCount: %d", req.Uri, resp.StatusCode, (i + 1)))
		} else {
			logging.Logger().Info(fmt.Sprintf("Retrying request [%s] Reason:[%s]. RetryCount: %d", req.Uri, err.Error(), (i + 1)))
		}
		time.Sleep(timeSleepBetweenRequests * time.Duration(i+1))
	}
	if err != nil {
		message := "Unable to send the request " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	// check for the returning http status
	statusCode := resp.StatusCode
	if statusCode >= 400 {
		body, err := resp.Body.ToString()
		if err != nil {
			message := "Server response with error can not be parsed " + err.Error()
			logging.Logger().Warn(message)
			return nil, errors.New(message)
		}
		return nil, errorMessage(statusCode, body)
	}
	return resp, nil
}

// errorMessage is an internal method to return formatted error message according to HTTP status code of the response.
func errorMessage(httpStatusCode int, responseBody string) error {
	if httpStatusCode >= 400 && httpStatusCode < 500 {
		message := fmt.Sprintf("Client error occurred; Response Code: %d, Response Body: %s", httpStatusCode, responseBody)
		logging.Logger().Warn(message)
		return errors.New(message)
	}
	if httpStatusCode >= 500 {
		message := fmt.Sprintf("Server error occurred; Response Code: %d, Response Body: %s", httpStatusCode, responseBody)
		logging.Logger().Info(message)
		return errors.New(message)
	}
	return nil
}

// Initializer for the package client
// Initializes the User-Agent parameter of the requests.
// TODO version information must be read from a MANIFEST file
func init() {
	userAgentParam.sdkName = "opsgenie-go-sdk"
	userAgentParam.version = "1.0.0"
	userAgentParam.os = runtime.GOOS
	userAgentParam.goVersion = runtime.Version()
	userAgentParam.timezone = time.Local.String()
}
