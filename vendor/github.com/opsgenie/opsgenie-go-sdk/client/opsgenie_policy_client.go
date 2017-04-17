package client

import (
	"errors"

	"github.com/opsgenie/opsgenie-go-sdk/logging"
	policy "github.com/opsgenie/opsgenie-go-sdk/policy"
)

const (
	enablePolicyURL  = "/v1/json/alert/policy/enable"
	disablePolicyURL = "/v1/json/alert/policy/disable"
)

// OpsGeniePolicyClient is the data type to make Policy API requests.
type OpsGeniePolicyClient struct {
	OpsGenieClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGeniePolicyClient.
func (cli *OpsGeniePolicyClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

// Enable method enables an Policy at OpsGenie.
func (cli *OpsGeniePolicyClient) Enable(req policy.EnablePolicyRequest) (*policy.EnablePolicyResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(enablePolicyURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var enablePolicyResp policy.EnablePolicyResponse
	if err = resp.Body.FromJsonTo(&enablePolicyResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &enablePolicyResp, nil
}

// Disable method disables an Policy at OpsGenie.
func (cli *OpsGeniePolicyClient) Disable(req policy.DisablePolicyRequest) (*policy.DisablePolicyResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(disablePolicyURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var disablePolicyResp policy.DisablePolicyResponse
	if err = resp.Body.FromJsonTo(&disablePolicyResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &disablePolicyResp, nil
}
