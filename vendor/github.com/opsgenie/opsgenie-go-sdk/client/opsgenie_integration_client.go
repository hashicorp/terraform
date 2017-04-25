package client

import (
	"errors"

	integration "github.com/opsgenie/opsgenie-go-sdk/integration"
	"github.com/opsgenie/opsgenie-go-sdk/logging"
)

const (
	enableIntegrationURL  = "/v1/json/integration/enable"
	disableIntegrationURL = "/v1/json/integration/disable"
)

// OpsGenieIntegrationClient is the data type to make Integration API requests.
type OpsGenieIntegrationClient struct {
	OpsGenieClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGenieIntegrationClient.
func (cli *OpsGenieIntegrationClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

// Enable method enables an Integration at OpsGenie.
func (cli *OpsGenieIntegrationClient) Enable(req integration.EnableIntegrationRequest) (*integration.EnableIntegrationResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(enableIntegrationURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var enableIntegrationResp integration.EnableIntegrationResponse
	if err = resp.Body.FromJsonTo(&enableIntegrationResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &enableIntegrationResp, nil
}

// Disable method disables an Integration at OpsGenie.
func (cli *OpsGenieIntegrationClient) Disable(req integration.DisableIntegrationRequest) (*integration.DisableIntegrationResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(disableIntegrationURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var disableIntegrationResp integration.DisableIntegrationResponse
	if err = resp.Body.FromJsonTo(&disableIntegrationResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	return &disableIntegrationResp, nil
}
