package client

import (
	"errors"

	"github.com/opsgenie/opsgenie-go-sdk/heartbeat"
	"github.com/opsgenie/opsgenie-go-sdk/logging"
)

const (
	addHeartbeatURL     = "/v1/json/heartbeat"
	updateHeartbeatURL  = "/v1/json/heartbeat"
	enableHeartbeatURL  = "/v1/json/heartbeat/enable"
	disableHeartbeatURL = "/v1/json/heartbeat/disable"
	deleteHeartbeatURL  = "/v1/json/heartbeat"
	getHeartbeatURL     = "/v1/json/heartbeat"
	listHeartbeatURL    = "/v1/json/heartbeat"
	sendHeartbeatURL    = "/v1/json/heartbeat/send"
)

// OpsGenieHeartbeatClient is the data type to make Heartbeat API requests.
type OpsGenieHeartbeatClient struct {
	OpsGenieClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGenieHeartbeatClient.
func (cli *OpsGenieHeartbeatClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

// Add method creates a heartbeat at OpsGenie.
func (cli *OpsGenieHeartbeatClient) Add(req heartbeat.AddHeartbeatRequest) (*heartbeat.AddHeartbeatResponse, error) {
	req.APIKey = cli.apiKey

	resp, err := cli.sendRequest(cli.buildPostRequest(addHeartbeatURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var addHeartbeatResp heartbeat.AddHeartbeatResponse
	if err = resp.Body.FromJsonTo(&addHeartbeatResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &addHeartbeatResp, nil
}

// Update method changes configuration of an existing heartbeat at OpsGenie.
func (cli *OpsGenieHeartbeatClient) Update(req heartbeat.UpdateHeartbeatRequest) (*heartbeat.UpdateHeartbeatResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(updateHeartbeatURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updateHeartbeatResp heartbeat.UpdateHeartbeatResponse
	if err = resp.Body.FromJsonTo(&updateHeartbeatResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &updateHeartbeatResp, nil
}

// Enable method enables an heartbeat at OpsGenie.
func (cli *OpsGenieHeartbeatClient) Enable(req heartbeat.EnableHeartbeatRequest) (*heartbeat.EnableHeartbeatResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(enableHeartbeatURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var enableHeartbeatResp heartbeat.EnableHeartbeatResponse
	if err = resp.Body.FromJsonTo(&enableHeartbeatResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &enableHeartbeatResp, nil
}

// Disable method disables an heartbeat at OpsGenie.
func (cli *OpsGenieHeartbeatClient) Disable(req heartbeat.DisableHeartbeatRequest) (*heartbeat.DisableHeartbeatResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(disableHeartbeatURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var disableHeartbeatResp heartbeat.DisableHeartbeatResponse
	if err = resp.Body.FromJsonTo(&disableHeartbeatResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &disableHeartbeatResp, nil

}

// Delete method deletes an heartbeat from OpsGenie.
func (cli *OpsGenieHeartbeatClient) Delete(req heartbeat.DeleteHeartbeatRequest) (*heartbeat.DeleteHeartbeatResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildDeleteRequest(deleteHeartbeatURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deleteHeartbeatResp heartbeat.DeleteHeartbeatResponse
	if err = resp.Body.FromJsonTo(&deleteHeartbeatResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &deleteHeartbeatResp, nil
}

// Get method retrieves an heartbeat with details from OpsGenie.
func (cli *OpsGenieHeartbeatClient) Get(req heartbeat.GetHeartbeatRequest) (*heartbeat.GetHeartbeatResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(getHeartbeatURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var getHeartbeatResp heartbeat.GetHeartbeatResponse
	if err = resp.Body.FromJsonTo(&getHeartbeatResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &getHeartbeatResp, nil
}

// List method retrieves heartbeats from OpsGenie.
func (cli *OpsGenieHeartbeatClient) List(req heartbeat.ListHeartbeatsRequest) (*heartbeat.ListHeartbeatsResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(listHeartbeatURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listHeartbeatsResp heartbeat.ListHeartbeatsResponse
	if err = resp.Body.FromJsonTo(&listHeartbeatsResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &listHeartbeatsResp, nil
}

// Send method sends an Heartbeat Signal to OpsGenie.
func (cli *OpsGenieHeartbeatClient) Send(req heartbeat.SendHeartbeatRequest) (*heartbeat.SendHeartbeatResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(sendHeartbeatURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sendHeartbeatResp heartbeat.SendHeartbeatResponse
	if err = resp.Body.FromJsonTo(&sendHeartbeatResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &sendHeartbeatResp, nil
}
